package azureblob

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBlockIDToPartNumber(t *testing.T) {
	partNumber, err := blockIDToPartNumber("Yzg4ODM0YjYtZmI3MC00OWNmLWJlYmEtNDliODFjNDE0MWM3MDAwMDAwMDAwMDAx")
	require.NoError(t, err)
	require.Equal(t, 2, partNumber)
}
