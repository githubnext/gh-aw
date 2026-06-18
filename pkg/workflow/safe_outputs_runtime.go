package workflow

import (
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var safeOutputsRuntimeLog = logger.New("workflow:safe_outputs_runtime")

// ========================================
// Safe Output Runtime Configuration
// ========================================
//
// This file contains functions that determine the runtime environment
// (runner images) for safe-outputs jobs and detect feature usage patterns
// that affect job configuration.

// formatFrameworkJobRunsOn returns the runs-on value for framework/generated jobs
// (activation, pre-activation, safe-outputs, unlock, APM, etc.).
//
// Precedence (highest to lowest):
//  1. safe-outputs.runs-on — explicit per-section override
//  2. runs-on-slim   — top-level field for all framework jobs
//  3. DefaultActivationJobRunnerImage — compiled-in default
func (c *Compiler) formatFrameworkJobRunsOn(data *WorkflowData) string {
	if data != nil && data.SafeOutputs != nil && data.SafeOutputs.RunsOn != "" {
		snippet := normalizeRunsOnSnippet(data.SafeOutputs.RunsOn)
		safeOutputsRuntimeLog.Printf("Framework job runs-on from safe-outputs: %s", snippet)
		return c.indentYAMLLines(snippet, "    ")
	}
	if data != nil && data.RunsOnSlim != "" {
		snippet := normalizeRunsOnSnippet(data.RunsOnSlim)
		safeOutputsRuntimeLog.Printf("Framework job runs-on from runs-on-slim: %s", snippet)
		return c.indentYAMLLines(snippet, "    ")
	}
	safeOutputsRuntimeLog.Printf("Framework job runs-on using default: %s", constants.DefaultActivationJobRunnerImage)
	return "runs-on: " + constants.DefaultActivationJobRunnerImage
}

// usesPatchesAndCheckouts checks if the workflow uses safe outputs that require
// git patches and checkouts (create-pull-request or push-to-pull-request-branch).
// Staged handlers are excluded because they only emit preview output and do not
// perform real git operations or API calls.
func usesPatchesAndCheckouts(safeOutputs *SafeOutputsConfig) bool {
	if safeOutputs == nil {
		return false
	}
	createPRNeedsCheckout := safeOutputs.CreatePullRequests != nil && !isHandlerStaged(safeOutputs.Staged, safeOutputs.CreatePullRequests.Staged)
	pushToPRNeedsCheckout := safeOutputs.PushToPullRequestBranch != nil && !isHandlerStaged(safeOutputs.Staged, safeOutputs.PushToPullRequestBranch.Staged)
	result := createPRNeedsCheckout || pushToPRNeedsCheckout
	safeOutputsRuntimeLog.Printf("usesPatchesAndCheckouts: createPR=%v(needsCheckout=%v), pushToPRBranch=%v(needsCheckout=%v), result=%v",
		safeOutputs.CreatePullRequests != nil, createPRNeedsCheckout,
		safeOutputs.PushToPullRequestBranch != nil, pushToPRNeedsCheckout,
		result)
	return result
}

// buildPRCheckoutCondition builds the `if:` condition gating the safe_outputs job's
// checkout and git-configuration steps. The steps should run only when a create_pull_request
// or push_to_pull_request_branch output will actually be processed, so the condition is the
// OR of whichever of those two safe outputs are configured. Callers should only invoke this
// when at least one of the two is configured (the default branch assumes push_to_pull_request_branch).
func buildPRCheckoutCondition(safeOutputs *SafeOutputsConfig) ConditionNode {
	switch {
	case safeOutputs.CreatePullRequests != nil && safeOutputs.PushToPullRequestBranch != nil:
		return BuildOr(
			BuildSafeOutputType("create_pull_request"),
			BuildSafeOutputType("push_to_pull_request_branch"),
		)
	case safeOutputs.CreatePullRequests != nil:
		return BuildSafeOutputType("create_pull_request")
	default:
		return BuildSafeOutputType("push_to_pull_request_branch")
	}
}
