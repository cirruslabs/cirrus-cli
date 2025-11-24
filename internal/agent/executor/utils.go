package executor

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"github.com/cirruslabs/cirrus-ci-annotations/model"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

func TempFileName(prefix, suffix string) (*os.File, error) {
	randBytes := make([]byte, 16)
	for i := 0; i < 10000; i++ {
		rand.Read(randBytes)
		path := filepath.Join(os.TempDir(), prefix+hex.EncodeToString(randBytes)+suffix)
		f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0644)
		if os.IsExist(err) {
			continue
		}
		return f, err
	}
	return nil, errors.New("failed to create temp file")
}

func EnsureFolderExists(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, 0755)
		if err != nil {
			slog.Warn("Failed to mkdir", "path", path, "err", err)
		}
	}
}

func allDirsEmpty(paths []string) bool {
	for _, path := range paths {
		if !isDirEmpty(path) {
			return false
		}
	}

	return true
}

func isDirEmpty(path string) bool {
	files, err := os.ReadDir(path)
	if os.IsNotExist(err) {
		return true
	}
	if err != nil {
		return false
	}
	return len(files) == 0
}

func ConvertAnnotations(annotations []model.Annotation) []*api.Annotation {
	result := make([]*api.Annotation, 0)
	for _, annotation := range annotations {
		protoAnnotation := api.Annotation{
			Type:       api.Annotation_GENERIC,
			Level:      api.Annotation_Level(api.Annotation_Level_value[strings.ToUpper(annotation.Level.String())]),
			Message:    annotation.Message,
			RawDetails: annotation.RawDetails,
		}
		protoAnnotation.FileLocation = &api.Annotation_FileLocation{
			Path:        annotation.Path,
			StartLine:   annotation.StartLine,
			EndLine:     annotation.EndLine,
			StartColumn: annotation.StartColumn,
			EndColumn:   annotation.EndColumn,
		}
		result = append(result, &protoAnnotation)
	}
	return result
}
