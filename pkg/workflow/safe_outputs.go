package workflow

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var safeOutputsLog = logger.New("workflow:safe_outputs")

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
		safeOutputs.UpdateIssues != nil ||
		safeOutputs.UpdatePullRequests != nil ||
		safeOutputs.PushToPullRequestBranch != nil ||
		safeOutputs.UploadAssets != nil ||
		safeOutputs.MissingTool != nil ||
		safeOutputs.NoOp != nil ||
		safeOutputs.LinkSubIssue != nil ||
		len(safeOutputs.Jobs) > 0

	if safeOutputsLog.Enabled() {
		safeOutputsLog.Printf("Safe outputs enabled check: %v", enabled)
	}

	return enabled
}

// ========================================
// Safe Output Configuration Extraction
// ========================================

// extractSafeOutputsConfig extracts output configuration from frontmatter
func (c *Compiler) extractSafeOutputsConfig(frontmatter map[string]any) *SafeOutputsConfig {
	safeOutputsLog.Print("Extracting safe-outputs configuration from frontmatter")

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
			if labels, exists := outputMap["add-labels"]; exists {
				if labelsMap, ok := labels.(map[string]any); ok {
					labelConfig := &AddLabelsConfig{}

					// Parse list job config (target, target-repo, allowed)
					listJobConfig, _ := ParseListJobConfig(labelsMap, "allowed")
					labelConfig.SafeOutputTargetConfig = listJobConfig.SafeOutputTargetConfig
					labelConfig.Allowed = listJobConfig.Allowed

					// Parse common base fields (github-token, max)
					c.parseBaseSafeOutputConfig(labelsMap, &labelConfig.BaseSafeOutputConfig, 0)

					config.AddLabels = labelConfig
				} else if labels == nil {
					// Handle null case: create empty config (allows any labels)
					config.AddLabels = &AddLabelsConfig{}
				}
			}

			// Parse add-reviewer configuration
			addReviewerConfig := c.parseAddReviewerConfig(outputMap)
			if addReviewerConfig != nil {
				config.AddReviewer = addReviewerConfig
			}

			// Parse assign-milestone configuration
			if milestone, exists := outputMap["assign-milestone"]; exists {
				if milestoneMap, ok := milestone.(map[string]any); ok {
					milestoneConfig := &AssignMilestoneConfig{}

					// Parse list job config (target, target-repo, allowed)
					listJobConfig, _ := ParseListJobConfig(milestoneMap, "allowed")
					milestoneConfig.SafeOutputTargetConfig = listJobConfig.SafeOutputTargetConfig
					milestoneConfig.Allowed = listJobConfig.Allowed

					// Parse common base fields (github-token, max)
					c.parseBaseSafeOutputConfig(milestoneMap, &milestoneConfig.BaseSafeOutputConfig, 0)

					config.AssignMilestone = milestoneConfig
				} else if milestone == nil {
					// Handle null case: create empty config (allows any milestones)
					config.AssignMilestone = &AssignMilestoneConfig{}
				}
			}

			// Handle assign-to-agent
			if assignToAgent, exists := outputMap["assign-to-agent"]; exists {
				if agentMap, ok := assignToAgent.(map[string]any); ok {
					agentConfig := &AssignToAgentConfig{}

					// Parse name (optional - specific to assign-to-agent)
					if defaultAgent, exists := agentMap["name"]; exists {
						if defaultAgentStr, ok := defaultAgent.(string); ok {
							agentConfig.DefaultAgent = defaultAgentStr
						}
					}

					// Parse target config (target, target-repo) - validation errors are handled gracefully
					targetConfig, _ := ParseTargetConfig(agentMap)
					agentConfig.SafeOutputTargetConfig = targetConfig

					// Parse common base fields (github-token, max)
					c.parseBaseSafeOutputConfig(agentMap, &agentConfig.BaseSafeOutputConfig, 0)

					config.AssignToAgent = agentConfig
				} else if assignToAgent == nil {
					// Handle null case: create empty config
					config.AssignToAgent = &AssignToAgentConfig{}
				}
			}

			// Handle update-issue
			updateIssuesConfig := c.parseUpdateIssuesConfig(outputMap)
			if updateIssuesConfig != nil {
				config.UpdateIssues = updateIssuesConfig
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
						safeOutputsLog.Printf("max-patch-size: float value %.2f truncated to integer %d", v, intVal)
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
	if safeOutputsLog.Enabled() {
		safeOutputsLog.Printf("Normalized safe output identifier: %s -> %s", identifier, normalized)
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

// GitHubScriptStepConfig holds configuration for building a GitHub Script step
type GitHubScriptStepConfig struct {
	// Step metadata
	StepName string // e.g., "Create Output Issue"
	StepID   string // e.g., "create_issue"

	// Main job reference for agent output
	MainJobName string

	// Environment variables specific to this safe output type
	// These are added after GH_AW_AGENT_OUTPUT
	CustomEnvVars []string

	// JavaScript script constant to format and include
	Script string

	// Token configuration (passed to addSafeOutputGitHubTokenForConfig or addSafeOutputCopilotGitHubTokenForConfig)
	Token string

	// UseCopilotToken indicates whether to use the Copilot token preference chain
	// (COPILOT_GITHUB_TOKEN > COPILOT_CLI_TOKEN > GH_AW_COPILOT_TOKEN (legacy) > GH_AW_GITHUB_TOKEN (legacy))
	// This should be true for Copilot-related operations like creating agent tasks,
	// assigning copilot to issues, or adding copilot as PR reviewer
	UseCopilotToken bool

	// UseAgentToken indicates whether to use the agent token preference chain
	// (config token > GH_AW_AGENT_TOKEN)
	// This should be true for agent assignment operations (assign-to-agent)
	UseAgentToken bool
}

// buildGitHubScriptStep creates a GitHub Script step with common scaffolding
// This extracts the repeated pattern found across safe output job builders
func (c *Compiler) buildGitHubScriptStep(data *WorkflowData, config GitHubScriptStepConfig) []string {
	safeOutputsLog.Printf("Building GitHub Script step: %s (useCopilotToken=%v, useAgentToken=%v)", config.StepName, config.UseCopilotToken, config.UseAgentToken)

	var steps []string

	// Add artifact download steps before the GitHub Script step
	steps = append(steps, buildAgentOutputDownloadSteps()...)

	// Step name and metadata
	steps = append(steps, fmt.Sprintf("      - name: %s\n", config.StepName))
	steps = append(steps, fmt.Sprintf("        id: %s\n", config.StepID))
	steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))

	// Environment variables section
	steps = append(steps, "        env:\n")

	// Read GH_AW_AGENT_OUTPUT from environment (set by artifact download step)
	// instead of directly from job outputs which may be masked by GitHub Actions
	steps = append(steps, "          GH_AW_AGENT_OUTPUT: ${{ env.GH_AW_AGENT_OUTPUT }}\n")

	// Add custom environment variables specific to this safe output type
	steps = append(steps, config.CustomEnvVars...)

	// Add custom environment variables from safe-outputs.env
	c.addCustomSafeOutputEnvVars(&steps, data)

	// With section for github-token
	steps = append(steps, "        with:\n")
	if config.UseAgentToken {
		c.addSafeOutputAgentGitHubTokenForConfig(&steps, data, config.Token)
	} else if config.UseCopilotToken {
		c.addSafeOutputCopilotGitHubTokenForConfig(&steps, data, config.Token)
	} else {
		c.addSafeOutputGitHubTokenForConfig(&steps, data, config.Token)
	}

	steps = append(steps, "          script: |\n")

	// Add the formatted JavaScript script
	formattedScript := FormatJavaScriptForYAML(config.Script)
	steps = append(steps, formattedScript...)

	return steps
}

// buildGitHubScriptStepWithoutDownload creates a GitHub Script step without artifact download steps
// This is useful when multiple script steps are needed in the same job and artifact downloads
// should only happen once at the beginning
func (c *Compiler) buildGitHubScriptStepWithoutDownload(data *WorkflowData, config GitHubScriptStepConfig) []string {
	safeOutputsLog.Printf("Building GitHub Script step without download: %s", config.StepName)

	var steps []string

	// Step name and metadata (no artifact download steps)
	steps = append(steps, fmt.Sprintf("      - name: %s\n", config.StepName))
	steps = append(steps, fmt.Sprintf("        id: %s\n", config.StepID))
	steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))

	// Environment variables section
	steps = append(steps, "        env:\n")

	// Read GH_AW_AGENT_OUTPUT from environment (set by artifact download step)
	// instead of directly from job outputs which may be masked by GitHub Actions
	steps = append(steps, "          GH_AW_AGENT_OUTPUT: ${{ env.GH_AW_AGENT_OUTPUT }}\n")

	// Add custom environment variables specific to this safe output type
	steps = append(steps, config.CustomEnvVars...)

	// Add custom environment variables from safe-outputs.env
	c.addCustomSafeOutputEnvVars(&steps, data)

	// With section for github-token
	steps = append(steps, "        with:\n")
	if config.UseAgentToken {
		c.addSafeOutputAgentGitHubTokenForConfig(&steps, data, config.Token)
	} else if config.UseCopilotToken {
		c.addSafeOutputCopilotGitHubTokenForConfig(&steps, data, config.Token)
	} else {
		c.addSafeOutputGitHubTokenForConfig(&steps, data, config.Token)
	}

	steps = append(steps, "          script: |\n")

	// Add the formatted JavaScript script
	formattedScript := FormatJavaScriptForYAML(config.Script)
	steps = append(steps, formattedScript...)

	return steps
}

