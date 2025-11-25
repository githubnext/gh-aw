package workflow

import (
	"fmt"
	"strings"
)

// ListSafeOutputJobParams encapsulates parameters for building list-based safe-output jobs.
// This enables deduplication of common patterns found in add_labels.go and add_reviewer.go.
type ListSafeOutputJobParams struct {
	// Job metadata
	JobName     string // e.g., "add_labels", "add_reviewer"
	StepName    string // e.g., "Add Labels", "Add Reviewers"
	StepID      string // e.g., "add_labels", "add_reviewer"
	MainJobName string

	// Environment variable prefix for this type (e.g., "LABELS", "REVIEWERS")
	EnvPrefix string

	// List of allowed items (labels, reviewers, etc.)
	AllowedItems []string

	// Maximum count (defaults to 3 if not specified)
	MaxCount int

	// Target configuration
	Target         string
	TargetRepoSlug string

	// Script to run
	Script string

	// Permissions for the job
	Permissions *Permissions

	// GitHub token override
	Token string

	// Context condition nodes to require when target is not specified
	// For add_labels: issue OR pull_request context
	// For add_reviewer: pull_request context only
	TriggeringContextConditions []ConditionNode

	// Output key name (e.g., "labels_added", "reviewers_added")
	OutputKey string
}

// buildListSafeOutputJob creates a safe-output job for list-based operations like
// adding labels or reviewers. This helper extracts the common boilerplate found in
// add_labels.go and add_reviewer.go.
func (c *Compiler) buildListSafeOutputJob(data *WorkflowData, params ListSafeOutputJobParams) (*Job, error) {
	// Set default max count if not specified
	maxCount := params.MaxCount
	if maxCount == 0 {
		maxCount = 3
	}

	// Build custom environment variables
	var customEnvVars []string

	// Pass the allowed list (empty string if no restrictions)
	allowedStr := strings.Join(params.AllowedItems, ",")
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_%s_ALLOWED: %q\n", params.EnvPrefix, allowedStr))

	// Pass the max limit
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_%s_MAX_COUNT: %d\n", params.EnvPrefix, maxCount))

	// Pass the target configuration if set
	if params.Target != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_%s_TARGET: %q\n", params.EnvPrefix, params.Target))
	}

	// Add standard environment variables (metadata + staged/target repo)
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, params.TargetRepoSlug)...)

	// Create outputs for the job
	outputs := map[string]string{
		params.OutputKey: fmt.Sprintf("${{ steps.%s.outputs.%s }}", params.StepID, params.OutputKey),
	}

	// Build job condition
	jobCondition := BuildSafeOutputType(params.JobName)

	// Add context conditions when target is not explicitly set
	if params.Target == "" && len(params.TriggeringContextConditions) > 0 {
		if len(params.TriggeringContextConditions) == 1 {
			jobCondition = buildAnd(jobCondition, params.TriggeringContextConditions[0])
		} else {
			// Multiple conditions are OR'd together
			orCondition := params.TriggeringContextConditions[0]
			for i := 1; i < len(params.TriggeringContextConditions); i++ {
				orCondition = buildOr(orCondition, params.TriggeringContextConditions[i])
			}
			jobCondition = buildAnd(jobCondition, orCondition)
		}
	}

	// Use the shared builder function to create the job
	return c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:        params.JobName,
		StepName:       params.StepName,
		StepID:         params.StepID,
		MainJobName:    params.MainJobName,
		CustomEnvVars:  customEnvVars,
		Script:         params.Script,
		Permissions:    params.Permissions,
		Outputs:        outputs,
		Condition:      jobCondition,
		Token:          params.Token,
		TargetRepoSlug: params.TargetRepoSlug,
	})
}
