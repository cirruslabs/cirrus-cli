package cache

import (
	"bufio"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const bufSize = 10 * 1024 * 1024

var (
	ErrFailedToInitialize = errors.New("cache initialization failed")
	ErrBlobNotFound       = errors.New("blob with the specified key not found")
	ErrInternal           = errors.New("internal cache error")
)

type Cache struct {
	namespaceDir string
}

func New(dir string, namespace string) (*Cache, error) {
	if dir == "" {
		userCacheDir, err := os.UserCacheDir()
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrFailedToInitialize, err)
		}
		dir = userCacheDir
	}

	namespaceDir := filepath.Join(dir, "cirrus", "projects", namespace)

	// Create a base directory, ignoring ErrExist since it may already be created
	// by a previous or parallel invocation of the CLI
	if err := os.MkdirAll(namespaceDir, 0700); err != nil {
		if !os.IsExist(err) {
			return nil, fmt.Errorf("%w: %v", ErrFailedToInitialize, err)
		}
	}

	return &Cache{
		namespaceDir: namespaceDir,
	}, nil
}

func (c *Cache) Get(key string) (*os.File, error) {
	file, err := os.OpenFile(c.blobPath(key), os.O_RDONLY, 0)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrBlobNotFound
		}
		return file, fmt.Errorf("%w: %v", ErrInternal, err)
	}

	return file, nil
}

func (c *Cache) Put(key string) (*PutOperation, error) {
	tmpBlobFile, err := ioutil.TempFile(c.namespaceDir, ".temporary-blob-")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInternal, err)
	}

	return &PutOperation{
		tmpBlobFile:   tmpBlobFile,
		tmpBlobWriter: bufio.NewWriterSize(tmpBlobFile, bufSize),
		finalBlobPath: c.blobPath(key),
	}, nil
}

func (c *Cache) Delete(key string) error {
	return os.Remove(c.blobPath(key))
}

func (c *Cache) blobPath(key string) string {
	if needsSanitization(key) {
		keyHash := sha256.Sum256([]byte(key))
		key = fmt.Sprintf("%x", keyHash)
	}

	return filepath.Join(c.namespaceDir, key)
}

func needsSanitization(key string) bool {
	if key == "" {
		return true
	}

	isSafeChar := func(c rune) bool {
		if '0' <= c && c <= '9' {
			return true
		}
		if 'a' <= c && c <= 'z' {
			return true
		}
		if 'A' <= c && c <= 'Z' {
			return true
		}
		if c == '-' || c == '_' {
			return true
		}
		return false
	}
	isUnsafeChar := func(c rune) bool {
		return !isSafeChar(c)
	}
	containsUnsafeChar := strings.IndexFunc(key, isUnsafeChar) != -1

	return containsUnsafeChar
}
