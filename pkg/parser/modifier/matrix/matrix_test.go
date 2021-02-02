package matrix_test

import (
	"github.com/cirruslabs/cirrus-cli/pkg/parser/modifier/matrix"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func getDocument(t *testing.T, path string, first bool) string {
	newPath := filepath.Join("testdata", path)

	file, err := os.Open(newPath)
	if err != nil {
		t.Fatalf("%s: %s", newPath, err)
	}
	defer file.Close()

	fileContentBytes, err := ioutil.ReadAll(file)
	if err != nil {
		t.Fatalf("%s: %s", newPath, err)
	}

	fileContent := string(fileContentBytes)
	divider := "---\n"

	index := strings.Index(fileContent, divider)

	if index < 0 {
		t.Fatalf("Can't find test case divider '%s' in test case %s", divider, path)
	}

	if first {
		return fileContent[:index]
	}
	return fileContent[(index + len(divider)):]
}

var goodCases = []string{
	// Just a document with an empty map
	"empty.yaml",
	// Examples from Google Docs document
	"gdoc-example1.yaml",
	"gdoc-example2.yaml",
	// Real examples from https://cirrus-ci.org/guide/writing-tasks/
	"real1.yaml",
	"real2.yaml",
	"real3.yaml",
	// Real examples from https://cirrus-ci.org/examples/
	"real4.yaml",
	"real5.yaml",
	// Encountered regressions
	"simple-slice.yaml",
	"simple-list.yaml",
	"doubly-nested-balanced.yaml",
	"doubly-nested-unbalanced.yaml",
	"matrix-inside-of-a-list-of-lists.yaml",
	"matrix-siblings.yaml",
	"multiple-matrices-on-the-same-level.yaml",
	"parallel.yaml",
	"one-sized-matrix.yaml",
	"expansion-order.yaml",
}

var badCases = []struct {
	File  string
	Error error
}{
	{"bad-matrix-without-collection.yaml", matrix.ErrMatrixNeedsCollection},
	{"bad-matrix-with-list-of-scalars.yaml", matrix.ErrMatrixNeedsListOfMaps},
	{"bad-only-task-and-docker-builder-expand.yaml", matrix.ErrMatrixIsMisplaced},
}

func runPreprocessor(input string, expand bool) (string, error) {
	var parsed yaml.Node
	err := yaml.Unmarshal([]byte(input), &parsed)
	if err != nil {
		return "", err
	}

	tree, err := node.NewFromNode(&parsed)
	if err != nil {
		return "", err
	}

	if expand {
		if err := matrix.ExpandMatrices(tree); err != nil {
			return "", err
		}
	}

	marshalYAML, err := tree.MarshalYAML()

	if err != nil {
		return "", err
	}
	if marshalYAML == nil {
		return "", nil
	}

	outputBytes, err := yaml.Marshal(&marshalYAML)
	if err != nil {
		return "", err
	}

	return string(outputBytes), nil
}

// Ensures that preprocessing works as expected.
func TestGoodCases(t *testing.T) {
	t.Parallel()
	for _, goodFile := range goodCases {
		currentFile := goodFile
		t.Run(currentFile, func(t *testing.T) {
			t.Parallel()
			input := getDocument(t, currentFile, true)
			output, err := runPreprocessor(input, true)
			if err != nil {
				t.Error(err)
				return
			}

			expectedOutput := getDocument(t, currentFile, false)
			expectedOutput, err = runPreprocessor(expectedOutput, false)
			if err != nil {
				t.Error(err)
				return
			}

			assert.Equal(t, expectedOutput, output)
		})
	}
}

// Ensures that we return correct errors for expected edge-cases.
func TestBadCases(t *testing.T) {
	for _, badCase := range badCases {
		newPath := filepath.Join("testdata", badCase.File)
		testCaseBytes, err := ioutil.ReadFile(newPath)
		if err != nil {
			assert.Equal(t, badCase.Error, err)
			continue
		}

		_, err = runPreprocessor(string(testCaseBytes), true)
		assert.Equal(t, badCase.Error, err)
	}
}
