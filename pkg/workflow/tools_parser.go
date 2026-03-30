// This file provides tool configuration parsing for agentic workflows.
//
// This file handles parsing of tool configurations from the frontmatter tools section.
// It extracts and validates tool configurations for all supported tools, converting
// YAML-parsed maps into strongly-typed Go structs.
//
// # Organization Rationale
//
// All tool parsing functions are grouped in this file because they:
//   - Share a common purpose (tool configuration parsing)
//   - Follow similar parsing patterns (map[string]any -> struct)
//   - Are called together during workflow compilation
//   - Provide a single source of truth for tool configuration
//
// This follows established patterns where domain-specific parsing is grouped by
// functionality rather than scattered across files. See skills/developer/SKILL.md
// for code organization principles.
//
// # Supported Tools
//
// Built-in Tools:
//   - github: GitHub API and repository operations
//   - bash: Shell command execution
//   - web-fetch: HTTP content fetching
//   - web-search: Web search capabilities
//   - edit: File editing operations
//   - playwright: Browser automation
//   - agentic-workflows: Nested workflow execution
//   - cache-memory: In-workflow memory caching
//   - repo-memory: Repository-backed persistent memory
//
// Configuration Tools:
//   - safety-prompt: Safety prompt injection
//   - timeout: Agent timeout configuration
//   - startup-timeout: Agent startup timeout
//
// Custom Tools:
//   - MCP servers and other custom tool configurations
//
// # Parse Function Pattern
//
// Each parse function follows the pattern:
//  1. Accept any type to handle various YAML representations
//  2. Type-assert to expected structure (bool, string, map, array)
//  3. Extract and validate configuration values
//  4. Return strongly-typed configuration struct
//
// This provides type safety while accommodating flexible YAML syntax.

package workflow

import (
	"fmt"
	"maps"
	"os"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
)

var toolsParserLog = logger.New("workflow:tools_parser")

