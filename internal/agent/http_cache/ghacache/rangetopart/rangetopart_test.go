package rangetopart_test

import (
	"context"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache/ghacache/rangetopart"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestRangeToPartSimple(_ *testing.T) {
	ctx := context.Background()

	rangeToPart := rangetopart.New()

	fmt.Println(rangeToPart.Tell(ctx, 0, 15))
	fmt.Println(rangeToPart.Tell(ctx, 15, 15))
	fmt.Println(rangeToPart.Tell(ctx, 30, 32))
}

func TestRangeToPartContention(t *testing.T) {
	ctx := context.Background()

	rangeToPart := rangetopart.New()

	secondRangePartCh := make(chan int32, 1)

	go func() {
		part, err := rangeToPart.Tell(ctx, 15, 15)
		if err != nil {
			panic(err)
		}

		secondRangePartCh <- part
	}()

	time.Sleep(time.Second)

	firstRangePartCh := make(chan int32, 1)

	go func() {
		part, err := rangeToPart.Tell(ctx, 0, 15)
		if err != nil {
			panic(err)
		}

		firstRangePartCh <- part
	}()

	assert.EqualValues(t, 1, <-firstRangePartCh)
	assert.EqualValues(t, 2, <-secondRangePartCh)
}
