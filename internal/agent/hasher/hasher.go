package hasher

import (
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type DiffEntry struct {
	Type DiffEntryType
	Path string
}

type DiffEntryType int

const (
	Created DiffEntryType = iota
	Modified
	Deleted
)

func (det DiffEntryType) String() string {
	switch det {
	case Created:
		return "created"
	case Modified:
		return "modified"
	case Deleted:
		return "deleted"
	default:
		return "unknown"
	}
}

type Hasher struct {
	globalHash hash.Hash
	fileHashes map[string]string
}

func New() *Hasher {
	return &Hasher{
		globalHash: sha256.New(),
		fileHashes: make(map[string]string),
	}
}

func (hasher *Hasher) SHA() string {
	digest := hasher.globalHash.Sum(nil)
	return fmt.Sprintf("%x", digest)
}

func (hasher *Hasher) Len() int {
	return len(hasher.fileHashes)
}

func (hasher *Hasher) DiffWithNewer(newer *Hasher) []DiffEntry {
	var result []DiffEntry

	for newPath, newHash := range newer.fileHashes {
		oldHash, ok := hasher.fileHashes[newPath]
		if !ok {
			result = append(result, DiffEntry{Type: Created, Path: newPath})
		} else if newHash != oldHash {
			result = append(result, DiffEntry{Type: Modified, Path: newPath})
		}
	}

	for oldPath := range hasher.fileHashes {
		_, ok := newer.fileHashes[oldPath]
		if !ok {
			result = append(result, DiffEntry{Type: Deleted, Path: oldPath})
		}
	}

	return result
}

func (hasher *Hasher) AddFolder(baseFolder string, folderPath string) error {
	if _, err := os.Stat(folderPath); os.IsNotExist(err) {
		return nil
	}
	return filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		fileHash, err := fileHash(path)
		// symlink can still be a directory
		if err != nil && strings.Contains(err.Error(), "is a directory") {
			return nil
		}
		if err != nil && os.IsNotExist(err) && (info.Mode()&os.ModeSymlink != 0) {
			destination, linkErr := os.Readlink(path)
			if linkErr == nil {
				hasher := sha256.New()
				_, err = hasher.Write([]byte(destination))
				fileHash = hasher.Sum(nil)
			}
		}
		if err != nil {
			return err
		}
		relativePath, err := filepath.Rel(baseFolder, path)
		if err != nil {
			return err
		}
		hasher.fileHashes[relativePath] = fmt.Sprintf("%x", fileHash)
		_, err = hasher.globalHash.Write(fileHash)
		return err
	})
}

func fileHash(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	shaHash := sha256.New()
	_, err = io.Copy(shaHash, f)
	if err != nil {
		return nil, err
	}
	return shaHash.Sum(nil), nil
}
