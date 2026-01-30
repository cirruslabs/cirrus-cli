package cache_test

import (
	"crypto/sha256"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/executor/cache"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// TestKeySanitization ensures that potentially problematic keys are sanitized.
func TestKeySanitization(t *testing.T) {
	dir := testutil.TempDir(t)

	// Create a cache in a temporary directory
	c, err := cache.New(dir, "")
	if err != nil {
		t.Fatal(err)
	}

	// Prepare examples to be sanitized
	var examples = map[string][]byte{
		"..":           []byte("parent directory"),
		"/file.tar.gz": []byte("file in a root directory"),
	}

	// Write to cache
	for key, value := range examples {
		cacheWrite(t, c, key, value)
	}

	// Examine cache directory
	dirEntries, err := os.ReadDir(filepath.Join(dir, "cirrus", "projects"))
	if err != nil {
		t.Fatal(err)
	}

	var expectedFiles []string
	for key := range examples {
		keyDigest := fmt.Sprintf("%x", sha256.Sum256([]byte(key)))
		expectedFiles = append(expectedFiles, keyDigest)
	}

	var actualFiles []string
	for _, dirEntry := range dirEntries {
		actualFiles = append(actualFiles, dirEntry.Name())
	}

	// Remove created blobs from disk to avoid wasting space
	for key := range examples {
		require.NoError(t, c.Delete(key))
	}

	require.ElementsMatch(t, expectedFiles, actualFiles)
}

// TestGetAndPut ensures that the cache correctly stores and retrieves different blobs at once.
func TestMultipleGetAndPut(t *testing.T) {
	dir := testutil.TempDir(t)

	// Create a cache in a temporary directory
	c, err := cache.New(dir, "")
	if err != nil {
		t.Fatal(err)
	}

	// Prepare examples
	var examples = map[string][]byte{
		"":      []byte("empty key"),
		"1/2/3": []byte("some slashes"),
	}

	// Write to cache
	for key, value := range examples {
		cacheWrite(t, c, key, value)
	}

	// Read from cache
	for key, value := range examples {
		require.EqualValues(t, value, cacheRead(t, c, key))
	}

	// Remove created blobs from disk to avoid wasting space
	for key := range examples {
		require.NoError(t, c.Delete(key))
	}
}

func cacheWrite(t *testing.T, c *cache.Cache, key string, data []byte) {
	putOp, err := c.Put(key)
	if err != nil {
		t.Fatal(err)
	}

	n, err := putOp.Write(data)
	require.Nil(t, err)
	require.Equal(t, n, len(data))

	if err := putOp.Finalize(); err != nil {
		t.Fatal(err)
	}
}

func cacheRead(t *testing.T, c *cache.Cache, key string) []byte {
	file, err := c.Get(key)
	if err != nil {
		t.Fatal(err)
	}

	data, err := io.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}

	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	return data
}
