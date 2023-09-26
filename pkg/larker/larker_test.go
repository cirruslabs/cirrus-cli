package larker_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/cirruslabs/cirrus-cli/pkg/larker"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs/local"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"text/template"
	"time"
)

func possiblySkip(t *testing.T) {
	if _, ok := os.LookupEnv("CIRRUS_INTERNAL_RUN_GITHUB_API_TESTS"); !ok {
		t.Skip("not running a test that might consume GitHub API rate limit")
	}
}

// TestSimpleTask ensures that .cirrus.star is able to generate the simplest possible configuration.
func TestSimpleTask(t *testing.T) {
	validateExpected(t, "testdata/simple-task")
}

// TestSugarCoatedTask ensures that .cirrus.star is able to use imported functions for task generation.
func TestSugarCoatedTask(t *testing.T) {
	validateExpected(t, "testdata/sugar-coated-task")
}

// TestNoTasks ensures that .cirrus.star that return empty list still generates a YAML
// that can be joined with another YAML using only string concatenation.
func TestNoTasks(t *testing.T) {
	validateExpected(t, "testdata/no-tasks")
}

func TestNoCtxMain(t *testing.T) {
	validateExpected(t, "testdata/no-ctx")
}

// TestMainReturnsDict ensures that we support overrides represented as dictionary.
func TestMainReturnsDict(t *testing.T) {
	validateExpected(t, "testdata/main-returns-dict")
}

// TestMainReturnsList ensures that we support list of tasks
// lists of tasks will produce repeated keys in the resulting YAML.
func TestMainReturnsList(t *testing.T) {
	dir := testutil.TempDirPopulatedWith(t, "testdata/main-returns-list")
	// Avoid validateExpect (will try to parse YAML without accepting repeated keys)
	resultConfig := loadStarlarkConfig(t, dir)
	expectedConfig := loadExpectedConfig(t, dir)
	assert.Equal(t, expectedConfig, resultConfig)
}

// For feature parity between Cirrus YAML and Starlark configs,
// accepting repeated keys is required (not the default behaviour of YAML).
// A solution for that is to accept a list of tuples from `main`
// which should be equivalent to a dictionary as output of
// `main` with the advantage that repeated keys can be used.
func TestMainReturnsTupleList(t *testing.T) {
	dir := testutil.TempDirPopulatedWith(t, "testdata/main-returns-tuple-list")
	// Avoid validateExpect (will try to parse YAML without accepting repeated keys)
	resultConfig := loadStarlarkConfig(t, dir)
	expectedConfig := loadExpectedConfig(t, dir)
	assert.Equal(t, expectedConfig, resultConfig)
}

// TestMainReturnsString ensures that we support overrides represented as a string,
// which is essentially a piece of YAML configuration.
func TestMainReturnsString(t *testing.T) {
	validateExpected(t, "testdata/main-returns-string")
}

func TestNoCtxHook(t *testing.T) {
	dir := testutil.TempDirPopulatedWith(t, "testdata/no-ctx")

	// Read the source code
	source, err := os.ReadFile(filepath.Join(dir, ".cirrus.star"))
	if err != nil {
		t.Fatal(err)
	}

	// Run the source code
	lrk := larker.New(larker.WithFileSystem(local.New(dir)))
	result, err := lrk.Hook(context.Background(), string(source), "on_build_created", []interface{}{})
	require.NoError(t, err)
	assert.Contains(t, string(result.OutputLogs), "it works fine without ctx argument!")
}

func loadStarlarkConfig(t *testing.T, dir string) string {
	// Read the source code
	source, err := os.ReadFile(filepath.Join(dir, ".cirrus.star"))
	if err != nil {
		t.Fatal(err)
	}

	// Run the source code to produce a YAML configuration
	lrk := larker.New(larker.WithFileSystem(local.New(dir)))
	result, err := lrk.Main(context.Background(), string(source))
	if err != nil && !errors.Is(err, larker.ErrNotFound) {
		t.Fatal(err)
	}

	return result.YAMLConfig
}

