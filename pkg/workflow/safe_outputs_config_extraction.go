package workflow

// ========================================
// Safe Output Configuration Extraction
// ========================================
//
// ## Schema Generation Architecture
//
// MCP tool schemas for Safe Outputs are managed through a hybrid approach:
//
// ### Static Schemas (30+ built-in safe output types)
// Defined in: pkg/workflow/js/safe_outputs_tools.json
// - Embedded at compile time via //go:embed directive in pkg/workflow/js.go
// - Contains complete MCP tool definitions with inputSchema for all built-in types
// - Examples: create_issue, create_pull_request, add_comment, update_project, etc.
// - Accessed via GetSafeOutputsToolsJSON() function
//
// ### Dynamic Schema Generation (custom safe-jobs)
// Implemented in: pkg/workflow/safe_outputs_config_generation.go
// - generateCustomJobToolDefinition() builds MCP tool schemas from SafeJobConfig
// - Converts job input definitions to JSON Schema format
// - Supports type mapping (string, boolean, number, choice/enum)
// - Enforces required fields and additionalProperties: false
// - Custom job tools are merged with static tools at runtime
//
// ### Schema Filtering
// Implemented in: pkg/workflow/safe_outputs_config_generation.go
// - generateFilteredToolsJSON() filters tools based on enabled safe-outputs
// - Only includes tools that are configured in the workflow frontmatter
// - Reduces MCP gateway overhead by exposing only necessary tools
//
// ### Validation
// Implemented in: pkg/workflow/safe_outputs_tools_schema_test.go
// - TestSafeOutputsToolsJSONCompliesWithMCPSchema validates against MCP spec
// - TestEachToolHasRequiredMCPFields checks name, description, inputSchema
// - TestNoTopLevelOneOfAllOfAnyOf prevents unsupported schema constructs
//
// This architecture ensures schema consistency by:
// 1. Using embedded JSON for static schemas (single source of truth)
// 2. Programmatic generation for dynamic schemas (type-safe)
// 3. Automated validation in CI (regression prevention)
//

// extractSafeOutputsConfig extracts output configuration from frontmatter
func (c *Compiler) extractSafeOutputsConfig(frontmatter map[string]any) *SafeOutputsConfig {
	safeOutputsConfigLog.Print("Extracting safe-outputs configuration from frontmatter")

	var config *SafeOutputsConfig

	if output, exists := frontmatter["safe-outputs"]; exists {
		if outputMap, ok := output.(map[string]any); ok {
			safeOutputsConfigLog.Printf("Processing safe-outputs configuration with %d top-level keys", len(outputMap))
			config = &SafeOutputsConfig{}

			c.extractIssueSafeOutputs(outputMap, config)
			c.extractDiscussionSafeOutputs(outputMap, config)
			c.extractProjectSafeOutputs(outputMap, config)
			c.extractPullRequestSafeOutputs(outputMap, config)
			c.extractPullRequestReviewSafeOutputs(outputMap, config)
			c.extractCommentAndLabelSafeOutputs(outputMap, config)
			c.extractAssignmentSafeOutputs(outputMap, config)
			c.extractSecuritySafeOutputs(outputMap, config)
			c.extractRepositorySafeOutputs(outputMap, config)
			c.extractAgentSignalSafeOutputs(outputMap, config)

			c.extractGlobalConfigFields(outputMap, config)
		}
	}

	// Apply default threat detection whenever safe-outputs are configured and threat-detection
	// is not explicitly disabled. Detection is always on unless threat-detection is false.
	if config != nil && config.ThreatDetection == nil {
		if output, exists := frontmatter["safe-outputs"]; exists {
			if outputMap, ok := output.(map[string]any); ok {
				if _, exists := outputMap["threat-detection"]; !exists {
					// Only apply default if threat-detection key doesn't exist
					safeOutputsConfigLog.Print("Applying default threat-detection configuration")
					config.ThreatDetection = &ThreatDetectionConfig{}
				}
			}
		}
	}

	// Force-disable threat detection when --use-samples is active: the replay driver
	// emits synthetic outputs solely for deterministic end-to-end tests, and running
	// an LLM-backed detection pass would defeat that determinism.
	if config != nil && c.useSamples && config.ThreatDetection != nil {
		safeOutputsConfigLog.Print("Disabling threat-detection because --use-samples is set")
		config.ThreatDetection = nil
	}

	if config != nil {
		safeOutputsConfigLog.Print("Successfully extracted safe-outputs configuration")
	} else {
		safeOutputsConfigLog.Print("No safe-outputs configuration found in frontmatter")
	}

	return config
}

