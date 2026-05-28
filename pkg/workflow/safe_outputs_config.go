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

	output, exists := frontmatter["safe-outputs"]
	if !exists {
		safeOutputsConfigLog.Print("No safe-outputs configuration found in frontmatter")
		return nil
	}

	outputMap, ok := output.(map[string]any)
	if !ok {
		safeOutputsConfigLog.Print("No safe-outputs configuration found in frontmatter")
		return nil
	}

	safeOutputsConfigLog.Printf("Processing safe-outputs configuration with %d top-level keys", len(outputMap))
	config := &SafeOutputsConfig{}

	c.applyProjectOutputConfigs(config, outputMap)
	c.applyIssueOutputConfigs(config, outputMap)
	c.applyDiscussionOutputConfigs(config, outputMap)
	c.applyCommentOutputConfigs(config, outputMap)
	c.applyPullRequestOutputConfigs(config, outputMap)
	c.applyAssignmentOutputConfigs(config, outputMap)
	c.applyAutomationOutputConfigs(config, outputMap)
	c.applySafeOutputsNetworkConfig(config, outputMap)
	c.applyMissingSignalConfigs(config, outputMap)
	c.applyFallbackOutputConfigs(config, outputMap)
	c.applySafeOutputsExecutionConfig(config, outputMap)
	c.applySafeOutputsPatchConfig(config, outputMap)
	c.applySafeOutputsMessageConfig(config, outputMap)
	c.applySafeOutputsExtensionConfig(config, outputMap)
	c.applySafeOutputsThreatConfig(config, outputMap)
	c.applyDefaultThreatDetection(config, outputMap)

	safeOutputsConfigLog.Print("Successfully extracted safe-outputs configuration")
	return config
}

func (c *Compiler) applyProjectOutputConfigs(config *SafeOutputsConfig, outputMap map[string]any) {
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
}

func (c *Compiler) applyIssueOutputConfigs(config *SafeOutputsConfig, outputMap map[string]any) {
	if issuesConfig := c.parseCreateIssuesConfig(outputMap); issuesConfig != nil {
		safeOutputsConfigLog.Print("Configured create-issue output handler")
		config.CreateIssues = issuesConfig
	}
	if closeIssuesConfig := c.parseCloseIssuesConfig(outputMap); closeIssuesConfig != nil {
		config.CloseIssues = closeIssuesConfig
	}
	if updateIssuesConfig := c.parseUpdateIssuesConfig(outputMap); updateIssuesConfig != nil {
		config.UpdateIssues = updateIssuesConfig
	}
	if addLabelsConfig := c.parseAddLabelsConfig(outputMap); addLabelsConfig != nil {
		config.AddLabels = addLabelsConfig
	}
	if removeLabelsConfig := c.parseRemoveLabelsConfig(outputMap); removeLabelsConfig != nil {
		config.RemoveLabels = removeLabelsConfig
	}
	if assignMilestoneConfig := c.parseAssignMilestoneConfig(outputMap); assignMilestoneConfig != nil {
		config.AssignMilestone = assignMilestoneConfig
	}
	if linkSubIssueConfig := c.parseLinkSubIssueConfig(outputMap); linkSubIssueConfig != nil {
		config.LinkSubIssue = linkSubIssueConfig
	}
	if setIssueTypeConfig := c.parseSetIssueTypeConfig(outputMap); setIssueTypeConfig != nil {
		config.SetIssueType = setIssueTypeConfig
	}
	if setIssueFieldConfig := c.parseSetIssueFieldConfig(outputMap); setIssueFieldConfig != nil {
		config.SetIssueField = setIssueFieldConfig
	}
}

func (c *Compiler) applyDiscussionOutputConfigs(config *SafeOutputsConfig, outputMap map[string]any) {
	if discussionsConfig := c.parseCreateDiscussionsConfig(outputMap); discussionsConfig != nil {
		config.CreateDiscussions = discussionsConfig
	}
	if closeDiscussionsConfig := c.parseCloseDiscussionsConfig(outputMap); closeDiscussionsConfig != nil {
		config.CloseDiscussions = closeDiscussionsConfig
	}
	if updateDiscussionsConfig := c.parseUpdateDiscussionsConfig(outputMap); updateDiscussionsConfig != nil {
		config.UpdateDiscussions = updateDiscussionsConfig
	}
}

