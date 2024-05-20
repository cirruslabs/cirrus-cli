package helpers

type ExitCodeError struct {
	exitCode int
	err      error
}

func NewExitCodeError(exitCode int, err error) ExitCodeError {
	return ExitCodeError{
		exitCode: exitCode,
		err:      err,
	}
}

func (exitCodeError ExitCodeError) ExitCode() int {
	return exitCodeError.exitCode
}

func (exitCodeError ExitCodeError) Error() string {
	return exitCodeError.err.Error()
}
