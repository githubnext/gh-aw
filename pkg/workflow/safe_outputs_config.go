package workflow

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/typeutil"
)

var safeOutputsConfigLog = logger.New("workflow:safe_outputs_config")

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

			// Handle create-issue
			issuesConfig := c.parseCreateIssuesConfig(outputMap)
			if issuesConfig != nil {
				safeOutputsConfigLog.Print("Configured create-issue output handler")
				config.CreateIssues = issuesConfig
			}

			// Handle create-agent-session
			agentSessionConfig := c.parseAgentSessionConfig(outputMap)
			if agentSessionConfig != nil {
				config.CreateAgentSessions = agentSessionConfig
			}

			// Handle update-project (smart project board management)
			updateProjectConfig := c.parseUpdateProjectConfig(outputMap)
			if updateProjectConfig != nil {
				config.UpdateProjects = updateProjectConfig
			}

			// Handle create-project
			createProjectConfig := c.parseCreateProjectsConfig(outputMap)
			if createProjectConfig != nil {
				config.CreateProjects = createProjectConfig
			}

			// Handle create-project-status-update (project status updates)
			createProjectStatusUpdateConfig := c.parseCreateProjectStatusUpdateConfig(outputMap)
			if createProjectStatusUpdateConfig != nil {
				config.CreateProjectStatusUpdates = createProjectStatusUpdateConfig
			}

			// Handle create-discussion
			discussionsConfig := c.parseCreateDiscussionsConfig(outputMap)
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

			// Handle mark-pull-request-as-ready-for-review
			markPRReadyConfig := c.parseMarkPullRequestAsReadyForReviewConfig(outputMap)
			if markPRReadyConfig != nil {
				config.MarkPullRequestAsReadyForReview = markPRReadyConfig
			}

			// Handle add-comment
			commentsConfig := c.parseCommentsConfig(outputMap)
			if commentsConfig != nil {
				config.AddComments = commentsConfig
			}

			// Handle create-pull-request
			pullRequestsConfig := c.parseCreatePullRequestsConfig(outputMap)
			if pullRequestsConfig != nil {
				safeOutputsConfigLog.Print("Configured create-pull-request output handler")
				config.CreatePullRequests = pullRequestsConfig
			}

			// Handle create-pull-request-review-comment
			prReviewCommentsConfig := c.parsePullRequestReviewCommentsConfig(outputMap)
			if prReviewCommentsConfig != nil {
				config.CreatePullRequestReviewComments = prReviewCommentsConfig
			}

			// Handle submit-pull-request-review
			submitPRReviewConfig := c.parseSubmitPullRequestReviewConfig(outputMap)
			if submitPRReviewConfig != nil {
				config.SubmitPullRequestReview = submitPRReviewConfig
			}

			// Handle reply-to-pull-request-review-comment
			replyToPRReviewCommentConfig := c.parseReplyToPullRequestReviewCommentConfig(outputMap)
			if replyToPRReviewCommentConfig != nil {
				config.ReplyToPullRequestReviewComment = replyToPRReviewCommentConfig
			}

			// Handle resolve-pull-request-review-thread
			resolvePRReviewThreadConfig := c.parseResolvePullRequestReviewThreadConfig(outputMap)
			if resolvePRReviewThreadConfig != nil {
				config.ResolvePullRequestReviewThread = resolvePRReviewThreadConfig
			}

			// Handle create-code-scanning-alert
			securityReportsConfig := c.parseCodeScanningAlertsConfig(outputMap)
			if securityReportsConfig != nil {
				config.CreateCodeScanningAlerts = securityReportsConfig
			}

			// Handle autofix-code-scanning-alert
			autofixCodeScanningAlertConfig := c.parseAutofixCodeScanningAlertConfig(outputMap)
			if autofixCodeScanningAlertConfig != nil {
				config.AutofixCodeScanningAlert = autofixCodeScanningAlertConfig
			}

			// Handle create-check-run
			createCheckRunConfig := c.parseCreateCheckRunConfig(outputMap)
			if createCheckRunConfig != nil {
				config.CreateCheckRun = createCheckRunConfig
			}

			// Parse allowed-domains configuration (additional domains, unioned with network.allowed; supports ecosystem identifiers)
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

			// Parse allowed-github-references configuration
			if allowGitHubRefs, exists := outputMap["allowed-github-references"]; exists {
				if refsArray, ok := allowGitHubRefs.([]any); ok {
					refStrings := []string{} // Initialize as empty slice, not nil
					for _, ref := range refsArray {
						if refStr, ok := ref.(string); ok {
							refStrings = append(refStrings, refStr)
						}
					}
					config.AllowGitHubReferences = refStrings
				}
			}

			// Parse add-labels configuration
			addLabelsConfig := c.parseAddLabelsConfig(outputMap)
			if addLabelsConfig != nil {
				config.AddLabels = addLabelsConfig
			}

			// Parse remove-labels configuration
			removeLabelsConfig := c.parseRemoveLabelsConfig(outputMap)
			if removeLabelsConfig != nil {
				config.RemoveLabels = removeLabelsConfig
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

			// Handle unassign-from-user
			unassignFromUserConfig := c.parseUnassignFromUserConfig(outputMap)
			if unassignFromUserConfig != nil {
				config.UnassignFromUser = unassignFromUserConfig
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

			// Handle merge-pull-request
			mergePullRequestConfig := c.parseMergePullRequestConfig(outputMap)
			if mergePullRequestConfig != nil {
				config.MergePullRequest = mergePullRequestConfig
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

			// Handle upload-artifact
			uploadArtifactConfig := c.parseUploadArtifactConfig(outputMap)
			if uploadArtifactConfig != nil {
				config.UploadArtifact = uploadArtifactConfig
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

			// Handle set-issue-type
			setIssueTypeConfig := c.parseSetIssueTypeConfig(outputMap)
			if setIssueTypeConfig != nil {
				config.SetIssueType = setIssueTypeConfig
			}

			// Handle set-issue-field
			setIssueFieldConfig := c.parseSetIssueFieldConfig(outputMap)
			if setIssueFieldConfig != nil {
				config.SetIssueField = setIssueFieldConfig
			}

			// Handle dispatch-workflow
			dispatchWorkflowConfig := c.parseDispatchWorkflowConfig(outputMap)
			if dispatchWorkflowConfig != nil {
				config.DispatchWorkflow = dispatchWorkflowConfig
			}

			// Handle dispatch_repository
			dispatchRepositoryConfig := c.parseDispatchRepositoryConfig(outputMap)
			if dispatchRepositoryConfig != nil {
				config.DispatchRepository = dispatchRepositoryConfig
			}

			// Handle call-workflow
			callWorkflowConfig := c.parseCallWorkflowConfig(outputMap)
			if callWorkflowConfig != nil {
				config.CallWorkflow = callWorkflowConfig
			}

			// Handle missing-tool (parse configuration if present, or enable by default)
			missingToolConfig := c.parseMissingToolConfig(outputMap)
			if missingToolConfig != nil {
				config.MissingTool = missingToolConfig
			} else {
				// Enable missing-tool by default if safe-outputs exists and it wasn't explicitly disabled
				if _, exists := outputMap["missing-tool"]; !exists {
					trueVal := "true"
					config.MissingTool = &MissingToolConfig{
						CreateIssue: &trueVal,
						TitlePrefix: "",
						Labels:      nil,
					}
				}
			}

			// Handle missing-data (parse configuration if present, or enable by default)
			missingDataConfig := c.parseMissingDataConfig(outputMap)
			if missingDataConfig != nil {
				config.MissingData = missingDataConfig
			} else {
				// Enable missing-data by default if safe-outputs exists and it wasn't explicitly disabled
				if _, exists := outputMap["missing-data"]; !exists {
					trueVal := "true"
					config.MissingData = &MissingDataConfig{
						CreateIssue: &trueVal,
						TitlePrefix: "",
						Labels:      nil,
					}
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
					config.NoOp.Max = defaultIntStr(1) // Default max
					trueVal := "true"
					config.NoOp.ReportAsIssue = &trueVal // Default to reporting to issue
				}
			}

			// Handle report-incomplete (parse configuration if present, or enable by default)
			reportIncompleteConfig := c.parseReportIncompleteConfig(outputMap)
			if reportIncompleteConfig != nil {
				config.ReportIncomplete = reportIncompleteConfig
			} else {
				// Enable report-incomplete by default if safe-outputs exists and it wasn't explicitly disabled.
				// This ensures agents always have a first-class channel to signal task incompletion.
				if _, exists := outputMap["report-incomplete"]; !exists {
					trueVal := "true"
					config.ReportIncomplete = &ReportIncompleteConfig{
						CreateIssue: &trueVal,
						TitlePrefix: "",
						Labels:      nil,
					}
				}
			}

			// Handle staged flag
			if staged, exists := outputMap["staged"]; exists {
				if stagedBool, ok := staged.(bool); ok {
					config.Staged = stagedBool
				}
			}
			if c.forceStaged {
				config.Staged = true
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

			// Handle max-patch-files configuration (maximum unique files allowed in
			// a create-pull-request patch). Mirrors max-patch-size handling above,
			// with explicit bounds checks before narrowing to int so that very
			// large source values can't overflow/wrap into a negative or wrapped
			// number that would silently fall back to the default.
			if maxPatchFiles, exists := outputMap["max-patch-files"]; exists {
				switch v := maxPatchFiles.(type) {
				case int:
					if v >= 1 {
						config.MaximumPatchFiles = v
					}
				case int64:
					if v >= 1 {
						if v > int64(math.MaxInt) {
							safeOutputsConfigLog.Printf("max-patch-files: int64 value %d exceeds platform int range, clamping to %d", v, math.MaxInt)
							config.MaximumPatchFiles = math.MaxInt
						} else {
							config.MaximumPatchFiles = int(v)
						}
					}
				case uint64:
					if v >= 1 {
						if v > uint64(math.MaxInt) {
							safeOutputsConfigLog.Printf("max-patch-files: uint64 value %d exceeds platform int range, clamping to %d", v, math.MaxInt)
							config.MaximumPatchFiles = math.MaxInt
						} else {
							config.MaximumPatchFiles = int(v)
						}
					}
				case float64:
					// Reject NaN/Inf and clamp out-of-range floats before
					// narrowing — `int(NaN)` and `int(±Inf)` are
					// implementation-defined and can produce surprising
					// values (including 0, which would silently fall back
					// to the default).
					if v != v || v > float64(math.MaxInt) || v < float64(math.MinInt) {
						safeOutputsConfigLog.Printf("max-patch-files: float value %.2f is out of range, ignoring", v)
						break
					}
					intVal := int(v)
					if v != float64(intVal) {
						safeOutputsConfigLog.Printf("max-patch-files: float value %.2f truncated to integer %d", v, intVal)
					}
					if intVal >= 1 {
						config.MaximumPatchFiles = intVal
					}
				}
			}

			// Set default value if not specified or invalid
			if config.MaximumPatchFiles == 0 {
				config.MaximumPatchFiles = 100 // Default to 100 unique files
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

			// Handle activation-comments at safe-outputs top level (templatable boolean)
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

			// Handle mentions configuration
			if mentions, exists := outputMap["mentions"]; exists {
				config.Mentions = parseMentionsConfig(mentions)
			}

			// Handle global footer flag
			if footer, exists := outputMap["footer"]; exists {
				if footerBool, ok := footer.(bool); ok {
					config.Footer = &footerBool
					safeOutputsConfigLog.Printf("Global footer control: %t", footerBool)
				}
			}

			// Handle group-reports flag
			if groupReports, exists := outputMap["group-reports"]; exists {
				if groupReportsBool, ok := groupReports.(bool); ok {
					config.GroupReports = groupReportsBool
					safeOutputsConfigLog.Printf("Group reports control: %t", groupReportsBool)
				}
			}

			// Handle report-failure-as-issue flag
			if reportFailureAsIssue, exists := outputMap["report-failure-as-issue"]; exists {
				if reportFailureAsIssueBool, ok := reportFailureAsIssue.(bool); ok {
					config.ReportFailureAsIssue = &reportFailureAsIssueBool
					safeOutputsConfigLog.Printf("Report failure as issue: %t", reportFailureAsIssueBool)
				}
			}

			// Handle failure-issue-repo (repository for failure issues, format: "owner/repo")
			if failureIssueRepo, exists := outputMap["failure-issue-repo"]; exists {
				if failureIssueRepoStr, ok := failureIssueRepo.(string); ok && failureIssueRepoStr != "" {
					config.FailureIssueRepo = failureIssueRepoStr
					safeOutputsConfigLog.Printf("Failure issue repo: %s", failureIssueRepoStr)
				}
			}

			// Handle max-bot-mentions (templatable integer)
			if err := preprocessIntFieldAsString(outputMap, "max-bot-mentions", safeOutputsConfigLog); err != nil {
				safeOutputsConfigLog.Printf("max-bot-mentions: %v", err)
			} else if maxBotMentions, exists := outputMap["max-bot-mentions"]; exists {
				if maxBotMentionsStr, ok := maxBotMentions.(string); ok {
					config.MaxBotMentions = &maxBotMentionsStr
				}
			}

			// Handle steps (user-provided steps injected after checkout/setup, before safe-output code)
			if steps, exists := outputMap["steps"]; exists {
				if stepsList, ok := steps.([]any); ok {
					config.Steps = stepsList
					safeOutputsConfigLog.Printf("Configured %d user-provided steps for safe-outputs", len(stepsList))
				}
			}

			// Handle id-token permission override ("write" to force-add, "none" to disable auto-detection)
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

			// Handle concurrency-group configuration
			if concurrencyGroup, exists := outputMap["concurrency-group"]; exists {
				if concurrencyGroupStr, ok := concurrencyGroup.(string); ok && concurrencyGroupStr != "" {
					config.ConcurrencyGroup = concurrencyGroupStr
					safeOutputsConfigLog.Printf("Configured concurrency-group for safe-outputs job: %s", concurrencyGroupStr)
				}
			}

			// Handle needs configuration
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

			// Handle environment configuration (override for safe-outputs job; falls back to top-level environment)
			config.Environment = c.extractTopLevelYAMLSection(outputMap, "environment")
			if config.Environment != "" {
				safeOutputsConfigLog.Printf("Configured environment override for safe-outputs job: %s", config.Environment)
			}

			// Handle jobs (safe-jobs must be under safe-outputs)
			if jobs, exists := outputMap["jobs"]; exists {
				if jobsMap, ok := jobs.(map[string]any); ok {
					c := &Compiler{} // Create a temporary compiler instance for parsing
					config.Jobs = c.parseSafeJobsConfig(jobsMap)
				}
			}

			// Handle scripts (inline handlers that run in the safe-output handler loop)
			if scripts, exists := outputMap["scripts"]; exists {
				if scriptsMap, ok := scripts.(map[string]any); ok {
					config.Scripts = parseSafeScriptsConfig(scriptsMap)
					safeOutputsConfigLog.Printf("Configured %d custom safe-output script(s)", len(config.Scripts))
				}
			}

			// Handle actions (custom GitHub Actions mounted as safe output tools)
			if actions, exists := outputMap["actions"]; exists {
				if actionsMap, ok := actions.(map[string]any); ok {
					config.Actions = parseActionsConfig(actionsMap)
					safeOutputsConfigLog.Printf("Configured %d custom safe-output action(s)", len(config.Actions))
				}
			}

			// Handle app configuration for GitHub App token minting
			if app, exists := outputMap["github-app"]; exists {
				if appMap, ok := app.(map[string]any); ok {
					config.GitHubApp = parseAppConfig(appMap)
				}
			}
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

	if config != nil {
		safeOutputsConfigLog.Print("Successfully extracted safe-outputs configuration")
	} else {
		safeOutputsConfigLog.Print("No safe-outputs configuration found in frontmatter")
	}

	return config
}

// parseBaseSafeOutputConfig parses common fields (max, github-token, github-app, staged) from a config map.
// If defaultMax is provided (> 0), it will be set as the default value for config.Max
// before parsing the max field from configMap. Supports both integer values and GitHub
// Actions expression strings (e.g. "${{ inputs.max }}").
func (c *Compiler) parseBaseSafeOutputConfig(configMap map[string]any, config *BaseSafeOutputConfig, defaultMax int) {
	// Set default max if provided
	if defaultMax > 0 {
		safeOutputsConfigLog.Printf("Setting default max: %d", defaultMax)
		config.Max = defaultIntStr(defaultMax)
	}

	// Parse max (this will override the default if present in configMap)
	if max, exists := configMap["max"]; exists {
		switch v := max.(type) {
		case string:
			// Accept GitHub Actions expression strings
			if strings.HasPrefix(v, "${{") && strings.HasSuffix(v, "}}") {
				safeOutputsConfigLog.Printf("Parsed max as GitHub Actions expression: %s", v)
				config.Max = &v
			}
		default:
			// Convert integer/float64/etc to string via typeutil.ParseIntValue
			if maxInt, ok := typeutil.ParseIntValue(max); ok {
				safeOutputsConfigLog.Printf("Parsed max as integer: %d", maxInt)
				s := defaultIntStr(maxInt)
				config.Max = s
			}
		}
	}

	// Parse github-token
	if githubToken, exists := configMap["github-token"]; exists {
		if githubTokenStr, ok := githubToken.(string); ok {
			safeOutputsConfigLog.Print("Parsed custom github-token from config")
			config.GitHubToken = githubTokenStr
		}
	}

	// Parse github-app (per-handler GitHub App credentials for token minting)
	if app, exists := configMap["github-app"]; exists {
		if appMap, ok := app.(map[string]any); ok {
			safeOutputsConfigLog.Print("Parsed custom github-app from config")
			config.GitHubApp = parseAppConfig(appMap)
		}
	}

	// Parse staged flag (per-handler staged mode)
	if staged, exists := configMap["staged"]; exists {
		if stagedBool, ok := staged.(bool); ok {
			safeOutputsConfigLog.Printf("Parsed staged flag: %t", stagedBool)
			config.Staged = stagedBool
		}
	}
}

// SafeOutputStepConfig holds configuration for building a single safe output step
// within the consolidated safe-outputs job
type SafeOutputStepConfig struct {
	StepName                   string            // Human-readable step name (e.g., "Create Issue")
	StepID                     string            // Step ID for referencing outputs (e.g., "create_issue")
	Script                     string            // JavaScript script to execute (for inline mode)
	ScriptName                 string            // Name of the script in the registry (for file mode)
	CustomEnvVars              []string          // Environment variables specific to this step
	Condition                  ConditionNode     // Step-level condition (if clause)
	Token                      string            // GitHub token for this step
	UseCopilotRequestsToken    bool              // Whether to use Copilot requests token preference chain
	UseCopilotCodingAgentToken bool              // Whether to use Copilot coding agent token preference chain
	PreSteps                   []string          // Optional steps to run before the script step
	PostSteps                  []string          // Optional steps to run after the script step
	Outputs                    map[string]string // Outputs from this step
	ContinueOnError            bool              // Whether to continue the job even if this step fails (continue-on-error: true)
}

// handlerRegistry maps handler names to their builder functions.
// Each entry is keyed by the handler name used in GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG
// and returns a config map (nil means the handler is disabled).
var handlerRegistry = map[string]handlerBuilder{
	"create_issue": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.CreateIssues == nil {
			return nil
		}
		c := cfg.CreateIssues
		builder := newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddStringSlice("allowed_labels", c.AllowedLabels).
			AddStringSlice("allowed_fields", c.AllowedFields).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfPositive("expires", c.Expires).
			AddStringSlice("labels", c.Labels).
			AddIfNotEmpty("title_prefix", c.TitlePrefix).
			AddStringSlice("assignees", c.Assignees).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddTemplatableBool("group", c.Group).
			AddTemplatableBool("close_older_issues", c.CloseOlderIssues).
			AddIfNotEmpty("close_older_key", c.CloseOlderKey).
			AddTemplatableBool("group_by_day", c.GroupByDay).
			AddTemplatableBool("footer", getEffectiveFooterForTemplatable(c.Footer, cfg.Footer)).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			AddBoolOrInt("deduplicate_by_title", c.DeduplicateByTitle)
		return builder.Build()
	},
	"add_comment": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.AddComments == nil {
			return nil
		}
		c := cfg.AddComments
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("target", c.Target).
			AddTemplatableBool("hide_older_comments", c.HideOlderComments).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddTemplatableStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddTemplatableBool("footer", getEffectiveFooterForTemplatable(c.Footer, cfg.Footer)).
			AddStringSlice("required_labels", c.RequiredLabels).
			AddIfNotEmpty("required_title_prefix", c.RequiredTitlePrefix).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"comment_memory": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.CommentMemory == nil {
			return nil
		}
		c := cfg.CommentMemory
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("target", c.Target).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("memory_id", c.MemoryID).
			AddTemplatableBool("footer", getEffectiveFooterForTemplatable(c.Footer, cfg.Footer)).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"create_discussion": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.CreateDiscussions == nil {
			return nil
		}
		c := cfg.CreateDiscussions
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("category", c.Category).
			AddIfNotEmpty("title_prefix", c.TitlePrefix).
			AddIfPositive("min_body_length", c.MinBodyLength).
			AddStringSlice("labels", c.Labels).
			AddStringSlice("allowed_labels", c.AllowedLabels).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddTemplatableBool("close_older_discussions", c.CloseOlderDiscussions).
			AddIfNotEmpty("close_older_key", c.CloseOlderKey).
			AddIfNotEmpty("required_category", c.RequiredCategory).
			AddIfPositive("expires", c.Expires).
			AddBoolPtr("fallback_to_issue", c.FallbackToIssue).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddTemplatableBool("footer", getEffectiveFooterForTemplatable(c.Footer, cfg.Footer)).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"close_issue": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.CloseIssues == nil {
			return nil
		}
		c := cfg.CloseIssues
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("target", c.Target).
			AddStringSlice("required_labels", c.RequiredLabels).
			AddIfNotEmpty("required_title_prefix", c.RequiredTitlePrefix).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("state_reason", c.StateReason).
			AddBoolPtr("allow_body", c.AllowBody).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"close_discussion": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.CloseDiscussions == nil {
			return nil
		}
		c := cfg.CloseDiscussions
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("target", c.Target).
			AddStringSlice("required_labels", c.RequiredLabels).
			AddIfNotEmpty("required_title_prefix", c.RequiredTitlePrefix).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddBoolPtr("allow_body", c.AllowBody).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"add_labels": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.AddLabels == nil {
			return nil
		}
		c := cfg.AddLabels
		config := newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddStringSlice("allowed", c.Allowed).
			AddStringSlice("blocked", c.Blocked).
			AddIfNotEmpty("target", c.Target).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddStringSlice("required_labels", c.RequiredLabels).
			AddIfNotEmpty("required_title_prefix", c.RequiredTitlePrefix).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
		// If config is empty, it means add_labels was explicitly configured with no options
		// (null config), which means "allow any labels". Return non-nil empty map to
		// indicate the handler is enabled.
		if len(config) == 0 {
			// Return empty map so handler is included in config
			return make(map[string]any)
		}
		return config
	},
	"remove_labels": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.RemoveLabels == nil {
			return nil
		}
		c := cfg.RemoveLabels
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddStringSlice("allowed", c.Allowed).
			AddStringSlice("blocked", c.Blocked).
			AddIfNotEmpty("target", c.Target).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddStringSlice("required_labels", c.RequiredLabels).
			AddIfNotEmpty("required_title_prefix", c.RequiredTitlePrefix).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"add_reviewer": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.AddReviewer == nil {
			return nil
		}
		c := cfg.AddReviewer
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddStringSlice("allowed", c.AllowedReviewers).
			AddStringSlice("allowed_team_reviewers", c.AllowedTeamReviewers).
			AddIfNotEmpty("target", c.Target).AddStringSlice("required_labels", c.RequiredLabels).
			AddIfNotEmpty("required_title_prefix", c.RequiredTitlePrefix).AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"assign_milestone": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.AssignMilestone == nil {
			return nil
		}
		c := cfg.AssignMilestone
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddStringSlice("allowed", c.Allowed).
			AddIfNotEmpty("target", c.Target).AddStringSlice("required_labels", c.RequiredLabels).
			AddIfNotEmpty("required_title_prefix", c.RequiredTitlePrefix).AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			AddIfTrue("auto_create", c.AutoCreate).
			Build()
	},
	"mark_pull_request_as_ready_for_review": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.MarkPullRequestAsReadyForReview == nil {
			return nil
		}
		c := cfg.MarkPullRequestAsReadyForReview
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("target", c.Target).
			AddStringSlice("required_labels", c.RequiredLabels).
			AddIfNotEmpty("required_title_prefix", c.RequiredTitlePrefix).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"create_code_scanning_alert": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.CreateCodeScanningAlerts == nil {
			return nil
		}
		c := cfg.CreateCodeScanningAlerts
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("driver", c.Driver).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"create_check_run": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.CreateCheckRun == nil {
			return nil
		}
		c := cfg.CreateCheckRun
		builder := newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("name", c.Name).
			AddIfTrue("staged", c.Staged)
		if c.Output != nil {
			builder.
				AddIfNotEmpty("output_title", c.Output.Title).
				AddIfNotEmpty("output_summary", c.Output.Summary)
		}
		// When a per-handler github-app is configured, the compiler mints a token in a
		// separate step (create-check-run-app-token) and passes it as github-token so the
		// JS handler can use it via createAuthenticatedGitHubClient.
		// Per-handler github-token takes precedence when github-app is NOT set.
		if c.GitHubApp != nil {
			//nolint:gosec // G101: False positive - this is a GitHub Actions expression template, not a hardcoded credential
			builder.AddIfNotEmpty("github-token", "${{ steps.create-check-run-app-token.outputs.token }}")
		} else {
			builder.AddIfNotEmpty("github-token", c.GitHubToken)
		}
		return builder.Build()
	},
	"create_agent_session": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.CreateAgentSessions == nil {
			return nil
		}
		c := cfg.CreateAgentSessions
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("base", c.Base).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"update_issue": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.UpdateIssues == nil {
			return nil
		}
		c := cfg.UpdateIssues
		builder := newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("target", c.Target).
			AddIfNotEmpty("title_prefix", c.TitlePrefix).
			AddStringSlice("required_labels", c.RequiredLabels).
			AddIfNotEmpty("required_title_prefix", c.RequiredTitlePrefix)
		// Boolean pointer fields indicate which fields can be updated
		if c.Status != nil {
			builder.AddDefault("allow_status", true)
		}
		if c.Title != nil {
			builder.AddDefault("allow_title", true)
		}
		// Body uses boolean value mode - add the actual boolean value
		builder.AddBoolPtrOrDefault("allow_body", c.Body, true)
		return builder.
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddTemplatableBool("footer", getEffectiveFooterForTemplatable(c.Footer, cfg.Footer)).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"update_discussion": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.UpdateDiscussions == nil {
			return nil
		}
		c := cfg.UpdateDiscussions
		builder := newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("target", c.Target)
		// Boolean pointer fields indicate which fields can be updated
		if c.Title != nil {
			builder.AddDefault("allow_title", true)
		}
		if c.Body != nil {
			builder.AddDefault("allow_body", true)
		}
		if c.Labels != nil {
			builder.AddDefault("allow_labels", true)
		}
		return builder.
			AddStringSlice("allowed_labels", c.AllowedLabels).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddTemplatableBool("footer", getEffectiveFooterForTemplatable(c.Footer, cfg.Footer)).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"link_sub_issue": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.LinkSubIssue == nil {
			return nil
		}
		c := cfg.LinkSubIssue
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddStringSlice("parent_required_labels", c.ParentRequiredLabels).
			AddIfNotEmpty("parent_title_prefix", c.ParentTitlePrefix).
			AddStringSlice("sub_required_labels", c.SubRequiredLabels).
			AddIfNotEmpty("sub_title_prefix", c.SubTitlePrefix).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"update_release": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.UpdateRelease == nil {
			return nil
		}
		c := cfg.UpdateRelease
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddTemplatableBool("footer", getEffectiveFooterForTemplatable(c.Footer, cfg.Footer)).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"create_pull_request_review_comment": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.CreatePullRequestReviewComments == nil {
			return nil
		}
		c := cfg.CreatePullRequestReviewComments
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("side", c.Side).
			AddIfNotEmpty("target", c.Target).
			AddStringSlice("required_labels", c.RequiredLabels).
			AddIfNotEmpty("required_title_prefix", c.RequiredTitlePrefix).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"submit_pull_request_review": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.SubmitPullRequestReview == nil {
			return nil
		}
		c := cfg.SubmitPullRequestReview
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("target", c.Target).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddStringSlice("allowed_events", c.AllowedEvents).
			AddIfTrue("supersede_older_reviews", c.SupersedeOlderReviews).AddStringSlice("required_labels", c.RequiredLabels).
			AddIfNotEmpty("required_title_prefix", c.RequiredTitlePrefix).AddIfNotEmpty("github-token", c.GitHubToken).
			AddStringPtr("footer", getEffectiveFooterString(c.Footer, cfg.Footer)).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"reply_to_pull_request_review_comment": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.ReplyToPullRequestReviewComment == nil {
			return nil
		}
		c := cfg.ReplyToPullRequestReviewComment
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("target", c.Target).
			AddStringSlice("required_labels", c.RequiredLabels).
			AddIfNotEmpty("required_title_prefix", c.RequiredTitlePrefix).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddTemplatableBool("footer", getEffectiveFooterForTemplatable(c.Footer, cfg.Footer)).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"resolve_pull_request_review_thread": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.ResolvePullRequestReviewThread == nil {
			return nil
		}
		c := cfg.ResolvePullRequestReviewThread
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("target", c.Target).
			AddStringSlice("required_labels", c.RequiredLabels).
			AddIfNotEmpty("required_title_prefix", c.RequiredTitlePrefix).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"create_pull_request": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.CreatePullRequests == nil {
			return nil
		}
		c := cfg.CreatePullRequests
		protectedFilesPolicy := "request_review"
		if c.ManifestFilesPolicy != nil {
			protectedFilesPolicy = *c.ManifestFilesPolicy
		}
		maxPatchSize := 1024 // default 1024 KB
		if cfg.MaximumPatchSize > 0 {
			maxPatchSize = cfg.MaximumPatchSize
		}
		if c.MaxPatchSize > 0 {
			maxPatchSize = c.MaxPatchSize
		}
		maxPatchFiles := 100 // default 100 unique files
		if cfg.MaximumPatchFiles > 0 {
			maxPatchFiles = cfg.MaximumPatchFiles
		}
		if c.MaxPatchFiles > 0 {
			maxPatchFiles = c.MaxPatchFiles
		}
		builder := newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("branch_prefix", c.BranchPrefix).
			AddIfNotEmpty("title_prefix", c.TitlePrefix).
			AddTemplatableStringSlice("labels", c.Labels).
			AddStringSlice("fallback_labels", c.FallbackLabels).
			AddStringSlice("reviewers", c.Reviewers).
			AddStringSlice("team_reviewers", c.TeamReviewers).
			AddStringSlice("assignees", c.Assignees).
			AddTemplatableBool("draft", c.Draft).
			AddIfNotEmpty("if_no_changes", c.IfNoChanges).
			AddTemplatableBool("allow_empty", c.AllowEmpty).
			AddTemplatableBool("auto_merge", c.AutoMerge).
			AddIfPositive("expires", c.Expires).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddTemplatableStringSlice("allowed_repos", c.AllowedRepos).
			AddTemplatableStringSlice("allowed_base_branches", c.AllowedBaseBranches).
			AddTemplatableStringSlice("allowed_branches", c.AllowedBranches).
			AddDefault("max_patch_size", maxPatchSize).
			AddDefault("max_patch_files", maxPatchFiles).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddTemplatableBool("footer", getEffectiveFooterForTemplatable(c.Footer, cfg.Footer)).
			AddBoolPtr("fallback_as_issue", c.FallbackAsIssue).
			AddTemplatableBool("auto_close_issue", c.AutoCloseIssue).
			AddIfNotEmpty("base_branch", c.BaseBranch).
			AddDefault("protected_files_policy", protectedFilesPolicy).
			AddStringSlice("protected_files", getAllManifestFiles()).
			AddStringSlice("protected_path_prefixes", getProtectedPathPrefixes()).
			AddDefault("protect_top_level_dot_folders", true).
			AddStringSlice("_protected_files_exclude", c.ProtectedFilesExclude).
			AddStringSlice("allowed_files", c.AllowedFiles).
			AddStringSlice("excluded_files", c.ExcludedFiles).
			AddIfTrue("preserve_branch_name", c.PreserveBranchName).
			AddIfTrue("recreate_ref", c.RecreateRef).
			AddIfNotEmpty("patch_format", c.PatchFormat).
			AddBoolPtr("signed_commits", c.SignedCommits).
			AddIfTrue("staged", c.Staged)
		return builder.Build()
	},
	"push_to_pull_request_branch": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.PushToPullRequestBranch == nil {
			return nil
		}
		c := cfg.PushToPullRequestBranch
		maxPatchSize := 1024 // default 1024 KB
		if cfg.MaximumPatchSize > 0 {
			maxPatchSize = cfg.MaximumPatchSize
		}
		if c.MaxPatchSize > 0 {
			maxPatchSize = c.MaxPatchSize
		}
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("target", c.Target).
			AddIfNotEmpty("title_prefix", c.TitlePrefix).
			AddTemplatableStringSlice("required_labels", c.RequiredLabels).
			AddIfNotEmpty("if_no_changes", c.IfNoChanges).
			AddIfTrue("ignore_missing_branch_failure", c.IgnoreMissingBranchFailure).
			AddIfNotEmpty("commit_title_suffix", c.CommitTitleSuffix).
			AddDefault("max_patch_size", maxPatchSize).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddTemplatableStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			AddStringPtr("protected_files_policy", c.ManifestFilesPolicy).
			AddStringSlice("protected_files", getAllManifestFiles()).
			AddStringSlice("protected_path_prefixes", getProtectedPathPrefixes()).
			AddDefault("protect_top_level_dot_folders", true).
			AddStringSlice("_protected_files_exclude", c.ProtectedFilesExclude).
			AddStringSlice("allowed_files", c.AllowedFiles).
			AddStringSlice("excluded_files", c.ExcludedFiles).
			AddIfNotEmpty("patch_format", c.PatchFormat).
			AddBoolPtr("fallback_as_pull_request", c.FallbackAsPullRequest).
			AddBoolPtr("signed_commits", c.SignedCommits).
			AddBoolPtr("check_branch_protection", c.CheckBranchProtection).
			Build()
	},
	"update_pull_request": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.UpdatePullRequests == nil {
			return nil
		}
		c := cfg.UpdatePullRequests
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("target", c.Target).
			AddBoolPtrOrDefault("allow_title", c.Title, true).
			AddBoolPtrOrDefault("allow_body", c.Body, true).
			AddBoolPtrOrDefault("update_branch", c.UpdateBranch, false).
			AddStringPtr("default_operation", c.Operation).
			AddTemplatableBool("footer", getEffectiveFooterForTemplatable(c.Footer, cfg.Footer)).AddStringSlice("required_labels", c.RequiredLabels).
			AddIfNotEmpty("required_title_prefix", c.RequiredTitlePrefix).AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"merge_pull_request": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.MergePullRequest == nil {
			return nil
		}
		c := cfg.MergePullRequest
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddStringSlice("required_labels", c.RequiredLabels).AddIfNotEmpty("required_title_prefix", c.RequiredTitlePrefix).AddStringSlice("allowed_branches", c.AllowedBranches).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"close_pull_request": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.ClosePullRequests == nil {
			return nil
		}
		c := cfg.ClosePullRequests
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("target", c.Target).
			AddStringSlice("required_labels", c.RequiredLabels).
			AddIfNotEmpty("required_title_prefix", c.RequiredTitlePrefix).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"hide_comment": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.HideComment == nil {
			return nil
		}
		c := cfg.HideComment
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddStringSlice("allowed_reasons", c.AllowedReasons).AddIfNotEmpty("target", c.Target).
			AddStringSlice("required_labels", c.RequiredLabels).
			AddIfNotEmpty("required_title_prefix", c.RequiredTitlePrefix).AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"dispatch_workflow": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.DispatchWorkflow == nil {
			return nil
		}
		c := cfg.DispatchWorkflow
		builder := newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddStringSlice("workflows", c.Workflows).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug)

		// Add workflow_files map if it has entries
		if len(c.WorkflowFiles) > 0 {
			builder.AddDefault("workflow_files", c.WorkflowFiles)
		}

		// Add aw_context_workflows list if it has entries
		if len(c.AwContextWorkflows) > 0 {
			builder.AddStringSlice("aw_context_workflows", c.AwContextWorkflows)
		}

		builder.AddIfNotEmpty("target-ref", c.TargetRef)
		builder.AddIfNotEmpty("github-token", c.GitHubToken)
		builder.AddIfTrue("staged", c.Staged)
		return builder.Build()
	},
	"dispatch_repository": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.DispatchRepository == nil || len(cfg.DispatchRepository.Tools) == 0 {
			return nil
		}
		// Serialize each tool as a sub-map
		tools := make(map[string]any, len(cfg.DispatchRepository.Tools))
		for toolKey, tool := range cfg.DispatchRepository.Tools {
			toolConfig := newHandlerConfigBuilder().
				AddIfNotEmpty("workflow", tool.Workflow).
				AddIfNotEmpty("event_type", tool.EventType).
				AddIfNotEmpty("repository", tool.Repository).
				AddStringSlice("allowed_repositories", tool.AllowedRepositories).
				AddTemplatableInt("max", tool.Max).
				AddIfNotEmpty("github-token", tool.GitHubToken).
				AddIfTrue("staged", tool.Staged).
				Build()
			tools[toolKey] = toolConfig
		}
		return map[string]any{"tools": tools}
	},
	"call_workflow": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.CallWorkflow == nil {
			return nil
		}
		c := cfg.CallWorkflow
		builder := newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddStringSlice("workflows", c.Workflows)

		// Add workflow_files map if it has entries
		if len(c.WorkflowFiles) > 0 {
			builder.AddDefault("workflow_files", c.WorkflowFiles)
		}

		builder.AddIfTrue("staged", c.Staged)
		return builder.Build()
	},
	"missing_tool": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.MissingTool == nil {
			return nil
		}
		c := cfg.MissingTool
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"missing_data": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.MissingData == nil {
			return nil
		}
		c := cfg.MissingData
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"noop": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.NoOp == nil {
			return nil
		}
		c := cfg.NoOp
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddStringPtr("report-as-issue", c.ReportAsIssue).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"report_incomplete": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.ReportIncomplete == nil {
			return nil
		}
		c := cfg.ReportIncomplete
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"create_report_incomplete_issue": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.ReportIncomplete == nil {
			return nil
		}
		c := cfg.ReportIncomplete
		// If create-issue is explicitly false, skip generating the issue handler.
		// For nil (default) or "true", always include; for expressions, include
		// the handler and embed the expression so it is evaluated at runtime.
		if c.CreateIssue != nil && *c.CreateIssue == "false" {
			return nil
		}
		builder := newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("title-prefix", c.TitlePrefix).
			AddStringSlice("labels", c.Labels).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged)
		// When create-issue is a GitHub Actions expression, embed it in the handler config.
		// GitHub Actions evaluates the expression before the handler runs; the JavaScript
		// handler then parses the resolved value via parseBoolTemplatable at runtime.
		if c.CreateIssue != nil && isExpression(*c.CreateIssue) {
			builder = builder.AddTemplatableBool("create-issue", c.CreateIssue)
		}
		return builder.Build()
	},
	"assign_to_agent": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.AssignToAgent == nil {
			return nil
		}
		c := cfg.AssignToAgent
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("name", c.DefaultAgent).
			AddIfNotEmpty("model", c.DefaultModel).
			AddIfNotEmpty("custom-agent", c.DefaultCustomAgent).
			AddIfNotEmpty("custom-instructions", c.DefaultCustomInstructions).
			AddStringSlice("allowed", c.Allowed).
			AddIfTrue("ignore-if-error", c.IgnoreIfError).
			AddIfNotEmpty("target", c.Target).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed-repos", c.AllowedRepos).
			AddIfNotEmpty("pull-request-repo", c.PullRequestRepoSlug).
			AddStringSlice("allowed-pull-request-repos", c.AllowedPullRequestRepos).
			AddIfNotEmpty("base-branch", c.BaseBranch).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"upload_asset": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.UploadAssets == nil {
			return nil
		}
		c := cfg.UploadAssets
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("branch", c.BranchName).
			AddIfPositive("max-size", c.MaxSizeKB).
			AddStringSlice("allowed-exts", c.AllowedExts).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"upload_artifact": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.UploadArtifact == nil {
			return nil
		}
		c := cfg.UploadArtifact
		b := newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfPositive("max-uploads", c.MaxUploads).
			AddTemplatableInt("retention-days", c.RetentionDays).
			AddTemplatableBool("skip-archive", c.SkipArchive).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged)
		if c.MaxSizeBytes > 0 {
			b = b.AddDefault("max-size-bytes", c.MaxSizeBytes)
		}
		if len(c.AllowedPaths) > 0 {
			b = b.AddStringSlice("allowed-paths", c.AllowedPaths)
		}
		if c.Defaults != nil {
			if c.Defaults.IfNoFiles != "" {
				b = b.AddIfNotEmpty("default-if-no-files", c.Defaults.IfNoFiles)
			}
		}
		if c.Filters != nil {
			if len(c.Filters.Include) > 0 {
				b = b.AddStringSlice("filters-include", c.Filters.Include)
			}
			if len(c.Filters.Exclude) > 0 {
				b = b.AddStringSlice("filters-exclude", c.Filters.Exclude)
			}
		}
		return b.Build()
	},
	"autofix_code_scanning_alert": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.AutofixCodeScanningAlert == nil {
			return nil
		}
		c := cfg.AutofixCodeScanningAlert
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	// Note: create_project, update_project and create_project_status_update are handled by the unified handler,
	// not the separate project handler manager, so they are included in this registry.
	"create_project": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.CreateProjects == nil {
			return nil
		}
		c := cfg.CreateProjects
		builder := newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("target_owner", c.TargetOwner).
			AddIfNotEmpty("title_prefix", c.TitlePrefix).
			AddIfNotEmpty("github-token", c.GitHubToken)
		if len(c.Views) > 0 {
			builder.AddDefault("views", c.Views)
		}
		if len(c.FieldDefinitions) > 0 {
			builder.AddDefault("field_definitions", c.FieldDefinitions)
		}
		builder.AddIfTrue("staged", c.Staged)
		return builder.Build()
	},
	"update_project": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.UpdateProjects == nil {
			return nil
		}
		c := cfg.UpdateProjects
		builder := newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfNotEmpty("project", c.Project).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos)
		if len(c.Views) > 0 {
			builder.AddDefault("views", c.Views)
		}
		if len(c.FieldDefinitions) > 0 {
			builder.AddDefault("field_definitions", c.FieldDefinitions)
		}
		builder.AddIfTrue("staged", c.Staged)
		return builder.Build()
	},
	"assign_to_user": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.AssignToUser == nil {
			return nil
		}
		c := cfg.AssignToUser
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddStringSlice("allowed", c.Allowed).
			AddStringSlice("blocked", c.Blocked).
			AddIfNotEmpty("target", c.Target).AddStringSlice("required_labels", c.RequiredLabels).
			AddIfNotEmpty("required_title_prefix", c.RequiredTitlePrefix).AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddTemplatableBool("unassign_first", c.UnassignFirst).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"unassign_from_user": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.UnassignFromUser == nil {
			return nil
		}
		c := cfg.UnassignFromUser
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddStringSlice("allowed", c.Allowed).
			AddStringSlice("blocked", c.Blocked).
			AddIfNotEmpty("target", c.Target).
			AddStringSlice("required_labels", c.RequiredLabels).
			AddIfNotEmpty("required_title_prefix", c.RequiredTitlePrefix).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"create_project_status_update": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.CreateProjectStatusUpdates == nil {
			return nil
		}
		c := cfg.CreateProjectStatusUpdates
		return newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfNotEmpty("project", c.Project).
			AddIfTrue("staged", c.Staged).
			Build()
	},
	"set_issue_type": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.SetIssueType == nil {
			return nil
		}
		c := cfg.SetIssueType
		config := newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddStringSlice("allowed", c.Allowed).
			AddIfNotEmpty("target", c.Target).
			AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
		// If config is empty, it means set_issue_type was explicitly configured with no options
		// (null config), which means "allow any type". Return non-nil empty map to
		// indicate the handler is enabled.
		if len(config) == 0 {
			return make(map[string]any)
		}
		return config
	},
	"set_issue_field": func(cfg *SafeOutputsConfig) map[string]any {
		if cfg.SetIssueField == nil {
			return nil
		}
		c := cfg.SetIssueField
		config := newHandlerConfigBuilder().
			AddTemplatableInt("max", c.Max).
			AddStringSlice("allowed_fields", c.AllowedFields).
			AddIfNotEmpty("target", c.Target).AddStringSlice("required_labels", c.RequiredLabels).
			AddIfNotEmpty("required_title_prefix", c.RequiredTitlePrefix).AddIfNotEmpty("target-repo", c.TargetRepoSlug).
			AddStringSlice("allowed_repos", c.AllowedRepos).
			AddIfNotEmpty("github-token", c.GitHubToken).
			AddIfTrue("staged", c.Staged).
			Build()
		if len(config) == 0 {
			return make(map[string]any)
		}
		return config
	},
}

