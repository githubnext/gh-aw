package workflow

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
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
