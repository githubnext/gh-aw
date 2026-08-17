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

	outputMap, ok := getSafeOutputsMap(frontmatter)
	if !ok {
		safeOutputsConfigLog.Print("No safe-outputs configuration found in frontmatter")
		return nil
	}

	safeOutputsConfigLog.Printf("Processing safe-outputs configuration with %d top-level keys", len(outputMap))
	config := &SafeOutputsConfig{}
	c.extractCoreSafeOutputHandlers(outputMap, config)
	c.extractAdditionalSafeOutputHandlers(outputMap, config)
	c.extractFallbackSafeOutputHandlers(outputMap, config)
	c.extractGlobalConfigFields(outputMap, config)
	applyDefaultThreatDetection(config, outputMap)

	// Force-disable threat detection when --use-samples is active: the replay driver
	// emits synthetic outputs solely for deterministic end-to-end tests, and running
	// an LLM-backed detection pass would defeat that determinism.
	if config != nil && c.useSamples && config.ThreatDetection != nil {
		safeOutputsConfigLog.Print("Disabling threat-detection because --use-samples is set")
		config.ThreatDetection = nil
	}

	safeOutputsConfigLog.Print("Successfully extracted safe-outputs configuration")
	return config
}

func getSafeOutputsMap(frontmatter map[string]any) (map[string]any, bool) {
	output, exists := frontmatter["safe-outputs"]
	if !exists {
		return nil, false
	}
	outputMap, ok := output.(map[string]any)
	if !ok {
		return nil, false
	}
	return outputMap, true
}

func (c *Compiler) extractCoreSafeOutputHandlers(outputMap map[string]any, config *SafeOutputsConfig) {
	if issuesConfig := c.parseCreateIssuesConfig(outputMap); issuesConfig != nil {
		safeOutputsConfigLog.Print("Configured create-issue output handler")
		config.CreateIssues = issuesConfig
	}
	if agentSessionConfig := c.parseAgentSessionConfig(outputMap); agentSessionConfig != nil {
		config.CreateAgentSessions = agentSessionConfig
	}
	if updateProjectConfig := c.parseUpdateProjectConfig(outputMap); updateProjectConfig != nil {
		config.UpdateProjects = updateProjectConfig
	}
	if createProjectConfig := c.parseCreateProjectsConfig(outputMap); createProjectConfig != nil {
		config.CreateProjects = createProjectConfig
	}
	if projectStatusConfig := c.parseCreateProjectStatusUpdateConfig(outputMap); projectStatusConfig != nil {
		config.CreateProjectStatusUpdates = projectStatusConfig
	}
	if discussionsConfig := c.parseCreateDiscussionsConfig(outputMap); discussionsConfig != nil {
		config.CreateDiscussions = discussionsConfig
	}
	if closeDiscussionsConfig := c.parseCloseDiscussionsConfig(outputMap); closeDiscussionsConfig != nil {
		config.CloseDiscussions = closeDiscussionsConfig
	}
	if closeIssuesConfig := c.parseCloseIssuesConfig(outputMap); closeIssuesConfig != nil {
		config.CloseIssues = closeIssuesConfig
	}
	if closePullRequestsConfig := c.parseClosePullRequestsConfig(outputMap); closePullRequestsConfig != nil {
		config.ClosePullRequests = closePullRequestsConfig
	}
	if markPRReadyConfig := c.parseMarkPullRequestAsReadyForReviewConfig(outputMap); markPRReadyConfig != nil {
		config.MarkPullRequestAsReadyForReview = markPRReadyConfig
	}
	if approveWorkflowRunConfig := c.parseApproveWorkflowRunConfig(outputMap); approveWorkflowRunConfig != nil {
		config.ApproveWorkflowRun = approveWorkflowRunConfig
	}
	if dismissPRReviewConfig := c.parseDismissPullRequestReviewConfig(outputMap); dismissPRReviewConfig != nil {
		config.DismissPullRequestReview = dismissPRReviewConfig
	}
	if commentsConfig := c.parseCommentsConfig(outputMap); commentsConfig != nil {
		config.AddComments = commentsConfig
	}
	if pullRequestsConfig := c.parseCreatePullRequestsConfig(outputMap); pullRequestsConfig != nil {
		safeOutputsConfigLog.Print("Configured create-pull-request output handler")
		config.CreatePullRequests = pullRequestsConfig
	}
}

