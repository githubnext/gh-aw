package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/workflow"
)

// TestCompileConfig tests the CompileConfig structure
func TestCompileConfig(t *testing.T) {
	config := CompileConfig{
		MarkdownFiles:  []string{"test.md"},
		Verbose:        true,
		EngineOverride: "copilot",
		Validate:       true,
		Watch:          false,
		WorkflowDir:    ".github/workflows",
		NoEmit:         false,
		Purge:          false,
		TrialMode:      false,
		Strict:         false,
		Dependabot:     false,
		ForceOverwrite: false,
		Zizmor:         false,
		Poutine:        false,
		Actionlint:     false,
	}

	// Verify all fields are accessible
	if len(config.MarkdownFiles) != 1 {
		t.Errorf("Expected 1 markdown file, got %d", len(config.MarkdownFiles))
	}
	if !config.Verbose {
		t.Error("Expected Verbose to be true")
	}
	if config.EngineOverride != "copilot" {
		t.Errorf("Expected EngineOverride to be 'copilot', got %q", config.EngineOverride)
	}
}

// TestCompilationStats tests the CompilationStats structure
func TestCompilationStats(t *testing.T) {
	stats := &CompilationStats{
		Total:           5,
		Errors:          2,
		Warnings:        3,
		FailedWorkflows: []string{"workflow1.md", "workflow2.md"},
	}

	if stats.Total != 5 {
		t.Errorf("Expected Total to be 5, got %d", stats.Total)
	}
	if stats.Errors != 2 {
		t.Errorf("Expected Errors to be 2, got %d", stats.Errors)
	}
	if stats.Warnings != 3 {
		t.Errorf("Expected Warnings to be 3, got %d", stats.Warnings)
	}
	if len(stats.FailedWorkflows) != 2 {
		t.Errorf("Expected 2 failed workflows, got %d", len(stats.FailedWorkflows))
	}
}

