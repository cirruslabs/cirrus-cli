package piper

import (
	"context"
	"errors"
	"io"
	"os"
)

type Piper struct {
	r, w    *os.File
	errChan chan error
}

func New(output io.Writer) (*Piper, error) {
	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	piper := &Piper{
		r:       r,
		w:       w,
		errChan: make(chan error),
	}

	go func() {
		_, err := io.Copy(output, r)
		piper.errChan <- err
		_ = r.Close()
	}()

	return piper, nil
}

func (piper *Piper) FileProxy() *os.File {
	return piper.w
}

func (piper *Piper) Close(ctx context.Context, force bool) (result error) {
	// Close our writing end (if not closed yet)
	if err := piper.w.Close(); err != nil && !errors.Is(err, os.ErrClosed) && result == nil {
		result = err
	}

	// In case there might be still processes holding the writing end of the pipe,
	// forcefully terminate the Goroutine started in New() by closing the read end
	// of the pipe
	if force {
		_ = piper.r.Close()
	}

	// Wait for the Goroutine started in New(): it will reach EOF once
	// all the copies of the writing end file descriptor are closed
	select {
	case err := <-piper.errChan:
		if err != nil && !errors.Is(err, os.ErrClosed) && result == nil {
			result = err
		}
	case <-ctx.Done():
		result = ctx.Err()
	}

	return result
}
