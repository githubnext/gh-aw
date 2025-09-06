package workflow

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
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

func (e *ClaudeEngine) GetInstallationSteps(workflowData *WorkflowData) []GitHubActionStep {
	var steps []GitHubActionStep

	// Check if network permissions are configured (only for Claude engine)
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.ID == "claude" && ShouldEnforceNetworkPermissions(workflowData.NetworkPermissions) {
		// Generate network hook generator and settings generator
		hookGenerator := &NetworkHookGenerator{}
		settingsGenerator := &ClaudeSettingsGenerator{}

		allowedDomains := GetAllowedDomains(workflowData.NetworkPermissions)

		// Add settings generation step
		settingsStep := settingsGenerator.GenerateSettingsWorkflowStep()
		steps = append(steps, settingsStep)

		// Add hook generation step
		hookStep := hookGenerator.GenerateNetworkHookWorkflowStep(allowedDomains)
		steps = append(steps, hookStep)
	}

	return steps
}

// GetDeclaredOutputFiles returns the output files that Claude may produce
func (e *ClaudeEngine) GetDeclaredOutputFiles() []string {
	return []string{"output.txt"}
}

// GetExecutionSteps returns the GitHub Actions steps for executing Claude
func (e *ClaudeEngine) GetExecutionSteps(workflowData *WorkflowData, logFile string) []GitHubActionStep {
	var steps []GitHubActionStep

	// Handle custom steps if they exist in engine config
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Steps) > 0 {
		for _, step := range workflowData.EngineConfig.Steps {
			stepYAML, err := e.convertStepToYAML(step)
			if err != nil {
				// Log error but continue with other steps
				continue
			}
			steps = append(steps, GitHubActionStep{stepYAML})
		}
	}

	// Determine the action version to use
	actionVersion := DefaultClaudeActionVersion // Default version
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Version != "" {
		actionVersion = workflowData.EngineConfig.Version
	}

	// Build claude_env based on hasOutput parameter and custom env vars
	hasOutput := workflowData.SafeOutputs != nil
	claudeEnv := ""
	if hasOutput {
		claudeEnv += "            GITHUB_AW_SAFE_OUTPUTS: ${{ env.GITHUB_AW_SAFE_OUTPUTS }}"
	}

	// Add custom environment variables from engine config
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Env) > 0 {
		for key, value := range workflowData.EngineConfig.Env {
			if claudeEnv != "" {
				claudeEnv += "\n"
			}
			claudeEnv += "            " + key + ": " + value
		}
	}

	inputs := map[string]string{
		"prompt_file":       "/tmp/aw-prompts/prompt.txt",
		"anthropic_api_key": "${{ secrets.ANTHROPIC_API_KEY }}",
		"mcp_config":        "/tmp/mcp-config/mcp-servers.json",
		"allowed_tools":     "", // Will be filled in during generation
		"timeout_minutes":   "", // Will be filled in during generation
	}

	// Only add max_turns if it's actually specified
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.MaxTurns != "" {
		inputs["max_turns"] = workflowData.EngineConfig.MaxTurns
	}
	if claudeEnv != "" {
		inputs["claude_env"] = "|\n" + claudeEnv
	}

	// Add model configuration if specified
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Model != "" {
		inputs["model"] = workflowData.EngineConfig.Model
	}

	// Add settings parameter if network permissions are configured
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.ID == "claude" && ShouldEnforceNetworkPermissions(workflowData.NetworkPermissions) {
		inputs["settings"] = ".claude/settings.json"
	}

	// Apply default Claude tools
	allowedTools := e.computeAllowedClaudeToolsString(workflowData.Tools, workflowData.SafeOutputs)

	var stepLines []string

	stepName := "Execute Claude Code Action"
	action := fmt.Sprintf("anthropics/claude-code-base-action@%s", actionVersion)

	stepLines = append(stepLines, fmt.Sprintf("      - name: %s", stepName))
	stepLines = append(stepLines, "        id: agentic_execution")
	stepLines = append(stepLines, fmt.Sprintf("        uses: %s", action))
	stepLines = append(stepLines, "        with:")

	// Add inputs in alphabetical order by key
	keys := make([]string, 0, len(inputs))
	for key := range inputs {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := inputs[key]
		if key == "allowed_tools" {
			if allowedTools != "" {
				// Add comment listing all allowed tools for readability
				comment := e.generateAllowedToolsComment(allowedTools, "          ")
				commentLines := strings.Split(comment, "\n")
				// Filter out empty lines to avoid breaking test logic
				for _, line := range commentLines {
					if line != "" {
						stepLines = append(stepLines, line)
					}
				}
				stepLines = append(stepLines, fmt.Sprintf("          %s: \"%s\"", key, allowedTools))
			}
		} else if key == "timeout_minutes" {
			// Always include timeout_minutes field
			if workflowData.TimeoutMinutes != "" {
				// TimeoutMinutes contains the full YAML line (e.g. "timeout_minutes: 5")
				stepLines = append(stepLines, "          "+workflowData.TimeoutMinutes)
			} else {
				stepLines = append(stepLines, "          timeout_minutes: 5") // Default timeout
			}
		} else if key == "max_turns" {
			// max_turns is only in the map when it should be included
			stepLines = append(stepLines, fmt.Sprintf("          max_turns: %s", value))
		} else if value != "" {
			if strings.HasPrefix(value, "|") {
				stepLines = append(stepLines, fmt.Sprintf("          %s: %s", key, value))
			} else {
				stepLines = append(stepLines, fmt.Sprintf("          %s: %s", key, value))
			}
		}
	}

	// Add environment section - always include environment section for GITHUB_AW_PROMPT
	stepLines = append(stepLines, "        env:")

	// Always add GITHUB_AW_PROMPT for agentic workflows
	stepLines = append(stepLines, "          GITHUB_AW_PROMPT: /tmp/aw-prompts/prompt.txt")

	if workflowData.SafeOutputs != nil {
		stepLines = append(stepLines, "          GITHUB_AW_SAFE_OUTPUTS: ${{ env.GITHUB_AW_SAFE_OUTPUTS }}")
	}

	if workflowData.EngineConfig != nil && workflowData.EngineConfig.MaxTurns != "" {
		stepLines = append(stepLines, fmt.Sprintf("          GITHUB_AW_MAX_TURNS: %s", workflowData.EngineConfig.MaxTurns))
	}

	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Env) > 0 {
		for key, value := range workflowData.EngineConfig.Env {
			stepLines = append(stepLines, fmt.Sprintf("          %s: %s", key, value))
		}
	}

	steps = append(steps, GitHubActionStep(stepLines))

	// Add the log capture step
	logCaptureLines := []string{
		"      - name: Capture Agentic Action logs",
		"        if: always()",
		"        run: |",
		"          # Copy the detailed execution file from Agentic Action if available",
		"          if [ -n \"${{ steps.agentic_execution.outputs.execution_file }}\" ] && [ -f \"${{ steps.agentic_execution.outputs.execution_file }}\" ]; then",
		"            cp ${{ steps.agentic_execution.outputs.execution_file }} " + logFile,
		"          else",
		"            echo \"No execution file output found from Agentic Action\" >> " + logFile,
		"          fi",
		"          ",
		"          # Ensure log file exists",
		"          touch " + logFile,
	}
	steps = append(steps, GitHubActionStep(logCaptureLines))

	return steps
}

