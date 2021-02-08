package parsererror_test

import (
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parsererror"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMultiLineMessageWithContext(t *testing.T) {
	pe := parsererror.NewRich(8, 3, "unacceptable condition")
	pe.Enrich(`first_task:
  container:
    image: debian:latest

  script: echo "I'm a first task!"

second_task:
  containers:

  script: true

third_task:
  container:
    image: debian:latest

  script: echo "I'm a third task!"
`)

	expected := `3:     image: debian:latest
4: 
5:   script: echo "I'm a first task!"
6: 
7: second_task:
8:   containers:
     ^
9: 
10:   script: true
11: 
12: third_task:
13:   container:
`

	assert.Equal(t, expected, pe.ContextLines())
}
