package workflow

import (
	"fmt"
	"strings"
)

// DispatchWorkflowConfig holds configuration for dispatching workflows
type DispatchWorkflowConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	AllowedWorkflows     []string `yaml:"allowed-workflows"` // Required: list of allowed workflow filenames
}

// parseDispatchWorkflowConfig handles dispatch-workflow configuration
func (c *Compiler) parseDispatchWorkflowConfig(outputMap map[string]any) *DispatchWorkflowConfig {
	if configData, exists := outputMap["dispatch-workflow"]; exists {
		config := &DispatchWorkflowConfig{}
		config.Max = 1 // Default max is 1

		if configMap, ok := configData.(map[string]any); ok {
			// Parse allowed-workflows (required)
			if allowedWorkflows, exists := configMap["allowed-workflows"]; exists {
				if workflowsArray, ok := allowedWorkflows.([]any); ok {
					var workflowStrings []string
					for _, workflow := range workflowsArray {
						if workflowStr, ok := workflow.(string); ok {
							workflowStrings = append(workflowStrings, workflowStr)
						}
					}
					config.AllowedWorkflows = workflowStrings
				}
			}

			// Parse common base fields
			c.parseBaseSafeOutputConfig(configMap, &config.BaseSafeOutputConfig)
		}

		// Validate that allowed-workflows is not empty
		if len(config.AllowedWorkflows) == 0 {
			return nil // Invalid configuration
		}

		return config
	}

	return nil
}

// buildDispatchWorkflowJob creates the dispatch_workflow job
func (c *Compiler) buildDispatchWorkflowJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.DispatchWorkflow == nil {
		return nil, fmt.Errorf("safe-outputs.dispatch-workflow configuration is required")
	}

	var steps []string

	// Build custom environment variables specific to dispatch-workflow
	var customEnvVars []string
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GITHUB_AW_WORKFLOW_NAME: %q\n", data.Name))
	
	// Pass the allowed workflows as a comma-separated list
	if len(data.SafeOutputs.DispatchWorkflow.AllowedWorkflows) > 0 {
		workflowsStr := strings.Join(data.SafeOutputs.DispatchWorkflow.AllowedWorkflows, ",")
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GITHUB_AW_ALLOWED_WORKFLOWS: %q\n", workflowsStr))
	}

	// Add common safe output job environment variables (staged)
	customEnvVars = append(customEnvVars, buildSafeOutputJobEnvVars(
		c.trialMode,
		c.trialLogicalRepoSlug,
		data.SafeOutputs.Staged,
		"", // No target repo for workflow dispatch
	)...)

	// Get token from config
	var token string
	if data.SafeOutputs.DispatchWorkflow != nil {
		token = data.SafeOutputs.DispatchWorkflow.GitHubToken
	}

	// Build the GitHub Script step using the common helper
	scriptSteps := c.buildGitHubScriptStep(data, GitHubScriptStepConfig{
		StepName:      "Dispatch Workflow",
		StepID:        "dispatch_workflow",
		MainJobName:   mainJobName,
		CustomEnvVars: customEnvVars,
		Script:        dispatchWorkflowScript,
		Token:         token,
	})
	steps = append(steps, scriptSteps...)

	// Create outputs for the job
	outputs := map[string]string{
		"workflow_name": "${{ steps.dispatch_workflow.outputs.workflow_name }}",
		"workflow_id":   "${{ steps.dispatch_workflow.outputs.workflow_id }}",
	}

	jobCondition := BuildSafeOutputType("dispatch_workflow", data.SafeOutputs.DispatchWorkflow.Min)

	// Set base permissions - need actions:write for workflow_dispatch
	permissions := "permissions:\n      contents: read\n      actions: write"

	job := &Job{
		Name:           "dispatch_workflow",
		If:             jobCondition.Render(),
		RunsOn:         c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions:    permissions,
		TimeoutMinutes: 10, // 10-minute timeout
		Steps:          steps,
		Outputs:        outputs,
		Needs:          []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}

