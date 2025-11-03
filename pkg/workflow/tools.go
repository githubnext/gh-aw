package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/goccy/go-yaml"
)

var toolsLog = logger.New("workflow:tools")

// applyDefaults applies default values for missing workflow sections
func (c *Compiler) applyDefaults(data *WorkflowData, markdownPath string) {
	toolsLog.Printf("Applying defaults to workflow: name=%s, path=%s", data.Name, markdownPath)

	// Check if this is a command trigger workflow (by checking if user specified "on.command")
	isCommandTrigger := false
	if data.On == "" {
		// Check the original frontmatter for command trigger
		content, err := os.ReadFile(markdownPath)
		if err == nil {
			result, err := parser.ExtractFrontmatterFromContent(string(content))
			if err == nil {
				if onValue, exists := result.Frontmatter["on"]; exists {
					// Check for new format: on.command
					if onMap, ok := onValue.(map[string]any); ok {
						if _, hasCommand := onMap["command"]; hasCommand {
							isCommandTrigger = true
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
				for key, value := range data.CommandOtherEvents {
					commandEventsMap[key] = value
				}
			}

			// Convert merged events to YAML
			mergedEventsYAML, err := yaml.Marshal(map[string]any{"on": commandEventsMap})
			if err == nil {
				yamlStr := strings.TrimSuffix(string(mergedEventsYAML), "\n")
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
			commandConditionTree := buildEventAwareCommandCondition(data.Command, data.CommandEvents, hasOtherEvents)

			if data.If == "" {
				data.If = commandConditionTree.Render()
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

	if data.Permissions == "" {
		// Default behavior: use read-all permissions
		data.Permissions = NewPermissionsReadAll().RenderToYAML()
	}

	// Generate concurrency configuration using the dedicated concurrency module
	data.Concurrency = GenerateConcurrencyConfig(data, isCommandTrigger)

	if data.RunName == "" {
		data.RunName = fmt.Sprintf(`run-name: "%s"`, data.Name)
	}

	if data.TimeoutMinutes == "" {
		data.TimeoutMinutes = fmt.Sprintf("timeout_minutes: %d", constants.DefaultAgenticWorkflowTimeoutMinutes)
	}

	if data.RunsOn == "" {
		data.RunsOn = "runs-on: ubuntu-latest"
	}
	// Apply default tools
	data.Tools = c.applyDefaultTools(data.Tools, data.SafeOutputs)
	// Update ParsedTools to reflect changes made by applyDefaultTools
	data.ParsedTools = NewTools(data.Tools)
}

// extractMapFromFrontmatter is a generic helper to extract a map[string]any from frontmatter
func extractMapFromFrontmatter(frontmatter map[string]any, key string) map[string]any {
	if value, exists := frontmatter[key]; exists {
		if valueMap, ok := value.(map[string]any); ok {
			return valueMap
		}
	}
	return make(map[string]any)
}

// extractToolsFromFrontmatter extracts tools section from frontmatter map
func extractToolsFromFrontmatter(frontmatter map[string]any) map[string]any {
	return extractMapFromFrontmatter(frontmatter, "tools")
}

// extractMCPServersFromFrontmatter extracts mcp-servers section from frontmatter
func extractMCPServersFromFrontmatter(frontmatter map[string]any) map[string]any {
	return extractMapFromFrontmatter(frontmatter, "mcp-servers")
}

// extractRuntimesFromFrontmatter extracts runtimes section from frontmatter map
func extractRuntimesFromFrontmatter(frontmatter map[string]any) map[string]any {
	return extractMapFromFrontmatter(frontmatter, "runtimes")
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
	for serverName, serverConfig := range mcpServers {
		result[serverName] = serverConfig
	}

	// Merge included tools
	return c.MergeTools(result, includedTools)
}

// mergeRuntimes merges runtime configurations from frontmatter and imports
func mergeRuntimes(topRuntimes map[string]any, importedRuntimesJSON string) (map[string]any, error) {
	toolsLog.Printf("Merging runtimes: topRuntimes=%d", len(topRuntimes))
	result := make(map[string]any)

	// Start with top-level runtimes
	for id, config := range topRuntimes {
		result[id] = config
	}

	// Merge imported runtimes (newline-separated JSON objects)
	if importedRuntimesJSON != "" {
		lines := strings.Split(strings.TrimSpace(importedRuntimesJSON), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || line == "{}" {
				continue
			}

			var importedRuntimes map[string]any
			if err := json.Unmarshal([]byte(line), &importedRuntimes); err != nil {
				return nil, fmt.Errorf("failed to parse imported runtimes JSON: %w", err)
			}

			// Merge imported runtimes - later imports override earlier ones
			for id, config := range importedRuntimes {
				result[id] = config
			}
		}
	}

	toolsLog.Printf("Merged %d total runtimes", len(result))
	return result, nil
}

// hasIssueTrigger checks if the workflow has an issue trigger in its 'on' section
func (c *Compiler) hasIssueTrigger(onSection string) bool {
	// Look for 'issues:', 'issue:', or 'issue_comment:' in the on section
	return strings.Contains(onSection, "issues:") ||
		strings.Contains(onSection, "issue:") ||
		strings.Contains(onSection, "issue_comment:")
}

// injectWorkflowDispatchForIssue adds workflow_dispatch trigger with issue_number input
func (c *Compiler) injectWorkflowDispatchForIssue(onSection string) string {
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
