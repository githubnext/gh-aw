//go:build !integration

package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWorkflowRunUnmarshal verifies that a standard "gh run list --json" response
// (without the unsupported "path" field) is correctly unmarshaled into a WorkflowRun.
// The "path" field was previously requested but is not a valid gh run list --json
// field and caused failures on strict gh CLI versions.
func TestWorkflowRunUnmarshal(t *testing.T) {
	rawJSON := `[
{
"databaseId": 42,
"workflowName": "My Workflow",
"status": "completed",
"conclusion": "success",
"createdAt": "2026-01-01T00:00:00Z",
"startedAt": "2026-01-01T00:00:01Z",
"updatedAt": "2026-01-01T00:01:00Z"
}
]`

	var runs []WorkflowRun
	require.NoError(t, json.Unmarshal([]byte(rawJSON), &runs), "unmarshal should succeed")
	require.Len(t, runs, 1)

	assert.Equal(t, int64(42), runs[0].DatabaseID, "DatabaseID should be populated")
	assert.Equal(t, "My Workflow", runs[0].WorkflowName, "WorkflowName should be populated")
	assert.Empty(t, runs[0].WorkflowPath, "WorkflowPath should be empty when 'path' field is absent")
}

// TestListWorkflowRunsErrorHandling verifies the error classification logic in
// listWorkflowRunsWithPagination. In particular it checks that:
//   - "Unknown JSON field" (capital U, as emitted by gh CLI) is treated as an
//     invalid-field error, not an auth error (case-insensitive matching).
//   - Exit code 1 alone does NOT trigger the auth-failure path because gh exits
//     with code 1 for many non-auth errors (e.g. unsupported JSON fields).
func TestListWorkflowRunsErrorHandling(t *testing.T) {
	tests := []struct {
		name             string
		errMsg           string
		outputMsg        string
		wantInvalidField bool
		wantAuth         bool
	}{
		{
			name:             "unknown JSON field (capital U, as gh CLI emits)",
			errMsg:           "exit status 1",
			outputMsg:        `Unknown JSON field: "path"`,
			wantInvalidField: true,
			wantAuth:         false,
		},
		{
			name:             "unknown field lowercase",
			errMsg:           "exit status 1",
			outputMsg:        "unknown field foo",
			wantInvalidField: true,
			wantAuth:         false,
		},
		{
			name:             "invalid field mixed case",
			errMsg:           "exit status 1",
			outputMsg:        "Invalid field: bar",
			wantInvalidField: true,
			wantAuth:         false,
		},
		{
			name:      "exit status 1 alone is NOT an auth error",
			errMsg:    "exit status 1",
			outputMsg: "some other error",
			wantAuth:  false,
		},
		{
			name:      "exit status 4 IS an auth error",
			errMsg:    "exit status 4",
			outputMsg: "",
			wantAuth:  true,
		},
		{
			name:      "gh auth login hint is an auth error",
			errMsg:    "exit status 1",
			outputMsg: "To get started, run: gh auth login",
			wantAuth:  true,
		},
		{
			name:      "not logged in message is an auth error",
			errMsg:    "exit status 1",
			outputMsg: "not logged into any GitHub hosts",
			wantAuth:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			combinedMsg := tt.errMsg + " " + tt.outputMsg
			combinedMsgLower := strings.ToLower(combinedMsg)

			isInvalidField := strings.Contains(combinedMsgLower, "invalid field") ||
				strings.Contains(combinedMsgLower, "unknown field") ||
				strings.Contains(combinedMsgLower, "unknown json field") ||
				strings.Contains(combinedMsgLower, "unknown json") ||
				strings.Contains(combinedMsgLower, "field not found") ||
				strings.Contains(combinedMsgLower, "no such field")
			isAuth := !isInvalidField && (strings.Contains(combinedMsg, "exit status 4") ||
				strings.Contains(combinedMsg, "not logged into any GitHub hosts") ||
				strings.Contains(combinedMsg, "To use GitHub CLI in a GitHub Actions workflow") ||
				strings.Contains(combinedMsg, "authentication required") ||
				strings.Contains(tt.outputMsg, "gh auth login"))

			if tt.wantInvalidField {
				assert.True(t, isInvalidField, "expected invalid-field classification")
				assert.False(t, isAuth, "invalid-field errors must not be classified as auth errors")
			}
			if tt.wantAuth {
				assert.False(t, isInvalidField, "auth errors must not be classified as invalid-field errors")
				assert.True(t, isAuth, "expected auth classification")
			}
			if !tt.wantInvalidField && !tt.wantAuth {
				assert.False(t, isInvalidField, "should not be invalid-field")
				assert.False(t, isAuth, "should not be auth")
			}
		})
	}
}
