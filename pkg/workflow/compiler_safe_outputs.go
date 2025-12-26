package workflow

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/goccy/go-yaml"
)

var compilerSafeOutputsLog = logger.New("workflow:compiler_safe_outputs")

// parseOnSection handles parsing of the "on" section from frontmatter, extracting command triggers,
// reactions, and stop-after configurations while detecting conflicts with other event types.
func (c *Compiler) parseOnSection(frontmatter map[string]any, workflowData *WorkflowData, markdownPath string) error {
	compilerSafeOutputsLog.Printf("Parsing on section: workflow=%s, markdownPath=%s", workflowData.Name, markdownPath)
	// Check if "slash_command" or "command" (deprecated) is used as a trigger in the "on" section
	// Also extract "reaction" from the "on" section
	var hasCommand bool
	var hasReaction bool
	var hasStopAfter bool
	var otherEvents map[string]any

	if onValue, exists := frontmatter["on"]; exists {
		// Check for new format: on.slash_command/on.command and on.reaction
		if onMap, ok := onValue.(map[string]any); ok {
			// Check for stop-after in the on section
			if _, hasStopAfterKey := onMap["stop-after"]; hasStopAfterKey {
				hasStopAfter = true
			}

			// Extract reaction from on section
			if reactionValue, hasReactionField := onMap["reaction"]; hasReactionField {
				hasReaction = true
				reactionStr, err := parseReactionValue(reactionValue)
				if err != nil {
					return err
				}
				// Validate reaction value
				if !isValidReaction(reactionStr) {
					return fmt.Errorf("invalid reaction value '%s': must be one of %v", reactionStr, getValidReactions())
				}
				// Set AIReaction even if it's "none" - "none" explicitly disables reactions
				workflowData.AIReaction = reactionStr
			}

			// Extract lock-for-agent from on.issues section
			if issuesValue, hasIssues := onMap["issues"]; hasIssues {
				if issuesMap, ok := issuesValue.(map[string]any); ok {
					if lockForAgent, hasLockForAgent := issuesMap["lock-for-agent"]; hasLockForAgent {
						if lockBool, ok := lockForAgent.(bool); ok {
							workflowData.LockForAgent = lockBool
							compilerSafeOutputsLog.Printf("lock-for-agent enabled for issues: %v", lockBool)
						}
					}
				}
			}

			// Extract lock-for-agent from on.issue_comment section
			if issueCommentValue, hasIssueComment := onMap["issue_comment"]; hasIssueComment {
				if issueCommentMap, ok := issueCommentValue.(map[string]any); ok {
					if lockForAgent, hasLockForAgent := issueCommentMap["lock-for-agent"]; hasLockForAgent {
						if lockBool, ok := lockForAgent.(bool); ok {
							workflowData.LockForAgent = lockBool
							compilerSafeOutputsLog.Printf("lock-for-agent enabled for issue_comment: %v", lockBool)
						}
					}
				}
			}

			// Check for slash_command (preferred) or command (deprecated)
			if _, hasSlashCommandKey := onMap["slash_command"]; hasSlashCommandKey {
				hasCommand = true
				// Set default command to filename if not specified in the command section
				if workflowData.Command == "" {
					baseName := strings.TrimSuffix(filepath.Base(markdownPath), ".md")
					workflowData.Command = baseName
				}
				// Check for conflicting events (but allow issues/pull_request with labeled/unlabeled types)
				conflictingEvents := []string{"issues", "issue_comment", "pull_request", "pull_request_review_comment"}
				for _, eventName := range conflictingEvents {
					if eventValue, hasConflict := onMap[eventName]; hasConflict {
						// Special case: allow issues/pull_request if they only have labeled/unlabeled types
						if (eventName == "issues" || eventName == "pull_request") && parser.IsLabelOnlyEvent(eventValue) {
							continue // Allow this - it doesn't conflict with command triggers
						}
						return fmt.Errorf("cannot use 'slash_command' with '%s' in the same workflow", eventName)
					}
				}

				// Clear the On field so applyDefaults will handle command trigger generation
				workflowData.On = ""
			} else if _, hasCommandKey := onMap["command"]; hasCommandKey {
				hasCommand = true
				// Set default command to filename if not specified in the command section
				if workflowData.Command == "" {
					baseName := strings.TrimSuffix(filepath.Base(markdownPath), ".md")
					workflowData.Command = baseName
				}
				// Check for conflicting events (but allow issues/pull_request with labeled/unlabeled types)
				conflictingEvents := []string{"issues", "issue_comment", "pull_request", "pull_request_review_comment"}
				for _, eventName := range conflictingEvents {
					if eventValue, hasConflict := onMap[eventName]; hasConflict {
						// Special case: allow issues/pull_request if they only have labeled/unlabeled types
						if (eventName == "issues" || eventName == "pull_request") && parser.IsLabelOnlyEvent(eventValue) {
							continue // Allow this - it doesn't conflict with command triggers
						}
						return fmt.Errorf("cannot use 'command' with '%s' in the same workflow", eventName)
					}
				}

				// Clear the On field so applyDefaults will handle command trigger generation
				workflowData.On = ""
			}
			// Extract other (non-conflicting) events excluding slash_command, command, reaction, and stop-after
			otherEvents = filterMapKeys(onMap, "slash_command", "command", "reaction", "stop-after")
		}
	}

	// Clear command field if no command trigger was found
	if !hasCommand {
		workflowData.Command = ""
	}

	// Auto-enable "eyes" reaction for command triggers if no explicit reaction was specified
	if hasCommand && !hasReaction && workflowData.AIReaction == "" {
		workflowData.AIReaction = "eyes"
	}

	// Store other events for merging in applyDefaults
	if hasCommand && len(otherEvents) > 0 {
		// We'll store this and handle it in applyDefaults
		workflowData.On = "" // This will trigger command handling in applyDefaults
		workflowData.CommandOtherEvents = otherEvents
	} else if (hasReaction || hasStopAfter) && len(otherEvents) > 0 {
		// Only re-marshal the "on" if we have to
		onEventsYAML, err := yaml.Marshal(map[string]any{"on": otherEvents})
		if err == nil {
			yamlStr := strings.TrimSuffix(string(onEventsYAML), "\n")
			// Post-process YAML to ensure cron expressions are quoted
			yamlStr = parser.QuoteCronExpressions(yamlStr)
			// Apply comment processing to filter fields (draft, forks, names)
			yamlStr = c.commentOutProcessedFieldsInOnSection(yamlStr, frontmatter)
			// Add zizmor ignore comment if workflow_run trigger is present
			yamlStr = c.addZizmorIgnoreForWorkflowRun(yamlStr)
			// Keep "on" quoted as it's a YAML boolean keyword
			workflowData.On = yamlStr
		} else {
			// Fallback to extracting the original on field (this will include reaction but shouldn't matter for compilation)
			workflowData.On = c.extractTopLevelYAMLSection(frontmatter, "on")
		}
	}

	return nil
}

