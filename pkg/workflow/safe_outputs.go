package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
)

// formatSafeOutputsRunsOn formats the runs-on value from SafeOutputsConfig for job output
func (c *Compiler) formatSafeOutputsRunsOn(safeOutputs *SafeOutputsConfig) string {
	if safeOutputs == nil || safeOutputs.RunsOn == "" {
		return "runs-on: ubuntu-latest" // Default
	}

	return fmt.Sprintf("runs-on: %s", safeOutputs.RunsOn)
}

// HasSafeOutputsEnabled checks if any safe-outputs are enabled
func HasSafeOutputsEnabled(safeOutputs *SafeOutputsConfig) bool {
	if safeOutputs == nil {
		return false
	}
	return safeOutputs.CreateIssues != nil ||
		safeOutputs.CreateAgentTasks != nil ||
		safeOutputs.CreateDiscussions != nil ||
		safeOutputs.AddComments != nil ||
		safeOutputs.CreatePullRequests != nil ||
		safeOutputs.CreatePullRequestReviewComments != nil ||
		safeOutputs.CreateCodeScanningAlerts != nil ||
		safeOutputs.AddLabels != nil ||
		safeOutputs.UpdateIssues != nil ||
		safeOutputs.PushToPullRequestBranch != nil ||
		safeOutputs.UploadAssets != nil ||
		safeOutputs.MissingTool != nil ||
		len(safeOutputs.Jobs) > 0
}

// generateSafeOutputsPromptSection generates the safe-outputs instruction section for prompts
// when safe-outputs are configured, informing the agent about available output capabilities
func generateSafeOutputsPromptSection(yaml *strings.Builder, safeOutputs *SafeOutputsConfig) {
	if safeOutputs == nil {
		return
	}

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

// extractSafeOutputsConfig extracts output configuration from frontmatter
func (c *Compiler) extractSafeOutputsConfig(frontmatter map[string]any) *SafeOutputsConfig {
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

			// Handle create-discussion
			discussionsConfig := c.parseDiscussionsConfig(outputMap)
			if discussionsConfig != nil {
				config.CreateDiscussions = discussionsConfig
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

					// Parse min (optional)
					if minCount, exists := labelsMap["min"]; exists {
						// Handle different numeric types that YAML parsers might return
						var minCountInt int
						var validMinCount bool
						switch v := minCount.(type) {
						case int:
							minCountInt = v
							validMinCount = true
						case int64:
							minCountInt = int(v)
							validMinCount = true
						case uint64:
							minCountInt = int(v)
							validMinCount = true
						case float64:
							minCountInt = int(v)
							validMinCount = true
						}
						if validMinCount {
							labelConfig.Min = minCountInt
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
