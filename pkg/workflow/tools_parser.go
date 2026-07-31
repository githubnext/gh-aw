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
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/setutil"
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
// knownTools is the set of built-in tool names that NewTools handles explicitly.
// It is a package-level variable to avoid re-allocating this map on every call.
var knownTools = map[string]struct{}{
	"github":            {},
	"bash":              {},
	"web-fetch":         {},
	"web-search":        {},
	"edit":              {},
	"playwright":        {},
	"agentic-workflows": {},
	"cache-memory":      {},
	"comment-memory":    {},
	"repo-memory":       {},
	"safety-prompt":     {},
	"timeout":           {},
	"startup-timeout":   {},
	"cli-proxy":         {},
}

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

	parseStandardTools(toolsMap, tools)
	customCount := parseCustomTools(toolsMap, tools)

	toolsParserLog.Printf("Parsed tools: github=%v, bash=%v, playwright=%v, custom=%d", tools.GitHub != nil, tools.Bash != nil, tools.Playwright != nil, customCount)
	return tools
}

func parseStandardTools(toolsMap map[string]any, tools *Tools) {
	if val, exists := toolsMap["github"]; exists {
		tools.GitHub = parseGitHubTool(val)
	}
	if val, exists := toolsMap["bash"]; exists {
		tools.Bash = parseBashTool(val)
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
	if val, exists := toolsMap["agentic-workflows"]; exists {
		tools.AgenticWorkflows = parseAgenticWorkflowsTool(val)
	}
	if val, exists := toolsMap["cache-memory"]; exists {
		tools.CacheMemory = parseCacheMemoryTool(val)
	}
	if val, exists := toolsMap["comment-memory"]; exists {
		tools.CommentMemory = parseCommentMemoryTool(val)
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
	parseCLIProxyTool(toolsMap, tools)
}

func parseCLIProxyTool(toolsMap map[string]any, tools *Tools) {
	val, exists := toolsMap["cli-proxy"]
	if !exists {
		return
	}
	if b, ok := val.(bool); ok {
		tools.CLIProxy = b
		return
	}
	toolsParserLog.Printf("Warning: cli-proxy must be a boolean (true/false), ignoring value: %v", val)
}

func parseCustomTools(toolsMap map[string]any, tools *Tools) int {
	customCount := 0
	for name, config := range toolsMap {
		if setutil.Contains(knownTools, name) {
			continue
		}
		tools.Custom[name] = parseMCPServerConfig(config)
		customCount++
	}
	return customCount
}

// parseGitHubTool converts raw github tool configuration to GitHubToolConfig
func parseGitHubTool(val any) *GitHubToolConfig {
	if val == nil {
		toolsParserLog.Print("GitHub tool enabled with default configuration")
		return newDefaultGitHubToolConfig()
	}

	// Handle string type (simple enable)
	if _, ok := val.(string); ok {
		toolsParserLog.Print("GitHub tool enabled with string configuration")
		return newDefaultGitHubToolConfig()
	}

	configMap, ok := val.(map[string]any)
	if !ok {
		return newDefaultGitHubToolConfig()
	}

	return parseGitHubToolMap(configMap)
}

func newDefaultGitHubToolConfig() *GitHubToolConfig {
	return &GitHubToolConfig{
		ReadOnly: true, // default to read-only for security
	}
}

func parseGitHubToolMap(configMap map[string]any) *GitHubToolConfig {
	toolsParserLog.Print("Parsing GitHub tool detailed configuration")
	config := newDefaultGitHubToolConfig()
	parseGitHubToolAllowed(configMap, config)
	parseGitHubToolBasicFields(configMap, config)
	parseGitHubToolGuardFields(configMap, config)
	return config
}

func parseGitHubToolAllowed(configMap map[string]any, config *GitHubToolConfig) {
	allowedSetting, ok := configMap["allowed"]
	if !ok {
		return
	}
	allowedTools, _ := parseGitHubAllowedToolsAndLimits(allowedSetting)
	config.Allowed = make(GitHubAllowedTools, 0, len(allowedTools))
	for _, toolName := range allowedTools {
		config.Allowed = append(config.Allowed, GitHubToolName(toolName))
	}
}

func parseGitHubToolBasicFields(configMap map[string]any, config *GitHubToolConfig) {
	if mode, ok := configMap["mode"].(string); ok {
		config.Mode = GitHubMCPMode(mode)
	}
	if mcpType, ok := configMap["type"].(string); ok {
		config.Type = mcpType
	}
	if version, ok := configMap["version"].(string); ok {
		config.Version = version
	}
	if args, ok := configMap["args"].([]any); ok {
		config.Args = parseStringSlice(args)
	}
	if readOnly, ok := configMap["read-only"].(bool); ok {
		config.ReadOnly = readOnly
	}
	if token, ok := configMap["github-token"].(string); ok {
		config.GitHubToken = token
	}
	if toolsets, ok := parseGitHubToolsets(configMap); ok {
		config.Toolset = toolsets
	}
	if lockdown, ok := configMap["lockdown"].(bool); ok {
		config.Lockdown = lockdown
	}
	if rawApp, exists := configMap["github-app"]; exists {
		if appMap, ok := rawApp.(map[string]any); ok {
			config.GitHubApp = parseAppConfig(appMap)
		}
	}
}

func parseGitHubToolGuardFields(configMap map[string]any, config *GitHubToolConfig) {
	parseGitHubRepoGuardFields(configMap, config)
	parseGitHubUserAndLabelGuardFields(configMap, config)
	parseGitHubReactionGuardFields(configMap, config)
	parseGitHubPrivateToPublicFlows(configMap, config)
	parseGitHubBoundedQueries(configMap, config)
}

func parseGitHubRepoGuardFields(configMap map[string]any, config *GitHubToolConfig) {
	if allowedRepos, ok := configMap["allowed-repos"]; ok {
		config.AllowedRepos = allowedRepos
	} else if repos, ok := configMap["repos"]; ok {
		config.AllowedRepos = repos
	}
	if integrity, ok := configMap["min-integrity"].(string); ok {
		config.MinIntegrity = GitHubIntegrityLevel(integrity)
	}
}

func parseGitHubUserAndLabelGuardFields(configMap map[string]any, config *GitHubToolConfig) {
	parseListOrExpressionField(configMap, "blocked-users", &config.BlockedUsers, &config.BlockedUsersExpr)
	parseListOrExpressionField(configMap, "approval-labels", &config.ApprovalLabels, &config.ApprovalLabelsExpr)
	parseListOrExpressionField(configMap, "trusted-users", &config.TrustedUsers, &config.TrustedUsersExpr)
}

func parseListOrExpressionField(configMap map[string]any, key string, values *[]string, expr *string) {
	switch raw := configMap[key].(type) {
	case []any:
		*values = parseStringSlice(raw)
	case []string:
		*values = raw
	case string:
		if hasExpressionMarker(raw) {
			*expr = raw
			return
		}
		parsed := parseCommaSeparatedOrNewlineList(raw)
		*values = parsed
		configMap[key] = toAnySlice(parsed)
	}
}

func parseGitHubReactionGuardFields(configMap map[string]any, config *GitHubToolConfig) {
	config.EndorsementReactions = parseOptionalStringList(configMap["endorsement-reactions"])
	config.DisapprovalReactions = parseOptionalStringList(configMap["disapproval-reactions"])
	if disapprovalIntegrity, ok := configMap["disapproval-integrity"].(string); ok {
		config.DisapprovalIntegrity = disapprovalIntegrity
	}
	if endorserMinIntegrity, ok := configMap["endorser-min-integrity"].(string); ok {
		config.EndorserMinIntegrity = endorserMinIntegrity
	}
}

func parseGitHubPrivateToPublicFlows(configMap map[string]any, config *GitHubToolConfig) {
	rawPtP, ok := configMap["private-to-public-flows"]
	if !ok {
		return
	}
	switch v := rawPtP.(type) {
	case string:
		config.PrivateToPublicFlows = v
	case []any:
		config.PrivateToPublicFlows = parseStringSlice(v)
	case []string:
		config.PrivateToPublicFlows = v
	default:
		toolsParserLog.Printf("Warning: private-to-public-flows has unsupported type %T (expected string \"allow\" or array of server IDs), ignoring", rawPtP)
	}
}

func parseGitHubBoundedQueries(configMap map[string]any, config *GitHubToolConfig) {
	rawBQ, ok := configMap["bounded-queries"]
	if !ok {
		return
	}
	if bqMap, ok := rawBQ.(map[string]any); ok {
		config.BoundedQueries = parseBoundedQueriesConfig(bqMap)
		return
	}
	config.BoundedQueries = &BoundedQueriesConfig{
		ParseError: fmt.Sprintf("bounded-queries must be a mapping object, got %T", rawBQ),
	}
}

func parseGitHubToolsets(configMap map[string]any) (GitHubToolsets, bool) {
	if toolsets, ok := parseGitHubToolsetField(configMap, "toolsets"); ok {
		return toolsets, true
	}
	return parseGitHubToolsetField(configMap, "toolset")
}

func parseGitHubToolsetField(configMap map[string]any, key string) (GitHubToolsets, bool) {
	switch toolset := configMap[key].(type) {
	case []any:
		return toGitHubToolsets(toolset), true
	case string:
		configMap[key] = []any{toolset}
		return GitHubToolsets{GitHubToolset(toolset)}, true
	default:
		return nil, false
	}
}

func toGitHubToolsets(values []any) GitHubToolsets {
	toolsets := make(GitHubToolsets, 0, len(values))
	for _, item := range values {
		if str, ok := item.(string); ok {
			toolsets = append(toolsets, GitHubToolset(str))
		}
	}
	return toolsets
}

func parseOptionalStringList(val any) []string {
	switch values := val.(type) {
	case []any:
		return parseStringSlice(values)
	case []string:
		return values
	default:
		return nil
	}
}

func parseStringSlice(values []any) []string {
	parsed := make([]string, 0, len(values))
	for _, item := range values {
		if str, ok := item.(string); ok {
			parsed = append(parsed, str)
		}
	}
	return parsed
}

// parseBoundedQueriesConfig converts a raw map into a BoundedQueriesConfig.
func parseBoundedQueriesConfig(bqMap map[string]any) *BoundedQueriesConfig {
	config := &BoundedQueriesConfig{}

	if rawRepos, ok := bqMap["private-repos"]; ok {
		switch repos := rawRepos.(type) {
		case []any:
			config.PrivateRepos = make([]*BoundedQueryPrivateRepo, 0, len(repos))
			for i, item := range repos {
				if repoMap, ok := item.(map[string]any); ok {
					entry := &BoundedQueryPrivateRepo{}
					if repo, ok := repoMap["repo"].(string); ok {
						entry.Repo = repo
					}
					if sensitivity, ok := repoMap["sensitivity"].(string); ok {
						entry.Sensitivity = sensitivity
					}
					config.PrivateRepos = append(config.PrivateRepos, entry)
				} else {
					config.ParseError = fmt.Sprintf("private-repos[%d] must be a mapping object, got %T", i, item)
					return config
				}
			}
		default:
			config.ParseError = fmt.Sprintf("private-repos must be an array, got %T", rawRepos)
			return config
		}
	}

	if runtime, ok := bqMap["runtime"].(string); ok {
		config.Runtime = runtime
	}
	if rawTimeout, hasTimeout := bqMap["timeout"]; hasTimeout {
		if timeout, ok := rawTimeout.(int); ok {
			config.Timeout = &timeout
		} else {
			config.ParseError = fmt.Sprintf("timeout must be an integer, got %T", rawTimeout)
			return config
		}
	}
	if memoryLimit, ok := bqMap["memory-limit"].(string); ok {
		config.MemoryLimit = memoryLimit
	}
	if interpreter, ok := bqMap["interpreter"].(string); ok {
		config.Interpreter = interpreter
	}
	if rawMax, hasMax := bqMap["max-invocations"]; hasMax {
		if maxInvocations, ok := rawMax.(int); ok {
			config.MaxInvocations = &maxInvocations
		} else {
			config.ParseError = fmt.Sprintf("max-invocations must be an integer, got %T", rawMax)
			return config
		}
	}

	return config
}
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

		// Handle mode field
		if mode, ok := configMap["mode"].(string); ok {
			config.Mode = mode
		}

		return config
	}

	return &PlaywrightToolConfig{}
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
	if boolVal, ok := val.(bool); ok && !boolVal {
		return nil
	}
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

// parseCommentMemoryTool converts raw comment-memory tool configuration
func parseCommentMemoryTool(val any) *CommentMemoryToolConfig {
	// comment-memory can be boolean, object, or null - store raw value
	return &CommentMemoryToolConfig{Raw: val}
}

// parseRepoMemoryTool converts raw repo-memory tool configuration
func parseRepoMemoryTool(val any) *RepoMemoryToolConfig {
	// repo-memory can be boolean, object, or array - store raw value
	return &RepoMemoryToolConfig{Raw: val}
}

// parseTimeoutTool converts raw timeout tool configuration to a TemplatableInt32 value.
// Accepts integers and GitHub Actions expressions (e.g. "${{ inputs.tool-timeout }}").
func parseTimeoutTool(val any) *TemplatableInt32 {
	switch v := val.(type) {
	case int:
		t := TemplatableInt32(strconv.Itoa(v))
		return &t
	case int64:
		t := TemplatableInt32(strconv.FormatInt(v, 10))
		return &t
	case uint:
		t := TemplatableInt32(strconv.FormatUint(uint64(v), 10))
		return &t
	case uint64:
		t := TemplatableInt32(strconv.FormatUint(v, 10))
		return &t
	case float64:
		t := TemplatableInt32(strconv.Itoa(int(v)))
		return &t
	case string:
		if isExpression(v) {
			t := TemplatableInt32(v)
			return &t
		}
		return nil // reject non-expression strings
	}
	return nil
}

// parseStartupTimeoutTool converts raw startup-timeout tool configuration to a TemplatableInt32 value.
// Accepts integers and GitHub Actions expressions (e.g. "${{ inputs.startup-timeout }}").
func parseStartupTimeoutTool(val any) *TemplatableInt32 {
	switch v := val.(type) {
	case int:
		t := TemplatableInt32(strconv.Itoa(v))
		return &t
	case int64:
		t := TemplatableInt32(strconv.FormatInt(v, 10))
		return &t
	case uint:
		t := TemplatableInt32(strconv.FormatUint(uint64(v), 10))
		return &t
	case uint64:
		t := TemplatableInt32(strconv.FormatUint(v, 10))
		return &t
	case float64:
		t := TemplatableInt32(strconv.Itoa(int(v)))
		return &t
	case string:
		if isExpression(v) {
			t := TemplatableInt32(v)
			return &t
		}
		return nil // reject non-expression strings
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

	parseMCPServerCommonFields(configMap, &config)
	parseMCPServerHTTPFields(configMap, &config)
	parseMCPServerContainerFields(configMap, &config)
	parseMCPServerCustomFields(configMap, &config)

	return config
}

func parseMCPServerCommonFields(configMap map[string]any, config *MCPServerConfig) {
	if command, ok := configMap["command"].(string); ok {
		config.Command = command
	}
	if args, ok := configMap["args"].([]any); ok {
		config.Args = parseStringSlice(args)
	}
	if env, ok := configMap["env"].(map[string]any); ok {
		config.Env = parseStringMap(env)
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
		config.Toolsets = parseStringSlice(toolsets)
	}
}

func parseMCPServerHTTPFields(configMap map[string]any, config *MCPServerConfig) {
	if url, ok := configMap["url"].(string); ok {
		config.URL = url
	}
	if headers, ok := configMap["headers"].(map[string]any); ok {
		config.Headers = parseStringMap(headers)
	}
}

func parseMCPServerContainerFields(configMap map[string]any, config *MCPServerConfig) {
	if container, ok := configMap["container"].(string); ok {
		config.Container = container
	}
	if entrypoint, ok := configMap["entrypoint"].(string); ok {
		config.Entrypoint = entrypoint
	}
	if entrypointArgs, ok := configMap["entrypointArgs"].([]any); ok {
		config.EntrypointArgs = parseStringSlice(entrypointArgs)
	}
	if mounts, ok := configMap["mounts"].([]any); ok {
		config.Mounts = parseStringSlice(mounts)
	}
}

func parseMCPServerCustomFields(configMap map[string]any, config *MCPServerConfig) {
	knownFields := map[string]struct{}{
		"command":        {},
		"args":           {},
		"env":            {},
		"mode":           {},
		"type":           {},
		"version":        {},
		"toolsets":       {},
		"url":            {},
		"headers":        {},
		"container":      {},
		"entrypoint":     {},
		"entrypointArgs": {},
		"mounts":         {},
	}
	for key, value := range configMap {
		if !setutil.Contains(knownFields, key) {
			config.CustomFields[key] = value
		}
	}
}

func parseStringMap(values map[string]any) map[string]string {
	parsed := make(map[string]string)
	for k, v := range values {
		if str, ok := v.(string); ok {
			parsed[k] = str
		}
	}
	return parsed
}