func (c *Compiler) applyCommentOutputConfigs(config *SafeOutputsConfig, outputMap map[string]any) {
	if commentsConfig := c.parseCommentsConfig(outputMap); commentsConfig != nil {
		config.AddComments = commentsConfig
	}
	if hideCommentConfig := c.parseHideCommentConfig(outputMap); hideCommentConfig != nil {
		config.HideComment = hideCommentConfig
	}
}

func (c *Compiler) applyPullRequestOutputConfigs(config *SafeOutputsConfig, outputMap map[string]any) {
	if closePullRequestsConfig := c.parseClosePullRequestsConfig(outputMap); closePullRequestsConfig != nil {
		config.ClosePullRequests = closePullRequestsConfig
	}
	if markReadyConfig := c.parseMarkPullRequestAsReadyForReviewConfig(outputMap); markReadyConfig != nil {
		config.MarkPullRequestAsReadyForReview = markReadyConfig
	}
	if pullRequestsConfig := c.parseCreatePullRequestsConfig(outputMap); pullRequestsConfig != nil {
		safeOutputsConfigLog.Print("Configured create-pull-request output handler")
		config.CreatePullRequests = pullRequestsConfig
	}
	if reviewCommentsConfig := c.parsePullRequestReviewCommentsConfig(outputMap); reviewCommentsConfig != nil {
		config.CreatePullRequestReviewComments = reviewCommentsConfig
	}
	if submitReviewConfig := c.parseSubmitPullRequestReviewConfig(outputMap); submitReviewConfig != nil {
		config.SubmitPullRequestReview = submitReviewConfig
	}
	if replyConfig := c.parseReplyToPullRequestReviewCommentConfig(outputMap); replyConfig != nil {
		config.ReplyToPullRequestReviewComment = replyConfig
	}
	if resolveThreadConfig := c.parseResolvePullRequestReviewThreadConfig(outputMap); resolveThreadConfig != nil {
		config.ResolvePullRequestReviewThread = resolveThreadConfig
	}
	if addReviewerConfig := c.parseAddReviewerConfig(outputMap); addReviewerConfig != nil {
		config.AddReviewer = addReviewerConfig
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

func (c *Compiler) applyAssignmentOutputConfigs(config *SafeOutputsConfig, outputMap map[string]any) {
	if assignToAgentConfig := c.parseAssignToAgentConfig(outputMap); assignToAgentConfig != nil {
		config.AssignToAgent = assignToAgentConfig
	}
	if assignToUserConfig := c.parseAssignToUserConfig(outputMap); assignToUserConfig != nil {
		config.AssignToUser = assignToUserConfig
	}
	if unassignFromUserConfig := c.parseUnassignFromUserConfig(outputMap); unassignFromUserConfig != nil {
		config.UnassignFromUser = unassignFromUserConfig
	}
}

func (c *Compiler) applyAutomationOutputConfigs(config *SafeOutputsConfig, outputMap map[string]any) {
	if securityReportsConfig := c.parseCodeScanningAlertsConfig(outputMap); securityReportsConfig != nil {
		config.CreateCodeScanningAlerts = securityReportsConfig
	}
	if autofixConfig := c.parseAutofixCodeScanningAlertConfig(outputMap); autofixConfig != nil {
		config.AutofixCodeScanningAlert = autofixConfig
	}
	if checkRunConfig := c.parseCreateCheckRunConfig(outputMap); checkRunConfig != nil {
		config.CreateCheckRun = checkRunConfig
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

func (c *Compiler) applySafeOutputsNetworkConfig(config *SafeOutputsConfig, outputMap map[string]any) {
	if allowedDomains, exists := outputMap["allowed-domains"]; exists {
		if domainsArray, ok := allowedDomains.([]any); ok {
			config.AllowedDomains = collectStringValues(domainsArray)
			safeOutputsConfigLog.Printf("Configured allowed-domains with %d domain(s)", len(config.AllowedDomains))
		}
	}
	if allowGitHubRefs, exists := outputMap["allowed-github-references"]; exists {
		if refsArray, ok := allowGitHubRefs.([]any); ok {
			refs := make([]string, 0, len(refsArray))
			refs = append(refs, collectStringValues(refsArray)...)
			config.AllowGitHubReferences = refs
		}
	}
}

func (c *Compiler) applyMissingSignalConfigs(config *SafeOutputsConfig, outputMap map[string]any) {
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
}

func (c *Compiler) applyFallbackOutputConfigs(config *SafeOutputsConfig, outputMap map[string]any) {
	if noopConfig := c.parseNoOpConfig(outputMap); noopConfig != nil {
		config.NoOp = noopConfig
	} else if _, exists := outputMap["noop"]; !exists {
		trueVal := "true"
		config.NoOp = &NoOpConfig{
			BaseSafeOutputConfig: BaseSafeOutputConfig{Max: defaultIntStr(1)},
			ReportAsIssue:        &trueVal,
		}
	}
	if reportIncompleteConfig := c.parseReportIncompleteConfig(outputMap); reportIncompleteConfig != nil {
		config.ReportIncomplete = reportIncompleteConfig
	} else if _, exists := outputMap["report-incomplete"]; !exists {
		trueVal := "true"
		config.ReportIncomplete = &ReportIncompleteConfig{CreateIssue: &trueVal, TitlePrefix: "", Labels: nil}
	}
}

func (c *Compiler) applySafeOutputsExecutionConfig(config *SafeOutputsConfig, outputMap map[string]any) {
	if staged, exists := outputMap["staged"]; exists {
		if stagedBool, ok := staged.(bool); ok {
			config.Staged = stagedBool
		}
	}
	if c.forceStaged {
		config.Staged = true
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
	if runsOn, exists := outputMap["runs-on"]; exists {
		if runsOnStr, ok := runsOn.(string); ok {
			config.RunsOn = runsOnStr
		}
	}
	c.applySafeOutputsJobConfig(config, outputMap)
}

func (c *Compiler) applySafeOutputsJobConfig(config *SafeOutputsConfig, outputMap map[string]any) {
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
				safeOutputsConfigLog.Printf(`Warning: unrecognized safe-outputs id-token value %q (expected "write" or "none"); ignoring`, idTokenStr)
			}
		}
	}
	if concurrencyGroup, exists := outputMap["concurrency-group"]; exists {
		if concurrencyGroupStr, ok := concurrencyGroup.(string); ok && concurrencyGroupStr != "" {
			config.ConcurrencyGroup = concurrencyGroupStr
			safeOutputsConfigLog.Printf("Configured concurrency-group for safe-outputs job: %s", concurrencyGroupStr)
		}
	}
	if needsValue, exists := outputMap["needs"]; exists {
		if needsArray, ok := needsValue.([]any); ok {
			for _, needStr := range collectStringValues(needsArray) {
				if needStr != "" {
					config.Needs = append(config.Needs, needStr)
				}
			}
			if len(config.Needs) > 0 {
				safeOutputsConfigLog.Printf("Configured %d explicit safe-outputs needs dependency(ies)", len(config.Needs))
			}
		}
	}
	config.Environment = c.extractTopLevelYAMLSection(outputMap, "environment")
	if config.Environment != "" {
		safeOutputsConfigLog.Printf("Configured environment override for safe-outputs job: %s", config.Environment)
	}
}

func (c *Compiler) applySafeOutputsPatchConfig(config *SafeOutputsConfig, outputMap map[string]any) {
	c.applyMaximumPatchSize(config, outputMap)
	c.applyMaximumPatchFiles(config, outputMap)
}

func (c *Compiler) applyMaximumPatchSize(config *SafeOutputsConfig, outputMap map[string]any) {
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
			if v != float64(intVal) {
				safeOutputsConfigLog.Printf("max-patch-size: float value %.2f truncated to integer %d", v, intVal)
			}
			if intVal >= 1 {
				config.MaximumPatchSize = intVal
			}
		}
	}
	if config.MaximumPatchSize == 0 {
		config.MaximumPatchSize = 1024
	}
}

