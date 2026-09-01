//go:build !integration

package cli

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureStderrForGuardPolicyReportTest captures stderr output produced while
// running fn, for use by the guard-policy dry-run report tests in this file.
func captureStderrForGuardPolicyReportTest(fn func()) string {
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	os.Stderr = w

	fn()

	if err := w.Close(); err != nil {
		panic(err)
	}
	os.Stderr = old

	out, err := io.ReadAll(r)
	if err != nil {
		panic(err)
	}
	return string(out)
}

// TestBuildGuardPolicyDryRunReport_NoGuardPolicy verifies that no report is
// produced when no guard-policy fields are configured.
func TestBuildGuardPolicyDryRunReport_NoGuardPolicy(t *testing.T) {
	assert.Nil(t, buildGuardPolicyDryRunReport("test.md", nil))
	assert.Nil(t, buildGuardPolicyDryRunReport("test.md", &workflow.GitHubToolConfig{}))
}

// TestBuildGuardPolicyDryRunReport_AllowedReposAndMinIntegrity verifies that a
// dry-run report is produced summarizing allowed-repos and min-integrity.
func TestBuildGuardPolicyDryRunReport_AllowedReposAndMinIntegrity(t *testing.T) {
	github := &workflow.GitHubToolConfig{
		AllowedRepos: workflow.GitHubReposScope{"owner/repo-b", "owner/repo-a"},
		MinIntegrity: workflow.GitHubIntegrityApproved,
	}

	report := buildGuardPolicyDryRunReport("test-workflow.md", github)
	require.NotNil(t, report)
	assert.Equal(t, "test-workflow.md", report.Workflow)
	assert.False(t, report.Lockdown)
	assert.Equal(t, "owner/repo-a, owner/repo-b", report.PermittedRepos)
	assert.Equal(t, "approved", report.MinIntegrity)
}

// TestBuildGuardPolicyDryRunReport_AllScope verifies that an "all" scope
// (or omitted allowed-repos) is reported as such.
func TestBuildGuardPolicyDryRunReport_AllScope(t *testing.T) {
	github := &workflow.GitHubToolConfig{
		MinIntegrity: workflow.GitHubIntegrityNone,
	}

	report := buildGuardPolicyDryRunReport("test-workflow.md", github)
	require.NotNil(t, report)
	assert.Equal(t, "all (default)", report.PermittedRepos)
	assert.Equal(t, "none", report.MinIntegrity)
}

// TestBuildGuardPolicyDryRunReport_BlockedTrustedApproval verifies that
// blocked-users, trusted-users, and approval-labels are surfaced in the report.
func TestBuildGuardPolicyDryRunReport_BlockedTrustedApproval(t *testing.T) {
	github := &workflow.GitHubToolConfig{
		MinIntegrity:   workflow.GitHubIntegrityApproved,
		BlockedUsers:   []string{"spam-bot"},
		TrustedUsers:   []string{"trusted-user"},
		ApprovalLabels: []string{"human-reviewed"},
	}

	report := buildGuardPolicyDryRunReport("test-workflow.md", github)
	require.NotNil(t, report)
	assert.Equal(t, []string{"spam-bot"}, report.BlockedUsers)
	assert.Equal(t, []string{"trusted-user"}, report.TrustedUsers)
	assert.Equal(t, []string{"human-reviewed"}, report.ApprovalLabels)

	rendered := formatGuardPolicyDryRunReport(report)
	assert.Contains(t, rendered, "blocked-users: spam-bot")
	assert.Contains(t, rendered, "trusted-users: trusted-user")
	assert.Contains(t, rendered, "approval-labels: human-reviewed")
}

// TestBuildGuardPolicyDryRunReport_Lockdown verifies that lockdown is
// surfaced in the report to make clear the guard-policy fields are ignored
// at runtime per §9.5 of scratchpad/github-mcp-access-control-specification.md.
func TestBuildGuardPolicyDryRunReport_Lockdown(t *testing.T) {
	github := &workflow.GitHubToolConfig{
		Lockdown:     true,
		AllowedRepos: workflow.GitHubReposScope{"all"},
		MinIntegrity: workflow.GitHubIntegrityApproved,
	}

	report := buildGuardPolicyDryRunReport("test-workflow.md", github)
	require.NotNil(t, report)
	assert.True(t, report.Lockdown)
	assert.Equal(t, "all", report.PermittedRepos)

	rendered := formatGuardPolicyDryRunReport(report)
	assert.Contains(t, rendered, "lockdown: true")
}

// TestBuildGuardPolicyDryRunReport_LockdownDeprecatedRepos verifies that the
// lockdown indicator is also surfaced for the deprecated repos alias.
func TestBuildGuardPolicyDryRunReport_LockdownDeprecatedRepos(t *testing.T) {
	github := &workflow.GitHubToolConfig{
		Lockdown:     true,
		Repos:        workflow.GitHubReposScope{"all"},
		MinIntegrity: workflow.GitHubIntegrityApproved,
	}

	report := buildGuardPolicyDryRunReport("test-workflow.md", github)
	require.NotNil(t, report)
	assert.True(t, report.Lockdown)
	assert.Equal(t, "all", report.PermittedRepos)

	rendered := formatGuardPolicyDryRunReport(report)
	assert.Contains(t, rendered, "lockdown: true")
	assert.Contains(t, rendered, "allowed-repos: all")
	for line := range strings.SplitSeq(rendered, "\n") {
		assert.False(t, strings.HasPrefix(strings.TrimSpace(line), "repos:"),
			"rendered output should not contain deprecated 'repos:' key: %q", line)
	}
}

// TestPrintGuardPolicyDryRunReport_OnlyWhenStrict verifies that the dry-run
// report is only emitted when --strict is set, and only when guard-policy
// fields are configured.
func TestPrintGuardPolicyDryRunReport_OnlyWhenStrict(t *testing.T) {
	workflowData := &workflow.WorkflowData{
		ParsedTools: &workflow.Tools{
			GitHub: &workflow.GitHubToolConfig{
				MinIntegrity: workflow.GitHubIntegrityApproved,
			},
		},
	}

	stderrOutput := captureStderrForGuardPolicyReportTest(func() {
		printGuardPolicyDryRunReport("test-workflow.md", workflowData, false)
	})
	assert.Empty(t, stderrOutput, "no report should be emitted when --strict is not set")

	stderrOutput = captureStderrForGuardPolicyReportTest(func() {
		printGuardPolicyDryRunReport("test-workflow.md", workflowData, true)
	})
	assert.Contains(t, stderrOutput, "guard policy dry-run report for test-workflow.md")

	noGuardPolicyData := &workflow.WorkflowData{ParsedTools: &workflow.Tools{}}
	stderrOutput = captureStderrForGuardPolicyReportTest(func() {
		printGuardPolicyDryRunReport("test-workflow.md", noGuardPolicyData, true)
	})
	assert.Empty(t, stderrOutput, "no report should be emitted without guard-policy fields")
}
