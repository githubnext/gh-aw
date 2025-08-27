package workflow

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	// DefaultClaudeActionVersion is the default version of the Claude Code base action
	DefaultClaudeActionVersion = "v0.0.56"
)

// ClaudeEngine represents the Claude Code agentic engine
type ClaudeEngine struct {
	BaseEngine
}

func NewClaudeEngine() *ClaudeEngine {
	return &ClaudeEngine{
		BaseEngine: BaseEngine{
			id:                     "claude",
			displayName:            "Claude Code",
			description:            "Uses Claude Code with full MCP tool support and allow-listing",
			experimental:           false,
			supportsToolsWhitelist: true,
			supportsHTTPTransport:  true, // Claude supports both stdio and HTTP transport
			supportsMaxTurns:       true, // Claude supports max-turns feature
		},
	}
}

func (e *ClaudeEngine) GetInstallationSteps(engineConfig *EngineConfig) []GitHubActionStep {
	var steps []GitHubActionStep

	// Check if network permissions are configured
	if ShouldEnforceNetworkPermissions(engineConfig) {
		// Generate network hook generator and settings generator
		hookGenerator := &NetworkHookGenerator{}
		settingsGenerator := &ClaudeSettingsGenerator{}

		allowedDomains := GetAllowedDomains(engineConfig)

		// Add hook generation step
		hookStep := hookGenerator.GenerateNetworkHookWorkflowStep(allowedDomains)
		steps = append(steps, hookStep)

		// Add settings generation step
		settingsStep := settingsGenerator.GenerateSettingsWorkflowStep()
		steps = append(steps, settingsStep)
	}

	return steps
}

// GetDeclaredOutputFiles returns the output files that Claude may produce
func (e *ClaudeEngine) GetDeclaredOutputFiles() []string {
	return []string{"output.txt"}
}

func (e *ClaudeEngine) GetExecutionConfig(workflowName string, logFile string, engineConfig *EngineConfig) ExecutionConfig {
	// Determine the action version to use
	actionVersion := DefaultClaudeActionVersion // Default version
	if engineConfig != nil && engineConfig.Version != "" {
		actionVersion = engineConfig.Version
	}

	config := ExecutionConfig{
		StepName: "Execute Claude Code Action",
		Action:   fmt.Sprintf("anthropics/claude-code-base-action@%s", actionVersion),
		Inputs: map[string]string{
			"prompt_file":       "/tmp/aw-prompts/prompt.txt",
			"anthropic_api_key": "${{ secrets.ANTHROPIC_API_KEY }}",
			"mcp_config":        "/tmp/mcp-config/mcp-servers.json",
			"claude_env":        "|\n            GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}\n            GITHUB_AW_OUTPUT: ${{ env.GITHUB_AW_OUTPUT }}",
			"allowed_tools":     "", // Will be filled in during generation
			"timeout_minutes":   "", // Will be filled in during generation
			"max_turns":         "", // Will be filled in during generation
		},
		Environment: map[string]string{
			"GH_TOKEN": "${{ secrets.GITHUB_TOKEN }}",
		},
	}

	// Add model configuration if specified
	if engineConfig != nil && engineConfig.Model != "" {
		config.Inputs["model"] = engineConfig.Model
	}

	// Add settings parameter if network permissions are configured
	if ShouldEnforceNetworkPermissions(engineConfig) {
		config.Inputs["settings"] = ".claude/settings.json"
	}

	return config
}

func (e *ClaudeEngine) RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string) {
	yaml.WriteString("          cat > /tmp/mcp-config/mcp-servers.json << 'EOF'\n")
	yaml.WriteString("          {\n")
	yaml.WriteString("            \"mcpServers\": {\n")

	// Generate configuration for each MCP tool
	for i, toolName := range mcpTools {
		isLast := i == len(mcpTools)-1

		switch toolName {
		case "github":
			githubTool := tools["github"]
			e.renderGitHubClaudeMCPConfig(yaml, githubTool, isLast)
		default:
			// Handle custom MCP tools (those with MCP-compatible type)
			if toolConfig, ok := tools[toolName].(map[string]any); ok {
				if hasMcp, _ := hasMCPConfig(toolConfig); hasMcp {
					if err := e.renderClaudeMCPConfig(yaml, toolName, toolConfig, isLast); err != nil {
						fmt.Printf("Error generating custom MCP configuration for %s: %v\n", toolName, err)
					}
				}
			}
		}
	}

	yaml.WriteString("            }\n")
	yaml.WriteString("          }\n")
	yaml.WriteString("          EOF\n")
}