func loadExpectedConfig(t *testing.T, dir string) string {
	expectedConfiguration, err := os.ReadFile(filepath.Join(dir, "expected.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	return string(expectedConfiguration)
}

func validateExpected(t *testing.T, testDir string) {
	dir := testutil.TempDirPopulatedWith(t, testDir)
	resultConfig := loadStarlarkConfig(t, dir)
	expectedConfig := loadExpectedConfig(t, dir)
	assert.YAMLEq(t, expectedConfig, resultConfig)
}

// TestLoadFileSystemLocal ensures that modules can be loaded from the local file system.
func TestLoadFileSystemLocal(t *testing.T) {
	dir := testutil.TempDirPopulatedWith(t, "testdata/load-fs-local")

	// Read the source code
	source, err := os.ReadFile(filepath.Join(dir, ".cirrus.star"))
	if err != nil {
		t.Fatal(err)
	}

	// Run the source code
	lrk := larker.New(larker.WithFileSystem(local.New(dir)))
	_, err = lrk.Main(context.Background(), string(source))
	if err != nil {
		t.Fatal(err)
	}
}

// TestTimeout ensures that context.Context can be used to stop the execution of a potentially long-running script.
func TestTimeout(t *testing.T) {
	dir := testutil.TempDirPopulatedWith(t, "testdata/timeout")

	// Read the source code
	source, err := os.ReadFile(filepath.Join(dir, ".cirrus.star"))
	if err != nil {
		t.Fatal(err)
	}

	// Once second should be more than enough since 10,000,000 iterations take more than a minute on a modern CPU.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Run the source code
	lrk := larker.New(larker.WithFileSystem(local.New(dir)))
	_, err = lrk.Main(ctx, string(source))
	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.DeadlineExceeded))
}

// TestCycle ensures that import cycles are detected.
func TestCycle(t *testing.T) {
	dir := testutil.TempDirPopulatedWith(t, "testdata/cycle")

	// Read the source code
	source, err := os.ReadFile(filepath.Join(dir, "a.star"))
	if err != nil {
		t.Fatal(err)
	}

	// Run the source code
	lrk := larker.New(larker.WithFileSystem(local.New(dir)))
	_, err = lrk.Main(context.Background(), string(source))
	assert.Error(t, err)
	assert.True(t, errors.Is(err, larker.ErrLoadFailed))
}

// TestLoadGitHelpers ensures that we can use https://github.com/cirrus-modules/helpers
// as demonstrated in it's README.md.
//
// Note that we lock the revision in the .cirrus.star's load statement to prevent failures in the future.
func TestLoadGitHelpers(t *testing.T) {
	possiblySkip(t)

	dir := testutil.TempDirPopulatedWith(t, "testdata/load-git-helpers")

	// Read the source code
	source, err := os.ReadFile(filepath.Join(dir, ".cirrus.star"))
	if err != nil {
		t.Fatal(err)
	}

	// Run the source code
	lrk := larker.New(larker.WithFileSystem(local.New(dir)))
	result, err := lrk.Main(context.Background(), string(source))
	if err != nil {
		t.Fatal(err)
	}

	// Compare the output
	expected, err := os.ReadFile(filepath.Join(dir, "expected.yml"))
	if err != nil {
		t.Fatal(err)
	}
	assert.YAMLEq(t, string(expected), result.YAMLConfig)
}

func TestLoadSeveralFiles(t *testing.T) {
	possiblySkip(t)

	dir := testutil.TempDirPopulatedWith(t, "testdata/load-several-files")

	// Read the source code
	source, err := os.ReadFile(filepath.Join(dir, ".cirrus.star"))
	if err != nil {
		t.Fatal(err)
	}

	// Run the source code
	lrk := larker.New(larker.WithFileSystem(local.New(dir)))
	result, err := lrk.Main(context.Background(), string(source))
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "Hello, World!\n", string(result.OutputLogs))
}