// buildAgentOutputDownloadSteps creates steps to download the agent output artifact
// and set the GH_AW_AGENT_OUTPUT environment variable for safe-output jobs
func buildAgentOutputDownloadSteps() []string {
	return buildArtifactDownloadSteps(ArtifactDownloadConfig{
		ArtifactName: "agent_output.json", // Use constant value directly to avoid import cycle
		DownloadPath: "/tmp/gh-aw/safeoutputs/",
		SetupEnvStep: true,
		EnvVarName:   "GH_AW_AGENT_OUTPUT",
		StepName:     "Download agent output artifact",
	})
}

// SafeOutputJobConfig holds configuration for building a safe output job
// This config struct extracts the common parameters across all safe output job builders
type SafeOutputJobConfig struct {
	// Job metadata
	JobName     string // e.g., "create_issue"
	StepName    string // e.g., "Create Output Issue"
	StepID      string // e.g., "create_issue"
	MainJobName string // Main workflow job name for dependencies

	// Custom environment variables specific to this safe output type
	CustomEnvVars []string

	// JavaScript script constant to include in the GitHub Script step
	Script string

	// Job configuration
	Permissions     *Permissions      // Job permissions
	Outputs         map[string]string // Job outputs
	Condition       ConditionNode     // Job condition (if clause)
	Needs           []string          // Job dependencies
	PreSteps        []string          // Optional steps to run before the GitHub Script step
	PostSteps       []string          // Optional steps to run after the GitHub Script step
	Token           string            // GitHub token for this output type
	UseCopilotToken bool              // Whether to use Copilot token preference chain
	UseAgentToken   bool              // Whether to use agent token preference chain (config token > GH_AW_AGENT_TOKEN)
	TargetRepoSlug  string            // Target repository for cross-repo operations
}

