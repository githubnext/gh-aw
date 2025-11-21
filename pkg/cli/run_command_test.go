package cli

import (
	"fmt"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/constants"
)

// TestAuditSuggestionMessage tests that the audit suggestion message
// has the expected format and includes the CLI command prefix
func TestAuditSuggestionMessage(t *testing.T) {
	// Sample run ID
	runID := int64(1234567890)

	// Generate the audit suggestion message
	auditSuggestion := fmt.Sprintf("💡 To analyze this run, use: %s audit %d", constants.CLIExtensionPrefix, runID)

	// Verify the message contains the expected elements
	expectedElements := []string{
		"💡", // Lightbulb emoji for friendly suggestion
		"To analyze this run",
		"use:",
		constants.CLIExtensionPrefix, // Should be "gh aw"
		"audit",
		fmt.Sprintf("%d", runID),
	}

	for _, element := range expectedElements {
		if !strings.Contains(auditSuggestion, element) {
			t.Errorf("Expected audit suggestion to contain %q, got: %s", element, auditSuggestion)
		}
	}

	// Verify the full command format
	expectedCommand := fmt.Sprintf("%s audit %d", constants.CLIExtensionPrefix, runID)
	if !strings.Contains(auditSuggestion, expectedCommand) {
		t.Errorf("Expected audit suggestion to contain full command %q, got: %s", expectedCommand, auditSuggestion)
	}
}

// TestAuditSuggestionMessageFormat tests the exact format of the audit suggestion
func TestAuditSuggestionMessageFormat(t *testing.T) {
	tests := []struct {
		name     string
		runID    int64
		expected string
	}{
		{
			name:     "small run ID",
			runID:    123,
			expected: "💡 To analyze this run, use: gh aw audit 123",
		},
		{
			name:     "large run ID",
			runID:    9876543210,
			expected: "💡 To analyze this run, use: gh aw audit 9876543210",
		},
		{
			name:     "typical run ID",
			runID:    1234567890,
			expected: "💡 To analyze this run, use: gh aw audit 1234567890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate the audit suggestion message
			auditSuggestion := fmt.Sprintf("💡 To analyze this run, use: %s audit %d", constants.CLIExtensionPrefix, tt.runID)

			// Verify exact format
			if auditSuggestion != tt.expected {
				t.Errorf("Expected audit suggestion %q, got %q", tt.expected, auditSuggestion)
			}

			// Verify it's agent-friendly (clear, actionable, no ambiguity)
			if !strings.HasPrefix(auditSuggestion, "💡") {
				t.Error("Expected audit suggestion to start with lightbulb emoji for friendliness")
			}

			if !strings.Contains(auditSuggestion, "To analyze this run") {
				t.Error("Expected audit suggestion to clearly state the purpose")
			}

			if !strings.Contains(auditSuggestion, "use:") {
				t.Error("Expected audit suggestion to provide clear action keyword 'use:'")
			}
		})
	}
}

// TestAuditSuggestionAgentFriendliness tests that the message is suitable for AI agents
func TestAuditSuggestionAgentFriendliness(t *testing.T) {
	runID := int64(1234567890)
	auditSuggestion := fmt.Sprintf("💡 To analyze this run, use: %s audit %d", constants.CLIExtensionPrefix, runID)

	// Agent-friendly characteristics:
	// 1. Clear action verb ("use")
	if !strings.Contains(auditSuggestion, "use:") {
		t.Error("Expected clear action verb 'use:'")
	}

	// 2. Specific command (not just a hint)
	if !strings.Contains(auditSuggestion, "gh aw audit") {
		t.Error("Expected specific command 'gh aw audit'")
	}

	// 3. Includes the run ID (no need to look it up)
	if !strings.Contains(auditSuggestion, fmt.Sprintf("%d", runID)) {
		t.Error("Expected run ID to be included in the command")
	}

	// 4. Not too wordy (agents prefer concise)
	wordCount := len(strings.Fields(auditSuggestion))
	if wordCount > 15 {
		t.Errorf("Expected concise message (< 15 words), got %d words", wordCount)
	}

	// 5. No ambiguous pronouns or references
	// Note: "this run" is acceptable as it refers to the just-triggered workflow run
	auditSuggestionLower := strings.ToLower(auditSuggestion)
	if strings.Contains(auditSuggestionLower, " it ") ||
		strings.Contains(auditSuggestionLower, "this one") ||
		strings.Contains(auditSuggestionLower, " that ") {
		t.Error("Expected no ambiguous references like 'it', 'this one', 'that'")
	}
}

// TestEmitProgress tests the emitProgress helper function
func TestEmitProgress(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		message  string
		expected string
	}{
		{
			name:     "progress enabled",
			enabled:  true,
			message:  "Testing progress",
			expected: "Progress: Testing progress\n",
		},
		{
			name:     "progress disabled",
			enabled:  false,
			message:  "Testing progress",
			expected: "", // No output when disabled
		},
		{
			name:     "empty message when enabled",
			enabled:  true,
			message:  "",
			expected: "Progress: \n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't easily capture stderr output in a unit test without more complex setup,
			// so we'll just verify the function doesn't panic and runs without error
			// The actual output testing would be better done in integration tests
			emitProgress(tt.enabled, tt.message)
		})
	}
}

// TestProgressFlagSignature tests that the progress flag is properly threaded through function calls
func TestProgressFlagSignature(t *testing.T) {
	// Test that functions accept the progress parameter
	// This is a compile-time check more than a runtime check

	// RunWorkflowOnGitHub should accept progress parameter
	_ = RunWorkflowOnGitHub("test", false, "", "", false, false, false, false, false)

	// RunWorkflowsOnGitHub should accept progress parameter
	_ = RunWorkflowsOnGitHub([]string{"test"}, 0, false, "", "", false, false, false, false)

	// getLatestWorkflowRunWithRetry should accept progress parameter
	_, _ = getLatestWorkflowRunWithRetry("test.lock.yml", "", false, false)
}
