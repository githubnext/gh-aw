package workflow

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var createDiscussionHandlerLog = logger.New("workflow:safe_output_handler_create_discussion")

// CreateDiscussionHandler handles create_discussion safe output messages
type CreateDiscussionHandler struct{}

// NewCreateDiscussionHandler creates a new create_discussion handler
func NewCreateDiscussionHandler() *CreateDiscussionHandler {
	return &CreateDiscussionHandler{}
}

// GetType returns the type identifier for this handler
func (h *CreateDiscussionHandler) GetType() string {
	return "create_discussion"
}

// IsEnabled checks if create_discussion is enabled in the workflow configuration
func (h *CreateDiscussionHandler) IsEnabled(data *WorkflowData) bool {
	return data.SafeOutputs != nil && data.SafeOutputs.CreateDiscussions != nil
}

// BuildStepConfig builds the step configuration for create_discussion
func (h *CreateDiscussionHandler) BuildStepConfig(c *Compiler, data *WorkflowData, ctx *SafeOutputContext) *SafeOutputStepConfig {
	if !h.IsEnabled(data) {
		return nil
	}

	createDiscussionHandlerLog.Printf("Building create_discussion step config for workflow: %s", data.Name)

	// Build custom environment variables specific to create-discussion
	var customEnvVars []string
	customEnvVars = append(customEnvVars, buildTitlePrefixEnvVar("GH_AW_DISCUSSION_TITLE_PREFIX", data.SafeOutputs.CreateDiscussions.TitlePrefix)...)
	customEnvVars = append(customEnvVars, buildCategoryEnvVar("GH_AW_DISCUSSION_CATEGORY", data.SafeOutputs.CreateDiscussions.Category)...)
	customEnvVars = append(customEnvVars, buildLabelsEnvVar("GH_AW_DISCUSSION_LABELS", data.SafeOutputs.CreateDiscussions.Labels)...)
	customEnvVars = append(customEnvVars, buildLabelsEnvVar("GH_AW_DISCUSSION_ALLOWED_LABELS", data.SafeOutputs.CreateDiscussions.AllowedLabels)...)
	customEnvVars = append(customEnvVars, buildAllowedReposEnvVar("GH_AW_ALLOWED_REPOS", data.SafeOutputs.CreateDiscussions.AllowedRepos)...)

	// Add close-older-discussions flag if enabled
	if data.SafeOutputs.CreateDiscussions.CloseOlderDiscussions {
		customEnvVars = append(customEnvVars, "          GH_AW_CLOSE_OLDER_DISCUSSIONS: \"true\"\n")
	}

	// Add expires value if set
	if data.SafeOutputs.CreateDiscussions.Expires > 0 {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_DISCUSSION_EXPIRES: \"%d\"\n", data.SafeOutputs.CreateDiscussions.Expires))
	}

	// Add temporary ID map from create_issue if available
	if ctx.TempIDMapAvailable {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_TEMPORARY_ID_MAP: ${{ steps.%s.outputs.temporary_id_map }}\n", ctx.TempIDMapSource))
		createDiscussionHandlerLog.Printf("Added temporary ID map reference from step: %s", ctx.TempIDMapSource)
	}

	// Build step condition
	condition := BuildSafeOutputType("create_discussion")
	if ctx.ThreatDetectionEnabled {
		condition = BuildAnd(condition, buildDetectionSuccessCondition())
	}

	// Create outputs map
	outputs := map[string]string{
		"discussion_number": "${{ steps.create_discussion.outputs.discussion_number }}",
		"discussion_url":    "${{ steps.create_discussion.outputs.discussion_url }}",
	}

	return &SafeOutputStepConfig{
		StepName:        "Create Discussion",
		StepID:          "create_discussion",
		ScriptName:      "create_discussion",
		CustomEnvVars:   customEnvVars,
		Condition:       condition,
		Token:           data.SafeOutputs.CreateDiscussions.GitHubToken,
		UseCopilotToken: false,
		UseAgentToken:   false,
		PreSteps:        nil,
		PostSteps:       nil,
		Outputs:         outputs,
	}
}

// GetOutputs returns the outputs that create_discussion produces
func (h *CreateDiscussionHandler) GetOutputs() map[string]string {
	return map[string]string{
		"create_discussion_discussion_number": "${{ steps.create_discussion.outputs.discussion_number }}",
		"create_discussion_discussion_url":    "${{ steps.create_discussion.outputs.discussion_url }}",
	}
}

// RequiresTempIDMap returns true if this handler needs access to the temporary ID map
func (h *CreateDiscussionHandler) RequiresTempIDMap() bool {
	return true // create_discussion consumes the temporary ID map from create_issue
}
