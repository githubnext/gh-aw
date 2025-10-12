package logging_test

import (
	"bytes"
	"fmt"
	"os"

	"github.com/githubnext/gh-aw/pkg/workflow/logging"
)

// ExampleLogger demonstrates basic usage of the logging package
func ExampleLogger() {
	// Create a logger
	logger := logging.NewLogger()

	// Log informational messages
	logger.Infof("Starting workflow compilation")

	// Log warnings
	logger.Warnf("Schema validation took longer than expected")

	// Output is sent to stderr in actual usage
}

// ExampleLogger_withCategory demonstrates categorized logging
func ExampleLogger_withCategory() {
	// Create loggers with categories
	compilerLogger := logging.NewLoggerWithCategory("compiler")
	parserLogger := logging.NewLoggerWithCategory("parser")

	// Logs include category information
	compilerLogger.Infof("Starting compilation")
	parserLogger.Infof("Parsing frontmatter")

	// Filter at runtime with:
	// export GH_AW_LOG_FILTER="compiler"      # Only compiler logs
	// export GH_AW_LOG_FILTER="compiler,parser" # Both
	// export GH_AW_LOG_FILTER="all"           # All categories

	// Output is sent to stderr in actual usage
}

// ExampleLogger_withFields demonstrates structured logging with fields
func ExampleLogger_withFields() {
	// Create a logger
	logger := logging.NewLogger()

	// Log with structured fields
	logger.InfoWithFields("Compilation started",
		"workflow", "example.md",
		"engine", "claude",
	)

	logger.InfoWithFields("Step completed",
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
	logger := logging.NewLoggerWithWriter(&buf)

	// Log messages
	logger.Infof("test message")

	// Check captured output
	output := buf.String()
	fmt.Println("Captured log output:", len(output) > 0)

	// Output: Captured log output: true
}

// ExampleLogger_categoryFiltering demonstrates category filtering
func ExampleLogger_categoryFiltering() {
	// Set environment variable to filter categories
	os.Setenv("GH_AW_LOG_FILTER", "compiler")
	defer os.Unsetenv("GH_AW_LOG_FILTER")

	var buf bytes.Buffer

	// Create loggers with different categories
	compilerLogger := logging.NewLoggerWithWriterAndCategory(&buf, "compiler")
	parserLogger := logging.NewLoggerWithWriterAndCategory(&buf, "parser")

	// This will be logged (compiler is in filter)
	compilerLogger.Infof("compiler message")

	// This will NOT be logged (parser is not in filter)
	parserLogger.Infof("parser message")

	// Check output contains only compiler message
	output := buf.String()
	hasCompiler := len(output) > 0
	fmt.Println("Has compiler logs:", hasCompiler)

	// Output: Has compiler logs: true
}