// extractIssueSafeOutputs parses issue-related safe output handlers.
func (c *Compiler) extractIssueSafeOutputs(outputMap map[string]any, config *SafeOutputsConfig) {
	// Handle create-issue
	if issuesConfig := c.parseCreateIssuesConfig(outputMap); issuesConfig != nil {
		safeOutputsConfigLog.Print("Configured create-issue output handler")
		config.CreateIssues = issuesConfig
	}

	// Handle close-issue
	if closeIssuesConfig := c.parseCloseIssuesConfig(outputMap); closeIssuesConfig != nil {
		config.CloseIssues = closeIssuesConfig
	}

	// Handle update-issue
	if updateIssuesConfig := c.parseUpdateIssuesConfig(outputMap); updateIssuesConfig != nil {
		config.UpdateIssues = updateIssuesConfig
	}

	// Handle set-issue-type
	if setIssueTypeConfig := c.parseSetIssueTypeConfig(outputMap); setIssueTypeConfig != nil {
		config.SetIssueType = setIssueTypeConfig
	}

	// Handle set-issue-field
	if setIssueFieldConfig := c.parseSetIssueFieldConfig(outputMap); setIssueFieldConfig != nil {
		config.SetIssueField = setIssueFieldConfig
	}

	// Handle link-sub-issue
	if linkSubIssueConfig := c.parseLinkSubIssueConfig(outputMap); linkSubIssueConfig != nil {
		config.LinkSubIssue = linkSubIssueConfig
	}

	// Parse assign-milestone configuration
	if assignMilestoneConfig := c.parseAssignMilestoneConfig(outputMap); assignMilestoneConfig != nil {
		config.AssignMilestone = assignMilestoneConfig
	}
}

// extractDiscussionSafeOutputs parses discussion-related safe output handlers.
func (c *Compiler) extractDiscussionSafeOutputs(outputMap map[string]any, config *SafeOutputsConfig) {
	// Handle create-discussion
	if discussionsConfig := c.parseCreateDiscussionsConfig(outputMap); discussionsConfig != nil {
		config.CreateDiscussions = discussionsConfig
	}

	// Handle close-discussion
	if closeDiscussionsConfig := c.parseCloseDiscussionsConfig(outputMap); closeDiscussionsConfig != nil {
		config.CloseDiscussions = closeDiscussionsConfig
	}

	// Handle update-discussion
	if updateDiscussionsConfig := c.parseUpdateDiscussionsConfig(outputMap); updateDiscussionsConfig != nil {
		config.UpdateDiscussions = updateDiscussionsConfig
	}
}

// extractProjectSafeOutputs parses project board related safe output handlers.
func (c *Compiler) extractProjectSafeOutputs(outputMap map[string]any, config *SafeOutputsConfig) {
	// Handle update-project (smart project board management)
	if updateProjectConfig := c.parseUpdateProjectConfig(outputMap); updateProjectConfig != nil {
		config.UpdateProjects = updateProjectConfig
	}

	// Handle create-project
	if createProjectConfig := c.parseCreateProjectsConfig(outputMap); createProjectConfig != nil {
		config.CreateProjects = createProjectConfig
	}

	// Handle create-project-status-update (project status updates)
	if createProjectStatusUpdateConfig := c.parseCreateProjectStatusUpdateConfig(outputMap); createProjectStatusUpdateConfig != nil {
		config.CreateProjectStatusUpdates = createProjectStatusUpdateConfig
	}
}