// parseCommaSeparatedOrNewlineList splits a string by commas and/or newlines,
// trims surrounding whitespace from each item, and discards empty items.
func parseCommaSeparatedOrNewlineList(s string) []string {
	// Normalize newlines to commas, then split on comma.
	normalized := strings.ReplaceAll(s, "\n", ",")
	parts := strings.Split(normalized, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// toAnySlice converts a []string to []any for storage in a map[string]any.
func toAnySlice(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

// NewTools creates a new Tools instance from a map
func NewTools(toolsMap map[string]any) *Tools {
	toolsParserLog.Printf("Creating tools configuration from map with %d entries", len(toolsMap))
	if toolsMap == nil {
		return &Tools{
			Custom: make(map[string]MCPServerConfig),
			raw:    make(map[string]any),
		}
	}

	tools := &Tools{
		Custom: make(map[string]MCPServerConfig),
		raw:    make(map[string]any),
	}

	// Copy raw map
	maps.Copy(tools.raw, toolsMap)

	// Extract and parse known tools
	if val, exists := toolsMap["github"]; exists {
		tools.GitHub = parseGitHubTool(val)
	}
	if val, exists := toolsMap["bash"]; exists {
		tools.Bash = parseBashTool(val)
		// Check if parsing returned nil - this indicates invalid configuration
		if tools.Bash == nil {
			toolsParserLog.Print("Warning: bash tool configuration is invalid (nil/anonymous syntax not supported)")
		}
	}
	if val, exists := toolsMap["web-fetch"]; exists {
		tools.WebFetch = parseWebFetchTool(val)
	}
	if val, exists := toolsMap["web-search"]; exists {
		tools.WebSearch = parseWebSearchTool(val)
	}
	if val, exists := toolsMap["edit"]; exists {
		tools.Edit = parseEditTool(val)
	}
	if val, exists := toolsMap["playwright"]; exists {
		tools.Playwright = parsePlaywrightTool(val)
	}
	if val, exists := toolsMap["qmd"]; exists {
		tools.Qmd = parseQmdTool(val)
	}
	if val, exists := toolsMap["agentic-workflows"]; exists {
		tools.AgenticWorkflows = parseAgenticWorkflowsTool(val)
	}
	if val, exists := toolsMap["cache-memory"]; exists {
		tools.CacheMemory = parseCacheMemoryTool(val)
	}
	if val, exists := toolsMap["repo-memory"]; exists {
		tools.RepoMemory = parseRepoMemoryTool(val)
	}
	if val, exists := toolsMap["timeout"]; exists {
		tools.Timeout = parseTimeoutTool(val)
	}
	if val, exists := toolsMap["startup-timeout"]; exists {
		tools.StartupTimeout = parseStartupTimeoutTool(val)
	}

	// Extract custom MCP tools (anything not in the known list)
	knownTools := map[string]bool{
		"github":            true,
		"bash":              true,
		"web-fetch":         true,
		"web-search":        true,
		"edit":              true,
		"playwright":        true,
		"qmd":               true,
		"agentic-workflows": true,
		"cache-memory":      true,
		"repo-memory":       true,
		"safety-prompt":     true,
		"timeout":           true,
		"startup-timeout":   true,
	}

	customCount := 0
	for name, config := range toolsMap {
		if !knownTools[name] {
			tools.Custom[name] = parseMCPServerConfig(config)
			customCount++
		}
	}

	toolsParserLog.Printf("Parsed tools: github=%v, bash=%v, playwright=%v, qmd=%v, custom=%d", tools.GitHub != nil, tools.Bash != nil, tools.Playwright != nil, tools.Qmd != nil, customCount)
	return tools
}

// parseGitHubTool converts raw github tool configuration to GitHubToolConfig
func parseGitHubTool(val any) *GitHubToolConfig {
	if val == nil {
		toolsParserLog.Print("GitHub tool enabled with default configuration")
		return &GitHubToolConfig{
			ReadOnly: true, // default to read-only for security
		}
	}

	// Handle string type (simple enable)
	if _, ok := val.(string); ok {
		toolsParserLog.Print("GitHub tool enabled with string configuration")
		return &GitHubToolConfig{
			ReadOnly: true, // default to read-only for security
		}
	}

	// Handle map type (detailed configuration)
	if configMap, ok := val.(map[string]any); ok {
		toolsParserLog.Print("Parsing GitHub tool detailed configuration")
		config := &GitHubToolConfig{
			ReadOnly: true, // default to read-only for security
		}

		if allowed, ok := configMap["allowed"].([]any); ok {
			config.Allowed = make(GitHubAllowedTools, 0, len(allowed))
			for _, item := range allowed {
				if str, ok := item.(string); ok {
					config.Allowed = append(config.Allowed, GitHubToolName(str))
				}
			}
		}

		if mode, ok := configMap["mode"].(string); ok {
			config.Mode = mode
		}

		if version, ok := configMap["version"].(string); ok {
			config.Version = version
		}

		if args, ok := configMap["args"].([]any); ok {
			config.Args = make([]string, 0, len(args))
			for _, item := range args {
				if str, ok := item.(string); ok {
					config.Args = append(config.Args, str)
				}
			}
		}

		if readOnly, ok := configMap["read-only"].(bool); ok {
			config.ReadOnly = readOnly
		}
		// else: defaults to true (set above)

		if token, ok := configMap["github-token"].(string); ok {
			config.GitHubToken = token
		}

		// Check for both "toolset" and "toolsets" (plural is more common in user configs)
		if toolset, ok := configMap["toolsets"].([]any); ok {
			config.Toolset = make(GitHubToolsets, 0, len(toolset))
			for _, item := range toolset {
				if str, ok := item.(string); ok {
					config.Toolset = append(config.Toolset, GitHubToolset(str))
				}
			}
		} else if toolset, ok := configMap["toolset"].([]any); ok {
			config.Toolset = make(GitHubToolsets, 0, len(toolset))
			for _, item := range toolset {
				if str, ok := item.(string); ok {
					config.Toolset = append(config.Toolset, GitHubToolset(str))
				}
			}
		}

		if lockdown, ok := configMap["lockdown"].(bool); ok {
			config.Lockdown = lockdown
		}

		// Parse app configuration for GitHub App token minting
		if rawApp, exists := configMap["github-app"]; exists {
			if appMap, ok := rawApp.(map[string]any); ok {
				config.GitHubApp = parseAppConfig(appMap)
			}
		}

		// Parse guard policy fields (flat syntax: allowed-repos/repos and min-integrity directly under github:)
		if allowedRepos, ok := configMap["allowed-repos"]; ok {
			config.AllowedRepos = allowedRepos // Store as-is, validation will happen later
		} else if repos, ok := configMap["repos"]; ok {
			// Deprecated: use 'allowed-repos' instead of 'repos'
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("'tools.github.repos' is deprecated. Use 'tools.github.allowed-repos' instead. Run 'gh aw fix' to automatically migrate."))
			config.AllowedRepos = repos // Populate canonical field for validation
		}
		if integrity, ok := configMap["min-integrity"].(string); ok {
			config.MinIntegrity = GitHubIntegrityLevel(integrity)
		}
		if blockedUsers, ok := configMap["blocked-users"].([]any); ok {
			config.BlockedUsers = make([]string, 0, len(blockedUsers))
			for _, item := range blockedUsers {
				if str, ok := item.(string); ok {
					config.BlockedUsers = append(config.BlockedUsers, str)
				}
			}
		} else if blockedUsers, ok := configMap["blocked-users"].([]string); ok {
			config.BlockedUsers = blockedUsers
		} else if blockedUsersStr, ok := configMap["blocked-users"].(string); ok {
			if isGitHubActionsExpression(blockedUsersStr) {
				// GitHub Actions expression: store as-is; raw map retains the string for JSON rendering.
				config.BlockedUsersExpr = blockedUsersStr
			} else {
				// Static comma/newline-separated string: parse at compile time.
				parsed := parseCommaSeparatedOrNewlineList(blockedUsersStr)
				config.BlockedUsers = parsed
				configMap["blocked-users"] = toAnySlice(parsed) // normalize raw map for JSON rendering
			}
		}
		if approvalLabels, ok := configMap["approval-labels"].([]any); ok {
			config.ApprovalLabels = make([]string, 0, len(approvalLabels))
			for _, item := range approvalLabels {
				if str, ok := item.(string); ok {
					config.ApprovalLabels = append(config.ApprovalLabels, str)
				}
			}
		} else if approvalLabels, ok := configMap["approval-labels"].([]string); ok {
			config.ApprovalLabels = approvalLabels
		} else if approvalLabelsStr, ok := configMap["approval-labels"].(string); ok {
			if isGitHubActionsExpression(approvalLabelsStr) {
				// GitHub Actions expression: store as-is; raw map retains the string for JSON rendering.
				config.ApprovalLabelsExpr = approvalLabelsStr
			} else {
				// Static comma/newline-separated string: parse at compile time.
				parsed := parseCommaSeparatedOrNewlineList(approvalLabelsStr)
				config.ApprovalLabels = parsed
				configMap["approval-labels"] = toAnySlice(parsed) // normalize raw map for JSON rendering
			}
		}

		return config
	}

	return &GitHubToolConfig{
		ReadOnly: true, // default to read-only for security
	}
}

// parseBashTool converts raw bash tool configuration to BashToolConfig
func parseBashTool(val any) *BashToolConfig {
	if val == nil {
		// nil is no longer supported - return nil to indicate invalid configuration
		// The compiler will handle this as a validation error
		toolsParserLog.Print("Bash tool configured with nil value (unsupported)")
		return nil
	}

	// Handle boolean values
	if boolVal, ok := val.(bool); ok {
		if boolVal {
			// bash: true means all commands allowed
			toolsParserLog.Print("Bash tool enabled with all commands allowed")
			return &BashToolConfig{}
		}
		// bash: false means explicitly disabled
		toolsParserLog.Print("Bash tool explicitly disabled")
		return &BashToolConfig{
			AllowedCommands: []string{}, // Empty slice indicates explicitly disabled
		}
	}

	// Handle array of allowed commands
	if cmdArray, ok := val.([]any); ok {
		config := &BashToolConfig{
			AllowedCommands: make([]string, 0, len(cmdArray)),
		}
		for _, item := range cmdArray {
			if str, ok := item.(string); ok {
				config.AllowedCommands = append(config.AllowedCommands, str)
			}
		}
		return config
	}

	// Invalid configuration
	return nil
}

// parsePlaywrightTool converts raw playwright tool configuration to PlaywrightToolConfig
func parsePlaywrightTool(val any) *PlaywrightToolConfig {
	if val == nil {
		toolsParserLog.Print("Playwright tool enabled with default configuration")
		return &PlaywrightToolConfig{}
	}
	toolsParserLog.Print("Parsing playwright tool configuration")

	if configMap, ok := val.(map[string]any); ok {
		config := &PlaywrightToolConfig{}

		// Handle version field - can be string or number
		if version, ok := configMap["version"].(string); ok {
			config.Version = version
		} else if versionNum, ok := configMap["version"].(int); ok {
			config.Version = strconv.Itoa(versionNum)
		} else if versionNum, ok := configMap["version"].(int64); ok {
			config.Version = strconv.FormatInt(versionNum, 10)
		} else if versionNum, ok := configMap["version"].(float64); ok {
			config.Version = fmt.Sprintf("%g", versionNum)
		}

		// Handle args field - can be []any or []string
		if argsValue, ok := configMap["args"]; ok {
			if arr, ok := argsValue.([]any); ok {
				config.Args = make([]string, 0, len(arr))
				for _, item := range arr {
					if str, ok := item.(string); ok {
						config.Args = append(config.Args, str)
					}
				}
			} else if arr, ok := argsValue.([]string); ok {
				config.Args = arr
			}
		}

		return config
	}

	return &PlaywrightToolConfig{}
}

// isUnexpandedImportInput reports whether s is an unexpanded import-schema placeholder
// of the form "${{ github.aw.import-inputs.<key> }}" that was left as a literal string
// because the caller did not supply the optional input. It returns false for any other
// value, including legitimate GitHub Actions expressions like "${{ hashFiles('...') }}".
func isUnexpandedImportInput(s string) bool {
	return strings.Contains(s, "github.aw.import-inputs.")
}

// parseQmdTool converts raw qmd tool configuration to QmdToolConfig.
// Supported fields:
//
//   - checkouts: list of named collections (with optional checkout per entry)
//   - searches:  list of GitHub search queries
//   - cache-key: optional GitHub Actions cache key
//   - gpu:       enable GPU acceleration for node-llama-cpp (default: false)
//   - runs-on:   override runner image for the indexing job
func parseQmdTool(val any) *QmdToolConfig {
	if val == nil {
		toolsParserLog.Print("qmd tool enabled with empty configuration")
		return &QmdToolConfig{}
	}

	if configMap, ok := val.(map[string]any); ok {
		config := &QmdToolConfig{}

		// Handle cache-key field. Skip values that are unexpanded import-schema placeholders
		// (exactly "${{ github.aw.import-inputs.cache-key }}") left as literal strings when
		// the caller does not supply the optional input. Legitimate GitHub Actions expressions
		// such as "qmd-${{ hashFiles('docs/**') }}" are kept as-is.
		if cacheKey, ok := configMap["cache-key"].(string); ok && cacheKey != "" && !isUnexpandedImportInput(cacheKey) {
			config.CacheKey = cacheKey
			toolsParserLog.Printf("qmd tool cache-key: %s", cacheKey)
		}

		// Handle gpu field (defaults to false — GPU disabled by default)
		if gpuVal, exists := configMap["gpu"]; exists {
			if gpuBool, ok := gpuVal.(bool); ok {
				config.GPU = gpuBool
				toolsParserLog.Printf("qmd tool gpu: %v", gpuBool)
			}
		}

		// Handle runs-on field (override runner image for the indexing job). Skip values that
		// are unexpanded import-schema placeholders. Legitimate GitHub Actions expressions are
		// kept as-is.
		if runsOnVal, exists := configMap["runs-on"]; exists {
			if runsOnStr, ok := runsOnVal.(string); ok && runsOnStr != "" && !isUnexpandedImportInput(runsOnStr) {
				config.RunsOn = runsOnStr
				toolsParserLog.Printf("qmd tool runs-on: %s", runsOnStr)
			}
		}

		// Handle checkouts field
		if checkoutsValue, ok := configMap["checkouts"]; ok {
			if arr, ok := checkoutsValue.([]any); ok {
				config.Checkouts = make([]*QmdDocCollection, 0, len(arr))
				for i, item := range arr {
					itemMap, ok := item.(map[string]any)
					if !ok {
						continue
					}
					col := parseQmdDocCollection(itemMap, i)
					config.Checkouts = append(config.Checkouts, col)
				}
				toolsParserLog.Printf("qmd tool parsed %d checkouts", len(config.Checkouts))
			}
		}

		// Handle searches field
		if searchesValue, ok := configMap["searches"]; ok {
			if arr, ok := searchesValue.([]any); ok {
				config.Searches = make([]*QmdSearchEntry, 0, len(arr))
				for _, item := range arr {
					itemMap, ok := item.(map[string]any)
					if !ok {
						continue
					}
					entry := parseQmdSearchEntry(itemMap)
					config.Searches = append(config.Searches, entry)
				}
				toolsParserLog.Printf("qmd tool parsed %d searches", len(config.Searches))
			}
		}

		return config
	}

	return &QmdToolConfig{}
}

// parseQmdDocCollection converts a raw map to a QmdDocCollection.
// The index parameter is used to generate a default name when none is provided.
func parseQmdDocCollection(m map[string]any, index int) *QmdDocCollection {
	col := &QmdDocCollection{}

	if name, ok := m["name"].(string); ok && name != "" {
		col.Name = name
	} else {
		col.Name = fmt.Sprintf("docs-%d", index)
	}

	if pattern, ok := m["pattern"].(string); ok {
		col.Pattern = pattern
	}

	if ignoreValue, ok := m["ignore"]; ok {
		if arr, ok := ignoreValue.([]any); ok {
			col.Ignore = make([]string, 0, len(arr))
			for _, item := range arr {
				if str, ok := item.(string); ok {
					col.Ignore = append(col.Ignore, str)
				}
			}
		} else if arr, ok := ignoreValue.([]string); ok {
			col.Ignore = arr
		}
	}

	if context, ok := m["context"].(string); ok {
		col.Context = context
	}

	if checkoutValue, ok := m["checkout"]; ok {
		if checkoutMap, ok := checkoutValue.(map[string]any); ok {
			if cfg, err := checkoutConfigFromMap(checkoutMap); err == nil {
				col.Checkout = cfg
			} else {
				toolsParserLog.Printf("qmd collection %q: ignoring invalid checkout config: %v", col.Name, err)
			}
		}
	}

	return col
}

// parseQmdSearchEntry converts a raw map to a QmdSearchEntry.
func parseQmdSearchEntry(m map[string]any) *QmdSearchEntry {
	entry := &QmdSearchEntry{}

	if n, ok := m["name"].(string); ok {
		entry.Name = n
	}
	if t, ok := m["type"].(string); ok {
		entry.Type = t
	}
	if q, ok := m["query"].(string); ok {
		entry.Query = q
	}
	entry.Min = parseYAMLInt(m["min"])
	entry.Max = parseYAMLInt(m["max"])

	if token, ok := m["github-token"].(string); ok {
		entry.GitHubToken = token
	}

	if appMap, ok := m["github-app"].(map[string]any); ok {
		entry.GitHubApp = parseAppConfig(appMap)
	}

	return entry
}

// parseYAMLInt converts a YAML-unmarshaled numeric value to int.
// goccy/go-yaml unmarshals integers as uint64; standard yaml/v3 uses int.
// float64 is also handled for completeness.
func parseYAMLInt(v any) int {
	if v == nil {
		return 0
	}
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case uint64:
		return int(n)
	case float64:
		return int(n)
	}
	return 0
}

// parseWebFetchTool converts raw web-fetch tool configuration
func parseWebFetchTool(val any) *WebFetchToolConfig {
	// web-fetch is either nil or an empty object
	return &WebFetchToolConfig{}
}

// parseWebSearchTool converts raw web-search tool configuration
func parseWebSearchTool(val any) *WebSearchToolConfig {
	// web-search is either nil or an empty object
	return &WebSearchToolConfig{}
}

// parseEditTool converts raw edit tool configuration
func parseEditTool(val any) *EditToolConfig {
	// edit is either nil or an empty object
	return &EditToolConfig{}
}

// parseAgenticWorkflowsTool converts raw agentic-workflows tool configuration
func parseAgenticWorkflowsTool(val any) *AgenticWorkflowsToolConfig {
	config := &AgenticWorkflowsToolConfig{}

	if boolVal, ok := val.(bool); ok {
		config.Enabled = boolVal
	} else if val == nil {
		config.Enabled = true // nil means enabled
	}

	return config
}

// parseCacheMemoryTool converts raw cache-memory tool configuration
func parseCacheMemoryTool(val any) *CacheMemoryToolConfig {
	// cache-memory can be boolean, object, or array - store raw value
	return &CacheMemoryToolConfig{Raw: val}
}

// parseRepoMemoryTool converts raw repo-memory tool configuration
func parseRepoMemoryTool(val any) *RepoMemoryToolConfig {
	// repo-memory can be boolean, object, or array - store raw value
	return &RepoMemoryToolConfig{Raw: val}
}

// parseTimeoutTool converts raw timeout tool configuration
func parseTimeoutTool(val any) *int {
	if intVal, ok := val.(int); ok {
		return &intVal
	}
	if floatVal, ok := val.(float64); ok {
		intVal := int(floatVal)
		return &intVal
	}
	return nil
}

// parseStartupTimeoutTool converts raw startup-timeout tool configuration
func parseStartupTimeoutTool(val any) *int {
	if intVal, ok := val.(int); ok {
		return &intVal
	}
	if floatVal, ok := val.(float64); ok {
		intVal := int(floatVal)
		return &intVal
	}
	return nil
}

// parseMCPServerConfig converts raw MCP server configuration to MCPServerConfig
func parseMCPServerConfig(val any) MCPServerConfig {
	config := MCPServerConfig{
		CustomFields: make(map[string]any),
	}

	// If val is nil, return empty config
	if val == nil {
		return config
	}

	// If it's not a map, store it as a custom field
	configMap, ok := val.(map[string]any)
	if !ok {
		config.CustomFields["value"] = val
		return config
	}

	// Parse common MCP server fields
	if command, ok := configMap["command"].(string); ok {
		config.Command = command
	}

	if args, ok := configMap["args"].([]any); ok {
		config.Args = make([]string, 0, len(args))
		for _, arg := range args {
			if str, ok := arg.(string); ok {
				config.Args = append(config.Args, str)
			}
		}
	}

	if env, ok := configMap["env"].(map[string]any); ok {
		config.Env = make(map[string]string)
		for k, v := range env {
			if str, ok := v.(string); ok {
				config.Env[k] = str
			}
		}
	}

	if mode, ok := configMap["mode"].(string); ok {
		config.Mode = mode
	}

	if mcpType, ok := configMap["type"].(string); ok {
		config.Type = mcpType
	}

	if version, ok := configMap["version"].(string); ok {
		config.Version = version
	} else if versionNum, ok := configMap["version"].(float64); ok {
		config.Version = fmt.Sprintf("%.0f", versionNum)
	}

	if toolsets, ok := configMap["toolsets"].([]any); ok {
		config.Toolsets = make([]string, 0, len(toolsets))
		for _, item := range toolsets {
			if str, ok := item.(string); ok {
				config.Toolsets = append(config.Toolsets, str)
			}
		}
	}

	// Parse HTTP-specific fields
	if url, ok := configMap["url"].(string); ok {
		config.URL = url
	}

	if headers, ok := configMap["headers"].(map[string]any); ok {
		config.Headers = make(map[string]string)
		for k, v := range headers {
			if str, ok := v.(string); ok {
				config.Headers[k] = str
			}
		}
	}

	// Parse container-specific fields
	if container, ok := configMap["container"].(string); ok {
		config.Container = container
	}

	if entrypoint, ok := configMap["entrypoint"].(string); ok {
		config.Entrypoint = entrypoint
	}

	if entrypointArgs, ok := configMap["entrypointArgs"].([]any); ok {
		config.EntrypointArgs = make([]string, 0, len(entrypointArgs))
		for _, arg := range entrypointArgs {
			if str, ok := arg.(string); ok {
				config.EntrypointArgs = append(config.EntrypointArgs, str)
			}
		}
	}

	if mounts, ok := configMap["mounts"].([]any); ok {
		config.Mounts = make([]string, 0, len(mounts))
		for _, mount := range mounts {
			if str, ok := mount.(string); ok {
				config.Mounts = append(config.Mounts, str)
			}
		}
	}

	// Store any unknown fields in CustomFields
	knownFields := map[string]bool{
		"command":        true,
		"args":           true,
		"env":            true,
		"mode":           true,
		"type":           true,
		"version":        true,
		"toolsets":       true,
		"url":            true,
		"headers":        true,
		"container":      true,
		"entrypoint":     true,
		"entrypointArgs": true,
		"mounts":         true,
	}

	for key, value := range configMap {
		if !knownFields[key] {
			config.CustomFields[key] = value
		}
	}

	return config
}