func (c *Compiler) applyMaximumPatchFiles(config *SafeOutputsConfig, outputMap map[string]any) {
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
			if v != v || v > float64(math.MaxInt) || v < float64(math.MinInt) {
				safeOutputsConfigLog.Printf("max-patch-files: float value %.2f is out of range, ignoring", v)
			} else {
				intVal := int(v)
				if v != float64(intVal) {
					safeOutputsConfigLog.Printf("max-patch-files: float value %.2f truncated to integer %d", v, intVal)
				}
				if intVal >= 1 {
					config.MaximumPatchFiles = intVal
				}
			}
		}
	}
	if config.MaximumPatchFiles == 0 {
		config.MaximumPatchFiles = 100
	}
}

func (c *Compiler) applySafeOutputsMessageConfig(config *SafeOutputsConfig, outputMap map[string]any) {
	if messages, exists := outputMap["messages"]; exists {
		if messagesMap, ok := messages.(map[string]any); ok {
			config.Messages = parseMessagesConfig(messagesMap)
		}
	}
	if err := preprocessBoolFieldAsString(outputMap, "activation-comments", safeOutputsConfigLog); err != nil {
		safeOutputsConfigLog.Printf("activation-comments: %v", err)
	} else if activationComments, exists := outputMap["activation-comments"]; exists {
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
			safeOutputsConfigLog.Printf("Global footer control: %t", footerBool)
		}
	}
	c.applySafeOutputsFailureConfig(config, outputMap)
}

