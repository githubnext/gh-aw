package workflow

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/workflow/compilerenv"
	"github.com/goccy/go-yaml"
)

var toolsLog = logger.New("workflow:tools")

// applyDefaults applies default values for missing workflow sections
func (c *Compiler) applyDefaults(data *WorkflowData, markdownPath string) error {
	toolsLog.Printf("Applying defaults to workflow: name=%s, path=%s", data.Name, markdownPath)

	// Populate cached values after all mutations to Permissions and Concurrency have been applied.
	// Using defer ensures the cache is always set on every return path, including early returns.
	// applyDefaults is the final stage that mutates data.Permissions (setting defaults), so
	// the values computed here represent the stable,
	// final state that validateWorkflowData will use. These caches eliminate repeated
	// YAML parsing, regex extraction, and expression parsing in the hot validateWorkflowData loop.
	defer func() {
		data.CachedPermissions = NewPermissionsParser(data.Permissions).ToPermissions()
		data.CachedPermissionScopeNamesErr = ValidatePermissionScopeNames(data.Permissions)
		data.CachedPermissionScopeNamesSet = true
		data.ConcurrencyGroupExpr = extractConcurrencyGroupFromYAML(data.Concurrency)
		// Pre-validate and cache the concurrency group expression so validateWorkflowData
		// can short-circuit without re-running the expensive ExpressionParser on every call.
		// CachedConcurrencyGroupExprSet is always true after applyDefaults regardless of whether
		// a group expression exists, so callers can distinguish "already computed" from "not yet computed".
		if data.ConcurrencyGroupExpr != "" {
			data.CachedConcurrencyGroupExprErr = validateConcurrencyGroupExpression(data.ConcurrencyGroupExpr)
		}
		data.CachedConcurrencyGroupExprSet = true
		// Cache the expanded + parsed toolsets for the GitHub tool so both
		// ValidatePermissions and validateToolConfiguration reuse one result.
		// Use GetToolsets() to stay aligned with the runtime normalization done by GitHubToolConfig.
		if data.ParsedTools != nil && data.ParsedTools.GitHub != nil {
			data.CachedParsedToolsets = ParseGitHubToolsets(data.ParsedTools.GitHub.GetToolsets())
		}
	}()

	// Check if this is a command trigger workflow (by checking if user specified "on.command")
	isCommandTrigger := false
	isLabelCommandTrigger := false
	if data.On == "" {
		// parseOnSection may have already detected the command trigger and populated data.Command
		// (this covers slash_command map format, slash_command shorthand "on: /name", and deprecated "command:")
		if len(data.Command) > 0 {
			isCommandTrigger = true
		} else if len(data.LabelCommand) > 0 {
			isLabelCommandTrigger = true
		} else {
			// Check the original frontmatter for command trigger
			content, err := os.ReadFile(markdownPath)
			if err == nil {
				result, err := parser.ExtractFrontmatterFromContent(string(content))
				if err == nil {
					if onValue, exists := result.Frontmatter["on"]; exists {
						// Check for slash_command or command (deprecated)
						if onMap, ok := onValue.(map[string]any); ok {
							if _, hasSlashCommand := onMap["slash_command"]; hasSlashCommand {
								isCommandTrigger = true
							} else if _, hasCommand := onMap["command"]; hasCommand {
								isCommandTrigger = true
							} else if _, hasLabelCommand := onMap["label_command"]; hasLabelCommand {
								isLabelCommandTrigger = true
							}
						}
					}
				}
			}
		}
	}

	if data.On == "" {
		if isCommandTrigger {
			toolsLog.Print("Workflow is command trigger, configuring command events")

			commandEventsMap := make(map[string]any)

			// In centralized slash-command mode, compile slash workflows as
			// workflow_dispatch-centric targets and preserve only non-slash events.
			var filteredEvents []CommentEventMapping
			if data.CommandCentralized {
				if len(data.CommandOtherEvents) > 0 {
					maps.Copy(commandEventsMap, data.CommandOtherEvents)
				}
				if _, hasWorkflowDispatch := commandEventsMap["workflow_dispatch"]; !hasWorkflowDispatch {
					commandEventsMap["workflow_dispatch"] = nil
				}
			} else {
				// Get the filtered command events based on CommandEvents field
				filteredEvents = FilterCommentEvents(data.CommandEvents)

				// Merge events for YAML generation (combines pull_request_comment and issue_comment into issue_comment)
				yamlEvents := MergeEventsForYAML(filteredEvents)

				// Build command events map from merged events
				for _, event := range yamlEvents {
					commandEventsMap[event.EventName] = map[string]any{
						"types": event.Types,
					}
				}

				// Check if there are other events to merge
				if len(data.CommandOtherEvents) > 0 {
					// Merge other events into command events
					maps.Copy(commandEventsMap, data.CommandOtherEvents)
				}
			}

			// If label_command is also configured alongside non-centralized slash_command, merge
			// label events into the existing command events map to avoid duplicate YAML keys.
			if len(data.LabelCommand) > 0 && !data.CommandCentralized {
				labelEventNames := FilterLabelCommandEvents(data.LabelCommandEvents)
				for _, eventName := range labelEventNames {
					if existingAny, ok := commandEventsMap[eventName]; ok {
						if existingMap, ok := existingAny.(map[string]any); ok {
							switch t := existingMap["types"].(type) {
							case []string:
								newTypes := make([]any, len(t)+1)
								for i, s := range t {
									newTypes[i] = s
								}
								newTypes[len(t)] = "labeled"
								existingMap["types"] = newTypes
							case []any:
								existingMap["types"] = append(t, "labeled")
							}
						}
					} else {
						commandEventsMap[eventName] = map[string]any{
							"types": []any{"labeled"},
						}
					}
				}
			}

			// Convert merged events to YAML
			mergedEventsYAML, err := yaml.Marshal(map[string]any{"on": commandEventsMap})
			if err == nil {
				yamlStr := strings.TrimSuffix(string(mergedEventsYAML), "\n")
				// Post-process YAML to ensure cron expressions are quoted
				yamlStr = parser.QuoteCronExpressions(yamlStr)
				// Apply comment processing to filter fields (draft, forks, names)
				// Pass empty frontmatter since this is for command triggers
				yamlStr = c.commentOutProcessedFieldsInOnSection(yamlStr, map[string]any{})
				// Keep "on" quoted as it's a YAML boolean keyword
				data.On = yamlStr
			} else {
				return fmt.Errorf("failed to marshal command events: %w", err)
			}

			// Add conditional logic for command workflows unless centralized mode is enabled.
			if !data.CommandCentralized {
				// Add conditional logic to check for command in issue content
				// Use event-aware condition that only applies command checks to comment-related events
				// Pass the filtered events to buildEventAwareCommandCondition
				hasOtherEvents := len(data.CommandOtherEvents) > 0
				commandConditionTree, err := buildEventAwareCommandCondition(data.Command, data.CommandEvents, hasOtherEvents)
				if err != nil {
					return fmt.Errorf("failed to build command condition: %w", err)
				}

				if data.If == "" {
					if len(data.LabelCommand) > 0 {
						// Combine: (slash_command condition) OR (label_command condition)
						// This allows the workflow to activate via either mechanism.
						labelConditionTree, err := buildLabelCommandCondition(data.LabelCommand, data.LabelCommandEvents, false)
						if err != nil {
							return fmt.Errorf("failed to build combined label-command condition: %w", err)
						}
						combined := &OrNode{Left: commandConditionTree, Right: labelConditionTree}
						data.If = RenderCondition(combined)
					} else {
						data.If = RenderCondition(commandConditionTree)
					}
				}
			} else if data.If == "" && len(data.LabelCommand) > 0 {
				// Centralized command mode compiles slash-command workflows as workflow_dispatch
				// targets. Label checks for dispatches must be derived from aw_context metadata.
				labelConditionTree, err := buildDispatchLabelCommandCondition(data.LabelCommand, data.LabelCommandEvents)
				if err != nil {
					return fmt.Errorf("failed to build label-command condition: %w", err)
				} else {
					data.If = RenderCondition(labelConditionTree)
				}
			}
		} else if isLabelCommandTrigger {
			toolsLog.Print("Workflow is label-command trigger, configuring label events")

			// Build the label-command events map
			labelEventsMap := make(map[string]any)
			if data.LabelCommandDecentralized {
				if len(data.LabelCommandOtherEvents) > 0 {
					maps.Copy(labelEventsMap, data.LabelCommandOtherEvents)
				}
				if ensureWorkflowDispatchItemNumberInput(labelEventsMap) {
					// Keep workflow_dispatch + item_number in decentralized mode so manual runs
					// retain the same fallback/concurrency behavior as inline label_command mode.
					data.HasDispatchItemNumber = true
				}
			} else {
				// Generate events: issues, pull_request, discussion with types: [labeled]
				filteredEvents := FilterLabelCommandEvents(data.LabelCommandEvents)
				for _, eventName := range filteredEvents {
					labelEventsMap[eventName] = map[string]any{
						"types": []any{"labeled"},
					}
				}

				if ensureWorkflowDispatchItemNumberInput(labelEventsMap) {
					// Signal that this workflow has a dispatch item_number input so that
					// applyWorkflowDispatchFallbacks and concurrency key building add the
					// necessary inputs.item_number fallbacks for manual workflow_dispatch runs.
					data.HasDispatchItemNumber = true
				}

				// Merge other events (if any) — this handles the no-clash requirement:
				// if the user also has e.g. "issues: {types: [labeled], names: [bug]}" as a
				// regular label trigger alongside label_command, merge the "types" arrays
				// rather than generating a duplicate "issues:" block or silently dropping config.
				if len(data.LabelCommandOtherEvents) > 0 {
					for eventKey, eventVal := range data.LabelCommandOtherEvents {
						if existing, exists := labelEventsMap[eventKey]; exists {
							// Merge types arrays from user config into the label_command-generated entry.
							existingMap, _ := existing.(map[string]any)
							userMap, _ := eventVal.(map[string]any)
							if existingMap != nil && userMap != nil {
								existingTypes, _ := existingMap["types"].([]any)
								userTypes, _ := userMap["types"].([]any)
								merged := make([]any, 0, safeAllocationCapacity(len(existingTypes), len(userTypes)))
								merged = append(merged, existingTypes...)
								merged = append(merged, userTypes...)
								existingMap["types"] = merged
								// Other fields (names, branches, etc.) from the user config are preserved.
								for k, v := range userMap {
									if k != "types" {
										existingMap[k] = v
									}
								}
							}
						} else {
							labelEventsMap[eventKey] = eventVal
						}
					}
				}
			}

			// Convert merged events to YAML
			mergedEventsYAML, err := yaml.Marshal(map[string]any{"on": labelEventsMap})
			if err != nil {
				return fmt.Errorf("failed to marshal label-command events: %w", err)
			}
			yamlStr := strings.TrimSuffix(string(mergedEventsYAML), "\n")
			yamlStr = parser.QuoteCronExpressions(yamlStr)
			// Pass frontmatter so label names in "names:" fields get commented out
			yamlStr = c.commentOutProcessedFieldsInOnSection(yamlStr, map[string]any{})
			data.On = yamlStr

			// Build the label-command condition
			hasOtherEvents := len(data.LabelCommandOtherEvents) > 0
			labelConditionTree, err := buildLabelCommandCondition(data.LabelCommand, data.LabelCommandEvents, hasOtherEvents)
			if err != nil {
				return fmt.Errorf("failed to build label-command condition: %w", err)
			}

			if data.If == "" {
				if data.LabelCommandDecentralized {
					labelConditionTree, err = buildDispatchLabelCommandCondition(data.LabelCommand, data.LabelCommandEvents)
					if err != nil {
						return fmt.Errorf("failed to build decentralized label-command condition: %w", err)
					}
				}
				data.If = RenderCondition(labelConditionTree)
			}
		} else {
			data.On = `on:
  # Start either every 10 minutes, or when some kind of human event occurs.
  # Because of the implicit "concurrency" section, only one instance of this
  # workflow will run at a time.
  schedule:
    - cron: "0/10 * * * *"
  issues:
    types: [opened, edited, closed]
  issue_comment:
    types: [created, edited]
  pull_request:
    types: [opened, edited, closed]
  push:
    branches:
      - main
  workflow_dispatch:`
		}
	}

	// Check if this workflow has an issue trigger and we're in trial mode
	// If so, inject workflow_dispatch with issue_number input
	if c.trialMode && c.hasIssueTrigger(data.On) {
		data.On = c.injectWorkflowDispatchForIssue(data.On)
	}

	// Generate concurrency configuration using the dedicated concurrency module
	data.Concurrency = GenerateConcurrencyConfig(data, isCommandTrigger || isLabelCommandTrigger)

	if data.RunName == "" {
		data.RunName = fmt.Sprintf(`run-name: "%s"`, data.Name)
	}

	if data.TimeoutMinutes == "" {
		defaultTimeoutMinutes := compilerenv.ResolveDefaultTimeoutMinutes(int(constants.DefaultAgenticWorkflowTimeout / time.Minute))
		data.TimeoutMinutes = fmt.Sprintf("timeout-minutes: %d", defaultTimeoutMinutes)
	}

	if data.RunsOn == "" {
		data.RunsOn = "runs-on: ubuntu-latest"
	}
	// Apply default tools
	data.Tools = c.applyDefaultTools(data.Tools, data.SafeOutputs, data.SandboxConfig, data.NetworkPermissions)
	// Update ParsedTools to reflect changes made by applyDefaultTools
	data.ParsedTools = NewTools(data.Tools)

	// Check if permissions is explicitly empty ({}) - this means user wants no permissions
	// and we should NOT apply defaults.
	if data.Permissions == "permissions: {}" {
		// Explicitly empty permissions - preserve the empty state
		// The agent job in dev mode will add contents: read if needed for local actions
		return nil
	}

	if data.Permissions == "" {
		// ============================================================================
		// PERMISSIONS DEFAULTS
		// ============================================================================
		// When no permissions are specified, set default to contents: read.
		// This provides minimal access needed for most workflows while following
		// the principle of least privilege.
		// ============================================================================
		perms := NewPermissionsContentsRead()
		yaml := perms.RenderToYAML()
		// RenderToYAML uses job-friendly indentation (6 spaces). WorkflowData.Permissions
		// is stored in workflow-level indentation (2 spaces) and later re-indented for jobs.
		lines := strings.Split(yaml, "\n")
		for i := 1; i < len(lines); i++ {
			if strings.HasPrefix(lines[i], "      ") {
				lines[i] = "  " + lines[i][6:]
			}
		}
		data.Permissions = strings.Join(lines, "\n")
	}

	return nil
}