// renderGitHubClaudeMCPConfig generates the GitHub MCP server configuration
// Always uses Docker MCP as the default
func (e *ClaudeEngine) renderGitHubClaudeMCPConfig(yaml *strings.Builder, githubTool any, isLast bool) {
	githubDockerImageVersion := getGitHubDockerImageVersion(githubTool)

	yaml.WriteString("              \"github\": {\n")

	// Always use Docker-based GitHub MCP server (services mode has been removed)
	yaml.WriteString("                \"command\": \"docker\",\n")
	yaml.WriteString("                \"args\": [\n")
	yaml.WriteString("                  \"run\",\n")
	yaml.WriteString("                  \"-i\",\n")
	yaml.WriteString("                  \"--rm\",\n")
	yaml.WriteString("                  \"-e\",\n")
	yaml.WriteString("                  \"GITHUB_PERSONAL_ACCESS_TOKEN\",\n")
	yaml.WriteString("                  \"ghcr.io/github/github-mcp-server:" + githubDockerImageVersion + "\"\n")
	yaml.WriteString("                ],\n")
	yaml.WriteString("                \"env\": {\n")
	yaml.WriteString("                  \"GITHUB_PERSONAL_ACCESS_TOKEN\": \"${{ secrets.GITHUB_TOKEN }}\"\n")
	yaml.WriteString("                }\n")

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}
}

// renderClaudeMCPConfig generates custom MCP server configuration for a single tool in Claude workflow mcp-servers.json
func (e *ClaudeEngine) renderClaudeMCPConfig(yaml *strings.Builder, toolName string, toolConfig map[string]any, isLast bool) error {
	fmt.Fprintf(yaml, "              \"%s\": {\n", toolName)

	// Use the shared MCP config renderer with JSON format
	renderer := MCPConfigRenderer{
		IndentLevel: "                ",
		Format:      "json",
	}

	err := renderSharedMCPConfig(yaml, toolName, toolConfig, isLast, renderer)
	if err != nil {
		return err
	}

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}

	return nil
}

// ParseLogMetrics implements engine-specific log parsing for Claude
func (e *ClaudeEngine) ParseLogMetrics(logContent string, verbose bool) LogMetrics {
	var metrics LogMetrics
	var maxTokenUsage int

	// First try to parse as JSON array (Claude logs are structured as JSON arrays)
	if strings.TrimSpace(logContent) != "" {
		if resultMetrics := e.parseClaudeJSONLog(logContent, verbose); resultMetrics.TokenUsage > 0 || resultMetrics.EstimatedCost > 0 {
			metrics.TokenUsage = resultMetrics.TokenUsage
			metrics.EstimatedCost = resultMetrics.EstimatedCost
		}
	}

	// Process line by line for error counting and fallback parsing
	lines := strings.Split(logContent, "\n")

	for _, line := range lines {
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// If we haven't found cost data yet from JSON parsing, try streaming JSON
		if metrics.TokenUsage == 0 || metrics.EstimatedCost == 0 {
			jsonMetrics := ExtractJSONMetrics(line, verbose)
			if jsonMetrics.TokenUsage > 0 || jsonMetrics.EstimatedCost > 0 {
				// Check if this is a Claude result payload with aggregated costs
				if e.isClaudeResultPayload(line) {
					// For Claude result payloads, use the aggregated values directly
					if resultMetrics := e.extractClaudeResultMetrics(line); resultMetrics.TokenUsage > 0 || resultMetrics.EstimatedCost > 0 {
						metrics.TokenUsage = resultMetrics.TokenUsage
						metrics.EstimatedCost = resultMetrics.EstimatedCost
					}
				} else {
					// For streaming JSON, keep the maximum token usage found
					if jsonMetrics.TokenUsage > maxTokenUsage {
						maxTokenUsage = jsonMetrics.TokenUsage
					}
					if metrics.EstimatedCost == 0 && jsonMetrics.EstimatedCost > 0 {
						metrics.EstimatedCost += jsonMetrics.EstimatedCost
					}
				}
				continue
			}
		}

		// Count errors and warnings
		lowerLine := strings.ToLower(line)
		if strings.Contains(lowerLine, "error") {
			metrics.ErrorCount++
		}
		if strings.Contains(lowerLine, "warning") {
			metrics.WarningCount++
		}
	}

	// If no result payload was found, use the maximum from streaming JSON
	if metrics.TokenUsage == 0 {
		metrics.TokenUsage = maxTokenUsage
	}

	return metrics
}

