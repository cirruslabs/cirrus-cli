package http_cache_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/cirruscimock"
	agentstorage "github.com/cirruslabs/cirrus-cli/internal/agent/storage"
	"github.com/cirruslabs/cirrus-cli/internal/testutil"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestHTTPCache(t *testing.T) {
	testutil.NeedsContainerization(t)

	conn := cirruscimock.ClientConn(t)
	backend := agentstorage.NewCirrusStoreBackend(api.NewCirrusCIServiceClient(conn), api.OldTaskIdentification("test", "test"))

	httpCacheObjectURL := "http://" + http_cache.Start(context.Background(), backend) +
		"/cache/" + uuid.NewString() + "/test.txt"

	// Ensure that the cache entry does not exist
	resp, err := http.Get(httpCacheObjectURL)
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	// Create the cache entry
	resp, err = http.Post(httpCacheObjectURL, "text/plain", strings.NewReader("Hello, World!"))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	resp, err = http.Get(httpCacheObjectURL)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	cacheEntryBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, "Hello, World!", string(cacheEntryBody))
}