func ensureWorkflowDispatchItemNumberInput(eventsMap map[string]any) bool {
	dispatchAny, hasDispatch := eventsMap["workflow_dispatch"]
	if !hasDispatch || dispatchAny == nil {
		eventsMap["workflow_dispatch"] = map[string]any{
			"inputs": map[string]any{
				"item_number": map[string]any{
					"description": "The number of the issue, pull request, or discussion",
					"required":    false,
					"default":     "",
					"type":        "string",
				},
			},
		}
		return true
	}

	dispatchMap, ok := dispatchAny.(map[string]any)
	if !ok {
		toolsLog.Print("Skipping workflow_dispatch item_number injection: workflow_dispatch is not a map")
		return false
	}

	inputsAny, hasInputs := dispatchMap["inputs"]
	if !hasInputs || inputsAny == nil {
		dispatchMap["inputs"] = map[string]any{}
		inputsAny = dispatchMap["inputs"]
	}
	inputsMap, ok := inputsAny.(map[string]any)
	if !ok {
		toolsLog.Print("Skipping workflow_dispatch item_number injection: workflow_dispatch.inputs is not a map")
		return false
	}

	if _, hasItemNumber := inputsMap["item_number"]; !hasItemNumber {
		inputsMap["item_number"] = map[string]any{
			"description": "The number of the issue, pull request, or discussion",
			"required":    false,
			"default":     "",
			"type":        "string",
		}
	}
	return true
}

