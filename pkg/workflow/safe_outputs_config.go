package workflow

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var safeOutputsConfigLog = logger.New("workflow:safe_outputs_config")

// ========================================
// Safe Output Configuration
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

// ========================================
// Safe Output Configuration Extraction
// ========================================

// extractSafeOutputsConfig extracts output configuration from frontmatter
func (c *Compiler) extractSafeOutputsConfig(frontmatter map[string]any) *SafeOutputsConfig {
	safeOutputsConfigLog.Print("Extracting safe-outputs configuration from frontmatter")

	var config *SafeOutputsConfig

	if output, exists := frontmatter["safe-outputs"]; exists {
		if outputMap, ok := output.(map[string]any); ok {
			config = &SafeOutputsConfig{}

			// Handle create-issue
			issuesConfig := c.parseIssuesConfig(outputMap)
			if issuesConfig != nil {
				config.CreateIssues = issuesConfig
			}

			// Handle create-agent-task
			agentTaskConfig := c.parseAgentTaskConfig(outputMap)
			if agentTaskConfig != nil {
				config.CreateAgentTasks = agentTaskConfig
			}

			// Handle update-project (smart project board management)
			updateProjectConfig := c.parseUpdateProjectConfig(outputMap)
			if updateProjectConfig != nil {
				config.UpdateProjects = updateProjectConfig
			}

			// Handle create-discussion
			discussionsConfig := c.parseDiscussionsConfig(outputMap)
			if discussionsConfig != nil {
				config.CreateDiscussions = discussionsConfig
			}

			// Handle close-discussion
			closeDiscussionsConfig := c.parseCloseDiscussionsConfig(outputMap)
			if closeDiscussionsConfig != nil {
				config.CloseDiscussions = closeDiscussionsConfig
			}

			// Handle close-issue
			closeIssuesConfig := c.parseCloseIssuesConfig(outputMap)
			if closeIssuesConfig != nil {
				config.CloseIssues = closeIssuesConfig
			}

			// Handle close-pull-request
			closePullRequestsConfig := c.parseClosePullRequestsConfig(outputMap)
			if closePullRequestsConfig != nil {
				config.ClosePullRequests = closePullRequestsConfig
			}

			// Handle add-comment
			commentsConfig := c.parseCommentsConfig(outputMap)
			if commentsConfig != nil {
				config.AddComments = commentsConfig
			}

			// Handle create-pull-request
			pullRequestsConfig := c.parsePullRequestsConfig(outputMap)
			if pullRequestsConfig != nil {
				config.CreatePullRequests = pullRequestsConfig
			}

			// Handle create-pull-request-review-comment
			prReviewCommentsConfig := c.parsePullRequestReviewCommentsConfig(outputMap)
			if prReviewCommentsConfig != nil {
				config.CreatePullRequestReviewComments = prReviewCommentsConfig
			}

			// Handle create-code-scanning-alert
			securityReportsConfig := c.parseCodeScanningAlertsConfig(outputMap)
			if securityReportsConfig != nil {
				config.CreateCodeScanningAlerts = securityReportsConfig
			}

			// Parse allowed-domains configuration
			if allowedDomains, exists := outputMap["allowed-domains"]; exists {
				if domainsArray, ok := allowedDomains.([]any); ok {
					var domainStrings []string
					for _, domain := range domainsArray {
						if domainStr, ok := domain.(string); ok {
							domainStrings = append(domainStrings, domainStr)
						}
					}
					config.AllowedDomains = domainStrings
				}
			}

			// Parse add-labels configuration
			addLabelsConfig := c.parseAddLabelsConfig(outputMap)
			if addLabelsConfig != nil {
				config.AddLabels = addLabelsConfig
			}

			// Parse add-reviewer configuration
			addReviewerConfig := c.parseAddReviewerConfig(outputMap)
			if addReviewerConfig != nil {
				config.AddReviewer = addReviewerConfig
			}

			// Parse assign-milestone configuration
			assignMilestoneConfig := c.parseAssignMilestoneConfig(outputMap)
			if assignMilestoneConfig != nil {
				config.AssignMilestone = assignMilestoneConfig
			}

			// Handle assign-to-agent
			assignToAgentConfig := c.parseAssignToAgentConfig(outputMap)
			if assignToAgentConfig != nil {
				config.AssignToAgent = assignToAgentConfig
			}

			// Handle assign-to-user
			assignToUserConfig := c.parseAssignToUserConfig(outputMap)
			if assignToUserConfig != nil {
				config.AssignToUser = assignToUserConfig
			}

			// Handle update-issue
			updateIssuesConfig := c.parseUpdateIssuesConfig(outputMap)
			if updateIssuesConfig != nil {
				config.UpdateIssues = updateIssuesConfig
			}

			// Handle update-discussion
			updateDiscussionsConfig := c.parseUpdateDiscussionsConfig(outputMap)
			if updateDiscussionsConfig != nil {
				config.UpdateDiscussions = updateDiscussionsConfig
			}

			// Handle update-pull-request
			updatePullRequestsConfig := c.parseUpdatePullRequestsConfig(outputMap)
			if updatePullRequestsConfig != nil {
				config.UpdatePullRequests = updatePullRequestsConfig
			}

			// Handle push-to-pull-request-branch
			pushToBranchConfig := c.parsePushToPullRequestBranchConfig(outputMap)
			if pushToBranchConfig != nil {
				config.PushToPullRequestBranch = pushToBranchConfig
			}

			// Handle upload-asset
			uploadAssetsConfig := c.parseUploadAssetConfig(outputMap)
			if uploadAssetsConfig != nil {
				config.UploadAssets = uploadAssetsConfig
			}

			// Handle update-release
			updateReleaseConfig := c.parseUpdateReleaseConfig(outputMap)
			if updateReleaseConfig != nil {
				config.UpdateRelease = updateReleaseConfig
			}

			// Handle link-sub-issue
			linkSubIssueConfig := c.parseLinkSubIssueConfig(outputMap)
			if linkSubIssueConfig != nil {
				config.LinkSubIssue = linkSubIssueConfig
			}

			// Handle hide-comment
			hideCommentConfig := c.parseHideCommentConfig(outputMap)
			if hideCommentConfig != nil {
				config.HideComment = hideCommentConfig
			}

			// Handle missing-tool (parse configuration if present, or enable by default)
			missingToolConfig := c.parseMissingToolConfig(outputMap)
			if missingToolConfig != nil {
				config.MissingTool = missingToolConfig
			} else {
				// Enable missing-tool by default if safe-outputs exists and it wasn't explicitly disabled
				if _, exists := outputMap["missing-tool"]; !exists {
					config.MissingTool = &MissingToolConfig{} // Default: enabled with no max limit
				}
			}

			// Handle noop (parse configuration if present, or enable by default as fallback)
			noopConfig := c.parseNoOpConfig(outputMap)
			if noopConfig != nil {
				config.NoOp = noopConfig
			} else {
				// Enable noop by default if safe-outputs exists and it wasn't explicitly disabled
				// This ensures there's always a fallback for transparency
				if _, exists := outputMap["noop"]; !exists {
					config.NoOp = &NoOpConfig{}
					config.NoOp.Max = 1 // Default max
				}
			}

			// Handle staged flag
			if staged, exists := outputMap["staged"]; exists {
				if stagedBool, ok := staged.(bool); ok {
					config.Staged = stagedBool
				}
			}

			// Handle env configuration
			if env, exists := outputMap["env"]; exists {
				if envMap, ok := env.(map[string]any); ok {
					config.Env = make(map[string]string)
					for key, value := range envMap {
						if valueStr, ok := value.(string); ok {
							config.Env[key] = valueStr
						}
					}
				}
			}

			// Handle github-token configuration
			if githubToken, exists := outputMap["github-token"]; exists {
				if githubTokenStr, ok := githubToken.(string); ok {
					config.GitHubToken = githubTokenStr
				}
			}

			// Handle max-patch-size configuration
			if maxPatchSize, exists := outputMap["max-patch-size"]; exists {
				switch v := maxPatchSize.(type) {
				case int:
					if v >= 1 {
						config.MaximumPatchSize = v
					}
				case int64:
					if v >= 1 {
						config.MaximumPatchSize = int(v)
					}
				case uint64:
					if v >= 1 {
						config.MaximumPatchSize = int(v)
					}
				case float64:
					intVal := int(v)
					// Warn if truncation occurs (value has fractional part)
					if v != float64(intVal) {
						safeOutputsConfigLog.Printf("max-patch-size: float value %.2f truncated to integer %d", v, intVal)
					}
					if intVal >= 1 {
						config.MaximumPatchSize = intVal
					}
				}
			}

			// Set default value if not specified or invalid
			if config.MaximumPatchSize == 0 {
				config.MaximumPatchSize = 1024 // Default to 1MB = 1024 KB
			}

			// Handle threat-detection
			threatDetectionConfig := c.parseThreatDetectionConfig(outputMap)
			if threatDetectionConfig != nil {
				config.ThreatDetection = threatDetectionConfig
			}

			// Handle runs-on configuration
			if runsOn, exists := outputMap["runs-on"]; exists {
				if runsOnStr, ok := runsOn.(string); ok {
					config.RunsOn = runsOnStr
				}
			}

			// Handle messages configuration
			if messages, exists := outputMap["messages"]; exists {
				if messagesMap, ok := messages.(map[string]any); ok {
					config.Messages = parseMessagesConfig(messagesMap)
				}
			}

			// Handle mentions configuration
			if mentions, exists := outputMap["mentions"]; exists {
				config.Mentions = parseMentionsConfig(mentions)
			}

			// Handle jobs (safe-jobs moved under safe-outputs)
			if jobs, exists := outputMap["jobs"]; exists {
				if jobsMap, ok := jobs.(map[string]any); ok {
					c := &Compiler{} // Create a temporary compiler instance for parsing
					jobsFrontmatter := map[string]any{"safe-jobs": jobsMap}
					config.Jobs = c.parseSafeJobsConfig(jobsFrontmatter)
				}
			}

			// Handle app configuration for GitHub App token minting
			if app, exists := outputMap["app"]; exists {
				if appMap, ok := app.(map[string]any); ok {
					config.App = parseAppConfig(appMap)
				}
			}
		}
	}

	// Apply default threat detection if safe-outputs are configured but threat-detection is missing
	// Don't apply default if threat-detection was explicitly configured (even if disabled)
	if config != nil && HasSafeOutputsEnabled(config) && config.ThreatDetection == nil {
		if output, exists := frontmatter["safe-outputs"]; exists {
			if outputMap, ok := output.(map[string]any); ok {
				if _, exists := outputMap["threat-detection"]; !exists {
					// Only apply default if threat-detection key doesn't exist
					config.ThreatDetection = &ThreatDetectionConfig{}
				}
			}
		}
	}

	return config
}