func (c *Compiler) applySafeOutputsFailureConfig(config *SafeOutputsConfig, outputMap map[string]any) {
	if groupReports, exists := outputMap["group-reports"]; exists {
		if groupReportsBool, ok := groupReports.(bool); ok {
			config.GroupReports = groupReportsBool
			safeOutputsConfigLog.Printf("Group reports control: %t", groupReportsBool)
		}
	}
	if reportFailureAsIssue, exists := outputMap["report-failure-as-issue"]; exists {
		if reportFailureAsIssueBool, ok := reportFailureAsIssue.(bool); ok {
			config.ReportFailureAsIssue = &reportFailureAsIssueBool
			safeOutputsConfigLog.Printf("Report failure as issue: %t", reportFailureAsIssueBool)
		}
	}
	if failureIssueRepo, exists := outputMap["failure-issue-repo"]; exists {
		if failureIssueRepoStr, ok := failureIssueRepo.(string); ok && failureIssueRepoStr != "" {
			config.FailureIssueRepo = failureIssueRepoStr
			safeOutputsConfigLog.Printf("Failure issue repo: %s", failureIssueRepoStr)
		}
	}
	if err := preprocessIntFieldAsString(outputMap, "max-bot-mentions", safeOutputsConfigLog); err != nil {
		safeOutputsConfigLog.Printf("max-bot-mentions: %v", err)
	} else if maxBotMentions, exists := outputMap["max-bot-mentions"]; exists {
		if maxBotMentionsStr, ok := maxBotMentions.(string); ok {
			config.MaxBotMentions = &maxBotMentionsStr
		}
	}
}

