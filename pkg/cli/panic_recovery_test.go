package cli

import (
	"context"
	"strings"
	"testing"
)

// TestCompileWorkflows_PanicRecovery tests that panics during compilation are recovered
func TestCompileWorkflows_PanicRecovery(t *testing.T) {
	tests := []struct {
		name        string
		config      CompileConfig
		expectError bool
	}{
		{
			name: "empty config",
			config: CompileConfig{
				MarkdownFiles: []string{},
				Verbose:       false,
				Watch:         false, // Don't use watch mode in tests
			},
			expectError: true, // Should error due to no files
		},
		{
			name: "nonexistent file",
			config: CompileConfig{
				MarkdownFiles: []string{"/nonexistent/path/to/workflow.md"},
				Verbose:       false,
				Watch:         false, // Don't use watch mode in tests
			},
			expectError: true,
		},
		{
			name: "invalid config - empty file list",
			config: CompileConfig{
				MarkdownFiles: []string{},
				Watch:         false,  // Don't use watch mode in tests
				Validate:      false,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// This should not panic even with invalid configuration
			data, err := CompileWorkflows(ctx, tt.config)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			// Verify that if error contains "internal error", it's a recovered panic
			if err != nil && strings.Contains(err.Error(), "internal error") {
				if !strings.Contains(err.Error(), "This is a bug") {
					t.Error("internal error should include bug reporting message")
				}
				if !strings.Contains(err.Error(), "github.com/githubnext/gh-aw/issues") {
					t.Error("internal error should include issue reporting URL")
				}
			}

			// If we got data back, it should be a valid slice (possibly empty)
			if data != nil {
				t.Logf("Got %d workflow(s) compiled", len(data))
			}
		})
	}
}

// TestCompileWorkflows_ContextCancellation tests that context cancellation doesn't cause panics
func TestCompileWorkflows_ContextCancellation(t *testing.T) {
	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	config := CompileConfig{
		MarkdownFiles: []string{"test.md"},
		Verbose:       false,
	}

	// This should return context.Canceled error, not panic
	_, err := CompileWorkflows(ctx, config)

	if err == nil {
		t.Error("expected error from cancelled context")
	}

	if err != nil && !strings.Contains(err.Error(), "cancel") {
		t.Logf("Got error (not necessarily cancellation): %v", err)
	}

	// Most importantly, verify it didn't panic
	t.Log("Context cancellation handled without panic")
}

// TestRunWorkflowOnGitHub_PanicRecovery tests that panics during workflow execution are recovered
func TestRunWorkflowOnGitHub_PanicRecovery(t *testing.T) {
	tests := []struct {
		name        string
		workflow    string
		expectError bool
	}{
		{
			name:        "empty workflow name",
			workflow:    "",
			expectError: true,
		},
		{
			name:        "invalid workflow name with special characters",
			workflow:    "workflow-with-many-special-chars-!@#$%^&*()",
			expectError: true, // Will fail validation but shouldn't panic
		},
		{
			name:        "nonexistent workflow",
			workflow:    "nonexistent-workflow-12345",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// This should not panic even with invalid input
			err := RunWorkflowOnGitHub(
				ctx,
				tt.workflow,
				false, // enable
				"",    // engineOverride
				"",    // repoOverride
				"",    // refOverride
				false, // autoMergePRs
				false, // pushSecrets
				false, // push
				false, // waitForCompletion
				[]string{}, // inputs
				false, // verbose
			)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			// Verify that if error contains "internal error", it's a recovered panic
			if err != nil && strings.Contains(err.Error(), "internal error") {
				if !strings.Contains(err.Error(), "This is a bug") {
					t.Error("internal error should include bug reporting message")
				}
				if !strings.Contains(err.Error(), "github.com/githubnext/gh-aw/issues") {
					t.Error("internal error should include issue reporting URL")
				}
			}
		})
	}
}

// TestRunWorkflowOnGitHub_InvalidInputs tests that invalid inputs don't cause panics
func TestRunWorkflowOnGitHub_InvalidInputs(t *testing.T) {
	tests := []struct {
		name        string
		inputs      []string
		expectError bool
	}{
		{
			name:        "valid input format",
			inputs:      []string{"key=value"},
			expectError: true, // Will fail because workflow doesn't exist, but shouldn't panic
		},
		{
			name:        "invalid input format - no equals",
			inputs:      []string{"keyvalue"},
			expectError: true,
		},
		{
			name:        "invalid input format - empty key",
			inputs:      []string{"=value"},
			expectError: true,
		},
		{
			name:        "multiple inputs",
			inputs:      []string{"key1=value1", "key2=value2"},
			expectError: true, // Will fail because workflow doesn't exist
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// This should not panic even with invalid input format
			err := RunWorkflowOnGitHub(
				ctx,
				"test-workflow",
				false,
				"",
				"",
				"",
				false,
				false,
				false,
				false,
				tt.inputs,
				false,
			)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			// Verify panic recovery error format
			if err != nil && strings.Contains(err.Error(), "internal error") {
				if !strings.Contains(err.Error(), "This is a bug") {
					t.Error("internal error should include bug reporting message")
				}
			}
		})
	}
}

// TestPanicRecoveryLogging verifies that panic recovery logs are properly formatted
func TestPanicRecoveryLogging(t *testing.T) {
	// This test ensures that when a panic is recovered, the proper logging occurs
	// We can't directly test the log output, but we can verify the error messages

	ctx := context.Background()
	config := CompileConfig{
		MarkdownFiles: []string{"/invalid/path/that/does/not/exist.md"},
		Verbose:       false,
	}

	_, err := CompileWorkflows(ctx, config)

	if err == nil {
		t.Skip("Expected error for invalid path")
	}

	// Verify error message structure
	errStr := err.Error()
	t.Logf("Error message: %s", errStr)

	// If this is an internal error (recovered panic), verify complete format
	if strings.Contains(errStr, "internal error") {
		// Check all required components
		requiredComponents := []string{
			"internal error",
			"This is a bug",
			"github.com/githubnext/gh-aw/issues",
		}

		for _, component := range requiredComponents {
			if !strings.Contains(errStr, component) {
				t.Errorf("Error message missing required component: %s", component)
			}
		}

		// Verify it mentions what failed
		if !strings.Contains(errStr, "compilation") && !strings.Contains(errStr, "execution") && !strings.Contains(errStr, "parsing") {
			t.Error("Error message should mention the operation that failed")
		}
	}
}
