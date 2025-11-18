package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var dispatchWorkflowLog = logger.New("workflow:dispatch_workflow")

// DispatchWorkflowConfig holds configuration for dispatching workflows from agent output
type DispatchWorkflowConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	AllowedWorkflows     []string `yaml:"allowed-workflows,omitempty"` // Allowlist of workflow files that can be dispatched
	TargetRepoSlug       string   `yaml:"target-repo,omitempty"`       // Target repository for cross-repo dispatch (format: "owner/repo")
}

// parseDispatchWorkflowConfig handles dispatch-workflow configuration
func (c *Compiler) parseDispatchWorkflowConfig(outputMap map[string]any) *DispatchWorkflowConfig {
	if configData, exists := outputMap["dispatch-workflow"]; exists {
		dispatchWorkflowLog.Print("Parsing dispatch-workflow configuration")
		config := &DispatchWorkflowConfig{}

		if configMap, ok := configData.(map[string]any); ok {
			// Parse common base fields with default max of 1
			c.parseBaseSafeOutputConfig(configMap, &config.BaseSafeOutputConfig, 1)

			// Parse allowed-workflows (supports both string and array)
			if allowedWorkflows, exists := configMap["allowed-workflows"]; exists {
				if workflowStr, ok := allowedWorkflows.(string); ok {
					// Single string format
					config.AllowedWorkflows = []string{workflowStr}
				} else if workflowsArray, ok := allowedWorkflows.([]any); ok {
					// Array format
					var workflowStrings []string
					for _, workflow := range workflowsArray {
						if workflowStr, ok := workflow.(string); ok {
							workflowStrings = append(workflowStrings, workflowStr)
						}
					}
					config.AllowedWorkflows = workflowStrings
				}
			}

			// Parse target-repo using shared helper with validation
			targetRepoSlug, isInvalid := parseTargetRepoWithValidation(configMap)
			if isInvalid {
				return nil // Invalid configuration, return nil to cause validation error
			}
			config.TargetRepoSlug = targetRepoSlug
		} else {
			// If configData is nil or not a map (e.g., "dispatch-workflow:" with no value),
			// still set the default max
			config.Max = 1
		}

		return config
	}

	return nil
}

// buildCreateOutputDispatchWorkflowJob creates the dispatch_workflow job
func (c *Compiler) buildCreateOutputDispatchWorkflowJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.DispatchWorkflow == nil {
		return nil, fmt.Errorf("safe-outputs.dispatch-workflow configuration is required")
	}

	// Validate that allowed-workflows is configured
	if len(data.SafeOutputs.DispatchWorkflow.AllowedWorkflows) == 0 {
		return nil, fmt.Errorf("safe-outputs.dispatch-workflow requires 'allowed-workflows' configuration. Example:\nsafe-outputs:\n  dispatch-workflow:\n    allowed-workflows: [\"workflow1.yml\", \"workflow2.yml\"]")
	}

	if dispatchWorkflowLog.Enabled() {
		dispatchWorkflowLog.Printf("Building dispatch-workflow job: workflow=%s, main_job=%s, allowed_workflows=%d",
			data.Name, mainJobName, len(data.SafeOutputs.DispatchWorkflow.AllowedWorkflows))
	}

	// Build custom environment variables specific to dispatch-workflow
	var customEnvVars []string

	// Add allowed workflows as comma-separated list
	if len(data.SafeOutputs.DispatchWorkflow.AllowedWorkflows) > 0 {
		workflowsStr := strings.Join(data.SafeOutputs.DispatchWorkflow.AllowedWorkflows, ",")
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_ALLOWED_WORKFLOWS: %q\n", workflowsStr))
	}

	// Add standard environment variables (metadata + staged/target repo)
	customEnvVars = append(customEnvVars, c.buildStandardSafeOutputEnvVars(data, data.SafeOutputs.DispatchWorkflow.TargetRepoSlug)...)

	// Get token from config
	var token string
	if data.SafeOutputs.DispatchWorkflow != nil {
		token = data.SafeOutputs.DispatchWorkflow.GitHubToken
	}

	// Create outputs for the job
	outputs := map[string]string{
		"workflow_name": "${{ steps.dispatch_workflow.outputs.workflow_name }}",
		"workflow_ref":  "${{ steps.dispatch_workflow.outputs.workflow_ref }}",
	}

	// Use the shared builder function to create the job
	return c.buildSafeOutputJob(data, SafeOutputJobConfig{
		JobName:        "dispatch_workflow",
		StepName:       "Dispatch Workflow",
		StepID:         "dispatch_workflow",
		MainJobName:    mainJobName,
		CustomEnvVars:  customEnvVars,
		Script:         getDispatchWorkflowScript(),
		Permissions:    NewPermissionsContentsRead(), // Read-only by default; actions permission handled via token
		Outputs:        outputs,
		Token:          token,
		TargetRepoSlug: data.SafeOutputs.DispatchWorkflow.TargetRepoSlug,
	})
}

// getDispatchWorkflowScript is defined in scripts.go and returns the bundled JavaScript implementation
