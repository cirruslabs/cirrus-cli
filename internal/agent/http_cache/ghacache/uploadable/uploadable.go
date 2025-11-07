package uploadable

import (
	"cmp"
	"fmt"
	"sync"

	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/blobstorage"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/ghacache/rangetopart"
	"golang.org/x/exp/slices"
)

type Uploadable struct {
	key      string
	version  string
	uploadID string
	parts    map[uint32]*Part

	RangeToPart *rangetopart.RangeToPart

	finalized bool
	mtx       sync.Mutex
}

type Part struct {
	Number uint32
	ETag   string
	Size   int64
}

func New(key string, version string, uploadID string) *Uploadable {
	return &Uploadable{
		key:      key,
		version:  version,
		uploadID: uploadID,
		parts:    map[uint32]*Part{},

		RangeToPart: rangetopart.New(),
	}
}

func (uploadable *Uploadable) Key() string {
	return uploadable.key
}

func (uploadable *Uploadable) Version() string {
	return uploadable.version
}

func (uploadable *Uploadable) UploadID() string {
	return uploadable.uploadID
}

func (uploadable *Uploadable) AppendPart(number uint32, etag string, size int64) error {
	uploadable.mtx.Lock()
	defer uploadable.mtx.Unlock()

	if uploadable.finalized {
		return fmt.Errorf("cannot finalize the uploadable twice")
	}

	uploadable.parts[number] = &Part{
		Number: number,
		ETag:   etag,
		Size:   size,
	}

	return nil
}

func (uploadable *Uploadable) Finalize() ([]*blobstorage.MultipartPart, int64, error) {
	uploadable.mtx.Lock()
	defer uploadable.mtx.Unlock()

	if uploadable.finalized {
		return nil, 0, fmt.Errorf("cannot finalize the uploadable twice")
	}

	// Mark the uploadable as finalized
	uploadable.finalized = true

	var parts []*blobstorage.MultipartPart
	var partsSize int64

	for _, part := range uploadable.parts {
		parts = append(parts, &blobstorage.MultipartPart{
			PartNumber: part.Number,
			ETag:       part.ETag,
		})

		partsSize += part.Size
	}

	slices.SortFunc(parts, func(a, b *blobstorage.MultipartPart) int {
		return cmp.Compare(a.PartNumber, b.PartNumber)
	})

	return parts, partsSize, nil
}
