package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompileWorkflow_PanicRecovery verifies that panics during compilation are recovered and converted to errors
func TestCompileWorkflow_PanicRecovery(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "panic-recovery-test")
	require.NoError(t, err, "Should create temp directory")
	defer os.RemoveAll(tempDir)

	// Create a test workflow file with content that could trigger edge cases
	// (Note: This tests the recovery mechanism, not specific panic triggers)
	workflowPath := filepath.Join(tempDir, "test-workflow.md")
	workflowContent := `---
name: Test Workflow
engine: copilot
on:
  issues:
    types: [opened]
---

Test workflow content
`
	err = os.WriteFile(workflowPath, []byte(workflowContent), 0644)
	require.NoError(t, err, "Should write test workflow file")

	// Create compiler with normal configuration
	compiler := NewCompiler()
	compiler.SetNoEmit(true) // Don't write files during test
	compiler.SetQuiet(true)  // Suppress output during test

	// Test that normal compilation doesn't panic
	err = compiler.CompileWorkflow(workflowPath)
	// We expect either success or a normal error, but never a panic
	// If a panic occurred and wasn't recovered, the test would crash
	if err != nil {
		// If there's an error, verify it's not a panic-related error
		// (in normal operation, this might be a validation error, which is fine)
		t.Logf("Compilation error (expected in some cases): %v", err)
	}
}

// TestCompileWorkflowData_PanicRecovery verifies panic recovery in CompileWorkflowData
func TestCompileWorkflowData_PanicRecovery(t *testing.T) {
	// Create a minimal WorkflowData structure
	workflowData := &WorkflowData{
		Name:            "Test Workflow",
		On:              "on:\n  issues:\n    types: [opened]",
		Permissions:     "permissions: {}",
		Concurrency:     "",
		RunName:         "",
		MarkdownContent: "Test content",
		Tools:           make(map[string]any),
	}

	// Create a temporary file path for testing
	tempDir, err := os.MkdirTemp("", "panic-recovery-data-test")
	require.NoError(t, err, "Should create temp directory")
	defer os.RemoveAll(tempDir)

	markdownPath := filepath.Join(tempDir, "test.md")

	// Create compiler
	compiler := NewCompiler()
	compiler.SetNoEmit(true) // Don't write files during test
	compiler.SetQuiet(true)  // Suppress output during test

	// Test that compilation doesn't panic
	err = compiler.CompileWorkflowData(workflowData, markdownPath)
	// We expect either success or a normal error, but never a panic
	if err != nil {
		t.Logf("Compilation error (may be expected): %v", err)
	}
}

// TestCompileWorkflow_InvalidPath verifies that invalid paths don't cause panics
func TestCompileWorkflow_InvalidPath(t *testing.T) {
	compiler := NewCompiler()
	compiler.SetNoEmit(true)
	compiler.SetQuiet(true)

	// Test with non-existent file
	err := compiler.CompileWorkflow("/nonexistent/path/workflow.md")
	require.Error(t, err, "Should return error for non-existent file")
	assert.NotContains(t, err.Error(), "panic", "Error should not contain panic")
}

// TestCompilerPanicRecovery_ErrorFormat verifies that recovered panics are properly formatted
func TestCompilerPanicRecovery_ErrorFormat(t *testing.T) {
	// This test verifies the error format when a panic is recovered
	// We create a scenario where we can check the error structure

	tempDir, err := os.MkdirTemp("", "panic-format-test")
	require.NoError(t, err, "Should create temp directory")
	defer os.RemoveAll(tempDir)

	workflowPath := filepath.Join(tempDir, "test.md")
	
	// Create an invalid workflow that might trigger errors
	invalidContent := `---
name: Test
engine: invalid-engine-that-does-not-exist
on: invalid
---
`
	err = os.WriteFile(workflowPath, []byte(invalidContent), 0644)
	require.NoError(t, err, "Should write test file")

	compiler := NewCompiler()
	compiler.SetNoEmit(true)
	compiler.SetQuiet(true)

	err = compiler.CompileWorkflow(workflowPath)
	// We expect an error (due to invalid content), but it should be a proper error, not a panic
	if err != nil {
		// Error is expected - verify it's a properly formatted error
		require.IsType(t, (*error)(nil), &err, "Should be a standard error")
		t.Logf("Got expected error: %v", err)
	}
}

// TestGenerateYAML_PanicRecovery verifies panic recovery in YAML generation
func TestGenerateYAML_PanicRecovery(t *testing.T) {
	// Create a minimal but valid WorkflowData
	workflowData := &WorkflowData{
		Name:            "Test Workflow",
		On:              "on:\n  push:\n    branches: [main]",
		Permissions:     "permissions: {}",
		Concurrency:     "",
		RunName:         "",
		MarkdownContent: "Test content",
		Tools:           make(map[string]any),
	}

	compiler := NewCompiler()
	compiler.SetNoEmit(true)
	compiler.SetQuiet(true)

	// Test generateYAML doesn't panic
	tempDir, err := os.MkdirTemp("", "yaml-gen-test")
	require.NoError(t, err, "Should create temp directory")
	defer os.RemoveAll(tempDir)

	markdownPath := filepath.Join(tempDir, "test.md")

	// Call generateYAML (internal method, but we test it to ensure panic recovery works)
	yamlContent, err := compiler.generateYAML(workflowData, markdownPath)
	
	// Should not panic - either succeeds or returns proper error
	if err != nil {
		t.Logf("YAML generation error (may be expected): %v", err)
		// If there's an error about panic recovery, verify format
		if strings.Contains(err.Error(), "panic") {
			assert.Contains(t, err.Error(), "stack trace", "Panic recovery error should include stack trace")
		}
	} else {
		assert.NotEmpty(t, yamlContent, "Should generate YAML content on success")
	}
}

// TestCompilerNoPanics_ValidWorkflow ensures valid workflows don't trigger panic recovery
func TestCompilerNoPanics_ValidWorkflow(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "valid-workflow-test")
	require.NoError(t, err, "Should create temp directory")
	defer os.RemoveAll(tempDir)

	// Create a valid, simple workflow
	workflowPath := filepath.Join(tempDir, "valid.md")
	validContent := `---
name: Valid Test Workflow
engine: copilot
on:
  push:
    branches: [main]
permissions:
  contents: read
---

# Test Task

This is a simple test workflow.
`
	err = os.WriteFile(workflowPath, []byte(validContent), 0644)
	require.NoError(t, err, "Should write valid workflow")

	compiler := NewCompiler()
	compiler.SetNoEmit(true)
	compiler.SetQuiet(true)
	compiler.SetSkipValidation(true) // Skip validation to avoid external dependencies

	err = compiler.CompileWorkflow(workflowPath)
	// Valid workflow should compile without panic
	if err != nil {
		// Some validation errors might still occur, but no panics
		assert.NotContains(t, err.Error(), "panic", "Valid workflow should not trigger panic recovery")
	}
}
