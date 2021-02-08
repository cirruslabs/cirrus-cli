package parser_test

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/pkg/parser"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBfuncChangesInclude(t *testing.T) {
	affectedFiles := []string{"note.txt", "drawing.svg", "go.mod", "dir/file.go"}

	config := `
container:
  image: debian:latest

simple_task:
  only_if: "changesInclude('*.txt')"
  script: true

complex_task:
  only_if: "changesInclude('document.pdf', '**.{png,bmp,svg}')"
  script: true

inverted_task:
  only_if: "!changesInclude('*.go')"
  script: true

doublestar_task:
  only_if: "!changesInclude('**.go')"
  script: true
`

	p := parser.New(parser.WithAffectedFiles(affectedFiles))
	result, err := p.Parse(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, result.Tasks, 3)
}
