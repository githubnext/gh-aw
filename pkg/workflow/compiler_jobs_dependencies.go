package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
)

// This file contains job dependency management functions for the compiler.
// These functions handle dependency resolution and job orchestration logic,
// determining which jobs need to run and in what order.

// isActivationJobNeeded determines if the activation job is required for the workflow.
// The activation job is always needed to perform the timestamp check.
// It also handles:
// 1. Command is configured (for team member checking)
// 2. Text output is needed (for compute-text action)
// 3. If condition is specified (to handle runtime conditions)
// 4. Permission checks are needed (consolidated team member validation)
func (c *Compiler) isActivationJobNeeded() bool {
	return true
}

// referencesCustomJobOutputs checks if a condition string references custom jobs.
// Returns true if the condition contains "needs.<customJobName>." patterns, which includes
// both outputs (needs.job.outputs.*) and results (needs.job.result).
func (c *Compiler) referencesCustomJobOutputs(condition string, customJobs map[string]any) bool {
	if condition == "" || customJobs == nil {
		return false
	}
	for jobName := range customJobs {
		// Check for patterns like "needs.ast_grep.outputs" or "needs.ast_grep.result"
		if strings.Contains(condition, fmt.Sprintf("needs.%s.", jobName)) {
			return true
		}
	}
	return false
}

// jobDependsOnPreActivation checks if a job config has pre_activation as a dependency.
func jobDependsOnPreActivation(jobConfig map[string]any) bool {
	if needs, hasNeeds := jobConfig["needs"]; hasNeeds {
		if needsList, ok := needs.([]any); ok {
			for _, need := range needsList {
				if needStr, ok := need.(string); ok && needStr == constants.PreActivationJobName {
					return true
				}
			}
		} else if needStr, ok := needs.(string); ok && needStr == constants.PreActivationJobName {
			return true
		}
	}
	return false
}

// jobDependsOnAgent checks if a job config has agent as a dependency.
// Jobs that depend on agent should run AFTER the agent job, not before it.
// The jobConfig parameter is expected to be a map representing the job's YAML configuration,
// where "needs" can be either a string (single dependency) or []any (multiple dependencies).
// Returns false if "needs" is missing, malformed, or doesn't contain the agent job.
func jobDependsOnAgent(jobConfig map[string]any) bool {
	if needs, hasNeeds := jobConfig["needs"]; hasNeeds {
		if needsList, ok := needs.([]any); ok {
			for _, need := range needsList {
				if needStr, ok := need.(string); ok && needStr == constants.AgentJobName {
					return true
				}
			}
		} else if needStr, ok := needs.(string); ok && needStr == constants.AgentJobName {
			return true
		}
	}
	return false
}

// getCustomJobsDependingOnPreActivation returns custom job names that explicitly depend on pre_activation.
// These jobs run after pre_activation but before activation, and activation should depend on them.
func (c *Compiler) getCustomJobsDependingOnPreActivation(customJobs map[string]any) []string {
	var jobNames []string
	for jobName, jobConfig := range customJobs {
		if configMap, ok := jobConfig.(map[string]any); ok {
			if jobDependsOnPreActivation(configMap) {
				jobNames = append(jobNames, jobName)
			}
		}
	}
	return jobNames
}

// getReferencedCustomJobs returns custom job names that are referenced in the given content.
// It looks for patterns like "needs.<jobName>." or "${{ needs.<jobName>." in the content.
func (c *Compiler) getReferencedCustomJobs(content string, customJobs map[string]any) []string {
	if content == "" || customJobs == nil {
		return nil
	}
	var referencedJobs []string
	for jobName := range customJobs {
		// Check for patterns like "needs.job_name." which covers:
		// - needs.job_name.outputs.X
		// - ${{ needs.job_name.outputs.X }}
		// - needs.job_name.result
		if strings.Contains(content, fmt.Sprintf("needs.%s.", jobName)) {
			referencedJobs = append(referencedJobs, jobName)
		}
	}
	return referencedJobs
}
