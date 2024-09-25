package cirrusenv

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/google/uuid"
	"os"
	"path/filepath"
	"strings"
)

type CirrusEnv struct {
	filepath string
}

func New(taskID string) (*CirrusEnv, error) {
	filename := fmt.Sprintf("cirrus-env-task-%s-%s", taskID, uuid.New().String())
	filepath := filepath.Join(os.TempDir(), filename)

	cirrusEnvFile, err := os.Create(filepath)
	if err != nil {
		return nil, err
	}

	if err := cirrusEnvFile.Close(); err != nil {
		return nil, err
	}

	return &CirrusEnv{
		filepath: filepath,
	}, nil
}

func (ce *CirrusEnv) Path() string {
	return ce.filepath
}

func (ce *CirrusEnv) Consume() (map[string]string, error) {
	result := map[string]string{}

	fileBytes, err := os.ReadFile(ce.filepath)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(fileBytes)
	scanner := bufio.NewScanner(buf)

	for scanner.Scan() {
		splits := strings.SplitN(scanner.Text(), "=", 2)
		if len(splits) != 2 {
			return nil, fmt.Errorf("CIRRUS_ENV file should contain lines in KEY=VALUE format")
		}

		result[splits[0]] = splits[1]
	}

	return result, nil
}

func (ce *CirrusEnv) Close() error {
	return os.Remove(ce.Path())
}
