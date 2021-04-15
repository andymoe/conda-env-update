package integration_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testLayerReuse(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually
		pack       occam.Pack
		docker     occam.Docker
	)

	it.Before(func() {
		pack = occam.NewPack()
		docker = occam.NewDocker()
	})

	context("when rebuilding an app without changes", func() {
		var (
			firstImage      occam.Image
			secondImage     occam.Image
			firstContainer  occam.Container
			secondContainer occam.Container
			name            string
			source          string
			imagesMap       map[string]interface{}
			containerMap    map[string]interface{}
		)

		it.Before(func() {
			var err error
			name, err = occam.RandomName()
			Expect(err).NotTo(HaveOccurred())

			imagesMap = map[string]interface{}{}
			containerMap = map[string]interface{}{}
		})

		it.After(func() {
			for containerID := range containerMap {
				Expect(docker.Container.Remove.Execute(containerID)).To(Succeed())
			}
			for imageID := range imagesMap {
				Expect(docker.Image.Remove.Execute(imageID)).To(Succeed())
			}

			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
			Expect(os.RemoveAll(source)).To(Succeed())
		})

		it.Focus("reuses the cached packages layer", func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "default_app"))
			Expect(err).NotTo(HaveOccurred())

			var logs fmt.Stringer
			firstImage, logs, err = pack.WithNoColor().Build.
				WithPullPolicy("never").
				WithBuildpacks(
					minicondaBuildpack,
					buildpack,
					buildPlanBuildpack,
				).
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String())

			imagesMap[firstImage.ID] = nil

			firstContainer, err = docker.Container.Run.
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				WithPublishAll().
				WithCommand("python app.py").
				Execute(firstImage.ID)
			Expect(err).NotTo(HaveOccurred())

			containerMap[firstContainer.ID] = nil

			Eventually(firstContainer).Should(Serve(ContainSubstring("Hello, world!")).OnPort(8080))

			secondImage, logs, err = pack.WithNoColor().Build.
				WithPullPolicy("never").
				WithBuildpacks(
					minicondaBuildpack,
					buildpack,
					buildPlanBuildpack,
				).
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String())

			imagesMap[secondImage.ID] = nil

			// Expect(logs).To(ContainLines(
			// 	MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, buildpackInfo.Buildpack.Name)),
			// 	ContainSubstring("Reusing cached layer"),
			// ))

			Expect(secondImage.Buildpacks[1].Key).To(Equal(buildpackInfo.Buildpack.ID))
			Expect(secondImage.Buildpacks[1].Layers["conda-env"].Metadata["built_at"]).To(Equal(firstImage.Buildpacks[1].Layers["conda-env"].Metadata["built_at"]))
			Expect(secondImage.Buildpacks[1].Layers["conda-env"].Metadata["cache_sha"]).To(Equal(firstImage.Buildpacks[1].Layers["conda-env"].Metadata["cache_sha"]))

			secondContainer, err = docker.Container.Run.
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				WithPublishAll().
				WithCommand("python app.py").
				Execute(secondImage.ID)
			Expect(err).NotTo(HaveOccurred())

			containerMap[secondContainer.ID] = nil

			Eventually(secondContainer).Should(Serve(ContainSubstring("Hello, world!")).OnPort(8080))
		})
	})
}
