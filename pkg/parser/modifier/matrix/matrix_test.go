package matrix_test

import (
	"github.com/cirruslabs/cirrus-cli/pkg/parser/modifier/matrix"
	"github.com/go-test/deep"
	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
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

	decoder := yaml.NewDecoder(file, yaml.UseOrderedMap())
	var document yaml.MapSlice

	for i := 0; i < index; i++ {
		if err := decoder.Decode(&document); err != nil {
			t.Fatalf("%s: %s", newPath, err)
		}
	}

	bytes, err := yaml.Marshal(document)
	if err != nil {
		t.Fatalf("%s: %s", newPath, err)
	}

	return string(bytes)
}

// Unmarshals YAML specified by yamlText to a yaml.MapSlice to simplify comparison.
func yamlAsStruct(t *testing.T, yamlText string) (result yaml.MapSlice) {
	if err := yaml.UnmarshalWithOptions([]byte(yamlText), &result, yaml.UseOrderedMap()); err != nil {
		t.Fatal(err)
	}

	return
}

var goodCases = []string{
	// Just a document with an empty map
	"empty.yaml",
	// Ensure that only normal tasks and Docker Builder tasks are expanded when matrix modification is used
	"only-task-and-docker-builder-expand.yaml",
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
}

var badCases = []struct {
	File  string
	Error error
}{
	{"bad-matrix-without-collection.yaml", matrix.ErrMatrixNeedsCollection},
	{"bad-matrix-with-list-of-scalars.yaml", matrix.ErrMatrixNeedsListOfMaps},
}

func runPreprocessor(input string) (string, error) {
	var tree yaml.MapSlice
	err := yaml.UnmarshalWithOptions([]byte(input), &tree, yaml.UseOrderedMap())
	if err != nil {
		return "", err
	}

	expanded, err := matrix.ExpandMatrices(tree)
	if err != nil {
		return "", err
	}

	outputBytes, err := yaml.Marshal(&expanded)
	if err != nil {
		return "", err
	}

	return string(outputBytes), nil
}

// Ensures that preprocessing works as expected.
func TestGoodCases(t *testing.T) {
	for _, goodFile := range goodCases {
		input := getDocument(t, goodFile, 1)
		expectedOutput := getDocument(t, goodFile, 2)

		output, err := runPreprocessor(input)
		if err != nil {
			t.Error(err)
			continue
		}

		t.Run(goodFile, func(t *testing.T) {
			diff := deep.Equal(yamlAsStruct(t, expectedOutput), yamlAsStruct(t, output))
			if diff != nil {
				t.Error("found difference")
				for _, d := range diff {
					t.Log(d)
				}
			}
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

		_, err = runPreprocessor(string(testCaseBytes))
		assert.Equal(t, badCase.Error, err)
	}
}
