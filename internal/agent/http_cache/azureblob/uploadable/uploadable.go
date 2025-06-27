package uploadable

import (
	"fmt"
	"github.com/google/uuid"
	"io"
	"os"
	"sync"
)

type Uploadable struct {
	id    string
	local bool
	parts map[uint32]*Part

	mtx sync.Mutex
}

type Part struct {
	eTag string
	file *os.File
}

func (part *Part) ETag() string {
	return part.eTag
}

func (part *Part) File() *os.File {
	_, _ = part.file.Seek(0, io.SeekStart)

	return part.file
}

func New(local bool) *Uploadable {
	uploadable := &Uploadable{
		local: local,
		parts: make(map[uint32]*Part),
	}

	if local {
		uploadable.id = uuid.NewString()
	}

	return uploadable
}

func (uploadable *Uploadable) ID() (string, bool) {
	uploadable.mtx.Lock()
	defer uploadable.mtx.Unlock()

	// Return ID and whether it was already computed
	return uploadable.id, uploadable.id != ""
}

func (uploadable *Uploadable) IDOrCompute(compute func() (string, error)) (string, error) {
	uploadable.mtx.Lock()
	defer uploadable.mtx.Unlock()

	// Return ID if already computed
	if uploadable.id != "" {
		return uploadable.id, nil
	}

	// Compute otherwise
	uploadID, err := compute()
	if err != nil {
		return "", err
	}

	uploadable.id = uploadID

	return uploadID, nil
}

func (uploadable *Uploadable) Local() bool {
	return uploadable.local
}

func (uploadable *Uploadable) AppendPart(number uint32, eTag string) error {
	uploadable.mtx.Lock()
	defer uploadable.mtx.Unlock()

	if _, ok := uploadable.parts[number]; ok {
		return fmt.Errorf("part %d already exists", number)
	}

	uploadable.parts[number] = &Part{
		eTag: eTag,
	}

	return nil
}

func (uploadable *Uploadable) AppendPartLocal(number uint32, r io.Reader) error {
	pattern := fmt.Sprintf("uploadable-%s-part-%d-*",
		uploadable.id, number)

	file, err := os.CreateTemp("", pattern)
	if err != nil {
		return err
	}

	if _, err := io.Copy(file, r); err != nil {
		return err
	}

	uploadable.mtx.Lock()
	defer uploadable.mtx.Unlock()

	if _, ok := uploadable.parts[number]; ok {
		_ = file.Close()
		_ = os.Remove(file.Name())

		return fmt.Errorf("part %d already exists", number)
	}

	uploadable.parts[number] = &Part{
		file: file,
	}

	return nil
}

func (uploadable *Uploadable) GetPart(number uint32) *Part {
	uploadable.mtx.Lock()
	defer uploadable.mtx.Unlock()

	return uploadable.parts[number]
}