func (c *Compiler) addHandlerManagerConfigEnvVar(steps *[]string, data *WorkflowData) {
	if data.SafeOutputs == nil {
		safeOutputsConfigLog.Print("No safe-outputs configuration, skipping handler manager config")
		return
	}

	safeOutputsConfigLog.Print("Building handler manager configuration for safe-outputs")
	// config holds both per-handler configs (keyed by handler name, e.g. "add_comment") and
	// global runtime knobs (e.g. "mentions") that safe_output_handler_manager.cjs forwards to
	// specific handlers at startup. Handler names are the reserved keys defined in handlerRegistry;
	// non-handler keys ("mentions") are documented in safe_outputs_config_generation.go.
	config := make(map[string]any)

	// Collect engine-specific manifest files and path prefixes (AgentFileProvider interface).
	// These are merged with the global runtime-derived lists so that engine-specific
	// instruction files (e.g. CLAUDE.md, .claude/, AGENTS.md) are automatically protected.
	extraManifestFiles, extraPathPrefixes := c.getEngineAgentFileInfo(data)
	fullManifestFiles := getAllManifestFiles(extraManifestFiles...)
	fullPathPrefixes := getProtectedPathPrefixes(extraPathPrefixes...)

	// For workflow_call relay workflows, inject the resolved platform repo and ref into the
	// dispatch_workflow handler config so dispatch targets the host repo, not the caller's.
	safeOutputs := data.SafeOutputs
	if hasWorkflowCallTrigger(data.On) && safeOutputs.DispatchWorkflow != nil {
		if safeOutputs.DispatchWorkflow.TargetRepoSlug == "" {
			safeOutputs = safeOutputsWithDispatchTargetRepo(safeOutputs, "${{ needs.activation.outputs.target_repo }}")
			safeOutputsConfigLog.Print("Injecting target_repo into dispatch_workflow config for workflow_call relay")
		}
		if safeOutputs.DispatchWorkflow.TargetRef == "" {
			safeOutputs = safeOutputsWithDispatchTargetRef(safeOutputs, "${{ needs.activation.outputs.target_ref }}")
			safeOutputsConfigLog.Print("Injecting target_ref into dispatch_workflow config for workflow_call relay")
		}
	}

	// Build configuration for each handler using the registry
	for handlerName, builder := range handlerRegistry {
		handlerConfig := builder(safeOutputs)
		// Include handler if:
		// 1. It returns a non-nil config (explicitly enabled, even if empty)
		// 2. For auto-enabled handlers, include even with empty config
		if handlerConfig != nil {
			injectCurrentCheckoutPatchWorkspacePath(handlerName, handlerConfig, data)
			// Augment protected-files protection with engine-specific files for handlers that use it.
			if _, hasProtected := handlerConfig["protected_files"]; hasProtected {
				// Extract per-handler exclusions set by the handler builder (sentinel key).
				// These are compile-time overrides and must not be forwarded to the runtime.
				excludeFiles := ParseStringArrayFromConfig(handlerConfig, "_protected_files_exclude", nil)
				delete(handlerConfig, "_protected_files_exclude")

				handlerConfig["protected_files"] = sliceutil.Exclude(fullManifestFiles, excludeFiles...)
				filteredPrefixes := sliceutil.Exclude(fullPathPrefixes, excludeFiles...)
				if len(filteredPrefixes) > 0 {
					handlerConfig["protected_path_prefixes"] = filteredPrefixes
				} else {
					delete(handlerConfig, "protected_path_prefixes")
				}
				// Compute which top-level dot-folder prefixes are excluded so the runtime
				// dot-folder check can skip them.
				if dotFolderExcludes := getDotFolderExcludes(excludeFiles); len(dotFolderExcludes) > 0 {
					handlerConfig["protected_dot_folder_excludes"] = dotFolderExcludes
				}
			}
			safeOutputsConfigLog.Printf("Adding %s handler configuration", handlerName)
			config[handlerName] = handlerConfig
		}
	}

	// Include top-level mentions configuration so the handler manager can pass it to
	// markdown-producing handlers that call sanitizeContent with allowed aliases.
	if safeOutputs.Mentions != nil {
		mentionsCfg := buildMentionsHandlerConfig(safeOutputs.Mentions)
		if len(mentionsCfg) > 0 {
			config["mentions"] = mentionsCfg
		}
	}

	// Only add the env var if there are handlers to configure
	if len(config) > 0 {
		safeOutputsConfigLog.Printf("Marshaling handler config with %d handlers", len(config))
		configJSON, err := json.Marshal(config)
		if err != nil {
			safeOutputsConfigLog.Printf("Failed to marshal handler config: %v", err)
			return
		}
		// Escape the JSON for YAML (handle quotes and special chars)
		configStr := string(configJSON)
		*steps = append(*steps, fmt.Sprintf("          GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG: %q\n", configStr))
		safeOutputsConfigLog.Printf("Added handler config env var: size=%d bytes", len(configStr))
	} else {
		safeOutputsConfigLog.Print("No handlers configured, skipping config env var")
	}
}

