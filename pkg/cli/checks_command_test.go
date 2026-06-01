//go:build !integration

package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureOutputMu guards os.Stdout/os.Stderr reassignment in captureOutput.
// This intentionally serializes tests using captureOutput because stdout/stderr
// are global process state. Tests that call captureOutput should not use
// t.Parallel.
var captureOutputMu sync.Mutex

// ---------------------------------------------------------------------------
// classifyCheckState – fixture-based unit tests
// ---------------------------------------------------------------------------

func TestClassifyCheckState(t *testing.T) {
	tests := []struct {
		name     string
		runs     []PRCheckRun
		statuses []PRCommitStatus
		want     CheckState
	}{
		{
			name: "empty check runs and statuses yields no_checks",
			want: CheckStateNoChecks,
		},
		{
			name: "all successful check runs yields success",
			runs: []PRCheckRun{
				{Name: "build", Status: "completed", Conclusion: "success"},
				{Name: "lint", Status: "completed", Conclusion: "success"},
			},
			want: CheckStateSuccess,
		},
		{
			name: "failed check run yields failed",
			runs: []PRCheckRun{
				{Name: "build", Status: "completed", Conclusion: "success"},
				{Name: "test", Status: "completed", Conclusion: "failure"},
			},
			want: CheckStateFailed,
		},
		{
			name: "in progress check run yields pending",
			runs: []PRCheckRun{
				{Name: "build", Status: "completed", Conclusion: "success"},
				{Name: "test", Status: "in_progress", Conclusion: ""},
			},
			want: CheckStatePending,
		},
		{
			name: "queued check run yields pending",
			runs: []PRCheckRun{
				{Name: "build", Status: "queued", Conclusion: ""},
			},
			want: CheckStatePending,
		},
		{
			name: "branch protection rule failure yields policy_blocked",
			runs: []PRCheckRun{
				{Name: "Branch protection rule check", Status: "completed", Conclusion: "failure"},
			},
			want: CheckStatePolicyBlocked,
		},
		{
			name: "action required on policy check yields policy_blocked",
			runs: []PRCheckRun{
				{Name: "build", Status: "completed", Conclusion: "success"},
				{Name: "required status check", Status: "completed", Conclusion: "action_required"},
			},
			want: CheckStatePolicyBlocked,
		},
		{
			name: "real failure with policy check yields failed",
			runs: []PRCheckRun{
				{Name: "required status check", Status: "completed", Conclusion: "failure"},
				{Name: "test suite", Status: "completed", Conclusion: "failure"},
			},
			want: CheckStateFailed,
		},
		{
			name:     "empty commit statuses yields no_checks",
			statuses: []PRCommitStatus{},
			want:     CheckStateNoChecks,
		},
		{
			name: "pending commit status yields pending",
			statuses: []PRCommitStatus{
				{Context: "ci/circleci", State: "pending"},
			},
			want: CheckStatePending,
		},
		{
			name: "failure commit status yields failed",
			statuses: []PRCommitStatus{
				{Context: "ci/circleci", State: "failure"},
			},
			want: CheckStateFailed,
		},
		{
			name: "error commit status yields failed",
			statuses: []PRCommitStatus{
				{Context: "ci/circleci", State: "error"},
			},
			want: CheckStateFailed,
		},
		{
			name: "success commit status yields success",
			statuses: []PRCommitStatus{
				{Context: "ci/circleci", State: "success"},
			},
			want: CheckStateSuccess,
		},
		{
			name: "successful run with pending status yields pending",
			runs: []PRCheckRun{
				{Name: "build", Status: "completed", Conclusion: "success"},
			},
			statuses: []PRCommitStatus{
				{Context: "ci/circleci", State: "pending"},
			},
			want: CheckStatePending,
		},
		{
			name: "timed out check run yields failed",
			runs: []PRCheckRun{
				{Name: "slow-test", Status: "completed", Conclusion: "timed_out"},
			},
			want: CheckStateFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyCheckState(tt.runs, tt.statuses)
			assert.Equal(t, tt.want, got, "classifyCheckState should return expected state")
		})
	}
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
		State:         CheckStateFailed,
		RequiredState: CheckStateSuccess,
		PRNumber:      "42",
		HeadSHA:       "abc123",
		CheckRuns: []PRCheckRun{
			{Name: "build", Status: "completed", Conclusion: "failure", HTMLURL: "https://example.com"},
		},
		Statuses:   []PRCommitStatus{},
		TotalCount: 1,
	}

	// Verify struct fields directly.
	require.Equal(t, CheckStateFailed, result.State, "state should be failed")
	require.Equal(t, CheckStateSuccess, result.RequiredState, "required_state should be success")
	require.Equal(t, "42", result.PRNumber, "PR number should be preserved")
	require.Equal(t, "abc123", result.HeadSHA, "head SHA should be preserved")
	require.Len(t, result.CheckRuns, 1, "should have one check run")
	assert.Equal(t, "build", result.CheckRuns[0].Name, "check run name should be preserved")

	// Marshal to JSON and verify key names match the json struct tags.
	data, err := json.Marshal(result)
	require.NoError(t, err, "should marshal to JSON without error")

	var decoded map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &decoded), "should unmarshal JSON without error")

	assert.Contains(t, decoded, "state", "JSON should contain 'state' key")
	assert.Contains(t, decoded, "required_state", "JSON should contain 'required_state' key")
	assert.Contains(t, decoded, "pr_number", "JSON should contain 'pr_number' key")
	assert.Contains(t, decoded, "head_sha", "JSON should contain 'head_sha' key")
	assert.Contains(t, decoded, "check_runs", "JSON should contain 'check_runs' key")
	assert.Contains(t, decoded, "statuses", "JSON should contain 'statuses' key")
	assert.Contains(t, decoded, "total_count", "JSON should contain 'total_count' key")

	assert.JSONEq(t, `"failed"`, string(decoded["state"]), "state JSON value should be 'failed'")
	assert.JSONEq(t, `"success"`, string(decoded["required_state"]), "required_state JSON value should be 'success'")
	assert.JSONEq(t, `"42"`, string(decoded["pr_number"]), "pr_number JSON value should be '42'")
	assert.JSONEq(t, `"abc123"`, string(decoded["head_sha"]), "head_sha JSON value should be 'abc123'")
}

