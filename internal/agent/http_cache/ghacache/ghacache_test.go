package ghacache_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/cirruslabs/cirrus-cli/internal/agent/client"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/blobstorage"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/ghacache/cirruscimock"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	actionscache "github.com/tonistiigi/go-actions-cache"
)

func TestGHA(t *testing.T) {
	testutil.NeedsContainerization(t)

	ctx := context.Background()

	client.InitClient(cirruscimock.ClientConn(t), "test", "test")

	httpCacheURL := "http://" + http_cache.Start(ctx, blobstorage.NewCirrusBlobStorage(client.CirrusClient)) + "/"

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"ac":  `[]`,
		"nbf": time.Now().Add(-time.Hour).Unix(),
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte("whatever"))
	require.NoError(t, err)

	ac, err := actionscache.New(tokenString, httpCacheURL, false, actionscache.Opt{})
	require.NoError(t, err)

	cacheKey := uuid.NewString()
	cacheValue := []byte("Hello, World!\n")

	// Ensure that an entry for our cache key is not present
	entry, err := ac.Load(ctx, cacheKey)
	require.NoError(t, err)
	require.Nil(t, entry)

	// Upload an entry for our cache key
	require.NoError(t, ac.Save(ctx, cacheKey, actionscache.NewBlob(cacheValue)))

	// Ensure that an entry for our cache key is present
	// and matches to what we've previously put in the cache
	entry, err = ac.Load(ctx, cacheKey)
	require.NoError(t, err)
	require.NotNil(t, entry)
	require.Equal(t, entry.Key, cacheKey)
	buf := &bytes.Buffer{}
	require.NoError(t, entry.WriteTo(ctx, buf))
	require.Equal(t, cacheValue, buf.Bytes())
}