// ========================================
// Safe Output Helpers
// ========================================

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

// parseMessagesConfig parses the messages configuration from safe-outputs frontmatter
func parseMessagesConfig(messagesMap map[string]any) *SafeOutputMessagesConfig {
	config := &SafeOutputMessagesConfig{}

	if footer, exists := messagesMap["footer"]; exists {
		if footerStr, ok := footer.(string); ok {
			config.Footer = footerStr
		}
	}

	if footerInstall, exists := messagesMap["footer-install"]; exists {
		if footerInstallStr, ok := footerInstall.(string); ok {
			config.FooterInstall = footerInstallStr
		}
	}

	if stagedTitle, exists := messagesMap["staged-title"]; exists {
		if stagedTitleStr, ok := stagedTitle.(string); ok {
			config.StagedTitle = stagedTitleStr
		}
	}

	if stagedDescription, exists := messagesMap["staged-description"]; exists {
		if stagedDescriptionStr, ok := stagedDescription.(string); ok {
			config.StagedDescription = stagedDescriptionStr
		}
	}

	if runStarted, exists := messagesMap["run-started"]; exists {
		if runStartedStr, ok := runStarted.(string); ok {
			config.RunStarted = runStartedStr
		}
	}

	if runSuccess, exists := messagesMap["run-success"]; exists {
		if runSuccessStr, ok := runSuccess.(string); ok {
			config.RunSuccess = runSuccessStr
		}
	}

	if runFailure, exists := messagesMap["run-failure"]; exists {
		if runFailureStr, ok := runFailure.(string); ok {
			config.RunFailure = runFailureStr
		}
	}

	return config
}