// generateJobName converts a workflow name to a valid YAML job identifier
func (c *Compiler) generateJobName(workflowName string) string {
	// Convert to lowercase and replace spaces and special characters with hyphens
	jobName := strings.ToLower(workflowName)

	// Replace spaces and common punctuation with hyphens
	jobName = strings.ReplaceAll(jobName, " ", "-")
	jobName = strings.ReplaceAll(jobName, ":", "-")
	jobName = strings.ReplaceAll(jobName, ".", "-")
	jobName = strings.ReplaceAll(jobName, ",", "-")
	jobName = strings.ReplaceAll(jobName, "(", "-")
	jobName = strings.ReplaceAll(jobName, ")", "-")
	jobName = strings.ReplaceAll(jobName, "/", "-")
	jobName = strings.ReplaceAll(jobName, "\\", "-")
	jobName = strings.ReplaceAll(jobName, "@", "-")
	jobName = strings.ReplaceAll(jobName, "'", "")
	jobName = strings.ReplaceAll(jobName, "\"", "")

	// Remove multiple consecutive hyphens
	for strings.Contains(jobName, "--") {
		jobName = strings.ReplaceAll(jobName, "--", "-")
	}

	// Remove leading/trailing hyphens
	jobName = strings.Trim(jobName, "-")

	// Ensure it's not empty and starts with a letter or underscore
	if jobName == "" || (!strings.ContainsAny(string(jobName[0]), "abcdefghijklmnopqrstuvwxyz_")) {
		jobName = "workflow-" + jobName
	}

	return jobName
}