// ---------------------------------------------------------------------------
// required_state — optional third-party commit status failures are excluded,
// but policy commit statuses (branch protection, etc.) are still included
// ---------------------------------------------------------------------------

// TestRequiredStateIgnoresCommitStatusFailures validates the core fix: a failing
// third-party commit status (e.g. Vercel, Netlify) must not pollute the
// required_state field. Check runs are posted by GitHub Actions; optional
// deployment commit statuses are posted by third-party integrations.
func TestRequiredStateFiltering(t *testing.T) {
	tests := []struct {
		name          string
		runs          []PRCheckRun
		statuses      []PRCommitStatus
		wantAggregate CheckState
		wantRequired  CheckState
	}{
		{
			name: "vercel failure only affects aggregate",
			runs: []PRCheckRun{
				{Name: "build", Status: "completed", Conclusion: "success"},
				{Name: "test", Status: "completed", Conclusion: "success"},
			},
			statuses: []PRCommitStatus{
				{Context: "vercel", State: "failure"},
			},
			wantAggregate: CheckStateFailed,
			wantRequired:  CheckStateSuccess,
		},
		{
			name: "netlify failure only affects aggregate",
			runs: []PRCheckRun{
				{Name: "ci", Status: "completed", Conclusion: "success"},
			},
			statuses: []PRCommitStatus{
				{Context: "netlify/my-site/deploy-preview", State: "failure"},
			},
			wantAggregate: CheckStateFailed,
			wantRequired:  CheckStateSuccess,
		},
		{
			name: "check run failure affects required state",
			runs: []PRCheckRun{
				{Name: "build", Status: "completed", Conclusion: "success"},
				{Name: "tests", Status: "completed", Conclusion: "failure"},
			},
			statuses: []PRCommitStatus{
				{Context: "vercel", State: "success"},
			},
			wantAggregate: CheckStateFailed,
			wantRequired:  CheckStateFailed,
		},
		{
			name: "non policy commit status without check runs yields no required checks",
			statuses: []PRCommitStatus{
				{Context: "ci/circleci", State: "success"},
			},
			wantAggregate: CheckStateSuccess,
			wantRequired:  CheckStateNoChecks,
		},
		{
			name: "policy commit status failure is retained for required state",
			runs: []PRCheckRun{
				{Name: "build", Status: "completed", Conclusion: "success"},
			},
			statuses: []PRCommitStatus{
				{Context: "branch protection rule check", State: "failure"},
				{Context: "vercel", State: "failure"},
			},
			wantAggregate: CheckStateFailed,
			wantRequired:  CheckStatePolicyBlocked,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aggregate := classifyCheckState(tt.runs, tt.statuses)
			assert.Equal(t, tt.wantAggregate, aggregate, "aggregate state should match expected value")

			required := classifyCheckState(tt.runs, filterCommitStatusesToPolicyChecks(tt.statuses))
			assert.Equal(t, tt.wantRequired, required, "required_state should match expected value")
		})
	}
}