// extractPullRequestSafeOutputs parses pull request lifecycle safe output handlers.
func (c *Compiler) extractPullRequestSafeOutputs(outputMap map[string]any, config *SafeOutputsConfig) {
	// Handle create-pull-request
	if pullRequestsConfig := c.parseCreatePullRequestsConfig(outputMap); pullRequestsConfig != nil {
		safeOutputsConfigLog.Print("Configured create-pull-request output handler")
		config.CreatePullRequests = pullRequestsConfig
	}

	// Handle close-pull-request
	if closePullRequestsConfig := c.parseClosePullRequestsConfig(outputMap); closePullRequestsConfig != nil {
		config.ClosePullRequests = closePullRequestsConfig
	}

	// Handle update-pull-request
	if updatePullRequestsConfig := c.parseUpdatePullRequestsConfig(outputMap); updatePullRequestsConfig != nil {
		config.UpdatePullRequests = updatePullRequestsConfig
	}

	// Handle merge-pull-request
	if mergePullRequestConfig := c.parseMergePullRequestConfig(outputMap); mergePullRequestConfig != nil {
		config.MergePullRequest = mergePullRequestConfig
	}

	// Handle mark-pull-request-as-ready-for-review
	if markPRReadyConfig := c.parseMarkPullRequestAsReadyForReviewConfig(outputMap); markPRReadyConfig != nil {
		config.MarkPullRequestAsReadyForReview = markPRReadyConfig
	}

	// Handle push-to-pull-request-branch
	if pushToBranchConfig := c.parsePushToPullRequestBranchConfig(outputMap); pushToBranchConfig != nil {
		config.PushToPullRequestBranch = pushToBranchConfig
	}
}

// extractPullRequestReviewSafeOutputs parses pull request review safe output handlers.
func (c *Compiler) extractPullRequestReviewSafeOutputs(outputMap map[string]any, config *SafeOutputsConfig) {
	// Handle create-pull-request-review-comment
	if prReviewCommentsConfig := c.parsePullRequestReviewCommentsConfig(outputMap); prReviewCommentsConfig != nil {
		config.CreatePullRequestReviewComments = prReviewCommentsConfig
	}

	// Handle submit-pull-request-review
	if submitPRReviewConfig := c.parseSubmitPullRequestReviewConfig(outputMap); submitPRReviewConfig != nil {
		config.SubmitPullRequestReview = submitPRReviewConfig
	}

	// Handle reply-to-pull-request-review-comment
	if replyToPRReviewCommentConfig := c.parseReplyToPullRequestReviewCommentConfig(outputMap); replyToPRReviewCommentConfig != nil {
		config.ReplyToPullRequestReviewComment = replyToPRReviewCommentConfig
	}

	// Handle resolve-pull-request-review-thread
	if resolvePRReviewThreadConfig := c.parseResolvePullRequestReviewThreadConfig(outputMap); resolvePRReviewThreadConfig != nil {
		config.ResolvePullRequestReviewThread = resolvePRReviewThreadConfig
	}

	// Handle dismiss-pull-request-review (and dismiss-review alias)
	if dismissPRReviewConfig := c.parseDismissPullRequestReviewConfig(outputMap); dismissPRReviewConfig != nil {
		config.DismissPullRequestReview = dismissPRReviewConfig
	}

	// Parse add-reviewer configuration
	if addReviewerConfig := c.parseAddReviewerConfig(outputMap); addReviewerConfig != nil {
		config.AddReviewer = addReviewerConfig
	}
}