// mergeSafeJobsFromIncludes merges safe-jobs from included files and detects conflicts
func (c *Compiler) mergeSafeJobsFromIncludes(topSafeJobs map[string]*SafeJobConfig, includedContentJSON string) (map[string]*SafeJobConfig, error) {
	if includedContentJSON == "" || includedContentJSON == "{}" {
		return topSafeJobs, nil
	}

	// Parse the included content as frontmatter to extract safe-jobs
	var includedContent map[string]any
	if err := json.Unmarshal([]byte(includedContentJSON), &includedContent); err != nil {
		return topSafeJobs, nil // Return original safe-jobs if parsing fails
	}

	// Extract safe-jobs from the included content
	includedSafeJobs := extractSafeJobsFromFrontmatter(includedContent)

	// Merge with conflict detection
	mergedSafeJobs, err := mergeSafeJobs(topSafeJobs, includedSafeJobs)
	if err != nil {
		return nil, fmt.Errorf("failed to merge safe-jobs: %w", err)
	}

	return mergedSafeJobs, nil
}

// mergeSafeJobsFromIncludedConfigs merges safe-jobs from included safe-outputs configurations
func (c *Compiler) mergeSafeJobsFromIncludedConfigs(topSafeJobs map[string]*SafeJobConfig, includedConfigs []string) (map[string]*SafeJobConfig, error) {
	compilerSafeOutputsLog.Printf("Merging safe-jobs from included configs: includedCount=%d", len(includedConfigs))
	result := topSafeJobs
	if result == nil {
		result = make(map[string]*SafeJobConfig)
	}

	for _, configJSON := range includedConfigs {
		if configJSON == "" || configJSON == "{}" {
			continue
		}

		// Parse the safe-outputs configuration
		var safeOutputsConfig map[string]any
		if err := json.Unmarshal([]byte(configJSON), &safeOutputsConfig); err != nil {
			continue // Skip invalid JSON
		}

		// Extract safe-jobs from the safe-outputs.jobs field
		includedSafeJobs := extractSafeJobsFromFrontmatter(map[string]any{
			"safe-outputs": safeOutputsConfig,
		})

		// Merge with conflict detection
		var err error
		result, err = mergeSafeJobs(result, includedSafeJobs)
		if err != nil {
			return nil, fmt.Errorf("failed to merge safe-jobs from includes: %w", err)
		}
	}

	return result, nil
}

