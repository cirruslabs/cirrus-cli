package uploadable_test

import (
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/ghacache/uploadable"
	"github.com/stretchr/testify/require"
	"io"
	"testing"
)

func TestSimple(t *testing.T) {
	uploadable, err := uploadable.New("hello", "1")
	require.NoError(t, err)

	const part1 = "Hello, "
	const part2 = "World!"

	require.NoError(t, uploadable.WriteChunk(fmt.Sprintf("bytes %d-%d/*",
		0, len(part1)), []byte(part1)))
	require.NoError(t, uploadable.WriteChunk(fmt.Sprintf("bytes %d-%d/*",
		len(part1), len(part1+part2)), []byte(part2)))

	reader, size, err := uploadable.Finalize()
	require.NoError(t, err)
	require.EqualValues(t, len(part1+part2), size)

	buf, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, part1+part2, string(buf))
}
