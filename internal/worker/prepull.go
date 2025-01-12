package worker

import "time"

type TartPrePull struct {
	Images        []string      `yaml:"images"`
	CheckInterval time.Duration `yaml:"check-interval"`
	LastCheck     time.Time
}

func (pull TartPrePull) NeedsPrePull() bool {
	if pull.CheckInterval == 0 {
		return true
	}

	return time.Now().After(pull.LastCheck.Add(pull.CheckInterval))
}
