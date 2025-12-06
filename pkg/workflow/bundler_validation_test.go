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

// TestValidateEmbeddedResourceRequires tests that ValidateEmbeddedResourceRequires
// correctly detects missing local requires in embedded resources
func TestValidateEmbeddedResourceRequires(t *testing.T) {
	tests := []struct {
		name        string
		sources     map[string]string
		expectError bool
		errorText   string
	}{
		{
			name: "all_dependencies_present",
			sources: map[string]string{
				"main.cjs":   `const { helper } = require("./helper.cjs");`,
				"helper.cjs": `module.exports = { helper: function() {} };`,
			},
			expectError: false,
		},
		{
			name: "missing_dependency",
			sources: map[string]string{
				"main.cjs": `const { missing } = require("./missing.cjs");`,
			},
			expectError: true,
			errorText:   "missing.cjs",
		},
		{
			name: "nested_directory_dependency_present",
			sources: map[string]string{
				"main.cjs":       `const { util } = require("./utils/util.cjs");`,
				"utils/util.cjs": `module.exports = { util: function() {} };`,
			},
			expectError: false,
		},
		{
			name: "parent_directory_dependency_missing",
			sources: map[string]string{
				"sub/main.cjs": `const { parent } = require("../parent.cjs");`,
			},
			expectError: true,
			errorText:   "parent.cjs",
		},
		{
			name: "parent_directory_dependency_present",
			sources: map[string]string{
				"sub/main.cjs": `const { parent } = require("../parent.cjs");`,
				"parent.cjs":   `module.exports = { parent: function() {} };`,
			},
			expectError: false,
		},
		{
			name: "multiple_files_all_present",
			sources: map[string]string{
				"main.cjs": `const { a } = require("./a.cjs"); const { b } = require("./b.cjs");`,
				"a.cjs":    `module.exports = { a: 1 };`,
				"b.cjs":    `module.exports = { b: 2 };`,
			},
			expectError: false,
		},
		{
			name: "multiple_files_one_missing",
			sources: map[string]string{
				"main.cjs": `const { a } = require("./a.cjs"); const { missing } = require("./missing.cjs");`,
				"a.cjs":    `module.exports = { a: 1 };`,
			},
			expectError: true,
			errorText:   "missing.cjs",
		},
		{
			name: "no_local_requires",
			sources: map[string]string{
				"main.cjs": `const fs = require("fs"); console.log("hello");`,
			},
			expectError: false,
		},
		{
			name: "auto_add_cjs_extension",
			sources: map[string]string{
				"main.cjs":   `const { helper } = require("./helper");`,
				"helper.cjs": `module.exports = { helper: function() {} };`,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmbeddedResourceRequires(tt.sources)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				} else if tt.errorText != "" && !strings.Contains(err.Error(), tt.errorText) {
					t.Errorf("Expected error to contain %q but got: %v", tt.errorText, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// TestValidateEmbeddedResourceRequires_RealSources tests the validation against actual embedded sources
func TestValidateEmbeddedResourceRequires_RealSources(t *testing.T) {
	sources := GetJavaScriptSources()

	if len(sources) == 0 {
		t.Fatal("GetJavaScriptSources() returned empty map")
	}

	t.Logf("Validating %d embedded JavaScript files", len(sources))

	err := ValidateEmbeddedResourceRequires(sources)
	if err != nil {
		t.Errorf("Validation failed on real embedded sources:\n%v\n\nThis indicates that some JavaScript files have local requires that reference files not in GetJavaScriptSources().\nPlease add the missing files to js.go", err)
	}
}
