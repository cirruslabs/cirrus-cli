package larker

import "errors"

type ExtendedError struct {
	err  error
	logs []byte
}

func (ee *ExtendedError) Error() string {
	return ee.err.Error()
}

func (ee *ExtendedError) Unwrap() error {
	return ee.err
}

func (ee *ExtendedError) Logs() []byte {
	return ee.logs
}

func (ee *ExtendedError) Is(err error) bool {
	return errors.Is(ee.err, err)
}
