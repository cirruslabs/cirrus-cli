package instance_test

import (
	"archive/tar"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

// TestCreateArchive ensures that create tar archives contain the files we've put in them at the expected paths.
func TestCreateArchive(t *testing.T) {
	dir := testutil.TempDir(t)

	// Create a simple file
	if err := ioutil.WriteFile(filepath.Join(dir, "file.txt"), []byte(""), 0600); err != nil {
		t.Fatal(err)
	}

	// Create a file inside of a directory
	subDir := filepath.Join(dir, "directory")
	if err := os.Mkdir(subDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(filepath.Join(subDir, "file-in-a-directory"), []byte(""), 0600); err != nil {
		t.Fatal(err)
	}

	// Create the archive
	archivePath, err := instance.CreateTempArchive(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Read the archive
	archiveFile, err := os.Open(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer archiveFile.Close()

	// Inspect the archive contents
	archive := tar.NewReader(archiveFile)

	header, err := archive.Next()
	require.NoError(t, err)
	assert.Equal(t, "directory/file-in-a-directory", header.Name)

	header, err = archive.Next()
	require.NoError(t, err)
	assert.Equal(t, "file.txt", header.Name)

	_, err = archive.Next()
	assert.Equal(t, io.EOF, err)
}
