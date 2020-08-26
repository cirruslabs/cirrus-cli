package larker_test

import (
	"context"
	"errors"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/cirruslabs/cirrus-cli/pkg/larker"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"
)

// TestSimpleTask ensures that .cirrus.star is able to generate the simplest possible configuration.
func TestSimpleTask(t *testing.T) {
	validateExpected(t, "testdata/simple-task")
}

// TestSugarCoatedTask ensures that .cirrus.star is able to use imported functions for task generation.
func TestSugarCoatedTask(t *testing.T) {
	validateExpected(t, "testdata/sugar-coated-task")
}

func validateExpected(t *testing.T, testDir string) {
	dir := testutil.TempDirPopulatedWith(t, testDir)

	// Read the source code
	source, err := ioutil.ReadFile(filepath.Join(dir, ".cirrus.star"))
	if err != nil {
		t.Fatal(err)
	}

	// Run the source code to produce a YAML configuration
	lrk := larker.New(larker.WithFileSystem(fs.NewLocalFileSystem(dir)))
	configuration, err := lrk.Main(context.Background(), string(source))
	if err != nil {
		t.Fatal(err)
	}

	expectedConfiguration, err := ioutil.ReadFile(filepath.Join(dir, "expected.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	assert.YAMLEq(t, string(expectedConfiguration), configuration)
}

// TestLoadFileSystemLocal ensures that modules can be loaded from the local file system.
func TestLoadFileSystemLocal(t *testing.T) {
	dir := testutil.TempDirPopulatedWith(t, "testdata/load-fs-local")

	// Read the source code
	source, err := ioutil.ReadFile(filepath.Join(dir, ".cirrus.star"))
	if err != nil {
		t.Fatal(err)
	}

	// Run the source code
	lrk := larker.New(larker.WithFileSystem(fs.NewLocalFileSystem(dir)))
	_, err = lrk.Main(context.Background(), string(source))
	if err != nil {
		t.Fatal(err)
	}
}

// TestTimeout ensures that context.Context can be used to stop the execution of a potentially long-running script.
func TestTimeout(t *testing.T) {
	dir := testutil.TempDirPopulatedWith(t, "testdata/timeout")

	// Read the source code
	source, err := ioutil.ReadFile(filepath.Join(dir, ".cirrus.star"))
	if err != nil {
		t.Fatal(err)
	}

	// Once second should be more than enough since 10,000,000 iterations take more than a minute on a modern CPU.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Run the source code
	lrk := larker.New(larker.WithFileSystem(fs.NewLocalFileSystem(dir)))
	_, err = lrk.Main(ctx, string(source))
	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.DeadlineExceeded))
}

// TestCycle ensures that import cycles are detected.
func TestCycle(t *testing.T) {
	dir := testutil.TempDirPopulatedWith(t, "testdata/cycle")

	// Read the source code
	source, err := ioutil.ReadFile(filepath.Join(dir, "a.star"))
	if err != nil {
		t.Fatal(err)
	}

	// Run the source code
	lrk := larker.New(larker.WithFileSystem(fs.NewLocalFileSystem(dir)))
	_, err = lrk.Main(context.Background(), string(source))
	assert.Error(t, err)
	assert.True(t, errors.Is(err, larker.ErrLoadFailed))
}