func (c *Compiler) applySafeOutputsExtensionConfig(config *SafeOutputsConfig, outputMap map[string]any) {
	if jobs, exists := outputMap["jobs"]; exists {
		if jobsMap, ok := jobs.(map[string]any); ok {
			compiler := &Compiler{}
			config.Jobs = compiler.parseSafeJobsConfig(jobsMap)
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

func (c *Compiler) applySafeOutputsThreatConfig(config *SafeOutputsConfig, outputMap map[string]any) {
	if threatDetectionConfig := c.parseThreatDetectionConfig(outputMap); threatDetectionConfig != nil {
		config.ThreatDetection = threatDetectionConfig
	}
}

func (c *Compiler) applyDefaultThreatDetection(config *SafeOutputsConfig, outputMap map[string]any) {
	if config.ThreatDetection != nil {
		return
	}
	if _, exists := outputMap["threat-detection"]; exists {
		return
	}
	safeOutputsConfigLog.Print("Applying default threat-detection configuration")
	config.ThreatDetection = &ThreatDetectionConfig{}
}

func collectStringValues(values []any) []string {
	var result []string
	for _, value := range values {
		if valueStr, ok := value.(string); ok {
			result = append(result, valueStr)
		}
	}
	return result
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
		return newHandlerConfigBuilder().
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
			Build()
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

	config := c.buildHandlerManagerConfig(data)
	if len(config) == 0 {
		safeOutputsConfigLog.Print("No handlers configured, skipping config env var")
		return
	}
	c.appendHandlerManagerConfigEnvVar(steps, config)
}

func (c *Compiler) buildHandlerManagerConfig(data *WorkflowData) map[string]any {
	safeOutputsConfigLog.Print("Building handler manager configuration for safe-outputs")
	config := make(map[string]any)
	safeOutputs := c.getHandlerManagerSafeOutputs(data)
	fullManifestFiles, fullPathPrefixes := c.getProtectedHandlerPaths(data)
	for handlerName, builder := range handlerRegistry {
		c.addHandlerManagerConfig(config, handlerName, builder, safeOutputs, fullManifestFiles, fullPathPrefixes, data)
	}
	if safeOutputs.Mentions != nil {
		mentionsCfg := buildMentionsHandlerConfig(safeOutputs.Mentions)
		if len(mentionsCfg) > 0 {
			config["mentions"] = mentionsCfg
		}
	}
	return config
}

func (c *Compiler) getHandlerManagerSafeOutputs(data *WorkflowData) *SafeOutputsConfig {
	safeOutputs := data.SafeOutputs
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

func (c *Compiler) getProtectedHandlerPaths(data *WorkflowData) ([]string, []string) {
	extraManifestFiles, extraPathPrefixes := c.getEngineAgentFileInfo(data)
	return getAllManifestFiles(extraManifestFiles...), getProtectedPathPrefixes(extraPathPrefixes...)
}

func (c *Compiler) addHandlerManagerConfig(config map[string]any, handlerName string, builder handlerBuilder, safeOutputs *SafeOutputsConfig, fullManifestFiles, fullPathPrefixes []string, data *WorkflowData) {
	handlerConfig := builder(safeOutputs)
	if handlerConfig == nil {
		return
	}
	injectCurrentCheckoutPatchWorkspacePath(handlerName, handlerConfig, data)
	c.applyProtectedHandlerConfig(handlerConfig, fullManifestFiles, fullPathPrefixes)
	safeOutputsConfigLog.Printf("Adding %s handler configuration", handlerName)
	config[handlerName] = handlerConfig
}

func (c *Compiler) applyProtectedHandlerConfig(handlerConfig map[string]any, fullManifestFiles, fullPathPrefixes []string) {
	if _, hasProtected := handlerConfig["protected_files"]; !hasProtected {
		return
	}
	excludeFiles := ParseStringArrayFromConfig(handlerConfig, "_protected_files_exclude", nil)
	delete(handlerConfig, "_protected_files_exclude")
	handlerConfig["protected_files"] = sliceutil.Exclude(fullManifestFiles, excludeFiles...)
	filteredPrefixes := sliceutil.Exclude(fullPathPrefixes, excludeFiles...)
	if len(filteredPrefixes) > 0 {
		handlerConfig["protected_path_prefixes"] = filteredPrefixes
	} else {
		delete(handlerConfig, "protected_path_prefixes")
	}
	if dotFolderExcludes := getDotFolderExcludes(excludeFiles); len(dotFolderExcludes) > 0 {
		handlerConfig["protected_dot_folder_excludes"] = dotFolderExcludes
	}
}

func (c *Compiler) appendHandlerManagerConfigEnvVar(steps *[]string, config map[string]any) {
	safeOutputsConfigLog.Printf("Marshaling handler config with %d handlers", len(config))
	configJSON, err := json.Marshal(config)
	if err != nil {
		safeOutputsConfigLog.Printf("Failed to marshal handler config: %v", err)
		return
	}
	configStr := string(configJSON)
	*steps = append(*steps, fmt.Sprintf("          GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG: %q\n", configStr))
	safeOutputsConfigLog.Printf("Added handler config env var: size=%d bytes", len(configStr))
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
