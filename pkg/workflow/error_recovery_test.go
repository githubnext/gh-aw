//go:build !integration

package workflow

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandErrorMessages_UnwrapsJoinedErrors(t *testing.T) {
	err := errors.Join(
		NewValidationError("engine", "", "cannot be empty", "Add engine"),
		NewValidationError("permissions", "", "invalid scope", "Fix permissions"),
	)

	messages := ExpandErrorMessages(err)
	require.Len(t, messages, 2, "Joined errors should expand into separate messages")
	assert.Contains(t, messages[0], "field 'engine'", "Engine validation error should be preserved")
	assert.Contains(t, messages[1], "field 'permissions'", "Permissions validation error should be preserved")
}

func TestBuildPrioritizedErrorReportFromMessages_DefaultLimit(t *testing.T) {
	messages := []string{
		"workflow.md:4:1: error: deprecated field",
		"workflow.md:3:1: error: network.allowed requires strict mode",
		"workflow.md:2:1: error: invalid engine value 'copiliot'",
		"workflow.md:6:1: error: runtime version conflict",
		"workflow.md:5:1: error: event filter is invalid",
		"workflow.md:7:1: error: tools.github config invalid",
	}

	report := BuildPrioritizedErrorReportFromMessages(messages, false)

	require.Equal(t, 6, report.TotalCount, "All non-suppressed errors should be counted")
	require.Len(t, report.DisplayedErrors, 5, "Default report should limit output to five errors")
	assert.Equal(t, SeverityCritical, report.DisplayedErrors[0].Severity, "Critical errors should be first")
	assert.Contains(t, report.DisplayedErrors[0].Message, "invalid engine", "Highest-priority error should be the invalid engine")
	assert.Equal(t, SeverityHigh, report.DisplayedErrors[1].Severity, "High-priority errors should immediately follow critical errors")
	assert.Contains(t, report.DisplayedErrors[1].Message, "network.allowed", "The next prioritized error should be the high-priority network error")
	assert.Equal(t, 1, report.HiddenCount, "One error should be hidden when limiting output")
	require.NotNil(t, report.RecoveryPlan, "Multi-error reports should include a recovery plan")
	assert.NotEmpty(t, report.RecoveryPlan.Steps, "Recovery plan should contain steps")
}

func TestExpandErrorMessages_SplitsBundledSchemaFailures(t *testing.T) {
	messages := ExpandErrorMessages(errors.New(`/tmp/workflow.md:9:5: error: Multiple schema validation failures:
- 'tools/github' (line 9, col 5): Unknown property: foo
- 'on' (line 11, col 3): Unknown property: pull-request
 9 | foo: bar`))

	require.Len(t, messages, 2, "Bundled schema failures should be split into separate display messages")
	assert.Contains(t, messages[0], "tools/github", "The tool schema failure should be preserved")
	assert.Contains(t, messages[1], "pull-request", "The event schema failure should be preserved")
}

func TestBuildPrioritizedErrorReportFromMessages_ShowAll(t *testing.T) {
	messages := []string{
		"workflow.md:1:1: error: invalid engine value 'copiliot'",
		"workflow.md:2:1: error: runtime version conflict",
		"workflow.md:3:1: error: deprecated field",
	}

	report := BuildPrioritizedErrorReportFromMessages(messages, true)

	require.Len(t, report.DisplayedErrors, 3, "Show-all reports should include every prioritized error")
	assert.Zero(t, report.HiddenCount, "No errors should be hidden in show-all mode")
}

func TestBuildPrioritizedErrorReportFromMessages_SuppressesCascadingSyntaxErrors(t *testing.T) {
	messages := []string{
		"workflow.md:2:1: error: failed to parse YAML frontmatter: mapping values are not allowed in this context",
		"[2026-01-01T00:00:00Z] Validation failed for field 'engine'\nReason: cannot be empty",
	}

	report := BuildPrioritizedErrorReportFromMessages(messages, true)

	require.Len(t, report.DisplayedErrors, 1, "The syntax root cause should suppress cascading configuration errors")
	assert.Equal(t, SeverityCritical, report.DisplayedErrors[0].Severity, "The remaining error should be critical")
	assert.Equal(t, 1, report.SuppressedCount, "One cascading error should be suppressed")
}

