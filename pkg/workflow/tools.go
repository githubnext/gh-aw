package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/goccy/go-yaml"
)

// applyDefaults applies default values for missing workflow sections
func (c *Compiler) applyDefaults(data *WorkflowData, markdownPath string) {
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

				// Clean up quoted keys - replace "on": with on: at the start of a line
				// This handles cases where YAML marshaling adds unnecessary quotes around reserved words like "on"
				yamlStr = UnquoteYAMLKey(yamlStr, "on")

				data.On = yamlStr
			} else {
				// If conversion fails, build a basic YAML string manually
				var builder strings.Builder
				builder.WriteString("on:")
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

	if data.Permissions == "" {
		// Default behavior: use read-all permissions
		data.Permissions = `permissions: read-all`
	}

	// Generate concurrency configuration using the dedicated concurrency module
	data.Concurrency = GenerateConcurrencyConfig(data, isCommandTrigger)

	if data.RunName == "" {
		data.RunName = fmt.Sprintf(`run-name: "%s"`, data.Name)
	}

	if data.TimeoutMinutes == "" {
		data.TimeoutMinutes = `timeout_minutes: 20`
	}

	if data.RunsOn == "" {
		data.RunsOn = "runs-on: ubuntu-latest"
	}
	// Apply default tools
	data.Tools = c.applyDefaultTools(data.Tools, data.SafeOutputs)
}

// extractToolsFromFrontmatter extracts tools section from frontmatter map
func extractToolsFromFrontmatter(frontmatter map[string]any) map[string]any {
	tools, exists := frontmatter["tools"]
	if !exists {
		return make(map[string]any)
	}

	if toolsMap, ok := tools.(map[string]any); ok {
		return toolsMap
	}

	return make(map[string]any)
}

// extractMCPServersFromFrontmatter extracts mcp-servers section from frontmatter
func extractMCPServersFromFrontmatter(frontmatter map[string]any) map[string]any {
	if mcpServers, exists := frontmatter["mcp-servers"]; exists {
		if mcpServersMap, ok := mcpServers.(map[string]any); ok {
			return mcpServersMap
		}
	}
	return make(map[string]any)
}

// extractRuntimesFromFrontmatter extracts runtimes section from frontmatter map
func extractRuntimesFromFrontmatter(frontmatter map[string]any) map[string]any {
	if runtimes, exists := frontmatter["runtimes"]; exists {
		if runtimesMap, ok := runtimes.(map[string]any); ok {
			return runtimesMap
		}
	}
	return make(map[string]any)
}

// mergeToolsAndMCPServers merges tools, mcp-servers, and included tools
func (c *Compiler) mergeToolsAndMCPServers(topTools, mcpServers map[string]any, includedTools string) (map[string]any, error) {
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

	return result, nil
}
