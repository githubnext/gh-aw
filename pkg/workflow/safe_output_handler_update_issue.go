package workflow

import (
	"github.com/githubnext/gh-aw/pkg/logger"
)

var updateIssueHandlerLog = logger.New("workflow:safe_output_handler_update_issue")

// UpdateIssueHandler handles update_issue safe output messages
type UpdateIssueHandler struct{}

// NewUpdateIssueHandler creates a new update_issue handler
func NewUpdateIssueHandler() *UpdateIssueHandler {
	return &UpdateIssueHandler{}
}

// GetType returns the type identifier for this handler
func (h *UpdateIssueHandler) GetType() string {
	return "update_issue"
}

// IsEnabled checks if update_issue is enabled in the workflow configuration
func (h *UpdateIssueHandler) IsEnabled(data *WorkflowData) bool {
	return data.SafeOutputs != nil && data.SafeOutputs.UpdateIssues != nil
}

// BuildStepConfig builds the step configuration for update_issue
func (h *UpdateIssueHandler) BuildStepConfig(c *Compiler, data *WorkflowData, ctx *SafeOutputContext) *SafeOutputStepConfig {
	if !h.IsEnabled(data) {
		return nil
	}

	updateIssueHandlerLog.Print("Building update_issue step config")

	// Build the step configuration using the existing builder
	stepConfig := c.buildUpdateIssueStepConfig(data, ctx.MainJobName, ctx.ThreatDetectionEnabled)

	// Add temporary ID map if available
	if ctx.TempIDMapAvailable && h.RequiresTempIDMap() {
		stepConfig.CustomEnvVars = append(stepConfig.CustomEnvVars,
			"          GH_AW_TEMPORARY_ID_MAP: ${{ steps."+ctx.TempIDMapSource+".outputs.temporary_id_map }}\n")
		updateIssueHandlerLog.Printf("Added temporary ID map reference from step: %s", ctx.TempIDMapSource)
	}

	return &stepConfig
}

// GetOutputs returns the outputs that update_issue produces
func (h *UpdateIssueHandler) GetOutputs() map[string]string {
	// update_issue doesn't produce outputs in the current implementation
	return map[string]string{}
}

// RequiresTempIDMap returns true if this handler needs access to the temporary ID map
func (h *UpdateIssueHandler) RequiresTempIDMap() bool {
	return true // update_issue may reference temporary IDs from create_issue
}
