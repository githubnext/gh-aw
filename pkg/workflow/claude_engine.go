package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/githubnext/gh-aw/pkg/constants"
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
			supportsToolsAllowlist: true,
			supportsHTTPTransport:  true,  // Claude supports both stdio and HTTP transport
			supportsMaxTurns:       true,  // Claude supports max-turns feature
			supportsWebFetch:       true,  // Claude has built-in WebFetch support
			supportsWebSearch:      true,  // Claude has built-in WebSearch support
			hasDefaultConcurrency:  false, // Claude does NOT have default concurrency enabled
		},
	}
}

func (e *ClaudeEngine) GetInstallationSteps(workflowData *WorkflowData) []GitHubActionStep {
	var steps []GitHubActionStep

	// Use version from engine config if provided, otherwise default to pinned version
	version := constants.DefaultClaudeCodeVersion
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Version != "" {
		version = workflowData.EngineConfig.Version
	}

	// Add npm package installation steps (includes Node.js setup)
	npmSteps := GenerateNpmInstallSteps(
		"@anthropic-ai/claude-code",
		version,
		"Install Claude Code CLI",
		"claude",
		true, // Include Node.js setup
	)
	steps = append(steps, npmSteps...)

	// Check if network permissions are configured (only for Claude engine)
	if workflowData.EngineConfig != nil && ShouldEnforceNetworkPermissions(workflowData.NetworkPermissions) {
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
	return []string{}
}

// GetVersionCommand returns the command to get Claude's version
func (e *ClaudeEngine) GetVersionCommand() string {
	return "claude --version"
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

	// Build claude CLI arguments based on configuration
	var claudeArgs []string

	// Add print flag for non-interactive mode
	claudeArgs = append(claudeArgs, "--print")

	// Add model if specified
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Model != "" {
		claudeArgs = append(claudeArgs, "--model", workflowData.EngineConfig.Model)
	}

	// Add max_turns if specified (in CLI it's max-turns)
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.MaxTurns != "" {
		claudeArgs = append(claudeArgs, "--max-turns", workflowData.EngineConfig.MaxTurns)
	}

	// Add MCP configuration only if there are MCP servers
	if HasMCPServers(workflowData) {
		claudeArgs = append(claudeArgs, "--mcp-config", "/tmp/gh-aw/mcp-config/mcp-servers.json")
	}

	// Add allowed tools configuration
	allowedTools := e.computeAllowedClaudeToolsString(workflowData.Tools, workflowData.SafeOutputs)
	if allowedTools != "" {
		claudeArgs = append(claudeArgs, "--allowed-tools", allowedTools)
	}

	// Add debug flag
	claudeArgs = append(claudeArgs, "--debug")

	// Always add verbose flag for enhanced debugging output
	claudeArgs = append(claudeArgs, "--verbose")

	// Add permission mode for non-interactive execution (bypass permissions)
	claudeArgs = append(claudeArgs, "--permission-mode", "bypassPermissions")

	// Add output format for structured output
	claudeArgs = append(claudeArgs, "--output-format", "stream-json")

	// Add network settings if configured
	if workflowData.EngineConfig != nil && ShouldEnforceNetworkPermissions(workflowData.NetworkPermissions) {
		claudeArgs = append(claudeArgs, "--settings", "/tmp/gh-aw/.claude/settings.json")
	}

	var stepLines []string

	stepName := "Execute Claude Code CLI"

	stepLines = append(stepLines, fmt.Sprintf("      - name: %s", stepName))
	stepLines = append(stepLines, "        id: agentic_execution")

	// Add allowed tools comment before the run section
	allowedToolsComment := e.generateAllowedToolsComment(e.computeAllowedClaudeToolsString(workflowData.Tools, workflowData.SafeOutputs), "        ")
	if allowedToolsComment != "" {
		// Split the comment into lines and add each line
		commentLines := strings.Split(strings.TrimSuffix(allowedToolsComment, "\n"), "\n")
		stepLines = append(stepLines, commentLines...)
	}

	// Add timeout at step level (GitHub Actions standard)
	if workflowData.TimeoutMinutes != "" {
		stepLines = append(stepLines, fmt.Sprintf("        timeout-minutes: %s", strings.TrimPrefix(workflowData.TimeoutMinutes, "timeout_minutes: ")))
	} else {
		stepLines = append(stepLines, fmt.Sprintf("        timeout-minutes: %d", constants.DefaultAgenticWorkflowTimeoutMinutes)) // Default timeout for agentic workflows
	}

	// Build the run command
	stepLines = append(stepLines, "        run: |")
	stepLines = append(stepLines, "          set -o pipefail")
	stepLines = append(stepLines, "          # Execute Claude Code CLI with prompt from file")

	// Build the command string with proper argument formatting
	// Use claude command directly (installed via npm install -g)
	commandParts := []string{"claude"}
	commandParts = append(commandParts, claudeArgs...)
	commandParts = append(commandParts, "$(cat /tmp/gh-aw/aw-prompts/prompt.txt)")

	// Join command parts with proper escaping for complex arguments
	command := ""
	for i, part := range commandParts {
		if i > 0 {
			command += " "
		}
		// For complex arguments that contain spaces or special characters, quote them
		if strings.Contains(part, " ") || strings.Contains(part, ",") {
			command += "\"" + part + "\""
		} else {
			command += part
		}
	}

	// Add the command with proper indentation and tee output (preserves exit code with pipefail)
	stepLines = append(stepLines, fmt.Sprintf("          %s 2>&1 | tee %s", command, logFile))

	// Add environment section - always include environment section for GITHUB_AW_PROMPT
	stepLines = append(stepLines, "        env:")

	// Add Anthropic API key
	stepLines = append(stepLines, "          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}")

	// Disable telemetry, error reporting, and bug command for privacy and security
	stepLines = append(stepLines, "          DISABLE_TELEMETRY: \"1\"")
	stepLines = append(stepLines, "          DISABLE_ERROR_REPORTING: \"1\"")
	stepLines = append(stepLines, "          DISABLE_BUG_COMMAND: \"1\"")

	// Always add GITHUB_AW_PROMPT for agentic workflows
	stepLines = append(stepLines, "          GITHUB_AW_PROMPT: /tmp/gh-aw/aw-prompts/prompt.txt")

	// Add GITHUB_AW_MCP_CONFIG for MCP server configuration only if there are MCP servers
	if HasMCPServers(workflowData) {
		stepLines = append(stepLines, "          GITHUB_AW_MCP_CONFIG: /tmp/gh-aw/mcp-config/mcp-servers.json")
	}

	// Set MCP_TIMEOUT to 60000ms for MCP server communication
	stepLines = append(stepLines, "          MCP_TIMEOUT: \"60000\"")

	if workflowData.SafeOutputs != nil {
		stepLines = append(stepLines, "          GITHUB_AW_SAFE_OUTPUTS: ${{ env.GITHUB_AW_SAFE_OUTPUTS }}")

		// Add staged flag if specified
		if workflowData.TrialMode || workflowData.SafeOutputs.Staged {
			stepLines = append(stepLines, "          GITHUB_AW_SAFE_OUTPUTS_STAGED: \"true\"")
		}
		if workflowData.TrialMode && workflowData.TrialTargetRepo != "" {
			stepLines = append(stepLines, fmt.Sprintf("          GITHUB_AW_TARGET_REPO_SLUG: %q", workflowData.TrialTargetRepo))
		}

		// Add branch name if upload assets is configured
		if workflowData.SafeOutputs.UploadAssets != nil {
			stepLines = append(stepLines, fmt.Sprintf("          GITHUB_AW_ASSETS_BRANCH: %q", workflowData.SafeOutputs.UploadAssets.BranchName))
			stepLines = append(stepLines, fmt.Sprintf("          GITHUB_AW_ASSETS_MAX_SIZE_KB: %d", workflowData.SafeOutputs.UploadAssets.MaxSizeKB))
			stepLines = append(stepLines, fmt.Sprintf("          GITHUB_AW_ASSETS_ALLOWED_EXTS: %q", strings.Join(workflowData.SafeOutputs.UploadAssets.AllowedExts, ",")))
		}
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

	// Add cleanup step for network proxy hook files (if proxy was enabled)
	if workflowData.EngineConfig != nil && ShouldEnforceNetworkPermissions(workflowData.NetworkPermissions) {
		cleanupStep := GitHubActionStep{
			"      - name: Clean up network proxy hook files",
			"        if: always()",
			"        run: |",
			"          rm -rf .claude/hooks/network_permissions.py || true",
			"          rm -rf .claude/hooks || true",
			"          rm -rf .claude || true",
		}
		steps = append(steps, cleanupStep)
	}

	return steps
}

// convertStepToYAML converts a step map to YAML string - uses proper YAML serialization
func (e *ClaudeEngine) convertStepToYAML(stepMap map[string]any) (string, error) {
	return ConvertStepToYAML(stepMap)
}

// expandNeutralToolsToClaudeTools converts neutral tools to Claude-specific tools format
func (e *ClaudeEngine) expandNeutralToolsToClaudeTools(tools map[string]any) map[string]any {
	result := make(map[string]any)

	// Copy existing tools that are not neutral tools
	for key, value := range tools {
		switch key {
		case "bash", "web-fetch", "web-search", "edit", "playwright":
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

	// Handle playwright tool by converting it to an MCP tool configuration
	if _, hasPlaywright := tools["playwright"]; hasPlaywright {
		// Create playwright as an MCP tool with the same tools available as copilot agent
		playwrightMCP := map[string]any{
			"allowed": GetCopilotAgentPlaywrightTools(),
		}
		result["playwright"] = playwrightMCP
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
			// Handle cache-memory as a special case - it provides file system access but no MCP tool
			if toolName == "cache-memory" {
				// Cache-memory now provides simple file share access at /tmp/gh-aw/cache-memory/
				// Add path-specific Read and Write tools for the cache directory only
				cacheDirPattern := "/tmp/gh-aw/cache-memory/*"

				// Add path-specific tools for cache directory access
				if !slices.Contains(allowedTools, fmt.Sprintf("Read(%s)", cacheDirPattern)) {
					allowedTools = append(allowedTools, fmt.Sprintf("Read(%s)", cacheDirPattern))
				}
				if !slices.Contains(allowedTools, fmt.Sprintf("Write(%s)", cacheDirPattern)) {
					allowedTools = append(allowedTools, fmt.Sprintf("Write(%s)", cacheDirPattern))
				}
				if !slices.Contains(allowedTools, fmt.Sprintf("Edit(%s)", cacheDirPattern)) {
					allowedTools = append(allowedTools, fmt.Sprintf("Edit(%s)", cacheDirPattern))
				}
				if !slices.Contains(allowedTools, fmt.Sprintf("MultiEdit(%s)", cacheDirPattern)) {
					allowedTools = append(allowedTools, fmt.Sprintf("MultiEdit(%s)", cacheDirPattern))
				}
				continue
			}

			// Check if this is an MCP tool (has MCP-compatible type) or standard MCP tool (github)
			if mcpConfig, ok := toolValue.(map[string]any); ok {
				// Check if it's explicitly marked as MCP type
				isCustomMCP := false
				if hasMcp, _ := hasMCPConfig(mcpConfig); hasMcp {
					isCustomMCP = true
				}

				// Handle standard MCP tools (github, playwright) or tools with MCP-compatible type
				if toolName == "github" || toolName == "playwright" || isCustomMCP {
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
					} else if toolName == "github" {
						// For GitHub tools without explicit allowed list, use default GitHub tools
						for _, defaultTool := range constants.DefaultGitHubTools {
							allowedTools = append(allowedTools, fmt.Sprintf("mcp__github__%s", defaultTool))
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

func (e *ClaudeEngine) RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) {
	yaml.WriteString("          cat > /tmp/gh-aw/mcp-config/mcp-servers.json << 'EOF'\n")
	yaml.WriteString("          {\n")
	yaml.WriteString("            \"mcpServers\": {\n")

	// Add safe-outputs MCP server if safe-outputs are configured
	totalServers := len(mcpTools)
	serverCount := 0

	// Generate configuration for each MCP tool using shared logic
	for _, toolName := range mcpTools {
		serverCount++
		isLast := serverCount == totalServers

		switch toolName {
		case "github":
			githubTool := tools["github"]
			e.renderGitHubClaudeMCPConfig(yaml, githubTool, isLast)
		case "playwright":
			playwrightTool := tools["playwright"]
			e.renderPlaywrightMCPConfig(yaml, playwrightTool, isLast)
		case "cache-memory":
			e.renderCacheMemoryMCPConfig(yaml, isLast, workflowData)
		case "safe-outputs":
			e.renderSafeOutputsMCPConfig(yaml, isLast)
		case "web-fetch":
			renderMCPFetchServerConfig(yaml, "json", "              ", isLast, false)
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
// Supports both local (Docker) and remote (hosted) modes
func (e *ClaudeEngine) renderGitHubClaudeMCPConfig(yaml *strings.Builder, githubTool any, isLast bool) {
	githubType := getGitHubType(githubTool)
	customGitHubToken := getGitHubToken(githubTool)
	readOnly := getGitHubReadOnly(githubTool)
	toolsets := getGitHubToolsets(githubTool)

	yaml.WriteString("              \"github\": {\n")

	// Check if remote mode is enabled (type: remote)
	if githubType == "remote" {
		// Remote mode - use hosted GitHub MCP server
		yaml.WriteString("                \"type\": \"http\",\n")
		yaml.WriteString("                \"url\": \"https://api.githubcopilot.com/mcp/\",\n")
		yaml.WriteString("                \"headers\": {\n")

		// Add custom github-token if specified, otherwise use GITHUB_MCP_TOKEN
		if customGitHubToken != "" {
			yaml.WriteString(fmt.Sprintf("                  \"Authorization\": \"Bearer %s\"", customGitHubToken))
		} else {
			yaml.WriteString("                  \"Authorization\": \"Bearer ${{ secrets.GITHUB_MCP_TOKEN }}\"")
		}

		// Add X-MCP-Readonly header if read-only mode is enabled
		if readOnly {
			yaml.WriteString(",\n")
			yaml.WriteString("                  \"X-MCP-Readonly\": \"true\"\n")
		} else {
			yaml.WriteString("\n")
		}

		yaml.WriteString("                }\n")
	} else {
		// Local mode - use Docker-based GitHub MCP server (default)
		githubDockerImageVersion := getGitHubDockerImageVersion(githubTool)
		customArgs := getGitHubCustomArgs(githubTool)

		yaml.WriteString("                \"command\": \"docker\",\n")
		yaml.WriteString("                \"args\": [\n")
		yaml.WriteString("                  \"run\",\n")
		yaml.WriteString("                  \"-i\",\n")
		yaml.WriteString("                  \"--rm\",\n")
		yaml.WriteString("                  \"-e\",\n")
		yaml.WriteString("                  \"GITHUB_PERSONAL_ACCESS_TOKEN\",\n")
		if readOnly {
			yaml.WriteString("                  \"-e\",\n")
			yaml.WriteString("                  \"GITHUB_READ_ONLY=1\",\n")
		}
		yaml.WriteString("                  \"ghcr.io/github/github-mcp-server:" + githubDockerImageVersion + "\"")

		// Append custom args if present
		writeArgsToYAML(yaml, customArgs, "                  ")

		yaml.WriteString("\n")
		yaml.WriteString("                ],\n")
		yaml.WriteString("                \"env\": {\n")

		// Use custom token if specified, otherwise use default
		if customGitHubToken != "" {
			yaml.WriteString(fmt.Sprintf("                  \"GITHUB_PERSONAL_ACCESS_TOKEN\": \"%s\"", customGitHubToken))
		} else {
			yaml.WriteString("                  \"GITHUB_PERSONAL_ACCESS_TOKEN\": \"${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}\"")
		}

		// Add GITHUB_TOOLSETS environment variable if toolsets are configured
		if toolsets != "" {
			yaml.WriteString(",\n")
			yaml.WriteString(fmt.Sprintf("                  \"GITHUB_TOOLSETS\": \"%s\"", toolsets))
		}

		yaml.WriteString("\n")
		yaml.WriteString("                }\n")
	}

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}
}

// renderPlaywrightMCPConfig generates the Playwright MCP server configuration
// Uses npx to launch Playwright MCP instead of Docker for better performance and simplicity
func (e *ClaudeEngine) renderPlaywrightMCPConfig(yaml *strings.Builder, playwrightTool any, isLast bool) {
	args := generatePlaywrightDockerArgs(playwrightTool)
	customArgs := getPlaywrightCustomArgs(playwrightTool)

	yaml.WriteString("              \"playwright\": {\n")
	yaml.WriteString("                \"command\": \"npx\",\n")
	yaml.WriteString("                \"args\": [\n")
	yaml.WriteString("                  \"@playwright/mcp@latest\",\n")
	yaml.WriteString("                  \"--output-dir\",\n")
	yaml.WriteString("                  \"/tmp/gh-aw/mcp-logs/playwright\"")
	if len(args.AllowedDomains) > 0 {
		yaml.WriteString(",\n")
		yaml.WriteString("                  \"--allowed-origins\",\n")
		yaml.WriteString("                  \"" + strings.Join(args.AllowedDomains, ";") + "\"")
	}

	// Append custom args if present
	writeArgsToYAML(yaml, customArgs, "                  ")

	yaml.WriteString("\n")
	yaml.WriteString("                ]\n")

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

// renderCacheMemoryMCPConfig handles cache-memory configuration without MCP server mounting
// Cache-memory is now a simple file share, not an MCP server
func (e *ClaudeEngine) renderCacheMemoryMCPConfig(yaml *strings.Builder, isLast bool, workflowData *WorkflowData) {
	// Cache-memory no longer uses MCP server mounting
	// The cache folder is available as a simple file share at /tmp/gh-aw/cache-memory/
	// The folder is created by the cache step and is accessible to all tools
	// No MCP configuration is needed for simple file access
}

// renderSafeOutputsMCPConfig generates the Safe Outputs MCP server configuration
func (e *ClaudeEngine) renderSafeOutputsMCPConfig(yaml *strings.Builder, isLast bool) {
	yaml.WriteString("              \"safe_outputs\": {\n")
	yaml.WriteString("                \"command\": \"node\",\n")
	yaml.WriteString("                \"args\": [\"/tmp/gh-aw/safe-outputs/mcp-server.cjs\"],\n")
	yaml.WriteString("                \"env\": {\n")
	yaml.WriteString("                  \"GITHUB_AW_SAFE_OUTPUTS\": \"${{ env.GITHUB_AW_SAFE_OUTPUTS }}\",\n")
	yaml.WriteString("                  \"GITHUB_AW_SAFE_OUTPUTS_CONFIG\": ${{ toJSON(env.GITHUB_AW_SAFE_OUTPUTS_CONFIG) }},\n")
	yaml.WriteString("                  \"GITHUB_AW_ASSETS_BRANCH\": \"${{ env.GITHUB_AW_ASSETS_BRANCH }}\",\n")
	yaml.WriteString("                  \"GITHUB_AW_ASSETS_MAX_SIZE_KB\": \"${{ env.GITHUB_AW_ASSETS_MAX_SIZE_KB }}\",\n")
	yaml.WriteString("                  \"GITHUB_AW_ASSETS_ALLOWED_EXTS\": \"${{ env.GITHUB_AW_ASSETS_ALLOWED_EXTS }}\"\n")
	yaml.WriteString("                }\n")

	if isLast {
		yaml.WriteString("              }\n")
	} else {
		yaml.WriteString("              },\n")
	}
}

// ParseLogMetrics implements engine-specific log parsing for Claude
func (e *ClaudeEngine) ParseLogMetrics(logContent string, verbose bool) LogMetrics {
	var metrics LogMetrics
	var maxTokenUsage int

	// First try to parse as JSON array (Claude logs are structured as JSON arrays)
	if strings.TrimSpace(logContent) != "" {
		if resultMetrics := e.parseClaudeJSONLog(logContent, verbose); resultMetrics.TokenUsage > 0 || resultMetrics.EstimatedCost > 0 || resultMetrics.Turns > 0 {
			metrics.TokenUsage = resultMetrics.TokenUsage
			metrics.EstimatedCost = resultMetrics.EstimatedCost
			metrics.Turns = resultMetrics.Turns
			metrics.ToolCalls = resultMetrics.ToolCalls         // Copy tool calls
			metrics.ToolSequences = resultMetrics.ToolSequences // Copy tool sequences
		}
	}

	// Process line by line for error counting and fallback parsing
	lines := strings.Split(logContent, "\n")

	for lineNum, line := range lines {
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// If we haven't found cost data yet from JSON parsing, try streaming JSON
		if metrics.TokenUsage == 0 || metrics.EstimatedCost == 0 || metrics.Turns == 0 {
			jsonMetrics := ExtractJSONMetrics(line, verbose)
			if jsonMetrics.TokenUsage > 0 || jsonMetrics.EstimatedCost > 0 {
				// Check if this is a Claude result payload with aggregated costs
				if e.isClaudeResultPayload(line) {
					// For Claude result payloads, use the aggregated values directly
					if resultMetrics := e.extractClaudeResultMetrics(line); resultMetrics.TokenUsage > 0 || resultMetrics.EstimatedCost > 0 || resultMetrics.Turns > 0 {
						metrics.TokenUsage = resultMetrics.TokenUsage
						metrics.EstimatedCost = resultMetrics.EstimatedCost
						metrics.Turns = resultMetrics.Turns
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

		// Collect individual error and warning details
		lowerLine := strings.ToLower(line)
		if strings.Contains(lowerLine, "error") {
			// Extract error message (remove timestamp and common prefixes)
			message := extractErrorMessage(line)
			if message != "" {
				metrics.Errors = append(metrics.Errors, LogError{
					Line:    lineNum + 1, // 1-based line numbering
					Type:    "error",
					Message: message,
				})
			}
		}
		if strings.Contains(lowerLine, "warning") {
			// Extract warning message (remove timestamp and common prefixes)
			message := extractErrorMessage(line)
			if message != "" {
				metrics.Errors = append(metrics.Errors, LogError{
					Line:    lineNum + 1, // 1-based line numbering
					Type:    "warning",
					Message: message,
				})
			}
		}
	}

	// If no result payload was found, use the maximum from streaming JSON
	if metrics.TokenUsage == 0 {
		metrics.TokenUsage = maxTokenUsage
	}

	// Detect permission errors and create missing-tool entries
	e.detectPermissionErrorsAndCreateMissingTools(logContent, verbose)

	return metrics
}

// isClaudeResultPayload checks if the JSON line is a Claude result payload with type: "result"
func (e *ClaudeEngine) isClaudeResultPayload(line string) bool {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "{") || !strings.HasSuffix(trimmed, "}") {
		return false
	}

	var jsonData map[string]any
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
	var jsonData map[string]any
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
		if usageMap, ok := usage.(map[string]any); ok {
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

	// Extract number of turns
	if numTurns, exists := jsonData["num_turns"]; exists {
		if turns := ConvertToInt(numTurns); turns > 0 {
			metrics.Turns = turns
		}
	}

	// Note: Duration extraction is handled in the main parsing logic where we have access to tool calls
	// This is because we need to distribute duration among tool calls

	return metrics
}

// parseClaudeJSONLog parses Claude logs as a JSON array or mixed format (debug logs + JSONL)
func (e *ClaudeEngine) parseClaudeJSONLog(logContent string, verbose bool) LogMetrics {
	var metrics LogMetrics

	// Try to parse the entire log as a JSON array first (old format)
	var logEntries []map[string]any
	if err := json.Unmarshal([]byte(logContent), &logEntries); err != nil {
		// If that fails, try to parse as mixed format (debug logs + JSONL)
		if verbose {
			fmt.Printf("Failed to parse Claude log as JSON array, trying JSONL format: %v\n", err)
		}

		logEntries = []map[string]any{}
		lines := strings.Split(logContent, "\n")

		for i := 0; i < len(lines); i++ {
			line := lines[i]
			trimmedLine := strings.TrimSpace(line)
			if trimmedLine == "" {
				continue // Skip empty lines
			}

			// If a line looks like a JSON array (starts with '['), try to parse it as an array
			if strings.HasPrefix(trimmedLine, "[") {
				buf := trimmedLine
				// If the closing bracket is not on the same line, accumulate subsequent lines
				if !strings.Contains(trimmedLine, "]") {
					j := i + 1
					for j < len(lines) {
						buf += "\n" + lines[j]
						if strings.Contains(lines[j], "]") {
							// Advance outer loop to the line we consumed
							i = j
							break
						}
						j++
					}
				}

				var arr []map[string]any
				if err := json.Unmarshal([]byte(buf), &arr); err == nil {
					logEntries = append(logEntries, arr...)
					continue
				}

				// If parsing as a single-line or multi-line array failed, attempt to extract a JSON array substring
				openIdx := strings.Index(buf, "[")
				closeIdx := strings.LastIndex(buf, "]")
				if openIdx != -1 && closeIdx != -1 && closeIdx > openIdx {
					sub := buf[openIdx : closeIdx+1]
					var arr2 []map[string]any
					if err2 := json.Unmarshal([]byte(sub), &arr2); err2 == nil {
						logEntries = append(logEntries, arr2...)
						continue
					}
				}
			}

			// Skip debug log lines that don't start with '{'
			if !strings.HasPrefix(trimmedLine, "{") {
				continue
			}

			// Try to parse each line as JSON
			var jsonEntry map[string]any
			if err := json.Unmarshal([]byte(trimmedLine), &jsonEntry); err != nil {
				// Skip invalid JSON lines (could be partial debug output)
				if verbose {
					fmt.Printf("Skipping invalid JSON line: %s\n", trimmedLine)
				}
				continue
			}

			logEntries = append(logEntries, jsonEntry)
		}

		if len(logEntries) == 0 {
			if verbose {
				fmt.Printf("No valid JSON entries found in Claude log\n")
			}
			return metrics
		}

		if verbose {
			fmt.Printf("Extracted %d JSON entries from mixed format Claude log\n", len(logEntries))
		}
	}

	// Look for the result entry with type: "result"
	toolCallMap := make(map[string]*ToolCallInfo) // Track tool calls across entries
	var currentSequence []string                  // Track tool sequence within current context

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
					if usageMap, ok := usage.(map[string]any); ok {
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

				// Extract number of turns
				if numTurns, exists := entry["num_turns"]; exists {
					if turns := ConvertToInt(numTurns); turns > 0 {
						metrics.Turns = turns
					}
				}

				// Extract duration information and distribute to tool calls
				if durationMs, exists := entry["duration_ms"]; exists {
					if duration := ConvertToFloat(durationMs); duration > 0 {
						totalDuration := time.Duration(duration * float64(time.Millisecond))
						// Distribute the total duration among tool calls
						// Since we don't have per-tool timing, we approximate by using the total duration
						// as the maximum duration for all tools that don't have duration set yet
						e.distributeTotalDurationToToolCalls(toolCallMap, totalDuration)
					}
				}

				if verbose {
					fmt.Printf("Extracted from Claude result payload: tokens=%d, cost=%.4f, turns=%d\n",
						metrics.TokenUsage, metrics.EstimatedCost, metrics.Turns)
				}
				break
			} else if typeStr == "assistant" {
				// Parse tool_use entries for tool call statistics and sequence
				if message, exists := entry["message"]; exists {
					if messageMap, ok := message.(map[string]any); ok {
						if content, exists := messageMap["content"]; exists {
							if contentArray, ok := content.([]any); ok {
								sequenceInMessage := e.parseToolCallsWithSequence(contentArray, toolCallMap)
								if len(sequenceInMessage) > 0 {
									currentSequence = append(currentSequence, sequenceInMessage...)
								}
							}
						}
					}
				}
			}
		}

		// Parse tool_use entries for tool call statistics from both assistant and user entries
		if entry["type"] == "user" || entry["type"] == "assistant" {
			if message, exists := entry["message"]; exists {
				if messageMap, ok := message.(map[string]any); ok {
					if content, exists := messageMap["content"]; exists {
						if contentArray, ok := content.([]any); ok {
							e.parseToolCalls(contentArray, toolCallMap)
						}
					}
				}
			}
		}
	}

	// Add the complete sequence if we found any tool calls
	if len(currentSequence) > 0 {
		metrics.ToolSequences = append(metrics.ToolSequences, currentSequence)
	}

	if verbose && len(metrics.ToolSequences) > 0 {
		totalTools := 0
		for _, seq := range metrics.ToolSequences {
			totalTools += len(seq)
		}
		fmt.Printf("Claude parser extracted %d tool sequences with %d total tool calls\n",
			len(metrics.ToolSequences), totalTools)
	}

	// Convert tool call map to slice
	for _, toolInfo := range toolCallMap {
		metrics.ToolCalls = append(metrics.ToolCalls, *toolInfo)
	}

	// Sort tool calls by name for consistent output
	sort.Slice(metrics.ToolCalls, func(i, j int) bool {
		return metrics.ToolCalls[i].Name < metrics.ToolCalls[j].Name
	})

	return metrics
}

// parseToolCallsWithSequence extracts tool call information from Claude log content array and returns sequence
func (e *ClaudeEngine) parseToolCallsWithSequence(contentArray []any, toolCallMap map[string]*ToolCallInfo) []string {
	var sequence []string

	for _, contentItem := range contentArray {
		if contentMap, ok := contentItem.(map[string]any); ok {
			if contentType, exists := contentMap["type"]; exists {
				if typeStr, ok := contentType.(string); ok && typeStr == "tool_use" {
					// Extract tool name
					if toolName, exists := contentMap["name"]; exists {
						if nameStr, ok := toolName.(string); ok {
							// Skip internal tools as per existing JavaScript logic (disabled for tool graph visualization)
							// internalTools := []string{
							//	"Read", "Write", "Edit", "MultiEdit", "LS", "Grep", "Glob", "TodoWrite",
							// }
							// if slices.Contains(internalTools, nameStr) {
							//	continue
							// }

							// Prettify tool name
							prettifiedName := PrettifyToolName(nameStr)

							// Special handling for bash - each invocation is unique
							if nameStr == "Bash" {
								if input, exists := contentMap["input"]; exists {
									if inputMap, ok := input.(map[string]any); ok {
										if command, exists := inputMap["command"]; exists {
											if commandStr, ok := command.(string); ok {
												// Create unique bash entry with command info, avoiding colons
												uniqueBashName := fmt.Sprintf("bash_%s", e.shortenCommand(commandStr))
												prettifiedName = uniqueBashName
											}
										}
									}
								}
							}

							// Add to sequence
							sequence = append(sequence, prettifiedName)

							// Initialize or update tool call info
							if toolInfo, exists := toolCallMap[prettifiedName]; exists {
								toolInfo.CallCount++
							} else {
								toolCallMap[prettifiedName] = &ToolCallInfo{
									Name:          prettifiedName,
									CallCount:     1,
									MaxOutputSize: 0, // Will be updated when we find tool results
									MaxDuration:   0, // Will be updated when we find execution timing
								}
							}
						}
					}
				} else if typeStr == "tool_result" {
					// Extract output size for tool results
					if content, exists := contentMap["content"]; exists {
						if contentStr, ok := content.(string); ok {
							// Estimate token count (rough approximation: 1 token = ~4 characters)
							outputSize := len(contentStr) / 4

							// Find corresponding tool call to update max output size
							if toolUseID, exists := contentMap["tool_use_id"]; exists {
								if _, ok := toolUseID.(string); ok {
									// This is simplified - in a full implementation we'd track tool_use_id to tool name mapping
									// For now, we'll update the max output size for all tools (conservative estimate)
									for _, toolInfo := range toolCallMap {
										if outputSize > toolInfo.MaxOutputSize {
											toolInfo.MaxOutputSize = outputSize
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return sequence
}

// parseToolCalls extracts tool call information from Claude log content array without sequence tracking
func (e *ClaudeEngine) parseToolCalls(contentArray []any, toolCallMap map[string]*ToolCallInfo) {
	for _, contentItem := range contentArray {
		if contentMap, ok := contentItem.(map[string]any); ok {
			if contentType, exists := contentMap["type"]; exists {
				if typeStr, ok := contentType.(string); ok && typeStr == "tool_use" {
					// Extract tool name
					if toolName, exists := contentMap["name"]; exists {
						if nameStr, ok := toolName.(string); ok {
							// Prettify tool name
							prettifiedName := PrettifyToolName(nameStr)

							// Special handling for bash - each invocation is unique
							if nameStr == "Bash" {
								if input, exists := contentMap["input"]; exists {
									if inputMap, ok := input.(map[string]any); ok {
										if command, exists := inputMap["command"]; exists {
											if commandStr, ok := command.(string); ok {
												// Create unique bash entry with command info, avoiding colons
												uniqueBashName := fmt.Sprintf("bash_%s", e.shortenCommand(commandStr))
												prettifiedName = uniqueBashName
											}
										}
									}
								}
							}

							// Initialize or update tool call info
							if toolInfo, exists := toolCallMap[prettifiedName]; exists {
								toolInfo.CallCount++
							} else {
								toolCallMap[prettifiedName] = &ToolCallInfo{
									Name:          prettifiedName,
									CallCount:     1,
									MaxOutputSize: 0, // Will be updated when we find tool results
									MaxDuration:   0, // Will be updated when we find execution timing
								}
							}
						}
					} else if typeStr == "tool_result" {
						// Extract output size for tool results
						if content, exists := contentMap["content"]; exists {
							if contentStr, ok := content.(string); ok {
								// Estimate token count (rough approximation: 1 token = ~4 characters)
								outputSize := len(contentStr) / 4

								// Find corresponding tool call to update max output size
								if toolUseID, exists := contentMap["tool_use_id"]; exists {
									if _, ok := toolUseID.(string); ok {
										// This is simplified - in a full implementation we'd track tool_use_id to tool name mapping
										// For now, we'll update the max output size for all tools (conservative estimate)
										for _, toolInfo := range toolCallMap {
											if outputSize > toolInfo.MaxOutputSize {
												toolInfo.MaxOutputSize = outputSize
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
}

// shortenCommand creates a short identifier for bash commands
func (e *ClaudeEngine) shortenCommand(command string) string {
	// Take first 20 characters and remove newlines
	shortened := strings.ReplaceAll(command, "\n", " ")
	if len(shortened) > 20 {
		shortened = shortened[:20] + "..."
	}
	return shortened
}

// distributeTotalDurationToToolCalls distributes the total workflow duration among tool calls
// Since Claude logs don't provide per-tool timing, we approximate by assigning the total duration
// to all tools that don't have a duration set yet, simulating that they all could have taken this long
func (e *ClaudeEngine) distributeTotalDurationToToolCalls(toolCallMap map[string]*ToolCallInfo, totalDuration time.Duration) {
	// Count tools that don't have duration set yet
	toolsWithoutDuration := 0
	for _, toolInfo := range toolCallMap {
		if toolInfo.MaxDuration == 0 {
			toolsWithoutDuration++
		}
	}

	// If no tools without duration, don't update anything
	if toolsWithoutDuration == 0 {
		return
	}

	// For Claude logs, since we only have total duration, we assign the total duration
	// as the maximum possible duration for each tool. This is conservative but gives
	// users an idea of the overall workflow timing
	for _, toolInfo := range toolCallMap {
		if toolInfo.MaxDuration == 0 {
			toolInfo.MaxDuration = totalDuration
		}
	}
}

// GetLogParserScriptId returns the JavaScript script name for parsing Claude logs
func (e *ClaudeEngine) GetLogParserScriptId() string {
	return "parse_claude_log"
}

// GetErrorPatterns returns regex patterns for extracting error messages from Claude logs
// including permission-related errors that should be captured as missing tools
func (e *ClaudeEngine) GetErrorPatterns() []ErrorPattern {
	return []ErrorPattern{
		// Specific, contextual error patterns - these are precise and unlikely to match informational text
		{
			Pattern:      `(?i)access denied.*only authorized.*can trigger.*workflow`,
			LevelGroup:   0,
			MessageGroup: 0,
			Description:  "Permission denied - workflow access restriction",
		},
		{
			Pattern:      `(?i)access denied.*user.*not authorized`,
			LevelGroup:   0,
			MessageGroup: 0,
			Description:  "Permission denied - user not authorized",
		},
		{
			Pattern:      `(?i)repository permission check failed`,
			LevelGroup:   0,
			MessageGroup: 0,
			Description:  "Repository permission check failure",
		},
		{
			Pattern:      `(?i)configuration error.*required permissions not specified`,
			LevelGroup:   0,
			MessageGroup: 0,
			Description:  "Configuration error - missing permissions",
		},
		{
			Pattern:      `(?i)\berror\b.*permission.*denied`,
			LevelGroup:   0,
			MessageGroup: 0,
			Description:  "Permission denied error (requires error context)",
		},
		{
			Pattern:      `(?i)\berror\b.*unauthorized`,
			LevelGroup:   0,
			MessageGroup: 0,
			Description:  "Unauthorized error (requires error context)",
		},
		{
			Pattern:      `(?i)\berror\b.*forbidden`,
			LevelGroup:   0,
			MessageGroup: 0,
			Description:  "Forbidden error (requires error context)",
		},
		{
			Pattern:      `(?i)\berror\b.*access.*restricted`,
			LevelGroup:   0,
			MessageGroup: 0,
			Description:  "Access restricted error (requires error context)",
		},
		{
			Pattern:      `(?i)\berror\b.*insufficient.*permission`,
			LevelGroup:   0,
			MessageGroup: 0,
			Description:  "Insufficient permissions error (requires error context)",
		},
	}
}

// detectPermissionErrorsAndCreateMissingTools scans Claude log content for permission errors
// and creates missing-tool entries in the safe outputs file
func (e *ClaudeEngine) detectPermissionErrorsAndCreateMissingTools(logContent string, verbose bool) {
	// Get permission error patterns
	patterns := e.GetErrorPatterns()

	// Try to parse as JSON array first (Claude logs are structured)
	var logEntries []map[string]any
	if err := json.Unmarshal([]byte(logContent), &logEntries); err == nil {
		e.processClaudeJSONForPermissionErrors(logEntries, verbose)
	} else {
		// Fallback to line-by-line processing for mixed format logs
		e.processClaudeTextForPermissionErrors(logContent, patterns, verbose)
	}
}

// processClaudeJSONForPermissionErrors processes structured Claude JSON logs for permission errors
func (e *ClaudeEngine) processClaudeJSONForPermissionErrors(logEntries []map[string]any, verbose bool) {
	for _, entry := range logEntries {
		if entry["type"] == "user" {
			if message, exists := entry["message"]; exists {
				if messageMap, ok := message.(map[string]any); ok {
					if content, exists := messageMap["content"]; exists {
						if contentArray, ok := content.([]any); ok {
							for _, contentItem := range contentArray {
								if contentMap, ok := contentItem.(map[string]any); ok {
									if contentMap["type"] == "tool_result" && contentMap["is_error"] == true {
										// Check if the error content matches permission patterns
										if contentStr, exists := contentMap["content"].(string); exists {
											if e.isPermissionError(contentStr) {
												// Extract tool name from tool_use_id or context
												toolName := e.extractToolNameFromContext(logEntries, contentMap, "unknown-tool")
												e.createMissingToolEntry(toolName, contentStr, verbose)
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
}

// processClaudeTextForPermissionErrors processes text-based Claude logs for permission errors
func (e *ClaudeEngine) processClaudeTextForPermissionErrors(logContent string, patterns []ErrorPattern, verbose bool) {
	lines := strings.Split(logContent, "\n")

	for _, pattern := range patterns {
		regex, err := regexp.Compile(pattern.Pattern)
		if err != nil {
			continue // Skip invalid patterns
		}

		for _, line := range lines {
			if regex.MatchString(line) {
				// Found a permission error, create missing-tool entry
				toolName := "permission-restricted-tool"
				e.createMissingToolEntry(toolName, line, verbose)
			}
		}
	}
}

// isPermissionError checks if an error message indicates a permission issue
func (e *ClaudeEngine) isPermissionError(errorContent string) bool {
	patterns := []string{
		`(?i)access denied.*only authorized.*can trigger.*workflow`,
		`(?i)access denied.*user.*not authorized`,
		`(?i)repository permission check failed`,
		`(?i)configuration error.*required permissions not specified`,
		`(?i)permission.*denied`,
		`(?i)unauthorized`,
		`(?i)forbidden`,
		`(?i)access.*restricted`,
		`(?i)insufficient.*permission`,
	}

	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, errorContent); matched {
			return true
		}
	}
	return false
}

// extractToolNameFromContext attempts to extract the tool name that failed due to permissions
func (e *ClaudeEngine) extractToolNameFromContext(logEntries []map[string]any, toolResult map[string]any, defaultTool string) string {
	// Try to find the corresponding tool_use entry by tool_use_id
	if toolUseID, exists := toolResult["tool_use_id"].(string); exists {
		for _, entry := range logEntries {
			if entry["type"] == "assistant" {
				if message, exists := entry["message"]; exists {
					if messageMap, ok := message.(map[string]any); ok {
						if content, exists := messageMap["content"]; exists {
							if contentArray, ok := content.([]any); ok {
								for _, contentItem := range contentArray {
									if contentMap, ok := contentItem.(map[string]any); ok {
										if contentMap["type"] == "tool_use" && contentMap["id"] == toolUseID {
											if name, exists := contentMap["name"].(string); exists {
												return name
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return defaultTool
}

// createMissingToolEntry creates a missing-tool entry in the safe outputs file
func (e *ClaudeEngine) createMissingToolEntry(toolName, reason string, verbose bool) {
	// Get the safe outputs file path from environment
	safeOutputsFile := os.Getenv("GITHUB_AW_SAFE_OUTPUTS")
	if safeOutputsFile == "" {
		if verbose {
			fmt.Printf("GITHUB_AW_SAFE_OUTPUTS not set, cannot write permission error missing-tool entry\n")
		}
		return
	}

	// Create missing-tool entry
	missingToolEntry := map[string]any{
		"type":         "missing-tool",
		"tool":         toolName,
		"reason":       fmt.Sprintf("Permission denied: %s", reason),
		"alternatives": "Check repository permissions and access controls",
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
	}

	// Convert to JSON and append to safe outputs file
	entryJSON, err := json.Marshal(missingToolEntry)
	if err != nil {
		if verbose {
			fmt.Printf("Failed to marshal missing-tool entry: %v\n", err)
		}
		return
	}

	// Append to the safe outputs file
	file, err := os.OpenFile(safeOutputsFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		if verbose {
			fmt.Printf("Failed to open safe outputs file: %v\n", err)
		}
		return
	}
	defer file.Close()

	if _, err := file.WriteString(string(entryJSON) + "\n"); err != nil {
		if verbose {
			fmt.Printf("Failed to write missing-tool entry: %v\n", err)
		}
		return
	}

	if verbose {
		fmt.Printf("Recorded permission error as missing tool: %s\n", toolName)
	}
}
