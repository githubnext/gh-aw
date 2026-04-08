package workflow

import (
	"maps"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/goccy/go-yaml"
)

var workflowCallLog = logger.New("workflow:compiler_workflow_call")

// workflowCallOutputEntry represents a single on.workflow_call.outputs entry
type workflowCallOutputEntry struct {
	Description string `yaml:"description"`
	Value       string `yaml:"value"`
}

// hasWorkflowCallTrigger checks if the on section contains a workflow_call trigger.
// Used to detect cross-repo reusable workflow usage for checkout and error handling.
func hasWorkflowCallTrigger(onSection string) bool {
	return strings.Contains(onSection, "workflow_call")
}

// generateArtifactPrefixStep creates a step that computes a stable, unique artifact name
// prefix from a hash of the workflow_call inputs and the run attempt. This ensures artifact
// names do not clash when the same reusable workflow is called multiple times within a
// single workflow run (e.g. two jobs in the calling workflow each invoking the same lock.yml).
//
// The computation is delegated to actions/setup/sh/compute_artifact_prefix.sh (copied to
// ${RUNNER_TEMP}/gh-aw/actions/ at runtime by the Setup Scripts step) which:
//   - Hashes INPUTS_JSON + GITHUB_RUN_ATTEMPT using sha256, taking the first 8 hex chars.
//   - Logs what it is hashing so the prefix is traceable in workflow logs.
//   - Yields a value like "a1b2c3d4-".
//
// Uniqueness guarantee:
//   - Two calls with different inputs → different prefixes.
//   - Two calls with the same inputs on different run attempts → different prefixes.
//   - Two calls with identical inputs on the same run attempt → same prefix (conflict).
//     Callers MUST provide different inputs to avoid this edge case.
//
// Security note: inputs are passed through an environment variable rather than being
// interpolated directly into the shell script to prevent template injection.
func generateArtifactPrefixStep() []string {
	return []string{
		"      - name: Compute artifact prefix\n",
		"        id: artifact-prefix\n",
		"        env:\n",
		"          INPUTS_JSON: ${{ toJSON(inputs) }}\n",
		"        run: bash \"${RUNNER_TEMP}/gh-aw/actions/compute_artifact_prefix.sh\"\n",
	}
}

// artifactPrefixExprForActivationJob returns the GitHub Actions expression for the artifact
// prefix used within the activation job itself (references a step output).
// Returns empty string for non-workflow_call workflows.
func artifactPrefixExprForActivationJob(data *WorkflowData) string {
	if !hasWorkflowCallTrigger(data.On) {
		return ""
	}
	return "${{ steps.artifact-prefix.outputs.prefix }}"
}

// artifactPrefixExprForDownstreamJob returns the GitHub Actions expression for the artifact
// prefix used in jobs that depend on the activation job (references an activation job output).
// Returns empty string for non-workflow_call workflows.
func artifactPrefixExprForDownstreamJob(data *WorkflowData) string {
	if !hasWorkflowCallTrigger(data.On) {
		return ""
	}
	return "${{ needs.activation.outputs.artifact_prefix }}"
}

// artifactPrefixExprForAgentDownstreamJob returns the expression for the artifact prefix in
// jobs that only directly depend on the agent job (not the activation job).
// Returns empty string for non-workflow_call workflows.
func artifactPrefixExprForAgentDownstreamJob(data *WorkflowData) string {
	if !hasWorkflowCallTrigger(data.On) {
		return ""
	}
	return "${{ needs.agent.outputs.artifact_prefix }}"
}

