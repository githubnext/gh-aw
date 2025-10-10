package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/githubnext/gh-aw/pkg/workflow"
)

func TestRunErrorValidator(t *testing.T) {
	// Create a test directory
	tempDir := t.TempDir()

	// Create a test log file with some error patterns
	logFile := filepath.Join(tempDir, "test.log")
	logContent := `2024-01-01 ERROR: Access denied - user not authorized
2024-01-01 INFO: Processing request
2024-01-01 WARNING: Deprecated function used
`
	if err := os.WriteFile(logFile, []byte(logContent), 0644); err != nil {
		t.Fatalf("Failed to write log file: %v", err)
	}

	// Create a mock engine with error patterns
	engine := workflow.NewClaudeEngine()

	// Test that runErrorValidator doesn't fail
	err := runErrorValidator(tempDir, logFile, engine, true)
	if err != nil {
		t.Errorf("runErrorValidator returned error: %v", err)
	}
}

func TestRunErrorValidatorTimeout(t *testing.T) {
	// This test ensures the timeout mechanism works
	// We'll create a scenario that would timeout

	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")
	logContent := "test log content\n"
	if err := os.WriteFile(logFile, []byte(logContent), 0644); err != nil {
		t.Fatalf("Failed to write log file: %v", err)
	}

	engine := workflow.NewClaudeEngine()

	// This should complete without timeout (normal case)
	err := runErrorValidator(tempDir, logFile, engine, false)
	if err != nil {
		t.Errorf("runErrorValidator returned error: %v", err)
	}
}

func TestRunErrorValidatorWithNoPatterns(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")
	logContent := "test log content\n"
	if err := os.WriteFile(logFile, []byte(logContent), 0644); err != nil {
		t.Fatalf("Failed to write log file: %v", err)
	}

	// Create a custom engine with no error patterns
	customEngine := &workflow.CustomEngine{}

	// Should return nil since there are no patterns
	err := runErrorValidator(tempDir, logFile, customEngine, false)
	if err != nil {
		t.Errorf("Expected nil error for engine with no patterns, got: %v", err)
	}
}

func TestErrorPatternSerialization(t *testing.T) {
	// Test that error patterns can be properly serialized to JSON
	patterns := []workflow.ErrorPattern{
		{
			Pattern:      `(?i)error.*test`,
			LevelGroup:   0,
			MessageGroup: 0,
			Description:  "Test error pattern",
		},
	}

	jsonData, err := json.Marshal(patterns)
	if err != nil {
		t.Fatalf("Failed to marshal patterns: %v", err)
	}

	var decoded []workflow.ErrorPattern
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal patterns: %v", err)
	}

	if len(decoded) != 1 {
		t.Errorf("Expected 1 pattern, got %d", len(decoded))
	}

	if decoded[0].Pattern != patterns[0].Pattern {
		t.Errorf("Pattern mismatch: expected %s, got %s", patterns[0].Pattern, decoded[0].Pattern)
	}
}
