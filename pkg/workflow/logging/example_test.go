package logging_test

import (
	"bytes"
	"fmt"

	"github.com/githubnext/gh-aw/pkg/workflow/logging"
)

// ExampleLogger demonstrates basic usage of the logging package
func ExampleLogger() {
	// Create a logger with verbose mode enabled
	logger := logging.NewLogger(true)

	// Log informational messages
	logger.Infof("Starting workflow compilation")

	// Log debug messages (only shown in verbose mode)
	logger.Debugf("Processing step %d of %d", 1, 5)

	// Log warnings
	logger.Warnf("Schema validation took longer than expected")

	// Output is sent to stderr in actual usage
}

// ExampleLogger_withFields demonstrates structured logging with fields
func ExampleLogger_withFields() {
	// Create a logger
	logger := logging.NewLogger(true)

	// Log with structured fields
	logger.InfoWithFields("Compilation started",
		"workflow", "example.md",
		"engine", "claude",
		"verbose", true,
	)

	logger.DebugWithFields("Step completed",
		"step", 3,
		"duration", "1.2s",
		"status", "success",
	)

	// Output is sent to stderr in actual usage
}

// ExampleNewLoggerWithWriter demonstrates using a custom writer for testing
func ExampleNewLoggerWithWriter() {
	// Create a buffer to capture log output
	var buf bytes.Buffer

	// Create logger with custom writer
	logger := logging.NewLoggerWithWriter(true, &buf)

	// Log messages
	logger.Infof("test message")

	// Check captured output
	output := buf.String()
	fmt.Println("Captured log output:", len(output) > 0)

	// Output: Captured log output: true
}

// ExampleLogger_IsVerbose demonstrates checking verbose mode
func ExampleLogger_IsVerbose() {
	verboseLogger := logging.NewLogger(true)
	quietLogger := logging.NewLogger(false)

	fmt.Println("Verbose logger:", verboseLogger.IsVerbose())
	fmt.Println("Quiet logger:", quietLogger.IsVerbose())

	// Output:
	// Verbose logger: true
	// Quiet logger: false
}