// buildSafeOutputJob creates a safe output job with common scaffolding
// This extracts the repeated pattern found across safe output job builders:
// 1. Validate configuration
// 2. Build custom environment variables
// 3. Invoke buildGitHubScriptStep
// 4. Create Job with standard metadata
func (c *Compiler) buildSafeOutputJob(data *WorkflowData, config SafeOutputJobConfig) (*Job, error) {
	safeOutputsLog.Printf("Building safe output job: %s", config.JobName)
	var steps []string

	// Add GitHub App token minting step if app is configured
	if data.SafeOutputs != nil && data.SafeOutputs.App != nil {
		safeOutputsLog.Print("Adding GitHub App token minting step with auto-computed permissions")
		steps = append(steps, c.buildGitHubAppTokenMintStep(data.SafeOutputs.App, config.Permissions)...)
	}

	// Add pre-steps if provided (e.g., checkout, git config for create-pull-request)
	if len(config.PreSteps) > 0 {
		safeOutputsLog.Printf("Adding %d pre-steps to job", len(config.PreSteps))
		steps = append(steps, config.PreSteps...)
	}

	// Build the GitHub Script step using the common helper
	scriptSteps := c.buildGitHubScriptStep(data, GitHubScriptStepConfig{
		StepName:        config.StepName,
		StepID:          config.StepID,
		MainJobName:     config.MainJobName,
		CustomEnvVars:   config.CustomEnvVars,
		Script:          config.Script,
		Token:           config.Token,
		UseCopilotToken: config.UseCopilotToken,
		UseAgentToken:   config.UseAgentToken,
	})
	steps = append(steps, scriptSteps...)

	// Add post-steps if provided (e.g., assignees, reviewers)
	if len(config.PostSteps) > 0 {
		steps = append(steps, config.PostSteps...)
	}

	// Add GitHub App token invalidation step if app is configured
	if data.SafeOutputs != nil && data.SafeOutputs.App != nil {
		safeOutputsLog.Print("Adding GitHub App token invalidation step")
		steps = append(steps, c.buildGitHubAppTokenInvalidationStep()...)
	}

	// Determine job condition
	jobCondition := config.Condition
	if jobCondition == nil {
		safeOutputsLog.Printf("No custom condition provided, using default for job: %s", config.JobName)
		jobCondition = BuildSafeOutputType(config.JobName)
	}

	// Determine job needs
	needs := config.Needs
	if len(needs) == 0 {
		needs = []string{config.MainJobName}
	}
	safeOutputsLog.Printf("Job %s needs: %v", config.JobName, needs)

	// Create the job with standard configuration
	job := &Job{
		Name:           config.JobName,
		If:             jobCondition.Render(),
		RunsOn:         c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions:    config.Permissions.RenderToYAML(),
		TimeoutMinutes: 10, // 10-minute timeout as required for all safe output jobs
		Steps:          steps,
		Outputs:        config.Outputs,
		Needs:          needs,
	}

	return job, nil
}

