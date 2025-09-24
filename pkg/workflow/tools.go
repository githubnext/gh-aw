package workflow

import (
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
			// Generate command-specific GitHub Actions events (updated to include reopened and pull_request)
			commandEvents := `on:
  issues:
    types: [opened, edited, reopened]
  issue_comment:
    types: [created, edited]
  pull_request:
    types: [opened, edited, reopened]
  pull_request_review_comment:
    types: [created, edited]`

			// Check if there are other events to merge
			if len(data.CommandOtherEvents) > 0 {
				// Merge command events with other events
				commandEventsMap := map[string]any{
					"issues": map[string]any{
						"types": []string{"opened", "edited", "reopened"},
					},
					"issue_comment": map[string]any{
						"types": []string{"created", "edited"},
					},
					"pull_request": map[string]any{
						"types": []string{"opened", "edited", "reopened"},
					},
					"pull_request_review_comment": map[string]any{
						"types": []string{"created", "edited"},
					},
				}

				// Merge other events into command events
				for key, value := range data.CommandOtherEvents {
					commandEventsMap[key] = value
				}

				// Convert merged events to YAML
				mergedEventsYAML, err := yaml.Marshal(map[string]any{"on": commandEventsMap})
				if err == nil {
					data.On = strings.TrimSuffix(string(mergedEventsYAML), "\n")
				} else {
					// If conversion fails, just use command events
					data.On = commandEvents
				}
			} else {
				data.On = commandEvents
			}

			// Add conditional logic to check for command in issue content
			// Use event-aware condition that only applies command checks to comment-related events
			hasOtherEvents := len(data.CommandOtherEvents) > 0
			commandConditionTree := buildEventAwareCommandCondition(data.Command, hasOtherEvents)

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
		data.TimeoutMinutes = `timeout_minutes: 5`
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
	return c.mergeTools(result, includedTools)
}
