package workflow

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
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
			issuesConfig := c.parseIssuesConfig(outputMap)
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
			pullRequestsConfig := c.parsePullRequestsConfig(outputMap)
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

			// Handle set-issue-type
			setIssueTypeConfig := c.parseSetIssueTypeConfig(outputMap)
			if setIssueTypeConfig != nil {
				config.SetIssueType = setIssueTypeConfig
			}

			// Handle dispatch-workflow
			dispatchWorkflowConfig := c.parseDispatchWorkflowConfig(outputMap)
			if dispatchWorkflowConfig != nil {
				config.DispatchWorkflow = dispatchWorkflowConfig
			}

			// Handle missing-tool (parse configuration if present, or enable by default)
			missingToolConfig := c.parseMissingToolConfig(outputMap)
			if missingToolConfig != nil {
				config.MissingTool = missingToolConfig
			} else {
				// Enable missing-tool by default if safe-outputs exists and it wasn't explicitly disabled
				// Auto-enabled missing-tool does NOT have create-issue enabled by default
				if _, exists := outputMap["missing-tool"]; !exists {
					config.MissingTool = &MissingToolConfig{
						CreateIssue: false, // Auto-enabled missing-tool doesn't create issues by default
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
				// Auto-enabled missing-data does NOT have create-issue enabled by default
				if _, exists := outputMap["missing-data"]; !exists {
					config.MissingData = &MissingDataConfig{
						CreateIssue: false, // Auto-enabled missing-data doesn't create issues by default
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

			// Handle app configuration for GitHub App token minting
			if app, exists := outputMap["github-app"]; exists {
				if appMap, ok := app.(map[string]any); ok {
					config.GitHubApp = parseAppConfig(appMap)
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

// parseBaseSafeOutputConfig parses common fields (max, github-token, staged) from a config map.
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
			// Convert integer/float64/etc to string via parseIntValue
			if maxInt, ok := parseIntValue(max); ok {
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

	// Parse staged flag (per-handler staged mode)
	if staged, exists := configMap["staged"]; exists {
		if stagedBool, ok := staged.(bool); ok {
			safeOutputsConfigLog.Printf("Parsed staged flag: %t", stagedBool)
			config.Staged = stagedBool
		}
	}
}

var safeOutputsAppLog = logger.New("workflow:safe_outputs_app")

// ========================================
// GitHub App Configuration
// ========================================

// GitHubAppConfig holds configuration for GitHub App-based token minting
type GitHubAppConfig struct {
	AppID        string   `yaml:"app-id,omitempty"`       // GitHub App ID (e.g., "${{ vars.APP_ID }}")
	PrivateKey   string   `yaml:"private-key,omitempty"`  // GitHub App private key (e.g., "${{ secrets.APP_PRIVATE_KEY }}")
	Owner        string   `yaml:"owner,omitempty"`        // Optional: owner of the GitHub App installation (defaults to current repository owner)
	Repositories []string `yaml:"repositories,omitempty"` // Optional: comma or newline-separated list of repositories to grant access to
}

// ========================================
// App Configuration Parsing
// ========================================

// parseAppConfig parses the app configuration from a map
func parseAppConfig(appMap map[string]any) *GitHubAppConfig {
	safeOutputsAppLog.Print("Parsing GitHub App configuration")
	appConfig := &GitHubAppConfig{}

	// Parse app-id (required)
	if appID, exists := appMap["app-id"]; exists {
		if appIDStr, ok := appID.(string); ok {
			appConfig.AppID = appIDStr
		}
	}

	// Parse private-key (required)
	if privateKey, exists := appMap["private-key"]; exists {
		if privateKeyStr, ok := privateKey.(string); ok {
			appConfig.PrivateKey = privateKeyStr
		}
	}

	// Parse owner (optional)
	if owner, exists := appMap["owner"]; exists {
		if ownerStr, ok := owner.(string); ok {
			appConfig.Owner = ownerStr
		}
	}

	// Parse repositories (optional)
	if repos, exists := appMap["repositories"]; exists {
		if reposArray, ok := repos.([]any); ok {
			var repoStrings []string
			for _, repo := range reposArray {
				if repoStr, ok := repo.(string); ok {
					repoStrings = append(repoStrings, repoStr)
				}
			}
			appConfig.Repositories = repoStrings
		}
	}

	return appConfig
}

// ========================================
// App Configuration Merging
// ========================================

// mergeAppFromIncludedConfigs merges app configuration from included safe-outputs configurations
// If the top-level workflow has an app configured, it takes precedence
// Otherwise, the first app configuration found in included configs is used
func (c *Compiler) mergeAppFromIncludedConfigs(topSafeOutputs *SafeOutputsConfig, includedConfigs []string) (*GitHubAppConfig, error) {
	safeOutputsAppLog.Printf("Merging app configuration: included_configs=%d", len(includedConfigs))
	// If top-level workflow already has app configured, use it (no merge needed)
	if topSafeOutputs != nil && topSafeOutputs.GitHubApp != nil {
		safeOutputsAppLog.Print("Using top-level app configuration")
		return topSafeOutputs.GitHubApp, nil
	}

	// Otherwise, find the first app configuration in included configs
	for _, configJSON := range includedConfigs {
		if configJSON == "" || configJSON == "{}" {
			continue
		}

		// Parse the safe-outputs configuration
		var safeOutputsConfig map[string]any
		if err := json.Unmarshal([]byte(configJSON), &safeOutputsConfig); err != nil {
			continue // Skip invalid JSON
		}

		// Extract app from the safe-outputs.github-app field
		if appData, exists := safeOutputsConfig["github-app"]; exists {
			if appMap, ok := appData.(map[string]any); ok {
				appConfig := parseAppConfig(appMap)

				// Return first valid app configuration found
				if appConfig.AppID != "" && appConfig.PrivateKey != "" {
					safeOutputsAppLog.Print("Found valid app configuration in included config")
					return appConfig, nil
				}
			}
		}
	}

	safeOutputsAppLog.Print("No app configuration found in included configs")
	return nil, nil
}

// ========================================
// GitHub App Token Steps Generation
// ========================================

// buildGitHubAppTokenMintStep generates the step to mint a GitHub App installation access token
// Permissions are automatically computed from the safe output job requirements
func (c *Compiler) buildGitHubAppTokenMintStep(app *GitHubAppConfig, permissions *Permissions) []string {
	safeOutputsAppLog.Printf("Building GitHub App token mint step: owner=%s, repos=%d", app.Owner, len(app.Repositories))
	var steps []string

	steps = append(steps, "      - name: Generate GitHub App token\n")
	steps = append(steps, "        id: safe-outputs-app-token\n")
	steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/create-github-app-token")))
	steps = append(steps, "        with:\n")
	steps = append(steps, fmt.Sprintf("          app-id: %s\n", app.AppID))
	steps = append(steps, fmt.Sprintf("          private-key: %s\n", app.PrivateKey))

	// Add owner - default to current repository owner if not specified
	owner := app.Owner
	if owner == "" {
		owner = "${{ github.repository_owner }}"
	}
	steps = append(steps, fmt.Sprintf("          owner: %s\n", owner))

	// Add repositories - behavior depends on configuration:
	// - If repositories is ["*"], omit the field to allow org-wide access
	// - If repositories is a single value, use inline format
	// - If repositories has multiple values, use block scalar format (newline-separated)
	//   to ensure clarity and proper parsing by actions/create-github-app-token
	// - If repositories is empty/not specified, default to current repository
	if len(app.Repositories) == 1 && app.Repositories[0] == "*" {
		// Org-wide access: omit repositories field entirely
		safeOutputsAppLog.Print("Using org-wide GitHub App token (repositories: *)")
	} else if len(app.Repositories) == 1 {
		// Single repository: use inline format for clarity
		steps = append(steps, fmt.Sprintf("          repositories: %s\n", app.Repositories[0]))
	} else if len(app.Repositories) > 1 {
		// Multiple repositories: use block scalar format (newline-separated)
		// This format is more readable and avoids potential issues with comma-separated parsing
		steps = append(steps, "          repositories: |-\n")
		for _, repo := range app.Repositories {
			steps = append(steps, fmt.Sprintf("            %s\n", repo))
		}
	} else {
		// Extract repo name from github.repository (which is "owner/repo")
		// Using GitHub Actions expression: split(github.repository, '/')[1]
		steps = append(steps, "          repositories: ${{ github.event.repository.name }}\n")
	}

	// Always add github-api-url from environment variable
	steps = append(steps, "          github-api-url: ${{ github.api_url }}\n")

	// Add permission-* fields automatically computed from job permissions
	// Sort keys to ensure deterministic compilation order
	if permissions != nil {
		permissionFields := convertPermissionsToAppTokenFields(permissions)

		// Extract and sort keys for deterministic ordering
		keys := make([]string, 0, len(permissionFields))
		for key := range permissionFields {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		// Add permissions in sorted order
		for _, key := range keys {
			steps = append(steps, fmt.Sprintf("          %s: %s\n", key, permissionFields[key]))
		}
	}

	return steps
}

// convertPermissionsToAppTokenFields converts job Permissions to permission-* action inputs
// This follows GitHub's recommendation for explicit permission control
// Note: This only includes permissions that are valid for GitHub App tokens.
// Some GitHub Actions permissions (like 'discussions', 'models') don't have
// corresponding GitHub App permissions and are skipped.
func convertPermissionsToAppTokenFields(permissions *Permissions) map[string]string {
	fields := make(map[string]string)

	// Map GitHub Actions permissions to GitHub App permissions
	// Only include permissions that exist in the actions/create-github-app-token action
	// See: https://github.com/actions/create-github-app-token#permissions

	// Repository permissions that map directly
	if level, ok := permissions.Get(PermissionActions); ok {
		fields["permission-actions"] = string(level)
	}
	if level, ok := permissions.Get(PermissionChecks); ok {
		fields["permission-checks"] = string(level)
	}
	if level, ok := permissions.Get(PermissionContents); ok {
		fields["permission-contents"] = string(level)
	}
	if level, ok := permissions.Get(PermissionDeployments); ok {
		fields["permission-deployments"] = string(level)
	}
	if level, ok := permissions.Get(PermissionIssues); ok {
		fields["permission-issues"] = string(level)
	}
	if level, ok := permissions.Get(PermissionPackages); ok {
		fields["permission-packages"] = string(level)
	}
	if level, ok := permissions.Get(PermissionPages); ok {
		fields["permission-pages"] = string(level)
	}
	if level, ok := permissions.Get(PermissionPullRequests); ok {
		fields["permission-pull-requests"] = string(level)
	}
	if level, ok := permissions.Get(PermissionSecurityEvents); ok {
		fields["permission-security-events"] = string(level)
	}
	if level, ok := permissions.Get(PermissionStatuses); ok {
		fields["permission-statuses"] = string(level)
	}
	if level, ok := permissions.Get(PermissionOrganizationProj); ok {
		fields["permission-organization-projects"] = string(level)
	}
	if level, ok := permissions.Get(PermissionDiscussions); ok {
		fields["permission-discussions"] = string(level)
	}

	// Note: The following GitHub Actions permissions do NOT have GitHub App equivalents:
	// - models (no GitHub App permission for this)
	// - id-token (not applicable to GitHub Apps)
	// - attestations (no GitHub App permission for this)
	// - repository-projects (removed - classic projects are sunset; use organization-projects for Projects v2 via PAT/GitHub App)

	return fields
}

// buildGitHubAppTokenInvalidationStep generates the step to invalidate the GitHub App token
// This step always runs (even on failure) to ensure tokens are properly cleaned up
// Only runs if a token was successfully minted
func (c *Compiler) buildGitHubAppTokenInvalidationStep() []string {
	var steps []string

	steps = append(steps, "      - name: Invalidate GitHub App token\n")
	steps = append(steps, "        if: always() && steps.safe-outputs-app-token.outputs.token != ''\n")
	steps = append(steps, "        env:\n")
	steps = append(steps, "          TOKEN: ${{ steps.safe-outputs-app-token.outputs.token }}\n")
	steps = append(steps, "        run: |\n")
	steps = append(steps, "          echo \"Revoking GitHub App installation token...\"\n")
	steps = append(steps, "          # GitHub CLI will auth with the token being revoked.\n")
	steps = append(steps, "          gh api \\\n")
	steps = append(steps, "            --method DELETE \\\n")
	steps = append(steps, "            -H \"Authorization: token $TOKEN\" \\\n")
	steps = append(steps, "            /installation/token || echo \"Token revoke may already be expired.\"\n")
	steps = append(steps, "          \n")
	steps = append(steps, "          echo \"Token invalidation step complete.\"\n")

	return steps
}

// ========================================
// Activation Token Steps Generation
// ========================================

// buildActivationAppTokenMintStep generates the step to mint a GitHub App installation access token
// for use in the pre-activation (reaction) and activation (status comment) jobs.
func (c *Compiler) buildActivationAppTokenMintStep(app *GitHubAppConfig, permissions *Permissions) []string {
	safeOutputsAppLog.Printf("Building activation GitHub App token mint step: owner=%s", app.Owner)
	var steps []string

	steps = append(steps, "      - name: Generate GitHub App token for activation\n")
	steps = append(steps, "        id: activation-app-token\n")
	steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/create-github-app-token")))
	steps = append(steps, "        with:\n")
	steps = append(steps, fmt.Sprintf("          app-id: %s\n", app.AppID))
	steps = append(steps, fmt.Sprintf("          private-key: %s\n", app.PrivateKey))

	// Add owner - default to current repository owner if not specified
	owner := app.Owner
	if owner == "" {
		owner = "${{ github.repository_owner }}"
	}
	steps = append(steps, fmt.Sprintf("          owner: %s\n", owner))

	// Default to current repository
	steps = append(steps, "          repositories: ${{ github.event.repository.name }}\n")

	// Always add github-api-url from environment variable
	steps = append(steps, "          github-api-url: ${{ github.api_url }}\n")

	// Add permission-* fields automatically computed from job permissions
	if permissions != nil {
		permissionFields := convertPermissionsToAppTokenFields(permissions)

		keys := make([]string, 0, len(permissionFields))
		for key := range permissionFields {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			steps = append(steps, fmt.Sprintf("          %s: %s\n", key, permissionFields[key]))
		}
	}

	return steps
}

// resolveActivationToken returns the GitHub token to use for activation steps (reactions, status comments).
// Priority: GitHub App minted token > custom github-token > GITHUB_TOKEN (default)
//
// When returning the app token reference, callers MUST ensure that buildActivationAppTokenMintStep
// has already been called to generate the 'activation-app-token' step, since this function returns
// a reference to that step's output (${{ steps.activation-app-token.outputs.token }}).
func (c *Compiler) resolveActivationToken(data *WorkflowData) string {
	if data.ActivationGitHubApp != nil {
		return "${{ steps.activation-app-token.outputs.token }}"
	}
	if data.ActivationGitHubToken != "" {
		return data.ActivationGitHubToken
	}
	return "${{ secrets.GITHUB_TOKEN }}"
}

var safeOutputMessagesLog = logger.New("workflow:safe_outputs_config_messages")

// ========================================
// Safe Output Messages Configuration
// ========================================

// setStringFromMap reads m[key] and assigns its string value to *dest if found.
func setStringFromMap(m map[string]any, key string, dest *string) {
	if val, exists := m[key]; exists {
		if str, ok := val.(string); ok {
			*dest = str
		}
	}
}

// parseMessagesConfig parses the messages configuration from safe-outputs frontmatter
func parseMessagesConfig(messagesMap map[string]any) *SafeOutputMessagesConfig {
	safeOutputMessagesLog.Printf("Parsing messages configuration with %d fields", len(messagesMap))
	config := &SafeOutputMessagesConfig{}

	if appendOnly, exists := messagesMap["append-only-comments"]; exists {
		if appendOnlyBool, ok := appendOnly.(bool); ok {
			config.AppendOnlyComments = appendOnlyBool
			safeOutputMessagesLog.Printf("Set append-only-comments: %t", appendOnlyBool)
		}
	}

	setStringFromMap(messagesMap, "footer", &config.Footer)
	setStringFromMap(messagesMap, "footer-install", &config.FooterInstall)
	setStringFromMap(messagesMap, "footer-workflow-recompile", &config.FooterWorkflowRecompile)
	setStringFromMap(messagesMap, "footer-workflow-recompile-comment", &config.FooterWorkflowRecompileComment)
	setStringFromMap(messagesMap, "staged-title", &config.StagedTitle)
	setStringFromMap(messagesMap, "staged-description", &config.StagedDescription)
	setStringFromMap(messagesMap, "run-started", &config.RunStarted)
	setStringFromMap(messagesMap, "run-success", &config.RunSuccess)
	setStringFromMap(messagesMap, "run-failure", &config.RunFailure)
	setStringFromMap(messagesMap, "detection-failure", &config.DetectionFailure)
	setStringFromMap(messagesMap, "pull-request-created", &config.PullRequestCreated)
	setStringFromMap(messagesMap, "issue-created", &config.IssueCreated)
	setStringFromMap(messagesMap, "commit-pushed", &config.CommitPushed)
	setStringFromMap(messagesMap, "agent-failure-issue", &config.AgentFailureIssue)
	setStringFromMap(messagesMap, "agent-failure-comment", &config.AgentFailureComment)

	return config
}

// parseMentionsConfig parses the mentions configuration from safe-outputs frontmatter
// Mentions can be:
// - false: always escapes mentions
// - true: always allows mentions (error in strict mode)
// - object: detailed configuration with allow-team-members, allow-context, allowed, max
func parseMentionsConfig(mentions any) *MentionsConfig {
	safeOutputMessagesLog.Printf("Parsing mentions configuration: type=%T", mentions)
	config := &MentionsConfig{}

	// Handle boolean value
	if boolVal, ok := mentions.(bool); ok {
		config.Enabled = &boolVal
		safeOutputMessagesLog.Printf("Mentions configured as boolean: %t", boolVal)
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
						// Normalize username by removing '@' prefix if present
						normalized := str
						if len(str) > 0 && str[0] == '@' {
							normalized = str[1:]
							safeOutputMessagesLog.Printf("Normalized mention '%s' to '%s'", str, normalized)
						}
						allowedStrings = append(allowedStrings, normalized)
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
	safeOutputMessagesLog.Print("Serializing messages configuration to JSON")
	jsonBytes, err := json.Marshal(messages)
	if err != nil {
		safeOutputMessagesLog.Printf("Failed to serialize messages config: %v", err)
		return "", fmt.Errorf("failed to serialize messages config: %w", err)
	}
	safeOutputMessagesLog.Printf("Serialized messages config: %d bytes", len(jsonBytes))
	return string(jsonBytes), nil
}
