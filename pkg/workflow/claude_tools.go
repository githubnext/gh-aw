package workflow

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/typeutil"
)

var claudeToolsLog = logger.New("workflow:claude_tools")

const defaultClaudeTmpWritePath = "/tmp"

var defaultClaudeToolNames = []string{"Task", "Glob", "Grep", "ExitPlanMode", "TodoWrite", "LS", "Read", "NotebookRead"}

func countNeutralClaudeTools(tools map[string]any) int {
	count := 0
	for key := range tools {
		if isNeutralClaudeTool(key) {
			count++
		}
	}
	return count
}

func isNeutralClaudeTool(key string) bool {
	switch key {
	case "bash", "web-fetch", "web-search", "edit", "playwright":
		return true
	default:
		return false
	}
}

func copyNonNeutralClaudeTools(tools map[string]any) map[string]any {
	result := make(map[string]any)
	for key, value := range tools {
		if !isNeutralClaudeTool(key) {
			result[key] = value
		}
	}
	return result
}

func ensureClaudeAllowedMap(tools map[string]any) map[string]any {
	claudeSection, _ := tools["claude"].(map[string]any)
	if claudeSection == nil {
		claudeSection = make(map[string]any)
		tools["claude"] = claudeSection
	}
	claudeAllowed, _ := claudeSection["allowed"].(map[string]any)
	if claudeAllowed == nil {
		claudeAllowed = make(map[string]any)
		claudeSection["allowed"] = claudeAllowed
	}
	return claudeAllowed
}

func addNeutralClaudeTools(result map[string]any, claudeAllowed map[string]any, tools map[string]any) {
	if bashTool, hasBash := tools["bash"]; hasBash {
		if bashCommands, ok := bashTool.([]any); ok {
			claudeAllowed["Bash"] = bashCommands
		} else {
			claudeAllowed["Bash"] = nil
		}
	}
	if _, hasWebFetch := tools["web-fetch"]; hasWebFetch {
		claudeAllowed["WebFetch"] = nil
	}
	if _, hasWebSearch := tools["web-search"]; hasWebSearch {
		claudeAllowed["WebSearch"] = nil
	}
	if editTool, hasEdit := tools["edit"]; hasEdit && !isExplicitlyDisabledTool(editTool) {
		addEditClaudeTools(claudeAllowed)
	}
	if _, hasPlaywright := tools["playwright"]; hasPlaywright {
		result["playwright"] = map[string]any{"allowed": GetPlaywrightTools()}
	}
}

func addEditClaudeTools(claudeAllowed map[string]any) {
	claudeAllowed["Edit"] = nil
	claudeAllowed["MultiEdit"] = nil
	claudeAllowed["NotebookEdit"] = nil
	claudeAllowed["Write"] = nil
}

func normalizeClaudeToolsInput(tools map[string]any) map[string]any {
	if tools == nil {
		return make(map[string]any)
	}
	if _, hasClaudeSection := tools["claude"]; hasClaudeSection {
		claudeToolsLog.Print("BUG: Claude section found in input tools, should only contain neutral tools")
		panic("BUG: computeAllowedClaudeToolsString should only receive neutral tools, not claude section tools")
	}
	return tools
}

func addDefaultClaudeAllowedTools(tools map[string]any) {
	claudeAllowed := ensureClaudeAllowedMap(tools)
	for _, defaultTool := range defaultClaudeToolNames {
		if _, exists := claudeAllowed[defaultTool]; !exists {
			claudeAllowed[defaultTool] = nil
		}
	}
	if _, hasBash := claudeAllowed["Bash"]; hasBash {
		if _, exists := claudeAllowed["KillBash"]; !exists {
			claudeAllowed["KillBash"] = nil
		}
		if _, exists := claudeAllowed["BashOutput"]; !exists {
			claudeAllowed["BashOutput"] = nil
		}
	}
}

