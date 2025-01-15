package worker

import (
	"math/rand/v2"
	"time"
)

type TartPrePull struct {
	Images        []string      `yaml:"images"`
	CheckInterval time.Duration `yaml:"check-interval"`
	Jitter        time.Duration `yaml:"jitter"`
	LastCheck     time.Time
}

func (pull TartPrePull) NeedsPrePull() bool {
	if pull.CheckInterval == 0 {
		return true
	}

	nextPullAt := pull.LastCheck.Add(pull.CheckInterval)

	if jitterNanoseconds := pull.Jitter.Nanoseconds(); jitterNanoseconds > 0 {
		nextPullAt.Add(time.Duration(rand.Int64N(jitterNanoseconds)))
	}

	return time.Now().After(nextPullAt)
}
