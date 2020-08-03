package cache

import (
	"bufio"
	"fmt"
	"os"
)

type PutOperation struct {
	tmpBlobFile   *os.File
	tmpBlobWriter *bufio.Writer
	finalBlobPath string
}

func (putOp *PutOperation) Write(b []byte) (int, error) {
	n, err := putOp.tmpBlobWriter.Write(b)
	if err != nil {
		return n, fmt.Errorf("%w: %v", ErrInternal, err)
	}

	return n, nil
}

func (putOp *PutOperation) Finalize() error {
	// Close the wrapped buffered I/O writer and file, so that their respective internal buffers are flushed
	if err := putOp.tmpBlobWriter.Flush(); err != nil {
		return fmt.Errorf("%w: %v", ErrInternal, err)
	}
	if err := putOp.tmpBlobFile.Close(); err != nil {
		return fmt.Errorf("%w: %v", ErrInternal, err)
	}

	// Atomically move the wrapped tmpBlobFile to it's final place
	if err := os.Rename(putOp.tmpBlobFile.Name(), putOp.finalBlobPath); err != nil {
		return fmt.Errorf("%w: %v", ErrInternal, err)
	}

	return nil
}
