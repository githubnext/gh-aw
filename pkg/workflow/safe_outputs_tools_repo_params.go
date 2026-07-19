package workflow

import (
	"fmt"
	"reflect"
)

// addRepoParameterIfNeeded adds a "repo" parameter to the tool's inputSchema
// if the safe output configuration has allowed-repos entries or a wildcard "*" target-repo
func addRepoParameterIfNeeded(tool map[string]any, toolName string, safeOutputs *SafeOutputsConfig) {
	safeOutputsConfigLog.Printf("Checking if repo parameter needed for tool: %s", toolName)
	if safeOutputs == nil {
		return
	}

	hasAllowedRepos, targetRepoSlug := repoParameterConfigForTool(toolName, safeOutputs)

	// Only add repo parameter if allowed-repos has entries or target-repo is wildcard ("*")
	if !hasAllowedRepos && targetRepoSlug != "*" {
		safeOutputsConfigLog.Printf("Skipping repo parameter for tool %s: no allowed-repos and target-repo is not wildcard", toolName)
		return
	}

	// Get the inputSchema
	inputSchema, ok := tool["inputSchema"].(map[string]any)
	if !ok {
		return
	}

	properties, ok := inputSchema["properties"].(map[string]any)
	if !ok {
		return
	}

	// Build repo parameter description
	var repoDescription string
	if targetRepoSlug == "*" {
		repoDescription = "Target repository for this operation in 'owner/repo' format. Any repository can be targeted."
	} else if targetRepoSlug != "" {
		repoDescription = fmt.Sprintf("Target repository for this operation in 'owner/repo' format. Default is %q. Must be the target-repo or in the allowed-repos list.", targetRepoSlug)
	} else {
		repoDescription = "Target repository for this operation in 'owner/repo' format. Must be the target-repo or in the allowed-repos list."
	}

	// Add repo parameter to properties
	properties["repo"] = map[string]any{
		"type":        "string",
		"description": repoDescription,
	}

	safeOutputsConfigLog.Printf("Added repo parameter to tool: %s (has allowed-repos or wildcard target-repo)", toolName)
}

func repoParameterConfigForTool(toolName string, safeOutputs *SafeOutputsConfig) (bool, string) {
	fieldName, ok := repoParameterConfigFields()[toolName]
	if !ok {
		return false, ""
	}
	configValue := reflect.ValueOf(safeOutputs).Elem().FieldByName(fieldName)
	if !configValue.IsValid() || configValue.IsNil() {
		return false, ""
	}
	config := configValue.Elem()
	allowedRepos := config.FieldByName("AllowedRepos")
	targetRepoSlug := config.FieldByName("TargetRepoSlug")
	if !allowedRepos.IsValid() || !targetRepoSlug.IsValid() {
		return false, ""
	}
	return allowedRepos.Len() > 0, targetRepoSlug.String()
}

func repoParameterConfigFields() map[string]string {
	return map[string]string{
		"create_issue":                          "CreateIssues",
		"create_discussion":                     "CreateDiscussions",
		"add_comment":                           "AddComments",
		"create_pull_request":                   "CreatePullRequests",
		"create_pull_request_review_comment":    "CreatePullRequestReviewComments",
		"reply_to_pull_request_review_comment":  "ReplyToPullRequestReviewComment",
		"dismiss_pull_request_review":           "DismissPullRequestReview",
		"create_agent_session":                  "CreateAgentSessions",
		"close_issue":                           "CloseIssues",
		"update_issue":                          "UpdateIssues",
		"close_discussion":                      "CloseDiscussions",
		"update_discussion":                     "UpdateDiscussions",
		"close_pull_request":                    "ClosePullRequests",
		"update_pull_request":                   "UpdatePullRequests",
		"merge_pull_request":                    "MergePullRequest",
		"add_labels":                            "AddLabels",
		"remove_labels":                         "RemoveLabels",
		"replace_label":                         "ReplaceLabel",
		"hide_comment":                          "HideComment",
		"link_sub_issue":                        "LinkSubIssue",
		"mark_pull_request_as_ready_for_review": "MarkPullRequestAsReadyForReview",
		"add_reviewer":                          "AddReviewer",
		"assign_milestone":                      "AssignMilestone",
		"assign_to_agent":                       "AssignToAgent",
		"assign_to_user":                        "AssignToUser",
		"unassign_from_user":                    "UnassignFromUser",
		"set_issue_type":                        "SetIssueType",
		"set_issue_field":                       "SetIssueField",
	}
}

// computeRepoParamForTool returns the "repo" input parameter definition that should
// be added to a tool's inputSchema, or nil if no repo parameter is needed.
// This mirrors the logic in addRepoParameterIfNeeded but returns the param instead
// of modifying a tool in place, making it usable for generateToolsMetaJSON.
func computeRepoParamForTool(toolName string, safeOutputs *SafeOutputsConfig) map[string]any {
	safeOutputsConfigLog.Printf("Computing repo parameter definition for tool: %s", toolName)
	// Reuse addRepoParameterIfNeeded by passing a scratch tool with an empty inputSchema.
	scratch := map[string]any{
		"name":        toolName,
		"inputSchema": map[string]any{"properties": map[string]any{}},
	}
	addRepoParameterIfNeeded(scratch, toolName, safeOutputs)

	inputSchema, ok := scratch["inputSchema"].(map[string]any)
	if !ok {
		return nil
	}
	properties, ok := inputSchema["properties"].(map[string]any)
	if !ok {
		return nil
	}
	repoProp, ok := properties["repo"].(map[string]any)
	if !ok {
		safeOutputsConfigLog.Printf("No repo parameter generated for tool: %s", toolName)
		return nil
	}
	safeOutputsConfigLog.Printf("Repo parameter computed for tool: %s", toolName)
	return repoProp
}