// mergeToolsAndMCPServers merges tools, mcp-servers, and included tools
func (c *Compiler) mergeToolsAndMCPServers(topTools, mcpServers map[string]any, includedTools string) (map[string]any, error) {
	toolsLog.Printf("Merging tools and MCP servers: topTools=%d, mcpServers=%d", len(topTools), len(mcpServers))

	// Start with top-level tools
	result := topTools
	if result == nil {
		result = make(map[string]any)
	}

	// Add MCP servers to the tools collection
	maps.Copy(result, mcpServers)

	// Merge included tools
	return c.MergeTools(result, includedTools)
}

// mergeRuntimes merges runtime configurations from frontmatter and imports
func mergeRuntimes(topRuntimes map[string]any, importedRuntimesJSON string) (map[string]any, error) {
	toolsLog.Printf("Merging runtimes: topRuntimes=%d", len(topRuntimes))
	result := make(map[string]any)

	// Start with top-level runtimes
	maps.Copy(result, topRuntimes)

	// Merge imported runtimes (newline-separated JSON objects)
	if importedRuntimesJSON != "" {
		lines := strings.SplitSeq(strings.TrimSpace(importedRuntimesJSON), "\n")
		for line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || line == "{}" {
				continue
			}

			var importedRuntimes map[string]any
			if err := json.Unmarshal([]byte(line), &importedRuntimes); err != nil {
				return nil, fmt.Errorf("failed to parse imported runtimes JSON: %w", err)
			}

			// Merge imported runtimes - later imports override earlier ones
			maps.Copy(result, importedRuntimes)
		}
	}

	toolsLog.Printf("Merged %d total runtimes", len(result))
	return result, nil
}

