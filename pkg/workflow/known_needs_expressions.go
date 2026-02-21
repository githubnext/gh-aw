package workflow

import (
	"fmt"
	"sort"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var knownNeedsLog = logger.New("workflow:known_needs")

// generateKnownNeedsExpressions generates expression mappings for all known needs.* variables
// that could be referenced in the prompt. This ensures that these variables are available
// for interpolation even if the markdown changes without recompilation.
//
// IMPORTANT: The prompt is generated in the ACTIVATION job, so it can only access outputs
// from jobs that the activation job depends on (i.e., jobs that run BEFORE activation).
// This typically includes:
// - needs.pre_activation.outputs.* (activated, matched_command) - only when pre_activation job exists
// - needs.<custom-job>.outputs.* for custom jobs that run before activation
//
// The function does NOT generate mappings for jobs that run AFTER activation:
// - needs.activation.outputs.* (activation is the current job)
// - needs.agent.outputs.* (agent runs AFTER activation)
// - needs.detection.outputs.* (detection runs AFTER activation)
// - needs.<safe-output-job>.outputs.* (these run AFTER agent)
//
// preActivationJobCreated indicates whether the pre_activation job was created for this workflow.
// When false, no pre_activation output mappings are generated to avoid actionlint errors.
//
// Returns a slice of ExpressionMapping that should be merged with other expression mappings.
func generateKnownNeedsExpressions(data *WorkflowData, preActivationJobCreated bool) []*ExpressionMapping {
	knownNeedsLog.Print("Generating known needs.* expressions for activation job")

	var mappings []*ExpressionMapping

	// Pre-activation job outputs (activation depends on pre_activation only when it exists)
	// Only generate these mappings when the pre_activation job was actually created;
	// otherwise referencing needs.pre_activation.outputs.* causes actionlint errors.
	if preActivationJobCreated {
		// Always include the "activated" output
		activatedExpr := fmt.Sprintf("needs.%s.outputs.%s", constants.PreActivationJobName, constants.ActivatedOutput)
		activatedEnvVar := fmt.Sprintf("GH_AW_NEEDS_%s_OUTPUTS_%s",
			normalizeJobNameForEnvVar(string(constants.PreActivationJobName)),
			normalizeOutputNameForEnvVar(constants.ActivatedOutput))
		mappings = append(mappings, &ExpressionMapping{
			Original: fmt.Sprintf("${{ %s }}", activatedExpr),
			EnvVar:   activatedEnvVar,
			Content:  activatedExpr,
		})

		// Only include "matched_command" when the workflow has a command trigger,
		// since matched_command is only declared in the pre_activation job outputs for command workflows.
		if len(data.Command) > 0 {
			matchedCmdExpr := fmt.Sprintf("needs.%s.outputs.%s", constants.PreActivationJobName, constants.MatchedCommandOutput)
			matchedCmdEnvVar := fmt.Sprintf("GH_AW_NEEDS_%s_OUTPUTS_%s",
				normalizeJobNameForEnvVar(string(constants.PreActivationJobName)),
				normalizeOutputNameForEnvVar(constants.MatchedCommandOutput))
			mappings = append(mappings, &ExpressionMapping{
				Original: fmt.Sprintf("${{ %s }}", matchedCmdExpr),
				EnvVar:   matchedCmdEnvVar,
				Content:  matchedCmdExpr,
			})
		}
	}

	// Custom job outputs from frontmatter jobs
	// Only include custom jobs that would run before activation
	// (i.e., jobs that don't depend on activation, pre_activation, agent, or detection)
	if data.Jobs != nil {
		customJobNames := getCustomJobsBeforeActivation(data)
		for _, jobName := range customJobNames {
			// For custom jobs, we can't know all possible outputs ahead of time
			// But we can add the most commonly used output name: "output"
			// Users can add more specific outputs if needed
			commonCustomOutputs := []string{
				"output",
			}
			for _, output := range commonCustomOutputs {
				expr := fmt.Sprintf("needs.%s.outputs.%s", jobName, output)
				envVar := fmt.Sprintf("GH_AW_NEEDS_%s_OUTPUTS_%s",
					normalizeJobNameForEnvVar(jobName),
					normalizeOutputNameForEnvVar(output))
				mappings = append(mappings, &ExpressionMapping{
					Original: fmt.Sprintf("${{ %s }}", expr),
					EnvVar:   envVar,
					Content:  expr,
				})
			}
		}
	}

	knownNeedsLog.Printf("Generated %d known needs.* expression mappings", len(mappings))
	return mappings
}

// normalizeJobNameForEnvVar converts a job name to a valid environment variable segment
// Examples: "activation" -> "ACTIVATION", "pre_activation" -> "PRE_ACTIVATION"
func normalizeJobNameForEnvVar(jobName string) string {
	// Already in the correct format for most job names
	// Just uppercase and replace hyphens with underscores
	result := ""
	for _, char := range jobName {
		if char == '-' {
			result += "_"
		} else if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '_' {
			if char >= 'a' && char <= 'z' {
				result += string(char - 32) // Convert to uppercase
			} else if char >= 'A' && char <= 'Z' {
				result += string(char)
			} else {
				result += string(char)
			}
		}
	}
	return result
}

// normalizeOutputNameForEnvVar converts an output name to a valid environment variable segment
// Examples: "text" -> "TEXT", "comment_id" -> "COMMENT_ID"
func normalizeOutputNameForEnvVar(outputName string) string {
	return normalizeJobNameForEnvVar(outputName)
}

// getSafeOutputJobNames returns a list of safe output job names based on the configuration
func getSafeOutputJobNames(data *WorkflowData) []string {
	var jobNames []string

	if data.SafeOutputs == nil {
		return jobNames
	}

	// These are the standard safe output job names that can be generated
	if data.SafeOutputs.CreateIssues != nil {
		jobNames = append(jobNames, "create_issue")
	}
	if data.SafeOutputs.CreateDiscussions != nil {
		jobNames = append(jobNames, "create_discussion")
	}
	if data.SafeOutputs.AddComments != nil {
		jobNames = append(jobNames, "add_comment")
	}
	if data.SafeOutputs.CreatePullRequests != nil {
		jobNames = append(jobNames, "create_pull_request")
	}
	// Add the consolidated safe outputs job if it exists
	// This is always named "safe_outputs" when multiple types are configured
	if hasMultipleSafeOutputTypes(data.SafeOutputs) {
		jobNames = append(jobNames, "safe_outputs")
	}

	// Also add custom safe-job names from safe-jobs configuration
	if data.SafeOutputs.Jobs != nil {
		for jobName := range data.SafeOutputs.Jobs {
			jobNames = append(jobNames, jobName)
		}
	}

	// Sort for consistent output
	sort.Strings(jobNames)

	return jobNames
}

// hasMultipleSafeOutputTypes checks if multiple safe output types are configured
func hasMultipleSafeOutputTypes(config *SafeOutputsConfig) bool {
	count := 0
	if config.CreateIssues != nil {
		count++
	}
	if config.CreateDiscussions != nil {
		count++
	}
	if config.AddComments != nil {
		count++
	}
	if config.CreatePullRequests != nil {
		count++
	}
	return count > 1
}

// getCustomJobsBeforeActivation returns a list of custom job names that run before the activation job
// A custom job runs before activation ONLY if it explicitly depends on pre_activation
// Note: Jobs without explicit 'needs' will automatically get 'needs: activation' added by the compiler,
// so they run AFTER activation, not before. Only jobs that explicitly depend on pre_activation run before activation.
func getCustomJobsBeforeActivation(data *WorkflowData) []string {
	var jobNames []string

	if data.Jobs == nil {
		return jobNames
	}

	// Extract job names that explicitly depend on pre_activation
	for jobName, jobConfig := range data.Jobs {
		jobMap, ok := jobConfig.(map[string]any)
		if !ok {
			continue
		}

		// Check if the job explicitly depends on pre_activation
		// Jobs without explicit needs will get 'needs: activation' added automatically,
		// so they run AFTER activation
		needsField, hasNeeds := jobMap["needs"]
		if !hasNeeds {
			// No explicit dependencies - this will get needs: activation added automatically
			// So it runs AFTER activation, not before
			continue
		}

		// Parse the needs field (can be string or array)
		needsList := parseNeedsField(needsField)

		// Check if it depends on pre_activation
		dependsOnPreActivation := false
		for _, dep := range needsList {
			if dep == string(constants.PreActivationJobName) {
				dependsOnPreActivation = true
				break
			}
		}

		// Only include if it depends on pre_activation (and not on activation/agent/detection)
		if dependsOnPreActivation {
			// Double-check it doesn't also depend on activation-related jobs
			hasActivationDependency := false
			for _, dep := range needsList {
				if dep == string(constants.ActivationJobName) ||
					dep == string(constants.AgentJobName) ||
					dep == string(constants.DetectionJobName) {
					hasActivationDependency = true
					break
				}
			}

			if !hasActivationDependency {
				jobNames = append(jobNames, jobName)
			}
		}
	}

	// Sort for consistent output
	sort.Strings(jobNames)

	return jobNames
}

// parseNeedsField parses the needs field from a job configuration
// The needs field can be a string (single dependency) or an array of strings
func parseNeedsField(needsField any) []string {
	switch v := needsField.(type) {
	case string:
		return []string{v}
	case []string:
		return v
	case []any:
		var result []string
		for _, item := range v {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	default:
		return []string{}
	}
}

// getCustomJobNames returns a list of all custom job names from frontmatter
func getCustomJobNames(data *WorkflowData) []string {
	var jobNames []string

	if data.Jobs == nil {
		return jobNames
	}

	// Extract job names from the Jobs map
	for jobName := range data.Jobs {
		jobNames = append(jobNames, jobName)
	}

	// Sort for consistent output
	sort.Strings(jobNames)

	return jobNames
}