func collectClaudeAllowedTools(tools map[string]any) []string {
	claudeConfig, ok := typeutil.LookupMap(tools, "claude")
	if !ok {
		return nil
	}
	allowedMap, ok := typeutil.LookupMap(claudeConfig, "allowed")
	if !ok {
		return nil
	}
	allowedTools := make([]string, 0, len(allowedMap))
	for toolName, toolValue := range allowedMap {
		if toolName == "Bash" {
			allowedTools = appendClaudeBashTools(allowedTools, toolValue)
			continue
		}
		if strings.HasPrefix(toolName, strings.ToUpper(toolName[:1])) {
			allowedTools = append(allowedTools, toolName)
		}
	}
	return allowedTools
}

func appendClaudeBashTools(allowedTools []string, toolValue any) []string {
	bashCommands, ok := toolValue.([]any)
	if !ok {
		return append(allowedTools, "Bash")
	}
	if hasClaudeBashWildcard(bashCommands, ":*") || hasClaudeBashWildcard(bashCommands, "*") {
		return append(allowedTools, "Bash")
	}
	for _, cmd := range bashCommands {
		cmdStr, ok := cmd.(string)
		if !ok {
			continue
		}
		normalized, _ := normalizeBashCommand(cmdStr)
		allowedTools = append(allowedTools, fmt.Sprintf("Bash(%s)", normalized))
	}
	return allowedTools
}

func hasClaudeBashWildcard(bashCommands []any, wildcard string) bool {
	for _, cmd := range bashCommands {
		if cmdStr, ok := cmd.(string); ok && cmdStr == wildcard {
			return true
		}
	}
	return false
}

func appendTopLevelClaudeTools(allowedTools []string, tools map[string]any, cacheMemoryConfig *CacheMemoryConfig) []string {
	for toolName, toolValue := range tools {
		switch toolName {
		case "claude":
			continue
		case "cache-memory":
			allowedTools = appendCacheMemoryClaudeTools(allowedTools, cacheMemoryConfig)
			continue
		case "agentic-workflows":
			allowedTools = append(allowedTools, "mcp__"+string(constants.AgenticWorkflowsMCPServerID))
			continue
		}
		mcpConfig, ok := toolValue.(map[string]any)
		if !ok {
			continue
		}
		if toolName == "github" {
			allowedTools = appendGitHubMCPTools(allowedTools, toolName, toolValue, mcpConfig)
			continue
		}
		if hasMcp, _ := hasMCPConfig(mcpConfig); toolName == "playwright" || hasMcp {
			allowedTools = appendGenericMCPTools(allowedTools, toolName, mcpConfig)
		}
	}
	return allowedTools
}

func appendCacheMemoryClaudeTools(allowedTools []string, cacheMemoryConfig *CacheMemoryConfig) []string {
	if cacheMemoryConfig == nil {
		return allowedTools
	}
	for _, cache := range cacheMemoryConfig.Caches {
		allowedTools = appendCacheDirectoryClaudeTools(allowedTools, cache.ID)
	}
	return allowedTools
}

func appendCacheDirectoryClaudeTools(allowedTools []string, cacheID string) []string {
	cacheDir := cacheMemoryDirFor(cacheID)
	cacheDirPattern := cacheDir + "/*"
	allowedTools = appendUniqueClaudeTools(allowedTools,
		fmt.Sprintf("Read(%s)", cacheDirPattern),
		fmt.Sprintf("Write(%s)", cacheDirPattern),
		fmt.Sprintf("Edit(%s)", cacheDirPattern),
		fmt.Sprintf("MultiEdit(%s)", cacheDirPattern),
	)
	if slices.Contains(allowedTools, "Bash") {
		return allowedTools
	}
	return appendCacheBashClaudeTools(allowedTools, cacheDir)
}

func appendCacheBashClaudeTools(allowedTools []string, cacheDir string) []string {
	cacheDirSlash := cacheDir + "/"
	return appendUniqueClaudeTools(allowedTools,
		fmt.Sprintf("Bash(mkdir -p %s)", cacheDirSlash),
		fmt.Sprintf("Bash(cat %s)", cacheDirSlash),
		fmt.Sprintf("Bash(cat > %s)", cacheDirSlash),
		fmt.Sprintf("Bash(mv %s)", cacheDirSlash),
		"BashOutput",
		"KillBash",
	)
}

func appendUniqueClaudeTools(allowedTools []string, tools ...string) []string {
	for _, tool := range tools {
		if !slices.Contains(allowedTools, tool) {
			allowedTools = append(allowedTools, tool)
		}
	}
	return allowedTools
}