// ---------------------------------------------------------------------------
// filterCommitStatusesToPolicyChecks – filter helper tests
// ---------------------------------------------------------------------------

func TestPolicyStatuses_FiltersNonPolicy(t *testing.T) {
	statuses := []PRCommitStatus{
		{Context: "vercel", State: "failure"},
		{Context: "netlify/deploy", State: "failure"},
		{Context: "branch protection rule check", State: "failure"},
	}
	filtered := filterCommitStatusesToPolicyChecks(statuses)
	require.Len(t, filtered, 1, "should retain only policy statuses")
	assert.Equal(t, "branch protection rule check", filtered[0].Context)
}

func TestPolicyStatuses_EmptyInput(t *testing.T) {
	assert.Nil(t, filterCommitStatusesToPolicyChecks(nil), "nil input should return nil")
	assert.Nil(t, filterCommitStatusesToPolicyChecks([]PRCommitStatus{}), "empty input should return nil")
}

func TestClassifyGHAPIError(t *testing.T) {
	tests := []struct {
		name        string
		exitCode    int
		stderr      string
		prNumber    string
		repo        string
		msgContains []string
	}{
		{
			name:     "404 not found uses current repository by default",
			exitCode: 1,
			stderr:   "HTTP 404: Not Found",
			prNumber: "42",
			msgContains: []string{
				"not found",
				"#42",
				"current repository",
			},
		},
		{
			name:     "404 not found includes explicit repository",
			exitCode: 1,
			stderr:   "HTTP 404: Not Found",
			prNumber: "99",
			repo:     "myorg/myrepo",
			msgContains: []string{
				"myorg/myrepo",
			},
		},
		{
			name:     "403 forbidden is classified as auth failure",
			exitCode: 1,
			stderr:   "HTTP 403: Forbidden",
			prNumber: "42",
			msgContains: []string{
				"authentication failed",
				"gh auth login",
			},
		},
		{
			name:     "401 unauthorized is classified as auth failure",
			exitCode: 1,
			stderr:   "HTTP 401: Unauthorized (Bad credentials)",
			prNumber: "42",
			msgContains: []string{
				"authentication failed",
			},
		},
		{
			name:     "bad credentials is classified as auth failure",
			exitCode: 1,
			stderr:   "Bad credentials",
			prNumber: "42",
			msgContains: []string{
				"authentication failed",
			},
		},
		{
			name:     "generic error keeps gh api call failed prefix",
			exitCode: 1,
			stderr:   "HTTP 500: Internal Server Error",
			prNumber: "42",
			msgContains: []string{
				"gh api call failed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := classifyGHAPIError(tt.exitCode, tt.stderr, tt.prNumber, tt.repo)
			require.Error(t, err, "should return an error")

			msg := err.Error()
			for _, snippet := range tt.msgContains {
				assert.Contains(t, msg, snippet, "error should contain expected message snippet")
			}
		})
	}
}

