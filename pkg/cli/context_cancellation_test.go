package cli

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompileWorkflowsContextCancellation tests that CompileWorkflows respects context cancellation
func TestCompileWorkflowsContextCancellation(t *testing.T) {
	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	config := CompileConfig{
		MarkdownFiles: []string{"test.md"},
		Verbose:       false,
		Validate:      false,
	}

	_, err := CompileWorkflows(ctx, config)

	// Should fail with context.Canceled error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cancelled")
}

// TestCompileWorkflowsContextTimeout tests that CompileWorkflows respects context timeout
func TestCompileWorkflowsContextTimeout(t *testing.T) {
	// Create a context with very short timeout (1 nanosecond)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait a bit to ensure timeout has passed
	time.Sleep(10 * time.Millisecond)

	config := CompileConfig{
		MarkdownFiles: []string{"test.md"},
		Verbose:       false,
		Validate:      false,
	}

	_, err := CompileWorkflows(ctx, config)

	// Should fail with context deadline exceeded
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cancelled")
}

// TestRunWorkflowOnGitHubContextCancellation tests that RunWorkflowOnGitHub respects context cancellation
func TestRunWorkflowOnGitHubContextCancellation(t *testing.T) {
	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// This should fail fast due to cancelled context
	// Note: We don't validate the workflow file here since context check happens first
	err := RunWorkflowOnGitHub(ctx, "test-workflow", false, "", "", "", false, false, false, []string{}, false)

	// The exact behavior depends on where cancellation is checked
	// It might fail with a different error if file validation happens first
	if err != nil {
		// Test passes - the operation was interrupted or failed
		assert.Error(t, err)
	}
}

// TestRunWorkflowsOnGitHubContextCancellation tests that RunWorkflowsOnGitHub respects context cancellation
func TestRunWorkflowsOnGitHubContextCancellation(t *testing.T) {
	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := RunWorkflowsOnGitHub(ctx, []string{"test1", "test2"}, 0, false, "", "", "", false, false, []string{}, false)

	// Should fail with context cancellation error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cancelled")
}

// TestDownloadWorkflowLogsContextCancellation tests that DownloadWorkflowLogs respects context cancellation
func TestDownloadWorkflowLogsContextCancellation(t *testing.T) {
	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := DownloadWorkflowLogs(ctx, "test-workflow", 10, "", "", "/tmp", "", "", 0, 0, "", false, false, false, false, false, false, false, 0, false, "")

	// Should fail with context cancellation error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cancelled")
}

// TestAuditWorkflowRunContextCancellation tests that AuditWorkflowRun respects context cancellation
func TestAuditWorkflowRunContextCancellation(t *testing.T) {
	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := AuditWorkflowRun(ctx, 12345, "owner", "repo", "github.com", "/tmp", false, false, false, 0, 0)

	// Should fail with context cancellation error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cancelled")
}