// applyDefaultTools adds default read-only GitHub MCP tools, creating github tool if not present
func (c *Compiler) applyDefaultTools(toolsConfig *ToolsConfig, safeOutputs *SafeOutputsConfig) *ToolsConfig {
	compilerSafeOutputsLog.Printf("Applying default tools: existingToolCount=%d", len(toolsConfig.GetToolNames()))
	// Always apply default GitHub tools (create github section if it doesn't exist)

	if toolsConfig == nil {
		toolsConfig = &ToolsConfig{
			Custom: make(map[string]any),
			raw:    make(map[string]any),
		}
	}

	// Get existing github tool configuration - GitHub is already parsed
	// Check if github is explicitly disabled by checking if it's nil when it should be set
	// We need to check the raw map to determine if github was explicitly set to false
	if toolsConfig.raw != nil {
		if githubRaw, exists := toolsConfig.raw["github"]; exists && githubRaw == false {
			// Remove the github tool entirely when set to false
			toolsConfig.GitHub = nil
			delete(toolsConfig.raw, "github")
		}
	}

	// Process github tool configuration if it exists
	if toolsConfig.GitHub != nil {
		// Only set allowed tools if explicitly configured
		// Don't add default tools - let the MCP server use all available tools
		// The GitHub field is already properly configured by ParseToolsConfig
	} else if toolsConfig.raw == nil || toolsConfig.raw["github"] != false {
		// GitHub tool doesn't exist and wasn't explicitly disabled - create default config
		toolsConfig.GitHub = &GitHubToolConfig{
			ReadOnly: true, // default to read-only for security
		}
	}

	// Add Git commands and file editing tools when safe-outputs includes create-pull-request or push-to-pull-request-branch
	if safeOutputs != nil && needsGitCommands(safeOutputs) {

		// Add edit tool with null value
		if toolsConfig.Edit == nil {
			toolsConfig.Edit = &EditToolConfig{}
		}

		gitCommands := []string{
			"git checkout:*",
			"git branch:*",
			"git switch:*",
			"git add:*",
			"git rm:*",
			"git commit:*",
			"git merge:*",
			"git status",
		}

		// Add bash tool with Git commands if not already present
		if toolsConfig.Bash == nil {
			// bash tool doesn't exist, add it with Git commands
			toolsConfig.Bash = &BashToolConfig{
				AllowedCommands: gitCommands,
			}
		} else {
			// bash tool exists, merge Git commands with existing commands
			existingCommands := toolsConfig.Bash.AllowedCommands
			// Convert existing commands to strings for comparison
			existingSet := make(map[string]bool)
			for _, cmd := range existingCommands {
				existingSet[cmd] = true
				// If we see :* or *, all bash commands are already allowed
				if cmd == ":*" || cmd == "*" {
					// Don't add specific Git commands since all are already allowed
					goto bashComplete
				}
			}

			// Add Git commands that aren't already present
			newCommands := make([]string, 0, len(existingCommands)+len(gitCommands))
			newCommands = append(newCommands, existingCommands...)
			for _, gitCmd := range gitCommands {
				if !existingSet[gitCmd] {
					newCommands = append(newCommands, gitCmd)
				}
			}
			toolsConfig.Bash.AllowedCommands = newCommands
		}
	bashComplete:
	}

	// Add default bash commands when bash is enabled but no specific commands are provided
	// This runs after git commands logic, so it only applies when git commands weren't added
	// Behavior:
	//   - bash: true → All commands allowed (converted to ["*"])
	//   - bash: false → Tool disabled (removed from tools)
	//   - bash: nil → Add default commands
	//   - bash: [] → No commands (empty array means no tools allowed)
	//   - bash: ["cmd1", "cmd2"] → Add default commands + specific commands
	if toolsConfig.Bash != nil {
		// Check the raw map to determine if bash was set to boolean values
		if toolsConfig.raw != nil {
			if bashRaw, exists := toolsConfig.raw["bash"]; exists {
				if bashRaw == true {
					// bash is true - convert to wildcard (allow all commands)
					toolsConfig.Bash = &BashToolConfig{
						AllowedCommands: []string{"*"},
					}
				} else if bashRaw == false {
					// bash is false - disable the tool by removing it
					toolsConfig.Bash = nil
					delete(toolsConfig.raw, "bash")
				}
			}
		}

		// Process bash array - merge default commands with custom commands if needed
		if toolsConfig.Bash != nil {
			bashCommands := toolsConfig.Bash.AllowedCommands
			if len(bashCommands) == 0 {
				// bash is nil (no commands specified) - only add defaults if git commands weren't processed
				if safeOutputs == nil || !needsGitCommands(safeOutputs) {
					toolsConfig.Bash.AllowedCommands = make([]string, len(constants.DefaultBashTools))
					copy(toolsConfig.Bash.AllowedCommands, constants.DefaultBashTools)
				}
			} else if len(bashCommands) > 0 {
				// bash has commands - merge default commands with custom commands to avoid duplicates
				existingCommands := make(map[string]bool)
				for _, cmd := range bashCommands {
					existingCommands[cmd] = true
				}

				// Start with default commands
				var mergedCommands []string
				for _, cmd := range constants.DefaultBashTools {
					if !existingCommands[cmd] {
						mergedCommands = append(mergedCommands, cmd)
					}
				}

				// Add the custom commands
				mergedCommands = append(mergedCommands, bashCommands...)
				toolsConfig.Bash.AllowedCommands = mergedCommands
			}
			// Note: bash with empty array (bash: []) means "no bash tools allowed" and is left as-is
		}
	}

	return toolsConfig
}

// needsGitCommands checks if safe outputs configuration requires Git commands
func needsGitCommands(safeOutputs *SafeOutputsConfig) bool {
	if safeOutputs == nil {
		return false
	}
	return safeOutputs.CreatePullRequests != nil || safeOutputs.PushToPullRequestBranch != nil
}
