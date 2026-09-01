package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var guardPolicyReportLog = logger.New("cli:compile_guard_policy_report")

// guardPolicyDryRunReport summarizes the effective GitHub guard-policy
// configuration for a single compiled workflow. It is produced in --strict
// mode as a compile-time dry-run of which repositories the guard policy
// would permit or deny, addressing Open Question #4 in
// scratchpad/guard-policies-specification.md ("Should we add a 'dry-run'
// mode to test policies before enforcement?").
type guardPolicyDryRunReport struct {
	Workflow       string
	Lockdown       bool
	PermittedRepos string
	MinIntegrity   string
	BlockedUsers   []string
	TrustedUsers   []string
	ApprovalLabels []string
}

// hasGuardPolicyFields reports whether the GitHub tool config has any
// guard-policy fields configured (allowed-repos, min-integrity, blocked-users,
// trusted-users, approval-labels).
func hasGuardPolicyFields(github *workflow.GitHubToolConfig) bool {
	if github == nil {
		return false
	}
	hasRepos := github.AllowedRepos != nil || github.Repos != nil
	hasMinIntegrity := github.MinIntegrity != ""
	hasBlockedUsers := len(github.BlockedUsers) > 0 || github.BlockedUsersExpr != ""
	hasApprovalLabels := len(github.ApprovalLabels) > 0 || github.ApprovalLabelsExpr != ""
	hasTrustedUsers := len(github.TrustedUsers) > 0 || github.TrustedUsersExpr != ""
	return hasRepos || hasMinIntegrity || hasBlockedUsers || hasApprovalLabels || hasTrustedUsers
}

// formatGuardPolicyReposScope renders a GitHubReposScope value ("all",
// "public", or an array of repository patterns) as a human-readable string
// for the dry-run report.
func formatGuardPolicyReposScope(scope workflow.GitHubReposScope) string {
	if len(scope) == 0 {
		return "all (default)"
	}
	patterns := append([]string(nil), scope...)
	sort.Strings(patterns)
	return strings.Join(patterns, ", ")
}

// buildGuardPolicyDryRunReport builds a guardPolicyDryRunReport for a
// workflow's GitHub tool configuration. Returns nil when no guard-policy
// fields are configured (nothing to report).
func buildGuardPolicyDryRunReport(workflowName string, github *workflow.GitHubToolConfig) *guardPolicyDryRunReport {
	if github == nil || !hasGuardPolicyFields(github) {
		guardPolicyReportLog.Print("No guard-policy fields configured, skipping dry-run report")
		return nil
	}

	repos := github.AllowedRepos
	if repos == nil {
		repos = github.Repos
	}

	minIntegrity := string(github.MinIntegrity)
	if minIntegrity == "" {
		minIntegrity = "none (default)"
	}

	return &guardPolicyDryRunReport{
		Workflow:       workflowName,
		Lockdown:       github.Lockdown,
		PermittedRepos: formatGuardPolicyReposScope(repos),
		MinIntegrity:   minIntegrity,
		BlockedUsers:   github.BlockedUsers,
		TrustedUsers:   github.TrustedUsers,
		ApprovalLabels: github.ApprovalLabels,
	}
}

// formatGuardPolicyDryRunReport renders a guardPolicyDryRunReport as a
// human-readable multi-line string for --strict compile output.
func formatGuardPolicyDryRunReport(report *guardPolicyDryRunReport) string {
	if report == nil {
		return ""
	}

	var b strings.Builder
	fmt.Fprintf(&b, "guard policy dry-run report for %s:\n", report.Workflow)
	if report.Lockdown {
		b.WriteString("  lockdown: true (guard-policy fields below are not evaluated at runtime)\n")
	}
	fmt.Fprintf(&b, "  allowed-repos: %s\n", report.PermittedRepos)
	fmt.Fprintf(&b, "  min-integrity: %s\n", report.MinIntegrity)
	if len(report.BlockedUsers) > 0 {
		fmt.Fprintf(&b, "  blocked-users: %s\n", strings.Join(report.BlockedUsers, ", "))
	}
	if len(report.TrustedUsers) > 0 {
		fmt.Fprintf(&b, "  trusted-users: %s\n", strings.Join(report.TrustedUsers, ", "))
	}
	if len(report.ApprovalLabels) > 0 {
		fmt.Fprintf(&b, "  approval-labels: %s\n", strings.Join(report.ApprovalLabels, ", "))
	}
	return strings.TrimRight(b.String(), "\n")
}

// printGuardPolicyDryRunReport prints a compile-time guard-policy dry-run
// report to stderr when --strict is set and the workflow has a GitHub guard
// policy configured. It is a no-op otherwise.
func printGuardPolicyDryRunReport(workflowName string, workflowData *workflow.WorkflowData, strict bool) {
	if !strict || workflowData == nil || workflowData.ParsedTools == nil {
		guardPolicyReportLog.Print("Strict mode disabled or tools not parsed, skipping guard-policy dry-run report")
		return
	}

	report := buildGuardPolicyDryRunReport(workflowName, workflowData.ParsedTools.GitHub)
	if report == nil {
		return
	}

	guardPolicyReportLog.Printf("Printing guard-policy dry-run report for workflow: %s", workflowName)
	fmt.Fprintln(os.Stderr, console.FormatInfoMessageStderr(formatGuardPolicyDryRunReport(report)))
}