const dispatchWorkflowScript = `
const fs = require('fs');
const core = require('@actions/core');
const github = require('@actions/github');

async function run() {
  try {
    const safeOutputsPath = process.env.GITHUB_AW_SAFE_OUTPUTS;
    if (!safeOutputsPath) {
      throw new Error('GITHUB_AW_SAFE_OUTPUTS environment variable is not set');
    }

    // Read safe outputs file
    const safeOutputsContent = fs.readFileSync(safeOutputsPath, 'utf8');
    const lines = safeOutputsContent.trim().split('\\n').filter(line => line.trim() !== '');

    if (lines.length === 0) {
      core.info('No dispatch_workflow entries found in safe outputs');
      return;
    }

    // Parse allowed workflows from environment
    const allowedWorkflowsStr = process.env.GITHUB_AW_ALLOWED_WORKFLOWS || '';
    const allowedWorkflows = allowedWorkflowsStr.split(',').map(w => w.trim()).filter(w => w !== '');

    if (allowedWorkflows.length === 0) {
      throw new Error('No allowed workflows configured');
    }

    core.info('Allowed workflows: ' + allowedWorkflows.join(', '));

    // Check if we're in staged mode
    const staged = process.env.GITHUB_AW_STAGED === 'true';

    // Parse entries
    const entries = lines.map(line => {
      try {
        const parsed = JSON.parse(line);
        // Normalize type to use underscores
        if (parsed.type) {
          parsed.type = parsed.type.replace(/-/g, '_');
        }
        return parsed;
      } catch (error) {
        core.warning('Failed to parse line: ' + line);
        return null;
      }
    }).filter(entry => entry !== null && entry.type === 'dispatch_workflow');

    if (entries.length === 0) {
      core.info('No dispatch_workflow entries found');
      return;
    }

    core.info('Found ' + entries.length + ' dispatch_workflow entries');

    // Get GitHub token
    const token = process.env.GITHUB_TOKEN || process.env.GH_TOKEN;
    if (!token) {
      throw new Error('GitHub token not found. Set GITHUB_TOKEN or GH_TOKEN environment variable.');
    }

    const octokit = github.getOctokit(token);
    const [owner, repo] = process.env.GITHUB_REPOSITORY.split('/');

    let dispatchedWorkflowName = '';
    let dispatchedWorkflowId = '';

    // Process each entry
    for (const entry of entries) {
      const { workflow_name, inputs } = entry;

      if (!workflow_name) {
        core.warning('Skipping entry without workflow_name');
        continue;
      }

      // Validate workflow is in allowed list
      if (!allowedWorkflows.includes(workflow_name)) {
        core.warning('Workflow "' + workflow_name + '" is not in the allowed list: ' + allowedWorkflows.join(', '));
        continue;
      }

      core.info('Processing dispatch for workflow: ' + workflow_name);

      if (staged) {
        // In staged mode, just log what would be dispatched
        await core.summary
          .addHeading('Workflow Dispatch (Staged)', 3)
          .addRaw('Would dispatch workflow: **' + workflow_name + '**', true)
          .addRaw(inputs ? 'With inputs: \'' + JSON.stringify(inputs) + '\'' : 'No inputs provided', true)
          .write();
        
        dispatchedWorkflowName = workflow_name;
        dispatchedWorkflowId = 'staged';
        core.info('[STAGED] Would dispatch workflow: ' + workflow_name);
      } else {
        // Actually dispatch the workflow
        try {
          const dispatchParams = {
            owner,
            repo,
            workflow_id: workflow_name,
            ref: process.env.GITHUB_REF || 'main',
          };

          // Only add inputs if they exist and are not empty
          if (inputs && Object.keys(inputs).length > 0) {
            dispatchParams.inputs = inputs;
          }

          await octokit.rest.actions.createWorkflowDispatch(dispatchParams);
          
          dispatchedWorkflowName = workflow_name;
          // The API doesn't return a workflow run ID immediately, so we set a placeholder
          dispatchedWorkflowId = 'dispatched';
          
          core.info('✓ Successfully dispatched workflow: ' + workflow_name);
          
          await core.summary
            .addHeading('Workflow Dispatched', 3)
            .addRaw('Successfully dispatched workflow: **' + workflow_name + '**', true)
            .addRaw(inputs ? 'With inputs: \'' + JSON.stringify(inputs) + '\'' : 'No inputs provided', true)
            .write();
        } catch (error) {
          core.error('Failed to dispatch workflow ' + workflow_name + ': ' + (error instanceof Error ? error.message : String(error)));
          throw error;
        }
      }
    }

    // Set outputs
    if (dispatchedWorkflowName) {
      core.setOutput('workflow_name', dispatchedWorkflowName);
      core.setOutput('workflow_id', dispatchedWorkflowId);
    }

  } catch (error) {
    core.setFailed(error instanceof Error ? error.message : String(error));
  }
}

run();
`
