package uploadable

import (
	"fmt"
	"sync"
)

type Uploadable struct {
	id    string
	parts map[int]*Part

	mtx sync.Mutex
}

type Part struct {
	ETag string
}

func New() *Uploadable {
	return &Uploadable{
		parts: make(map[int]*Part),
	}
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

func (uploadable *Uploadable) AppendPart(number int, eTag string) error {
	uploadable.mtx.Lock()
	defer uploadable.mtx.Unlock()

	if _, ok := uploadable.parts[number]; ok {
		return fmt.Errorf("part %d already exists", number)
	}

	uploadable.parts[number] = &Part{
		ETag: eTag,
	}

	return nil
}

func (uploadable *Uploadable) GetPart(number int) *Part {
	uploadable.mtx.Lock()
	defer uploadable.mtx.Unlock()

	return uploadable.parts[number]
}
