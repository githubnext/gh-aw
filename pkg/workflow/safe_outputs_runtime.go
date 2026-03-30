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

// formatSafeOutputsRunsOn formats the runs-on value from SafeOutputsConfig for job output.
// Falls back to the default activation job runner image when not explicitly set.
func (c *Compiler) formatSafeOutputsRunsOn(safeOutputs *SafeOutputsConfig) string {
	if safeOutputs == nil || safeOutputs.RunsOn == "" {
		safeOutputsRuntimeLog.Printf("Safe outputs runs-on not set, using default: %s", constants.DefaultActivationJobRunnerImage)
		return "runs-on: " + constants.DefaultActivationJobRunnerImage
	}

	safeOutputsRuntimeLog.Printf("Safe outputs runs-on: %s", safeOutputs.RunsOn)
	return "runs-on: " + safeOutputs.RunsOn
}

// formatFrameworkJobRunsOn returns the runs-on value for framework/generated jobs
// (activation, pre-activation, safe-outputs, unlock, APM, etc.).
//
// Precedence (highest to lowest):
//  1. safe-outputs.runs-on — explicit per-section override
//  2. runs-on-slim   — top-level field for all framework jobs
//  3. DefaultActivationJobRunnerImage — compiled-in default
func (c *Compiler) formatFrameworkJobRunsOn(data *WorkflowData) string {
	if data != nil && data.SafeOutputs != nil && data.SafeOutputs.RunsOn != "" {
		safeOutputsRuntimeLog.Printf("Framework job runs-on from safe-outputs: %s", data.SafeOutputs.RunsOn)
		return "runs-on: " + data.SafeOutputs.RunsOn
	}
	if data != nil && data.RunsOnSlim != "" {
		safeOutputsRuntimeLog.Printf("Framework job runs-on from runs-on-slim: %s", data.RunsOnSlim)
		return "runs-on: " + data.RunsOnSlim
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