// buildMentionsHandlerConfig converts a MentionsConfig into the map format used by
// GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG so safe_output_handler_manager.cjs can pass
// the top-level mentions policy through to mention-aware handlers.
func buildMentionsHandlerConfig(m *MentionsConfig) map[string]any {
	cfg := make(map[string]any)
	if m.Enabled != nil {
		cfg["enabled"] = *m.Enabled
	}
	if m.AllowTeamMembers != nil {
		cfg["allowTeamMembers"] = *m.AllowTeamMembers
	}
	if m.AllowContext != nil {
		cfg["allowContext"] = *m.AllowContext
	}
	if len(m.Allowed) > 0 {
		cfg["allowed"] = m.Allowed
	}
	if m.Max != nil {
		cfg["max"] = *m.Max
	}
	return cfg
}

// safeOutputsWithDispatchTargetRepo returns a shallow copy of cfg with the dispatch_workflow
// TargetRepoSlug overridden to targetRepo. Only DispatchWorkflow is deep-copied; all other
// pointer fields remain shared. This avoids mutating the original config.
func safeOutputsWithDispatchTargetRepo(cfg *SafeOutputsConfig, targetRepo string) *SafeOutputsConfig {
	dispatchCopy := *cfg.DispatchWorkflow
	dispatchCopy.TargetRepoSlug = targetRepo
	configCopy := *cfg
	configCopy.DispatchWorkflow = &dispatchCopy
	return &configCopy
}