// TestLoadTypoStarVsStart ensures that we return a user-friendly hint when loading of the module
// that ends with ".start" fails.
func TestLoadTypoStarVsStart(t *testing.T) {
	possiblySkip(t)

	dir := testutil.TempDir(t)

	lrk := larker.New(larker.WithFileSystem(local.New(dir)))

	// No hint
	_, err := lrk.Main(context.Background(), "load(\"some/lib.star\", \"symbol\")\n")
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "perhaps you've meant")

	// Hint when loading from Git
	_, err = lrk.Main(context.Background(),
		"load(\"github.com/cirrus-modules/helpers/dir/lib.start\", \"symbol\")\n")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "instead of the .start?")

	// Hint when loading from FS
	_, err = lrk.Main(context.Background(), "load(\"dir/lib.start\", \"symbol\")\n")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "instead of the .start?")
}

// TestBuiltinFS ensures that filesystem-related builtins provided by the cirrus.fs module work correctly.
func TestBuiltinFS(t *testing.T) {
	dir := testutil.TempDirPopulatedWith(t, "testdata/builtin-fs")

	// Read the source code
	source, err := os.ReadFile(filepath.Join(dir, ".cirrus.star"))
	if err != nil {
		t.Fatal(err)
	}

	// Run the source code
	lrk := larker.New(larker.WithFileSystem(local.New(dir)))
	_, err = lrk.Main(context.Background(), string(source))
	if err != nil {
		t.Fatal(err)
	}
}

// TestBuiltinEnv ensures that we expose the environment passed through options as the cirrus.env dict.
func TestBuiltinEnv(t *testing.T) {
	dir := testutil.TempDirPopulatedWith(t, "testdata/builtin-env")

	// Read the source code
	source, err := os.ReadFile(filepath.Join(dir, ".cirrus.star"))
	if err != nil {
		t.Fatal(err)
	}

	// Run the source code
	lrk := larker.New(larker.WithFileSystem(local.New(dir)), larker.WithEnvironment(map[string]string{
		"SOME_VARIABLE": "some value",
	}))
	_, err = lrk.Main(context.Background(), string(source))
	if err != nil {
		t.Fatal(err)
	}
}

// TestBuiltinChangesInclude ensures that we expose the changes_include()
// through cirrus module and it works properly.
func TestBuiltinChangesInclude(t *testing.T) {
	dir := testutil.TempDirPopulatedWith(t, "testdata/builtin-changes-include")

	// Read the source code
	source, err := os.ReadFile(filepath.Join(dir, ".cirrus.star"))
	if err != nil {
		t.Fatal(err)
	}

	affectedFiles := []string{
		"ci/build.sh",
		"CHANGELOG.md",
	}

	// Run the source code
	lrk := larker.New(larker.WithFileSystem(local.New(dir)), larker.WithAffectedFiles(affectedFiles))
	_, err = lrk.Main(context.Background(), string(source))
	if err != nil {
		t.Fatal(err)
	}
}

// TestBuiltinChangesInclude ensures that we expose the changes_include_only()
// through cirrus module and it works properly.
func TestBuiltinChangesIncludeOnly(t *testing.T) {
	dir := testutil.TempDirPopulatedWith(t, "testdata/builtin-changes-include-only")

	// Read the source code
	source, err := os.ReadFile(filepath.Join(dir, ".cirrus.star"))
	if err != nil {
		t.Fatal(err)
	}

	affectedFiles := []string{
		"go.mod",
		"main.go",
		"dir/file.go",
	}

	// Run the source code
	lrk := larker.New(larker.WithFileSystem(local.New(dir)), larker.WithAffectedFiles(affectedFiles))
	_, err = lrk.Main(context.Background(), string(source))
	if err != nil {
		t.Fatal(err)
	}
}

