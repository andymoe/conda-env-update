package main

import (
	"os"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/fs"
	"github.com/paketo-buildpacks/packit/pexec"
	"github.com/paketo-buildpacks/packit/scribe"
	condaenvupdate "github.com/paketo-community/conda-env-update"
)

func main() {
	logger := scribe.NewLogger(os.Stdout)
	summer := fs.NewChecksumCalculator()
	condaRunner := condaenvupdate.NewCondaRunner(pexec.NewExecutable("conda"), summer)

	packit.Run(condaenvupdate.Detect(), condaenvupdate.Build(condaRunner, logger, chronos.DefaultClock))
}
