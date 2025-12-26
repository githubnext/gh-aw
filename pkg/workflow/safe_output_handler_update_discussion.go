package workflow

import (
	"github.com/githubnext/gh-aw/pkg/logger"
)

var updateDiscussionHandlerLog = logger.New("workflow:safe_output_handler_update_discussion")

// UpdateDiscussionHandler handles update_discussion safe output messages
type UpdateDiscussionHandler struct{}

// NewUpdateDiscussionHandler creates a new update_discussion handler
func NewUpdateDiscussionHandler() *UpdateDiscussionHandler {
	return &UpdateDiscussionHandler{}
}

// GetType returns the type identifier for this handler
func (h *UpdateDiscussionHandler) GetType() string {
	return "update_discussion"
}

// IsEnabled checks if update_discussion is enabled in the workflow configuration
func (h *UpdateDiscussionHandler) IsEnabled(data *WorkflowData) bool {
	return data.SafeOutputs != nil && data.SafeOutputs.UpdateDiscussions != nil
}

// BuildStepConfig builds the step configuration for update_discussion
func (h *UpdateDiscussionHandler) BuildStepConfig(c *Compiler, data *WorkflowData, ctx *SafeOutputContext) *SafeOutputStepConfig {
	if !h.IsEnabled(data) {
		return nil
	}

	updateDiscussionHandlerLog.Print("Building update_discussion step config")

	// Build the step configuration using the existing builder
	stepConfig := c.buildUpdateDiscussionStepConfig(data, ctx.MainJobName, ctx.ThreatDetectionEnabled)

	// Add temporary ID map if available
	if ctx.TempIDMapAvailable && h.RequiresTempIDMap() {
		stepConfig.CustomEnvVars = append(stepConfig.CustomEnvVars,
			"          GH_AW_TEMPORARY_ID_MAP: ${{ steps."+ctx.TempIDMapSource+".outputs.temporary_id_map }}\n")
		updateDiscussionHandlerLog.Printf("Added temporary ID map reference from step: %s", ctx.TempIDMapSource)
	}

	return &stepConfig
}

// GetOutputs returns the outputs that update_discussion produces
func (h *UpdateDiscussionHandler) GetOutputs() map[string]string {
	return map[string]string{
		"update_discussion_discussion_number": "${{ steps.update_discussion.outputs.discussion_number }}",
		"update_discussion_discussion_url":    "${{ steps.update_discussion.outputs.discussion_url }}",
	}
}

// RequiresTempIDMap returns true if this handler needs access to the temporary ID map
func (h *UpdateDiscussionHandler) RequiresTempIDMap() bool {
	return true // update_discussion may reference temporary IDs from create_issue
}
