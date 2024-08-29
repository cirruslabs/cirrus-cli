package rangetopart

import (
	"context"
	"errors"
	"sync/atomic"
)

var ErrUnevenChunkSize = errors.New("cannot figure out the part number because uneven chunk size is used")

type RangeToPart struct {
	firstRangeLength atomic.Int64
	firstRangeCtx    context.Context
	firstRangeCancel context.CancelFunc
}

func New() *RangeToPart {
	ctx, cancel := context.WithCancel(context.Background())

	return &RangeToPart{
		firstRangeCtx:    ctx,
		firstRangeCancel: cancel,
	}
}

func (rangeToPart *RangeToPart) Tell(ctx context.Context, start int64, length int64) (int32, error) {
	// If it's the first range, then its part number is 1
	if start == 0 {
		rangeToPart.firstRangeLength.Store(length)
		rangeToPart.firstRangeCancel()

		return 1, nil
	}

	// Wait for the first range in another Tell() invocation
	// and calculate the part number by dividing the current
	// range start by the first range length
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case <-rangeToPart.firstRangeCtx.Done():
		firstRangeLength := rangeToPart.firstRangeLength.Load()

		if start%firstRangeLength != 0 {
			return 0, ErrUnevenChunkSize
		}

		return int32(1 + (start / firstRangeLength)), nil
	}
}
