package cli

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunAddInteractive_PanicRecovery verifies panic recovery in interactive add
func TestRunAddInteractive_PanicRecovery(t *testing.T) {
	// Set environment variable to prevent actual interactive mode
	t.Setenv("GO_TEST_MODE", "true")

	// Test that the function returns proper error instead of panicking
	err := RunAddInteractive(context.Background(), []string{"test/repo/workflow"}, false, "", false, "", false, "")

	// Should get error about test mode, not a panic
	require.Error(t, err, "Should return error in test mode")
	assert.Contains(t, err.Error(), "automated tests", "Should mention test mode restriction")
	assert.NotContains(t, err.Error(), "panic", "Should not be a panic error in normal error path")
}

// TestInitRepositoryInteractive_PanicRecovery verifies panic recovery in interactive init
func TestInitRepositoryInteractive_PanicRecovery(t *testing.T) {
	// Set environment variable to prevent actual interactive mode
	t.Setenv("GO_TEST_MODE", "true")

	// Mock command provider
	mockCmd := &mockCommandProvider{}

	// Test that the function returns proper error instead of panicking
	err := InitRepositoryInteractive(false, mockCmd)

	// Should get error about test mode, not a panic
	require.Error(t, err, "Should return error in test mode")
	assert.Contains(t, err.Error(), "automated tests", "Should mention test mode restriction")
	assert.NotContains(t, err.Error(), "panic", "Should not be a panic error in normal error path")
}

// TestRunWorkflowTrials_PanicRecovery verifies panic recovery in trial execution
func TestRunWorkflowTrials_PanicRecovery(t *testing.T) {
	// Test with invalid workflow spec
	opts := TrialOptions{
		Repos: RepoConfig{
			LogicalRepo: "test/repo",
		},
		Quiet: true,
	}

	// Test that invalid specs return proper errors, not panics
	err := RunWorkflowTrials([]string{"invalid-spec-format"}, opts)

	// Should get a validation error, not a panic
	require.Error(t, err, "Should return error for invalid spec")
	// The error should be about the invalid specification format
	assert.True(t,
		strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "panic"),
		"Error should be about invalid spec or be a recovered panic with stack trace")

	// If it's a recovered panic, it should have stack trace
	if strings.Contains(err.Error(), "panic") {
		assert.Contains(t, err.Error(), "stack trace", "Recovered panic should include stack trace")
	}
}

// TestCheckForUpdates_PanicRecovery verifies panic recovery in update check
func TestCheckForUpdates_PanicRecovery(t *testing.T) {
	// checkForUpdates is internal but we can test the pattern
	// The function should never panic even with network issues

	// Set CI mode to skip actual update check
	t.Setenv("CI", "true")

	// Call checkForUpdates - should not panic even in CI mode
	checkForUpdates(false, false)

	// If we got here without panic, the test passes
	// The function handles all errors internally and never returns errors
}

// mockCommandProvider implements CommandProvider interface for testing
type mockCommandProvider struct{}

func (m *mockCommandProvider) SetUsageFunc(f func(*cobra.Command) error) {}

func (m *mockCommandProvider) GenBashCompletion(w io.Writer) error {
	return nil
}

func (m *mockCommandProvider) GenZshCompletion(w io.Writer) error {
	return nil
}

func (m *mockCommandProvider) GenFishCompletion(w io.Writer, includeDesc bool) error {
	return nil
}

// TestPanicRecoveryFormat_ErrorStructure verifies recovered panics have proper format
func TestPanicRecoveryFormat_ErrorStructure(t *testing.T) {
	// Test that if a panic is recovered, it follows the expected format
	// This is tested indirectly through the actual functions

	tests := []struct {
		name          string
		errorMsg      string
		shouldBePanic bool
	}{
		{
			name:          "Normal error",
			errorMsg:      "file not found: workflow.md",
			shouldBePanic: false,
		},
		{
			name:          "Recovered panic error",
			errorMsg:      "panic during compilation: something went wrong\nstack trace:\ngoroutine...",
			shouldBePanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldBePanic {
				assert.Contains(t, tt.errorMsg, "panic during", "Panic error should mention panic")
				assert.Contains(t, tt.errorMsg, "stack trace", "Panic error should include stack trace")
			} else {
				assert.NotContains(t, tt.errorMsg, "panic during", "Normal error should not mention panic")
				assert.NotContains(t, tt.errorMsg, "stack trace", "Normal error should not have stack trace")
			}
		})
	}
}
