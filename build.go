package condaenvupdate

import (
	"time"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/scribe"
)

//go:generate faux --interface Runner --output fakes/runner.go
type Runner interface {
	Execute(condaEnvPath string, condaCachePath string, workingDir string) error
}

// todo:
// layer caching logic - look at Determining Node Modules Layer Resue section of nodjs docks
// paketo.io website for some insirpation of what this logic might look like. May just
// see if the lockfiles match as a starting point to determin layer reuse.
// dynamic layer type settings (candaLayer.Launch boolean comes form the buildpack plan)
// Logging
// test for all this...
func Build(runner Runner, logger scribe.Logger, clock chronos.Clock) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)

		// Get conda-env-layer
		condaLayer, err := context.Layers.Get("conda-env")
		if err != nil {
			return packit.BuildResult{}, err
		}

		condaCacheLayer, err := context.Layers.Get("conda-env-cache")
		if err != nil {
			return packit.BuildResult{}, err
		}

		condaCacheLayer.Cache = true

		// Cache check and potential reuse

		condaLayer, err = condaLayer.Reset()
		if err != nil {
			return packit.BuildResult{}, err
		}

		condaLayer.Launch = true

		// if no vendor, run conda clean -pt TODO: Investigate
		err = runner.Execute(condaLayer.Path, condaCacheLayer.Path, context.WorkingDir)
		if err != nil {
			return packit.BuildResult{}, err
		}

		condaLayer.Metadata = map[string]interface{}{
			"built_at": clock.Now().Format(time.RFC3339Nano),
		}

		return packit.BuildResult{
			Layers: []packit.Layer{
				condaLayer,
				condaCacheLayer,
			},
		}, nil
	}
}