// parseMentionsConfig parses the mentions configuration from safe-outputs frontmatter
// Mentions can be:
// - false: always escapes mentions
// - true: always allows mentions (error in strict mode)
// - object: detailed configuration with allow-team-members, allow-context, allowed, max
func parseMentionsConfig(mentions any) *MentionsConfig {
	config := &MentionsConfig{}

	// Handle boolean value
	if boolVal, ok := mentions.(bool); ok {
		config.Enabled = &boolVal
		return config
	}

	// Handle object configuration
	if mentionsMap, ok := mentions.(map[string]any); ok {
		// Parse allow-team-members
		if allowTeamMembers, exists := mentionsMap["allow-team-members"]; exists {
			if val, ok := allowTeamMembers.(bool); ok {
				config.AllowTeamMembers = &val
			}
		}

		// Parse allow-context
		if allowContext, exists := mentionsMap["allow-context"]; exists {
			if val, ok := allowContext.(bool); ok {
				config.AllowContext = &val
			}
		}

		// Parse allowed list
		if allowed, exists := mentionsMap["allowed"]; exists {
			if allowedArray, ok := allowed.([]any); ok {
				var allowedStrings []string
				for _, item := range allowedArray {
					if str, ok := item.(string); ok {
						allowedStrings = append(allowedStrings, str)
					}
				}
				config.Allowed = allowedStrings
			}
		}

		// Parse max
		if maxVal, exists := mentionsMap["max"]; exists {
			switch v := maxVal.(type) {
			case int:
				if v >= 1 {
					config.Max = &v
				}
			case int64:
				intVal := int(v)
				if intVal >= 1 {
					config.Max = &intVal
				}
			case uint64:
				intVal := int(v)
				if intVal >= 1 {
					config.Max = &intVal
				}
			case float64:
				intVal := int(v)
				// Warn if truncation occurs
				if v != float64(intVal) {
					safeOutputsConfigLog.Printf("mentions.max: float value %.2f truncated to integer %d", v, intVal)
				}
				if intVal >= 1 {
					config.Max = &intVal
				}
			}
		}
	}

	return config
}

