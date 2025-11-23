package workflow

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestBundledScriptsHaveValidJavaScriptSyntax validates that all bundled scripts
// produce syntactically valid JavaScript that can be parsed by Node.js
func TestBundledScriptsHaveValidJavaScriptSyntax(t *testing.T) {
	// Check if node is available
	_, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node.js not found, skipping JavaScript syntax validation")
	}

	// List of scripts that use bundling
	scriptsToTest := []struct {
		name   string
		getter func() string
	}{
		{"collect_ndjson_output", getCollectJSONLOutputScript},
		{"compute_text", getComputeTextScript},
		{"sanitize_output", getSanitizeOutputScript},
		{"create_issue", getCreateIssueScript},
		{"add_labels", getAddLabelsScript},
		{"create_discussion", getCreateDiscussionScript},
		{"update_issue", getUpdateIssueScript},
		{"create_code_scanning_alert", getCreateCodeScanningAlertScript},
		{"create_pr_review_comment", getCreatePRReviewCommentScript},
		{"add_comment", getAddCommentScript},
		{"upload_assets", getUploadAssetsScript},
		{"parse_firewall_logs", getParseFirewallLogsScript},
		{"push_to_pull_request_branch", getPushToPullRequestBranchScript},
		{"create_pull_request", getCreatePullRequestScript},
		{"interpolate_prompt", getInterpolatePromptScript},
		{"parse_claude_log", getParseClaudeLogScript},
		{"parse_codex_log", getParseCodexLogScript},
		{"parse_copilot_log", getParseCopilotLogScript},
	}

	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "bundler-validation-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	for _, script := range scriptsToTest {
		t.Run(script.name, func(t *testing.T) {
			// Get the bundled script
			bundled := script.getter()
			if bundled == "" {
				t.Fatal("Bundled script is empty")
			}

			// Write to a temporary file
			tmpFile := filepath.Join(tmpDir, script.name+".js")
			err := os.WriteFile(tmpFile, []byte(bundled), 0644)
			if err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			// Validate JavaScript syntax using node --check
			cmd := exec.Command("node", "--check", tmpFile)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Errorf("JavaScript syntax validation failed for %s:\n%s\nError: %v", script.name, string(output), err)
			}
		})
	}
}

// TestBundleCreatePullRequestScript specifically tests the create_pull_request script
// that was failing in the issue
func TestBundleCreatePullRequestScript(t *testing.T) {
	// Check if node is available
	_, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node.js not found, skipping JavaScript syntax validation")
	}

	// Get the bundled create_pull_request script
	bundled := getCreatePullRequestScript()
	if bundled == "" {
		t.Fatal("Bundled create_pull_request script is empty")
	}

	// Create a temporary file
	tmpFile := filepath.Join(os.TempDir(), "create_pull_request_test.js")
	err = os.WriteFile(tmpFile, []byte(bundled), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	defer os.Remove(tmpFile)

	// Validate JavaScript syntax using node --check
	cmd := exec.Command("node", "--check", tmpFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("JavaScript syntax validation failed for create_pull_request:\n%s\nError: %v\n\nBundled content (first 2000 chars):\n%s",
			string(output), err, truncateString(bundled, 2000))
	}

	t.Logf("Successfully validated create_pull_request script (%d bytes)", len(bundled))
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "... (truncated)"
}

// TestValidateLineLength tests the line length validation function
func TestValidateLineLength(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		shouldError bool
	}{
		{
			name:        "empty content",
			content:     "",
			shouldError: false,
		},
		{
			name:        "single short line",
			content:     "const x = 1;",
			shouldError: false,
		},
		{
			name:        "multiple short lines",
			content:     "const x = 1;\nconst y = 2;\nconst z = 3;",
			shouldError: false,
		},
		{
			name:        "line at exactly 20k characters",
			content:     generateString(MaxLineLengthForActions),
			shouldError: false,
		},
		{
			name:        "line exceeding 20k characters by 1",
			content:     generateString(MaxLineLengthForActions + 1),
			shouldError: true,
		},
		{
			name:        "line far exceeding 20k characters",
			content:     generateString(25000),
			shouldError: true,
		},
		{
			name:        "multiple lines with one exceeding limit",
			content:     "const x = 1;\n" + generateString(MaxLineLengthForActions+1) + "\nconst y = 2;",
			shouldError: true,
		},
		{
			name:        "multiple lines all under limit",
			content:     "const x = 1;\n" + generateString(MaxLineLengthForActions-100) + "\nconst y = 2;",
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateLineLength(tt.content)
			if tt.shouldError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

// generateString generates a string of the specified length
func generateString(length int) string {
	if length <= 0 {
		return ""
	}
	// Use a pattern that mimics JavaScript code
	pattern := "const variable = 'value'; "
	result := ""
	for len(result) < length {
		result += pattern
	}
	return result[:length]
}

// TestBundledScriptsLineLengthValidation validates that all bundled scripts
// have lines within the GitHub Actions character limit
func TestBundledScriptsLineLengthValidation(t *testing.T) {
	scriptsToTest := []struct {
		name   string
		getter func() string
	}{
		{"safe_outputs_mcp_server", GetSafeOutputsMCPServerScript},
		{"check_membership", getCheckMembershipScript},
		{"update_project", getUpdateProjectScript},
		{"collect_ndjson_output", getCollectJSONLOutputScript},
		{"compute_text", getComputeTextScript},
		{"sanitize_output", getSanitizeOutputScript},
		{"create_issue", getCreateIssueScript},
		{"add_labels", getAddLabelsScript},
		{"create_discussion", getCreateDiscussionScript},
		{"update_issue", getUpdateIssueScript},
		{"create_code_scanning_alert", getCreateCodeScanningAlertScript},
		{"create_pr_review_comment", getCreatePRReviewCommentScript},
		{"add_comment", getAddCommentScript},
		{"upload_assets", getUploadAssetsScript},
		{"parse_firewall_logs", getParseFirewallLogsScript},
		{"push_to_pull_request_branch", getPushToPullRequestBranchScript},
		{"create_pull_request", getCreatePullRequestScript},
		{"interpolate_prompt", getInterpolatePromptScript},
		{"parse_claude_log", getParseClaudeLogScript},
		{"parse_codex_log", getParseCodexLogScript},
		{"parse_copilot_log", getParseCopilotLogScript},
	}

	for _, script := range scriptsToTest {
		t.Run(script.name, func(t *testing.T) {
			bundled := script.getter()
			if bundled == "" {
				t.Fatal("Bundled script is empty")
			}

			// Validate line length
			err := validateLineLength(bundled)
			if err != nil {
				t.Errorf("Line length validation failed for %s: %v", script.name, err)
			}

			// Also log the max line length for informational purposes
			maxLen := 0
			for _, line := range splitLines(bundled) {
				if len(line) > maxLen {
					maxLen = len(line)
				}
			}
			t.Logf("%s: max line length = %d characters (limit: %d)", script.name, maxLen, MaxLineLengthForActions)
		})
	}
}

// splitLines splits a string into lines
func splitLines(s string) []string {
	if s == "" {
		return []string{}
	}
	return strings.Split(s, "\n")
}
