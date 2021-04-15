package condaenvupdate_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	condaenvupdate "github.com/paketo-community/conda-env-update"
	"github.com/paketo-community/conda-env-update/fakes"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testCondaRunner(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		workingDir     string
		condaLayerPath string
		condaCachePath string

		executable *fakes.Executable
		summer     *fakes.Summer
		runner     condaenvupdate.CondaRunner
	)

	it.Before(func() {
		var err error

		workingDir, err = os.MkdirTemp("", "working-dir")
		Expect(err).NotTo(HaveOccurred())
		condaLayerPath = "a-conda-layer"
		condaCachePath = "a-conda-cache-path"

		executable = &fakes.Executable{}
		summer = &fakes.Summer{}
		runner = condaenvupdate.NewCondaRunner(executable, summer)
	})

	it.After(func() {
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	context("ShouldRun", func() {
		it("returns true, with no sha, and no error when no lockfile is present", func() {
			run, sha, err := runner.ShouldRun(workingDir, map[string]interface{}{})
			Expect(run).To(BeTrue())
			Expect(sha).To(Equal(""))
			Expect(err).NotTo(HaveOccurred())
		})

		context("when there is an error checking if a lockfile is present", func() {
			it.Before(func() {
				Expect(os.Chmod(workingDir, 0000)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Chmod(workingDir, os.ModePerm)).To(Succeed())
			})

			it("returns false, with no sha, and an error", func() {
				run, sha, err := runner.ShouldRun(workingDir, map[string]interface{}{})
				Expect(run).To(BeFalse())
				Expect(sha).To(Equal(""))
				Expect(err).To(HaveOccurred())
			})
		})

		context("when a lockfile is present", func() {
			it.Before(func() {
				Expect(os.WriteFile(filepath.Join(workingDir, "package-list.txt"), nil, os.ModePerm)).To(Succeed())
			})
			context("and the lockfile sha is unchanged", func() {
				it("return false, with the existing sha, and no error", func() {
					summer.SumCall.Returns.String = "a-sha"
					Expect(os.WriteFile(filepath.Join(workingDir, "package-list.txt"), nil, os.ModePerm)).To(Succeed())

					metadata := map[string]interface{}{
						"lockfile-sha": "a-sha",
					}

					run, sha, err := runner.ShouldRun(workingDir, metadata)
					Expect(run).To(BeFalse())
					Expect(sha).To(Equal("a-sha"))
					Expect(err).NotTo(HaveOccurred())
				})
				context("and there is and error summing the lock file", func() {
					it.Before(func() {
						summer.SumCall.Returns.Error = errors.New("summing lockfile failed")
					})

					it("returns false, with no sha, and an error", func() {
						run, sha, err := runner.ShouldRun(workingDir, map[string]interface{}{})
						Expect(run).To(BeFalse())
						Expect(sha).To(Equal(""))
						Expect(err).To(MatchError("summing lockfile failed"))

					})
				})
			})

			it("returns true, with a new sha, and no error when the lockfile has changed", func() {
				summer.SumCall.Returns.String = "a-new-sha"
				metadata := map[string]interface{}{
					"lockfile-sha": "a-sha",
				}

				run, sha, err := runner.ShouldRun(workingDir, metadata)
				Expect(run).To(BeTrue())
				Expect(sha).To(Equal("a-new-sha"))
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
	context("Execute", func() {
		context("when a vendor dir is present", func() {
			var vendorPath string
			it.Before(func() {
				vendorPath = filepath.Join(workingDir, "vendor")
				Expect(os.Mkdir(vendorPath, os.ModePerm))
			})

			it("runs conda create with additional vedor args", func() {
				err := runner.Execute(condaLayerPath, condaCachePath, workingDir)
				Expect(err).NotTo(HaveOccurred())

				Expect(executable.ExecuteCall.Receives.Execution.Args).To(Equal([]string{
					"create",
					"--file", filepath.Join(workingDir, "package-list.txt"),
					"--prefix", condaLayerPath,
					"--yes",
					"--quiet",
					"--channel", vendorPath,
					"--override-channels",
					"--offline",
				}))
			})

		})

		context("when a lockfile exists", func() {
			it.Before(func() {
				Expect(os.WriteFile(filepath.Join(workingDir, condaenvupdate.LockfileName), nil, os.ModePerm)).To(Succeed())
			})
			it("runs conda create", func() {
				err := runner.Execute(condaLayerPath, condaCachePath, workingDir)
				Expect(err).NotTo(HaveOccurred())

				Expect(executable.ExecuteCall.Receives.Execution.Args).To(Equal([]string{
					"create",
					"--file", filepath.Join(workingDir, "package-list.txt"),
					"--prefix", condaLayerPath,
					"--yes",
					"--quiet",
				}))

			})
		})

		context("when no vendor dir or lockfile exists", func() {
			it("runs conda env update", func() {
				err := runner.Execute(condaLayerPath, condaCachePath, workingDir)
				Expect(err).NotTo(HaveOccurred())

				Expect(executable.ExecuteCall.Receives.Execution.Args).To(Equal([]string{
					"env",
					"update",
					"--prefix", condaLayerPath,
					"--file", filepath.Join(workingDir, "environment.yml"),
				}))
				Expect(executable.ExecuteCall.Receives.Execution.Env).To(Equal(append(os.Environ(), fmt.Sprintf("CONDA_PKGS_DIRS=%s", "a-conda-cache-path"))))
			})

			context.Pend("failures cases", func() {
				context("when the conda command fails to run", func() {
					it.Before(func() {
						executable.ExecuteCall.Returns.Error = errors.New("failed to run conda command")
					})

					it("returns an error", func() {
						err := runner.Execute(condaLayerPath, condaCachePath, workingDir)
						Expect(err).To(MatchError("failed to run conda command"))
					})
				})
			})
		})
	})
}
