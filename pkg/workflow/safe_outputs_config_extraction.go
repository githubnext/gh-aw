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
			c.extractSafeOutputHandlerConfigs(outputMap, config)
			c.extractDefaultSafeOutputHandlerConfigs(outputMap, config)
			c.extractGlobalConfigFields(outputMap, config)
		}
	}

	c.applyThreatDetectionDefaults(frontmatter, config)

	if config != nil {
		safeOutputsConfigLog.Print("Successfully extracted safe-outputs configuration")
	} else {
		safeOutputsConfigLog.Print("No safe-outputs configuration found in frontmatter")
	}

	return config
}

func (c *Compiler) extractSafeOutputHandlerConfigs(outputMap map[string]any, config *SafeOutputsConfig) {
	c.extractSafeOutputHandlerConfigsIssuesAndPRs(outputMap, config)
	c.extractSafeOutputHandlerConfigsSecurityAndLabels(outputMap, config)
	c.extractSafeOutputHandlerConfigsAssigneesAndUpdates(outputMap, config)
	c.extractSafeOutputHandlerConfigsAssetsAndWorkflow(outputMap, config)
}

func (c *Compiler) extractSafeOutputHandlerConfigsIssuesAndPRs(outputMap map[string]any, config *SafeOutputsConfig) {
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
	if statusUpdateConfig := c.parseCreateProjectStatusUpdateConfig(outputMap); statusUpdateConfig != nil {
		config.CreateProjectStatusUpdates = statusUpdateConfig
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
	if dismissPRReviewConfig := c.parseDismissPullRequestReviewConfig(outputMap); dismissPRReviewConfig != nil {
		config.DismissPullRequestReview = dismissPRReviewConfig
	}
	if commentsConfig := c.parseCommentsConfig(outputMap); commentsConfig != nil {
		config.AddComments = commentsConfig
	}
}

func (c *Compiler) extractSafeOutputHandlerConfigsSecurityAndLabels(outputMap map[string]any, config *SafeOutputsConfig) {
	if pullRequestsConfig := c.parseCreatePullRequestsConfig(outputMap); pullRequestsConfig != nil {
		safeOutputsConfigLog.Print("Configured create-pull-request output handler")
		config.CreatePullRequests = pullRequestsConfig
	}
	if prReviewCommentsConfig := c.parsePullRequestReviewCommentsConfig(outputMap); prReviewCommentsConfig != nil {
		config.CreatePullRequestReviewComments = prReviewCommentsConfig
	}
	if submitPRReviewConfig := c.parseSubmitPullRequestReviewConfig(outputMap); submitPRReviewConfig != nil {
		config.SubmitPullRequestReview = submitPRReviewConfig
	}
	if replyConfig := c.parseReplyToPullRequestReviewCommentConfig(outputMap); replyConfig != nil {
		config.ReplyToPullRequestReviewComment = replyConfig
	}
	if resolveConfig := c.parseResolvePullRequestReviewThreadConfig(outputMap); resolveConfig != nil {
		config.ResolvePullRequestReviewThread = resolveConfig
	}
	if securityReportsConfig := c.parseCodeScanningAlertsConfig(outputMap); securityReportsConfig != nil {
		config.CreateCodeScanningAlerts = securityReportsConfig
	}
	if autofixConfig := c.parseAutofixCodeScanningAlertConfig(outputMap); autofixConfig != nil {
		config.AutofixCodeScanningAlert = autofixConfig
	}
	if createCheckRunConfig := c.parseCreateCheckRunConfig(outputMap); createCheckRunConfig != nil {
		config.CreateCheckRun = createCheckRunConfig
	}
	if addLabelsConfig := c.parseAddLabelsConfig(outputMap); addLabelsConfig != nil {
		config.AddLabels = addLabelsConfig
	}
	if removeLabelsConfig := c.parseRemoveLabelsConfig(outputMap); removeLabelsConfig != nil {
		config.RemoveLabels = removeLabelsConfig
	}
	if replaceLabelConfig := c.parseReplaceLabelConfig(outputMap); replaceLabelConfig != nil {
		config.ReplaceLabel = replaceLabelConfig
	}
}

func (c *Compiler) extractSafeOutputHandlerConfigsAssigneesAndUpdates(outputMap map[string]any, config *SafeOutputsConfig) {
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
}

func (c *Compiler) extractSafeOutputHandlerConfigsAssetsAndWorkflow(outputMap map[string]any, config *SafeOutputsConfig) {
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

func (c *Compiler) extractDefaultSafeOutputHandlerConfigs(outputMap map[string]any, config *SafeOutputsConfig) {
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
	c.extractDefaultNoOpConfig(outputMap, config)
	c.extractDefaultReportIncompleteConfig(outputMap, config)
}

func (c *Compiler) extractDefaultNoOpConfig(outputMap map[string]any, config *SafeOutputsConfig) {
	if noopConfig := c.parseNoOpConfig(outputMap); noopConfig != nil {
		config.NoOp = noopConfig
	} else if _, exists := outputMap["noop"]; !exists {
		config.NoOp = &NoOpConfig{}
		config.NoOp.Max = defaultIntStr(1)
		trueVal := "true"
		config.NoOp.ReportAsIssue = &trueVal
	}
}

func (c *Compiler) extractDefaultReportIncompleteConfig(outputMap map[string]any, config *SafeOutputsConfig) {
	if reportIncompleteConfig := c.parseReportIncompleteConfig(outputMap); reportIncompleteConfig != nil {
		config.ReportIncomplete = reportIncompleteConfig
	} else if _, exists := outputMap["report-incomplete"]; !exists {
		trueVal := "true"
		config.ReportIncomplete = &ReportIncompleteConfig{CreateIssue: &trueVal, TitlePrefix: "", Labels: nil}
	}
}

func (c *Compiler) applyThreatDetectionDefaults(frontmatter map[string]any, config *SafeOutputsConfig) {
	if config != nil && config.ThreatDetection == nil {
		if output, exists := frontmatter["safe-outputs"]; exists {
			if outputMap, ok := output.(map[string]any); ok {
				if _, exists := outputMap["threat-detection"]; !exists {
					safeOutputsConfigLog.Print("Applying default threat-detection configuration")
					config.ThreatDetection = &ThreatDetectionConfig{}
				}
			}
		}
	}
	if config != nil && c.useSamples && config.ThreatDetection != nil {
		safeOutputsConfigLog.Print("Disabling threat-detection because --use-samples is set")
		config.ThreatDetection = nil
	}
}
