package lrucache

import (
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strconv"

	"github.com/bartventer/httpcache/store"
	"github.com/bartventer/httpcache/store/driver"
	"github.com/hashicorp/golang-lru/v2"
)

const Scheme = "lrucache"

var ErrInitializationFailed = errors.New(fmt.Sprintf("%s: failed to initialize", Scheme))

type Cache struct {
	lru *lru.Cache[string, []byte]
}

func init() {
	store.Register(Scheme, driver.DriverFunc(func(url *url.URL) (driver.Conn, error) {
		return NewFromURL(url)
	}))
}

func New(size int) (*Cache, error) {
	if size < 0 {
		return nil, fmt.Errorf("%w: cache size cannot be negative",
			ErrInitializationFailed)
	}

	if size == 0 {
		return nil, fmt.Errorf("%w: cache size cannot be zero",
			ErrInitializationFailed)
	}

	lru, err := lru.New[string, []byte](size)
	if err != nil {
		return nil, err
	}

	return &Cache{
		lru: lru,
	}, nil
}

func NewFromURL(url *url.URL) (*Cache, error) {
	sizeRaw := url.Query().Get("size")

	size, err := strconv.Atoi(sizeRaw)
	if err != nil {
		return nil, fmt.Errorf("%w: cannot parse cache size parameter from URL: %v",
			ErrInitializationFailed, err)
	}

	return New(size)
}

func (cache *Cache) Get(key string) ([]byte, error) {
	value, ok := cache.lru.Get(key)
	if !ok {
		return nil, errors.Join(
			driver.ErrNotExist,
			fmt.Errorf("%s: key %q does not exist", Scheme, key),
		)
	}

	// Return a copy to prevent external mutation
	value = slices.Clone(value)

	return value, nil
}

func (cache *Cache) Set(key string, value []byte) error {
	// Store a copy to prevent external mutation
	value = slices.Clone(value)

	cache.lru.Add(key, value)

	return nil
}

func (cache *Cache) Delete(key string) error {
	present := cache.lru.Remove(key)

	if !present {
		return errors.Join(
			driver.ErrNotExist,
			fmt.Errorf("%s: key %q does not exist", Scheme, key),
		)
	}

	return nil
}