// convertStepToYAML converts a step map to YAML string - temporary helper
func (e *ClaudeEngine) convertStepToYAML(stepMap map[string]any) (string, error) {
	// Simple YAML generation for steps - this mirrors the compiler logic
	var stepYAML []string

	// Add step name
	if name, hasName := stepMap["name"]; hasName {
		if nameStr, ok := name.(string); ok {
			stepYAML = append(stepYAML, fmt.Sprintf("      - name: %s", nameStr))
		}
	}

	// Add run command
	if run, hasRun := stepMap["run"]; hasRun {
		if runStr, ok := run.(string); ok {
			stepYAML = append(stepYAML, "        run: |")
			// Split command into lines and indent them properly
			runLines := strings.Split(runStr, "\n")
			for _, line := range runLines {
				stepYAML = append(stepYAML, "          "+line)
			}
		}
	}

	// Add uses action
	if uses, hasUses := stepMap["uses"]; hasUses {
		if usesStr, ok := uses.(string); ok {
			stepYAML = append(stepYAML, fmt.Sprintf("        uses: %s", usesStr))
		}
	}

	// Add with parameters
	if with, hasWith := stepMap["with"]; hasWith {
		if withMap, ok := with.(map[string]any); ok {
			stepYAML = append(stepYAML, "        with:")
			// Sort keys for stable output
			keys := make([]string, 0, len(withMap))
			for key := range withMap {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				stepYAML = append(stepYAML, fmt.Sprintf("          %s: %v", key, withMap[key]))
			}
		}
	}

	return strings.Join(stepYAML, "\n"), nil
}

