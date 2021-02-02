package matrix_test

import (
	"github.com/cirruslabs/cirrus-cli/pkg/parser/modifier/matrix"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

// Retrieves the specified document (where the first document index is 1) from YAML file located at path.
func getDocument(t *testing.T, path string, index int) string {
	newPath := filepath.Join("testdata", path)

	file, err := os.Open(newPath)
	if err != nil {
		t.Fatalf("%s: %s", newPath, err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	var document yaml.Node

	for i := 0; i <= index; i++ {
		if err := decoder.Decode(&document); err != nil {
			t.Fatalf("%s: %s", newPath, err)
		}
	}

	// the actual node is the first and only child of the document
	bytes, err := yaml.Marshal(document.Content[0])
	if err != nil {
		t.Fatalf("%s: %s", newPath, err)
	}

	return string(bytes)
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
			input := getDocument(t, currentFile, 0)
			output, err := runPreprocessor(input, true)
			if err != nil {
				t.Error(err)
				return
			}

			expectedOutput := getDocument(t, currentFile, 1)
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
	t.Parallel()
	for _, badCase := range badCases {
		currentCase := badCase
		t.Run(currentCase.File, func(t *testing.T) {
			t.Parallel()
			newPath := filepath.Join("testdata", currentCase.File)
			testCaseBytes, err := ioutil.ReadFile(newPath)
			if err != nil {
				assert.Equal(t, currentCase.Error, err)
				return
			}

			_, err = runPreprocessor(string(testCaseBytes), true)
			assert.Equal(t, currentCase.Error, err)
		})
	}
}
