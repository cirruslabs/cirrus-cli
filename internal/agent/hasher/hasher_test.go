package hasher_test

import (
	"github.com/cirruslabs/cirrus-cli/internal/agent/hasher"
	"github.com/cirruslabs/cirrus-cli/internal/agent/testutil"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

func TestDiffWithNewer(t *testing.T) {
	var testCases = []struct {
		Name       string
		OldFiller  func(dir string)
		NewFiller  func(dir string)
		Difference []hasher.DiffEntry
	}{
		{
			Name: "empty directories",
			OldFiller: func(dir string) {
				// do nothing
			},
			NewFiller: func(dir string) {
				// do nothing
			},
			Difference: []hasher.DiffEntry(nil),
		},
		{
			Name: "no change",
			OldFiller: func(dir string) {
				os.WriteFile(filepath.Join(dir, "same.txt"), []byte("same contents"), 0600)
			},
			NewFiller: func(dir string) {
				os.WriteFile(filepath.Join(dir, "same.txt"), []byte("same contents"), 0600)
			},
			Difference: []hasher.DiffEntry(nil),
		},
		{
			Name: "file creation",
			OldFiller: func(dir string) {
				os.WriteFile(filepath.Join(dir, "control.txt"), []byte("control sample"), 0600)
			},
			NewFiller: func(dir string) {
				os.WriteFile(filepath.Join(dir, "control.txt"), []byte("control sample"), 0600)
				os.WriteFile(filepath.Join(dir, "creation.txt"), []byte(""), 0600)
			},
			Difference: []hasher.DiffEntry{
				{hasher.Created, "creation.txt"},
			},
		},
		{
			Name: "file modification",
			OldFiller: func(dir string) {
				os.WriteFile(filepath.Join(dir, "control.txt"), []byte("control sample"), 0600)
				os.WriteFile(filepath.Join(dir, "modification.txt"), []byte("old contents"), 0600)
			},
			NewFiller: func(dir string) {
				os.WriteFile(filepath.Join(dir, "control.txt"), []byte("control sample"), 0600)
				os.WriteFile(filepath.Join(dir, "modification.txt"), []byte("new contents"), 0600)
			},
			Difference: []hasher.DiffEntry{
				{hasher.Modified, "modification.txt"},
			},
		},
		{
			Name: "file deletion",
			OldFiller: func(dir string) {
				os.WriteFile(filepath.Join(dir, "control.txt"), []byte("control sample"), 0600)
				os.WriteFile(filepath.Join(dir, "deletion.txt"), []byte("old contents"), 0600)
			},
			NewFiller: func(dir string) {
				os.WriteFile(filepath.Join(dir, "control.txt"), []byte("control sample"), 0600)
			},
			Difference: []hasher.DiffEntry{
				{hasher.Deleted, "deletion.txt"},
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.Name, func(t *testing.T) {
			// Create and fill old directory
			oldDir := testutil.TempDir(t)
			testCase.OldFiller(oldDir)

			// Create and fill new directory
			newDir := testutil.TempDir(t)
			testCase.NewFiller(newDir)

			// Hash directories
			oldHasher := hasher.New()
			if err := oldHasher.AddFolder(oldDir, oldDir); err != nil {
				t.Fatal(err)
			}

			newHasher := hasher.New()
			if err := newHasher.AddFolder(newDir, newDir); err != nil {
				t.Fatal(err)
			}

			assert.EqualValues(t, testCase.Difference, oldHasher.DiffWithNewer(newHasher))
		})
	}
}