// serializeMessagesConfig converts SafeOutputMessagesConfig to JSON for passing as environment variable
func serializeMessagesConfig(messages *SafeOutputMessagesConfig) (string, error) {
	if messages == nil {
		return "", nil
	}
	jsonBytes, err := json.Marshal(messages)
	if err != nil {
		return "", fmt.Errorf("failed to serialize messages config: %w", err)
	}
	return string(jsonBytes), nil
}

// generateSafeOutputsConfig generates a JSON configuration for safe outputs
func generateSafeOutputsConfig(data *WorkflowData) string {
	// Pass the safe-outputs configuration for validation
	if data.SafeOutputs == nil {
		return ""
	}
	safeOutputsConfigLog.Print("Generating safe outputs configuration for workflow")
	// Create a simplified config object for validation
	safeOutputsConfig := make(map[string]any)

	// Handle safe-outputs configuration if present
	if data.SafeOutputs != nil {
		if data.SafeOutputs.CreateIssues != nil {
			issueConfig := map[string]any{}
			// Always include max (use configured value or default)
			maxValue := 1 // default
			if data.SafeOutputs.CreateIssues.Max > 0 {
				maxValue = data.SafeOutputs.CreateIssues.Max
			}
			issueConfig["max"] = maxValue
			if len(data.SafeOutputs.CreateIssues.AllowedLabels) > 0 {
				issueConfig["allowed_labels"] = data.SafeOutputs.CreateIssues.AllowedLabels
			}
			safeOutputsConfig["create_issue"] = issueConfig
		}
		if data.SafeOutputs.CreateAgentTasks != nil {
			agentTaskConfig := map[string]any{}
			// Always include max (use configured value or default)
			maxValue := 1 // default
			if data.SafeOutputs.CreateAgentTasks.Max > 0 {
				maxValue = data.SafeOutputs.CreateAgentTasks.Max
			}
			agentTaskConfig["max"] = maxValue
			safeOutputsConfig["create_agent_task"] = agentTaskConfig
		}
		if data.SafeOutputs.AddComments != nil {
			commentConfig := map[string]any{}
			if data.SafeOutputs.AddComments.Target != "" {
				commentConfig["target"] = data.SafeOutputs.AddComments.Target
			}
			// Always include max (use configured value or default)
			maxValue := 1 // default
			if data.SafeOutputs.AddComments.Max > 0 {
				maxValue = data.SafeOutputs.AddComments.Max
			}
			commentConfig["max"] = maxValue
			safeOutputsConfig["add_comment"] = commentConfig
		}
		if data.SafeOutputs.CreateDiscussions != nil {
			discussionConfig := map[string]any{}
			// Always include max (use configured value or default)
			maxValue := 1 // default
			if data.SafeOutputs.CreateDiscussions.Max > 0 {
				maxValue = data.SafeOutputs.CreateDiscussions.Max
			}
			discussionConfig["max"] = maxValue
			if len(data.SafeOutputs.CreateDiscussions.AllowedLabels) > 0 {
				discussionConfig["allowed_labels"] = data.SafeOutputs.CreateDiscussions.AllowedLabels
			}
			safeOutputsConfig["create_discussion"] = discussionConfig
		}
		if data.SafeOutputs.CloseDiscussions != nil {
			closeDiscussionConfig := map[string]any{}
			// Always include max (use configured value or default)
			maxValue := 1 // default
			if data.SafeOutputs.CloseDiscussions.Max > 0 {
				maxValue = data.SafeOutputs.CloseDiscussions.Max
			}
			closeDiscussionConfig["max"] = maxValue
			if data.SafeOutputs.CloseDiscussions.RequiredCategory != "" {
				closeDiscussionConfig["required_category"] = data.SafeOutputs.CloseDiscussions.RequiredCategory
			}
			if len(data.SafeOutputs.CloseDiscussions.RequiredLabels) > 0 {
				closeDiscussionConfig["required_labels"] = data.SafeOutputs.CloseDiscussions.RequiredLabels
			}
			if data.SafeOutputs.CloseDiscussions.RequiredTitlePrefix != "" {
				closeDiscussionConfig["required_title_prefix"] = data.SafeOutputs.CloseDiscussions.RequiredTitlePrefix
			}
			safeOutputsConfig["close_discussion"] = closeDiscussionConfig
		}
		if data.SafeOutputs.CloseIssues != nil {
			closeIssueConfig := map[string]any{}
			// Always include max (use configured value or default)
			maxValue := 1 // default
			if data.SafeOutputs.CloseIssues.Max > 0 {
				maxValue = data.SafeOutputs.CloseIssues.Max
			}
			closeIssueConfig["max"] = maxValue
			if len(data.SafeOutputs.CloseIssues.RequiredLabels) > 0 {
				closeIssueConfig["required_labels"] = data.SafeOutputs.CloseIssues.RequiredLabels
			}
			if data.SafeOutputs.CloseIssues.RequiredTitlePrefix != "" {
				closeIssueConfig["required_title_prefix"] = data.SafeOutputs.CloseIssues.RequiredTitlePrefix
			}
			safeOutputsConfig["close_issue"] = closeIssueConfig
		}
		if data.SafeOutputs.CreatePullRequests != nil {
			prConfig := map[string]any{}
			// Note: max is always 1 for pull requests, not configurable
			if len(data.SafeOutputs.CreatePullRequests.AllowedLabels) > 0 {
				prConfig["allowed_labels"] = data.SafeOutputs.CreatePullRequests.AllowedLabels
			}
			// Pass allow_empty flag to MCP server so it can skip patch generation
			if data.SafeOutputs.CreatePullRequests.AllowEmpty {
				prConfig["allow_empty"] = true
			}
			safeOutputsConfig["create_pull_request"] = prConfig
		}
		if data.SafeOutputs.CreatePullRequestReviewComments != nil {
			prReviewCommentConfig := map[string]any{}
			// Always include max (use configured value or default)
			maxValue := 10 // default
			if data.SafeOutputs.CreatePullRequestReviewComments.Max > 0 {
				maxValue = data.SafeOutputs.CreatePullRequestReviewComments.Max
			}
			prReviewCommentConfig["max"] = maxValue
			safeOutputsConfig["create_pull_request_review_comment"] = prReviewCommentConfig
		}
		if data.SafeOutputs.CreateCodeScanningAlerts != nil {
			// Security reports typically have unlimited max, but check if configured
			securityReportConfig := map[string]any{}
			// Always include max (use configured value or default of 0 for unlimited)
			maxValue := 0 // default: unlimited
			if data.SafeOutputs.CreateCodeScanningAlerts.Max > 0 {
				maxValue = data.SafeOutputs.CreateCodeScanningAlerts.Max
			}
			securityReportConfig["max"] = maxValue
			safeOutputsConfig["create_code_scanning_alert"] = securityReportConfig
		}
		if data.SafeOutputs.AddLabels != nil {
			labelConfig := map[string]any{}
			// Always include max (use configured value or default)
			maxValue := 3 // default
			if data.SafeOutputs.AddLabels.Max > 0 {
				maxValue = data.SafeOutputs.AddLabels.Max
			}
			labelConfig["max"] = maxValue
			if len(data.SafeOutputs.AddLabels.Allowed) > 0 {
				labelConfig["allowed"] = data.SafeOutputs.AddLabels.Allowed
			}
			safeOutputsConfig["add_labels"] = labelConfig
		}
		if data.SafeOutputs.AddReviewer != nil {
			reviewerConfig := map[string]any{}
			// Always include max (use configured value or default)
			maxValue := 3 // default
			if data.SafeOutputs.AddReviewer.Max > 0 {
				maxValue = data.SafeOutputs.AddReviewer.Max
			}
			reviewerConfig["max"] = maxValue
			if len(data.SafeOutputs.AddReviewer.Reviewers) > 0 {
				reviewerConfig["reviewers"] = data.SafeOutputs.AddReviewer.Reviewers
			}
			safeOutputsConfig["add_reviewer"] = reviewerConfig
		}
		if data.SafeOutputs.AssignMilestone != nil {
			assignMilestoneConfig := map[string]any{}
			// Always include max (use configured value or default)
			maxValue := 1 // default
			if data.SafeOutputs.AssignMilestone.Max > 0 {
				maxValue = data.SafeOutputs.AssignMilestone.Max
			}
			assignMilestoneConfig["max"] = maxValue
			if len(data.SafeOutputs.AssignMilestone.Allowed) > 0 {
				assignMilestoneConfig["allowed"] = data.SafeOutputs.AssignMilestone.Allowed
			}
			safeOutputsConfig["assign_milestone"] = assignMilestoneConfig
		}
		if data.SafeOutputs.AssignToAgent != nil {
			assignToAgentConfig := map[string]any{}
			if data.SafeOutputs.AssignToAgent.Max > 0 {
				assignToAgentConfig["max"] = data.SafeOutputs.AssignToAgent.Max
			}
			if data.SafeOutputs.AssignToAgent.DefaultAgent != "" {
				assignToAgentConfig["default_agent"] = data.SafeOutputs.AssignToAgent.DefaultAgent
			}
			safeOutputsConfig["assign_to_agent"] = assignToAgentConfig
		}
		if data.SafeOutputs.AssignToUser != nil {
			assignToUserConfig := map[string]any{}
			// Always include max (use configured value or default)
			maxValue := 1 // default
			if data.SafeOutputs.AssignToUser.Max > 0 {
				maxValue = data.SafeOutputs.AssignToUser.Max
			}
			assignToUserConfig["max"] = maxValue
			if len(data.SafeOutputs.AssignToUser.Allowed) > 0 {
				assignToUserConfig["allowed"] = data.SafeOutputs.AssignToUser.Allowed
			}
			safeOutputsConfig["assign_to_user"] = assignToUserConfig
		}
		if data.SafeOutputs.UpdateIssues != nil {
			updateConfig := map[string]any{}
			// Always include max (use configured value or default)
			maxValue := 1 // default
			if data.SafeOutputs.UpdateIssues.Max > 0 {
				maxValue = data.SafeOutputs.UpdateIssues.Max
			}
			updateConfig["max"] = maxValue
			safeOutputsConfig["update_issue"] = updateConfig
		}
		if data.SafeOutputs.UpdateDiscussions != nil {
			updateDiscussionConfig := map[string]any{}
			// Always include max (use configured value or default)
			maxValue := 1 // default
			if data.SafeOutputs.UpdateDiscussions.Max > 0 {
				maxValue = data.SafeOutputs.UpdateDiscussions.Max
			}
			updateDiscussionConfig["max"] = maxValue
			if len(data.SafeOutputs.UpdateDiscussions.AllowedLabels) > 0 {
				updateDiscussionConfig["allowed_labels"] = data.SafeOutputs.UpdateDiscussions.AllowedLabels
			}
			safeOutputsConfig["update_discussion"] = updateDiscussionConfig
		}
		if data.SafeOutputs.UpdatePullRequests != nil {
			updatePRConfig := map[string]any{}
			// Always include max (use configured value or default)
			maxValue := 1 // default
			if data.SafeOutputs.UpdatePullRequests.Max > 0 {
				maxValue = data.SafeOutputs.UpdatePullRequests.Max
			}
			updatePRConfig["max"] = maxValue
			safeOutputsConfig["update_pull_request"] = updatePRConfig
		}
		if data.SafeOutputs.PushToPullRequestBranch != nil {
			pushToBranchConfig := map[string]any{}
			if data.SafeOutputs.PushToPullRequestBranch.Target != "" {
				pushToBranchConfig["target"] = data.SafeOutputs.PushToPullRequestBranch.Target
			}
			// Always include max (use configured value or default of 0 for unlimited)
			maxValue := 0 // default: unlimited
			if data.SafeOutputs.PushToPullRequestBranch.Max > 0 {
				maxValue = data.SafeOutputs.PushToPullRequestBranch.Max
			}
			pushToBranchConfig["max"] = maxValue
			safeOutputsConfig["push_to_pull_request_branch"] = pushToBranchConfig
		}
		if data.SafeOutputs.UploadAssets != nil {
			uploadConfig := map[string]any{}
			// Always include max (use configured value or default of 0 for unlimited)
			maxValue := 0 // default: unlimited
			if data.SafeOutputs.UploadAssets.Max > 0 {
				maxValue = data.SafeOutputs.UploadAssets.Max
			}
			uploadConfig["max"] = maxValue
			safeOutputsConfig["upload_asset"] = uploadConfig
		}
		if data.SafeOutputs.MissingTool != nil {
			missingToolConfig := map[string]any{}
			// Always include max (use configured value or default of 0 for unlimited)
			maxValue := 0 // default: unlimited
			if data.SafeOutputs.MissingTool.Max > 0 {
				maxValue = data.SafeOutputs.MissingTool.Max
			}
			missingToolConfig["max"] = maxValue
			safeOutputsConfig["missing_tool"] = missingToolConfig
		}
		if data.SafeOutputs.UpdateProjects != nil {
			updateProjectConfig := map[string]any{}
			// Always include max (use configured value or default)
			maxValue := 10 // default
			if data.SafeOutputs.UpdateProjects.Max > 0 {
				maxValue = data.SafeOutputs.UpdateProjects.Max
			}
			updateProjectConfig["max"] = maxValue
			safeOutputsConfig["update_project"] = updateProjectConfig
		}
		if data.SafeOutputs.UpdateRelease != nil {
			updateReleaseConfig := map[string]any{}
			// Always include max (use configured value or default)
			maxValue := 1 // default
			if data.SafeOutputs.UpdateRelease.Max > 0 {
				maxValue = data.SafeOutputs.UpdateRelease.Max
			}
			updateReleaseConfig["max"] = maxValue
			safeOutputsConfig["update_release"] = updateReleaseConfig
		}
		if data.SafeOutputs.LinkSubIssue != nil {
			linkSubIssueConfig := map[string]any{}
			// Always include max (use configured value or default)
			maxValue := 5 // default
			if data.SafeOutputs.LinkSubIssue.Max > 0 {
				maxValue = data.SafeOutputs.LinkSubIssue.Max
			}
			linkSubIssueConfig["max"] = maxValue
			safeOutputsConfig["link_sub_issue"] = linkSubIssueConfig
		}
		if data.SafeOutputs.NoOp != nil {
			noopConfig := map[string]any{}
			// Always include max (use configured value or default)
			maxValue := 1 // default
			if data.SafeOutputs.NoOp.Max > 0 {
				maxValue = data.SafeOutputs.NoOp.Max
			}
			noopConfig["max"] = maxValue
			safeOutputsConfig["noop"] = noopConfig
		}
	}

	// Add safe-jobs configuration from SafeOutputs.Jobs
	if len(data.SafeOutputs.Jobs) > 0 {
		for jobName, jobConfig := range data.SafeOutputs.Jobs {
			safeJobConfig := map[string]any{}

			// Add description if present
			if jobConfig.Description != "" {
				safeJobConfig["description"] = jobConfig.Description
			}

			// Add output if present
			if jobConfig.Output != "" {
				safeJobConfig["output"] = jobConfig.Output
			}

			// Add inputs information
			if len(jobConfig.Inputs) > 0 {
				inputsConfig := make(map[string]any)
				for inputName, inputDef := range jobConfig.Inputs {
					inputConfig := map[string]any{
						"type":        inputDef.Type,
						"description": inputDef.Description,
						"required":    inputDef.Required,
					}
					if inputDef.Default != "" {
						inputConfig["default"] = inputDef.Default
					}
					if len(inputDef.Options) > 0 {
						inputConfig["options"] = inputDef.Options
					}
					inputsConfig[inputName] = inputConfig
				}
				safeJobConfig["inputs"] = inputsConfig
			}

			safeOutputsConfig[jobName] = safeJobConfig
		}
	}

	// Add mentions configuration
	if data.SafeOutputs.Mentions != nil {
		mentionsConfig := make(map[string]any)

		// Handle enabled flag (simple boolean mode)
		if data.SafeOutputs.Mentions.Enabled != nil {
			mentionsConfig["enabled"] = *data.SafeOutputs.Mentions.Enabled
		}

		// Handle allow-team-members
		if data.SafeOutputs.Mentions.AllowTeamMembers != nil {
			mentionsConfig["allowTeamMembers"] = *data.SafeOutputs.Mentions.AllowTeamMembers
		}

		// Handle allow-context
		if data.SafeOutputs.Mentions.AllowContext != nil {
			mentionsConfig["allowContext"] = *data.SafeOutputs.Mentions.AllowContext
		}

		// Handle allowed list
		if len(data.SafeOutputs.Mentions.Allowed) > 0 {
			mentionsConfig["allowed"] = data.SafeOutputs.Mentions.Allowed
		}

		// Handle max
		if data.SafeOutputs.Mentions.Max != nil {
			mentionsConfig["max"] = *data.SafeOutputs.Mentions.Max
		}

		// Only add mentions config if it has any fields
		if len(mentionsConfig) > 0 {
			safeOutputsConfig["mentions"] = mentionsConfig
		}
	}

	configJSON, _ := json.Marshal(safeOutputsConfig)
	return string(configJSON)
}