func TestNewValidationError_ClassifiesSeverity(t *testing.T) {
	err := NewValidationError("network.allowed", "example.com", "requires strict mode", "Enable strict mode")

	require.NotNil(t, err, "Validation error should be created")
	assert.Equal(t, SeverityHigh, err.Severity, "Network strict-mode errors should be high priority")
	assert.Equal(t, "permissions", err.Category, "Network strict-mode errors should be categorized as permissions")
}

func TestBuildPrioritizedErrorReportFromMessages_DuplicateKeyErrorsGetSpecificSuggestion(t *testing.T) {
	message := `/tmp/smoke-antigravity-duplicate-engine.md:10:1: error: mapping key "engine" already defined at [7:1]
   7 | engine:
   8 |   id: antigravity
   9 | strict: true
> 10 | engine:
       ^
  11 |   id: copilot`

	report := BuildPrioritizedErrorReportFromMessages([]string{message}, true)

	require.Len(t, report.DisplayedErrors, 1)
	prioritized := report.DisplayedErrors[0]
	assert.Equal(t, SeverityCritical, prioritized.Severity)
	assert.Equal(t, "syntax", prioritized.Category)
	assert.Contains(t, prioritized.Message, "> 10 | engine:")
	assert.Equal(t, "Remove the duplicate engine key at line 10; the first definition is at line 7.", prioritized.Suggestion)
}

func TestBuildPrioritizedErrorReportFromMessages_DuplicateKeySuggestionPreservesOriginalKeyCasing(t *testing.T) {
	message := `/tmp/smoke-antigravity-duplicate-engine.md:10:1: error: Mapping key "Engine" already defined at [7:1]`

	report := BuildPrioritizedErrorReportFromMessages([]string{message}, true)

	require.Len(t, report.DisplayedErrors, 1)
	assert.Equal(t, "Remove the duplicate Engine key at line 10; the first definition is at line 7.", report.DisplayedErrors[0].Suggestion)
}

func TestBuildPrioritizedErrorReportFromMessages_UnknownPermissionScopesUsePermissionHint(t *testing.T) {
	message := `/tmp/spec-enforcer-invalid-permission.md:5:3: error: Unknown property: unknown-scope (Valid permission scopes: actions, all, attestations, checks, copilot-requests, contents, deployments, discussions, id-token, issues, metadata, models, organization-projects, packages, pages, pull-requests, repository-projects, security-events, statuses, vulnerability-alerts)
2 | on:
3 |   schedule: daily
4 | permissions:
5 |   unknown-scope: read
      ^^^^^^^^^^^^^
6 | engine:
7 |   id: claude`

	report := BuildPrioritizedErrorReportFromMessages([]string{message}, true)

	require.Len(t, report.DisplayedErrors, 1)
	prioritized := report.DisplayedErrors[0]
	assert.Equal(t, SeverityMedium, prioritized.Severity)
	assert.Equal(t, "permissions", prioritized.Category)
	assert.Contains(t, prioritized.Message, "5 |   unknown-scope: read")
	assert.Equal(t, "Remove or replace the unknown permission scope `unknown-scope` in `permissions:` and re-run `gh aw compile`.", prioritized.Suggestion)
	assert.NotContains(t, prioritized.Suggestion, "event or filter")
}

func TestBuildPrioritizedErrorReportFromMessages_UnknownPermissionScopeHintPreservesOriginalScopeCasing(t *testing.T) {
	message := `/tmp/spec-enforcer-invalid-permission.md:5:3: error: unknown permission scope "Contents"
5 | permissions:
6 |   Contents: read`

	report := BuildPrioritizedErrorReportFromMessages([]string{message}, true)

	require.Len(t, report.DisplayedErrors, 1)
	assert.Equal(t, "Remove or replace the unknown permission scope `Contents` in `permissions:` and re-run `gh aw compile`.", report.DisplayedErrors[0].Suggestion)
}
