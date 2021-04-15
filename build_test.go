package condaenvupdate_test

import (
	"os"
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

		now = time.Now()
		clock = chronos.NewClock(func() time.Time {
			return now
		})

		logger := scribe.NewLogger(os.Stdout)
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
			"built_at": clock.Now().Format(time.RFC3339Nano),
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

	})
}
