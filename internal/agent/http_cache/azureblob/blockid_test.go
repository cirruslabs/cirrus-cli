package azureblob

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBlockIDToPartNumberJS(t *testing.T) {
	// Azure SDK for JS format
	partNumber, err := blockIDToPartNumber("Yzg4ODM0YjYtZmI3MC00OWNmLWJlYmEtNDliODFjNDE0MWM3MDAwMDAwMDAwMDAx")
	require.NoError(t, err)
	require.EqualValues(t, 2, partNumber)
}

func TestBlockIDToPartNumberGolang(t *testing.T) {
	// Azure SDK for Golang format
	partNumber, err := blockIDToPartNumber("6eiRh4oEQZ1cibBlHr1wFAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==")
	require.NoError(t, err)
	require.EqualValues(t, 1, partNumber)

	partNumber, err = blockIDToPartNumber("6eiRh4oEQZ1cibBlHr1wFAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==")
	require.NoError(t, err)
	require.EqualValues(t, 2, partNumber)
}
