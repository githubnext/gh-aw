package workflow

import (
	"strings"
	"testing"
)

// TestGetScriptFunctions tests that all script getter functions return non-empty scripts
func TestGetScriptFunctions(t *testing.T) {
	tests := []struct {
		name    string
		getFunc func() string
		minSize int // Minimum expected size in bytes
	}{
		{
			name:    "getCollectJSONLOutputScript",
			getFunc: getCollectJSONLOutputScript,
			minSize: 100,
		},
		{
			name:    "getComputeTextScript",
			getFunc: getComputeTextScript,
			minSize: 100,
		},
		{
			name:    "getSanitizeOutputScript",
			getFunc: getSanitizeOutputScript,
			minSize: 100,
		},
		{
			name:    "getCreateIssueScript",
			getFunc: getCreateIssueScript,
			minSize: 100,
		},
		{
			name:    "getAddLabelsScript",
			getFunc: getAddLabelsScript,
			minSize: 100,
		},
		{
			name:    "getParseFirewallLogsScript",
			getFunc: getParseFirewallLogsScript,
			minSize: 100,
		},
		{
			name:    "getCreateDiscussionScript",
			getFunc: getCreateDiscussionScript,
			minSize: 100,
		},
		{
			name:    "getUpdateIssueScript",
			getFunc: getUpdateIssueScript,
			minSize: 100,
		},
		{
			name:    "getCreateCodeScanningAlertScript",
			getFunc: getCreateCodeScanningAlertScript,
			minSize: 100,
		},
		{
			name:    "getCreatePRReviewCommentScript",
			getFunc: getCreatePRReviewCommentScript,
			minSize: 100,
		},
		{
			name:    "getAddCommentScript",
			getFunc: getAddCommentScript,
			minSize: 100,
		},
		{
			name:    "getUploadAssetsScript",
			getFunc: getUploadAssetsScript,
			minSize: 100,
		},
		{
			name:    "getPushToPullRequestBranchScript",
			getFunc: getPushToPullRequestBranchScript,
			minSize: 100,
		},
		{
			name:    "getCreatePullRequestScript",
			getFunc: getCreatePullRequestScript,
			minSize: 100,
		},
		{
			name:    "getInterpolatePromptScript",
			getFunc: getInterpolatePromptScript,
			minSize: 100,
		},
		{
			name:    "getParseClaudeLogScript",
			getFunc: getParseClaudeLogScript,
			minSize: 100,
		},
		{
			name:    "getParseCodexLogScript",
			getFunc: getParseCodexLogScript,
			minSize: 100,
		},
		{
			name:    "getParseCopilotLogScript",
			getFunc: getParseCopilotLogScript,
			minSize: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the function
			script := tt.getFunc()

			// Verify the script is not empty
			if script == "" {
				t.Errorf("%s returned empty script", tt.name)
			}

			// Verify the script meets minimum size requirements
			if len(script) < tt.minSize {
				t.Errorf("%s returned script smaller than expected: got %d bytes, want at least %d bytes",
					tt.name, len(script), tt.minSize)
			}

			// Verify the script is valid JavaScript (basic check)
			// All our scripts should have some common JavaScript patterns
			hasValidJSPattern := strings.Contains(script, "function") ||
				strings.Contains(script, "const") ||
				strings.Contains(script, "var") ||
				strings.Contains(script, "let") ||
				strings.Contains(script, "async") ||
				strings.Contains(script, "require")

			if !hasValidJSPattern {
				t.Errorf("%s returned script that doesn't look like JavaScript", tt.name)
			}
		})
	}
}

// TestScriptBundlingIdempotency tests that calling script functions multiple times returns the same result
func TestScriptBundlingIdempotency(t *testing.T) {
	tests := []struct {
		name    string
		getFunc func() string
	}{
		{"getCollectJSONLOutputScript", getCollectJSONLOutputScript},
		{"getComputeTextScript", getComputeTextScript},
		{"getSanitizeOutputScript", getSanitizeOutputScript},
		{"getCreateIssueScript", getCreateIssueScript},
		{"getParseClaudeLogScript", getParseClaudeLogScript},
		{"getParseCodexLogScript", getParseCodexLogScript},
		{"getParseCopilotLogScript", getParseCopilotLogScript},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the function twice
			first := tt.getFunc()
			second := tt.getFunc()

			// Verify both calls return the same script
			if first != second {
				t.Errorf("%s returned different scripts on consecutive calls", tt.name)
			}

			// Verify the result is cached (same memory address would be ideal, but we can't test that easily)
			// Instead, we verify the content is exactly identical
			if len(first) != len(second) {
				t.Errorf("%s returned scripts with different lengths: first=%d, second=%d",
					tt.name, len(first), len(second))
			}
		})
	}
}

// TestScriptContainsExpectedPatterns tests that scripts contain expected patterns
func TestScriptContainsExpectedPatterns(t *testing.T) {
	tests := []struct {
		name            string
		getFunc         func() string
		expectedPattern string
		description     string
	}{
		{
			name:            "create_issue contains create logic",
			getFunc:         getCreateIssueScript,
			expectedPattern: "create",
			description:     "create issue script should contain 'create'",
		},
		{
			name:            "add_labels contains label logic",
			getFunc:         getAddLabelsScript,
			expectedPattern: "label",
			description:     "add labels script should contain 'label'",
		},
		{
			name:            "create_discussion contains discussion logic",
			getFunc:         getCreateDiscussionScript,
			expectedPattern: "discussion",
			description:     "create discussion script should contain 'discussion'",
		},
		{
			name:            "update_issue contains update logic",
			getFunc:         getUpdateIssueScript,
			expectedPattern: "update",
			description:     "update issue script should contain 'update'",
		},
		{
			name:            "add_comment contains comment logic",
			getFunc:         getAddCommentScript,
			expectedPattern: "comment",
			description:     "add comment script should contain 'comment'",
		},
		{
			name:            "create_pull_request contains PR logic",
			getFunc:         getCreatePullRequestScript,
			expectedPattern: "pull",
			description:     "create pull request script should contain 'pull'",
		},
		{
			name:            "parse_claude_log contains parsing logic",
			getFunc:         getParseClaudeLogScript,
			expectedPattern: "parse",
			description:     "parse claude log script should contain 'parse'",
		},
		{
			name:            "parse_codex_log contains parsing logic",
			getFunc:         getParseCodexLogScript,
			expectedPattern: "parse",
			description:     "parse codex log script should contain 'parse'",
		},
		{
			name:            "parse_copilot_log contains parsing logic",
			getFunc:         getParseCopilotLogScript,
			expectedPattern: "parse",
			description:     "parse copilot log script should contain 'parse'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := tt.getFunc()
			scriptLower := strings.ToLower(script)

			if !strings.Contains(scriptLower, tt.expectedPattern) {
				t.Errorf("%s: expected pattern %q not found in script", tt.description, tt.expectedPattern)
			}
		})
	}
}

// TestScriptNonEmpty tests that embedded source scripts are not empty
// SKIPPED: Scripts are now loaded from external files at runtime using require() pattern
func TestScriptNonEmpty(t *testing.T) {
	t.Skip("Script embedding tests skipped - scripts now use require() pattern to load external files")
}

// TestScriptBundlingDoesNotFail tests that bundling never returns empty strings
// SKIPPED: Scripts are now loaded from external files at runtime using require() pattern
func TestScriptBundlingDoesNotFail(t *testing.T) {
	t.Skip("Script bundling tests skipped - scripts now use require() pattern to load external files")
}
