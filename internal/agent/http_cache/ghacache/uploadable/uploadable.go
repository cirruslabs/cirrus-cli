package uploadable

import (
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/ghacache/httprange"
	"io"
	"math"
	"os"
	"sync"
)

type Uploadable struct {
	Key     string
	Version string

	file      *os.File
	finalized bool
	mtx       sync.Mutex
}

func New(key string, version string) (*Uploadable, error) {
	file, err := os.CreateTemp("", "")
	if err != nil {
		return nil, err
	}

	_ = os.Remove(file.Name())

	return &Uploadable{
		Key:     key,
		Version: version,
		file:    file,
	}, nil
}

func (uploadable *Uploadable) WriteChunk(contentRange string, buf []byte) error {
	uploadable.mtx.Lock()
	defer uploadable.mtx.Unlock()

	if uploadable.finalized {
		return fmt.Errorf("cannot write a chunk to a finalized uploadable")
	}

	httpRanges, err := httprange.ParseRange(contentRange, math.MaxInt64)
	if err != nil {
		return fmt.Errorf("failed to parse Content-Range header: %w", err)
	}

	if len(httpRanges) != 1 {
		return fmt.Errorf("expected a single range in the \"Content-Range\" header, "+
			"got %d ranges instead", len(httpRanges))
	}

	_, err = uploadable.file.WriteAt(buf, httpRanges[0].Start)
	if err != nil {
		return fmt.Errorf("failed to write the chunk to a file associated "+
			"with this uploadable: %w", err)
	}

	return nil
}

func (uploadable *Uploadable) Finalize() (io.ReadCloser, int64, error) {
	uploadable.mtx.Lock()
	defer uploadable.mtx.Unlock()

	if uploadable.finalized {
		return nil, 0, fmt.Errorf("cannot finalize the uploadable twice")
	}

	// Determine the file size (needed for the consumer)
	size, err := uploadable.file.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, 0, err
	}

	// Rewind the file
	if _, err := uploadable.file.Seek(0, io.SeekStart); err != nil {
		return nil, 0, err
	}

	// Mark the uploadable as finalized
	uploadable.finalized = true

	return uploadable.file, size, nil
}
