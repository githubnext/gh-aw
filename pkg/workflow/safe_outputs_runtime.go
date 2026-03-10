package workflow

import "github.com/github/gh-aw/pkg/constants"

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
		return "runs-on: " + constants.DefaultActivationJobRunnerImage
	}

	return "runs-on: " + safeOutputs.RunsOn
}

// formatDetectionRunsOn resolves the runner for the detection job using the following priority:
// 1. safe-outputs.detection.runs-on (detection-specific override)
// 2. agentRunsOn (the agent job's runner, passed by the caller)
func (c *Compiler) formatDetectionRunsOn(safeOutputs *SafeOutputsConfig, agentRunsOn string) string {
	if safeOutputs != nil && safeOutputs.ThreatDetection != nil && safeOutputs.ThreatDetection.RunsOn != "" {
		return "runs-on: " + safeOutputs.ThreatDetection.RunsOn
	}
	return agentRunsOn
}

// usesPatchesAndCheckouts checks if the workflow uses safe outputs that require
// git patches and checkouts (create-pull-request or push-to-pull-request-branch)
func usesPatchesAndCheckouts(safeOutputs *SafeOutputsConfig) bool {
	if safeOutputs == nil {
		return false
	}
	return safeOutputs.CreatePullRequests != nil || safeOutputs.PushToPullRequestBranch != nil
}