func generateSafeOutputsConfig(data *WorkflowData) string {
	// Pass the safe-outputs configuration for validation
	if data.SafeOutputs == nil {
		return ""
	}
	safeOutputsLog.Print("Generating safe outputs configuration for workflow")
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

	configJSON, _ := json.Marshal(safeOutputsConfig)
	return string(configJSON)
}

// generateFilteredToolsJSON filters the ALL_TOOLS array based on enabled safe outputs
// Returns a JSON string containing only the tools that are enabled in the workflow
func generateFilteredToolsJSON(data *WorkflowData) (string, error) {
	if data.SafeOutputs == nil {
		return "[]", nil
	}

	safeOutputsLog.Print("Generating filtered tools JSON for workflow")

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

	if safeOutputsLog.Enabled() {
		safeOutputsLog.Printf("Filtered %d tools from %d total tools", len(filteredTools), len(allTools))
	}

	// Marshal the filtered tools back to JSON with indentation for better readability
	// and to reduce merge conflicts in generated lockfiles
	filteredJSON, err := json.MarshalIndent(filteredTools, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal filtered tools: %w", err)
	}

	return string(filteredJSON), nil
}

// applySafeOutputEnvToMap adds safe-output related environment variables to an env map
// This extracts the duplicated safe-output env setup logic across all engines (copilot, codex, claude, custom)
func applySafeOutputEnvToMap(env map[string]string, data *WorkflowData) {
	if data.SafeOutputs == nil {
		return
	}

	env["GH_AW_SAFE_OUTPUTS"] = "${{ env.GH_AW_SAFE_OUTPUTS }}"

	// Add staged flag if specified
	if data.TrialMode || data.SafeOutputs.Staged {
		env["GH_AW_SAFE_OUTPUTS_STAGED"] = "true"
	}
	if data.TrialMode && data.TrialLogicalRepo != "" {
		env["GH_AW_TARGET_REPO_SLUG"] = data.TrialLogicalRepo
	}

	// Add branch name if upload assets is configured
	if data.SafeOutputs.UploadAssets != nil {
		env["GH_AW_ASSETS_BRANCH"] = fmt.Sprintf("%q", data.SafeOutputs.UploadAssets.BranchName)
		env["GH_AW_ASSETS_MAX_SIZE_KB"] = fmt.Sprintf("%d", data.SafeOutputs.UploadAssets.MaxSizeKB)
		env["GH_AW_ASSETS_ALLOWED_EXTS"] = fmt.Sprintf("%q", strings.Join(data.SafeOutputs.UploadAssets.AllowedExts, ","))
	}
}

// applySafeOutputEnvToSlice adds safe-output related environment variables to a YAML string slice
// This is for engines that build YAML line-by-line (like Claude)
func applySafeOutputEnvToSlice(stepLines *[]string, workflowData *WorkflowData) {
	if workflowData.SafeOutputs == nil {
		return
	}

	*stepLines = append(*stepLines, "          GH_AW_SAFE_OUTPUTS: ${{ env.GH_AW_SAFE_OUTPUTS }}")

	// Add staged flag if specified
	if workflowData.TrialMode || workflowData.SafeOutputs.Staged {
		*stepLines = append(*stepLines, "          GH_AW_SAFE_OUTPUTS_STAGED: \"true\"")
	}
	if workflowData.TrialMode && workflowData.TrialLogicalRepo != "" {
		*stepLines = append(*stepLines, fmt.Sprintf("          GH_AW_TARGET_REPO_SLUG: %q", workflowData.TrialLogicalRepo))
	}

	// Add branch name if upload assets is configured
	if workflowData.SafeOutputs.UploadAssets != nil {
		*stepLines = append(*stepLines, fmt.Sprintf("          GH_AW_ASSETS_BRANCH: %q", workflowData.SafeOutputs.UploadAssets.BranchName))
		*stepLines = append(*stepLines, fmt.Sprintf("          GH_AW_ASSETS_MAX_SIZE_KB: %d", workflowData.SafeOutputs.UploadAssets.MaxSizeKB))
		*stepLines = append(*stepLines, fmt.Sprintf("          GH_AW_ASSETS_ALLOWED_EXTS: %q", strings.Join(workflowData.SafeOutputs.UploadAssets.AllowedExts, ",")))
	}
}