func (c *Compiler) extractAdditionalSafeOutputHandlers(outputMap map[string]any, config *SafeOutputsConfig) {
	c.extractReviewAndSecurityHandlers(outputMap, config)
	c.extractIssueAndPRMutationHandlers(outputMap, config)
	c.extractWorkflowDispatchHandlers(outputMap, config)
}

func (c *Compiler) extractReviewAndSecurityHandlers(outputMap map[string]any, config *SafeOutputsConfig) {
	if prReviewCommentsConfig := c.parsePullRequestReviewCommentsConfig(outputMap); prReviewCommentsConfig != nil {
		config.CreatePullRequestReviewComments = prReviewCommentsConfig
	}
	if submitPRReviewConfig := c.parseSubmitPullRequestReviewConfig(outputMap); submitPRReviewConfig != nil {
		config.SubmitPullRequestReview = submitPRReviewConfig
	}
	if replyToPRReviewCommentConfig := c.parseReplyToPullRequestReviewCommentConfig(outputMap); replyToPRReviewCommentConfig != nil {
		config.ReplyToPullRequestReviewComment = replyToPRReviewCommentConfig
	}
	if resolvePRReviewThreadConfig := c.parseResolvePullRequestReviewThreadConfig(outputMap); resolvePRReviewThreadConfig != nil {
		config.ResolvePullRequestReviewThread = resolvePRReviewThreadConfig
	}
	if securityReportsConfig := c.parseCodeScanningAlertsConfig(outputMap); securityReportsConfig != nil {
		config.CreateCodeScanningAlerts = securityReportsConfig
	}
	if autofixCodeScanningAlertConfig := c.parseAutofixCodeScanningAlertConfig(outputMap); autofixCodeScanningAlertConfig != nil {
		config.AutofixCodeScanningAlert = autofixCodeScanningAlertConfig
	}
	if createCheckRunConfig := c.parseCreateCheckRunConfig(outputMap); createCheckRunConfig != nil {
		config.CreateCheckRun = createCheckRunConfig
	}
}

func (c *Compiler) extractIssueAndPRMutationHandlers(outputMap map[string]any, config *SafeOutputsConfig) {
	if addLabelsConfig := c.parseAddLabelsConfig(outputMap); addLabelsConfig != nil {
		config.AddLabels = addLabelsConfig
	}
	if removeLabelsConfig := c.parseRemoveLabelsConfig(outputMap); removeLabelsConfig != nil {
		config.RemoveLabels = removeLabelsConfig
	}
	if replaceLabelConfig := c.parseReplaceLabelConfig(outputMap); replaceLabelConfig != nil {
		config.ReplaceLabel = replaceLabelConfig
	}
	if addReviewerConfig := c.parseAddReviewerConfig(outputMap); addReviewerConfig != nil {
		config.AddReviewer = addReviewerConfig
	}
	if assignMilestoneConfig := c.parseAssignMilestoneConfig(outputMap); assignMilestoneConfig != nil {
		config.AssignMilestone = assignMilestoneConfig
	}
	if assignToAgentConfig := c.parseAssignToAgentConfig(outputMap); assignToAgentConfig != nil {
		config.AssignToAgent = assignToAgentConfig
	}
	if assignToUserConfig := c.parseAssignToUserConfig(outputMap); assignToUserConfig != nil {
		config.AssignToUser = assignToUserConfig
	}
	if unassignFromUserConfig := c.parseUnassignFromUserConfig(outputMap); unassignFromUserConfig != nil {
		config.UnassignFromUser = unassignFromUserConfig
	}
	if updateIssuesConfig := c.parseUpdateIssuesConfig(outputMap); updateIssuesConfig != nil {
		config.UpdateIssues = updateIssuesConfig
	}
	if updateDiscussionsConfig := c.parseUpdateDiscussionsConfig(outputMap); updateDiscussionsConfig != nil {
		config.UpdateDiscussions = updateDiscussionsConfig
	}
	if updatePullRequestsConfig := c.parseUpdatePullRequestsConfig(outputMap); updatePullRequestsConfig != nil {
		config.UpdatePullRequests = updatePullRequestsConfig
	}
	if mergePullRequestConfig := c.parseMergePullRequestConfig(outputMap); mergePullRequestConfig != nil {
		config.MergePullRequest = mergePullRequestConfig
	}
	if pushToBranchConfig := c.parsePushToPullRequestBranchConfig(outputMap); pushToBranchConfig != nil {
		config.PushToPullRequestBranch = pushToBranchConfig
	}
	if uploadAssetsConfig := c.parseUploadAssetConfig(outputMap); uploadAssetsConfig != nil {
		config.UploadAssets = uploadAssetsConfig
	}
	if uploadArtifactConfig := c.parseUploadArtifactConfig(outputMap); uploadArtifactConfig != nil {
		config.UploadArtifact = uploadArtifactConfig
	}
	if updateReleaseConfig := c.parseUpdateReleaseConfig(outputMap); updateReleaseConfig != nil {
		config.UpdateRelease = updateReleaseConfig
	}
	if linkSubIssueConfig := c.parseLinkSubIssueConfig(outputMap); linkSubIssueConfig != nil {
		config.LinkSubIssue = linkSubIssueConfig
	}
	if hideCommentConfig := c.parseHideCommentConfig(outputMap); hideCommentConfig != nil {
		config.HideComment = hideCommentConfig
	}
	if setIssueTypeConfig := c.parseSetIssueTypeConfig(outputMap); setIssueTypeConfig != nil {
		config.SetIssueType = setIssueTypeConfig
	}
	if setIssueFieldConfig := c.parseSetIssueFieldConfig(outputMap); setIssueFieldConfig != nil {
		config.SetIssueField = setIssueFieldConfig
	}
}

