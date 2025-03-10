package progressreader

import (
	"io"
	"time"
)

type LogFunc func(bytes int64, duration time.Duration)

type Reader struct {
	inner             io.Reader
	lastLog           time.Time
	interval          time.Duration
	logFunc           LogFunc
	bytesSinceLastLog int64
}

func New(inner io.Reader, interval time.Duration, logFunc LogFunc) *Reader {
	return &Reader{
		inner:    inner,
		lastLog:  time.Now(),
		interval: interval,
		logFunc:  logFunc,
	}
}

func (reader *Reader) Read(p []byte) (int, error) {
	n, err := reader.inner.Read(p)

	reader.bytesSinceLastLog += int64(n)

	if durationSinceLastLog := time.Since(reader.lastLog); durationSinceLastLog >= reader.interval {
		reader.logFunc(reader.bytesSinceLastLog, durationSinceLastLog)

		reader.lastLog = time.Now()
		reader.bytesSinceLastLog = 0
	}

	return n, err
}
