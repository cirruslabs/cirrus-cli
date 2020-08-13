package commandstatus

import "fmt"

type Status int

const (
	Undefined Status = iota
	Success
	Failure
)

func (status Status) String() string {
	switch status {
	case Undefined:
		return "undefined"
	case Success:
		return "succeeded"
	case Failure:
		return "failed"
	default:
		return fmt.Sprintf("entered unhandled status %d", int(status))
	}
}