func TestPrintChecksJSON(t *testing.T) {
	result := &ChecksResult{
		State:         CheckStateSuccess,
		RequiredState: CheckStateSuccess,
		PRNumber:      "10",
		HeadSHA:       "abc123",
		CheckRuns: []PRCheckRun{
			{Name: "build", Status: "completed", Conclusion: "success", HTMLURL: "https://example.com/build"},
		},
		Statuses: []PRCommitStatus{
			{State: "success", Context: "ci/build"},
		},
		TotalCount: 2,
	}

	stdout, stderr := captureOutput(t, func() error {
		return printChecksJSON(result)
	})
	assert.Empty(t, stderr, "printChecksJSON should not write to stderr")

	var got ChecksResult
	require.NoError(t, json.Unmarshal([]byte(stdout), &got), "printChecksJSON output should be valid JSON")
	assert.Equal(t, *result, got, "printChecksJSON should output the provided result data")
}

func TestPrintChecksText(t *testing.T) {
	tests := []struct {
		name                string
		state               CheckState
		expectedStdout      string
		expectedStderrParts []string
	}{
		{
			name:           "success",
			state:          CheckStateSuccess,
			expectedStdout: "success\n",
			expectedStderrParts: []string{
				"PR #10: all checks passed (3 total)",
			},
		},
		{
			name:           "failed",
			state:          CheckStateFailed,
			expectedStdout: "failed\n",
			expectedStderrParts: []string{
				"PR #10: checks failed (3 total)",
			},
		},
		{
			name:           "pending",
			state:          CheckStatePending,
			expectedStdout: "pending\n",
			expectedStderrParts: []string{
				"PR #10: checks pending (3 total)",
			},
		},
		{
			name:           "no checks",
			state:          CheckStateNoChecks,
			expectedStdout: "no_checks\n",
			expectedStderrParts: []string{
				"PR #10: no checks configured or triggered",
			},
		},
		{
			name:           "policy blocked",
			state:          CheckStatePolicyBlocked,
			expectedStdout: "policy_blocked\n",
			expectedStderrParts: []string{
				"PR #10: blocked by policy or account gate (3 total)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ChecksResult{State: tt.state, PRNumber: "10", TotalCount: 3}
			stdout, stderr := captureOutput(t, func() error {
				return printChecksText(result)
			})

			assert.Equal(t, tt.expectedStdout, stdout, "printChecksText should write normalized state to stdout")
			for _, part := range tt.expectedStderrParts {
				assert.Contains(t, stderr, part, "printChecksText should write expected status message to stderr")
			}
		})
	}
}

func captureOutput(t *testing.T, fn func() error) (string, string) {
	t.Helper()
	// Do not call this helper from tests using t.Parallel; it manipulates
	// process-global stdout/stderr and must be serialized.
	captureOutputMu.Lock()
	defer captureOutputMu.Unlock()

	stdoutReader, stdoutWriter, err := os.Pipe()
	require.NoError(t, err, "should create stdout pipe")

	stderrReader, stderrWriter, err := os.Pipe()
	require.NoError(t, err, "should create stderr pipe")
	t.Cleanup(func() {
		_ = stdoutReader.Close()
		_ = stderrReader.Close()
	})

	origStdout := os.Stdout
	origStderr := os.Stderr
	var runErr error
	func() {
		os.Stdout = stdoutWriter
		os.Stderr = stderrWriter
		defer func() {
			os.Stdout = origStdout
			os.Stderr = origStderr
			_ = stdoutWriter.Close()
			_ = stderrWriter.Close()
		}()

		runErr = fn()
	}()
	require.NoError(t, runErr, "output function should not return an error")

	var stdoutBuf bytes.Buffer
	_, err = io.Copy(&stdoutBuf, stdoutReader)
	require.NoError(t, err, "should read stdout output")

	var stderrBuf bytes.Buffer
	_, err = io.Copy(&stderrBuf, stderrReader)
	require.NoError(t, err, "should read stderr output")

	return stdoutBuf.String(), stderrBuf.String()
}
