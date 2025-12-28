package workflow

import (
	"fmt"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
)

// ========================================
// Safe Output Configuration Helpers
// ========================================

// formatSafeOutputsRunsOn formats the runs-on value from SafeOutputsConfig for job output
func (c *Compiler) formatSafeOutputsRunsOn(safeOutputs *SafeOutputsConfig) string {
	if safeOutputs == nil || safeOutputs.RunsOn == "" {
		return fmt.Sprintf("runs-on: %s", constants.DefaultActivationJobRunnerImage)
	}

	return fmt.Sprintf("runs-on: %s", safeOutputs.RunsOn)
}

// HasSafeOutputsEnabled checks if any safe-outputs are enabled
func HasSafeOutputsEnabled(safeOutputs *SafeOutputsConfig) bool {
	if safeOutputs == nil {
		return false
	}
	enabled := safeOutputs.CreateIssues != nil ||
		safeOutputs.CreateAgentTasks != nil ||
		safeOutputs.CreateDiscussions != nil ||
		safeOutputs.UpdateDiscussions != nil ||
		safeOutputs.CloseDiscussions != nil ||
		safeOutputs.CloseIssues != nil ||
		safeOutputs.AddComments != nil ||
		safeOutputs.CreatePullRequests != nil ||
		safeOutputs.CreatePullRequestReviewComments != nil ||
		safeOutputs.CreateCodeScanningAlerts != nil ||
		safeOutputs.AddLabels != nil ||
		safeOutputs.AddReviewer != nil ||
		safeOutputs.AssignMilestone != nil ||
		safeOutputs.AssignToAgent != nil ||
		safeOutputs.AssignToUser != nil ||
		safeOutputs.UpdateIssues != nil ||
		safeOutputs.UpdatePullRequests != nil ||
		safeOutputs.PushToPullRequestBranch != nil ||
		safeOutputs.UploadAssets != nil ||
		safeOutputs.MissingTool != nil ||
		safeOutputs.NoOp != nil ||
		safeOutputs.LinkSubIssue != nil ||
		safeOutputs.HideComment != nil ||
		len(safeOutputs.Jobs) > 0

	if safeOutputsConfigLog.Enabled() {
		safeOutputsConfigLog.Printf("Safe outputs enabled check: %v", enabled)
	}

	return enabled
}

// GetEnabledSafeOutputToolNames returns a list of enabled safe output tool names
// that can be used in the prompt to inform the agent which tools are available
func GetEnabledSafeOutputToolNames(safeOutputs *SafeOutputsConfig) []string {
	if safeOutputs == nil {
		return nil
	}

	var tools []string

	// Check each tool field and add to list if enabled
	if safeOutputs.CreateIssues != nil {
		tools = append(tools, "create_issue")
	}
	if safeOutputs.CreateAgentTasks != nil {
		tools = append(tools, "create_agent_task")
	}
	if safeOutputs.CreateDiscussions != nil {
		tools = append(tools, "create_discussion")
	}
	if safeOutputs.UpdateDiscussions != nil {
		tools = append(tools, "update_discussion")
	}
	if safeOutputs.CloseDiscussions != nil {
		tools = append(tools, "close_discussion")
	}
	if safeOutputs.CloseIssues != nil {
		tools = append(tools, "close_issue")
	}
	if safeOutputs.ClosePullRequests != nil {
		tools = append(tools, "close_pull_request")
	}
	if safeOutputs.AddComments != nil {
		tools = append(tools, "add_comment")
	}
	if safeOutputs.CreatePullRequests != nil {
		tools = append(tools, "create_pull_request")
	}
	if safeOutputs.CreatePullRequestReviewComments != nil {
		tools = append(tools, "create_pull_request_review_comment")
	}
	if safeOutputs.CreateCodeScanningAlerts != nil {
		tools = append(tools, "create_code_scanning_alert")
	}
	if safeOutputs.AddLabels != nil {
		tools = append(tools, "add_labels")
	}
	if safeOutputs.AddReviewer != nil {
		tools = append(tools, "add_reviewer")
	}
	if safeOutputs.AssignMilestone != nil {
		tools = append(tools, "assign_milestone")
	}
	if safeOutputs.AssignToAgent != nil {
		tools = append(tools, "assign_to_agent")
	}
	if safeOutputs.AssignToUser != nil {
		tools = append(tools, "assign_to_user")
	}
	if safeOutputs.UpdateIssues != nil {
		tools = append(tools, "update_issue")
	}
	if safeOutputs.UpdatePullRequests != nil {
		tools = append(tools, "update_pull_request")
	}
	if safeOutputs.PushToPullRequestBranch != nil {
		tools = append(tools, "push_to_pull_request_branch")
	}
	if safeOutputs.UploadAssets != nil {
		tools = append(tools, "upload_asset")
	}
	if safeOutputs.UpdateRelease != nil {
		tools = append(tools, "update_release")
	}
	if safeOutputs.UpdateProjects != nil {
		tools = append(tools, "update_project")
	}
	if safeOutputs.LinkSubIssue != nil {
		tools = append(tools, "link_sub_issue")
	}
	if safeOutputs.HideComment != nil {
		tools = append(tools, "hide_comment")
	}
	if safeOutputs.MissingTool != nil {
		tools = append(tools, "missing_tool")
	}
	if safeOutputs.NoOp != nil {
		tools = append(tools, "noop")
	}

	// Add custom job tools
	for jobName := range safeOutputs.Jobs {
		tools = append(tools, jobName)
	}

	// Sort tools to ensure deterministic compilation
	sort.Strings(tools)

	if safeOutputsConfigLog.Enabled() {
		safeOutputsConfigLog.Printf("Enabled safe output tools: %v", tools)
	}

	return tools
}

// normalizeSafeOutputIdentifier converts dashes to underscores for safe output identifiers.
//
// This is a NORMALIZE function (format standardization pattern). Use this when ensuring
// consistency across the system while remaining resilient to LLM-generated variations.
//
// Safe output identifiers may appear in different formats:
//   - YAML configuration: "create-issue" (dash-separated)
//   - JavaScript code: "create_issue" (underscore-separated)
//   - Internal usage: can vary based on source
//
// This function normalizes all variations to a canonical underscore-separated format,
// ensuring consistent internal representation regardless of input format.
//
// Example inputs and outputs:
//
//	normalizeSafeOutputIdentifier("create-issue")      // returns "create_issue"
//	normalizeSafeOutputIdentifier("create_issue")      // returns "create_issue" (unchanged)
//	normalizeSafeOutputIdentifier("add-comment")       // returns "add_comment"
//
// Note: This function assumes the input is already a valid identifier. It does NOT
// perform character validation or sanitization - it only converts between naming
// conventions. Both dash-separated and underscore-separated formats are valid;
// this function simply standardizes to the internal representation.
//
// See package documentation for guidance on when to use sanitize vs normalize patterns.
func normalizeSafeOutputIdentifier(identifier string) string {
	normalized := strings.ReplaceAll(identifier, "-", "_")
	if safeOutputsConfigLog.Enabled() {
		safeOutputsConfigLog.Printf("Normalized safe output identifier: %s -> %s", identifier, normalized)
	}
	return normalized
}
