package workflow

import (
	"math"
	"strings"

	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/typeutil"
)

// applyIssueHandlers applies issue, discussion, project, and comment handler configs.
func (c *Compiler) applyIssueHandlers(outputMap map[string]any, config *SafeOutputsConfig) {
	if v := c.parseCreateIssuesConfig(outputMap); v != nil {
		config.CreateIssues = v
	}
	if v := c.parseAgentSessionConfig(outputMap); v != nil {
		config.CreateAgentSessions = v
	}
	if v := c.parseUpdateProjectConfig(outputMap); v != nil {
		config.UpdateProjects = v
	}
	if v := c.parseCreateProjectsConfig(outputMap); v != nil {
		config.CreateProjects = v
	}
	if v := c.parseCreateProjectStatusUpdateConfig(outputMap); v != nil {
		config.CreateProjectStatusUpdates = v
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
	if v := c.parseCommentsConfig(outputMap); v != nil {
		config.AddComments = v
	}
	if v := c.parseUpdateDiscussionsConfig(outputMap); v != nil {
		config.UpdateDiscussions = v
	}
	if v := c.parseUpdateIssuesConfig(outputMap); v != nil {
		config.UpdateIssues = v
	}
}

// applyPRHandlers applies pull-request-related handler configs.
func (c *Compiler) applyPRHandlers(outputMap map[string]any, config *SafeOutputsConfig) {
	if v := c.parseCreatePullRequestsConfig(outputMap); v != nil {
		config.CreatePullRequests = v
	}
	if v := c.parseClosePullRequestsConfig(outputMap); v != nil {
		config.ClosePullRequests = v
	}
	if v := c.parseMarkPullRequestAsReadyForReviewConfig(outputMap); v != nil {
		config.MarkPullRequestAsReadyForReview = v
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
	if v := c.parseMergePullRequestConfig(outputMap); v != nil {
		config.MergePullRequest = v
	}
	if v := c.parsePushToPullRequestBranchConfig(outputMap); v != nil {
		config.PushToPullRequestBranch = v
	}
	if v := c.parseUpdatePullRequestsConfig(outputMap); v != nil {
		config.UpdatePullRequests = v
	}
}

// applySecurityAndUploadHandlers applies security, upload, and miscellaneous handler configs.
func (c *Compiler) applySecurityAndUploadHandlers(outputMap map[string]any, config *SafeOutputsConfig) {
	if v := c.parseCodeScanningAlertsConfig(outputMap); v != nil {
		config.CreateCodeScanningAlerts = v
	}
	if v := c.parseAutofixCodeScanningAlertConfig(outputMap); v != nil {
		config.AutofixCodeScanningAlert = v
	}
	if v := c.parseCreateCheckRunConfig(outputMap); v != nil {
		config.CreateCheckRun = v
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
	if v := c.parseHideCommentConfig(outputMap); v != nil {
		config.HideComment = v
	}
	if v := c.parseSetIssueTypeConfig(outputMap); v != nil {
		config.SetIssueType = v
	}
	if v := c.parseSetIssueFieldConfig(outputMap); v != nil {
		config.SetIssueField = v
	}
}

// applyLabelAndAssignmentHandlers applies label, assignment, and dispatch handler configs.
func (c *Compiler) applyLabelAndAssignmentHandlers(outputMap map[string]any, config *SafeOutputsConfig) {
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

// applyDefaultFallbackHandlers applies missing-tool, missing-data, noop, and report-incomplete.
func (c *Compiler) applyDefaultFallbackHandlers(outputMap map[string]any, config *SafeOutputsConfig) {
	if v := c.parseMissingToolConfig(outputMap); v != nil {
		config.MissingTool = v
	} else if _, exists := outputMap["missing-tool"]; !exists {
		trueVal := "true"
		config.MissingTool = &MissingToolConfig{CreateIssue: &trueVal}
	}
	if v := c.parseMissingDataConfig(outputMap); v != nil {
		config.MissingData = v
	} else if _, exists := outputMap["missing-data"]; !exists {
		trueVal := "true"
		config.MissingData = &MissingDataConfig{CreateIssue: &trueVal}
	}
	if v := c.parseNoOpConfig(outputMap); v != nil {
		config.NoOp = v
	} else if _, exists := outputMap["noop"]; !exists {
		trueVal := "true"
		config.NoOp = &NoOpConfig{}
		config.NoOp.Max = defaultIntStr(1)
		config.NoOp.ReportAsIssue = &trueVal
	}
	if v := c.parseReportIncompleteConfig(outputMap); v != nil {
		config.ReportIncomplete = v
	} else if _, exists := outputMap["report-incomplete"]; !exists {
		trueVal := "true"
		config.ReportIncomplete = &ReportIncompleteConfig{CreateIssue: &trueVal}
	}
}

// parseSafeOutputsGlobalConfig parses global fields: allowed-domains, URLs, GitHub refs,
// staged, env, github-token.
func (c *Compiler) parseSafeOutputsGlobalConfig(outputMap map[string]any, config *SafeOutputsConfig) {
	if allowedDomains, exists := outputMap["allowed-domains"]; exists {
		if domainsArray, ok := allowedDomains.([]any); ok {
			var domainStrings []string
			for _, domain := range domainsArray {
				if domainStr, ok := domain.(string); ok {
					domainStrings = append(domainStrings, domainStr)
				}
			}
			config.AllowedDomains = domainStrings
			safeOutputsConfigLog.Printf("Configured allowed-domains with %d domain(s)", len(domainStrings))
		}
	}
	if urls, exists := outputMap["urls"]; exists {
		if urlsStr, ok := urls.(string); ok {
			config.URLs = urlsStr
		}
	}
	if allowGitHubRefs, exists := outputMap["allowed-github-references"]; exists {
		if refsArray, ok := allowGitHubRefs.([]any); ok {
			refStrings := []string{}
			for _, ref := range refsArray {
				if refStr, ok := ref.(string); ok {
					refStrings = append(refStrings, refStr)
				}
			}
			config.AllowGitHubReferences = refStrings
		}
	}
	if err := preprocessBoolFieldAsString(outputMap, "staged", safeOutputsConfigLog); err != nil {
		safeOutputsConfigLog.Printf("staged: %v", err)
	} else if staged, exists := outputMap["staged"]; exists {
		if stagedStr, ok := staged.(string); ok && stagedStr != "" {
			value := TemplatableBool(stagedStr)
			config.Staged = &value
		}
	}
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
	if githubToken, exists := outputMap["github-token"]; exists {
		if githubTokenStr, ok := githubToken.(string); ok {
			config.GitHubToken = githubTokenStr
		}
	}
}

// parseMaxPatchSize parses the max-patch-size field and returns the integer value.
func parseMaxPatchSize(outputMap map[string]any) int {
	maxPatchSize, exists := outputMap["max-patch-size"]
	if !exists {
		return 0
	}
	switch v := maxPatchSize.(type) {
	case int:
		if v >= 1 {
			return v
		}
	case int64:
		if v >= 1 {
			return int(v)
		}
	case uint64:
		if v >= 1 {
			return int(v)
		}
	case float64:
		intVal := int(v)
		if v != float64(intVal) {
			safeOutputsConfigLog.Printf("max-patch-size: float value %.2f truncated to integer %d", v, intVal)
		}
		if intVal >= 1 {
			return intVal
		}
	}
	return 0
}

// parseMaxPatchFiles parses the max-patch-files field and returns the integer value.
func parseMaxPatchFiles(outputMap map[string]any) int {
	maxPatchFiles, exists := outputMap["max-patch-files"]
	if !exists {
		return 0
	}
	switch v := maxPatchFiles.(type) {
	case int:
		if v >= 1 {
			return v
		}
	case int64:
		if v >= 1 {
			if v > int64(math.MaxInt) {
				safeOutputsConfigLog.Printf("max-patch-files: int64 value %d exceeds platform int range, clamping to %d", v, math.MaxInt)
				return math.MaxInt
			}
			return int(v)
		}
	case uint64:
		if v >= 1 {
			if v > uint64(math.MaxInt) {
				safeOutputsConfigLog.Printf("max-patch-files: uint64 value %d exceeds platform int range, clamping to %d", v, math.MaxInt)
				return math.MaxInt
			}
			return int(v)
		}
	case float64:
		if v != v || v > float64(math.MaxInt) || v < float64(math.MinInt) {
			safeOutputsConfigLog.Printf("max-patch-files: float value %.2f is out of range, ignoring", v)
			return 0
		}
		intVal := int(v)
		if v != float64(intVal) {
			safeOutputsConfigLog.Printf("max-patch-files: float value %.2f truncated to integer %d", v, intVal)
		}
		if intVal >= 1 {
			return intVal
		}
	}
	return 0
}

// parseTimeoutMinutes parses the timeout-minutes field and returns the integer value.
func parseTimeoutMinutes(outputMap map[string]any) int {
	timeoutMinutes, exists := outputMap["timeout-minutes"]
	if !exists {
		return 0
	}
	switch v := timeoutMinutes.(type) {
	case int:
		if v >= 1 {
			return v
		}
	case int64:
		if v >= 1 {
			if v > int64(math.MaxInt) {
				safeOutputsConfigLog.Printf("timeout-minutes: int64 value %d exceeds platform int range, clamping to %d", v, math.MaxInt)
				return math.MaxInt
			}
			return int(v)
		}
	case uint64:
		if v >= 1 {
			if v > uint64(math.MaxInt) {
				safeOutputsConfigLog.Printf("timeout-minutes: uint64 value %d exceeds platform int range, clamping to %d", v, math.MaxInt)
				return math.MaxInt
			}
			return int(v)
		}
	case float64:
		if v != v || v > float64(math.MaxInt) || v < float64(math.MinInt) {
			safeOutputsConfigLog.Printf("timeout-minutes: float value %.2f is out of range, ignoring", v)
			return 0
		}
		intVal := int(v)
		if v != float64(intVal) {
			safeOutputsConfigLog.Printf("timeout-minutes: float value %.2f truncated to integer %d", v, intVal)
		}
		if intVal >= 1 {
			return intVal
		}
	}
	return 0
}

// parsePatchAndTimeoutSettings parses patch-size, patch-files, threat-detection, runs-on, and timeout.
func (c *Compiler) parsePatchAndTimeoutSettings(outputMap map[string]any, config *SafeOutputsConfig) {
	config.MaximumPatchSize = parseMaxPatchSize(outputMap)
	if config.MaximumPatchSize == 0 {
		config.MaximumPatchSize = 4096
	}
	config.MaximumPatchFiles = parseMaxPatchFiles(outputMap)
	if config.MaximumPatchFiles == 0 {
		config.MaximumPatchFiles = 100
	}
	if v := c.parseThreatDetectionConfig(outputMap); v != nil {
		config.ThreatDetection = v
	}
	if runsOn, exists := outputMap["runs-on"]; exists {
		config.RunsOn = renderRunsOnSnippet(runsOn)
	}
	config.TimeoutMinutes = parseTimeoutMinutes(outputMap)
}

// parseSafeOutputsMessageSettings parses messages, activation-comments, mentions, footer,
// group-reports, report-failure-as-issue, and failure-issue-repo.
func parseSafeOutputsMessageSettings(outputMap map[string]any, config *SafeOutputsConfig) {
	if messages, exists := outputMap["messages"]; exists {
		if messagesMap, ok := messages.(map[string]any); ok {
			config.Messages = parseMessagesConfig(messagesMap)
		}
	}
	if err := preprocessBoolFieldAsString(outputMap, "activation-comments", safeOutputsConfigLog); err != nil {
		safeOutputsConfigLog.Printf("activation-comments: %v", err)
	}
	if activationComments, exists := outputMap["activation-comments"]; exists {
		if activationCommentsStr, ok := activationComments.(string); ok && activationCommentsStr != "" {
			if config.Messages == nil {
				config.Messages = &SafeOutputMessagesConfig{}
			}
			config.Messages.ActivationComments = activationCommentsStr
		}
	}
	if mentions, exists := outputMap["mentions"]; exists {
		config.Mentions = parseMentionsConfig(mentions)
	}
	if footer, exists := outputMap["footer"]; exists {
		if footerBool, ok := footer.(bool); ok {
			config.Footer = &footerBool
		}
	}
	if groupReports, exists := outputMap["group-reports"]; exists {
		if groupReportsBool, ok := groupReports.(bool); ok {
			config.GroupReports = groupReportsBool
		}
	}
	parseReportFailureAsIssueSetting(outputMap, config)
	if failureIssueRepo, exists := outputMap["failure-issue-repo"]; exists {
		if failureIssueRepoStr, ok := failureIssueRepo.(string); ok && failureIssueRepoStr != "" {
			config.FailureIssueRepo = failureIssueRepoStr
		}
	}
}

// parseReportFailureCategories splits a []any of category strings into included and excluded lists.
func parseReportFailureCategories(categoriesList []any) (included, excluded []string) {
	included = make([]string, 0, len(categoriesList))
	excluded = make([]string, 0, len(categoriesList))
	for _, cat := range categoriesList {
		if catStr, ok := cat.(string); ok {
			if category, found := strings.CutPrefix(catStr, "!"); found {
				excluded = append(excluded, category)
			} else {
				included = append(included, catStr)
			}
		}
	}
	return included, excluded
}

// parseReportFailureAsIssueSetting parses the report-failure-as-issue field.
func parseReportFailureAsIssueSetting(outputMap map[string]any, config *SafeOutputsConfig) {
	reportFailureAsIssue, exists := outputMap["report-failure-as-issue"]
	if !exists {
		return
	}
	if categoriesList, ok := reportFailureAsIssue.([]any); ok {
		included, excluded := parseReportFailureCategories(categoriesList)
		config.ReportFailureAsIssue = reportFailureAsIssue
		config.ReportFailureAsIssueCategories = included
		config.ReportFailureAsIssueExcludedCategories = excluded
		if len(included) > 0 && len(excluded) > 0 {
			safeOutputsConfigLog.Printf("Report failure as issue with include filter: %v, exclude filter: %v", included, excluded)
		} else if len(included) > 0 {
			safeOutputsConfigLog.Printf("Report failure as issue with include filter: %v", included)
		} else if len(excluded) > 0 {
			safeOutputsConfigLog.Printf("Report failure as issue with exclude filter: %v", excluded)
		}
		return
	}
	if err := preprocessBoolFieldAsString(outputMap, "report-failure-as-issue", safeOutputsConfigLog); err != nil {
		safeOutputsConfigLog.Printf("Failed to preprocess report-failure-as-issue field: %v (ignoring invalid value and leaving field unset)", err)
		return
	}
	if str, ok := outputMap["report-failure-as-issue"].(string); ok {
		switch str {
		case "true":
			config.ReportFailureAsIssue = true
		case "false":
			config.ReportFailureAsIssue = false
		default:
			config.ReportFailureAsIssue = str
		}
		safeOutputsConfigLog.Printf("Report failure as issue: %v", config.ReportFailureAsIssue)
	} else if v, ok := outputMap["report-failure-as-issue"].(bool); ok {
		config.ReportFailureAsIssue = v
		safeOutputsConfigLog.Printf("Report failure as issue: %t", v)
	}
}

// parseSafeOutputsExtensionSettings parses max-bot-mentions, steps, id-token, concurrency-group,
// needs, and environment override fields.
func parseSafeOutputsExtensionSettings(outputMap map[string]any, config *SafeOutputsConfig) {
	if err := preprocessIntFieldAsString(outputMap, "max-bot-mentions", safeOutputsConfigLog); err != nil {
		safeOutputsConfigLog.Printf("max-bot-mentions: %v", err)
	} else if maxBotMentions, exists := outputMap["max-bot-mentions"]; exists {
		if maxBotMentionsStr, ok := maxBotMentions.(string); ok {
			config.MaxBotMentions = &maxBotMentionsStr
		}
	}
	if steps, exists := outputMap["steps"]; exists {
		if stepsList, ok := steps.([]any); ok {
			config.Steps = stepsList
			safeOutputsConfigLog.Printf("Configured %d user-provided steps for safe-outputs", len(stepsList))
		}
	}
	if idToken, exists := outputMap["id-token"]; exists {
		if idTokenStr, ok := idToken.(string); ok {
			if idTokenStr == "write" || idTokenStr == "none" {
				config.IDToken = &idTokenStr
				safeOutputsConfigLog.Printf("Configured id-token permission override: %s", idTokenStr)
			} else {
				safeOutputsConfigLog.Printf("Warning: unrecognized safe-outputs id-token value %q (expected \"write\" or \"none\"); ignoring", idTokenStr)
			}
		}
	}
	if concurrencyGroup, exists := outputMap["concurrency-group"]; exists {
		if s, ok := concurrencyGroup.(string); ok && s != "" {
			config.ConcurrencyGroup = s
			safeOutputsConfigLog.Printf("Configured concurrency-group for safe-outputs job: %s", s)
		}
	}
	if needsValue, exists := outputMap["needs"]; exists {
		if needsArray, ok := needsValue.([]any); ok {
			for _, need := range needsArray {
				if needStr, ok := need.(string); ok && needStr != "" {
					config.Needs = append(config.Needs, needStr)
				}
			}
			if len(config.Needs) > 0 {
				safeOutputsConfigLog.Printf("Configured %d explicit safe-outputs needs dependency(ies)", len(config.Needs))
			}
		}
	}
}

// parseSafeOutputsJobsAndActions parses environment override, jobs, scripts, actions, and github-app.
func (c *Compiler) parseSafeOutputsJobsAndActions(outputMap map[string]any, config *SafeOutputsConfig) {
	config.Environment = c.extractTopLevelYAMLSection(outputMap, "environment")
	if config.Environment != "" {
		safeOutputsConfigLog.Printf("Configured environment override for safe-outputs job: %s", config.Environment)
	}
	if jobs, exists := outputMap["jobs"]; exists {
		if jobsMap, ok := jobs.(map[string]any); ok {
			tmpCompiler := NewCompiler()
			config.Jobs = tmpCompiler.parseSafeJobsConfig(jobsMap)
		}
	}
	if scripts, exists := outputMap["scripts"]; exists {
		if scriptsMap, ok := scripts.(map[string]any); ok {
			config.Scripts = parseSafeScriptsConfig(scriptsMap)
			safeOutputsConfigLog.Printf("Configured %d custom safe-output script(s)", len(config.Scripts))
		}
	}
	if actions, exists := outputMap["actions"]; exists {
		if actionsMap, ok := actions.(map[string]any); ok {
			config.Actions = parseActionsConfig(actionsMap)
			safeOutputsConfigLog.Printf("Configured %d custom safe-output action(s)", len(config.Actions))
		}
	}
	if app, exists := outputMap["github-app"]; exists {
		if appMap, ok := app.(map[string]any); ok {
			config.GitHubApp = parseAppConfig(appMap)
		}
	}
}

// applyDefaultThreatDetection applies default threat detection if not explicitly disabled.
func (c *Compiler) applyDefaultThreatDetection(frontmatter map[string]any, config *SafeOutputsConfig) {
	if config == nil {
		return
	}
	if config.ThreatDetection == nil {
		if output, exists := frontmatter["safe-outputs"]; exists {
			if outputMap, ok := output.(map[string]any); ok {
				if _, exists := outputMap["threat-detection"]; !exists {
					safeOutputsConfigLog.Print("Applying default threat-detection configuration")
					config.ThreatDetection = &ThreatDetectionConfig{}
				}
			}
		}
	}
	if c.useSamples && config.ThreatDetection != nil {
		safeOutputsConfigLog.Print("Disabling threat-detection because --use-samples is set")
		config.ThreatDetection = nil
	}
}

// resolveDispatchWorkflowSafeOutputs injects workflow_call relay target repo/ref into
// the dispatch_workflow config when not already set.
func resolveDispatchWorkflowSafeOutputs(safeOutputs *SafeOutputsConfig, data *WorkflowData) *SafeOutputsConfig {
	if !hasWorkflowCallTrigger(data.On) || safeOutputs.DispatchWorkflow == nil {
		return safeOutputs
	}
	if safeOutputs.DispatchWorkflow.TargetRepoSlug == "" {
		safeOutputs = safeOutputsWithDispatchTargetRepo(safeOutputs, "${{ needs.activation.outputs.target_repo }}")
		safeOutputsConfigLog.Print("Injecting target_repo into dispatch_workflow config for workflow_call relay")
	}
	if safeOutputs.DispatchWorkflow.TargetRef == "" {
		safeOutputs = safeOutputsWithDispatchTargetRef(safeOutputs, "${{ needs.activation.outputs.target_ref }}")
		safeOutputsConfigLog.Print("Injecting target_ref into dispatch_workflow config for workflow_call relay")
	}
	return safeOutputs
}

// populateHandlerManagerConfig builds the handler config map from the handler registry.
func populateHandlerManagerConfig(safeOutputs *SafeOutputsConfig, data *WorkflowData, manifestFiles, pathPrefixes []string) map[string]any {
	config := make(map[string]any)
	for handlerName, builder := range handlerRegistry {
		handlerConfig := builder(safeOutputs)
		if handlerConfig == nil {
			continue
		}
		injectCurrentCheckoutPatchWorkspacePath(handlerName, handlerConfig, data)
		injectCheckoutMapping(handlerName, handlerConfig, data)
		if _, hasProtected := handlerConfig["protected_files"]; hasProtected {
			excludeFiles := ParseStringArrayFromConfig(handlerConfig, "_protected_files_exclude", nil)
			delete(handlerConfig, "_protected_files_exclude")
			handlerConfig["protected_files"] = sliceutil.Exclude(manifestFiles, excludeFiles...)
			filteredPrefixes := sliceutil.Exclude(pathPrefixes, excludeFiles...)
			if len(filteredPrefixes) > 0 {
				handlerConfig["protected_path_prefixes"] = filteredPrefixes
			} else {
				delete(handlerConfig, "protected_path_prefixes")
			}
			if dotFolderExcludes := getDotFolderExcludes(excludeFiles); len(dotFolderExcludes) > 0 {
				handlerConfig["protected_dot_folder_excludes"] = dotFolderExcludes
			}
		}
		safeOutputsConfigLog.Printf("Adding %s handler configuration", handlerName)
		config[handlerName] = handlerConfig
	}
	return config
}

// parseMaxField parses the max field from a config map into BaseSafeOutputConfig.
func parseMaxField(configMap map[string]any, config *BaseSafeOutputConfig, defaultMax int) {
	if defaultMax > 0 {
		safeOutputsConfigLog.Printf("Setting default max: %d", defaultMax)
		config.Max = defaultIntStr(defaultMax)
	}
	if max, exists := configMap["max"]; exists {
		switch v := max.(type) {
		case string:
			if strings.HasPrefix(v, "${{") && strings.HasSuffix(v, "}}") {
				safeOutputsConfigLog.Printf("Parsed max as GitHub Actions expression: %s", v)
				config.Max = &v
			}
		default:
			if maxInt, ok := typeutil.ParseIntValue(max); ok {
				safeOutputsConfigLog.Printf("Parsed max as integer: %d", maxInt)
				s := defaultIntStr(maxInt)
				config.Max = s
			}
		}
	}
}
