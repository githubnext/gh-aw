package workflow

import (
	"github.com/githubnext/gh-aw/pkg/logger"
)

var closeIssueHandlerLog = logger.New("workflow:safe_output_handler_close_issue")

// CloseIssueHandler handles close_issue safe output messages
type CloseIssueHandler struct{}

// NewCloseIssueHandler creates a new close_issue handler
func NewCloseIssueHandler() *CloseIssueHandler {
	return &CloseIssueHandler{}
}

// GetType returns the type identifier for this handler
func (h *CloseIssueHandler) GetType() string {
	return "close_issue"
}

// IsEnabled checks if close_issue is enabled in the workflow configuration
func (h *CloseIssueHandler) IsEnabled(data *WorkflowData) bool {
	return data.SafeOutputs != nil && data.SafeOutputs.CloseIssues != nil
}

// BuildStepConfig builds the step configuration for close_issue
func (h *CloseIssueHandler) BuildStepConfig(c *Compiler, data *WorkflowData, ctx *SafeOutputContext) *SafeOutputStepConfig {
	if !h.IsEnabled(data) {
		return nil
	}

	closeIssueHandlerLog.Print("Building close_issue step config")

	// Build the step configuration using the existing builder
	stepConfig := c.buildCloseIssueStepConfig(data, ctx.MainJobName, ctx.ThreatDetectionEnabled)

	// Add temporary ID map if available
	if ctx.TempIDMapAvailable && h.RequiresTempIDMap() {
		stepConfig.CustomEnvVars = append(stepConfig.CustomEnvVars,
			"          GH_AW_TEMPORARY_ID_MAP: ${{ steps."+ctx.TempIDMapSource+".outputs.temporary_id_map }}\n")
		closeIssueHandlerLog.Printf("Added temporary ID map reference from step: %s", ctx.TempIDMapSource)
	}

	return &stepConfig
}

// GetOutputs returns the outputs that close_issue produces
func (h *CloseIssueHandler) GetOutputs() map[string]string {
	// close_issue doesn't produce outputs in the current implementation
	return map[string]string{}
}

// RequiresTempIDMap returns true if this handler needs access to the temporary ID map
func (h *CloseIssueHandler) RequiresTempIDMap() bool {
	return true // close_issue may reference temporary IDs from create_issue
}
