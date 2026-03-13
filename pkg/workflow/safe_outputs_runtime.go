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

// usesPatchesAndCheckouts checks if the workflow uses safe outputs that require
// git patches and checkouts (create-pull-request or push-to-pull-request-branch)
func usesPatchesAndCheckouts(safeOutputs *SafeOutputsConfig) bool {
	if safeOutputs == nil {
		return false
	}
	result := safeOutputs.CreatePullRequests != nil || safeOutputs.PushToPullRequestBranch != nil
	safeOutputsRuntimeLog.Printf("usesPatchesAndCheckouts: createPR=%v, pushToPRBranch=%v, result=%v",
		safeOutputs.CreatePullRequests != nil, safeOutputs.PushToPullRequestBranch != nil, result)
	return result
}