// isClaudeResultPayload checks if the JSON line is a Claude result payload with type: "result"
func (e *ClaudeEngine) isClaudeResultPayload(line string) bool {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "{") || !strings.HasSuffix(trimmed, "}") {
		return false
	}

	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(trimmed), &jsonData); err != nil {
		return false
	}

	typeField, exists := jsonData["type"]
	if !exists {
		return false
	}

	typeStr, ok := typeField.(string)
	return ok && typeStr == "result"
}

// extractClaudeResultMetrics extracts metrics from Claude result payload
func (e *ClaudeEngine) extractClaudeResultMetrics(line string) LogMetrics {
	var metrics LogMetrics

	trimmed := strings.TrimSpace(line)
	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(trimmed), &jsonData); err != nil {
		return metrics
	}

	// Extract total_cost_usd directly
	if totalCost, exists := jsonData["total_cost_usd"]; exists {
		if cost := ConvertToFloat(totalCost); cost > 0 {
			metrics.EstimatedCost = cost
		}
	}

	// Extract usage information with all token types
	if usage, exists := jsonData["usage"]; exists {
		if usageMap, ok := usage.(map[string]interface{}); ok {
			inputTokens := ConvertToInt(usageMap["input_tokens"])
			outputTokens := ConvertToInt(usageMap["output_tokens"])
			cacheCreationTokens := ConvertToInt(usageMap["cache_creation_input_tokens"])
			cacheReadTokens := ConvertToInt(usageMap["cache_read_input_tokens"])

			totalTokens := inputTokens + outputTokens + cacheCreationTokens + cacheReadTokens
			if totalTokens > 0 {
				metrics.TokenUsage = totalTokens
			}
		}
	}

	return metrics
}

// parseClaudeJSONLog parses Claude logs as a JSON array to find the result payload
func (e *ClaudeEngine) parseClaudeJSONLog(logContent string, verbose bool) LogMetrics {
	var metrics LogMetrics

	// Try to parse the entire log as a JSON array
	var logEntries []map[string]interface{}
	if err := json.Unmarshal([]byte(logContent), &logEntries); err != nil {
		if verbose {
			fmt.Printf("Failed to parse Claude log as JSON array: %v\n", err)
		}
		return metrics
	}

	// Look for the result entry with type: "result"
	for _, entry := range logEntries {
		if entryType, exists := entry["type"]; exists {
			if typeStr, ok := entryType.(string); ok && typeStr == "result" {
				// Found the result payload, extract cost and token data
				if totalCost, exists := entry["total_cost_usd"]; exists {
					if cost := ConvertToFloat(totalCost); cost > 0 {
						metrics.EstimatedCost = cost
					}
				}

				// Extract usage information with all token types
				if usage, exists := entry["usage"]; exists {
					if usageMap, ok := usage.(map[string]interface{}); ok {
						inputTokens := ConvertToInt(usageMap["input_tokens"])
						outputTokens := ConvertToInt(usageMap["output_tokens"])
						cacheCreationTokens := ConvertToInt(usageMap["cache_creation_input_tokens"])
						cacheReadTokens := ConvertToInt(usageMap["cache_read_input_tokens"])

						totalTokens := inputTokens + outputTokens + cacheCreationTokens + cacheReadTokens
						if totalTokens > 0 {
							metrics.TokenUsage = totalTokens
						}
					}
				}

				if verbose {
					fmt.Printf("Extracted from Claude result payload: tokens=%d, cost=%.4f\n",
						metrics.TokenUsage, metrics.EstimatedCost)
				}
				break
			}
		}
	}

	return metrics
}
