package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
)

// This file contains the buildMainJob function for building the main agent job,
// as well as the shouldAddCheckoutStep helper function.
// The agent job is the primary job that runs the AI agent with configured tools and environment.

// buildMainJob creates the main workflow job that runs the AI agent.
// This job handles:
// 1. Step generation (checkout, custom steps, agent execution, output collection)
// 2. Dependency resolution with custom jobs
// 3. Output configuration (model, safe outputs, patch)
// 4. Environment variable setup for safe outputs
func (c *Compiler) buildMainJob(data *WorkflowData, activationJobCreated bool) (*Job, error) {
	log.Printf("Building main job for workflow: %s", data.Name)
	var steps []string

	// Find custom jobs that depend on pre_activation - these are handled by the activation job
	customJobsBeforeActivation := c.getCustomJobsDependingOnPreActivation(data.Jobs)

	var jobCondition = data.If
	if activationJobCreated {
		// If the if condition references custom jobs that run before activation,
		// the activation job handles the condition, so clear it here
		if c.referencesCustomJobOutputs(data.If, data.Jobs) && len(customJobsBeforeActivation) > 0 {
			jobCondition = "" // Activation job handles this condition
		} else if !c.referencesCustomJobOutputs(data.If, data.Jobs) {
			jobCondition = "" // Main job depends on activation job, so no need for inline condition
		}
		// Note: If data.If references custom jobs that DON'T depend on pre_activation,
		// we keep the condition on the agent job
	}

	// Note: workflow_run repository safety check is applied exclusively to activation job

	// Permission checks are now handled by the separate check_membership job
	// No role checks needed in the main job

	// Build step content using the generateMainJobSteps helper method
	// but capture it into a string instead of writing directly
	var stepBuilder strings.Builder
	c.generateMainJobSteps(&stepBuilder, data)

	// Split the steps content into individual step entries
	stepsContent := stepBuilder.String()
	if stepsContent != "" {
		steps = append(steps, stepsContent)
	}

	var depends []string
	if activationJobCreated {
		depends = []string{constants.ActivationJobName} // Depend on the activation job only if it exists
	}

	// Add custom jobs as dependencies only if they don't depend on pre_activation or agent
	// Custom jobs that depend on pre_activation are now dependencies of activation,
	// so the agent job gets them transitively through activation
	// Custom jobs that depend on agent should run AFTER the agent job, not before it
	if data.Jobs != nil {
		for jobName := range data.Jobs {
			// Only add as direct dependency if it doesn't depend on pre_activation or agent
			// (jobs that depend on pre_activation are handled through activation)
			// (jobs that depend on agent are post-execution jobs like failure handlers)
			if configMap, ok := data.Jobs[jobName].(map[string]any); ok {
				if !jobDependsOnPreActivation(configMap) && !jobDependsOnAgent(configMap) {
					depends = append(depends, jobName)
				}
			}
		}
	}

	// IMPORTANT: Even though jobs that depend on pre_activation are transitively accessible
	// through the activation job, if the workflow content directly references their outputs
	// (e.g., ${{ needs.search_issues.outputs.* }}), we MUST add them as direct dependencies.
	// This is required for GitHub Actions expression evaluation and actionlint validation.
	referencedJobs := c.getReferencedCustomJobs(data.MarkdownContent, data.Jobs)
	for _, jobName := range referencedJobs {
		// Check if this job is already in depends
		alreadyDepends := false
		for _, dep := range depends {
			if dep == jobName {
				alreadyDepends = true
				break
			}
		}
		// Add it if not already present
		if !alreadyDepends {
			depends = append(depends, jobName)
			compilerJobsLog.Printf("Added direct dependency on custom job '%s' because it's referenced in workflow content", jobName)
		}
	}

	// Build outputs for all engines (GH_AW_SAFE_OUTPUTS functionality)
	// Build job outputs
	// Always include model output for reuse in other jobs
	outputs := map[string]string{
		"model": "${{ steps.generate_aw_info.outputs.model }}",
	}

	// Add safe-output specific outputs if the workflow uses the safe-outputs feature
	if data.SafeOutputs != nil {
		outputs["output"] = "${{ steps.collect_output.outputs.output }}"
		outputs["output_types"] = "${{ steps.collect_output.outputs.output_types }}"
		outputs["has_patch"] = "${{ steps.collect_output.outputs.has_patch }}"
	}

	// Build job-level environment variables for safe outputs
	var env map[string]string
	if data.SafeOutputs != nil {
		env = make(map[string]string)

		// Set GH_AW_SAFE_OUTPUTS to fixed path
		env["GH_AW_SAFE_OUTPUTS"] = "/tmp/gh-aw/safeoutputs/outputs.jsonl"

		// Set GH_AW_MCP_LOG_DIR for safe outputs MCP server logging
		// Store in mcp-logs directory so it's included in mcp-logs artifact
		env["GH_AW_MCP_LOG_DIR"] = "/tmp/gh-aw/mcp-logs/safeoutputs"

		// Set config and tools paths (files are written to these paths)
		env["GH_AW_SAFE_OUTPUTS_CONFIG_PATH"] = "/tmp/gh-aw/safeoutputs/config.json"
		env["GH_AW_SAFE_OUTPUTS_TOOLS_PATH"] = "/tmp/gh-aw/safeoutputs/tools.json"

		// Add asset-related environment variables if upload-assets is configured
		if data.SafeOutputs.UploadAssets != nil {
			env["GH_AW_ASSETS_BRANCH"] = fmt.Sprintf("%q", data.SafeOutputs.UploadAssets.BranchName)
			env["GH_AW_ASSETS_MAX_SIZE_KB"] = fmt.Sprintf("%d", data.SafeOutputs.UploadAssets.MaxSizeKB)
			env["GH_AW_ASSETS_ALLOWED_EXTS"] = fmt.Sprintf("%q", strings.Join(data.SafeOutputs.UploadAssets.AllowedExts, ","))
		}
	}

	// Generate agent concurrency configuration
	agentConcurrency := GenerateJobConcurrencyConfig(data)

	job := &Job{
		Name:        constants.AgentJobName,
		If:          jobCondition,
		RunsOn:      c.indentYAMLLines(data.RunsOn, "    "),
		Environment: c.indentYAMLLines(data.Environment, "    "),
		Container:   c.indentYAMLLines(data.Container, "    "),
		Services:    c.indentYAMLLines(data.Services, "    "),
		Permissions: c.indentYAMLLines(data.Permissions, "    "),
		Concurrency: c.indentYAMLLines(agentConcurrency, "    "),
		Env:         env,
		Steps:       steps,
		Needs:       depends,
		Outputs:     outputs,
	}

	return job, nil
}