// hasIssueTrigger checks if the workflow has an issue trigger in its 'on' section
func (c *Compiler) hasIssueTrigger(onSection string) bool {
	hasIssue := strings.Contains(onSection, "issues:") ||
		strings.Contains(onSection, "issue:") ||
		strings.Contains(onSection, "issue_comment:")
	toolsLog.Printf("Checking for issue trigger: has_issue=%t", hasIssue)
	return hasIssue
}

// injectWorkflowDispatchForIssue adds workflow_dispatch trigger with issue_number input
func (c *Compiler) injectWorkflowDispatchForIssue(onSection string) string {
	toolsLog.Print("Injecting workflow_dispatch trigger for issue workflows")
	// Parse the existing on section to understand its structure
	var onData map[string]any
	if err := yaml.Unmarshal([]byte(onSection), &onData); err != nil {
		// If parsing fails, append workflow_dispatch manually
		return onSection + "\n  workflow_dispatch:\n    inputs:\n      issue_number:\n        description: 'Issue number for trial mode'\n        required: true\n        type: string"
	}

	// Get the 'on' section
	if onMap, exists := onData["on"]; exists {
		if triggers, ok := onMap.(map[string]any); ok {
			// Add workflow_dispatch with issue_number input
			triggers["workflow_dispatch"] = map[string]any{
				"inputs": map[string]any{
					"issue_number": map[string]any{
						"description": "Issue number for trial mode",
						"required":    true,
						"type":        "string",
					},
				},
			}

			// Convert back to YAML
			updatedOnData := map[string]any{"on": triggers}
			if yamlBytes, err := yaml.Marshal(updatedOnData); err == nil {
				yamlStr := string(yamlBytes)
				// Keep "on" quoted as it's a YAML boolean keyword
				return strings.TrimSuffix(yamlStr, "\n")
			}
		}
	}

	// Fallback: append workflow_dispatch manually
	return onSection + "\n  workflow_dispatch:\n    inputs:\n      issue_number:\n        description: 'Issue number for trial mode'\n        required: true\n        type: string"
}

