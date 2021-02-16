package logger

type Lightweight interface {
	Debugf(format string, args ...interface{})
}

type LightweightStub struct{}

func (*LightweightStub) Debugf(format string, args ...interface{}) {}
