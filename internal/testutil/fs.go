package testutil

import (
	"io/ioutil"
	"os"
	"testing"
)

// tempDir supplements an alternative to TB.TempDir()[1], which is only available in 1.15.
// [1]: https://github.com/golang/go/issues/35998
func TempDir(t *testing.T) string {
	dir, err := ioutil.TempDir("", t.Name())
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Fatal(err)
		}
	})

	return dir
}

// TempChdir switches to a temporary per-test directory.
func TempChdir(t *testing.T) {
	if err := os.Chdir(TempDir(t)); err != nil {
		t.Fatal(err)
	}
}
