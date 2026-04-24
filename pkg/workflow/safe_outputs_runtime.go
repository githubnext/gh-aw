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

// formatDetectionJobRunsOn returns the runs-on value for the detection job.
//
// The detection job runs on a fresh runner separate from the agent job.
// Use safe-outputs.threat-detection.runs-on to override; otherwise ubuntu-latest is used.
// Notably, runs-on-slim and safe-outputs.runs-on do NOT propagate to the detection job
// because the detection job has different networking and compute requirements.
func (c *Compiler) formatDetectionJobRunsOn(data *WorkflowData) string {
	if data != nil && data.SafeOutputs != nil && data.SafeOutputs.ThreatDetection != nil &&
		data.SafeOutputs.ThreatDetection.RunsOn != "" {
		safeOutputsRuntimeLog.Printf("Detection job runs-on from threat-detection config: %s", data.SafeOutputs.ThreatDetection.RunsOn)
		return "runs-on: " + data.SafeOutputs.ThreatDetection.RunsOn
	}
	safeOutputsRuntimeLog.Printf("Detection job runs-on using default: ubuntu-latest")
	return "runs-on: ubuntu-latest"
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