// TestPrintCompilationSummary tests the printCompilationSummary function
func TestPrintCompilationSummary(t *testing.T) {
	tests := []struct {
		name  string
		stats *CompilationStats
	}{
		{
			name: "no workflows",
			stats: &CompilationStats{
				Total:    0,
				Errors:   0,
				Warnings: 0,
			},
		},
		{
			name: "successful compilation",
			stats: &CompilationStats{
				Total:    5,
				Errors:   0,
				Warnings: 0,
			},
		},
		{
			name: "with warnings",
			stats: &CompilationStats{
				Total:    5,
				Errors:   0,
				Warnings: 3,
			},
		},
		{
			name: "with errors",
			stats: &CompilationStats{
				Total:           5,
				Errors:          2,
				Warnings:        1,
				FailedWorkflows: []string{"workflow1.md", "workflow2.md"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// printCompilationSummary writes to stderr, we just verify it doesn't panic
			printCompilationSummary(tt.stats)
		})
	}
}

// Note: TestHandleFileDeleted is already tested in commands_file_watching_test.go

// TestCompileWorkflowWithValidation_InvalidFile tests error handling
func TestCompileWorkflowWithValidation_InvalidFile(t *testing.T) {
	compiler := workflow.NewCompiler(false, "", "test")

	// Try to compile a non-existent file
	err := CompileWorkflowWithValidation(
		compiler,
		"/nonexistent/file.md",
		false, // verbose
		false, // zizmor
		false, // poutine
		false, // actionlint
		false, // strict
		false, // validateActionSHAs
	)

	if err == nil {
		t.Error("Expected error when compiling non-existent file, got nil")
	}
}

// TestCompileWorkflows_DependabotValidation tests dependabot flag validation
// Uses the fast validateCompileConfig function instead of full compilation
func TestCompileWorkflows_DependabotValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      CompileConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "dependabot with specific files",
			config: CompileConfig{
				Dependabot:    true,
				MarkdownFiles: []string{"test.md"},
			},
			expectError: true,
			errorMsg:    "cannot be used with specific workflow files",
		},
		{
			name: "dependabot with custom workflow dir",
			config: CompileConfig{
				Dependabot:  true,
				WorkflowDir: "custom/workflows",
			},
			expectError: true,
			errorMsg:    "cannot be used with custom --dir",
		},
		{
			name: "dependabot with default settings",
			config: CompileConfig{
				Dependabot:  true,
				WorkflowDir: "",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use fast validation function instead of full compilation
			err := validateCompileConfig(tt.config)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else if err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// TestCompileWorkflows_PurgeValidation tests purge flag validation
// Uses the fast validateCompileConfig function instead of full compilation
func TestCompileWorkflows_PurgeValidation(t *testing.T) {
	config := CompileConfig{
		Purge:         true,
		MarkdownFiles: []string{"test.md"},
	}

	// Use fast validation function instead of full compilation
	err := validateCompileConfig(config)

	if err == nil {
		t.Error("Expected error when using purge with specific files, got nil")
	}

	if !strings.Contains(err.Error(), "can only be used when compiling all markdown files") {
		t.Errorf("Expected error about purge flag, got: %v", err)
	}
}

// TestCompileWorkflows_WorkflowDirValidation tests workflow directory validation
// Uses the fast validateCompileConfig function instead of full compilation
func TestCompileWorkflows_WorkflowDirValidation(t *testing.T) {
	tests := []struct {
		name        string
		workflowDir string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "absolute path not allowed",
			workflowDir: "/absolute/path",
			expectError: true,
			errorMsg:    "must be a relative path",
		},
		{
			name:        "relative path allowed",
			workflowDir: "custom/workflows",
			expectError: false,
		},
		{
			name:        "default empty path",
			workflowDir: "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := CompileConfig{
				WorkflowDir: tt.workflowDir,
			}

			// Use fast validation function instead of full compilation
			err := validateCompileConfig(config)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else if err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// TestCompileWorkflowDataWithValidation_NoEmit tests validation without emission
func TestCompileWorkflowDataWithValidation_NoEmit(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")

	// Create a simple test workflow
	workflowContent := `---
on: push
permissions:
  contents: read
---

# Test Workflow

This is a test workflow.
`
	if err := os.WriteFile(testFile, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create compiler with noEmit flag
	compiler := workflow.NewCompiler(false, "", "test")
	compiler.SetNoEmit(true)

	// Parse the workflow
	workflowData, err := compiler.ParseWorkflowFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Compile without emitting
	err = CompileWorkflowDataWithValidation(
		compiler,
		workflowData,
		testFile,
		false, // verbose
		false, // zizmor
		false, // poutine
		false, // actionlint
		false, // strict
		false, // validateActionSHAs
	)

	// Should complete without error
	if err != nil {
		t.Errorf("Unexpected error with noEmit: %v", err)
	}

	// Verify lock file was not created
	lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
	if _, err := os.Stat(lockFile); !os.IsNotExist(err) {
		t.Error("Lock file should not exist with noEmit flag")
	}
}

// TestCompileWorkflowWithValidation_YAMLValidation tests YAML validation
func TestCompileWorkflowWithValidation_YAMLValidation(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")

	// Create a simple test workflow
	workflowContent := `---
on: push
permissions:
  contents: read
engine: copilot
---

# Test Workflow

This is a test workflow.
`
	if err := os.WriteFile(testFile, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create compiler
	compiler := workflow.NewCompiler(false, "", "test")

	// Compile the workflow
	err := CompileWorkflowWithValidation(
		compiler,
		testFile,
		false, // verbose
		false, // zizmor
		false, // poutine
		false, // actionlint
		false, // strict
		false, // validateActionSHAs
	)

	// Should complete without error
	if err != nil {
		t.Errorf("Unexpected error during compilation: %v", err)
	}

	// Verify lock file was created
	lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
	if _, err := os.Stat(lockFile); os.IsNotExist(err) {
		t.Error("Lock file should have been created")
	}

	// Clean up
	os.Remove(lockFile)
}

// TestCompileConfig_DefaultValues tests default configuration values
func TestCompileConfig_DefaultValues(t *testing.T) {
	config := CompileConfig{}

	// Verify default values
	if config.Verbose {
		t.Error("Expected Verbose to default to false")
	}
	if config.Validate {
		t.Error("Expected Validate to default to false")
	}
	if config.Watch {
		t.Error("Expected Watch to default to false")
	}
	if config.NoEmit {
		t.Error("Expected NoEmit to default to false")
	}
	if config.Purge {
		t.Error("Expected Purge to default to false")
	}
	if config.TrialMode {
		t.Error("Expected TrialMode to default to false")
	}
	if config.Strict {
		t.Error("Expected Strict to default to false")
	}
	if config.Dependabot {
		t.Error("Expected Dependabot to default to false")
	}
	if config.ForceOverwrite {
		t.Error("Expected ForceOverwrite to default to false")
	}
	if config.Zizmor {
		t.Error("Expected Zizmor to default to false")
	}
	if config.Poutine {
		t.Error("Expected Poutine to default to false")
	}
	if config.Actionlint {
		t.Error("Expected Actionlint to default to false")
	}
}

// TestCompilationStats_DefaultValues tests default stats values
func TestCompilationStats_DefaultValues(t *testing.T) {
	stats := &CompilationStats{}

	if stats.Total != 0 {
		t.Errorf("Expected Total to default to 0, got %d", stats.Total)
	}
	if stats.Errors != 0 {
		t.Errorf("Expected Errors to default to 0, got %d", stats.Errors)
	}
	if stats.Warnings != 0 {
		t.Errorf("Expected Warnings to default to 0, got %d", stats.Warnings)
	}
	if len(stats.FailedWorkflows) != 0 {
		t.Errorf("Expected FailedWorkflows to default to empty, got %d items", len(stats.FailedWorkflows))
	}
}

// TestCompileWorkflows_EmptyMarkdownFiles tests compilation with no files specified
func TestCompileWorkflows_EmptyMarkdownFiles(t *testing.T) {
	// This test requires being in a git repository
	// We'll skip if not in a git repo
	_, err := findGitRoot()
	if err != nil {
		t.Skip("Not in a git repository, skipping test")
	}

	config := CompileConfig{
		MarkdownFiles: []string{},
		WorkflowDir:   ".github/workflows",
	}

	// This will try to compile all files in .github/workflows
	// It may fail if the directory doesn't exist, which is expected
	CompileWorkflows(config)

	// We don't check for specific error here as it depends on the repository state
	// The test just ensures the function handles empty MarkdownFiles correctly
}

// TestCompileWorkflows_TrialMode tests trial mode configuration
func TestCompileWorkflows_TrialMode(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")

	// Create a simple test workflow
	workflowContent := `---
on: push
permissions:
  contents: read
engine: copilot
---

# Test Workflow

This is a test workflow.
`
	if err := os.WriteFile(testFile, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	config := CompileConfig{
		MarkdownFiles:        []string{testFile},
		TrialMode:            true,
		TrialLogicalRepoSlug: "owner/trial-repo",
	}

	_, err := CompileWorkflows(config)

	// The compilation may fail for various reasons in a test environment,
	// but it should not panic and should handle trial mode settings
	_ = err // We're just testing that the config is processed
}

// TestValidationResult tests the ValidationResult structure
func TestValidationResult(t *testing.T) {
	result := ValidationResult{
		Workflow: "test-workflow.md",
		Valid:    false,
		Errors: []ValidationError{
			{
				Type:    "schema_validation",
				Message: "Unknown property: toolz",
				Line:    5,
			},
		},
		Warnings:     []ValidationError{},
		CompiledFile: ".github/workflows/test-workflow.lock.yml",
	}

	if result.Workflow != "test-workflow.md" {
		t.Errorf("Expected workflow 'test-workflow.md', got %q", result.Workflow)
	}
	if result.Valid {
		t.Error("Expected Valid to be false")
	}
	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(result.Errors))
	}
	if result.Errors[0].Type != "schema_validation" {
		t.Errorf("Expected error type 'schema_validation', got %q", result.Errors[0].Type)
	}
}

// TestCompileConfig_JSONOutput tests the JSONOutput field
func TestCompileConfig_JSONOutput(t *testing.T) {
	config := CompileConfig{
		MarkdownFiles: []string{"test.md"},
		JSONOutput:    true,
	}

	if !config.JSONOutput {
		t.Error("Expected JSONOutput to be true")
	}
}
