package larker

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
