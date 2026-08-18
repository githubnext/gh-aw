package workflow

import "fmt"

type repoTargetConfig struct {
	allowedRepos   []string
	targetRepoSlug string
}

type repoTargetAccessor func(*SafeOutputsConfig) *repoTargetConfig

var repoTargetAccessors = map[string]repoTargetAccessor{
	"create_issue": func(config *SafeOutputsConfig) *repoTargetConfig {
		if output := config.CreateIssues; output != nil {
			return &repoTargetConfig{output.AllowedRepos, output.TargetRepoSlug}
		}
		return nil
	},
	"create_discussion": func(config *SafeOutputsConfig) *repoTargetConfig {
		if output := config.CreateDiscussions; output != nil {
			return &repoTargetConfig{output.AllowedRepos, output.TargetRepoSlug}
		}
		return nil
	},
	"add_comment": func(config *SafeOutputsConfig) *repoTargetConfig {
		if output := config.AddComments; output != nil {
			return &repoTargetConfig{output.AllowedRepos, output.TargetRepoSlug}
		}
		return nil
	},
	"create_pull_request": func(config *SafeOutputsConfig) *repoTargetConfig {
		if output := config.CreatePullRequests; output != nil {
			return &repoTargetConfig{output.AllowedRepos, output.TargetRepoSlug}
		}
		return nil
	},
	"create_pull_request_review_comment": func(config *SafeOutputsConfig) *repoTargetConfig {
		if output := config.CreatePullRequestReviewComments; output != nil {
			return &repoTargetConfig{output.AllowedRepos, output.TargetRepoSlug}
		}
		return nil
	},
	"reply_to_pull_request_review_comment": func(config *SafeOutputsConfig) *repoTargetConfig {
		if output := config.ReplyToPullRequestReviewComment; output != nil {
			return &repoTargetConfig{output.AllowedRepos, output.TargetRepoSlug}
		}
		return nil
	},
	"dismiss_pull_request_review": func(config *SafeOutputsConfig) *repoTargetConfig {
		if output := config.DismissPullRequestReview; output != nil {
			return &repoTargetConfig{output.AllowedRepos, output.TargetRepoSlug}
		}
		return nil
	},
	"create_agent_session": func(config *SafeOutputsConfig) *repoTargetConfig {
		if output := config.CreateAgentSessions; output != nil {
			return &repoTargetConfig{output.AllowedRepos, output.TargetRepoSlug}
		}
		return nil
	},
	"close_issue": func(config *SafeOutputsConfig) *repoTargetConfig {
		if output := config.CloseIssues; output != nil {
			return &repoTargetConfig{output.AllowedRepos, output.TargetRepoSlug}
		}
		return nil
	},
	"update_issue": func(config *SafeOutputsConfig) *repoTargetConfig {
		if output := config.UpdateIssues; output != nil {
			return &repoTargetConfig{output.AllowedRepos, output.TargetRepoSlug}
		}
		return nil
	},
	"close_discussion": func(config *SafeOutputsConfig) *repoTargetConfig {
		if output := config.CloseDiscussions; output != nil {
			return &repoTargetConfig{output.AllowedRepos, output.TargetRepoSlug}
		}
		return nil
	},
	"update_discussion": func(config *SafeOutputsConfig) *repoTargetConfig {
		if output := config.UpdateDiscussions; output != nil {
			return &repoTargetConfig{output.AllowedRepos, output.TargetRepoSlug}
		}
		return nil
	},
	"close_pull_request": func(config *SafeOutputsConfig) *repoTargetConfig {
		if output := config.ClosePullRequests; output != nil {
			return &repoTargetConfig{output.AllowedRepos, output.TargetRepoSlug}
		}
		return nil
	},
	"update_pull_request": func(config *SafeOutputsConfig) *repoTargetConfig {
		if output := config.UpdatePullRequests; output != nil {
			return &repoTargetConfig{output.AllowedRepos, output.TargetRepoSlug}
		}
		return nil
	},
	"merge_pull_request": func(config *SafeOutputsConfig) *repoTargetConfig {
		if output := config.MergePullRequest; output != nil {
			return &repoTargetConfig{output.AllowedRepos, output.TargetRepoSlug}
		}
		return nil
	},
	"add_labels": func(config *SafeOutputsConfig) *repoTargetConfig {
		if output := config.AddLabels; output != nil {
			return &repoTargetConfig{output.AllowedRepos, output.TargetRepoSlug}
		}
		return nil
	},
	"remove_labels": func(config *SafeOutputsConfig) *repoTargetConfig {
		if output := config.RemoveLabels; output != nil {
			return &repoTargetConfig{output.AllowedRepos, output.TargetRepoSlug}
		}
		return nil
	},
	"replace_label": func(config *SafeOutputsConfig) *repoTargetConfig {
		if output := config.ReplaceLabel; output != nil {
			return &repoTargetConfig{output.AllowedRepos, output.TargetRepoSlug}
		}
		return nil
	},
	"hide_comment": func(config *SafeOutputsConfig) *repoTargetConfig {
		if output := config.HideComment; output != nil {
			return &repoTargetConfig{output.AllowedRepos, output.TargetRepoSlug}
		}
		return nil
	},
	"link_sub_issue": func(config *SafeOutputsConfig) *repoTargetConfig {
		if output := config.LinkSubIssue; output != nil {
			return &repoTargetConfig{output.AllowedRepos, output.TargetRepoSlug}
		}
		return nil
	},
	"mark_pull_request_as_ready_for_review": func(config *SafeOutputsConfig) *repoTargetConfig {
		if output := config.MarkPullRequestAsReadyForReview; output != nil {
			return &repoTargetConfig{output.AllowedRepos, output.TargetRepoSlug}
		}
		return nil
	},
	"add_reviewer": func(config *SafeOutputsConfig) *repoTargetConfig {
		if output := config.AddReviewer; output != nil {
			return &repoTargetConfig{output.AllowedRepos, output.TargetRepoSlug}
		}
		return nil
	},
	"assign_milestone": func(config *SafeOutputsConfig) *repoTargetConfig {
		if output := config.AssignMilestone; output != nil {
			return &repoTargetConfig{output.AllowedRepos, output.TargetRepoSlug}
		}
		return nil
	},
	"assign_to_agent": func(config *SafeOutputsConfig) *repoTargetConfig {
		if output := config.AssignToAgent; output != nil {
			return &repoTargetConfig{output.AllowedRepos, output.TargetRepoSlug}
		}
		return nil
	},
	"assign_to_user": func(config *SafeOutputsConfig) *repoTargetConfig {
		if output := config.AssignToUser; output != nil {
			return &repoTargetConfig{output.AllowedRepos, output.TargetRepoSlug}
		}
		return nil
	},
	"unassign_from_user": func(config *SafeOutputsConfig) *repoTargetConfig {
		if output := config.UnassignFromUser; output != nil {
			return &repoTargetConfig{output.AllowedRepos, output.TargetRepoSlug}
		}
		return nil
	},
	"set_issue_type": func(config *SafeOutputsConfig) *repoTargetConfig {
		if output := config.SetIssueType; output != nil {
			return &repoTargetConfig{output.AllowedRepos, output.TargetRepoSlug}
		}
		return nil
	},
	"set_issue_field": func(config *SafeOutputsConfig) *repoTargetConfig {
		if output := config.SetIssueField; output != nil {
			return &repoTargetConfig{output.AllowedRepos, output.TargetRepoSlug}
		}
		return nil
	},
}

// addRepoParameterIfNeeded adds a "repo" parameter to the tool's inputSchema
// if the safe output configuration has allowed-repos entries or a wildcard "*" target-repo
func addRepoParameterIfNeeded(tool map[string]any, toolName string, safeOutputs *SafeOutputsConfig) {
	safeOutputsConfigLog.Printf("Checking if repo parameter needed for tool: %s", toolName)
	if safeOutputs == nil {
		return
	}

	accessor, ok := repoTargetAccessors[toolName]
	if !ok {
		return
	}
	targetConfig := accessor(safeOutputs)
	if targetConfig == nil {
		return
	}

	// Only add repo parameter if allowed-repos has entries or target-repo is wildcard ("*")
	if len(targetConfig.allowedRepos) == 0 && targetConfig.targetRepoSlug != "*" {
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
	if targetConfig.targetRepoSlug == "*" {
		repoDescription = "Target repository for this operation in 'owner/repo' format. Any repository can be targeted."
	} else if targetConfig.targetRepoSlug != "" {
		repoDescription = fmt.Sprintf("Target repository for this operation in 'owner/repo' format. Default is %q. Must be the target-repo or in the allowed-repos list.", targetConfig.targetRepoSlug)
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
