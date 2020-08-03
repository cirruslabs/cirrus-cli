package cache

import (
	"fmt"
	"os"
)

type PutOperation struct {
	tmpBlobFile   *os.File
	finalBlobPath string
}

func (putOp *PutOperation) Write(b []byte) (int, error) {
	n, err := putOp.tmpBlobFile.Write(b)
	if err != nil {
		return n, fmt.Errorf("%w: %v", ErrInternal, err)
	}

	return n, nil
}

func (putOp *PutOperation) Finalize() error {
	// Close the wrapped tmpBlobFile so that internal buffers (if any) are flushed
	if err := putOp.tmpBlobFile.Close(); err != nil {
		return fmt.Errorf("%w: %v", ErrInternal, err)
	}

	// Atomically move the wrapped tmpBlobFile to it's final place
	if err := os.Rename(putOp.tmpBlobFile.Name(), putOp.finalBlobPath); err != nil {
		return fmt.Errorf("%w: %v", ErrInternal, err)
	}

	return nil
}
