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
			c.applyIssueAndCommentOutputs(outputMap, config)
			c.applyProjectAndLabelOutputs(outputMap, config)
			c.applyUpdateAndUploadOutputs(outputMap, config)
			c.applyDefaultOutputHandlers(outputMap, config)
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

// applyIssueAndCommentOutputs parses issue, discussion, comment, and PR output handlers.
func (c *Compiler) applyIssueAndCommentOutputs(outputMap map[string]any, config *SafeOutputsConfig) {
	if v := c.parseCreateIssuesConfig(outputMap); v != nil {
		safeOutputsConfigLog.Print("Configured create-issue output handler")
		config.CreateIssues = v
	}
	if v := c.parseAgentSessionConfig(outputMap); v != nil {
		config.CreateAgentSessions = v
	}
	if v := c.parseCreateDiscussionsConfig(outputMap); v != nil {
		config.CreateDiscussions = v
	}
	if v := c.parseCloseDiscussionsConfig(outputMap); v != nil {
		config.CloseDiscussions = v
	}
	if v := c.parseCloseIssuesConfig(outputMap); v != nil {
		config.CloseIssues = v
	}
	if v := c.parseClosePullRequestsConfig(outputMap); v != nil {
		config.ClosePullRequests = v
	}
	if v := c.parseMarkPullRequestAsReadyForReviewConfig(outputMap); v != nil {
		config.MarkPullRequestAsReadyForReview = v
	}
	if v := c.parseDismissPullRequestReviewConfig(outputMap); v != nil {
		config.DismissPullRequestReview = v
	}
	if v := c.parseCommentsConfig(outputMap); v != nil {
		config.AddComments = v
	}
	if v := c.parseCreatePullRequestsConfig(outputMap); v != nil {
		safeOutputsConfigLog.Print("Configured create-pull-request output handler")
		config.CreatePullRequests = v
	}
	if v := c.parsePullRequestReviewCommentsConfig(outputMap); v != nil {
		config.CreatePullRequestReviewComments = v
	}
	if v := c.parseSubmitPullRequestReviewConfig(outputMap); v != nil {
		config.SubmitPullRequestReview = v
	}
	if v := c.parseReplyToPullRequestReviewCommentConfig(outputMap); v != nil {
		config.ReplyToPullRequestReviewComment = v
	}
	if v := c.parseResolvePullRequestReviewThreadConfig(outputMap); v != nil {
		config.ResolvePullRequestReviewThread = v
	}
	if v := c.parseCodeScanningAlertsConfig(outputMap); v != nil {
		config.CreateCodeScanningAlerts = v
	}
	if v := c.parseAutofixCodeScanningAlertConfig(outputMap); v != nil {
		config.AutofixCodeScanningAlert = v
	}
	if v := c.parseCreateCheckRunConfig(outputMap); v != nil {
		config.CreateCheckRun = v
	}
	if v := c.parseHideCommentConfig(outputMap); v != nil {
		config.HideComment = v
	}
}

// applyProjectAndLabelOutputs parses project, label, assignment, and identity output handlers.
func (c *Compiler) applyProjectAndLabelOutputs(outputMap map[string]any, config *SafeOutputsConfig) {
	if v := c.parseUpdateProjectConfig(outputMap); v != nil {
		config.UpdateProjects = v
	}
	if v := c.parseCreateProjectsConfig(outputMap); v != nil {
		config.CreateProjects = v
	}
	if v := c.parseCreateProjectStatusUpdateConfig(outputMap); v != nil {
		config.CreateProjectStatusUpdates = v
	}
	if v := c.parseAddLabelsConfig(outputMap); v != nil {
		config.AddLabels = v
	}
	if v := c.parseRemoveLabelsConfig(outputMap); v != nil {
		config.RemoveLabels = v
	}
	if v := c.parseReplaceLabelConfig(outputMap); v != nil {
		config.ReplaceLabel = v
	}
	if v := c.parseAddReviewerConfig(outputMap); v != nil {
		config.AddReviewer = v
	}
	if v := c.parseAssignMilestoneConfig(outputMap); v != nil {
		config.AssignMilestone = v
	}
	if v := c.parseAssignToAgentConfig(outputMap); v != nil {
		config.AssignToAgent = v
	}
	if v := c.parseAssignToUserConfig(outputMap); v != nil {
		config.AssignToUser = v
	}
	if v := c.parseUnassignFromUserConfig(outputMap); v != nil {
		config.UnassignFromUser = v
	}
	if v := c.parseSetIssueTypeConfig(outputMap); v != nil {
		config.SetIssueType = v
	}
	if v := c.parseSetIssueFieldConfig(outputMap); v != nil {
		config.SetIssueField = v
	}
}