// replaceIssueNumberReferences replaces github.event.issue.number with inputs.issue_number in YAML content
func (c *Compiler) replaceIssueNumberReferences(yamlContent string) string {
	// Replace all occurrences of github.event.issue.number with inputs.issue_number
	return strings.ReplaceAll(yamlContent, "github.event.issue.number", "inputs.issue_number")
}

// applyDefaultTools adds default read-only GitHub MCP tools, creating github tool if not present
func (c *Compiler) applyDefaultTools(tools map[string]any, safeOutputs *SafeOutputsConfig, sandboxConfig *SandboxConfig, networkPermissions *NetworkPermissions) map[string]any {
	toolsLog.Printf("Applying default tools: existingToolCount=%d", len(tools))
	// Always apply default GitHub tools (create github section if it doesn't exist)

	if tools == nil {
		tools = make(map[string]any)
	}

	// Get existing github tool configuration
	githubTool := tools["github"]

	// Check if github is explicitly disabled (github: false)
	if githubTool == false {
		// Remove the github tool entirely when set to false
		delete(tools, "github")
	} else {
		// Process github tool configuration
		var githubConfig map[string]any

		if toolConfig, ok := githubTool.(map[string]any); ok {
			githubConfig = make(map[string]any)
			maps.Copy(githubConfig, toolConfig)
		} else {
			githubConfig = make(map[string]any)
		}

		// Parse the existing GitHub tool configuration for type safety
		parsedConfig := parseGitHubTool(githubTool)

		// Create a set of existing tools for efficient lookup
		existingToolsSet := make(map[string]bool)
		if parsedConfig != nil {
			for _, tool := range parsedConfig.Allowed {
				existingToolsSet[string(tool)] = true
			}
		}

		// Only set allowed tools if explicitly configured
		// Don't add default tools - let the MCP server use all available tools
		if len(existingToolsSet) > 0 {
			// Convert back to []any for the map
			existingAllowed := make([]any, 0, len(parsedConfig.Allowed))
			for _, tool := range parsedConfig.Allowed {
				existingAllowed = append(existingAllowed, string(tool))
			}
			githubConfig["allowed"] = existingAllowed
		}
		tools["github"] = githubConfig
	}

	// Enable edit and bash tools by default when sandbox is enabled
	// The sandbox is enabled when:
	// 1. Explicitly configured via sandbox.agent (awf)
	// 2. Auto-enabled by firewall default enablement (when network restrictions are present)
	if isSandboxEnabled(sandboxConfig, networkPermissions) {
		toolsLog.Print("Sandbox enabled, applying default edit and bash tools")

		// Add edit tool if not present
		if _, exists := tools["edit"]; !exists {
			tools["edit"] = true
			toolsLog.Print("Added edit tool (sandbox enabled)")
		}

		// Add bash tool with wildcard if not present
		if _, exists := tools["bash"]; !exists {
			tools["bash"] = []any{"*"}
			toolsLog.Print("Added bash tool with wildcard (sandbox enabled)")
		}
	}

	// Add Git commands and file editing tools when safe-outputs includes create-pull-request or push-to-pull-request-branch
	if safeOutputs != nil && needsGitCommands(safeOutputs) {

		// Add edit tool with null value
		if _, exists := tools["edit"]; !exists {
			tools["edit"] = nil
		}
		gitCommands := []any{
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
		if _, exists := tools["bash"]; !exists {
			// bash tool doesn't exist, add it with Git commands
			tools["bash"] = gitCommands
		} else {
			// bash tool exists, merge Git commands with existing commands
			existingBash := tools["bash"]
			if existingCommands, ok := existingBash.([]any); ok {
				// Convert existing commands to strings for comparison
				existingSet := make(map[string]bool)
				for _, cmd := range existingCommands {
					if cmdStr, ok := cmd.(string); ok {
						existingSet[cmdStr] = true
						// If we see :* or *, all bash commands are already allowed
						if cmdStr == ":*" || cmdStr == "*" {
							// Don't add specific Git commands since all are already allowed
							goto bashComplete
						}
					}
				}

				// Add Git commands that aren't already present
				newCommands := append([]any(nil), existingCommands...)
				for _, gitCmd := range gitCommands {
					if gitCmdStr, ok := gitCmd.(string); ok {
						if !existingSet[gitCmdStr] {
							newCommands = append(newCommands, gitCmd)
						}
					}
				}
				tools["bash"] = newCommands
			} else if existingBash == false {
				// bash: false was set, but git commands are required for PR operations
				// Override with git commands only (minimum needed for PR functionality)
				toolsLog.Print("Overriding bash: false with git commands (required for PR operations)")
				tools["bash"] = gitCommands
			} else if existingBash == nil {
				_ = existingBash // Keep the nil value as-is
			}
		}
	bashComplete:
	}

	// Add default bash commands when bash is enabled but no specific commands are provided
	// This runs after git commands logic, so it only applies when git commands weren't added
	// Behavior:
	//   - bash: true → All commands allowed (converted to ["*"])
	//   - bash: false → Tool disabled (removed from tools), unless git commands were needed for PR operations
	//   - bash: nil → Add default commands
	//   - bash: [] → No commands (empty array means no tools allowed)
	//   - bash: ["cmd1", "cmd2"] → Add default commands + specific commands
	if bashTool, exists := tools["bash"]; exists {
		// Check if bash was left as nil or true after git processing
		if bashTool == nil {
			// bash is nil - only add defaults if this wasn't processed by git commands
			// If git commands were needed, bash would have been set to git commands or left as nil intentionally
			if safeOutputs == nil || !needsGitCommands(safeOutputs) {
				defaultCommands := make([]any, len(constants.DefaultBashTools))
				for i, cmd := range constants.DefaultBashTools {
					defaultCommands[i] = cmd
				}
				tools["bash"] = defaultCommands
			}
		} else if bashTool == true {
			// bash is true - convert to wildcard (allow all commands)
			tools["bash"] = []any{"*"}
		} else if bashTool == false {
			// bash is false - disable the tool by removing it
			delete(tools, "bash")
		} else if bashArray, ok := bashTool.([]any); ok {
			// bash is an array - merge default commands with custom commands
			if len(bashArray) > 0 {
				// Create a set to track existing commands to avoid duplicates
				existingCommands := make(map[string]bool)
				for _, cmd := range bashArray {
					if cmdStr, ok := cmd.(string); ok {
						existingCommands[cmdStr] = true
					}
				}

				// Start with default commands (append handles capacity automatically)
				var mergedCommands []any
				for _, cmd := range constants.DefaultBashTools {
					if !existingCommands[cmd] {
						mergedCommands = append(mergedCommands, cmd)
					}
				}

				// Add the custom commands
				mergedCommands = append(mergedCommands, bashArray...)
				tools["bash"] = mergedCommands
			}
			// Note: bash with empty array (bash: []) means "no bash tools allowed" and is left as-is
		}
	}

	return tools
}
