package lrucache_test

import (
	"net/url"
	"testing"

	"github.com/bartventer/httpcache/store/driver"
	"github.com/cirruslabs/cirrus-cli/internal/evaluator/lrucache"
	"github.com/stretchr/testify/require"
)

func TestLRUCache(t *testing.T) {
	u, err := url.Parse("lrucache://?size=2")
	require.NoError(t, err)

	cache, err := lrucache.NewFromURL(u)
	require.NoError(t, err)

	const (
		firstKey   = "foo-key"
		firstValue = "foo-value"

		secondKey   = "bar-key"
		secondValue = "bar-value"

		thirdKey   = "baz-key"
		thirdValue = "baz-value"
	)

	// A newly initialized LRU cache should contain nothing
	_, err = cache.Get(firstKey)
	require.ErrorIs(t, err, driver.ErrNotExist)
	_, err = cache.Get(secondKey)
	require.ErrorIs(t, err, driver.ErrNotExist)
	_, err = cache.Get(thirdKey)
	require.ErrorIs(t, err, driver.ErrNotExist)

	// It should contain first key after we insert it
	err = cache.Set(firstKey, []byte(firstValue))
	require.NoError(t, err)
	value, err := cache.Get(firstKey)
	require.NoError(t, err)
	require.Equal(t, []byte(firstValue), value)

	// It should contain second key after we insert it
	err = cache.Set(secondKey, []byte(secondValue))
	require.NoError(t, err)
	value, err = cache.Get(secondKey)
	require.NoError(t, err)
	require.Equal(t, []byte(secondValue), value)

	// It should contain third key after we insert it
	err = cache.Set(thirdKey, []byte(thirdValue))
	require.NoError(t, err)
	value, err = cache.Get(thirdKey)
	require.NoError(t, err)
	require.Equal(t, []byte(thirdValue), value)

	// However, since the LRU cache size is 2 entries, it should evict first key
	_, err = cache.Get(firstKey)
	require.ErrorIs(t, err, driver.ErrNotExist)

	// ...but not the second key
	value, err = cache.Get(secondKey)
	require.NoError(t, err)
	require.Equal(t, []byte(secondValue), value)

	// However, if we delete the second key manually, it should be gone
	err = cache.Delete(secondKey)
	require.NoError(t, err)
	_, err = cache.Get(secondKey)
	require.ErrorIs(t, err, driver.ErrNotExist)
}
