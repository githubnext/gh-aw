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

	// Build the auto-generated outputs map based on configured safe output types
	generatedOutputs := buildWorkflowCallOutputsMap(safeOutputs)
	if len(generatedOutputs) == 0 {
		return onSection
	}

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
