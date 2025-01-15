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
		//nolint:gosec // G404 is not applicable as we don't need cryptographically secure numbers here
		nextPullAt.Add(time.Duration(rand.Int64N(jitterNanoseconds)))
	}

	return time.Now().After(nextPullAt)
}
