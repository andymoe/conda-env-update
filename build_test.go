package condaenvupdate_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/scribe"
	condaenvupdate "github.com/paketo-community/conda-env-update"
	"github.com/paketo-community/conda-env-update/fakes"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layersDir  string
		workingDir string
		cnbDir     string

		runner *fakes.Runner
		clock  chronos.Clock
		now    time.Time
		buffer *bytes.Buffer

		build packit.BuildFunc
	)

	it.Before(func() {
		var err error
		layersDir, err = os.MkdirTemp("", "layers")
		Expect(err).NotTo(HaveOccurred())

		cnbDir, err = os.MkdirTemp("", "cnb")
		Expect(err).NotTo(HaveOccurred())

		workingDir, err = os.MkdirTemp("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		runner = &fakes.Runner{}
		runner.ShouldRunCall.Returns.Bool = true
		runner.ShouldRunCall.Returns.String = "some-sha"

		now = time.Now()
		clock = chronos.NewClock(func() time.Time {
			return now
		})

		buffer = bytes.NewBuffer(nil)
		logger := scribe.NewLogger(buffer)

		build = condaenvupdate.Build(runner, logger, clock)
	})

	it.After(func() {
		Expect(os.RemoveAll(layersDir)).To(Succeed())
		Expect(os.RemoveAll(cnbDir)).To(Succeed())
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	it("returns a result that builds correctly", func() {
		context := packit.BuildContext{
			WorkingDir: workingDir,
			CNBPath:    cnbDir,
			Stack:      "some-stack",
			BuildpackInfo: packit.BuildpackInfo{
				Name:    "Some Buildpack",
				Version: "some-version",
			},
			Plan: packit.BuildpackPlan{
				Entries: []packit.BuildpackPlanEntry{},
			},
			Layers: packit.Layers{Path: layersDir},
		}

		result, err := build(context)
		Expect(err).NotTo(HaveOccurred())

		expectedCondaLayer, err := context.Layers.Get("conda-env")
		Expect(err).NotTo(HaveOccurred())
		expectedCondaLayer.Launch = true
		expectedCondaLayer.Metadata = map[string]interface{}{
			"built_at":     clock.Now().Format(time.RFC3339Nano),
			"lockfile-sha": "some-sha",
		}

		expectedCondaCacheLayer, err := context.Layers.Get("conda-env-cache")
		Expect(err).NotTo(HaveOccurred())
		expectedCondaCacheLayer.Cache = true

		Expect(result).To(Equal(packit.BuildResult{
			Layers: []packit.Layer{
				expectedCondaLayer,
				expectedCondaCacheLayer,
			},
		}))

		Expect(runner.ExecuteCall.Receives.CondaEnvPath).To(Equal(expectedCondaLayer.Path))
		Expect(runner.ExecuteCall.Receives.CondaCachePath).To(Equal(expectedCondaCacheLayer.Path))
		Expect(runner.ExecuteCall.Receives.WorkingDir).To(Equal(workingDir))
		// Expect(buffer.String()).To(ContainSubstring("Executing build process")) TODO: Add logging tests

	})
	//TODO: Assert somewhere that the Build/Cache/Launch layer flags for the
	//conda packages layer are being set appropriately based on build plan
	//entries -- (i think?) the buildpack should only flag the layer at build/launch if it's been requested via a build plan entry.
	context("cached packages should be reused", func() {
		it.Before(func() {
			runner.ShouldRunCall.Returns.Bool = false
			runner.ShouldRunCall.Returns.String = "cached-sha"

		})
		it("reuses cached conda env layer instead of running build process", func() {
			context := packit.BuildContext{
				WorkingDir: workingDir,
				CNBPath:    cnbDir,
				Stack:      "some-stack",
				BuildpackInfo: packit.BuildpackInfo{
					Name:    "Some Buildpack",
					Version: "some-version",
				},
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{},
				},
				Layers: packit.Layers{Path: layersDir},
			}

			result, err := build(context)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers).To(ContainElement(packit.Layer{
				Name:             "conda-env",
				Path:             filepath.Join(layersDir, "conda-env"),
				Launch:           true,
				SharedEnv:        packit.Environment{},
				BuildEnv:         packit.Environment{},
				LaunchEnv:        packit.Environment{},
				ProcessLaunchEnv: map[string]packit.Environment{},
			}))

			Expect(runner.ExecuteCall.CallCount).To(BeZero())
			// Expect(buffer.String()).To(ContainSubstring("Reusing cached layer")) TODO: Add logging tests
		})
	})

	context("failure cases", func() {
		context("conda layer cannot be fetched", func() {
			it.Before(func() {
				Expect(os.WriteFile(filepath.Join(layersDir, "conda-env.toml"), nil, 0000)).To(Succeed())
			})

			it("returns an error", func() {
				context := packit.BuildContext{
					WorkingDir: workingDir,
					CNBPath:    cnbDir,
					Stack:      "some-stack",
					BuildpackInfo: packit.BuildpackInfo{
						Name:    "Some Buildpack",
						Version: "some-version",
					},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{},
					},
					Layers: packit.Layers{Path: layersDir},
				}

				_, err := build(context)
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})

		context("conda cache layer cannot be fetched", func() {
			it.Before(func() {
				Expect(os.WriteFile(filepath.Join(layersDir, "conda-env-cache.toml"), nil, 0000)).To(Succeed())
			})

			it("returns an error", func() {
				context := packit.BuildContext{
					WorkingDir: workingDir,
					CNBPath:    cnbDir,
					Stack:      "some-stack",
					BuildpackInfo: packit.BuildpackInfo{
						Name:    "Some Buildpack",
						Version: "some-version",
					},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{},
					},
					Layers: packit.Layers{Path: layersDir},
				}

				_, err := build(context)
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})

		context("runner ShouldRun fails", func() {
			it.Before(func() {
				runner.ShouldRunCall.Returns.Error = errors.New("some-shouldrun-error")
			})

			it("returns an error", func() {
				context := packit.BuildContext{
					WorkingDir: workingDir,
					CNBPath:    cnbDir,
					Stack:      "some-stack",
					BuildpackInfo: packit.BuildpackInfo{
						Name:    "Some Buildpack",
						Version: "some-version",
					},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{},
					},
					Layers: packit.Layers{Path: layersDir},
				}

				_, err := build(context)
				Expect(err).To(MatchError("some-shouldrun-error"))
			})
		})

		context("layer cannot be reset", func() {
			it.Before(func() {
				runner.ShouldRunCall.Returns.Bool = true
				Expect(os.Chmod(layersDir, 0500)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Chmod(layersDir, os.ModePerm)).To(Succeed())
			})

			it("returns an error", func() {
				context := packit.BuildContext{
					WorkingDir: workingDir,
					CNBPath:    cnbDir,
					Stack:      "some-stack",
					BuildpackInfo: packit.BuildpackInfo{
						Name:    "Some Buildpack",
						Version: "some-version",
					},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{},
					},
					Layers: packit.Layers{Path: layersDir},
				}

				_, err := build(context)
				Expect(err).To(MatchError(ContainSubstring("error could not create directory")))
			})
		})

		context("install process fails to execute", func() {
			it.Before(func() {
				runner.ShouldRunCall.Returns.Bool = true
				runner.ExecuteCall.Returns.Error = errors.New("some execution error")
			})

			it("returns an error", func() {
				context := packit.BuildContext{
					WorkingDir: workingDir,
					CNBPath:    cnbDir,
					Stack:      "some-stack",
					BuildpackInfo: packit.BuildpackInfo{
						Name:    "Some Buildpack",
						Version: "some-version",
					},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{},
					},
					Layers: packit.Layers{Path: layersDir},
				}

				_, err := build(context)
				Expect(err).To(MatchError(ContainSubstring("some execution error")))
			})
		})
	})
}
