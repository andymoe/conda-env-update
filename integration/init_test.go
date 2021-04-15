package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

var (
	buildpack          string
	minicondaBuildpack string
	buildPlanBuildpack string

	buildpackInfo struct {
		Buildpack struct {
			ID   string
			Name string
		}
	}

	config struct {
		MinicondaBuildpack string `json:"miniconda"`
		BuildPlanBuildpack string `json:"build-plan"`
	}
)

func TestIntegration(t *testing.T) {
	Expect := NewWithT(t).Expect

	root, err := filepath.Abs("./..")
	Expect(err).ToNot(HaveOccurred())

	file, err := os.Open("../buildpack.toml")
	Expect(err).NotTo(HaveOccurred())

	_, err = toml.DecodeReader(file, &buildpackInfo)
	Expect(err).NotTo(HaveOccurred())
	Expect(file.Close()).To(Succeed())

	file, err = os.Open("../integration.json")
	Expect(err).NotTo(HaveOccurred())

	Expect(json.NewDecoder(file).Decode(&config)).To(Succeed())
	Expect(file.Close()).To(Succeed())

	buildpackStore := occam.NewBuildpackStore()

	buildpack, err = buildpackStore.Get.
		WithVersion("1.2.3").
		Execute(root)
	Expect(err).NotTo(HaveOccurred())

	minicondaBuildpack, err = buildpackStore.Get.
		WithVersion("1.2.3").
		Execute(config.MinicondaBuildpack)
	Expect(err).NotTo(HaveOccurred())

	buildPlanBuildpack, err = buildpackStore.Get.
		Execute(config.BuildPlanBuildpack)
	Expect(err).NotTo(HaveOccurred())

	SetDefaultEventuallyTimeout(5 * time.Second)

	suite := spec.New("Integration", spec.Report(report.Terminal{}), spec.Parallel())
	suite("Default", testDefault)
	suite("LayerReuse", testLayerReuse)
	suite.Run(t)
}
