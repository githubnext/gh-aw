package workflow

import (
	"github.com/githubnext/gh-aw/pkg/logger"
)

var closeDiscussionHandlerLog = logger.New("workflow:safe_output_handler_close_discussion")

// CloseDiscussionHandler handles close_discussion safe output messages
type CloseDiscussionHandler struct{}

// NewCloseDiscussionHandler creates a new close_discussion handler
func NewCloseDiscussionHandler() *CloseDiscussionHandler {
	return &CloseDiscussionHandler{}
}

// GetType returns the type identifier for this handler
func (h *CloseDiscussionHandler) GetType() string {
	return "close_discussion"
}

// IsEnabled checks if close_discussion is enabled in the workflow configuration
func (h *CloseDiscussionHandler) IsEnabled(data *WorkflowData) bool {
	return data.SafeOutputs != nil && data.SafeOutputs.CloseDiscussions != nil
}

// BuildStepConfig builds the step configuration for close_discussion
func (h *CloseDiscussionHandler) BuildStepConfig(c *Compiler, data *WorkflowData, ctx *SafeOutputContext) *SafeOutputStepConfig {
	if !h.IsEnabled(data) {
		return nil
	}

	closeDiscussionHandlerLog.Print("Building close_discussion step config")

	// Build the step configuration using the existing builder
	stepConfig := c.buildCloseDiscussionStepConfig(data, ctx.MainJobName, ctx.ThreatDetectionEnabled)

	// Add temporary ID map if available
	if ctx.TempIDMapAvailable && h.RequiresTempIDMap() {
		stepConfig.CustomEnvVars = append(stepConfig.CustomEnvVars,
			"          GH_AW_TEMPORARY_ID_MAP: ${{ steps."+ctx.TempIDMapSource+".outputs.temporary_id_map }}\n")
		closeDiscussionHandlerLog.Printf("Added temporary ID map reference from step: %s", ctx.TempIDMapSource)
	}

	return &stepConfig
}

// GetOutputs returns the outputs that close_discussion produces
func (h *CloseDiscussionHandler) GetOutputs() map[string]string {
	// close_discussion doesn't produce outputs in the current implementation
	return map[string]string{}
}

// RequiresTempIDMap returns true if this handler needs access to the temporary ID map
func (h *CloseDiscussionHandler) RequiresTempIDMap() bool {
	return true // close_discussion may reference temporary IDs from create_issue
}
