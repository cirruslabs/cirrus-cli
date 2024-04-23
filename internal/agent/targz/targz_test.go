package targz_test

import (
	"archive/tar"
	"compress/gzip"
	"github.com/cirruslabs/cirrus-cli/internal/agent/targz"
	"github.com/cirruslabs/cirrus-cli/internal/agent/testutil"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"path/filepath"
	"testing"
)

type PartialTarHeader struct {
	Typeflag byte
	Name     string
	Linkname string
	Contents []byte
}

func TarGzContentsHelper(t *testing.T, path string) []PartialTarHeader {
	var result []PartialTarHeader

	archive, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()

	gzReader, err := gzip.NewReader(archive)
	if err != nil {
		t.Fatal(err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}

			t.Fatal(err)
		}

		contents, err := io.ReadAll(tarReader)
		if err != nil {
			t.Fatal(err)
		}

		result = append(result, PartialTarHeader{
			Typeflag: header.Typeflag,
			Name:     header.Name,
			Linkname: header.Linkname,
			Contents: contents,
		})
	}

	return result
}

func TestArchive(t *testing.T) {
	testCases := []struct {
		Name     string
		Populate func(dir string)
		Expected []PartialTarHeader
	}{
		{"empty folder", func(dir string) {
			// do nothing
		}, []PartialTarHeader{
			{tar.TypeDir, "", "", []byte{}},
		}},
		{"single file", func(dir string) {
			os.WriteFile(filepath.Join(dir, "file.txt"), []byte("contents"), 0600)
		}, []PartialTarHeader{
			{tar.TypeDir, "", "", []byte{}},
			{tar.TypeReg, "/file.txt", "", []byte("contents")},
		}},
		{"file inside of a directory", func(dir string) {
			subDir := filepath.Join(dir, "sub-directory")
			os.Mkdir(subDir, 0700)
			os.WriteFile(filepath.Join(subDir, "file.txt"), []byte("contents"), 0600)
		}, []PartialTarHeader{
			{tar.TypeDir, "", "", []byte{}},
			{tar.TypeDir, "/sub-directory", "", []byte{}},
			{tar.TypeReg, "/sub-directory/file.txt", "", []byte("contents")},
		}},
		{"relative links are kept as is", func(dir string) {
			os.Symlink("../../../etc/passwd", filepath.Join(dir, "symlink1"))
			os.Symlink(".cirrus.yml", filepath.Join(dir, "symlink2"))
		}, []PartialTarHeader{
			{tar.TypeDir, "", "", []byte{}},
			{tar.TypeSymlink, "/symlink1", "../../../etc/passwd", []byte{}},
			{tar.TypeSymlink, "/symlink2", ".cirrus.yml", []byte{}},
		}},
		{"absolute links are made relative", func(dir string) {
			os.Symlink(dir, filepath.Join(dir, "symlink"))
		}, []PartialTarHeader{
			{tar.TypeDir, "", "", []byte{}},
			{tar.TypeSymlink, "/symlink", ".", []byte{}},
		}},
		{"absolute links outside base are NOT made relative", func(dir string) {
			os.Symlink("/tmp", filepath.Join(dir, "symlink"))
		}, []PartialTarHeader{
			{tar.TypeDir, "", "", []byte{}},
			{tar.TypeSymlink, "/symlink", "/tmp", []byte{}},
		}},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.Name, func(t *testing.T) {
			// Create and populate the folder that will be archived
			folderPath := testutil.TempDir(t)
			testCase.Populate(folderPath)

			// Make up a place where the archive will be stored
			dest := filepath.Join(testutil.TempDir(t), "archive.tar.gz")

			// Create archive
			if err := targz.Archive(folderPath, []string{folderPath}, dest); err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, testCase.Expected, TarGzContentsHelper(t, dest))
		})
	}
}

func TestArchiveMultiple(t *testing.T) {
	// Create a base folder
	baseFolder := testutil.TempDir(t)

	// Create sub-folders
	subFolder1 := filepath.Join(baseFolder, "left", "hot")
	os.MkdirAll(subFolder1, 0700)
	subFolder2 := filepath.Join(baseFolder, "right", "cold")
	os.MkdirAll(subFolder2, 0700)

	os.WriteFile(filepath.Join(baseFolder, "should-not-be-included.txt"), []byte("doesn't matter"), 0600)

	// Make up a place where the archive will be stored
	dest := filepath.Join(testutil.TempDir(t), "archive.tar.gz")

	// Create archive
	if err := targz.Archive(baseFolder, []string{subFolder1, subFolder2}, dest); err != nil {
		t.Fatal(err)
	}

	expected := []PartialTarHeader{
		{tar.TypeDir, "/left/hot", "", []byte{}},
		{tar.TypeDir, "/right/cold", "", []byte{}},
	}
	assert.Equal(t, expected, TarGzContentsHelper(t, dest))
}