// safeOutputsWithDispatchTargetRef returns a shallow copy of cfg with the dispatch_workflow
// TargetRef overridden to targetRef. Only DispatchWorkflow is deep-copied; all other
// pointer fields remain shared. This avoids mutating the original config.
func safeOutputsWithDispatchTargetRef(cfg *SafeOutputsConfig, targetRef string) *SafeOutputsConfig {
	dispatchCopy := *cfg.DispatchWorkflow
	dispatchCopy.TargetRef = targetRef
	configCopy := *cfg
	configCopy.DispatchWorkflow = &dispatchCopy
	return &configCopy
}

// getEngineAgentFileInfo returns the engine-specific manifest filenames and path prefixes
// by type-asserting the active engine to AgentFileProvider.  Returns empty slices when
// the engine is not set or does not implement the interface.
func (c *Compiler) getEngineAgentFileInfo(data *WorkflowData) (manifestFiles []string, pathPrefixes []string) {
	if data == nil || data.EngineConfig == nil {
		return nil, nil
	}
	engine, err := c.engineRegistry.GetEngine(data.EngineConfig.ID)
	if err != nil {
		safeOutputsConfigLog.Printf("Engine lookup failed for %q: %v — skipping agent manifest file injection", data.EngineConfig.ID, err)
		return nil, nil
	}
	if engine == nil {
		return nil, nil
	}
	provider, ok := engine.(AgentFileProvider)
	if !ok {
		return nil, nil
	}
	safeOutputsConfigLog.Printf("Engine %s provides AgentFileProvider: files=%v, prefixes=%v",
		data.EngineConfig.ID, provider.GetAgentManifestFiles(), provider.GetAgentManifestPathPrefixes())
	return provider.GetAgentManifestFiles(), provider.GetAgentManifestPathPrefixes()
}
