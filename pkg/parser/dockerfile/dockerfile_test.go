package dockerfile_test

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/dockerfile"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLocalContextSourcePaths(t *testing.T) {
	dockerfileContents := `
FROM debian:latest AS builder

ARG FILE_FROM_ARG
ENV FILE_FROM_ENV=from-env.txt

# Root-scoped directory should be made relative
COPY /root-scoped-directory/ /root-scoped-directory

# Both cases should be expanded properly
COPY $FILE_FROM_ARG /from-arg.txt
COPY ${FILE_FROM_ENV} /from-env.txt

# Both cases should NOT be expanded
COPY \$FILE_FROM_ARG /escaped-variable-from-arg.txt
COPY \${FILE_FROM_ENV} /escaped-variable-from-env.txt

ADD add-works.txt /add-works.txt

FROM ubuntu:latest

# None of these should be included in the result since
# they don't copy the files from the local context
COPY --from=alpine:latest from-alpine.txt /from-alpine.txt
COPY --from=builder from-builder.txt /from-builder.txt
ADD https://example.com/index.html /index.html
`

	expected := []string{
		"$FILE_FROM_ARG",
		"${FILE_FROM_ENV}",
		"add-works.txt",
		"from-arg.txt",
		"from-env.txt",
		"root-scoped-directory",
	}

	dockerArguments := map[string]string{
		"FILE_FROM_ARG": "from-arg.txt",
	}
	actual, err := dockerfile.LocalContextSourcePaths(context.Background(), []byte(dockerfileContents), dockerArguments)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, expected, actual)
}
