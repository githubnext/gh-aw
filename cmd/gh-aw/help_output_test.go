//go:build integration

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestHelpOutputGolden verifies that the help output matches the golden file.
// This test catches unintended changes to the help text that would affect user experience.
//
// To update the golden file when making intentional changes:
//
//	UPDATE_GOLDEN=1 go test -v -tags integration -run TestHelpOutputGolden ./cmd/gh-aw
func TestHelpOutputGolden(t *testing.T) {
	t.Run("help output matches golden file", func(t *testing.T) {
		// Run the binary with --help
		cmd := exec.Command("./gh-aw", "--help")
		cmd.Dir = "../.." // Run from repo root where binary is located

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to run gh-aw --help: %v\nOutput: %s", err, output)
		}

		actualOutput := string(output)

		// Read the golden file
		goldenPath := filepath.Join("testdata", "help_output.golden.txt")
		goldenBytes, err := os.ReadFile(goldenPath)
		if err != nil {
			t.Fatalf("Failed to read golden file %s: %v", goldenPath, err)
		}

		expectedOutput := string(goldenBytes)

		// Check if we should update the golden file
		if os.Getenv("UPDATE_GOLDEN") == "1" {
			err := os.WriteFile(goldenPath, output, 0644)
			if err != nil {
				t.Fatalf("Failed to update golden file: %v", err)
			}
			t.Logf("Updated golden file: %s", goldenPath)
			return
		}

		// Compare outputs
		if actualOutput != expectedOutput {
			t.Errorf("Help output does not match golden file.\n\nTo update the golden file, run:\n  UPDATE_GOLDEN=1 go test -v -tags integration -run TestHelpOutputGolden ./cmd/gh-aw\n\nDifferences:\n%s",
				generateDiff(expectedOutput, actualOutput))
		}
	})
}

// generateDiff creates a simple diff-like output showing mismatches
func generateDiff(expected, actual string) string {
	expectedLines := strings.Split(expected, "\n")
	actualLines := strings.Split(actual, "\n")

	var diff strings.Builder
	maxLines := len(expectedLines)
	if len(actualLines) > maxLines {
		maxLines = len(actualLines)
	}

	for i := 0; i < maxLines; i++ {
		expectedLine := ""
		actualLine := ""

		if i < len(expectedLines) {
			expectedLine = expectedLines[i]
		}
		if i < len(actualLines) {
			actualLine = actualLines[i]
		}

		if expectedLine != actualLine {
			if expectedLine != "" {
				diff.WriteString("- ")
				diff.WriteString(expectedLine)
				diff.WriteString("\n")
			}
			if actualLine != "" {
				diff.WriteString("+ ")
				diff.WriteString(actualLine)
				diff.WriteString("\n")
			}
		}
	}

	return diff.String()
}
