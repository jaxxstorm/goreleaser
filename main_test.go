package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func init() {
	_ = os.Unsetenv("GITHUB_TOKEN")
	_ = os.Unsetenv("GITLAB_TOKEN")
}

func TestReleaseProject(t *testing.T) {
	_, back := setup(t)
	defer back()
	_, err := releaseProject(testParams())
	assert.NoError(t, err)
}

func TestCheckConfig(t *testing.T) {
	_, back := setup(t)
	defer back()
	_, err := checkConfig(testParams().Config)
	assert.NoError(t, err)
}

func TestCheckConfigFails(t *testing.T) {
	_, back := setup(t)
	defer back()
	var filename = "fail.yaml"
	assert.NoError(t, ioutil.WriteFile(filename, []byte("nope: 1"), 0644))
	_, err := checkConfig(filename)
	assert.Error(t, err)
}

func TestReleaseProjectSkipPublish(t *testing.T) {
	_, back := setup(t)
	defer back()
	params := testParams()
	params.Snapshot = true
	params.SkipPublish = true
	_, err := releaseProject(params)
	assert.NoError(t, err)
}

func TestConfigFileIsSetAndDontExist(t *testing.T) {
	_, back := setup(t)
	defer back()
	params := testParams()
	params.Config = "/this/wont/exist"
	_, err := releaseProject(params)
	assert.Error(t, err)
}

func TestConfigFlagNotSetButExists(t *testing.T) {
	for _, name := range []string{
		".goreleaser.yml",
		".goreleaser.yaml",
		"goreleaser.yml",
		"goreleaser.yaml",
	} {
		t.Run(name, func(t *testing.T) {
			folder, back := setup(t)
			defer back()
			err := os.Rename(
				filepath.Join(folder, "goreleaser.yml"),
				filepath.Join(folder, name),
			)
			assert.NoError(t, err)
			proj, err := loadConfig("")
			assert.NoError(t, err)
			assert.NotEqual(t, config.Project{}, proj)
		})
	}
}

func TestConfigFileDoesntExist(t *testing.T) {
	folder, back := setup(t)
	defer back()
	err := os.Remove(filepath.Join(folder, "goreleaser.yml"))
	assert.NoError(t, err)
	proj, err := loadConfig("")
	assert.NoError(t, err)
	assert.Equal(t, config.Project{}, proj)
}

func TestReleaseNotesFileDontExist(t *testing.T) {
	_, back := setup(t)
	defer back()
	params := testParams()
	params.ReleaseNotes = "/this/also/wont/exist"
	_, err := releaseProject(params)
	assert.Error(t, err)
}

func TestCustomReleaseNotesFile(t *testing.T) {
	_, back := setup(t)
	defer back()
	releaseNotes, err := ioutil.TempFile("", "")
	assert.NoError(t, err)
	createFile(t, releaseNotes.Name(), "nothing important at all")
	var params = testParams()
	params.ReleaseNotes = releaseNotes.Name()
	_, err = releaseProject(params)
	assert.NoError(t, err)
}

func TestCustomReleaseHeaderFileDontExist(t *testing.T) {
	_, back := setup(t)
	defer back()
	params := testParams()
	params.ReleaseHeader = "/header/that/dont/exist"
	params.Snapshot = false
	_, err := releaseProject(params)
	assert.Error(t, err)
}

func TestCustomReleaseHeaderFile(t *testing.T) {
	_, back := setup(t)
	defer back()
	releaseHeader, err := ioutil.TempFile("", "")
	assert.NoError(t, err)
	createFile(t, releaseHeader.Name(), "some release header")
	params := testParams()
	params.ReleaseHeader = releaseHeader.Name()
	params.Snapshot = false
	params.SkipPublish = true
	_, err = releaseProject(params)
	assert.NoError(t, err)
}

func TestCustomReleaseFooterFileDontExist(t *testing.T) {
	_, back := setup(t)
	defer back()
	params := testParams()
	params.ReleaseFooter = "/footer/that/dont/exist"
	params.Snapshot = false
	_, err := releaseProject(params)
	assert.Error(t, err)
}