// extractCommentAndLabelSafeOutputs parses comment and label safe output handlers.
func (c *Compiler) extractCommentAndLabelSafeOutputs(outputMap map[string]any, config *SafeOutputsConfig) {
	// Handle add-comment
	if commentsConfig := c.parseCommentsConfig(outputMap); commentsConfig != nil {
		config.AddComments = commentsConfig
	}

	// Handle hide-comment
	if hideCommentConfig := c.parseHideCommentConfig(outputMap); hideCommentConfig != nil {
		config.HideComment = hideCommentConfig
	}

	// Parse add-labels configuration
	if addLabelsConfig := c.parseAddLabelsConfig(outputMap); addLabelsConfig != nil {
		config.AddLabels = addLabelsConfig
	}

	// Parse remove-labels configuration
	if removeLabelsConfig := c.parseRemoveLabelsConfig(outputMap); removeLabelsConfig != nil {
		config.RemoveLabels = removeLabelsConfig
	}

	// Parse replace-label configuration
	if replaceLabelConfig := c.parseReplaceLabelConfig(outputMap); replaceLabelConfig != nil {
		config.ReplaceLabel = replaceLabelConfig
	}
}

// extractAssignmentSafeOutputs parses assignment and agent session safe output handlers.
func (c *Compiler) extractAssignmentSafeOutputs(outputMap map[string]any, config *SafeOutputsConfig) {
	// Handle create-agent-session
	if agentSessionConfig := c.parseAgentSessionConfig(outputMap); agentSessionConfig != nil {
		config.CreateAgentSessions = agentSessionConfig
	}

	// Handle assign-to-agent
	if assignToAgentConfig := c.parseAssignToAgentConfig(outputMap); assignToAgentConfig != nil {
		config.AssignToAgent = assignToAgentConfig
	}

	// Handle assign-to-user
	if assignToUserConfig := c.parseAssignToUserConfig(outputMap); assignToUserConfig != nil {
		config.AssignToUser = assignToUserConfig
	}

	// Handle unassign-from-user
	if unassignFromUserConfig := c.parseUnassignFromUserConfig(outputMap); unassignFromUserConfig != nil {
		config.UnassignFromUser = unassignFromUserConfig
	}
}

// extractSecuritySafeOutputs parses code scanning, check run, and workflow approval handlers.
func (c *Compiler) extractSecuritySafeOutputs(outputMap map[string]any, config *SafeOutputsConfig) {
	// Handle create-code-scanning-alert
	if securityReportsConfig := c.parseCodeScanningAlertsConfig(outputMap); securityReportsConfig != nil {
		config.CreateCodeScanningAlerts = securityReportsConfig
	}

	// Handle autofix-code-scanning-alert
	if autofixCodeScanningAlertConfig := c.parseAutofixCodeScanningAlertConfig(outputMap); autofixCodeScanningAlertConfig != nil {
		config.AutofixCodeScanningAlert = autofixCodeScanningAlertConfig
	}

	// Handle create-check-run
	if createCheckRunConfig := c.parseCreateCheckRunConfig(outputMap); createCheckRunConfig != nil {
		config.CreateCheckRun = createCheckRunConfig
	}

	// Handle approve-workflow-run
	if approveWorkflowRunConfig := c.parseApproveWorkflowRunConfig(outputMap); approveWorkflowRunConfig != nil {
		config.ApproveWorkflowRun = approveWorkflowRunConfig
	}
}

