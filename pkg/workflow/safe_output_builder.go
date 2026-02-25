package workflow

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var safeOutputBuilderLog = logger.New("workflow:safe_output_builder")

// ======================================
// Generic Env Var Builders
// ======================================

// BuildTargetEnvVar builds a target environment variable line for safe-output jobs.
// envVarName should be the full env var name like "GH_AW_CLOSE_ISSUE_TARGET".
// Returns an empty slice if target is empty.
func BuildTargetEnvVar(envVarName string, target string) []string {
	if target == "" {
		return nil
	}
	return []string{fmt.Sprintf("          %s: %q\n", envVarName, target)}
}

// BuildRequiredLabelsEnvVar builds a required-labels environment variable line for safe-output jobs.
// envVarName should be the full env var name like "GH_AW_CLOSE_ISSUE_REQUIRED_LABELS".
// Returns an empty slice if requiredLabels is empty.
func BuildRequiredLabelsEnvVar(envVarName string, requiredLabels []string) []string {
	if len(requiredLabels) == 0 {
		return nil
	}
	labelsStr := strings.Join(requiredLabels, ",")
	return []string{fmt.Sprintf("          %s: %q\n", envVarName, labelsStr)}
}

// BuildRequiredTitlePrefixEnvVar builds a required-title-prefix environment variable line for safe-output jobs.
// envVarName should be the full env var name like "GH_AW_CLOSE_ISSUE_REQUIRED_TITLE_PREFIX".
// Returns an empty slice if requiredTitlePrefix is empty.
func BuildRequiredTitlePrefixEnvVar(envVarName string, requiredTitlePrefix string) []string {
	if requiredTitlePrefix == "" {
		return nil
	}
	return []string{fmt.Sprintf("          %s: %q\n", envVarName, requiredTitlePrefix)}
}

// BuildRequiredCategoryEnvVar builds a required-category environment variable line for discussion safe-output jobs.
// envVarName should be the full env var name like "GH_AW_CLOSE_DISCUSSION_REQUIRED_CATEGORY".
// Returns an empty slice if requiredCategory is empty.
func BuildRequiredCategoryEnvVar(envVarName string, requiredCategory string) []string {
	if requiredCategory == "" {
		return nil
	}
	return []string{fmt.Sprintf("          %s: %q\n", envVarName, requiredCategory)}
}

// BuildMaxCountEnvVar builds a max count environment variable line for safe-output jobs.
// envVarName should be the full env var name like "GH_AW_CLOSE_ISSUE_MAX_COUNT".
func BuildMaxCountEnvVar(envVarName string, maxCount int) []string {
	return []string{fmt.Sprintf("          %s: %d\n", envVarName, maxCount)}
}

// overrideEnvVarLine replaces the first env var line in lines that starts with keyPrefix
// with newLine. If no match is found, newLine is appended.
func overrideEnvVarLine(lines []string, keyPrefix string, newLine string) []string {
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, keyPrefix) {
			lines[i] = newLine
			return lines
		}
	}
	return append(lines, newLine)
}

// BuildAllowedListEnvVar builds an allowed list environment variable line for safe-output jobs.
// envVarName should be the full env var name like "GH_AW_LABELS_ALLOWED".
// Always outputs the env var, even when empty (empty string means "allow all").
func BuildAllowedListEnvVar(envVarName string, allowed []string) []string {
	allowedStr := strings.Join(allowed, ",")
	return []string{fmt.Sprintf("          %s: %q\n", envVarName, allowedStr)}
}

// ======================================
// Close Job Env Var Builders
// ======================================

// BuildCloseJobEnvVars builds common environment variables for close operations.
// prefix should be like "GH_AW_CLOSE_ISSUE" or "GH_AW_CLOSE_PR".
// Returns a slice of environment variable lines.
func BuildCloseJobEnvVars(prefix string, config CloseJobConfig) []string {
	var envVars []string

	// Add target
	envVars = append(envVars, BuildTargetEnvVar(prefix+"_TARGET", config.Target)...)

	// Add required labels
	envVars = append(envVars, BuildRequiredLabelsEnvVar(prefix+"_REQUIRED_LABELS", config.RequiredLabels)...)

	// Add required title prefix
	envVars = append(envVars, BuildRequiredTitlePrefixEnvVar(prefix+"_REQUIRED_TITLE_PREFIX", config.RequiredTitlePrefix)...)

	return envVars
}