// applyUpdateAndUploadOutputs parses update, upload, dispatch, and workflow call handlers.
func (c *Compiler) applyUpdateAndUploadOutputs(outputMap map[string]any, config *SafeOutputsConfig) {
	if v := c.parseUpdateIssuesConfig(outputMap); v != nil {
		config.UpdateIssues = v
	}
	if v := c.parseUpdateDiscussionsConfig(outputMap); v != nil {
		config.UpdateDiscussions = v
	}
	if v := c.parseUpdatePullRequestsConfig(outputMap); v != nil {
		config.UpdatePullRequests = v
	}
	if v := c.parseMergePullRequestConfig(outputMap); v != nil {
		config.MergePullRequest = v
	}
	if v := c.parsePushToPullRequestBranchConfig(outputMap); v != nil {
		config.PushToPullRequestBranch = v
	}
	if v := c.parseUploadAssetConfig(outputMap); v != nil {
		config.UploadAssets = v
	}
	if v := c.parseUploadArtifactConfig(outputMap); v != nil {
		config.UploadArtifact = v
	}
	if v := c.parseUpdateReleaseConfig(outputMap); v != nil {
		config.UpdateRelease = v
	}
	if v := c.parseLinkSubIssueConfig(outputMap); v != nil {
		config.LinkSubIssue = v
	}
	if v := c.parseDispatchWorkflowConfig(outputMap); v != nil {
		config.DispatchWorkflow = v
	}
	if v := c.parseDispatchRepositoryConfig(outputMap); v != nil {
		config.DispatchRepository = v
	}
	if v := c.parseCallWorkflowConfig(outputMap); v != nil {
		config.CallWorkflow = v
	}
}

// applyDefaultOutputHandlers applies defaults for missing-tool, missing-data, noop, and report-incomplete.
func (c *Compiler) applyDefaultOutputHandlers(outputMap map[string]any, config *SafeOutputsConfig) {
	// Handle missing-tool (parse configuration if present, or enable by default)
	if v := c.parseMissingToolConfig(outputMap); v != nil {
		config.MissingTool = v
	} else if _, exists := outputMap["missing-tool"]; !exists {
		trueVal := "true"
		config.MissingTool = &MissingToolConfig{CreateIssue: &trueVal}
	}

	// Handle missing-data (parse configuration if present, or enable by default)
	if v := c.parseMissingDataConfig(outputMap); v != nil {
		config.MissingData = v
	} else if _, exists := outputMap["missing-data"]; !exists {
		trueVal := "true"
		config.MissingData = &MissingDataConfig{CreateIssue: &trueVal}
	}

	// Handle noop (parse configuration if present, or enable by default as fallback)
	if v := c.parseNoOpConfig(outputMap); v != nil {
		config.NoOp = v
	} else if _, exists := outputMap["noop"]; !exists {
		trueVal := "true"
		noopMax := defaultIntStr(1)
		config.NoOp = &NoOpConfig{ReportAsIssue: &trueVal}
		config.NoOp.Max = noopMax
	}

	// Handle report-incomplete (parse configuration if present, or enable by default)
	if v := c.parseReportIncompleteConfig(outputMap); v != nil {
		config.ReportIncomplete = v
	} else if _, exists := outputMap["report-incomplete"]; !exists {
		trueVal := "true"
		config.ReportIncomplete = &ReportIncompleteConfig{CreateIssue: &trueVal}
	}
}
