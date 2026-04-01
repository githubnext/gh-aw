package workflow

import "fmt"

// addRepoParameterIfNeeded adds a "repo" parameter to the tool's inputSchema
// if the safe output configuration has allowed-repos entries or a wildcard "*" target-repo
func addRepoParameterIfNeeded(tool map[string]any, toolName string, safeOutputs *SafeOutputsConfig) {
	safeOutputsConfigLog.Printf("Checking if repo parameter needed for tool: %s", toolName)
	if safeOutputs == nil {
		return
	}

	// Determine if this tool should have a repo parameter based on allowed-repos and target-repo configuration (including wildcard "*")
	var hasAllowedRepos bool
	var targetRepoSlug string

	switch toolName {
	case "create_issue":
		if config := safeOutputs.CreateIssues; config != nil {
			hasAllowedRepos = len(config.AllowedRepos) > 0
			targetRepoSlug = config.TargetRepoSlug
		}
	case "create_discussion":
		if config := safeOutputs.CreateDiscussions; config != nil {
			hasAllowedRepos = len(config.AllowedRepos) > 0
			targetRepoSlug = config.TargetRepoSlug
		}
	case "add_comment":
		if config := safeOutputs.AddComments; config != nil {
			hasAllowedRepos = len(config.AllowedRepos) > 0
			targetRepoSlug = config.TargetRepoSlug
		}
	case "create_pull_request":
		if config := safeOutputs.CreatePullRequests; config != nil {
			hasAllowedRepos = len(config.AllowedRepos) > 0
			targetRepoSlug = config.TargetRepoSlug
		}
	case "create_pull_request_review_comment":
		if config := safeOutputs.CreatePullRequestReviewComments; config != nil {
			hasAllowedRepos = len(config.AllowedRepos) > 0
			targetRepoSlug = config.TargetRepoSlug
		}
	case "reply_to_pull_request_review_comment":
		if config := safeOutputs.ReplyToPullRequestReviewComment; config != nil {
			hasAllowedRepos = len(config.AllowedRepos) > 0
			targetRepoSlug = config.TargetRepoSlug
		}
	case "create_agent_session":
		if config := safeOutputs.CreateAgentSessions; config != nil {
			hasAllowedRepos = len(config.AllowedRepos) > 0
			targetRepoSlug = config.TargetRepoSlug
		}
	case "close_issue", "update_issue":
		if config := safeOutputs.CloseIssues; config != nil && toolName == "close_issue" {
			hasAllowedRepos = len(config.AllowedRepos) > 0
			targetRepoSlug = config.TargetRepoSlug
		} else if config := safeOutputs.UpdateIssues; config != nil && toolName == "update_issue" {
			hasAllowedRepos = len(config.AllowedRepos) > 0
			targetRepoSlug = config.TargetRepoSlug
		}
	case "close_discussion", "update_discussion":
		if config := safeOutputs.CloseDiscussions; config != nil && toolName == "close_discussion" {
			hasAllowedRepos = len(config.AllowedRepos) > 0
			targetRepoSlug = config.TargetRepoSlug
		} else if config := safeOutputs.UpdateDiscussions; config != nil && toolName == "update_discussion" {
			hasAllowedRepos = len(config.AllowedRepos) > 0
			targetRepoSlug = config.TargetRepoSlug
		}
	case "close_pull_request", "update_pull_request":
		if config := safeOutputs.ClosePullRequests; config != nil && toolName == "close_pull_request" {
			hasAllowedRepos = len(config.AllowedRepos) > 0
			targetRepoSlug = config.TargetRepoSlug
		} else if config := safeOutputs.UpdatePullRequests; config != nil && toolName == "update_pull_request" {
			hasAllowedRepos = len(config.AllowedRepos) > 0
			targetRepoSlug = config.TargetRepoSlug
		}
	case "add_labels", "remove_labels", "hide_comment", "link_sub_issue", "mark_pull_request_as_ready_for_review",
		"add_reviewer", "assign_milestone", "assign_to_agent", "assign_to_user", "unassign_from_user",
		"set_issue_type":
		// These use SafeOutputTargetConfig - check the appropriate config
		switch toolName {
		case "add_labels":
			if config := safeOutputs.AddLabels; config != nil {
				hasAllowedRepos = len(config.AllowedRepos) > 0
				targetRepoSlug = config.TargetRepoSlug
			}
		case "remove_labels":
			if config := safeOutputs.RemoveLabels; config != nil {
				hasAllowedRepos = len(config.AllowedRepos) > 0
				targetRepoSlug = config.TargetRepoSlug
			}
		case "hide_comment":
			if config := safeOutputs.HideComment; config != nil {
				hasAllowedRepos = len(config.AllowedRepos) > 0
				targetRepoSlug = config.TargetRepoSlug
			}
		case "link_sub_issue":
			if config := safeOutputs.LinkSubIssue; config != nil {
				hasAllowedRepos = len(config.AllowedRepos) > 0
				targetRepoSlug = config.TargetRepoSlug
			}
		case "mark_pull_request_as_ready_for_review":
			if config := safeOutputs.MarkPullRequestAsReadyForReview; config != nil {
				hasAllowedRepos = len(config.AllowedRepos) > 0
				targetRepoSlug = config.TargetRepoSlug
			}
		case "add_reviewer":
			if config := safeOutputs.AddReviewer; config != nil {
				hasAllowedRepos = len(config.AllowedRepos) > 0
				targetRepoSlug = config.TargetRepoSlug
			}
		case "assign_milestone":
			if config := safeOutputs.AssignMilestone; config != nil {
				hasAllowedRepos = len(config.AllowedRepos) > 0
				targetRepoSlug = config.TargetRepoSlug
			}
		case "assign_to_agent":
			if config := safeOutputs.AssignToAgent; config != nil {
				hasAllowedRepos = len(config.AllowedRepos) > 0
				targetRepoSlug = config.TargetRepoSlug
			}
		case "assign_to_user":
			if config := safeOutputs.AssignToUser; config != nil {
				hasAllowedRepos = len(config.AllowedRepos) > 0
				targetRepoSlug = config.TargetRepoSlug
			}
		case "unassign_from_user":
			if config := safeOutputs.UnassignFromUser; config != nil {
				hasAllowedRepos = len(config.AllowedRepos) > 0
				targetRepoSlug = config.TargetRepoSlug
			}
		case "set_issue_type":
			if config := safeOutputs.SetIssueType; config != nil {
				hasAllowedRepos = len(config.AllowedRepos) > 0
				targetRepoSlug = config.TargetRepoSlug
			}
		}
	}

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
