package workflow

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var claudeToolsLog = logger.New("workflow:claude_tools")

// expandNeutralToolsToClaudeTools converts neutral tool names to Claude-specific tool configurations
func (e *ClaudeEngine) expandNeutralToolsToClaudeTools(toolsConfig *ToolsConfig) *ToolsConfig {
	if toolsConfig == nil {
		return &ToolsConfig{
			Custom: make(map[string]any),
			raw:    make(map[string]any),
		}
	}

	result := &ToolsConfig{
		Custom: make(map[string]any),
		raw:    make(map[string]any),
	}

	// Count neutral tools
	neutralToolCount := 0
	if toolsConfig.Bash != nil {
		neutralToolCount++
	}
	if toolsConfig.WebFetch != nil {
		neutralToolCount++
	}
	if toolsConfig.WebSearch != nil {
		neutralToolCount++
	}
	if toolsConfig.Edit != nil {
		neutralToolCount++
	}
	if toolsConfig.Playwright != nil {
		neutralToolCount++
	}

	if neutralToolCount > 0 {
		claudeToolsLog.Printf("Expanding %d neutral tools to Claude-specific tools", neutralToolCount)
	}

	// Copy non-neutral tools (GitHub, custom MCP servers, etc.)
	if toolsConfig.GitHub != nil {
		result.GitHub = toolsConfig.GitHub
	}
	if toolsConfig.Serena != nil {
		result.Serena = toolsConfig.Serena
	}
	if toolsConfig.AgenticWorkflows != nil {
		result.AgenticWorkflows = toolsConfig.AgenticWorkflows
	}
	if toolsConfig.CacheMemory != nil {
		result.CacheMemory = toolsConfig.CacheMemory
	}
	if toolsConfig.RepoMemory != nil {
		result.RepoMemory = toolsConfig.RepoMemory
	}
	if toolsConfig.SafetyPrompt != nil {
		result.SafetyPrompt = toolsConfig.SafetyPrompt
	}
	if toolsConfig.Timeout != nil {
		result.Timeout = toolsConfig.Timeout
	}
	if toolsConfig.StartupTimeout != nil {
		result.StartupTimeout = toolsConfig.StartupTimeout
	}

	// Copy custom tools (MCP servers)
	for name, config := range toolsConfig.Custom {
		result.Custom[name] = config
	}

	// Create or get existing claude section from custom tools
	var claudeSection map[string]any
	if existing, hasClaudeSection := result.Custom["claude"]; hasClaudeSection {
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
	if toolsConfig.Bash != nil {
		// bash -> Bash, KillBash, BashOutput
		if len(toolsConfig.Bash.AllowedCommands) > 0 {
			// Convert []string to []any for storage
			bashCommands := make([]any, len(toolsConfig.Bash.AllowedCommands))
			for i, cmd := range toolsConfig.Bash.AllowedCommands {
				bashCommands[i] = cmd
			}
			claudeAllowed["Bash"] = bashCommands
		} else {
			claudeAllowed["Bash"] = nil // Allow all bash commands
		}
	}

	if toolsConfig.WebFetch != nil {
		// web-fetch -> WebFetch
		claudeAllowed["WebFetch"] = nil
	}

	if toolsConfig.WebSearch != nil {
		// web-search -> WebSearch
		claudeAllowed["WebSearch"] = nil
	}

	if toolsConfig.Edit != nil {
		// edit -> Edit, MultiEdit, NotebookEdit, Write
		claudeAllowed["Edit"] = nil
		claudeAllowed["MultiEdit"] = nil
		claudeAllowed["NotebookEdit"] = nil
		claudeAllowed["Write"] = nil
	}

	// Handle playwright tool by converting it to an MCP tool configuration
	if toolsConfig.Playwright != nil {
		// Create playwright as an MCP tool with the same tools available as copilot agent
		playwrightMCP := map[string]any{
			"allowed": GetCopilotAgentPlaywrightTools(),
		}
		result.Custom["playwright"] = playwrightMCP
	}

	// Update claude section
	claudeSection["allowed"] = claudeAllowed
	result.Custom["claude"] = claudeSection

	return result
}

// computeAllowedClaudeToolsString generates the tool specification string for Claude's --allowed-tools flag.
//
// Why --allowed-tools instead of --tools (introduced in v2.0.31)?
// While --tools is simpler (e.g., "Bash,Edit,Read"), it lacks the fine-grained control gh-aw requires:
// - Specific bash commands: Bash(git:*), Bash(ls)
// - MCP tool prefixes: mcp__github__issue_read, mcp__github__*
// - Path-specific access: Read(/tmp/gh-aw/cache-memory/*)
//
// This function:
// 1. validates that only neutral tools are provided (no claude section)
// 2. converts neutral tools to Claude-specific tools format
// 3. adds default Claude tools and git commands based on safe outputs configuration
// 4. generates the allowed tools string for Claude
func (e *ClaudeEngine) computeAllowedClaudeToolsString(toolsConfig *ToolsConfig, safeOutputs *SafeOutputsConfig, cacheMemoryConfig *CacheMemoryConfig) string {
	claudeToolsLog.Print("Computing allowed Claude tools string")

	// Initialize tools config if nil
	if toolsConfig == nil {
		toolsConfig = &ToolsConfig{
			Custom: make(map[string]any),
			raw:    make(map[string]any),
		}
	}

	// Enforce that only neutral tools are provided - fail if claude section is present in custom tools
	if _, hasClaudeSection := toolsConfig.Custom["claude"]; hasClaudeSection {
		panic("computeAllowedClaudeToolsString should only receive neutral tools, not claude section tools")
	}

	// Convert neutral tools to Claude-specific tools
	toolsConfig = e.expandNeutralToolsToClaudeTools(toolsConfig)

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

	// Ensure claude section exists with the new format (it should exist after expandNeutralToolsToClaudeTools)
	var claudeSection map[string]any
	if existing, hasClaudeSection := toolsConfig.Custom["claude"]; hasClaudeSection {
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
	toolsConfig.Custom["claude"] = claudeSection

	claudeToolsLog.Printf("Added %d default Claude tools to allowed list", len(defaultClaudeTools))

	var allowedTools []string

	// Process claude-specific tools from the claude section (new format only)
	if claudeSection, hasClaudeSection := toolsConfig.Custom["claude"]; hasClaudeSection {
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
	// Handle cache-memory as a special case - it provides file system access but no MCP tool
	if toolsConfig.CacheMemory != nil {
		// Cache-memory provides file share access
		// Default cache uses /tmp/gh-aw/cache-memory/, others use /tmp/gh-aw/cache-memory-{id}/
		// Add path-specific Read and Write tools for each cache directory
		if cacheMemoryConfig != nil {
			for _, cache := range cacheMemoryConfig.Caches {
				var cacheDirPattern string
				if cache.ID == "default" {
					cacheDirPattern = "/tmp/gh-aw/cache-memory/*"
				} else {
					cacheDirPattern = fmt.Sprintf("/tmp/gh-aw/cache-memory-%s/*", cache.ID)
				}

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
			}
		}
	}

	// Process GitHub tool
	if toolsConfig.GitHub != nil {
		githubConfig := toolsConfig.GitHub
		if len(githubConfig.Allowed) > 0 {
			// Check for wildcard access first
			hasWildcard := false
			for _, item := range githubConfig.Allowed {
				if item == "*" {
					hasWildcard = true
					break
				}
			}

			if hasWildcard {
				// For wildcard access, just add the server name with mcp__ prefix
				allowedTools = append(allowedTools, "mcp__github")
			} else {
				// For specific tools, add each one individually
				for _, item := range githubConfig.Allowed {
					allowedTools = append(allowedTools, fmt.Sprintf("mcp__github__%s", item))
				}
			}
		} else {
			// For GitHub tools without explicit allowed list, use appropriate default GitHub tools based on mode
			githubMode := githubConfig.Mode
			if githubMode == "" {
				githubMode = "remote" // default mode
			}
			var defaultTools []string
			if githubMode == "remote" {
				defaultTools = constants.DefaultGitHubToolsRemote
			} else {
				defaultTools = constants.DefaultGitHubToolsLocal
			}
			for _, defaultTool := range defaultTools {
				allowedTools = append(allowedTools, fmt.Sprintf("mcp__github__%s", defaultTool))
			}
		}
	}

	// Process custom MCP tools (including playwright if it was converted)
	for toolName, toolValue := range toolsConfig.Custom {
		if toolName == "claude" {
			// Skip the claude section as we've already processed it
			continue
		}

		// Check if this is an MCP tool (has MCP-compatible type) or standard MCP tool
		if mcpConfig, ok := toolValue.(map[string]any); ok {
			// Check if it's explicitly marked as MCP type
			isCustomMCP := false
			if hasMcp, _ := hasMCPConfig(mcpConfig); hasMcp {
				isCustomMCP = true
			}

			// Handle standard MCP tools (playwright) or tools with MCP-compatible type
			if toolName == "playwright" || isCustomMCP {
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

	// Handle SafeOutputs requirement for file write access
	if safeOutputs != nil {
		// Check if a general "Write" permission is already granted
		hasGeneralWrite := slices.Contains(allowedTools, "Write")

		// If no general Write permission and SafeOutputs is configured,
		// add specific write permission for GH_AW_SAFE_OUTPUTS
		if !hasGeneralWrite {
			allowedTools = append(allowedTools, "Write")
			// Ideally we would only give permission to the exact file, but that doesn't seem
			// to be working with Claude. See https://github.com/githubnext/gh-aw/issues/244#issuecomment-3240319103
			//allowedTools = append(allowedTools, "Write(${{ env.GH_AW_SAFE_OUTPUTS }})")
		}
	}

	// Sort the allowed tools alphabetically for consistent output
	sort.Strings(allowedTools)

	if log.Enabled() {
		claudeToolsLog.Printf("Generated allowed tools string with %d tools", len(allowedTools))
	}

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
		fmt.Fprintf(&comment, "%s# - %s\n", indent, tool)
	}

	return comment.String()
}