// buildWorkflowMetadataEnvVars builds workflow name and source environment variables
// This extracts the duplicated workflow metadata setup logic from safe-output job builders
func buildWorkflowMetadataEnvVars(workflowName string, workflowSource string) []string {
	var customEnvVars []string

	// Add workflow name
	customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_WORKFLOW_NAME: %q\n", workflowName))

	// Add workflow source and source URL if present
	if workflowSource != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_WORKFLOW_SOURCE: %q\n", workflowSource))
		sourceURL := buildSourceURL(workflowSource)
		if sourceURL != "" {
			customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_WORKFLOW_SOURCE_URL: %q\n", sourceURL))
		}
	}

	return customEnvVars
}

// buildWorkflowMetadataEnvVarsWithTrackerID builds workflow metadata env vars including tracker-id
func buildWorkflowMetadataEnvVarsWithTrackerID(workflowName string, workflowSource string, trackerID string) []string {
	customEnvVars := buildWorkflowMetadataEnvVars(workflowName, workflowSource)

	// Add tracker-id if present
	if trackerID != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_TRACKER_ID: %q\n", trackerID))
	}

	return customEnvVars
}

// buildSafeOutputJobEnvVars builds environment variables for safe-output jobs with staged/target repo handling
// This extracts the duplicated env setup logic in safe-output job builders (create_issue, add_comment, etc.)
func buildSafeOutputJobEnvVars(trialMode bool, trialLogicalRepoSlug string, staged bool, targetRepoSlug string) []string {
	var customEnvVars []string

	// Pass the staged flag if it's set to true
	if trialMode || staged {
		customEnvVars = append(customEnvVars, "          GH_AW_SAFE_OUTPUTS_STAGED: \"true\"\n")
	}

	// Set GH_AW_TARGET_REPO_SLUG - prefer target-repo config over trial target repo
	if targetRepoSlug != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_TARGET_REPO_SLUG: %q\n", targetRepoSlug))
	} else if trialMode && trialLogicalRepoSlug != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_TARGET_REPO_SLUG: %q\n", trialLogicalRepoSlug))
	}

	return customEnvVars
}

// buildStandardSafeOutputEnvVars builds the standard set of environment variables
// that all safe-output job builders need: metadata + staged/target repo handling
// This reduces duplication in safe-output job builders
func (c *Compiler) buildStandardSafeOutputEnvVars(data *WorkflowData, targetRepoSlug string) []string {
	var customEnvVars []string

	// Add workflow metadata (name, source, and tracker-id)
	customEnvVars = append(customEnvVars, buildWorkflowMetadataEnvVarsWithTrackerID(data.Name, data.Source, data.TrackerID)...)

	// Add engine metadata (id, version, model) for XML comment marker
	customEnvVars = append(customEnvVars, buildEngineMetadataEnvVars(data.EngineConfig)...)

	// Add common safe output job environment variables (staged/target repo)
	customEnvVars = append(customEnvVars, buildSafeOutputJobEnvVars(
		c.trialMode,
		c.trialLogicalRepoSlug,
		data.SafeOutputs.Staged,
		targetRepoSlug,
	)...)

	// Add messages config if present
	if data.SafeOutputs.Messages != nil {
		messagesJSON, err := serializeMessagesConfig(data.SafeOutputs.Messages)
		if err != nil {
			safeOutputsLog.Printf("Warning: failed to serialize messages config: %v", err)
		} else if messagesJSON != "" {
			customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_SAFE_OUTPUT_MESSAGES: %q\n", messagesJSON))
		}
	}

	return customEnvVars
}

// buildEngineMetadataEnvVars builds engine metadata environment variables (id, version, model)
// These are used by the JavaScript footer generation to create XML comment markers for traceability
func buildEngineMetadataEnvVars(engineConfig *EngineConfig) []string {
	var customEnvVars []string

	if engineConfig == nil {
		return customEnvVars
	}

	// Add engine ID if present
	if engineConfig.ID != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_ENGINE_ID: %q\n", engineConfig.ID))
	}

	// Add engine version if present
	if engineConfig.Version != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_ENGINE_VERSION: %q\n", engineConfig.Version))
	}

	// Add engine model if present
	if engineConfig.Model != "" {
		customEnvVars = append(customEnvVars, fmt.Sprintf("          GH_AW_ENGINE_MODEL: %q\n", engineConfig.Model))
	}

	return customEnvVars
}