func TestCustomReleaseFooterFile(t *testing.T) {
	_, back := setup(t)
	defer back()
	releaseFooter, err := ioutil.TempFile("", "")
	assert.NoError(t, err)
	createFile(t, releaseFooter.Name(), "some release footer")
	params := testParams()
	params.ReleaseFooter = releaseFooter.Name()
	params.Snapshot = false
	params.SkipPublish = true
	_, err = releaseProject(params)
	assert.NoError(t, err)
}

func TestBrokenPipe(t *testing.T) {
	_, back := setup(t)
	defer back()
	createFile(t, "main.go", "not a valid go file")
	_, err := releaseProject(testParams())
	assert.Error(t, err)
}

func TestInitProject(t *testing.T) {
	_, back := setup(t)
	defer back()
	var filename = "test_goreleaser.yml"
	assert.NoError(t, initProject(filename))

	file, err := os.Open(filename)
	assert.NoError(t, err)
	out, err := ioutil.ReadAll(file)
	assert.NoError(t, err)

	assert.NoError(t, yaml.Unmarshal(out, &config.Project{}))
}

func TestInitProjectFileExist(t *testing.T) {
	_, back := setup(t)
	defer back()
	var filename = "test_goreleaser.yml"
	createFile(t, filename, "")
	assert.Error(t, initProject(filename))
}

func TestInitProjectDefaultPipeFails(t *testing.T) {
	folder, back := setup(t)
	defer back()
	var filename = "test_goreleaser.yml"
	assert.NoError(t, os.Chmod(folder, 0000))
	assert.EqualError(t, initProject(filename), `stat test_goreleaser.yml: permission denied`)
}

func testParams() releaseOptions {
	return releaseOptions{
		Parallelism: 4,
		Snapshot:    true,
		Timeout:     time.Minute,
	}
}

func setup(t *testing.T) (current string, back func()) {
	folder, err := ioutil.TempDir("", "")
	assert.NoError(t, err)
	previous, err := os.Getwd()
	assert.NoError(t, err)
	assert.NoError(t, os.Chdir(folder))
	createGoreleaserYaml(t)
	createMainGo(t)
	goModInit(t)
	testlib.GitInit(t)
	testlib.GitAdd(t)
	testlib.GitCommit(t, "asdf")
	testlib.GitTag(t, "v0.0.1")
	testlib.GitCommit(t, "asas89d")
	testlib.GitCommit(t, "assssf")
	testlib.GitCommit(t, "assd")
	testlib.GitTag(t, "v0.0.2")
	testlib.GitRemoteAdd(t, "git@github.com:goreleaser/fake.git")
	return folder, func() {
		assert.NoError(t, os.Chdir(previous))
	}
}

func createFile(t *testing.T, filename, contents string) {
	assert.NoError(t, ioutil.WriteFile(filename, []byte(contents), 0644))
}

func createMainGo(t *testing.T) {
	createFile(t, "main.go", "package main\nfunc main() {println(0)}")
}

func goModInit(t *testing.T) {
	createFile(t, "go.mod", `module foo

go 1.14
`)
}

func createGoreleaserYaml(t *testing.T) {
	var yaml = `build:
  binary: fake
  goos:
    - linux
  goarch:
    - amd64
release:
  github:
    owner: goreleaser
    name: fake
`
	createFile(t, "goreleaser.yml", yaml)
}

func TestVersion(t *testing.T) {
	for name, tt := range map[string]struct {
		version, commit, date, builtBy string
		out                            string
	}{
		"all empty": {
			out: "version: ",
		},
		"complete": {
			version: "1.2.3",
			date:    "12/12/12",
			commit:  "aaaa",
			builtBy: "me",
			out:     "version: 1.2.3\ncommit: aaaa\nbuilt at: 12/12/12\nbuilt by: me",
		},
		"only version": {
			version: "1.2.3",
			out:     "version: 1.2.3",
		},
		"version and date": {
			version: "1.2.3",
			date:    "12/12/12",
			out:     "version: 1.2.3\nbuilt at: 12/12/12",
		},
		"version, date, built by": {
			version: "1.2.3",
			date:    "12/12/12",
			builtBy: "me",
			out:     "version: 1.2.3\nbuilt at: 12/12/12\nbuilt by: me",
		},
	} {
		tt := tt
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tt.out, buildVersion(tt.version, tt.commit, tt.date, tt.builtBy))
		})
	}
}