// TestBuiltinStarlib ensures that Starlib's modules that we expose through cirrus.* are working properly.
func TestBuiltinStarlib(t *testing.T) {
	dir := testutil.TempDirPopulatedWith(t, "testdata/builtin-starlib")

	// Start a local HTTP server for testing the "http" module
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodOptions {
			writer.Header().Set("Allow", "OPTIONS")
			writer.WriteHeader(http.StatusOK)
		}
	})
	mux.HandleFunc("/json", func(writer http.ResponseWriter, request *http.Request) {
		jsonObject := struct {
			Slideshow string `json:"slideshow"`
		}{
			Slideshow: "doesn't matter",
		}

		jsonBytes, err := json.Marshal(&jsonObject)
		if err != nil {
			writer.WriteHeader(http.StatusInternalServerError)

			return
		}

		if _, err := writer.Write(jsonBytes); err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
		}
	})
	mux.HandleFunc("/status/418", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusTeapot)
	})
	httpServer := httptest.NewServer(mux)

	// Prepare a ZIP file for test_zipfile()
	zipFile, err := os.Create(filepath.Join(dir, "test.zip"))
	if err != nil {
		t.Fatal(err)
	}

	zipWriter := zip.NewWriter(zipFile)
	txtFile, err := zipWriter.Create("test.txt")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := txtFile.Write([]byte("test\n")); err != nil {
		t.Fatal(err)
	}

	if err := zipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := zipFile.Close(); err != nil {
		t.Fatal(err)
	}

	// Read the source code
	source, err := os.ReadFile(filepath.Join(dir, ".cirrus.star"))
	if err != nil {
		t.Fatal(err)
	}

	// Starlark script is templated w.r.t. HTTP server URL, so expand it
	vars := struct {
		HTTPBinURL string
	}{
		HTTPBinURL: httpServer.URL,
	}

	template, err := template.New("").Parse(string(source))
	require.NoError(t, err)

	var expandedTemplate bytes.Buffer
	require.NoError(t, template.Execute(&expandedTemplate, &vars))

	// Run the source code
	lrk := larker.New(larker.WithFileSystem(local.New(dir)))
	_, err = lrk.Main(context.Background(), expandedTemplate.String())
	if err != nil {
		t.Fatal(err)
	}
}

func TestTestMode(t *testing.T) {
	dir := testutil.TempDirPopulatedWith(t, "testdata/test-mode")

	// Read the source code
	source, err := os.ReadFile(filepath.Join(dir, ".cirrus.star"))
	if err != nil {
		t.Fatal(err)
	}

	// Run the source code with testing mode disabled
	lrk := larker.New(larker.WithFileSystem(local.New(dir)))
	result, err := lrk.Main(context.Background(), string(source))
	require.NoError(t, err)
	assert.Contains(t, string(result.OutputLogs), "testing mode disabled")

	// Run the source code with testing mode enabled
	lrk = larker.New(larker.WithFileSystem(local.New(dir)), larker.WithTestMode())
	result, err = lrk.Main(context.Background(), string(source))
	require.NoError(t, err)
	assert.Contains(t, string(result.OutputLogs), "testing mode enabled")
}

// TestDynamicFSResolution ensures that load("cirrus", "fs") methods can dynamically
// re-resolve FS if a non-relative path is detected (one that points to GitHub or
// Git repository).
func TestDynamicFSResolution(t *testing.T) {
	possiblySkip(t)

	dir := testutil.TempDirPopulatedWith(t, "testdata/dynamic-fs-resolution")

	// Read the source code
	source, err := os.ReadFile(filepath.Join(dir, ".cirrus.star"))
	if err != nil {
		t.Fatal(err)
	}

	// Run the source code with testing mode disabled
	lrk := larker.New(larker.WithFileSystem(local.New(dir)))
	_, err = lrk.Main(context.Background(), string(source))
	require.NoError(t, err)
}
