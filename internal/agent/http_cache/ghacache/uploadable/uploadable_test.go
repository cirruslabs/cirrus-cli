package uploadable_test

import (
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/ghacache/uploadable"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPartsAreOrdered(t *testing.T) {
	uploadable := uploadable.New("key", "version", "upload-id")

	require.NoError(t, uploadable.AppendPart(2, "etag-2", 42))
	require.NoError(t, uploadable.AppendPart(1, "etag-1", 12))
	require.NoError(t, uploadable.AppendPart(3, "etag-3", 46))

	parts, size, err := uploadable.BuildCommitRequestParts()
	require.NoError(t, err)

	require.Equal(t, []*api.MultipartCacheUploadCommitRequest_Part{
		{PartNumber: 1, Etag: "etag-1"},
		{PartNumber: 2, Etag: "etag-2"},
		{PartNumber: 3, Etag: "etag-3"},
	}, parts)
	require.EqualValues(t, 100, size)
}
