package pack_test

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/buildpack/imgutil"
	"github.com/buildpack/imgutil/fakes"
	"github.com/golang/mock/gomock"
	"github.com/heroku/color"
	"github.com/pkg/errors"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpack/pack"
	pubbldr "github.com/buildpack/pack/builder"
	"github.com/buildpack/pack/internal/api"
	"github.com/buildpack/pack/internal/blob"
	"github.com/buildpack/pack/internal/builder"
	"github.com/buildpack/pack/internal/buildpackage"
	"github.com/buildpack/pack/internal/dist"
	ifakes "github.com/buildpack/pack/internal/fakes"
	"github.com/buildpack/pack/internal/image"
	ilogging "github.com/buildpack/pack/internal/logging"
	"github.com/buildpack/pack/logging"
	h "github.com/buildpack/pack/testhelpers"
	"github.com/buildpack/pack/testmocks"
)

func TestCreateBuilder(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "create_builder", testCreateBuilder, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testCreateBuilder(t *testing.T, when spec.G, it spec.S) {
	when("#CreateBuilder", func() {
		var (
			mockController     *gomock.Controller
			mockDownloader     *testmocks.MockDownloader
			mockImageFactory   *testmocks.MockImageFactory
			mockImageFetcher   *testmocks.MockImageFetcher
			fakeBuildImage     *fakes.Image
			fakeRunImage       *fakes.Image
			fakeRunImageMirror *fakes.Image
			opts               pack.CreateBuilderOptions
			subject            *pack.Client
			logger             logging.Logger
			out                bytes.Buffer
			tmpDir             string
		)

		it.Before(func() {
			logger = ilogging.NewLogWithWriters(&out, &out)
			mockController = gomock.NewController(t)
			mockDownloader = testmocks.NewMockDownloader(mockController)
			mockImageFetcher = testmocks.NewMockImageFetcher(mockController)
			mockImageFactory = testmocks.NewMockImageFactory(mockController)

			fakeBuildImage = fakes.NewImage("some/build-image", "", nil)
			h.AssertNil(t, fakeBuildImage.SetLabel("io.buildpacks.stack.id", "some.stack.id"))
			h.AssertNil(t, fakeBuildImage.SetLabel("io.buildpacks.stack.mixins", `["mixinX", "build:mixinY"]`))
			h.AssertNil(t, fakeBuildImage.SetEnv("CNB_USER_ID", "1234"))
			h.AssertNil(t, fakeBuildImage.SetEnv("CNB_GROUP_ID", "4321"))

			fakeRunImage = fakes.NewImage("some/run-image", "", nil)
			h.AssertNil(t, fakeRunImage.SetLabel("io.buildpacks.stack.id", "some.stack.id"))

			fakeRunImageMirror = fakes.NewImage("localhost:5000/some/run-image", "", nil)
			h.AssertNil(t, fakeRunImageMirror.SetLabel("io.buildpacks.stack.id", "some.stack.id"))
			
			mockDownloader.EXPECT().Download(gomock.Any(), "https://example.fake/bp-one.tgz").Return(blob.NewBlob(filepath.Join("testdata", "buildpack")), nil).AnyTimes()
			mockDownloader.EXPECT().Download(gomock.Any(), "some/buildpack/dir").Return(blob.NewBlob(filepath.Join("testdata", "buildpack")), nil).AnyTimes()
			mockDownloader.EXPECT().Download(gomock.Any(), "file:///some-lifecycle").Return(blob.NewBlob(filepath.Join("testdata", "lifecycle")), nil).AnyTimes()
			mockDownloader.EXPECT().Download(gomock.Any(), "file:///some-lifecycle-platform-0-1").Return(blob.NewBlob(filepath.Join("testdata", "lifecycle-platform-0.1")), nil).AnyTimes()

			var err error
			subject, err = pack.NewClient(
				pack.WithLogger(logger),
				pack.WithDownloader(mockDownloader),
				pack.WithImageFactory(mockImageFactory),
				pack.WithFetcher(mockImageFetcher),
			)
			h.AssertNil(t, err)

			opts = pack.CreateBuilderOptions{
				BuilderName: "some/builder",
				Config: pubbldr.Config{
					Description: "Some description",
					Buildpacks: []pubbldr.BuildpackConfig{
						{
							BuildpackInfo: dist.BuildpackInfo{ID: "bp.one", Version: "1.2.3"},
							URI:           "https://example.fake/bp-one.tgz",
						},
					},
					Order: []dist.OrderEntry{{
						Group: []dist.BuildpackRef{
							{BuildpackInfo: dist.BuildpackInfo{ID: "bp.one", Version: "1.2.3"}, Optional: false},
						}},
					},
					Stack: pubbldr.StackConfig{
						ID:              "some.stack.id",
						BuildImage:      "some/build-image",
						RunImage:        "some/run-image",
						RunImageMirrors: []string{"localhost:5000/some/run-image"},
					},
					Lifecycle: pubbldr.LifecycleConfig{URI: "file:///some-lifecycle"},
				},
				Publish: false,
				NoPull:  false,
			}

			tmpDir, err = ioutil.TempDir("", "create-builder-test")
			h.AssertNil(t, err)
		})

		it.After(func() {
			mockController.Finish()
			h.AssertNil(t, os.RemoveAll(tmpDir))
		})

		var prepareFetcherWithRunImages = func() {
			mockImageFetcher.EXPECT().Fetch(gomock.Any(), "some/run-image", true, gomock.Any()).Return(fakeRunImage, nil).AnyTimes()
			mockImageFetcher.EXPECT().Fetch(gomock.Any(), "localhost:5000/some/run-image", true, gomock.Any()).Return(fakeRunImageMirror, nil).AnyTimes()
		}

		var prepareFetcherWithBuildImage = func() {
			mockImageFetcher.EXPECT().Fetch(gomock.Any(), "some/build-image", true, gomock.Any()).Return(fakeBuildImage, nil).AnyTimes()
		}

		var configureBuilderWithLifecycleAPIv0_1 = func() {
			opts.Config.Lifecycle = pubbldr.LifecycleConfig{URI: "file:///some-lifecycle-platform-0-1"}
		}

		var successfullyCreateBuilder = func() *builder.Builder {
			t.Helper()

			err := subject.CreateBuilder(context.TODO(), opts)
			h.AssertNil(t, err)

			h.AssertEq(t, fakeBuildImage.IsSaved(), true)
			bldr, err := builder.FromImage(fakeBuildImage)
			h.AssertNil(t, err)

			return bldr
		}

		when("validating the builder config", func() {
			it("should fail when the stack ID is empty", func() {
				opts.Config.Stack.ID = ""

				err := subject.CreateBuilder(context.TODO(), opts)

				h.AssertError(t, err, "stack.id is required")
			})

			it("should fail when the stack ID from the builder config does not match the stack ID from the build image", func() {
				mockImageFetcher.EXPECT().Fetch(gomock.Any(), "some/build-image", true, true).Return(fakeBuildImage, nil)
				h.AssertNil(t, fakeBuildImage.SetLabel("io.buildpacks.stack.id", "other.stack.id"))
				prepareFetcherWithRunImages()

				err := subject.CreateBuilder(context.TODO(), opts)

				h.AssertError(t, err, "stack 'some.stack.id' from builder config is incompatible with stack 'other.stack.id' from build image")
			})

			it("should fail when the build image is empty", func() {
				opts.Config.Stack.BuildImage = ""

				err := subject.CreateBuilder(context.TODO(), opts)

				h.AssertError(t, err, "stack.build-image is required")
			})

			it("should fail when the run image is empty", func() {
				opts.Config.Stack.RunImage = ""

				err := subject.CreateBuilder(context.TODO(), opts)

				h.AssertError(t, err, "stack.run-image is required")
			})

			it("should fail when lifecycle version is not a semver", func() {
				prepareFetcherWithBuildImage()
				prepareFetcherWithRunImages()
				opts.Config.Lifecycle.URI = ""
				opts.Config.Lifecycle.Version = "not-semver"

				err := subject.CreateBuilder(context.TODO(), opts)

				h.AssertError(t, err, "'lifecycle.version' must be a valid semver")
			})

			it("should fail when both lifecycle version and uri are present", func() {
				prepareFetcherWithBuildImage()
				prepareFetcherWithRunImages()
				opts.Config.Lifecycle.URI = "file://some-lifecycle"
				opts.Config.Lifecycle.Version = "something"

				err := subject.CreateBuilder(context.TODO(), opts)

				h.AssertError(t, err, "'lifecycle' can only declare 'version' or 'uri', not both")
			})

			it("should fail when buildpack ID does not match downloaded buildpack", func() {
				prepareFetcherWithBuildImage()
				prepareFetcherWithRunImages()
				opts.Config.Buildpacks[0].ID = "does.not.match"

				err := subject.CreateBuilder(context.TODO(), opts)

				h.AssertError(t, err, "buildpack from URI 'https://example.fake/bp-one.tgz' has ID 'bp.one' which does not match ID 'does.not.match' from builder config")
			})

			it("should fail when buildpack version does not match downloaded buildpack", func() {
				prepareFetcherWithBuildImage()
				prepareFetcherWithRunImages()
				opts.Config.Buildpacks[0].Version = "0.0.0"

				err := subject.CreateBuilder(context.TODO(), opts)

				h.AssertError(t, err, "buildpack from URI 'https://example.fake/bp-one.tgz' has version '1.2.3' which does not match version '0.0.0' from builder config")
			})
		})

		when("validating the run image config", func() {
			it("should fail when the stack ID from the builder config does not match the stack ID from the run image", func() {
				prepareFetcherWithRunImages()
				h.AssertNil(t, fakeRunImage.SetLabel("io.buildpacks.stack.id", "other.stack.id"))

				err := subject.CreateBuilder(context.TODO(), opts)

				h.AssertError(t, err, "stack 'some.stack.id' from builder config is incompatible with stack 'other.stack.id' from run image 'some/run-image'")
			})

			it("should fail when the stack ID from the builder config does not match the stack ID from the run image mirrors", func() {
				prepareFetcherWithRunImages()
				h.AssertNil(t, fakeRunImageMirror.SetLabel("io.buildpacks.stack.id", "other.stack.id"))

				err := subject.CreateBuilder(context.TODO(), opts)

				h.AssertError(t, err, "stack 'some.stack.id' from builder config is incompatible with stack 'other.stack.id' from run image 'localhost:5000/some/run-image'")
			})

			it("should warn when the run image cannot be found", func() {
				mockImageFetcher.EXPECT().Fetch(gomock.Any(), "some/build-image", true, true).Return(fakeBuildImage, nil)

				mockImageFetcher.EXPECT().Fetch(gomock.Any(), "some/run-image", false, false).Return(nil, errors.Wrap(image.ErrNotFound, "yikes!"))
				mockImageFetcher.EXPECT().Fetch(gomock.Any(), "some/run-image", true, false).Return(nil, errors.Wrap(image.ErrNotFound, "yikes!"))

				mockImageFetcher.EXPECT().Fetch(gomock.Any(), "localhost:5000/some/run-image", false, false).Return(nil, errors.Wrap(image.ErrNotFound, "yikes!"))
				mockImageFetcher.EXPECT().Fetch(gomock.Any(), "localhost:5000/some/run-image", true, false).Return(nil, errors.Wrap(image.ErrNotFound, "yikes!"))

				err := subject.CreateBuilder(context.TODO(), opts)
				h.AssertNil(t, err)

				h.AssertContains(t, out.String(), "Warning: run image 'some/run-image' is not accessible")
			})

			when("publish is true", func() {
				it("should only try to validate the remote run image", func() {
					mockImageFetcher.EXPECT().Fetch(gomock.Any(), "some/build-image", true, gomock.Any()).Times(0)
					mockImageFetcher.EXPECT().Fetch(gomock.Any(), "some/run-image", true, gomock.Any()).Times(0)
					mockImageFetcher.EXPECT().Fetch(gomock.Any(), "localhost:5000/some/run-image", true, gomock.Any()).Times(0)

					mockImageFetcher.EXPECT().Fetch(gomock.Any(), "some/build-image", false, gomock.Any()).Return(fakeBuildImage, nil)
					mockImageFetcher.EXPECT().Fetch(gomock.Any(), "some/run-image", false, gomock.Any()).Return(fakeRunImage, nil)
					mockImageFetcher.EXPECT().Fetch(gomock.Any(), "localhost:5000/some/run-image", false, gomock.Any()).Return(fakeRunImageMirror, nil)

					opts.Publish = true

					err := subject.CreateBuilder(context.TODO(), opts)
					h.AssertNil(t, err)
				})
			})
		})

		when("only lifecycle version is provided", func() {
			it("should download from predetermined uri", func() {
				prepareFetcherWithBuildImage()
				prepareFetcherWithRunImages()
				opts.Config.Lifecycle.URI = ""
				opts.Config.Lifecycle.Version = "3.4.5"

				mockDownloader.EXPECT().Download(
					gomock.Any(),
					"https://github.com/buildpack/lifecycle/releases/download/v3.4.5/lifecycle-v3.4.5+linux.x86-64.tgz",
				).Return(
					blob.NewBlob(filepath.Join("testdata", "lifecycle")), nil,
				)

				err := subject.CreateBuilder(context.TODO(), opts)
				h.AssertNil(t, err)
			})
		})

		when("no lifecycle version or URI is provided", func() {
			it("should download default lifecycle", func() {
				prepareFetcherWithBuildImage()
				prepareFetcherWithRunImages()
				opts.Config.Lifecycle.URI = ""
				opts.Config.Lifecycle.Version = ""

				mockDownloader.EXPECT().Download(
					gomock.Any(),
					fmt.Sprintf(
						"https://github.com/buildpack/lifecycle/releases/download/v%s/lifecycle-v%s+linux.x86-64.tgz",
						builder.DefaultLifecycleVersion,
						builder.DefaultLifecycleVersion,
					),
				).Return(
					blob.NewBlob(filepath.Join("testdata", "lifecycle")), nil,
				)

				err := subject.CreateBuilder(context.TODO(), opts)
				h.AssertNil(t, err)
			})
		})

		when("buildpack mixins are not satisfied", func() {
			it("should return an error", func() {
				prepareFetcherWithBuildImage()
				prepareFetcherWithRunImages()
				h.AssertNil(t, fakeBuildImage.SetLabel("io.buildpacks.stack.mixins", ""))

				err := subject.CreateBuilder(context.TODO(), opts)

				h.AssertError(t, err, "validating buildpacks: buildpack 'bp.one@1.2.3' requires missing mixin(s): build:mixinY, mixinX")
			})
		})

		when("creation succeeds", func() {
			it("should set basic metadata", func() {
				prepareFetcherWithBuildImage()
				prepareFetcherWithRunImages()

				bldr := successfullyCreateBuilder()

				h.AssertEq(t, bldr.Name(), "some/builder")
				h.AssertEq(t, bldr.Description(), "Some description")
				h.AssertEq(t, bldr.UID, 1234)
				h.AssertEq(t, bldr.GID, 4321)
				h.AssertEq(t, bldr.StackID, "some.stack.id")
			})

			it("should set buildpack and order metadata", func() {
				prepareFetcherWithBuildImage()
				prepareFetcherWithRunImages()

				bldr := successfullyCreateBuilder()

				bpInfo := dist.BuildpackInfo{
					ID:      "bp.one",
					Version: "1.2.3",
				}
				h.AssertEq(t, bldr.Buildpacks(), []builder.BuildpackMetadata{{
					BuildpackInfo: bpInfo,
					Latest:        true,
				}})
				h.AssertEq(t, bldr.Order(), dist.Order{{
					Group: []dist.BuildpackRef{{
						BuildpackInfo: bpInfo,
						Optional:      false,
					}},
				}})
			})

			it("should embed the lifecycle", func() {
				prepareFetcherWithBuildImage()
				prepareFetcherWithRunImages()

				bldr := successfullyCreateBuilder()

				h.AssertEq(t, bldr.LifecycleDescriptor().Info.Version.String(), "3.4.5")
				h.AssertEq(t, bldr.LifecycleDescriptor().API.PlatformVersion.String(), "0.2")

				layerTar, err := fakeBuildImage.FindLayerWithPath("/cnb/lifecycle")
				h.AssertNil(t, err)
				assertTarHasFile(t, layerTar, "/cnb/lifecycle/detector")
				assertTarHasFile(t, layerTar, "/cnb/lifecycle/restorer")
				assertTarHasFile(t, layerTar, "/cnb/lifecycle/analyzer")
				assertTarHasFile(t, layerTar, "/cnb/lifecycle/builder")
				assertTarHasFile(t, layerTar, "/cnb/lifecycle/exporter")
				assertTarHasFile(t, layerTar, "/cnb/lifecycle/launcher")
			})
		})

		when("creation succeeds for platform API < 0.2", func() {
			it("should set basic metadata", func() {
				configureBuilderWithLifecycleAPIv0_1()
				prepareFetcherWithBuildImage()
				prepareFetcherWithRunImages()

				bldr := successfullyCreateBuilder()

				h.AssertEq(t, bldr.Name(), "some/builder")
				h.AssertEq(t, bldr.Description(), "Some description")
				h.AssertEq(t, bldr.UID, 1234)
				h.AssertEq(t, bldr.GID, 4321)
				h.AssertEq(t, bldr.StackID, "some.stack.id")
			})

			it("should set buildpack and order metadata", func() {
				configureBuilderWithLifecycleAPIv0_1()
				prepareFetcherWithBuildImage()
				prepareFetcherWithRunImages()

				bldr := successfullyCreateBuilder()

				bpInfo := dist.BuildpackInfo{
					ID:      "bp.one",
					Version: "1.2.3",
				}
				h.AssertEq(t, bldr.Buildpacks(), []builder.BuildpackMetadata{{
					BuildpackInfo: bpInfo,
					Latest:        true,
				}})
				h.AssertEq(t, bldr.Order(), dist.Order{{
					Group: []dist.BuildpackRef{{
						BuildpackInfo: bpInfo,
						Optional:      false,
					}},
				}})
			})

			it("should embed the lifecycle", func() {
				configureBuilderWithLifecycleAPIv0_1()
				prepareFetcherWithBuildImage()
				prepareFetcherWithRunImages()

				bldr := successfullyCreateBuilder()

				h.AssertEq(t, bldr.LifecycleDescriptor().Info.Version.String(), "3.4.5")
				h.AssertEq(t, bldr.LifecycleDescriptor().API.PlatformVersion.String(), "0.1")

				layerTar, err := fakeBuildImage.FindLayerWithPath("/cnb/lifecycle")
				h.AssertNil(t, err)
				assertTarHasFile(t, layerTar, "/cnb/lifecycle/detector")
				assertTarHasFile(t, layerTar, "/cnb/lifecycle/restorer")
				assertTarHasFile(t, layerTar, "/cnb/lifecycle/analyzer")
				assertTarHasFile(t, layerTar, "/cnb/lifecycle/builder")
				assertTarHasFile(t, layerTar, "/cnb/lifecycle/exporter")
				assertTarHasFile(t, layerTar, "/cnb/lifecycle/cacher")
				assertTarHasFile(t, layerTar, "/cnb/lifecycle/launcher")
			})
		})

		when("windows", func() {
			it.Before(func() {
				h.SkipIf(t, runtime.GOOS != "windows", "Skipped on non-windows")
			})

			it("disallows directory-based buildpacks", func() {
				opts.Config.Buildpacks[0].URI = "testdata/buildpack"

				err := subject.CreateBuilder(context.TODO(), opts)
				h.AssertError(t,
					err,
					"buildpack 'testdata/buildpack': directory-based buildpacks are not currently supported on Windows")
			})
		})

		when("is posix", func() {
			it.Before(func() {
				h.SkipIf(t, runtime.GOOS == "windows", "Skipped on windows")
			})

			it("supports directory buildpacks", func() {
				prepareFetcherWithBuildImage()
				prepareFetcherWithRunImages()
				opts.Config.Buildpacks[0].URI = "some/buildpack/dir"

				err := subject.CreateBuilder(context.TODO(), opts)
				h.AssertNil(t, err)
			})
		})

		when("packages", func() {
			createBuildpack := func(descriptor dist.BuildpackDescriptor) string {
				bp, err := ifakes.NewFakeBuildpackBlob(descriptor, 0644)
				h.AssertNil(t, err)
				url := fmt.Sprintf("https://example.com/bp.%s.tgz", h.RandString(12))
				mockDownloader.EXPECT().Download(gomock.Any(), url).Return(bp, nil).AnyTimes()
				return url
			}

			when("package image lives in registry", func() {
				var nestedPackage *fakes.Image

				it.Before(func() {
					nestedPackage = fakes.NewImage("nested/package-"+h.RandString(12), "", nil)
					mockImageFactory.EXPECT().NewImage(nestedPackage.Name(), false).Return(nestedPackage, nil)

					bpd := dist.BuildpackDescriptor{
						API:    api.MustParse("0.2"),
						Info:   dist.BuildpackInfo{ID: "bp.nested", Version: "2.3.4"},
						Stacks: []dist.Stack{{ID: "some.stack.id"}},
					}

					h.AssertNil(t, subject.CreatePackage(context.TODO(), pack.CreatePackageOptions{
						Name: nestedPackage.Name(),
						Config: buildpackage.Config{
							Default:    bpd.Info,
							Buildpacks: []dist.BuildpackURI{{URI: createBuildpack(bpd)}},
							Stacks:     bpd.Stacks,
						},
						Publish: true,
					}))
				})

				shouldCallImageFetcherWith := func(demon, pull bool) {
					mockImageFetcher.EXPECT().Fetch(gomock.Any(), nestedPackage.Name(), demon, pull).Return(nestedPackage, nil)
				}

				shouldNotFindImageWhenCallingImageFetcherWith := func(demon, pull bool) {
					mockImageFetcher.EXPECT().Fetch(gomock.Any(), nestedPackage.Name(), demon, pull).Return(nil, image.ErrNotFound)
				}

				shouldCreateLocalImage := func() imgutil.Image {
					img := fakes.NewImage("some/package"+h.RandString(12), "", nil)
					mockImageFactory.EXPECT().NewImage(img.Name(), true).Return(img, nil)
					return img
				}

				shouldCreateRemoteImage := func() *fakes.Image {
					img := fakes.NewImage("some/package"+h.RandString(12), "", nil)
					mockImageFactory.EXPECT().NewImage(img.Name(), false).Return(img, nil)
					return img
				}

				when("publish=false and no-pull=false", func() {
					it("should pull and use local image", func() {
						shouldCallImageFetcherWith(true, true)
						localImage := shouldCreateLocalImage()
						h.AssertNil(t, subject.CreatePackage(context.TODO(), pack.CreatePackageOptions{
							Name: localImage.Name(),
							Config: buildpackage.Config{
								Default:  dist.BuildpackInfo{ID: "bp.nested", Version: "2.3.4"},
								Packages: []dist.ImageRef{{Ref: nestedPackage.Name()}},
								Stacks:   []dist.Stack{{ID: "some.stack.id"}},
							},
							Publish: false,
							NoPull:  false,
						}))
					})
				})

				when("publish=true and no-pull=false", func() {
					it("should use remote image", func() {
						shouldCallImageFetcherWith(false, true)
						packageImage := shouldCreateRemoteImage()

						h.AssertNil(t, subject.CreatePackage(context.TODO(), pack.CreatePackageOptions{
							Name: packageImage.Name(),
							Config: buildpackage.Config{
								Default:  dist.BuildpackInfo{ID: "bp.nested", Version: "2.3.4"},
								Packages: []dist.ImageRef{{Ref: nestedPackage.Name()}},
								Stacks:   []dist.Stack{{ID: "some.stack.id"}},
							},
							Publish: true,
							NoPull:  false,
						}))
					})
				})

				when("publish=true and no-pull=true", func() {
					it("should not pull image and push to registry", func() {
						shouldCallImageFetcherWith(false, false)
						packageImage := shouldCreateRemoteImage()

						h.AssertNil(t, subject.CreatePackage(context.TODO(), pack.CreatePackageOptions{
							Name: packageImage.Name(),
							Config: buildpackage.Config{
								Default:  dist.BuildpackInfo{ID: "bp.nested", Version: "2.3.4"},
								Packages: []dist.ImageRef{{Ref: nestedPackage.Name()}},
								Stacks:   []dist.Stack{{ID: "some.stack.id"}},
							},
							Publish: true,
							NoPull:  true,
						}))
					})
				})

				when("publish=false no-pull=true and there is no local image", func() {
					it("should fail without trying to retrieve image from registry", func() {
						shouldNotFindImageWhenCallingImageFetcherWith(true, false)

						h.AssertError(t, subject.CreatePackage(context.TODO(), pack.CreatePackageOptions{
							Name: "some/package",
							Config: buildpackage.Config{
								Default:  dist.BuildpackInfo{ID: "bp.nested", Version: "2.3.4"},
								Packages: []dist.ImageRef{{Ref: nestedPackage.Name()}},
								Stacks:   []dist.Stack{{ID: "some.stack.id"}},
							},
							Publish: false,
							NoPull:  true,
						}), "not found")
					})
				})
			})

			when("package image is not a valid package", func() {
				it("should error", func() {
					notPackageImage := fakes.NewImage("not/package", "", nil)
					mockImageFetcher.EXPECT().Fetch(gomock.Any(), notPackageImage.Name(), true, true).Return(notPackageImage, nil)

					h.AssertError(t, subject.CreatePackage(context.TODO(), pack.CreatePackageOptions{
						Name: "",
						Config: buildpackage.Config{
							Default:  dist.BuildpackInfo{ID: "bp.1.id", Version: "bp.1.version"},
							Packages: []dist.ImageRef{{Ref: notPackageImage.Name()}},
							Stacks:   []dist.Stack{{ID: "stack.1.id"}},
						},
						Publish: false,
						NoPull:  false,
					}), "label 'io.buildpacks.buildpack.layers' not present on package 'not/package'")
				})
			})
		})
	})
}

func assertTarHasFile(t *testing.T, tarFile, path string) {
	t.Helper()

	exist := tarHasFile(t, tarFile, path)
	if !exist {
		t.Fatalf("%s does not exist in %s", path, tarFile)
	}
}

func tarHasFile(t *testing.T, tarFile, path string) (exist bool) {
	t.Helper()

	r, err := os.Open(tarFile)
	h.AssertNil(t, err)
	defer r.Close()

	tr := tar.NewReader(r)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		h.AssertNil(t, err)

		if header.Name == path {
			return true
		}
	}

	return false
}
