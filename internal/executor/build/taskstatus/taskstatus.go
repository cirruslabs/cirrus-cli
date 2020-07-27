package taskstatus

import "fmt"

type Status int

const (
	New Status = iota
	Succeeded
	Failed
	TimedOut
)

func (status Status) String() string {
	switch status {
	case New:
		return "new"
	case Succeeded:
		return "succeeded"
	case Failed:
		return "failed"
	case TimedOut:
		return "timed out"
	default:
		return fmt.Sprintf("entered unhandled status %d", int(status))
	}
}
