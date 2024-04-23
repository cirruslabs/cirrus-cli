package terminalwrapper

type Operation interface {
	isOperation()
}

type LogOperation struct {
	Message string
}

func (*LogOperation) isOperation() {}

type ExitOperation struct {
	Success bool
}

func (*ExitOperation) isOperation() {}