// injectWorkflowCallOutputs adds on.workflow_call.outputs declarations for safe-output results
// when the workflow uses workflow_call as a trigger.
//
// This enables callers of the workflow to access results such as:
//   - created_issue_number / created_issue_url  (when create-issue is configured)
//   - created_pr_number / created_pr_url        (when create-pull-request is configured)
//   - comment_id / comment_url                  (when add-comment is configured)
//   - push_commit_sha / push_commit_url         (when push-to-pull-request-branch is configured)
//
// The function is a no-op if safeOutputs is nil or workflow_call is not in the on section.
// Any outputs the user has already declared in the on.workflow_call.outputs section are preserved.
func (c *Compiler) injectWorkflowCallOutputs(onSection string, safeOutputs *SafeOutputsConfig) string {
	if safeOutputs == nil || !strings.Contains(onSection, "workflow_call") {
		return onSection
	}

	workflowCallLog.Print("Injecting workflow_call outputs for safe-output results")

	// Build the auto-generated outputs map based on configured safe output types
	generatedOutputs := buildWorkflowCallOutputsMap(safeOutputs)
	if len(generatedOutputs) == 0 {
		workflowCallLog.Print("No workflow_call outputs to inject (no safe-output types configured)")
		return onSection
	}

	workflowCallLog.Printf("Generated %d workflow_call outputs to inject", len(generatedOutputs))

	// Parse the on section YAML
	var onData map[string]any
	if err := yaml.Unmarshal([]byte(onSection), &onData); err != nil {
		workflowCallLog.Printf("Warning: failed to parse on section for workflow_call outputs injection: %v", err)
		return onSection
	}

	// Get the 'on' map
	onMap, ok := onData["on"].(map[string]any)
	if !ok {
		return onSection
	}

	// Get the workflow_call entry
	workflowCallVal, hasWorkflowCall := onMap["workflow_call"]
	if !hasWorkflowCall {
		return onSection
	}

	// Convert workflow_call to a map (it may be nil if declared without options)
	var workflowCallMap map[string]any
	if workflowCallVal == nil {
		workflowCallMap = make(map[string]any)
	} else if m, ok := workflowCallVal.(map[string]any); ok {
		workflowCallMap = m
	} else {
		workflowCallMap = make(map[string]any)
	}

	// Merge auto-generated outputs with any existing user-defined outputs.
	// User-defined outputs take precedence (their keys overwrite generated ones).
	mergedOutputs := make(map[string]workflowCallOutputEntry)
	maps.Copy(mergedOutputs, generatedOutputs)
	if existingOutputs, hasOutputs := workflowCallMap["outputs"].(map[string]any); hasOutputs {
		for k, v := range existingOutputs {
			// User-defined entries may be maps with description+value or plain strings
			if outputMap, ok := v.(map[string]any); ok {
				entry := workflowCallOutputEntry{}
				if desc, ok := outputMap["description"].(string); ok {
					entry.Description = desc
				}
				if val, ok := outputMap["value"].(string); ok {
					entry.Value = val
				}
				mergedOutputs[k] = entry
			}
		}
	}

	workflowCallLog.Printf("Merged workflow_call outputs: total=%d", len(mergedOutputs))
	workflowCallMap["outputs"] = mergedOutputs
	onMap["workflow_call"] = workflowCallMap

	// Re-marshal to YAML
	newOnData := map[string]any{"on": onMap}
	newYAML, err := yaml.Marshal(newOnData)
	if err != nil {
		workflowCallLog.Printf("Warning: failed to marshal on section with workflow_call outputs: %v", err)
		return onSection
	}

	return strings.TrimSuffix(string(newYAML), "\n")
}

// buildWorkflowCallOutputsMap constructs the outputs map for on.workflow_call.outputs
// based on which safe output types are configured.
func buildWorkflowCallOutputsMap(safeOutputs *SafeOutputsConfig) map[string]workflowCallOutputEntry {
	workflowCallLog.Printf("Building workflow_call outputs map: create_issues=%t, create_prs=%t, add_comments=%t, push_to_pr=%t",
		safeOutputs.CreateIssues != nil,
		safeOutputs.CreatePullRequests != nil,
		safeOutputs.AddComments != nil,
		safeOutputs.PushToPullRequestBranch != nil)

	outputs := make(map[string]workflowCallOutputEntry)

	if safeOutputs.CreateIssues != nil {
		outputs["created_issue_number"] = workflowCallOutputEntry{
			Description: "Number of the first created issue",
			Value:       "${{ jobs.safe_outputs.outputs.created_issue_number }}",
		}
		outputs["created_issue_url"] = workflowCallOutputEntry{
			Description: "URL of the first created issue",
			Value:       "${{ jobs.safe_outputs.outputs.created_issue_url }}",
		}
	}

	if safeOutputs.CreatePullRequests != nil {
		outputs["created_pr_number"] = workflowCallOutputEntry{
			Description: "Number of the first created pull request",
			Value:       "${{ jobs.safe_outputs.outputs.created_pr_number }}",
		}
		outputs["created_pr_url"] = workflowCallOutputEntry{
			Description: "URL of the first created pull request",
			Value:       "${{ jobs.safe_outputs.outputs.created_pr_url }}",
		}
	}

	if safeOutputs.AddComments != nil {
		outputs["comment_id"] = workflowCallOutputEntry{
			Description: "ID of the first added comment",
			Value:       "${{ jobs.safe_outputs.outputs.comment_id }}",
		}
		outputs["comment_url"] = workflowCallOutputEntry{
			Description: "URL of the first added comment",
			Value:       "${{ jobs.safe_outputs.outputs.comment_url }}",
		}
	}

	if safeOutputs.PushToPullRequestBranch != nil {
		outputs["push_commit_sha"] = workflowCallOutputEntry{
			Description: "SHA of the pushed commit",
			Value:       "${{ jobs.safe_outputs.outputs.push_commit_sha }}",
		}
		outputs["push_commit_url"] = workflowCallOutputEntry{
			Description: "URL of the pushed commit",
			Value:       "${{ jobs.safe_outputs.outputs.push_commit_url }}",
		}
	}

	return outputs
}
