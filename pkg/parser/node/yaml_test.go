package node_test

import (
	"fmt"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/go-test/deep"
	"testing"
)

func TestLineAndColumnNumbers(t *testing.T) {
	config := `sequence_test:
  - 1
  - 2

mapping_test:
  first_key: first_value
  second_key: second_value

anchors_test: &defaults
  STATE: OFF

aliases_test:
  <<: *defaults
`

	actual, err := node.NewFromText(config)
	if err != nil {
		t.Fatal(err)
	}

	root := &node.Node{
		Name:   "root",
		Value:  &node.MapValue{},
		Line:   1,
		Column: 1,
	}

	sequenceTestNode := &node.Node{
		Name:   "sequence_test",
		Value:  &node.ListValue{},
		Parent: root,
		Line:   1,
		Column: 1,
	}

	sequenceTestNode.Children = []*node.Node{
		{Parent: sequenceTestNode, Value: &node.ScalarValue{Value: "1"}, Line: 2, Column: 5},
		{Parent: sequenceTestNode, Value: &node.ScalarValue{Value: "2"}, Line: 3, Column: 5},
	}

	mappingTestNode := &node.Node{
		Name:   "mapping_test",
		Value:  &node.MapValue{},
		Parent: root,
		Line:   5,
		Column: 1,
	}

	mappingTestNode.Children = []*node.Node{
		{Name: "first_key", Parent: mappingTestNode, Value: &node.ScalarValue{Value: "first_value"}, Line: 6, Column: 3},
		{Name: "second_key", Parent: mappingTestNode, Value: &node.ScalarValue{Value: "second_value"}, Line: 7, Column: 3},
	}

	anchorsTestNode := &node.Node{
		Name:   "anchors_test",
		Value:  &node.MapValue{},
		Parent: root,
		Line:   9,
		Column: 1,
	}

	anchorsTestNode.Children = []*node.Node{
		{Name: "STATE", Parent: anchorsTestNode, Value: &node.ScalarValue{Value: "OFF"}, Line: 10, Column: 3},
	}

	aliasesTestNode := &node.Node{
		Name:   "aliases_test",
		Value:  &node.MapValue{},
		Parent: root,
		Line:   12,
		Column: 1,
	}

	aliasesTestNode.Children = []*node.Node{
		{Name: "STATE", Parent: aliasesTestNode, Value: &node.ScalarValue{Value: "OFF"}, Line: 10, Column: 3},
	}

	root.Children = []*node.Node{
		sequenceTestNode,
		mappingTestNode,
		anchorsTestNode,
		aliasesTestNode,
	}

	if diff := deep.Equal(root, actual); diff != nil {
		fmt.Println(diff)
		t.Fail()
	}
}
