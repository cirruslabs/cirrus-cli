package azureblob

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"strconv"
)

const (
	// Azure SDK for JS limits its block size to 48 bytes[1]
	//
	// [1]: https://github.com/Azure/azure-sdk-for-js/blob/fc4cbf0e10e15cbbe7cf873294db7d6e2bd02851/sdk/storage/storage-blob/src/utils/utils.common.ts#L483-L484
	jsMaxSourceStringLength = 48

	// Constant from Azure SDK for JS[1]
	//
	// [1]: https://github.com/Azure/azure-sdk-for-js/blob/fc4cbf0e10e15cbbe7cf873294db7d6e2bd02851/sdk/storage/storage-blob/src/utils/utils.common.ts#L486-L487
	jsMaxBlockIndexLength = 6

	// Azure SDK for Golang uses fixed 64-byte blocks[1]
	//
	// [1]: https://github.com/Azure/azure-sdk-for-go/blob/42f1cc3136bf4f0a7d9a36d634576bbf8521beef/sdk/storage/azblob/blockblob/chunkwriting.go#L227
	golangFixedBlockLength = 64
)

func blockIDToPartNumber(blockIDRaw string) (uint32, error) {
	// Decode the Base64-encoded block ID
	blockIDBytes, err := base64.StdEncoding.DecodeString(blockIDRaw)
	if err != nil {
		return 0, err
	}

	var blockIndex uint32

	switch len(blockIDBytes) {
	case jsMaxSourceStringLength:
		// Extract the index from the block ID
		blockID := string(blockIDBytes)

		if len(blockID) < jsMaxBlockIndexLength {
			return 0, fmt.Errorf("block ID is too small to contain the index number")
		}

		rawBlockIndex := blockID[len(blockID)-jsMaxBlockIndexLength:]

		// Parse the index as an integer
		preBlockIndex, err := strconv.ParseUint(rawBlockIndex, 10, 32)
		if err != nil {
			return 0, err
		}

		blockIndex = uint32(preBlockIndex)
	case golangFixedBlockLength:
		// Reverse of what Azure SDK for Golang does[1]
		//
		// [1]: https://github.com/Azure/azure-sdk-for-go/blob/42f1cc3136bf4f0a7d9a36d634576bbf8521beef/sdk/storage/azblob/blockblob/chunkwriting.go#L243
		blockIndex = binary.BigEndian.Uint32(blockIDBytes[16:])
	default:
		return 0, fmt.Errorf("unknown block ID format: expected block ID of size %d or %d bytes, got %d bytes",
			jsMaxSourceStringLength, golangFixedBlockLength, len(blockIDBytes))
	}

	// Part numbers start with 1
	return blockIndex + 1, nil
}
