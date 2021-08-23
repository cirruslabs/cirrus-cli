package cachinglayer

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs"
	"github.com/hashicorp/golang-lru"
)

// fs.Stat() cache parameters.
const maxStatCacheEntries = 8192

// fs.Get() cache parameters.
const (
	maxGetCacheBudgetBytes    = 64 * 1024 * 1024
	maxGetCacheEntrySizeBytes = 65536
	maxGetCacheEntries        = maxGetCacheBudgetBytes / maxGetCacheEntrySizeBytes
)

// fs.ReadDir() cache parameters.
const maxReadDirCacheEntries = 8192

type CachingLayer struct {
	fs           fs.FileSystem
	statCache    *lru.Cache
	getCache     *lru.Cache
	readDirCache *lru.Cache
}

func Wrap(fs fs.FileSystem) (*CachingLayer, error) {
	statCache, err := lru.New(maxStatCacheEntries)
	if err != nil {
		return nil, err
	}

	getCache, err := lru.New(maxGetCacheEntries)
	if err != nil {
		return nil, err
	}

	readDirCache, err := lru.New(maxReadDirCacheEntries)
	if err != nil {
		return nil, err
	}

	return &CachingLayer{
		fs:           fs,
		statCache:    statCache,
		getCache:     getCache,
		readDirCache: readDirCache,
	}, nil
}

func (cl *CachingLayer) Stat(ctx context.Context, path string) (*fs.FileInfo, error) {
	fileInfo, ok := cl.statCache.Get(path)
	if !ok {
		fileInfo, err := cl.fs.Stat(ctx, path)
		if err != nil {
			return fileInfo, err
		}

		cl.statCache.Add(path, fileInfo)

		return fileInfo, nil
	}

	return fileInfo.(*fs.FileInfo), nil
}

func (cl *CachingLayer) Get(ctx context.Context, path string) ([]byte, error) {
	fileBytes, ok := cl.getCache.Get(path)
	if !ok {
		fileBytes, err := cl.fs.Get(ctx, path)
		if err != nil {
			return fileBytes, err
		}

		if len(fileBytes) <= maxGetCacheEntrySizeBytes {
			cl.getCache.Add(path, fileBytes)
		}

		return fileBytes, nil
	}

	return fileBytes.([]byte), nil
}

func (cl *CachingLayer) ReadDir(ctx context.Context, path string) ([]string, error) {
	directoryContent, ok := cl.readDirCache.Get(path)
	if !ok {
		directoryContent, err := cl.fs.ReadDir(ctx, path)
		if err != nil {
			return directoryContent, err
		}

		cl.readDirCache.Add(path, directoryContent)

		return directoryContent, nil
	}

	return directoryContent.([]string), nil
}

func (cl *CachingLayer) Join(elem ...string) string {
	return cl.fs.Join(elem...)
}