func appendGitHubMCPTools(allowedTools []string, toolName string, toolValue any, mcpConfig map[string]any) []string {
	githubConfig := parseGitHubTool(toolValue)
	if githubConfig != nil && len(githubConfig.Allowed) > 0 {
		for _, tool := range githubConfig.Allowed {
			if string(tool) == "*" {
				return append(allowedTools, "mcp__"+toolName)
			}
			allowedTools = append(allowedTools, fmt.Sprintf("mcp__%s__%s", toolName, string(tool)))
		}
		return allowedTools
	}
	defaultTools := constants.DefaultGitHubToolsLocal
	if getGitHubType(mcpConfig) == "remote" {
		defaultTools = constants.DefaultGitHubToolsRemote
	}
	for _, defaultTool := range defaultTools {
		allowedTools = append(allowedTools, "mcp__github__"+defaultTool)
	}
	return allowedTools
}

func appendGenericMCPTools(allowedTools []string, toolName string, mcpConfig map[string]any) []string {
	allowed, hasAllowed := mcpConfig["allowed"]
	if !hasAllowed {
		return append(allowedTools, "mcp__"+toolName)
	}
	allowedSlice, ok := allowed.([]any)
	if !ok {
		return allowedTools
	}
	return appendAllowedMCPEntries(allowedTools, toolName, allowedSlice)
}

func appendAllowedMCPEntries(allowedTools []string, toolName string, allowedSlice []any) []string {
	for _, item := range allowedSlice {
		if str, ok := item.(string); ok && str == "*" {
			return append(allowedTools, "mcp__"+toolName)
		}
	}
	for _, item := range allowedSlice {
		str, ok := item.(string)
		if ok {
			allowedTools = append(allowedTools, fmt.Sprintf("mcp__%s__%s", toolName, str))
		}
	}
	return allowedTools
}

func appendSandboxWritableClaudeTools(allowedTools []string, sandboxConfig *SandboxConfig) []string {
	if sandboxConfig == nil {
		return allowedTools
	}
	writablePaths := []string{defaultClaudeTmpWritePath}
	if sandboxConfig.Agent != nil && sandboxConfig.Agent.Config != nil && sandboxConfig.Agent.Config.Filesystem != nil {
		writablePaths = append(writablePaths, sandboxConfig.Agent.Config.Filesystem.AllowWrite...)
	}
	seenPatterns := make(map[string]struct{}, len(writablePaths))
	for _, writablePath := range writablePaths {
		allowedTools = appendPathScopedClaudeTools(allowedTools, writablePath, seenPatterns)
	}
	return allowedTools
}

func appendPathScopedClaudeTools(allowedTools []string, writablePath string, seenPatterns map[string]struct{}) []string {
	path := strings.TrimSpace(writablePath)
	if path == "" || !strings.HasPrefix(path, "/") {
		return allowedTools
	}
	pattern := path
	if !strings.ContainsAny(pattern, "*?[]{}") {
		pattern = strings.TrimRight(pattern, "/") + "/*"
	}
	if _, seen := seenPatterns[pattern]; seen {
		return allowedTools
	}
	seenPatterns[pattern] = struct{}{}
	return appendUniqueClaudeTools(allowedTools,
		fmt.Sprintf("Read(%s)", pattern),
		fmt.Sprintf("Write(%s)", pattern),
		fmt.Sprintf("Edit(%s)", pattern),
		fmt.Sprintf("MultiEdit(%s)", pattern),
	)
}

func appendSafeOutputsClaudeTools(allowedTools []string, safeOutputs *SafeOutputsConfig) []string {
	if safeOutputs == nil {
		return allowedTools
	}
	allowedTools = append(allowedTools, "mcp__"+string(constants.SafeOutputsMCPServerID))
	if !slices.Contains(allowedTools, "Write") {
		allowedTools = append(allowedTools, "Write")
	}
	return allowedTools
}

func appendMCPScriptsClaudeTools(allowedTools []string, mcpScripts *MCPScriptsConfig) []string {
	if HasMCPScripts(mcpScripts) {
		allowedTools = append(allowedTools, "mcp__"+string(constants.MCPScriptsMCPServerID))
	}
	return allowedTools
}

