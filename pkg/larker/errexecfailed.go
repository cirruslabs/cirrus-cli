package larker

import "fmt"

type ErrExecFailed struct {
	err error
}

func (eee *ErrExecFailed) Error() string {
	return fmt.Sprintf("exec failed: %v", eee.err)
}

func (eee *ErrExecFailed) Unwrap() error {
	return eee.err
}
