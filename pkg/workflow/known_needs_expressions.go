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
// The function generates mappings for:
// - needs.activation.outputs.* (text, title, body, comment_id, comment_repo, issue_locked)
// - needs.pre_activation.outputs.* (activated, matched_command)
// - needs.detection.outputs.* (success)
// - needs.agent.outputs.* (output, output_types, has_patch, secret_verification_result, checkout_pr_success)
// - needs.<safe-output-job>.outputs.* for all safe output jobs
// - needs.<custom-job>.outputs.* for all custom jobs
//
// Returns a slice of ExpressionMapping that should be merged with other expression mappings.
func generateKnownNeedsExpressions(data *WorkflowData) []*ExpressionMapping {
	knownNeedsLog.Print("Generating known needs.* expressions")

	var mappings []*ExpressionMapping

	// Activation job outputs
	activationOutputs := []string{
		"text",
		"title",
		"body",
		"comment_id",
		"comment_repo",
		"issue_locked",
	}
	for _, output := range activationOutputs {
		expr := fmt.Sprintf("needs.%s.outputs.%s", constants.ActivationJobName, output)
		envVar := fmt.Sprintf("GH_AW_NEEDS_%s_OUTPUTS_%s",
			normalizeJobNameForEnvVar(string(constants.ActivationJobName)),
			normalizeOutputNameForEnvVar(output))
		mappings = append(mappings, &ExpressionMapping{
			Original: fmt.Sprintf("${{ %s }}", expr),
			EnvVar:   envVar,
			Content:  expr,
		})
	}

	// Pre-activation job outputs
	preActivationOutputs := []string{
		constants.ActivatedOutput,
		constants.MatchedCommandOutput,
	}
	for _, output := range preActivationOutputs {
		expr := fmt.Sprintf("needs.%s.outputs.%s", constants.PreActivationJobName, output)
		envVar := fmt.Sprintf("GH_AW_NEEDS_%s_OUTPUTS_%s",
			normalizeJobNameForEnvVar(string(constants.PreActivationJobName)),
			normalizeOutputNameForEnvVar(output))
		mappings = append(mappings, &ExpressionMapping{
			Original: fmt.Sprintf("${{ %s }}", expr),
			EnvVar:   envVar,
			Content:  expr,
		})
	}

	// Detection job outputs
	detectionOutputs := []string{
		"success",
	}
	for _, output := range detectionOutputs {
		expr := fmt.Sprintf("needs.%s.outputs.%s", constants.DetectionJobName, output)
		envVar := fmt.Sprintf("GH_AW_NEEDS_%s_OUTPUTS_%s",
			normalizeJobNameForEnvVar(string(constants.DetectionJobName)),
			normalizeOutputNameForEnvVar(output))
		mappings = append(mappings, &ExpressionMapping{
			Original: fmt.Sprintf("${{ %s }}", expr),
			EnvVar:   envVar,
			Content:  expr,
		})
	}

	// Agent job outputs
	agentOutputs := []string{
		"output",
		"output_types",
		"has_patch",
		"secret_verification_result",
		"checkout_pr_success",
	}
	for _, output := range agentOutputs {
		expr := fmt.Sprintf("needs.%s.outputs.%s", constants.AgentJobName, output)
		envVar := fmt.Sprintf("GH_AW_NEEDS_%s_OUTPUTS_%s",
			normalizeJobNameForEnvVar(string(constants.AgentJobName)),
			normalizeOutputNameForEnvVar(output))
		mappings = append(mappings, &ExpressionMapping{
			Original: fmt.Sprintf("${{ %s }}", expr),
			EnvVar:   envVar,
			Content:  expr,
		})
	}

	// Safe output job outputs (if safe outputs are enabled)
	if data.SafeOutputs != nil {
		// Common safe output job outputs (these are standard across all safe output types)
		safeOutputJobTypes := getSafeOutputJobNames(data)
		for _, jobName := range safeOutputJobTypes {
			// Each safe output job can have various outputs depending on type
			// Add the most common ones that are referenced
			commonOutputs := []string{
				"issue_url",
				"issue_number",
				"discussion_url",
				"discussion_number",
				"pull_request_url",
				"pull_request_number",
				"comment_url",
				"comment_id",
				"temporary_id_map",
			}
			for _, output := range commonOutputs {
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

	// Custom job outputs from frontmatter jobs
	if data.Jobs != nil {
		customJobNames := getCustomJobNames(data)
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

// getCustomJobNames returns a list of custom job names from frontmatter
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
