package uploadable

import (
	"cmp"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/ghacache/rangetopart"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"golang.org/x/exp/slices"
	"sync"
)

type Uploadable struct {
	key      string
	version  string
	uploadID string
	parts    map[uint32]*Part

	RangeToPart *rangetopart.RangeToPart

	mtx sync.Mutex
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

func (uploadable *Uploadable) AppendPart(number uint32, etag string, size int64) {
	uploadable.mtx.Lock()
	defer uploadable.mtx.Unlock()

	uploadable.parts[number] = &Part{
		Number: number,
		ETag:   etag,
		Size:   size,
	}
}

func (uploadable *Uploadable) BuildCommitRequestParts() ([]*api.MultipartCacheUploadCommitRequest_Part, int64) {
	uploadable.mtx.Lock()
	defer uploadable.mtx.Unlock()

	var parts []*api.MultipartCacheUploadCommitRequest_Part
	var partsSize int64

	for _, part := range uploadable.parts {
		parts = append(parts, &api.MultipartCacheUploadCommitRequest_Part{
			PartNumber: part.Number,
			Etag:       part.ETag,
		})

		partsSize += part.Size
	}

	slices.SortFunc(parts, func(a, b *api.MultipartCacheUploadCommitRequest_Part) int {
		return cmp.Compare(a.PartNumber, b.PartNumber)
	})

	return parts, partsSize
}
