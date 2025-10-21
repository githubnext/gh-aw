package logger_test

import (
	"fmt"
	"os"

	"github.com/githubnext/gh-aw/pkg/logger"
)

func ExampleNew() {
	// Set DEBUG environment variable to enable loggers
	os.Setenv("DEBUG", "app:*")
	defer os.Unsetenv("DEBUG")

	// Create a logger for a specific namespace
	log := logger.New("app:feature")

	// Check if logger is enabled
	if log.Enabled() {
		fmt.Println("Logger is enabled")
	}

	// Output: Logger is enabled
}

func ExampleLogger_Printf() {
	// Enable all loggers
	os.Setenv("DEBUG", "*")
	defer os.Unsetenv("DEBUG")

	log := logger.New("app:feature")
	
	// Printf uses standard fmt.Printf formatting
	log.Printf("Processing %d items", 42)
	
	// Output to stderr: app:feature Processing 42 items
}

func ExampleLogger_LazyPrintf() {
	os.Setenv("DEBUG", "app:*")
	defer os.Unsetenv("DEBUG")

	log := logger.New("app:expensive")
	
	// The lazy function is only called if the logger is enabled
	log.LazyPrintf(func() string {
		// This expensive operation only runs when logging is enabled
		result := "computed result"
		return fmt.Sprintf("Expensive computation: %s", result)
	})
	
	// Output to stderr: app:expensive Expensive computation: computed result
}

func ExampleNew_patterns() {
	// Example patterns for DEBUG environment variable
	
	// Enable all loggers
	os.Setenv("DEBUG", "*")
	
	// Enable all loggers in workflow namespace
	os.Setenv("DEBUG", "workflow:*")
	
	// Enable multiple namespaces
	os.Setenv("DEBUG", "workflow:*,cli:*")
	
	// Enable all except specific patterns
	os.Setenv("DEBUG", "*,-workflow:test")
	
	// Enable namespace but exclude specific loggers
	os.Setenv("DEBUG", "workflow:*,-workflow:cache")
	
	defer os.Unsetenv("DEBUG")
}
