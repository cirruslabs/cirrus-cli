package azureblob

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBlockIDToIndex(t *testing.T) {
	blockIndex, err := blockIDToIndex("Yzg4ODM0YjYtZmI3MC00OWNmLWJlYmEtNDliODFjNDE0MWM3MDAwMDAwMDAwMDAx")
	require.NoError(t, err)
	require.Equal(t, 1, blockIndex)
}