// expandNeutralToolsToClaudeTools converts neutral tools to Claude-specific tools format
func (e *ClaudeEngine) expandNeutralToolsToClaudeTools(tools map[string]any) map[string]any {
	result := make(map[string]any)

	// Copy existing tools that are not neutral tools
	for key, value := range tools {
		switch key {
		case "bash", "web-fetch", "web-search", "edit":
			// These are neutral tools that need conversion - skip copying, will be converted below
			continue
		default:
			// Copy MCP servers and other non-neutral tools as-is
			result[key] = value
		}
	}

	// Create or get existing claude section
	var claudeSection map[string]any
	if existing, hasClaudeSection := result["claude"]; hasClaudeSection {
		if claudeMap, ok := existing.(map[string]any); ok {
			claudeSection = claudeMap
		} else {
			claudeSection = make(map[string]any)
		}
	} else {
		claudeSection = make(map[string]any)
	}

	// Get existing allowed tools from Claude section
	var claudeAllowed map[string]any
	if allowed, hasAllowed := claudeSection["allowed"]; hasAllowed {
		if allowedMap, ok := allowed.(map[string]any); ok {
			claudeAllowed = allowedMap
		} else {
			claudeAllowed = make(map[string]any)
		}
	} else {
		claudeAllowed = make(map[string]any)
	}

	// Convert neutral tools to Claude tools
	if bashTool, hasBash := tools["bash"]; hasBash {
		// bash -> Bash, KillBash, BashOutput
		if bashCommands, ok := bashTool.([]any); ok {
			claudeAllowed["Bash"] = bashCommands
		} else {
			claudeAllowed["Bash"] = nil // Allow all bash commands
		}
	}

	if _, hasWebFetch := tools["web-fetch"]; hasWebFetch {
		// web-fetch -> WebFetch
		claudeAllowed["WebFetch"] = nil
	}

	if _, hasWebSearch := tools["web-search"]; hasWebSearch {
		// web-search -> WebSearch
		claudeAllowed["WebSearch"] = nil
	}

	if editTool, hasEdit := tools["edit"]; hasEdit {
		// edit -> Edit, MultiEdit, NotebookEdit, Write
		claudeAllowed["Edit"] = nil
		claudeAllowed["MultiEdit"] = nil
		claudeAllowed["NotebookEdit"] = nil
		claudeAllowed["Write"] = nil

		// If edit tool has specific configuration, we could handle it here
		// For now, treating it as enabling all edit capabilities
		_ = editTool
	}

	// Update claude section
	claudeSection["allowed"] = claudeAllowed
	result["claude"] = claudeSection

	return result
}

