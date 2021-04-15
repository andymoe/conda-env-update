package condaenvupdate

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/paketo-buildpacks/packit/pexec"
)

const (
	LockfileName    = "package-list.txt"
	LockfileShaName = "lockfile-sha"
)

//go:generate faux --interface Executable --output fakes/executable.go
type Executable interface {
	Execute(pexec.Execution) error
}

//go:generate faux --interface Summer --output fakes/summer.go
type Summer interface {
	Sum(arg ...string) (string, error)
}

type CondaRunner struct {
	executable Executable
	summer     Summer
}

func NewCondaRunner(executable Executable, summer Summer) CondaRunner {
	return CondaRunner{
		executable: executable,
		summer:     summer,
	}
}

func (c CondaRunner) ShouldRun(workingDir string, metadata map[string]interface{}) (run bool, sha string, err error) {
	lockfilePath := filepath.Join(workingDir, LockfileName)
	_, err = os.Stat(lockfilePath)

	if errors.Is(err, os.ErrNotExist) {
		return true, "", nil
	}

	if err != nil {
		return false, "", err
	}

	// Maube we always rebuid on error here too? or do we panic in some way?
	updatedLockfileSha, err := c.summer.Sum(lockfilePath)
	if err != nil {
		return false, "", err
	}

	if updatedLockfileSha == metadata[LockfileShaName] {
		return false, updatedLockfileSha, nil
	}

	return true, updatedLockfileSha, nil
}

// TODO: make this work again and not be terrible :)
func (c CondaRunner) Execute(condaLayerPath string, condaCachePath string, workingDir string) error {
	// conda create <vendor args> (vendor dir exists) - no layer reuse
	vendorDirExists, err := fileExists(filepath.Join(workingDir, "vendor"))
	if err != nil {
		panic(err)
		return err
	}

	lockfileExists, err := fileExists(filepath.Join(workingDir, LockfileName))
	if err != nil {
		panic(err)
		return err
	}

	args := []string{
		"create",
		"--file", filepath.Join(workingDir, LockfileName),
		"--prefix", condaLayerPath,
		"--yes",
		"--quiet",
	}

	if vendorDirExists {
		vendorArgs := []string{
			"--channel", filepath.Join(workingDir, "vendor"),
			"--override-channels",
			"--offline",
		}
		args = append(args, vendorArgs...)
	}

	if !lockfileExists && !vendorDirExists {
		args = []string{
			"env",
			"update",
			"--prefix", condaLayerPath,
			"--file", filepath.Join(workingDir, "environment.yml"),
		}
	}

	err = c.executable.Execute(pexec.Execution{
		Args: args,
		Env:  append(os.Environ(), fmt.Sprintf("CONDA_PKGS_DIRS=%s", condaCachePath)),
	})

	if err != nil {
		panic(err)
	}
	return nil
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
