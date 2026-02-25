//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// classifyCheckState – fixture-based unit tests
// ---------------------------------------------------------------------------

func TestClassifyCheckState_NoChecks(t *testing.T) {
	state := classifyCheckState([]PRCheckRun{}, []PRCommitStatus{})
	assert.Equal(t, CheckStateNoChecks, state, "empty check runs and statuses should yield no_checks")
}

func TestClassifyCheckState_AllSuccess(t *testing.T) {
	runs := []PRCheckRun{
		{Name: "build", Status: "completed", Conclusion: "success"},
		{Name: "lint", Status: "completed", Conclusion: "success"},
	}
	state := classifyCheckState(runs, nil)
	assert.Equal(t, CheckStateSuccess, state, "all successful check runs should yield success")
}

func TestClassifyCheckState_Failed(t *testing.T) {
	runs := []PRCheckRun{
		{Name: "build", Status: "completed", Conclusion: "success"},
		{Name: "test", Status: "completed", Conclusion: "failure"},
	}
	state := classifyCheckState(runs, nil)
	assert.Equal(t, CheckStateFailed, state, "at least one failed check run should yield failed")
}

func TestClassifyCheckState_Pending(t *testing.T) {
	runs := []PRCheckRun{
		{Name: "build", Status: "completed", Conclusion: "success"},
		{Name: "test", Status: "in_progress", Conclusion: ""},
	}
	state := classifyCheckState(runs, nil)
	assert.Equal(t, CheckStatePending, state, "in-progress check run should yield pending")
}

func TestClassifyCheckState_Queued(t *testing.T) {
	runs := []PRCheckRun{
		{Name: "build", Status: "queued", Conclusion: ""},
	}
	state := classifyCheckState(runs, nil)
	assert.Equal(t, CheckStatePending, state, "queued check run should yield pending")
}

func TestClassifyCheckState_PolicyBlocked(t *testing.T) {
	runs := []PRCheckRun{
		{Name: "Branch protection rule check", Status: "completed", Conclusion: "failure"},
	}
	state := classifyCheckState(runs, nil)
	assert.Equal(t, CheckStatePolicyBlocked, state, "branch protection rule failure should yield policy_blocked")
}

func TestClassifyCheckState_PolicyBlockedActionRequired(t *testing.T) {
	runs := []PRCheckRun{
		{Name: "build", Status: "completed", Conclusion: "success"},
		{Name: "required status check", Status: "completed", Conclusion: "action_required"},
	}
	state := classifyCheckState(runs, nil)
	assert.Equal(t, CheckStatePolicyBlocked, state, "action_required on policy check should yield policy_blocked")
}

func TestClassifyCheckState_PolicyBlockedWithFailures(t *testing.T) {
	// If both a policy check and a real failure are present, failed takes priority.
	runs := []PRCheckRun{
		{Name: "required status check", Status: "completed", Conclusion: "failure"},
		{Name: "test suite", Status: "completed", Conclusion: "failure"},
	}
	state := classifyCheckState(runs, nil)
	assert.Equal(t, CheckStateFailed, state, "real failure alongside policy check should yield failed, not policy_blocked")
}

func TestClassifyCheckState_CommitStatusNoChecks(t *testing.T) {
	state := classifyCheckState(nil, []PRCommitStatus{})
	assert.Equal(t, CheckStateNoChecks, state, "empty commit statuses should yield no_checks")
}

func TestClassifyCheckState_CommitStatusPending(t *testing.T) {
	statuses := []PRCommitStatus{
		{Context: "ci/circleci", State: "pending"},
	}
	state := classifyCheckState(nil, statuses)
	assert.Equal(t, CheckStatePending, state, "pending commit status should yield pending")
}

func TestClassifyCheckState_CommitStatusFailed(t *testing.T) {
	statuses := []PRCommitStatus{
		{Context: "ci/circleci", State: "failure"},
	}
	state := classifyCheckState(nil, statuses)
	assert.Equal(t, CheckStateFailed, state, "failure commit status should yield failed")
}

func TestClassifyCheckState_CommitStatusError(t *testing.T) {
	statuses := []PRCommitStatus{
		{Context: "ci/circleci", State: "error"},
	}
	state := classifyCheckState(nil, statuses)
	assert.Equal(t, CheckStateFailed, state, "error commit status should yield failed")
}

func TestClassifyCheckState_CommitStatusSuccess(t *testing.T) {
	statuses := []PRCommitStatus{
		{Context: "ci/circleci", State: "success"},
	}
	state := classifyCheckState(nil, statuses)
	assert.Equal(t, CheckStateSuccess, state, "success commit status should yield success")
}

func TestClassifyCheckState_MixedRunsAndStatuses(t *testing.T) {
	runs := []PRCheckRun{
		{Name: "build", Status: "completed", Conclusion: "success"},
	}
	statuses := []PRCommitStatus{
		{Context: "ci/circleci", State: "pending"},
	}
	state := classifyCheckState(runs, statuses)
	assert.Equal(t, CheckStatePending, state, "pending status with successful run should yield pending")
}

func TestClassifyCheckState_TimedOut(t *testing.T) {
	runs := []PRCheckRun{
		{Name: "slow-test", Status: "completed", Conclusion: "timed_out"},
	}
	state := classifyCheckState(runs, nil)
	assert.Equal(t, CheckStateFailed, state, "timed_out should yield failed")
}

// ---------------------------------------------------------------------------
// isPolicyCheck – pattern matching tests
// ---------------------------------------------------------------------------

