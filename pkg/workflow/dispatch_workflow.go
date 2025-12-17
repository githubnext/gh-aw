package workflow

import (
	"encoding/json"
	"fmt"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var dispatchWorkflowLog = logger.New("workflow:dispatch_workflow")

// DispatchWorkflowConfig holds configuration for dispatching workflows
type DispatchWorkflowConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	AllowedWorkflows     []string `yaml:"allowed-workflows"` // Mandatory list of workflow IDs that can be dispatched
}

// parseDispatchWorkflowConfig handles dispatch-workflow configuration
func (c *Compiler) parseDispatchWorkflowConfig(outputMap map[string]any) *DispatchWorkflowConfig {
	if configData, exists := outputMap["dispatch-workflow"]; exists {
		dispatchWorkflowLog.Print("Parsing dispatch-workflow configuration")
		config := &DispatchWorkflowConfig{}
		config.Max = 1 // Default max is 1

		if configMap, ok := configData.(map[string]any); ok {
			// Parse allowed-workflows (required)
			if allowedWorkflows, exists := configMap["allowed-workflows"]; exists {
				if workflowsArray, ok := allowedWorkflows.([]any); ok {
					for _, workflow := range workflowsArray {
						if workflowStr, ok := workflow.(string); ok {
							config.AllowedWorkflows = append(config.AllowedWorkflows, workflowStr)
						}
					}
				}
			}

			// Parse common base fields
			c.parseBaseSafeOutputConfig(configMap, &config.BaseSafeOutputConfig, 1)
		}

		if len(config.AllowedWorkflows) == 0 {
			dispatchWorkflowLog.Print("Error: allowed-workflows is required and must not be empty")
			return nil // Invalid configuration
		}

		dispatchWorkflowLog.Printf("Parsed dispatch-workflow config: %d allowed workflows, max=%d", len(config.AllowedWorkflows), config.Max)
		return config
	}

	return nil
}

// buildDispatchWorkflowJob creates the dispatch_workflow job
func (c *Compiler) buildDispatchWorkflowJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.DispatchWorkflow == nil {
		return nil, fmt.Errorf("safe-outputs.dispatch-workflow configuration is required")
	}

	config := data.SafeOutputs.DispatchWorkflow

	if dispatchWorkflowLog.Enabled() {
		dispatchWorkflowLog.Printf("Building dispatch-workflow job: workflow=%s, allowed_workflows=%d, max=%d",
			data.Name, len(config.AllowedWorkflows), config.Max)
	}

	// Build custom environment variables
	var customEnvVars []string

	// Add workflow name for recursive call prevention
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_WORKFLOW_NAME: %q\n", data.Name))

	// Add allowed workflows list
	if len(config.AllowedWorkflows) > 0 {
		workflowsJSON, err := json.Marshal(config.AllowedWorkflows)
		if err == nil {
			customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_ALLOWED_WORKFLOWS: %q\n", string(workflowsJSON)))
		}
	}

	// Add max count
	customEnvVars = append(customEnvVars, BuildMaxCountEnvVar("GH_AW_DISPATCH_WORKFLOW_MAX_COUNT", config.Max)...)

	// Add common safe output job environment variables (staged mode)
	customEnvVars = append(customEnvVars, buildSafeOutputJobEnvVars(
		c.trialMode,
		c.trialLogicalRepoSlug,
		data.SafeOutputs.Staged,
		"", // No target repo slug for workflow dispatch
	)...)

	// Get token from config
	var token string
	if config.GitHubToken != "" {
		token = config.GitHubToken
	}

	// Create outputs for the job
	outputs := map[string]string{
		"workflow_id": "${{ steps.dispatch_workflow.outputs.workflow_id }}",
		"ref":         "${{ steps.dispatch_workflow.outputs.ref }}",
	}

	// Use the shared builder function to create the job
	return c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:       "dispatch_workflow",
		StepName:      "Dispatch Workflow",
		StepID:        "dispatch_workflow",
		MainJobName:   mainJobName,
		CustomEnvVars: customEnvVars,
		Script:        getDispatchWorkflowScript(),
		Permissions:   NewPermissionsContentsReadActionsWrite(),
		Outputs:       outputs,
		Token:         token,
	})
}

// getDispatchWorkflowScript returns the bundled dispatch_workflow script
func getDispatchWorkflowScript() string {
	return DefaultScriptRegistry.GetWithMode("dispatch_workflow", RuntimeModeGitHubScript)
}
