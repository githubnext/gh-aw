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
		safeOutputs.AddComments != nil ||
		safeOutputs.CreatePullRequests != nil ||
		safeOutputs.CreatePullRequestReviewComments != nil ||
		safeOutputs.CreateCodeScanningAlerts != nil ||
		safeOutputs.AddLabels != nil ||
		safeOutputs.AssignMilestone != nil ||
		safeOutputs.AssignToAgent != nil ||
		safeOutputs.UpdateIssues != nil ||
		safeOutputs.PushToPullRequestBranch != nil ||
		safeOutputs.UploadAssets != nil ||
		safeOutputs.MissingTool != nil ||
		safeOutputs.NoOp != nil ||
		len(safeOutputs.Jobs) > 0

	if safeOutputsLog.Enabled() {
		safeOutputsLog.Printf("Safe outputs enabled check: %v", enabled)
	}

	return enabled
}

// ========================================
// Safe Output Prompt Generation
// ========================================

// generateSafeOutputsPromptSection generates the safe-outputs instruction section for prompts
// when safe-outputs are configured, informing the agent about available output capabilities
func generateSafeOutputsPromptSection(yaml *strings.Builder, safeOutputs *SafeOutputsConfig) {
	if safeOutputs == nil {
		return
	}

	safeOutputsLog.Print("Generating safe outputs prompt section")

	// Add output instructions for all engines (GH_AW_SAFE_OUTPUTS functionality)
	yaml.WriteString("          \n")
	yaml.WriteString("          ---\n")
	yaml.WriteString("          \n")
	yaml.WriteString("          ## ")
	written := false
	if safeOutputs.AddComments != nil {
		yaml.WriteString("Adding a Comment to an Issue or Pull Request")
		written = true
	}
	if safeOutputs.CreateIssues != nil {
		if written {
			yaml.WriteString(", ")
		}
		yaml.WriteString("Creating an Issue")
		written = true
	}
	if safeOutputs.CreateAgentTasks != nil {
		if written {
			yaml.WriteString(", ")
		}
		yaml.WriteString("Creating an Agent Task")
		written = true
	}
	if safeOutputs.CreatePullRequests != nil {
		if written {
			yaml.WriteString(", ")
		}
		yaml.WriteString("Creating a Pull Request")
		written = true
	}

	if safeOutputs.AddLabels != nil {
		if written {
			yaml.WriteString(", ")
		}
		yaml.WriteString("Adding Labels to Issues or Pull Requests")
		written = true
	}

	if safeOutputs.UpdateIssues != nil {
		if written {
			yaml.WriteString(", ")
		}
		yaml.WriteString("Updating Issues")
		written = true
	}

	if safeOutputs.PushToPullRequestBranch != nil {
		if written {
			yaml.WriteString(", ")
		}
		yaml.WriteString("Pushing Changes to Branch")
		written = true
	}

	if safeOutputs.CreateCodeScanningAlerts != nil {
		if written {
			yaml.WriteString(", ")
		}
		yaml.WriteString("Creating Code Scanning Alert")
		written = true
	}

	if safeOutputs.UploadAssets != nil {
		if written {
			yaml.WriteString(", ")
		}
		yaml.WriteString("Uploading Assets")
		written = true
	}

	// Missing-tool is enabled by default when safe-outputs is configured
	if safeOutputs.MissingTool != nil {
		if written {
			yaml.WriteString(", ")
		}
		yaml.WriteString("Reporting Missing Tools or Functionality")
	}

	yaml.WriteString("\n")
	yaml.WriteString("          \n")
	yaml.WriteString(fmt.Sprintf("          **IMPORTANT**: To do the actions mentioned in the header of this section, use the **%s** tools, do NOT attempt to use `gh`, do NOT attempt to use the GitHub API. You don't have write access to the GitHub repo.\n", constants.SafeOutputsMCPServerID))
	yaml.WriteString("          \n")

	if safeOutputs.AddComments != nil {
		yaml.WriteString("          **Adding a Comment to an Issue or Pull Request**\n")
		yaml.WriteString("          \n")
		yaml.WriteString(fmt.Sprintf("          To add a comment to an issue or pull request, use the add-comments tool from %s\n", constants.SafeOutputsMCPServerID))
		yaml.WriteString("          \n")
	}

	if safeOutputs.CreateIssues != nil {
		yaml.WriteString("          **Creating an Issue**\n")
		yaml.WriteString("          \n")
		yaml.WriteString(fmt.Sprintf("          To create an issue, use the create-issue tool from %s\n", constants.SafeOutputsMCPServerID))
		yaml.WriteString("          \n")
	}

	if safeOutputs.CreateAgentTasks != nil {
		yaml.WriteString("          **Creating an Agent Task**\n")
		yaml.WriteString("          \n")
		yaml.WriteString(fmt.Sprintf("          To create a GitHub Copilot agent task, use the create-agent-task tool from %s\n", constants.SafeOutputsMCPServerID))
		yaml.WriteString("          \n")
	}

	if safeOutputs.CreatePullRequests != nil {
		yaml.WriteString("          **Creating a Pull Request**\n")
		yaml.WriteString("          \n")
		yaml.WriteString("          To create a pull request:\n")
		yaml.WriteString("          1. Make any file changes directly in the working directory\n")
		yaml.WriteString("          2. If you haven't done so already, create a local branch using an appropriate unique name\n")
		yaml.WriteString("          3. Add and commit your changes to the branch. Be careful to add exactly the files you intend, and check there are no extra files left un-added. Check you haven't deleted or changed any files you didn't intend to.\n")
		yaml.WriteString("          4. Do not push your changes. That will be done by the tool.\n")
		yaml.WriteString(fmt.Sprintf("          5. Create the pull request with the create-pull-request tool from %s\n", constants.SafeOutputsMCPServerID))
		yaml.WriteString("          \n")
	}

	if safeOutputs.AddLabels != nil {
		yaml.WriteString("          **Adding Labels to Issues or Pull Requests**\n")
		yaml.WriteString("          \n")
		yaml.WriteString(fmt.Sprintf("          To add labels to an issue or a pull request, use the add-labels tool from %s\n", constants.SafeOutputsMCPServerID))
		yaml.WriteString("          \n")
	}

	if safeOutputs.UpdateIssues != nil {
		yaml.WriteString("          **Updating an Issue**\n")
		yaml.WriteString("          \n")
		yaml.WriteString(fmt.Sprintf("          To udpate an issue, use the update-issue tool from %s\n", constants.SafeOutputsMCPServerID))
		yaml.WriteString("          \n")
	}

	if safeOutputs.PushToPullRequestBranch != nil {
		yaml.WriteString("          **Pushing Changes to Pull Request Branch**\n")
		yaml.WriteString("          \n")
		yaml.WriteString("          To push changes to the branch of a pull request:\n")
		yaml.WriteString("          1. Make any file changes directly in the working directory\n")
		yaml.WriteString("          2. Add and commit your changes to the local copy of the pull request branch. Be careful to add exactly the files you intend, and check there are no extra files left un-added. Check you haven't deleted or changed any files you didn't intend to.\n")
		yaml.WriteString(fmt.Sprintf("          3. Push the branch to the repo by using the push-to-pull-request-branch tool from %s\n", constants.SafeOutputsMCPServerID))
		yaml.WriteString("          \n")
	}

	if safeOutputs.CreateCodeScanningAlerts != nil {
		yaml.WriteString("          **Creating Code Scanning Alert**\n")
		yaml.WriteString("          \n")
		yaml.WriteString(fmt.Sprintf("          To create code scanning alert use the create-code-scanning-alert tool from %s\n", constants.SafeOutputsMCPServerID))
		yaml.WriteString("          \n")
	}

	if safeOutputs.UploadAssets != nil {
		yaml.WriteString("          **Uploading Assets**\n")
		yaml.WriteString("          \n")
		yaml.WriteString("          To upload files as URL-addressable assets:\n")
		yaml.WriteString(fmt.Sprintf("          1. Use the `upload asset` tool from %s\n", constants.SafeOutputsMCPServerID))
		yaml.WriteString("          2. Provide the path to the file you want to upload\n")
		yaml.WriteString("          3. The tool will copy the file to a staging area and return a GitHub raw content URL\n")
		yaml.WriteString("          4. Assets are uploaded to an orphaned git branch after workflow completion\n")
		yaml.WriteString("          \n")
	}

	// Missing-tool instructions are only included when configured
	if safeOutputs.MissingTool != nil {
		yaml.WriteString("          **Reporting Missing Tools or Functionality**\n")
		yaml.WriteString("          \n")
		yaml.WriteString(fmt.Sprintf("          To report a missing tool use the missing-tool tool from %s.\n", constants.SafeOutputsMCPServerID))
		yaml.WriteString("          \n")
	}

	if safeOutputs.CreatePullRequestReviewComments != nil {
		yaml.WriteString("          **Creating a Pull Request Review Comment**\n")
		yaml.WriteString("          \n")
		yaml.WriteString(fmt.Sprintf("          To create a pull request review comment, use the create-pull-request-review-comment tool from %s\n", constants.SafeOutputsMCPServerID))
		yaml.WriteString("          \n")
	}
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

					// Parse allowed labels (optional)
					if allowed, exists := labelsMap["allowed"]; exists {
						if allowedArray, ok := allowed.([]any); ok {
							var allowedStrings []string
							for _, label := range allowedArray {
								if labelStr, ok := label.(string); ok {
									allowedStrings = append(allowedStrings, labelStr)
								}
							}
							labelConfig.Allowed = allowedStrings
						}
					}

					// Parse max (optional)
					if maxCount, exists := labelsMap["max"]; exists {
						// Handle different numeric types that YAML parsers might return
						var maxCountInt int
						var validMaxCount bool
						switch v := maxCount.(type) {
						case int:
							maxCountInt = v
							validMaxCount = true
						case int64:
							maxCountInt = int(v)
							validMaxCount = true
						case uint64:
							maxCountInt = int(v)
							validMaxCount = true
						case float64:
							maxCountInt = int(v)
							validMaxCount = true
						}
						if validMaxCount {
							labelConfig.Max = maxCountInt
						}
					}

					// Parse github-token
					if githubToken, exists := labelsMap["github-token"]; exists {
						if githubTokenStr, ok := githubToken.(string); ok {
							labelConfig.GitHubToken = githubTokenStr
						}
					}

					// Parse target
					if target, exists := labelsMap["target"]; exists {
						if targetStr, ok := target.(string); ok {
							labelConfig.Target = targetStr
						}
					}

					// Parse target-repo
					if targetRepo, exists := labelsMap["target-repo"]; exists {
						if targetRepoStr, ok := targetRepo.(string); ok {
							labelConfig.TargetRepoSlug = targetRepoStr
						}
					}

					config.AddLabels = labelConfig
				} else if labels == nil {
					// Handle null case: create empty config (allows any labels)
					config.AddLabels = &AddLabelsConfig{}
				}
			}

			// Parse assign-milestone configuration
			if milestone, exists := outputMap["assign-milestone"]; exists {
				if milestoneMap, ok := milestone.(map[string]any); ok {
					milestoneConfig := &AssignMilestoneConfig{}

					// Parse allowed milestones (optional)
					if allowed, exists := milestoneMap["allowed"]; exists {
						if allowedArray, ok := allowed.([]any); ok {
							var allowedStrings []string
							for _, ms := range allowedArray {
								if msStr, ok := ms.(string); ok {
									allowedStrings = append(allowedStrings, msStr)
								}
							}
							milestoneConfig.Allowed = allowedStrings
						}
					}

					// Parse max (optional)
					if maxCount, exists := milestoneMap["max"]; exists {
						// Handle different numeric types that YAML parsers might return
						var maxCountInt int
						var validMaxCount bool
						switch v := maxCount.(type) {
						case int:
							maxCountInt = v
							validMaxCount = true
						case int64:
							maxCountInt = int(v)
							validMaxCount = true
						case uint64:
							maxCountInt = int(v)
							validMaxCount = true
						case float64:
							maxCountInt = int(v)
							validMaxCount = true
						}
						if validMaxCount {
							milestoneConfig.Max = maxCountInt
						}
					}

					// Parse github-token
					if githubToken, exists := milestoneMap["github-token"]; exists {
						if githubTokenStr, ok := githubToken.(string); ok {
							milestoneConfig.GitHubToken = githubTokenStr
						}
					}

					// Parse target
					if target, exists := milestoneMap["target"]; exists {
						if targetStr, ok := target.(string); ok {
							milestoneConfig.Target = targetStr
						}
					}

					// Parse target-repo
					if targetRepo, exists := milestoneMap["target-repo"]; exists {
						if targetRepoStr, ok := targetRepo.(string); ok {
							milestoneConfig.TargetRepoSlug = targetRepoStr
						}
					}

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

					// Parse default-agent (optional)
					if defaultAgent, exists := agentMap["default-agent"]; exists {
						if defaultAgentStr, ok := defaultAgent.(string); ok {
							agentConfig.DefaultAgent = defaultAgentStr
						}
					}

					// Parse max (optional)
					if maxCount, exists := agentMap["max"]; exists {
						// Handle different numeric types that YAML parsers might return
						var maxCountInt int
						var validMaxCount bool
						switch v := maxCount.(type) {
						case int:
							maxCountInt = v
							validMaxCount = true
						case int64:
							maxCountInt = int(v)
							validMaxCount = true
						case uint64:
							maxCountInt = int(v)
							validMaxCount = true
						case float64:
							maxCountInt = int(v)
							validMaxCount = true
						}
						if validMaxCount {
							agentConfig.Max = maxCountInt
						}
					}

					// Parse github-token
					if githubToken, exists := agentMap["github-token"]; exists {
						if githubTokenStr, ok := githubToken.(string); ok {
							agentConfig.GitHubToken = githubTokenStr
						}
					}

					// Parse target-repo
					if targetRepo, exists := agentMap["target-repo"]; exists {
						if targetRepoStr, ok := targetRepo.(string); ok {
							agentConfig.TargetRepoSlug = targetRepoStr
						}
					}

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

			// Handle jobs (safe-jobs moved under safe-outputs)
			if jobs, exists := outputMap["jobs"]; exists {
				if jobsMap, ok := jobs.(map[string]any); ok {
					c := &Compiler{} // Create a temporary compiler instance for parsing
					jobsFrontmatter := map[string]any{"safe-jobs": jobsMap}
					config.Jobs = c.parseSafeJobsConfig(jobsFrontmatter)
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
	return strings.ReplaceAll(identifier, "-", "_")
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
}

// buildGitHubScriptStep creates a GitHub Script step with common scaffolding
// This extracts the repeated pattern found across safe output job builders
func (c *Compiler) buildGitHubScriptStep(data *WorkflowData, config GitHubScriptStepConfig) []string {
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
	if config.UseCopilotToken {
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
	if config.UseCopilotToken {
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
	})
	steps = append(steps, scriptSteps...)

	// Add post-steps if provided (e.g., assignees, reviewers)
	if len(config.PostSteps) > 0 {
		steps = append(steps, config.PostSteps...)
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
			if data.SafeOutputs.CreateIssues.Max > 0 {
				issueConfig["max"] = data.SafeOutputs.CreateIssues.Max
			}
			safeOutputsConfig["create_issue"] = issueConfig
		}
		if data.SafeOutputs.CreateAgentTasks != nil {
			agentTaskConfig := map[string]any{}
			if data.SafeOutputs.CreateAgentTasks.Max > 0 {
				agentTaskConfig["max"] = data.SafeOutputs.CreateAgentTasks.Max
			}
			safeOutputsConfig["create_agent_task"] = agentTaskConfig
		}
		if data.SafeOutputs.AddComments != nil {
			commentConfig := map[string]any{}
			if data.SafeOutputs.AddComments.Target != "" {
				commentConfig["target"] = data.SafeOutputs.AddComments.Target
			}
			if data.SafeOutputs.AddComments.Max > 0 {
				commentConfig["max"] = data.SafeOutputs.AddComments.Max
			}
			safeOutputsConfig["add_comment"] = commentConfig
		}
		if data.SafeOutputs.CreateDiscussions != nil {
			discussionConfig := map[string]any{}
			if data.SafeOutputs.CreateDiscussions.Max > 0 {
				discussionConfig["max"] = data.SafeOutputs.CreateDiscussions.Max
			}
			safeOutputsConfig["create_discussion"] = discussionConfig
		}
		if data.SafeOutputs.CloseDiscussions != nil {
			closeDiscussionConfig := map[string]any{}
			if data.SafeOutputs.CloseDiscussions.Max > 0 {
				closeDiscussionConfig["max"] = data.SafeOutputs.CloseDiscussions.Max
			}
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
		if data.SafeOutputs.CreatePullRequests != nil {
			prConfig := map[string]any{}
			// Note: max is always 1 for pull requests, not configurable
			safeOutputsConfig["create_pull_request"] = prConfig
		}
		if data.SafeOutputs.CreatePullRequestReviewComments != nil {
			prReviewCommentConfig := map[string]any{}
			if data.SafeOutputs.CreatePullRequestReviewComments.Max > 0 {
				prReviewCommentConfig["max"] = data.SafeOutputs.CreatePullRequestReviewComments.Max
			}
			safeOutputsConfig["create_pull_request_review_comment"] = prReviewCommentConfig
		}
		if data.SafeOutputs.CreateCodeScanningAlerts != nil {
			// Security reports typically have unlimited max, but check if configured
			securityReportConfig := map[string]any{}
			if data.SafeOutputs.CreateCodeScanningAlerts.Max > 0 {
				securityReportConfig["max"] = data.SafeOutputs.CreateCodeScanningAlerts.Max
			}
			safeOutputsConfig["create_code_scanning_alert"] = securityReportConfig
		}
		if data.SafeOutputs.AddLabels != nil {
			labelConfig := map[string]any{}
			if data.SafeOutputs.AddLabels.Max > 0 {
				labelConfig["max"] = data.SafeOutputs.AddLabels.Max
			}
			if len(data.SafeOutputs.AddLabels.Allowed) > 0 {
				labelConfig["allowed"] = data.SafeOutputs.AddLabels.Allowed
			}
			safeOutputsConfig["add_labels"] = labelConfig
		}
		if data.SafeOutputs.AssignMilestone != nil {
			assignMilestoneConfig := map[string]any{}
			if data.SafeOutputs.AssignMilestone.Max > 0 {
				assignMilestoneConfig["max"] = data.SafeOutputs.AssignMilestone.Max
			}
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
			if data.SafeOutputs.UpdateIssues.Max > 0 {
				updateConfig["max"] = data.SafeOutputs.UpdateIssues.Max
			}
			safeOutputsConfig["update_issue"] = updateConfig
		}
		if data.SafeOutputs.PushToPullRequestBranch != nil {
			pushToBranchConfig := map[string]any{}
			if data.SafeOutputs.PushToPullRequestBranch.Target != "" {
				pushToBranchConfig["target"] = data.SafeOutputs.PushToPullRequestBranch.Target
			}
			if data.SafeOutputs.PushToPullRequestBranch.Max > 0 {
				pushToBranchConfig["max"] = data.SafeOutputs.PushToPullRequestBranch.Max
			}
			safeOutputsConfig["push_to_pull_request_branch"] = pushToBranchConfig
		}
		if data.SafeOutputs.UploadAssets != nil {
			uploadConfig := map[string]any{}
			if data.SafeOutputs.UploadAssets.Max > 0 {
				uploadConfig["max"] = data.SafeOutputs.UploadAssets.Max
			}
			safeOutputsConfig["upload_asset"] = uploadConfig
		}
		if data.SafeOutputs.MissingTool != nil {
			missingToolConfig := map[string]any{}
			if data.SafeOutputs.MissingTool.Max > 0 {
				missingToolConfig["max"] = data.SafeOutputs.MissingTool.Max
			}
			safeOutputsConfig["missing_tool"] = missingToolConfig
		}
		if data.SafeOutputs.UpdateProjects != nil {
			updateProjectConfig := map[string]any{}
			if data.SafeOutputs.UpdateProjects.Max > 0 {
				updateProjectConfig["max"] = data.SafeOutputs.UpdateProjects.Max
			}
			safeOutputsConfig["update_project"] = updateProjectConfig
		}
		if data.SafeOutputs.UpdateRelease != nil {
			updateReleaseConfig := map[string]any{}
			if data.SafeOutputs.UpdateRelease.Max > 0 {
				updateReleaseConfig["max"] = data.SafeOutputs.UpdateRelease.Max
			}
			safeOutputsConfig["update_release"] = updateReleaseConfig
		}
		if data.SafeOutputs.NoOp != nil {
			noopConfig := map[string]any{}
			if data.SafeOutputs.NoOp.Max > 0 {
				noopConfig["max"] = data.SafeOutputs.NoOp.Max
			}
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
	if data.SafeOutputs.AssignMilestone != nil {
		enabledTools["assign_milestone"] = true
	}
	if data.SafeOutputs.AssignToAgent != nil {
		enabledTools["assign_to_agent"] = true
	}
	if data.SafeOutputs.UpdateIssues != nil {
		enabledTools["update_issue"] = true
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

	// Filter tools to only include enabled ones
	var filteredTools []map[string]any
	for _, tool := range allTools {
		toolName, ok := tool["name"].(string)
		if !ok {
			continue
		}
		if enabledTools[toolName] {
			filteredTools = append(filteredTools, tool)
		}
	}

	if safeOutputsLog.Enabled() {
		safeOutputsLog.Printf("Filtered %d tools from %d total tools", len(filteredTools), len(allTools))
	}

	// Marshal the filtered tools back to JSON
	filteredJSON, err := json.Marshal(filteredTools)
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

	// Add common safe output job environment variables (staged/target repo)
	customEnvVars = append(customEnvVars, buildSafeOutputJobEnvVars(
		c.trialMode,
		c.trialLogicalRepoSlug,
		data.SafeOutputs.Staged,
		targetRepoSlug,
	)...)

	return customEnvVars
}
