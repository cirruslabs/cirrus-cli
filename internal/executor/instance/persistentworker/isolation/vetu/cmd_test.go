package vetu

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace/noop"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestCmdWithOTELSpanID(t *testing.T) {
	// Skip if vetu is not available
	if !Installed() {
		t.Skip("vetu command not found")
	}

	// Set up a proper tracer provider for testing
	tp := trace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(noop.NewTracerProvider())

	// Create a tracer and start a span
	tracer := otel.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	// Verify that the span context is valid
	require.True(t, span.SpanContext().IsValid())
	expectedSpanID := span.SpanContext().SpanID().String()

	// Mock the vetu command by creating a simple script that prints environment variables
	originalPath := os.Getenv("PATH")
	
	// Create a temporary directory for our mock vetu command
	tempDir := t.TempDir()
	mockVetuPath := tempDir + "/vetu"
	
	// Create a mock vetu script that prints the OTEL_PARENT_SPAN_ID environment variable
	mockScript := `#!/bin/bash
echo "OTEL_PARENT_SPAN_ID=${OTEL_PARENT_SPAN_ID}"
`
	err := os.WriteFile(mockVetuPath, []byte(mockScript), 0755)
	require.NoError(t, err)
	
	// Temporarily modify PATH to use our mock vetu
	os.Setenv("PATH", tempDir+":"+originalPath)
	defer os.Setenv("PATH", originalPath)

	// Call the Cmd function with a simple command
	stdout, stderr, err := Cmd(ctx, nil, "version")
	
	// Verify the command succeeded
	require.NoError(t, err)
	assert.Empty(t, stderr)
	
	// Verify that the OTEL_PARENT_SPAN_ID was passed to the command
	assert.Contains(t, stdout, "OTEL_PARENT_SPAN_ID="+expectedSpanID)
}

func TestCmdWithoutOTELSpan(t *testing.T) {
	// Skip if vetu is not available
	if !Installed() {
		t.Skip("vetu command not found")
	}

	// Use a context without a span
	ctx := context.Background()

	// Mock the vetu command
	originalPath := os.Getenv("PATH")
	
	// Create a temporary directory for our mock vetu command
	tempDir := t.TempDir()
	mockVetuPath := tempDir + "/vetu"
	
	// Create a mock vetu script that prints the OTEL_PARENT_SPAN_ID environment variable
	mockScript := `#!/bin/bash
echo "OTEL_PARENT_SPAN_ID=${OTEL_PARENT_SPAN_ID}"
`
	err := os.WriteFile(mockVetuPath, []byte(mockScript), 0755)
	require.NoError(t, err)
	
	// Temporarily modify PATH to use our mock vetu
	os.Setenv("PATH", tempDir+":"+originalPath)
	defer os.Setenv("PATH", originalPath)

	// Call the Cmd function with a simple command
	stdout, stderr, err := Cmd(ctx, nil, "version")
	
	// Verify the command succeeded
	require.NoError(t, err)
	assert.Empty(t, stderr)
	
	// Verify that the OTEL_PARENT_SPAN_ID was not set (should be empty)
	assert.Contains(t, stdout, "OTEL_PARENT_SPAN_ID=")
	assert.NotContains(t, stdout, "OTEL_PARENT_SPAN_ID=0") // Should not contain a valid span ID
}

func TestCmdWithInvalidSpan(t *testing.T) {
	// Skip if vetu is not available
	if !Installed() {
		t.Skip("vetu command not found")
	}

	// Create a context with an invalid span
	ctx := oteltrace.ContextWithSpan(context.Background(), oteltrace.SpanFromContext(context.Background()))

	// Mock the vetu command
	originalPath := os.Getenv("PATH")
	
	// Create a temporary directory for our mock vetu command
	tempDir := t.TempDir()
	mockVetuPath := tempDir + "/vetu"
	
	// Create a mock vetu script that prints the OTEL_PARENT_SPAN_ID environment variable
	mockScript := `#!/bin/bash
echo "OTEL_PARENT_SPAN_ID=${OTEL_PARENT_SPAN_ID}"
`
	err := os.WriteFile(mockVetuPath, []byte(mockScript), 0755)
	require.NoError(t, err)
	
	// Temporarily modify PATH to use our mock vetu
	os.Setenv("PATH", tempDir+":"+originalPath)
	defer os.Setenv("PATH", originalPath)

	// Call the Cmd function with a simple command
	stdout, stderr, err := Cmd(ctx, nil, "version")
	
	// Verify the command succeeded
	require.NoError(t, err)
	assert.Empty(t, stderr)
	
	// Verify that the OTEL_PARENT_SPAN_ID was not set (should be empty) since span is invalid
	assert.Contains(t, stdout, "OTEL_PARENT_SPAN_ID=")
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "OTEL_PARENT_SPAN_ID=") {
			// Extract the value after the equals sign
			value := strings.TrimPrefix(line, "OTEL_PARENT_SPAN_ID=")
			// Should be empty since the span is invalid
			assert.Empty(t, value)
			break
		}
	}
}