func TestIsPolicyCheck(t *testing.T) {
	tests := []struct {
		name      string
		checkName string
		expected  bool
	}{
		{
			name:      "branch protection pattern",
			checkName: "Branch protection rule check",
			expected:  true,
		},
		{
			name:      "required status check pattern",
			checkName: "Required status check",
			expected:  true,
		},
		{
			name:      "mergeability pattern",
			checkName: "Mergeability check",
			expected:  true,
		},
		{
			name:      "policy check pattern",
			checkName: "policy check for org",
			expected:  true,
		},
		{
			name:      "normal test run",
			checkName: "unit tests",
			expected:  false,
		},
		{
			name:      "build check",
			checkName: "build / linux",
			expected:  false,
		},
		{
			name:      "empty string",
			checkName: "",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isPolicyCheck(tt.checkName)
			assert.Equal(t, tt.expected, got, "isPolicyCheck(%q) should return %v", tt.checkName, tt.expected)
		})
	}
}

// ---------------------------------------------------------------------------
// NewChecksCommand – command shape tests
// ---------------------------------------------------------------------------

func TestChecksCommand(t *testing.T) {
	cmd := NewChecksCommand()
	require.NotNil(t, cmd, "checks command should not be nil")
	assert.Equal(t, "checks", cmd.Name(), "command name should be 'checks'")
	assert.True(t, cmd.HasAvailableFlags(), "command should expose flags")

	repoFlag := cmd.Flags().Lookup("repo")
	require.NotNil(t, repoFlag, "should have --repo flag")
	assert.Empty(t, repoFlag.DefValue, "--repo default should be empty")

	jsonFlag := cmd.Flags().Lookup("json")
	require.NotNil(t, jsonFlag, "should have --json flag")
	assert.Equal(t, "false", jsonFlag.DefValue, "--json default should be false")
}

func TestChecksCommand_RequiresArg(t *testing.T) {
	cmd := NewChecksCommand()
	err := cmd.Args(cmd, []string{})
	assert.Error(t, err, "checks command should require exactly one argument")
}

func TestChecksCommand_AcceptsOneArg(t *testing.T) {
	cmd := NewChecksCommand()
	err := cmd.Args(cmd, []string{"42"})
	assert.NoError(t, err, "checks command should accept exactly one argument")
}

func TestChecksCommand_RejectsMultipleArgs(t *testing.T) {
	cmd := NewChecksCommand()
	err := cmd.Args(cmd, []string{"42", "43"})
	assert.Error(t, err, "checks command should reject more than one argument")
}

// ---------------------------------------------------------------------------
// ChecksResult JSON serialization
// ---------------------------------------------------------------------------

func TestChecksResultJSONShape(t *testing.T) {
	result := &ChecksResult{
		State:    CheckStateFailed,
		PRNumber: "42",
		HeadSHA:  "abc123",
		CheckRuns: []PRCheckRun{
			{Name: "build", Status: "completed", Conclusion: "failure", HTMLURL: "https://example.com"},
		},
		Statuses:   []PRCommitStatus{},
		TotalCount: 1,
	}

	require.Equal(t, CheckStateFailed, result.State, "state should be failed")
	require.Equal(t, "42", result.PRNumber, "PR number should be preserved")
	require.Equal(t, "abc123", result.HeadSHA, "head SHA should be preserved")
	require.Len(t, result.CheckRuns, 1, "should have one check run")
	assert.Equal(t, "build", result.CheckRuns[0].Name, "check run name should be preserved")
}

// ---------------------------------------------------------------------------
// classifyGHAPIError – error classification tests
// ---------------------------------------------------------------------------

func TestClassifyGHAPIError_NotFound(t *testing.T) {
	err := classifyGHAPIError(1, "HTTP 404: Not Found", "42", "")
	require.Error(t, err, "should return an error")
	msg := err.Error()
	assert.Contains(t, msg, "not found", "error should mention not found")
	assert.Contains(t, msg, "#42", "error should mention PR number")
	assert.Contains(t, msg, "current repository", "error should mention current repository when no repo override")
}

func TestClassifyGHAPIError_NotFoundWithRepo(t *testing.T) {
	err := classifyGHAPIError(1, "HTTP 404: Not Found", "99", "myorg/myrepo")
	require.Error(t, err, "should return an error")
	msg := err.Error()
	assert.Contains(t, msg, "myorg/myrepo", "error should mention the specified repo")
}

func TestClassifyGHAPIError_Forbidden(t *testing.T) {
	err := classifyGHAPIError(1, "HTTP 403: Forbidden", "42", "")
	require.Error(t, err, "should return an error")
	msg := err.Error()
	assert.Contains(t, msg, "authentication failed", "error should mention auth failure")
	assert.Contains(t, msg, "gh auth login", "error should suggest running gh auth login")
}

func TestClassifyGHAPIError_Unauthorized(t *testing.T) {
	err := classifyGHAPIError(1, "HTTP 401: Unauthorized (Bad credentials)", "42", "")
	require.Error(t, err, "should return an error")
	msg := err.Error()
	assert.Contains(t, msg, "authentication failed", "error should mention auth failure")
}

func TestClassifyGHAPIError_BadCredentials(t *testing.T) {
	err := classifyGHAPIError(1, "Bad credentials", "42", "")
	require.Error(t, err, "should return an error")
	msg := err.Error()
	assert.Contains(t, msg, "authentication failed", "bad credentials should yield auth error")
}

func TestClassifyGHAPIError_Generic(t *testing.T) {
	err := classifyGHAPIError(1, "HTTP 500: Internal Server Error", "42", "")
	require.Error(t, err, "should return an error")
	msg := err.Error()
	assert.Contains(t, msg, "gh api call failed", "generic errors should surface exit code message")
}
