// Package cli provides command-line interface functionality for gh-aw.
// This file (logs_utils.go) contains utility functions used by the logs command.
//
// Key responsibilities:
//   - Discovering agentic workflow names from .lock.yml files
//   - Configuration loading from environment variables
//   - Utility functions for slice operations
package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var logsUtilsLog = logger.New("cli:logs_utils")

// getIntFromEnv is a generic helper that reads an integer value from an environment variable,
// validates it against min/max bounds, and returns a default value if invalid.
// This follows the configuration helper pattern from pkg/workflow/config_helpers.go.
//
// Parameters:
//   - envVar: The environment variable name (e.g., "GH_AW_MAX_CONCURRENT_DOWNLOADS")
//   - defaultValue: The default value to return if env var is not set or invalid
//   - minValue: Minimum allowed value (inclusive)
//   - maxValue: Maximum allowed value (inclusive)
//   - log: Optional logger for debug output
//
// Returns the parsed integer value, or defaultValue if:
//   - Environment variable is not set
//   - Value cannot be parsed as an integer
//   - Value is outside the [minValue, maxValue] range
//
// Invalid values trigger warning messages to stderr.
func getIntFromEnv(envVar string, defaultValue, minValue, maxValue int, log *logger.Logger) int {
	envValue := os.Getenv(envVar)
	if envValue == "" {
		return defaultValue
	}

	val, err := strconv.Atoi(envValue)
	if err != nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(
			fmt.Sprintf("Invalid %s value '%s' (must be a number), using default %d", envVar, envValue, defaultValue),
		))
		return defaultValue
	}

	if val < minValue || val > maxValue {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(
			fmt.Sprintf("%s value %d is out of bounds (must be %d-%d), using default %d", envVar, val, minValue, maxValue, defaultValue),
		))
		return defaultValue
	}

	if log != nil {
		log.Printf("Using %s=%d", envVar, val)
	}
	return val
}

// getAgenticWorkflowNames reads all .lock.yml files and extracts their workflow names
func getAgenticWorkflowNames(verbose bool) ([]string, error) {
	logsUtilsLog.Print("Discovering agentic workflow names from .lock.yml files")
	var workflowNames []string

	// Look for .lock.yml files in .github/workflows directory
	workflowsDir := ".github/workflows"
	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		if verbose {
			fmt.Println(console.FormatWarningMessage("No .github/workflows directory found"))
		}
		return workflowNames, nil
	}

	files, err := filepath.Glob(filepath.Join(workflowsDir, "*.lock.yml"))
	if err != nil {
		return nil, fmt.Errorf("failed to glob .lock.yml files: %w", err)
	}

	for _, file := range files {
		if verbose {
			fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Reading workflow file: %s", file)))
		}

		content, err := os.ReadFile(file)
		if err != nil {
			if verbose {
				fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Failed to read %s: %v", file, err)))
			}
			continue
		}

		// Extract the workflow name using simple string parsing
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "name:") {
				// Parse the name field
				parts := strings.SplitN(trimmed, ":", 2)
				if len(parts) == 2 {
					name := strings.TrimSpace(parts[1])
					// Remove quotes if present
					name = strings.Trim(name, `"'`)
					if name != "" {
						workflowNames = append(workflowNames, name)
						if verbose {
							fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Found agentic workflow: %s", name)))
						}
						break
					}
				}
			}
		}
	}

	if verbose {
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Found %d agentic workflows", len(workflowNames))))
	}

	return workflowNames, nil
}