// ======================================
// List Job Env Var Builders
// ======================================

// BuildListJobEnvVars builds common environment variables for list-based operations.
// prefix should be like "GH_AW_LABELS" or "GH_AW_REVIEWERS".
// Returns a slice of environment variable lines.
func BuildListJobEnvVars(prefix string, config ListJobConfig, maxCount int) []string {
	var envVars []string

	// Add allowed list
	envVars = append(envVars, BuildAllowedListEnvVar(prefix+"_ALLOWED", config.Allowed)...)

	// Add blocked list
	envVars = append(envVars, BuildAllowedListEnvVar(prefix+"_BLOCKED", config.Blocked)...)

	// Add max count
	envVars = append(envVars, BuildMaxCountEnvVar(prefix+"_MAX_COUNT", maxCount)...)

	// Add target
	envVars = append(envVars, BuildTargetEnvVar(prefix+"_TARGET", config.Target)...)

	return envVars
}

// ======================================
// List Job Builder Helpers
// ======================================

// ListJobBuilderConfig contains parameters for building list-based safe-output jobs
type ListJobBuilderConfig struct {
	JobName        string        // e.g., "add_labels", "assign_milestone"
	StepName       string        // e.g., "Add Labels", "Assign Milestone"
	StepID         string        // e.g., "add_labels", "assign_milestone"
	EnvPrefix      string        // e.g., "GH_AW_LABELS", "GH_AW_MILESTONE"
	OutputName     string        // e.g., "labels_added", "assigned_milestones"
	Script         string        // JavaScript script for the operation
	Permissions    *Permissions  // Job permissions
	DefaultMax     int           // Default max count if not specified in config
	ExtraCondition ConditionNode // Additional condition to append (optional)
}

// BuildListSafeOutputJob builds a list-based safe-output job using shared logic.
// This consolidates the common builder pattern used by add-labels, assign-milestone, and assign-to-user.
func (c *Compiler) BuildListSafeOutputJob(data *WorkflowData, mainJobName string, listJobConfig ListJobConfig, baseSafeOutputConfig BaseSafeOutputConfig, builderConfig ListJobBuilderConfig) (*Job, error) {
	safeOutputBuilderLog.Printf("Building list safe-output job: %s", builderConfig.JobName)

	// Handle max count with default – use literal integer if set, else fall back to DefaultMax
	maxCount := builderConfig.DefaultMax
	if n := templatableIntValue(baseSafeOutputConfig.Max); n > 0 {
		maxCount = n
	}
	safeOutputBuilderLog.Printf("Max count set to: %d", maxCount)

	// Build custom environment variables using shared helpers
	customEnvVars := BuildListJobEnvVars(builderConfig.EnvPrefix, listJobConfig, maxCount)

	// If max is a GitHub Actions expression, override with the expression value
	if baseSafeOutputConfig.Max != nil && templatableIntValue(baseSafeOutputConfig.Max) == 0 {
		exprLine := buildTemplatableIntEnvVar(builderConfig.EnvPrefix+"_MAX_COUNT", baseSafeOutputConfig.Max)
		if len(exprLine) > 0 {
			prefix := builderConfig.EnvPrefix + "_MAX_COUNT:"
			customEnvVars = overrideEnvVarLine(customEnvVars, prefix, exprLine[0])
		}
	}

	// Add standard environment variables (metadata + staged/target repo)
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, listJobConfig.TargetRepoSlug)...)

	// Create outputs for the job
	outputs := map[string]string{
		builderConfig.OutputName: fmt.Sprintf("${{ steps.%s.outputs.%s }}", builderConfig.StepID, builderConfig.OutputName),
	}

	// Build base job condition
	jobCondition := BuildSafeOutputType(builderConfig.JobName)

	// Add extra condition if provided
	if builderConfig.ExtraCondition != nil {
		jobCondition = BuildAnd(jobCondition, builderConfig.ExtraCondition)
	}

	// Use the shared builder function to create the job
	return c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:        builderConfig.JobName,
		StepName:       builderConfig.StepName,
		StepID:         builderConfig.StepID,
		MainJobName:    mainJobName,
		CustomEnvVars:  customEnvVars,
		Script:         builderConfig.Script,
		Permissions:    builderConfig.Permissions,
		Outputs:        outputs,
		Condition:      jobCondition,
		Token:          baseSafeOutputConfig.GitHubToken,
		TargetRepoSlug: listJobConfig.TargetRepoSlug,
	})
}