// computeAllowedClaudeToolsString
// 1. validates that only neutral tools are provided (no claude section)
// 2. converts neutral tools to Claude-specific tools format
// 3. adds default Claude tools and git commands based on safe outputs configuration
// 4. generates the allowed tools string for Claude
func (e *ClaudeEngine) computeAllowedClaudeToolsString(tools map[string]any, safeOutputs *SafeOutputsConfig) string {
	// Initialize tools map if nil
	if tools == nil {
		tools = make(map[string]any)
	}

	// Enforce that only neutral tools are provided - fail if claude section is present
	if _, hasClaudeSection := tools["claude"]; hasClaudeSection {
		panic("computeAllowedClaudeToolsString should only receive neutral tools, not claude section tools")
	}

	// Convert neutral tools to Claude-specific tools
	tools = e.expandNeutralToolsToClaudeTools(tools)

	defaultClaudeTools := []string{
		"Task",
		"Glob",
		"Grep",
		"ExitPlanMode",
		"TodoWrite",
		"LS",
		"Read",
		"NotebookRead",
	}

	// Ensure claude section exists with the new format
	var claudeSection map[string]any
	if existing, hasClaudeSection := tools["claude"]; hasClaudeSection {
		if claudeMap, ok := existing.(map[string]any); ok {
			claudeSection = claudeMap
		} else {
			claudeSection = make(map[string]any)
		}
	} else {
		claudeSection = make(map[string]any)
	}

	// Get existing allowed tools from the new format (map structure)
	var claudeExistingAllowed map[string]any
	if allowed, hasAllowed := claudeSection["allowed"]; hasAllowed {
		if allowedMap, ok := allowed.(map[string]any); ok {
			claudeExistingAllowed = allowedMap
		} else {
			claudeExistingAllowed = make(map[string]any)
		}
	} else {
		claudeExistingAllowed = make(map[string]any)
	}

	// Add default tools that aren't already present
	for _, defaultTool := range defaultClaudeTools {
		if _, exists := claudeExistingAllowed[defaultTool]; !exists {
			claudeExistingAllowed[defaultTool] = nil // Add tool with null value
		}
	}

	// Check if Bash tools are present and add implicit KillBash and BashOutput
	if _, hasBash := claudeExistingAllowed["Bash"]; hasBash {
		// Implicitly add KillBash and BashOutput when any Bash tools are allowed
		if _, exists := claudeExistingAllowed["KillBash"]; !exists {
			claudeExistingAllowed["KillBash"] = nil
		}
		if _, exists := claudeExistingAllowed["BashOutput"]; !exists {
			claudeExistingAllowed["BashOutput"] = nil
		}
	}

	// Update the claude section with the new format
	claudeSection["allowed"] = claudeExistingAllowed
	tools["claude"] = claudeSection

	var allowedTools []string

	// Process claude-specific tools from the claude section (new format only)
	if claudeSection, hasClaudeSection := tools["claude"]; hasClaudeSection {
		if claudeConfig, ok := claudeSection.(map[string]any); ok {
			if allowed, hasAllowed := claudeConfig["allowed"]; hasAllowed {
				// In the new format, allowed is a map where keys are tool names
				if allowedMap, ok := allowed.(map[string]any); ok {
					for toolName, toolValue := range allowedMap {
						if toolName == "Bash" {
							// Handle Bash tool with specific commands
							if bashCommands, ok := toolValue.([]any); ok {
								// Check for :* wildcard first - if present, ignore all other bash commands
								for _, cmd := range bashCommands {
									if cmdStr, ok := cmd.(string); ok {
										if cmdStr == ":*" {
											// :* means allow all bash and ignore other commands
											allowedTools = append(allowedTools, "Bash")
											goto nextClaudeTool
										}
									}
								}
								// Process the allowed bash commands (no :* found)
								for _, cmd := range bashCommands {
									if cmdStr, ok := cmd.(string); ok {
										if cmdStr == "*" {
											// Wildcard means allow all bash
											allowedTools = append(allowedTools, "Bash")
											goto nextClaudeTool
										}
									}
								}
								// Add individual bash commands with Bash() prefix
								for _, cmd := range bashCommands {
									if cmdStr, ok := cmd.(string); ok {
										allowedTools = append(allowedTools, fmt.Sprintf("Bash(%s)", cmdStr))
									}
								}
							} else {
								// Bash with no specific commands or null value - allow all bash
								allowedTools = append(allowedTools, "Bash")
							}
						} else if strings.HasPrefix(toolName, strings.ToUpper(toolName[:1])) {
							// Tool name starts with uppercase letter - regular Claude tool
							allowedTools = append(allowedTools, toolName)
						}
					nextClaudeTool:
					}
				}
			}
		}
	}

	// Process top-level tools (MCP tools and claude)
	for toolName, toolValue := range tools {
		if toolName == "claude" {
			// Skip the claude section as we've already processed it
			continue
		} else {
			// Check if this is an MCP tool (has MCP-compatible type) or standard MCP tool (github)
			if mcpConfig, ok := toolValue.(map[string]any); ok {
				// Check if it's explicitly marked as MCP type
				isCustomMCP := false
				if hasMcp, _ := hasMCPConfig(mcpConfig); hasMcp {
					isCustomMCP = true
				}

				// Handle standard MCP tools (github) or tools with MCP-compatible type
				if toolName == "github" || isCustomMCP {
					if allowed, hasAllowed := mcpConfig["allowed"]; hasAllowed {
						if allowedSlice, ok := allowed.([]any); ok {
							// Check for wildcard access first
							hasWildcard := false
							for _, item := range allowedSlice {
								if str, ok := item.(string); ok && str == "*" {
									hasWildcard = true
									break
								}
							}

							if hasWildcard {
								// For wildcard access, just add the server name with mcp__ prefix
								allowedTools = append(allowedTools, fmt.Sprintf("mcp__%s", toolName))
							} else {
								// For specific tools, add each one individually
								for _, item := range allowedSlice {
									if str, ok := item.(string); ok {
										allowedTools = append(allowedTools, fmt.Sprintf("mcp__%s__%s", toolName, str))
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// Handle SafeOutputs requirement for file write access
	if safeOutputs != nil {
		// Check if a general "Write" permission is already granted
		hasGeneralWrite := slices.Contains(allowedTools, "Write")

		// If no general Write permission and SafeOutputs is configured,
		// add specific write permission for GITHUB_AW_SAFE_OUTPUTS
		if !hasGeneralWrite {
			allowedTools = append(allowedTools, "Write")
			// Ideally we would only give permission to the exact file, but that doesn't seem
			// to be working with Claude. See https://github.com/githubnext/gh-aw/issues/244#issuecomment-3240319103
			//allowedTools = append(allowedTools, "Write(${{ env.GITHUB_AW_SAFE_OUTPUTS }})")
		}
	}

	// Sort the allowed tools alphabetically for consistent output
	sort.Strings(allowedTools)

	return strings.Join(allowedTools, ",")
}

// generateAllowedToolsComment generates a multi-line comment showing each allowed tool
func (e *ClaudeEngine) generateAllowedToolsComment(allowedToolsStr string, indent string) string {
	if allowedToolsStr == "" {
		return ""
	}

	tools := strings.Split(allowedToolsStr, ",")
	if len(tools) == 0 {
		return ""
	}

	var comment strings.Builder
	comment.WriteString(indent + "# Allowed tools (sorted):\n")
	for _, tool := range tools {
		comment.WriteString(fmt.Sprintf("%s# - %s\n", indent, tool))
	}

	return comment.String()
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

	err := renderSharedMCPConfig(yaml, toolName, toolConfig, renderer)
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

// GetLogParserScript returns the JavaScript script name for parsing Claude logs
func (e *ClaudeEngine) GetLogParserScript() string {
	return "parse_claude_log"
}
