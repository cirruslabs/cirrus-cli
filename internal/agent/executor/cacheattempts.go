package executor

import (
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"time"
)

type CacheAttempts struct {
	cacheRetrievalAttempts map[string]*api.CacheRetrievalAttempt
}

func NewCacheAttempts() *CacheAttempts {
	return &CacheAttempts{
		cacheRetrievalAttempts: make(map[string]*api.CacheRetrievalAttempt),
	}
}

func (ca *CacheAttempts) Failed(key string, error string) {
	ca.cacheRetrievalAttempts[key] = &api.CacheRetrievalAttempt{Error: error}
}

func (ca *CacheAttempts) Hit(key string, size uint64, downloadedIn, extractedIn time.Duration) {
	ca.cacheRetrievalAttempts[key] = &api.CacheRetrievalAttempt{
		Result: &api.CacheRetrievalAttempt_Hit_{
			Hit: &api.CacheRetrievalAttempt_Hit{
				SizeBytes:         size,
				DownloadedInNanos: uint64(downloadedIn.Nanoseconds()),
				ExtractedInNanos:  uint64(extractedIn.Nanoseconds()),
			},
		},
	}
}

func (ca *CacheAttempts) PopulatedIn(key string, populatedIn time.Duration) {
	ca.cacheRetrievalAttempts[key] = &api.CacheRetrievalAttempt{
		Result: &api.CacheRetrievalAttempt_Miss_{
			Miss: &api.CacheRetrievalAttempt_Miss{
				PopulatedInNanos: uint64(populatedIn.Nanoseconds()),
			},
		},
	}
}

func (ca *CacheAttempts) Miss(key string, size uint64, archivedIn, uploadedIn time.Duration) {
	if attempt, ok := ca.cacheRetrievalAttempts[key]; ok {
		if miss, ok := attempt.Result.(*api.CacheRetrievalAttempt_Miss_); ok {
			miss.Miss.SizeBytes = size
			miss.Miss.ArchivedInNanos = uint64(archivedIn.Nanoseconds())
			miss.Miss.UploadedInNanos = uint64(uploadedIn.Nanoseconds())
			return
		}
	}

	ca.cacheRetrievalAttempts[key] = &api.CacheRetrievalAttempt{
		Result: &api.CacheRetrievalAttempt_Miss_{
			Miss: &api.CacheRetrievalAttempt_Miss{
				SizeBytes:       size,
				ArchivedInNanos: uint64(archivedIn.Nanoseconds()),
				UploadedInNanos: uint64(uploadedIn.Nanoseconds()),
			},
		},
	}
}

func (ca *CacheAttempts) ToProto() map[string]*api.CacheRetrievalAttempt {
	return ca.cacheRetrievalAttempts
}