// extractRepositorySafeOutputs parses asset, release, and dispatch safe output handlers.
func (c *Compiler) extractRepositorySafeOutputs(outputMap map[string]any, config *SafeOutputsConfig) {
	// Handle upload-asset
	if uploadAssetsConfig := c.parseUploadAssetConfig(outputMap); uploadAssetsConfig != nil {
		config.UploadAssets = uploadAssetsConfig
	}

	// Handle upload-artifact
	if uploadArtifactConfig := c.parseUploadArtifactConfig(outputMap); uploadArtifactConfig != nil {
		config.UploadArtifact = uploadArtifactConfig
	}

	// Handle update-release
	if updateReleaseConfig := c.parseUpdateReleaseConfig(outputMap); updateReleaseConfig != nil {
		config.UpdateRelease = updateReleaseConfig
	}

	// Handle dispatch-workflow
	if dispatchWorkflowConfig := c.parseDispatchWorkflowConfig(outputMap); dispatchWorkflowConfig != nil {
		config.DispatchWorkflow = dispatchWorkflowConfig
	}

	// Handle dispatch_repository
	if dispatchRepositoryConfig := c.parseDispatchRepositoryConfig(outputMap); dispatchRepositoryConfig != nil {
		config.DispatchRepository = dispatchRepositoryConfig
	}

	// Handle call-workflow
	if callWorkflowConfig := c.parseCallWorkflowConfig(outputMap); callWorkflowConfig != nil {
		config.CallWorkflow = callWorkflowConfig
	}
}

// extractAgentSignalSafeOutputs parses the agent signalling handlers (missing-tool,
// missing-data, noop, report-incomplete), which are enabled by default when
// safe-outputs is configured and the key was not explicitly provided.
func (c *Compiler) extractAgentSignalSafeOutputs(outputMap map[string]any, config *SafeOutputsConfig) {
	// Handle missing-tool (parse configuration if present, or enable by default)
	if missingToolConfig := c.parseMissingToolConfig(outputMap); missingToolConfig != nil {
		config.MissingTool = missingToolConfig
	} else if _, exists := outputMap["missing-tool"]; !exists {
		// Enable missing-tool by default if safe-outputs exists and it wasn't explicitly disabled
		trueVal := "true"
		config.MissingTool = &MissingToolConfig{
			CreateIssue: &trueVal,
			TitlePrefix: "",
			Labels:      nil,
		}
	}

	// Handle missing-data (parse configuration if present, or enable by default)
	if missingDataConfig := c.parseMissingDataConfig(outputMap); missingDataConfig != nil {
		config.MissingData = missingDataConfig
	} else if _, exists := outputMap["missing-data"]; !exists {
		// Enable missing-data by default if safe-outputs exists and it wasn't explicitly disabled
		trueVal := "true"
		config.MissingData = &MissingDataConfig{
			CreateIssue: &trueVal,
			TitlePrefix: "",
			Labels:      nil,
		}
	}

	// Handle noop (parse configuration if present, or enable by default as fallback)
	if noopConfig := c.parseNoOpConfig(outputMap); noopConfig != nil {
		config.NoOp = noopConfig
	} else if _, exists := outputMap["noop"]; !exists {
		// Enable noop by default if safe-outputs exists and it wasn't explicitly disabled
		// This ensures there's always a fallback for transparency
		config.NoOp = &NoOpConfig{}
		config.NoOp.Max = defaultIntStr(1) // Default max
		// Implicit noop is for transparency logging only; it must not create
		// issues without a maintenance workflow to expire them, so report-as-issue
		// defaults to false here (users can opt in with an explicit noop: block).
		falseVal := "false"
		config.NoOp.ReportAsIssue = &falseVal
		config.NoOp.Implicit = true // Not authored by the user
	}

	// Handle report-incomplete (parse configuration if present, or enable by default)
	if reportIncompleteConfig := c.parseReportIncompleteConfig(outputMap); reportIncompleteConfig != nil {
		config.ReportIncomplete = reportIncompleteConfig
	} else if _, exists := outputMap["report-incomplete"]; !exists {
		// Enable report-incomplete by default if safe-outputs exists and it wasn't explicitly disabled.
		// This ensures agents always have a first-class channel to signal task incompletion.
		trueVal := "true"
		config.ReportIncomplete = &ReportIncompleteConfig{
			CreateIssue: &trueVal,
			TitlePrefix: "",
			Labels:      nil,
		}
	}
}
