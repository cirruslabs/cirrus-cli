package azureblob

import (
	"encoding/base64"
	"fmt"
	"strconv"
)

const maxBlockIndexLength = 6

func blockIDToIndex(blockIDRaw string) (int, error) {
	// Decode the Base64-encoded block ID
	blockIDBytes, err := base64.StdEncoding.DecodeString(blockIDRaw)
	if err != nil {
		return 0, err
	}

	// Extract the index from the block ID
	blockID := string(blockIDBytes)

	if len(blockID) < maxBlockIndexLength {
		return 0, fmt.Errorf("block ID is too small to contain the index number")
	}

	rawBlockIndex := blockID[len(blockID)-maxBlockIndexLength:]

	// Parse the index as an integer
	blockIndex, err := strconv.Atoi(rawBlockIndex)
	if err != nil {
		return 0, err
	}

	return blockIndex, nil
}
