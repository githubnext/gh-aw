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
	"github.com/goccy/go-yaml"
)

var toolsLog = logger.New("workflow:tools")

// applyDefaults applies default values for missing workflow sections
func (c *Compiler) applyDefaults(data *WorkflowData, markdownPath string) error {
	toolsLog.Printf("Applying defaults to workflow: name=%s, path=%s", data.Name, markdownPath)

	// Populate cached values after all mutations to Permissions and Concurrency have been applied.
	// Using defer ensures the cache is always set on every return path, including early returns.
	// applyDefaults is the final stage that mutates data.Permissions (setting defaults and
	// injecting feature-flag permissions), so the values computed here represent the stable,
	// final state that validateWorkflowData will use. These caches eliminate repeated
	// YAML parsing and regex extraction in the hot validateWorkflowData loop.
	defer func() {
		data.CachedPermissions = NewPermissionsParser(data.Permissions).ToPermissions()
		data.ConcurrencyGroupExpr = extractConcurrencyGroupFromYAML(data.Concurrency)
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

			// Get the filtered command events based on CommandEvents field
			filteredEvents := FilterCommentEvents(data.CommandEvents)

			// Merge events for YAML generation (combines pull_request_comment and issue_comment into issue_comment)
			yamlEvents := MergeEventsForYAML(filteredEvents)

			// Build command events map from merged events
			commandEventsMap := make(map[string]any)
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

			// If label_command is also configured alongside slash_command, merge label events
			// into the existing command events map to avoid duplicate YAML keys.
			if len(data.LabelCommand) > 0 {
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
				// If conversion fails, build a basic YAML string manually
				var builder strings.Builder
				builder.WriteString(`"on":`)
				for _, event := range filteredEvents {
					builder.WriteString("\n  ")
					builder.WriteString(event.EventName)
					builder.WriteString(":\n    types: [")
					for i, t := range event.Types {
						if i > 0 {
							builder.WriteString(", ")
						}
						builder.WriteString(t)
					}
					builder.WriteString("]")
				}
				data.On = builder.String()
			}

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
		} else if isLabelCommandTrigger {
			toolsLog.Print("Workflow is label-command trigger, configuring label events")

			// Build the label-command events map
			// Generate events: issues, pull_request, discussion with types: [labeled]
			filteredEvents := FilterLabelCommandEvents(data.LabelCommandEvents)
			labelEventsMap := make(map[string]any)
			for _, eventName := range filteredEvents {
				labelEventsMap[eventName] = map[string]any{
					"types": []any{"labeled"},
				}
			}

			// Add workflow_dispatch with item_number input for manual testing.
			// Not required so the workflow can be triggered without providing a value;
			// the activation job falls back to the event payload when item_number is omitted.
			labelEventsMap["workflow_dispatch"] = map[string]any{
				"inputs": map[string]any{
					"item_number": map[string]any{
						"description": "The number of the issue, pull request, or discussion",
						"required":    false,
						"default":     "",
						"type":        "string",
					},
				},
			}
			// Signal that this workflow has a dispatch item_number input so that
			// applyWorkflowDispatchFallbacks and concurrency key building add the
			// necessary inputs.item_number fallbacks for manual workflow_dispatch runs.
			data.HasDispatchItemNumber = true

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
							merged := make([]any, 0, len(existingTypes)+len(userTypes))
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
		data.TimeoutMinutes = fmt.Sprintf("timeout-minutes: %d", int(constants.DefaultAgenticWorkflowTimeout/time.Minute))
	}

	if data.RunsOn == "" {
		data.RunsOn = "runs-on: ubuntu-latest"
	}
	// Apply default tools
	data.Tools = c.applyDefaultTools(data.Tools, data.SafeOutputs, data.SandboxConfig, data.NetworkPermissions)
	// Update ParsedTools to reflect changes made by applyDefaultTools
	data.ParsedTools = NewTools(data.Tools)

	// Check if permissions is explicitly empty ({}) - this means user wants no permissions
	// In this case, we should NOT apply default read-all.
	// Exception: if copilot-requests feature is enabled, we still need to fall through
	// so the injection block below can add copilot-requests: write.
	if data.Permissions == "permissions: {}" && !isFeatureEnabled(constants.CopilotRequestsFeatureFlag, data) {
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

	// When the copilot-requests feature is enabled, inject copilot-requests: write permission.
	// This is required so that the GitHub Actions token has the necessary scope
	// to authenticate with the Copilot API.
	if isFeatureEnabled(constants.CopilotRequestsFeatureFlag, data) {
		perms := NewPermissionsParser(data.Permissions).ToPermissions()
		perms.Set(PermissionCopilotRequests, PermissionWrite)
		yaml := perms.RenderToYAML()
		// Adjust from job-level indentation (6 spaces) to workflow-level (2 spaces)
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