func (c *Compiler) extractWorkflowDispatchHandlers(outputMap map[string]any, config *SafeOutputsConfig) {
	if dispatchWorkflowConfig := c.parseDispatchWorkflowConfig(outputMap); dispatchWorkflowConfig != nil {
		config.DispatchWorkflow = dispatchWorkflowConfig
	}
	if dispatchRepositoryConfig := c.parseDispatchRepositoryConfig(outputMap); dispatchRepositoryConfig != nil {
		config.DispatchRepository = dispatchRepositoryConfig
	}
	if callWorkflowConfig := c.parseCallWorkflowConfig(outputMap); callWorkflowConfig != nil {
		config.CallWorkflow = callWorkflowConfig
	}
}

func (c *Compiler) extractFallbackSafeOutputHandlers(outputMap map[string]any, config *SafeOutputsConfig) {
	if missingToolConfig := c.parseMissingToolConfig(outputMap); missingToolConfig != nil {
		config.MissingTool = missingToolConfig
	} else if _, exists := outputMap["missing-tool"]; !exists {
		trueVal := "true"
		config.MissingTool = &MissingToolConfig{CreateIssue: &trueVal, TitlePrefix: "", Labels: nil}
	}

	if missingDataConfig := c.parseMissingDataConfig(outputMap); missingDataConfig != nil {
		config.MissingData = missingDataConfig
	} else if _, exists := outputMap["missing-data"]; !exists {
		trueVal := "true"
		config.MissingData = &MissingDataConfig{CreateIssue: &trueVal, TitlePrefix: "", Labels: nil}
	}

	if noopConfig := c.parseNoOpConfig(outputMap); noopConfig != nil {
		config.NoOp = noopConfig
	} else if _, exists := outputMap["noop"]; !exists {
		config.NoOp = &NoOpConfig{}
		config.NoOp.Max = defaultIntStr(1) // Default max
		// Implicit noop is for transparency logging only; it must not create
		// issues without a maintenance workflow to expire them, so report-as-issue
		// defaults to false here (users can opt in with an explicit noop: block).
		falseVal := "false"
		config.NoOp.ReportAsIssue = &falseVal
		config.NoOp.Implicit = true // Not authored by the user
	}

	if reportIncompleteConfig := c.parseReportIncompleteConfig(outputMap); reportIncompleteConfig != nil {
		config.ReportIncomplete = reportIncompleteConfig
	} else if _, exists := outputMap["report-incomplete"]; !exists {
		trueVal := "true"
		config.ReportIncomplete = &ReportIncompleteConfig{CreateIssue: &trueVal, TitlePrefix: "", Labels: nil}
	}
}

func applyDefaultThreatDetection(config *SafeOutputsConfig, outputMap map[string]any) {
	if config.ThreatDetection != nil {
		return
	}
	if _, exists := outputMap["threat-detection"]; exists {
		return
	}
	// Only apply default if threat-detection key doesn't exist
	safeOutputsConfigLog.Print("Applying default threat-detection configuration")
	config.ThreatDetection = &ThreatDetectionConfig{}
}