func dedupeClaudeAllowedTools(allowedTools []string) []string {
	if len(allowedTools) < 2 {
		return allowedTools
	}
	seen := make(map[string]struct{}, len(allowedTools))
	deduped := make([]string, 0, len(allowedTools))
	for _, tool := range allowedTools {
		if _, ok := seen[tool]; ok {
			continue
		}
		seen[tool] = struct{}{}
		deduped = append(deduped, tool)
	}
	return deduped
}

// expandNeutralToolsToClaudeTools converts neutral tool names to Claude-specific tool configurations
func (e *ClaudeEngine) expandNeutralToolsToClaudeTools(tools map[string]any) map[string]any {
	claudeToolsLog.Printf("Starting neutral tools expansion: input_tools=%d", len(tools))

	neutralToolCount := countNeutralClaudeTools(tools)
	if neutralToolCount > 0 {
		claudeToolsLog.Printf("Expanding %d neutral tools to Claude-specific tools", neutralToolCount)
	}

	result := copyNonNeutralClaudeTools(tools)
	claudeAllowed := ensureClaudeAllowedMap(result)
	addNeutralClaudeTools(result, claudeAllowed, tools)

	claudeToolsLog.Printf("Expansion complete: result_tools=%d, claude_allowed=%d", len(result), len(claudeAllowed))
	return result
}

func isExplicitlyDisabledTool(tool any) bool {
	enabled, ok := tool.(bool)
	return ok && !enabled
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
//
// System MCP servers (safeoutputs, mcpscripts, agenticworkflows) are not present in the
// user-visible tools map but must be explicitly added to --allowed-tools when
// --permission-mode acceptEdits is in use, because acceptEdits actually enforces the
// allowlist (unlike bypassPermissions which silently ignores it).
// Panics if callers pass a Claude-specific tools section instead of neutral tools.
func (e *ClaudeEngine) computeAllowedClaudeToolsString(tools map[string]any, safeOutputs *SafeOutputsConfig, cacheMemoryConfig *CacheMemoryConfig, mcpScripts *MCPScriptsConfig, sandboxConfig *SandboxConfig) string {
	claudeToolsLog.Print("Computing allowed Claude tools string")

	tools = normalizeClaudeToolsInput(tools)
	claudeToolsLog.Print("Converting neutral tools to Claude-specific format")
	tools = e.expandNeutralToolsToClaudeTools(tools)
	addDefaultClaudeAllowedTools(tools)
	claudeToolsLog.Printf("Added %d default Claude tools to allowed list", len(defaultClaudeToolNames))

	allowedTools := collectClaudeAllowedTools(tools)
	allowedTools = appendTopLevelClaudeTools(allowedTools, tools, cacheMemoryConfig)
	allowedTools = appendSandboxWritableClaudeTools(allowedTools, sandboxConfig)
	allowedTools = appendSafeOutputsClaudeTools(allowedTools, safeOutputs)
	allowedTools = appendMCPScriptsClaudeTools(allowedTools, mcpScripts)
	allowedTools = dedupeClaudeAllowedTools(allowedTools)
	sort.Strings(allowedTools)

	claudeToolsLog.Printf("Generated allowed tools string with %d tools", len(allowedTools))
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

	// Pre-size the builder using the exact output size:
	//   - header line:  indent + "# Allowed tools (sorted):\n"
	//   - per tool:     indent + "# - " + toolName + "\n"
	// allowedToolsStr is comma-separated, so subtracting (len(tools)-1) gives the
	// total bytes contributed by tool names alone.
	toolNameBytes := len(allowedToolsStr) - (len(tools) - 1)
	var comment strings.Builder
	comment.Grow(
		len(indent) +
			len("# Allowed tools (sorted):\n") +
			len(tools)*len(indent) +
			len(tools)*len("# - \n") +
			toolNameBytes,
	)
	comment.WriteString(indent)
	comment.WriteString("# Allowed tools (sorted):\n")
	for _, tool := range tools {
		comment.WriteString(indent)
		comment.WriteString("# - ")
		comment.WriteString(tool)
		comment.WriteByte('\n')
	}

	return comment.String()
}