// shouldAddCheckoutStep determines if the checkout step should be added to the main job.
// Returns true if checkout is needed and not already present in custom steps.
// Checkout is needed when:
// - Custom agent file is specified (requires file system access)
// - Permissions grant contents:read access and custom steps don't have checkout
func (c *Compiler) shouldAddCheckoutStep(data *WorkflowData) bool {
	// Check condition 1: If custom steps already contain checkout, don't add another one
	if data.CustomSteps != "" && ContainsCheckout(data.CustomSteps) {
		log.Print("Skipping checkout step: custom steps already contain checkout")
		return false // Custom steps already have checkout
	}

	// Check condition 2: If custom agent file is specified (via imports), checkout is required
	if data.AgentFile != "" {
		log.Printf("Adding checkout step: custom agent file specified: %s", data.AgentFile)
		return true // Custom agent file requires checkout to access the file
	}

	// Check condition 3: If permissions don't grant contents access, don't add checkout
	permParser := NewPermissionsParser(data.Permissions)
	if !permParser.HasContentsReadAccess() {
		log.Print("Skipping checkout step: no contents read access in permissions")
		return false // No contents read access, so checkout is not needed
	}

	// If we get here, permissions allow contents access and custom steps (if any) don't contain checkout
	return true // Add checkout because it's needed and not already present
}