// generateFilteredToolsJSON filters the ALL_TOOLS array based on enabled safe outputs
// Returns a JSON string containing only the tools that are enabled in the workflow
func generateFilteredToolsJSON(data *WorkflowData) (string, error) {
	if data.SafeOutputs == nil {
		return "[]", nil
	}

	safeOutputsConfigLog.Print("Generating filtered tools JSON for workflow")

	// Load the full tools JSON
	allToolsJSON := GetSafeOutputsToolsJSON()

	// Parse the JSON to get all tools
	var allTools []map[string]any
	if err := json.Unmarshal([]byte(allToolsJSON), &allTools); err != nil {
		return "", fmt.Errorf("failed to parse safe outputs tools JSON: %w", err)
	}

	// Create a set of enabled tool names
	enabledTools := make(map[string]bool)

	// Check which safe outputs are enabled and add their corresponding tool names
	if data.SafeOutputs.CreateIssues != nil {
		enabledTools["create_issue"] = true
	}
	if data.SafeOutputs.CreateAgentTasks != nil {
		enabledTools["create_agent_task"] = true
	}
	if data.SafeOutputs.CreateDiscussions != nil {
		enabledTools["create_discussion"] = true
	}
	if data.SafeOutputs.UpdateDiscussions != nil {
		enabledTools["update_discussion"] = true
	}
	if data.SafeOutputs.CloseDiscussions != nil {
		enabledTools["close_discussion"] = true
	}
	if data.SafeOutputs.CloseIssues != nil {
		enabledTools["close_issue"] = true
	}
	if data.SafeOutputs.ClosePullRequests != nil {
		enabledTools["close_pull_request"] = true
	}
	if data.SafeOutputs.AddComments != nil {
		enabledTools["add_comment"] = true
	}
	if data.SafeOutputs.CreatePullRequests != nil {
		enabledTools["create_pull_request"] = true
	}
	if data.SafeOutputs.CreatePullRequestReviewComments != nil {
		enabledTools["create_pull_request_review_comment"] = true
	}
	if data.SafeOutputs.CreateCodeScanningAlerts != nil {
		enabledTools["create_code_scanning_alert"] = true
	}
	if data.SafeOutputs.AddLabels != nil {
		enabledTools["add_labels"] = true
	}
	if data.SafeOutputs.AddReviewer != nil {
		enabledTools["add_reviewer"] = true
	}
	if data.SafeOutputs.AssignMilestone != nil {
		enabledTools["assign_milestone"] = true
	}
	if data.SafeOutputs.AssignToAgent != nil {
		enabledTools["assign_to_agent"] = true
	}
	if data.SafeOutputs.AssignToUser != nil {
		enabledTools["assign_to_user"] = true
	}
	if data.SafeOutputs.UpdateIssues != nil {
		enabledTools["update_issue"] = true
	}
	if data.SafeOutputs.UpdatePullRequests != nil {
		enabledTools["update_pull_request"] = true
	}
	if data.SafeOutputs.PushToPullRequestBranch != nil {
		enabledTools["push_to_pull_request_branch"] = true
	}
	if data.SafeOutputs.UploadAssets != nil {
		enabledTools["upload_asset"] = true
	}
	if data.SafeOutputs.MissingTool != nil {
		enabledTools["missing_tool"] = true
	}
	if data.SafeOutputs.UpdateRelease != nil {
		enabledTools["update_release"] = true
	}
	if data.SafeOutputs.NoOp != nil {
		enabledTools["noop"] = true
	}
	if data.SafeOutputs.LinkSubIssue != nil {
		enabledTools["link_sub_issue"] = true
	}
	if data.SafeOutputs.HideComment != nil {
		enabledTools["hide_comment"] = true
	}
	if data.SafeOutputs.UpdateProjects != nil {
		enabledTools["update_project"] = true
	}

	// Filter tools to only include enabled ones and enhance descriptions
	var filteredTools []map[string]any
	for _, tool := range allTools {
		toolName, ok := tool["name"].(string)
		if !ok {
			continue
		}
		if enabledTools[toolName] {
			// Create a copy of the tool to avoid modifying the original
			enhancedTool := make(map[string]any)
			for k, v := range tool {
				enhancedTool[k] = v
			}

			// Enhance the description with configuration details
			if description, ok := enhancedTool["description"].(string); ok {
				enhancedDescription := enhanceToolDescription(toolName, description, data.SafeOutputs)
				enhancedTool["description"] = enhancedDescription
			}

			filteredTools = append(filteredTools, enhancedTool)
		}
	}

	if safeOutputsConfigLog.Enabled() {
		safeOutputsConfigLog.Printf("Filtered %d tools from %d total tools", len(filteredTools), len(allTools))
	}

	// Marshal the filtered tools back to JSON with indentation for better readability
	// and to reduce merge conflicts in generated lockfiles
	filteredJSON, err := json.MarshalIndent(filteredTools, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal filtered tools: %w", err)
	}

	return string(filteredJSON), nil
}
