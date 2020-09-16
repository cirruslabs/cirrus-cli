package bounded_test

import (
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/loader/git/bounded"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestByteBounds(t *testing.T) {
	const maxBytes = 65536

	fs := bounded.NewFilesystem(maxBytes, 1)

	file, err := fs.Create("some-file.txt")
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < maxBytes; i++ {
		_, err := file.Write([]byte("A"))
		if err != nil {
			t.Fatal(err)
		}
	}

	_, err = file.Write([]byte("A"))
	require.Error(t, err)
	assert.True(t, errors.Is(err, bounded.ErrExhausted))
}

func TestFileBounds(t *testing.T) {
	const maxFiles = 128

	fs := bounded.NewFilesystem(1, maxFiles)

	for i := 0; i < maxFiles; i++ {
		_, err := fs.Create(fmt.Sprintf("%d.txt", i))
		if err != nil {
			t.Fatal(err)
		}
	}

	_, err := fs.Create("some-file.txt")
	require.Error(t, err)
	assert.True(t, errors.Is(err, bounded.ErrExhausted))
}
