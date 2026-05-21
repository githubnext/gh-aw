package parser

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/types"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var mcpLog = logger.New("parser:mcp")

// ValidMCPTypes defines all supported MCP server types.
// "local" is an alias for "stdio" and gets normalized during parsing.
var ValidMCPTypes = []string{"stdio", "http", "local"}

// IsMCPType checks if a type string is a valid MCP server type.
// Returns true for "stdio", "http", and "local" (which is an alias for "stdio").
func IsMCPType(typeStr string) bool {
	switch typeStr {
	case "stdio", "http", "local":
		return true
	default:
		return false
	}
}

// RegistryMCPServerConfig represents a parser-layer MCP server configuration.
// It is intentionally distinct from workflow.MCPServerConfig, which models
// workflow-facing YAML tool configuration.
// It embeds BaseMCPServerConfig for common fields and adds parser-specific fields.
type RegistryMCPServerConfig struct {
	types.BaseMCPServerConfig

	// Parser-specific fields
	Name      string   `json:"name"`       // Server name/identifier
	Registry  string   `json:"registry"`   // URI to installation location from registry
	ProxyArgs []string `json:"proxy-args"` // custom proxy arguments for container-based tools
	Allowed   []string `json:"allowed"`    // allowed tools
}

// MCPServerInfo contains the inspection results for an MCP server
type MCPServerInfo struct {
	Config    RegistryMCPServerConfig
	Connected bool
	Error     error
	Tools     []*mcp.Tool
	Resources []*mcp.Resource
	Roots     []*mcp.Root
}

// extractSafeOutputsConfig extracts the safe-outputs MCP server config from frontmatter.
func extractSafeOutputsConfig(frontmatter map[string]any, serverFilter string) *RegistryMCPServerConfig {
	safeOutputsSection, hasSafeOutputs := frontmatter["safe-outputs"]
	if !hasSafeOutputs {
		return nil
	}
	if serverFilter != "" && !strings.Contains(constants.SafeOutputsMCPServerID.String(), strings.ToLower(serverFilter)) {
		return nil
	}
	config := RegistryMCPServerConfig{
		BaseMCPServerConfig: types.BaseMCPServerConfig{
			Type:    "stdio",
			Command: "node",
			Env:     make(map[string]string),
		},
		Name: constants.SafeOutputsMCPServerID.String(),
	}
	if safeOutputsMap, ok := safeOutputsSection.(map[string]any); ok {
		for toolType := range safeOutputsMap {
			switch toolType {
			case "create-issue", "create-discussion", "add-comment", "create-pull-request",
				"create-pull-request-review-comment", "create-code-scanning-alert",
				"add-labels", "update-issue", "push-to-pull-request-branch", "missing-tool":
				config.Allowed = append(config.Allowed, toolType)
			}
		}
	}
	return &config
}

// extractSafeJobsConfig merges safe-jobs entries into the safe-outputs MCP server config.
func extractSafeJobsConfig(frontmatter map[string]any, serverFilter string, configs []RegistryMCPServerConfig) []RegistryMCPServerConfig {
	safeJobsSection, hasSafeJobs := frontmatter["safe-jobs"]
	if !hasSafeJobs {
		return configs
	}
	if serverFilter != "" && !strings.Contains(constants.SafeOutputsMCPServerID.String(), strings.ToLower(serverFilter)) {
		return configs
	}
	var config *RegistryMCPServerConfig
	for i := range configs {
		if configs[i].Name == constants.SafeOutputsMCPServerID.String() {
			config = &configs[i]
			break
		}
	}
	if config == nil {
		newConfig := RegistryMCPServerConfig{
			BaseMCPServerConfig: types.BaseMCPServerConfig{
				Type:    "stdio",
				Command: "node",
				Env:     make(map[string]string),
			},
			Name: constants.SafeOutputsMCPServerID.String(),
		}
		configs = append(configs, newConfig)
		config = &configs[len(configs)-1]
	}
	if safeJobsMap, ok := safeJobsSection.(map[string]any); ok {
		for jobName := range safeJobsMap {
			config.Allowed = append(config.Allowed, jobName)
		}
	}
	return configs
}

// extractMCPScriptsConfig extracts the mcp-scripts MCP server config from frontmatter.
func extractMCPScriptsConfig(frontmatter map[string]any, serverFilter string) *RegistryMCPServerConfig {
	mcpScriptsSection, hasMCPScripts := frontmatter["mcp-scripts"]
	if !hasMCPScripts {
		return nil
	}
	if serverFilter != "" && !strings.Contains(constants.MCPScriptsMCPServerID.String(), strings.ToLower(serverFilter)) {
		return nil
	}
	config := RegistryMCPServerConfig{
		BaseMCPServerConfig: types.BaseMCPServerConfig{
			Type:    "http",
			Command: "",
			Env:     make(map[string]string),
		},
		Name: constants.MCPScriptsMCPServerID.String(),
	}
	if mcpScriptsMap, ok := mcpScriptsSection.(map[string]any); ok {
		for toolName := range mcpScriptsMap {
			if toolName == "mode" {
				continue
			}
			config.Allowed = append(config.Allowed, toolName)
		}
	}
	return &config
}

// ExtractMCPConfigurations extracts MCP server configurations from workflow frontmatter
func ExtractMCPConfigurations(frontmatter map[string]any, serverFilter string) ([]RegistryMCPServerConfig, error) {
	mcpLog.Printf("Extracting MCP configurations with filter: %s", serverFilter)
	var configs []RegistryMCPServerConfig

	if safeOutputsCfg := extractSafeOutputsConfig(frontmatter, serverFilter); safeOutputsCfg != nil {
		mcpLog.Print("Found safe-outputs configuration")
		configs = append(configs, *safeOutputsCfg)
	}
	configs = extractSafeJobsConfig(frontmatter, serverFilter, configs)
	if mcpScriptsCfg := extractMCPScriptsConfig(frontmatter, serverFilter); mcpScriptsCfg != nil {
		mcpLog.Print("Found mcp-scripts configuration")
		configs = append(configs, *mcpScriptsCfg)
	}

	mcpServersSection, hasMCPServers := frontmatter["mcp-servers"]
	if !hasMCPServers {
		mcpLog.Print("No mcp-servers section found, checking for built-in tools")
		if err := extractBuiltinMCPTools(frontmatter, serverFilter, &configs); err != nil {
			return nil, err
		}
		mcpLog.Printf("Extracted %d MCP configurations total", len(configs))
		return configs, nil
	}
	mcpServers, ok := mcpServersSection.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("mcp-servers section must be a map, got %T. Example:\nmcp-servers:\n  my-server:\n    command: \"npx @my/tool\"\n    args: [\"--port\", \"3000\"]", mcpServersSection)
	}
	if err := extractBuiltinMCPTools(frontmatter, serverFilter, &configs); err != nil {
		return nil, err
	}
	mcpLog.Printf("Processing %d custom MCP servers", len(mcpServers))
	for serverName, serverValue := range mcpServers {
		if serverFilter != "" && !strings.Contains(strings.ToLower(serverName), strings.ToLower(serverFilter)) {
			continue
		}
		toolConfig, ok := serverValue.(map[string]any)
		if !ok {
			continue
		}
		config, err := ParseMCPConfig(serverName, toolConfig, toolConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to parse MCP config for %s: %w", serverName, err)
		}
		mcpLog.Printf("Parsed custom MCP server: %s (type=%s)", serverName, config.Type)
		configs = append(configs, config)
	}
	mcpLog.Printf("Extracted %d MCP configurations total", len(configs))
	return configs, nil
}

// extractBuiltinMCPTools reads the tools section and appends github/playwright configs to configs.
// It returns an error if a removed tool (serena) is present.
func extractBuiltinMCPTools(frontmatter map[string]any, serverFilter string, configs *[]RegistryMCPServerConfig) error {
	toolsSection, hasTools := frontmatter["tools"]
	if !hasTools {
		return nil
	}
	tools, ok := toolsSection.(map[string]any)
	if !ok {
		return nil
	}
	for toolName, toolValue := range tools {
		if toolName == "serena" {
			return errors.New("tools.serena is removed")
		}
		if toolName == "github" || toolName == "playwright" {
			config, err := processBuiltinMCPTool(toolName, toolValue, serverFilter)
			if err != nil {
				return err
			}
			if config != nil {
				mcpLog.Printf("Added built-in MCP tool: %s", toolName)
				*configs = append(*configs, *config)
			}
		}
	}
	return nil
}

// parseGitHubMCPOptions extracts mode and github-token options from the github tool config.
func parseGitHubMCPOptions(toolValue any) (useRemote bool, customGitHubToken string) {
	toolConfig, ok := toolValue.(map[string]any)
	if !ok {
		return false, ""
	}
	if modeField, hasMode := toolConfig["mode"]; hasMode {
		if modeStr, ok := modeField.(string); ok && modeStr == "remote" {
			useRemote = true
		}
	}
	if token, hasToken := toolConfig["github-token"]; hasToken {
		if tokenStr, ok := token.(string); ok {
			customGitHubToken = tokenStr
		}
	}
	return useRemote, customGitHubToken
}

// buildGitHubMCPBaseConfig builds the base RegistryMCPServerConfig for the GitHub MCP tool.
func buildGitHubMCPBaseConfig(useRemote bool, customGitHubToken string) RegistryMCPServerConfig {
	if useRemote {
		config := RegistryMCPServerConfig{
			BaseMCPServerConfig: types.BaseMCPServerConfig{
				Type:    "http",
				URL:     "https://api.githubcopilot.com/mcp/",
				Headers: make(map[string]string),
				Env:     make(map[string]string),
			},
			Name: "github",
		}
		if customGitHubToken != "" {
			config.Env["GITHUB_TOKEN"] = customGitHubToken
		}
		config.Headers["X-MCP-Readonly"] = "true"
		return config
	}
	config := RegistryMCPServerConfig{
		BaseMCPServerConfig: types.BaseMCPServerConfig{
			Type:    "docker",
			Command: "docker",
			Args: []string{
				"run", "-i", "--rm", "-e", "GITHUB_PERSONAL_ACCESS_TOKEN",
				"-e", "GITHUB_READ_ONLY=1",
				"ghcr.io/github/github-mcp-server:" + string(constants.DefaultGitHubMCPServerVersion),
			},
			Env: make(map[string]string),
		},
		Name: "github",
	}
	if githubToken, err := GetGitHubToken(); err == nil {
		config.Env["GITHUB_PERSONAL_ACCESS_TOKEN"] = githubToken
	} else {
		config.Env["GITHUB_PERSONAL_ACCESS_TOKEN"] = "${GITHUB_TOKEN_REQUIRED}"
	}
	return config
}

// applyGitHubMCPCustomizations applies allowed tools, version, and extra args from the tool config.
func applyGitHubMCPCustomizations(toolConfig map[string]any, config *RegistryMCPServerConfig, useRemote bool) {
	if allowed, hasAllowed := toolConfig["allowed"]; hasAllowed {
		if allowedSlice, ok := allowed.([]any); ok {
			for _, item := range allowedSlice {
				if str, ok := item.(string); ok {
					config.Allowed = append(config.Allowed, str)
				}
			}
		}
	}
	if useRemote {
		return
	}
	if version, exists := toolConfig["version"]; exists {
		if versionStr := stringutil.ParseVersionValue(version); versionStr != "" {
			dockerImage := "ghcr.io/github/github-mcp-server:" + versionStr
			for i, arg := range config.Args {
				if strings.HasPrefix(arg, "ghcr.io/github/github-mcp-server:") {
					config.Args[i] = dockerImage
					break
				}
			}
		}
	}
	if argsValue, exists := toolConfig["args"]; exists {
		if argsSlice, ok := argsValue.([]any); ok {
			for _, arg := range argsSlice {
				if argStr, ok := arg.(string); ok {
					config.Args = append(config.Args, argStr)
				}
			}
		}
		if argsSlice, ok := argsValue.([]string); ok {
			config.Args = append(config.Args, argsSlice...)
		}
	}
}

// buildPlaywrightMCPConfig builds a RegistryMCPServerConfig for the Playwright MCP tool.
func buildPlaywrightMCPConfig(toolValue any) RegistryMCPServerConfig {
	config := RegistryMCPServerConfig{
		BaseMCPServerConfig: types.BaseMCPServerConfig{
			Type:    "docker",
			Command: "docker",
			Args: []string{
				"run", "-i", "--rm", "--shm-size=2gb", "--cap-add=SYS_ADMIN",
				"-v", "/tmp/gh-aw/mcp-logs:/tmp/gh-aw/mcp-logs",
				"mcr.microsoft.com/playwright:" + string(constants.DefaultPlaywrightBrowserVersion),
			},
			Env: make(map[string]string),
		},
		Name: "playwright",
	}
	toolConfig, ok := toolValue.(map[string]any)
	if !ok {
		return config
	}
	if version, exists := toolConfig["version"]; exists {
		if versionStr := stringutil.ParseVersionValue(version); versionStr != "" {
			dockerImage := "mcr.microsoft.com/playwright:" + versionStr
			for i, arg := range config.Args {
				if strings.HasPrefix(arg, "mcr.microsoft.com/playwright:") {
					config.Args[i] = dockerImage
					break
				}
			}
		}
	}
	if argsValue, exists := toolConfig["args"]; exists {
		if argsSlice, ok := argsValue.([]any); ok {
			for _, arg := range argsSlice {
				if argStr, ok := arg.(string); ok {
					config.Args = append(config.Args, argStr)
				}
			}
		}
		if argsSlice, ok := argsValue.([]string); ok {
			config.Args = append(config.Args, argsSlice...)
		}
	}
	return config
}

// processBuiltinMCPTool handles built-in MCP tools (github and playwright)
func processBuiltinMCPTool(toolName string, toolValue any, serverFilter string) (*RegistryMCPServerConfig, error) {
	if serverFilter != "" && !strings.Contains(strings.ToLower(toolName), strings.ToLower(serverFilter)) {
		return nil, nil
	}
	if toolName == "github" {
		useRemote, customGitHubToken := parseGitHubMCPOptions(toolValue)
		config := buildGitHubMCPBaseConfig(useRemote, customGitHubToken)
		if toolConfig, ok := toolValue.(map[string]any); ok {
			applyGitHubMCPCustomizations(toolConfig, &config, useRemote)
		}
		return &config, nil
	}
	if toolName == "playwright" {
		config := buildPlaywrightMCPConfig(toolValue)
		return &config, nil
	}
	return nil, nil
}

// parseMCPType infers or parses the MCP server type from the config map.
func parseMCPType(mcpConfig map[string]any, toolName string) (string, error) {
	if typeVal, hasType := mcpConfig["type"]; hasType {
		typeStr, ok := typeVal.(string)
		if !ok {
			return "", fmt.Errorf("type field must be a string, got %T. Valid types are: stdio, http. Example:\nmcp-servers:\n  %s:\n    type: stdio\n    command: \"npx @my/tool\"", typeVal, toolName)
		}
		if typeStr == "local" {
			return "stdio", nil
		}
		return typeStr, nil
	}
	if _, hasURL := mcpConfig["url"]; hasURL {
		mcpLog.Printf("Inferred MCP type 'http' for tool %s based on url field", toolName)
		return "http", nil
	}
	if _, hasCommand := mcpConfig["command"]; hasCommand {
		mcpLog.Printf("Inferred MCP type 'stdio' for tool %s based on command field", toolName)
		return "stdio", nil
	}
	if _, hasContainer := mcpConfig["container"]; hasContainer {
		mcpLog.Printf("Inferred MCP type 'stdio' for tool %s based on container field", toolName)
		return "stdio", nil
	}
	return "", fmt.Errorf("unable to determine MCP type for tool '%s': missing type, url, command, or container. Must specify one of: 'type' (stdio/http), 'url' (for HTTP MCP), 'command' (for command-based), or 'container' (for Docker-based). Example:\nmcp-servers:\n  %s:\n    command: \"npx @my/tool\"\n    args: [\"--port\", \"3000\"]", toolName, toolName)
}

// appendMCPContainerEnvArgs adds sorted env vars to the Docker run args and config.Env.
func appendMCPContainerEnvArgs(mcpConfig map[string]any, config *RegistryMCPServerConfig) {
	env, hasEnv := mcpConfig["env"]
	if !hasEnv {
		return
	}
	envMap, ok := env.(map[string]any)
	if !ok {
		return
	}
	var envKeys []string
	for key := range envMap {
		envKeys = append(envKeys, key)
	}
	sort.Strings(envKeys)
	for _, key := range envKeys {
		if valueStr, ok := envMap[key].(string); ok {
			config.Args = append(config.Args, "-e", key)
			config.Env[key] = valueStr
		}
	}
}

// appendMCPContainerMountArgs adds sorted volume mounts to the Docker run args and config.Mounts.
func appendMCPContainerMountArgs(mcpConfig map[string]any, config *RegistryMCPServerConfig) {
	mounts, hasMounts := mcpConfig["mounts"]
	if !hasMounts {
		return
	}
	mountsSlice, ok := mounts.([]any)
	if !ok {
		return
	}
	var mountStrings []string
	for _, mount := range mountsSlice {
		if mountStr, ok := mount.(string); ok {
			mountStrings = append(mountStrings, mountStr)
			config.Mounts = append(config.Mounts, mountStr)
		}
	}
	sort.Strings(mountStrings)
	for _, mountStr := range mountStrings {
		config.Args = append(config.Args, "-v", mountStr)
	}
}

// parseMCPStdioContainer populates config from a container-based stdio MCP server definition.
func parseMCPStdioContainer(mcpConfig map[string]any, toolName string, config *RegistryMCPServerConfig) error {
	container, hasContainer := mcpConfig["container"]
	if !hasContainer {
		return nil
	}
	containerStr, ok := container.(string)
	if !ok {
		return nil
	}
	mcpLog.Printf("Tool %s uses container: %s", toolName, containerStr)
	config.Container = containerStr
	config.Command = "docker"
	config.Args = []string{"run", "--rm", "-i"}
	appendMCPContainerEnvArgs(mcpConfig, config)
	appendMCPContainerMountArgs(mcpConfig, config)
	if entrypoint, hasEntrypoint := mcpConfig["entrypoint"]; hasEntrypoint {
		if entrypointStr, ok := entrypoint.(string); ok {
			config.Entrypoint = entrypointStr
			config.Args = append(config.Args, "--entrypoint", entrypointStr)
		}
	}
	config.Args = append(config.Args, containerStr)
	if entrypointArgs, hasEntrypointArgs := mcpConfig["entrypointArgs"]; hasEntrypointArgs {
		if entrypointArgsSlice, ok := entrypointArgs.([]any); ok {
			for _, arg := range entrypointArgsSlice {
				if argStr, ok := arg.(string); ok {
					config.Args = append(config.Args, argStr)
				}
			}
		}
	}
	return nil
}

// parseMCPStdioCommand populates config from a command-based stdio MCP server definition.
func parseMCPStdioCommand(mcpConfig map[string]any, toolName string, config *RegistryMCPServerConfig) error {
	command, hasCommand := mcpConfig["command"]
	if !hasCommand {
		return fmt.Errorf(
			"stdio MCP tool '%s' must specify either 'command' or 'container' field. Cannot specify both. "+
				"Example with command:\n"+
				"mcp-servers:\n"+
				"  %s:\n"+
				"    command: \"npx @my/tool\"\n"+
				"    args: [\"--port\", \"3000\"]\n\n"+
				"Example with container:\n"+
				"mcp-servers:\n"+
				"  %s:\n"+
				"    container: \"myorg/my-tool:latest\"\n"+
				"    env:\n"+
				"      API_KEY: \"${{ secrets.API_KEY }}\"",
			toolName, toolName, toolName,
		)
	}
	commandStr, ok := command.(string)
	if !ok {
		return fmt.Errorf("command field must be a string, got %T. Example:\nmcp-servers:\n  %s:\n    command: \"npx @my/tool\"\n    args: [\"--port\", \"3000\"]", command, toolName)
	}
	config.Command = commandStr
	if args, hasArgs := mcpConfig["args"]; hasArgs {
		if argsSlice, ok := args.([]any); ok {
			for _, arg := range argsSlice {
				if argStr, ok := arg.(string); ok {
					config.Args = append(config.Args, argStr)
				}
			}
		}
	}
	return nil
}

// parseMCPStdioEnvNetwork extracts env vars and network proxy-args from a stdio config.
func parseMCPStdioEnvNetwork(mcpConfig map[string]any, config *RegistryMCPServerConfig) {
	if env, hasEnv := mcpConfig["env"]; hasEnv {
		if envMap, ok := env.(map[string]any); ok {
			for key, value := range envMap {
				if valueStr, ok := value.(string); ok {
					config.Env[key] = valueStr
				}
			}
		}
	}
	if network, hasNetwork := mcpConfig["network"]; hasNetwork {
		if networkMap, ok := network.(map[string]any); ok {
			if proxyArgs, hasProxyArgs := networkMap["proxy-args"]; hasProxyArgs {
				if proxyArgsSlice, ok := proxyArgs.([]any); ok {
					for _, arg := range proxyArgsSlice {
						if argStr, ok := arg.(string); ok {
							config.ProxyArgs = append(config.ProxyArgs, argStr)
						}
					}
				}
			}
		}
	}
}

// parseMCPStdio dispatches container vs command stdio parsing, then extracts env and network.
func parseMCPStdio(mcpConfig map[string]any, toolName string, config *RegistryMCPServerConfig) error {
	if _, hasContainer := mcpConfig["container"]; hasContainer {
		if err := parseMCPStdioContainer(mcpConfig, toolName, config); err != nil {
			return err
		}
	} else {
		if err := parseMCPStdioCommand(mcpConfig, toolName, config); err != nil {
			return err
		}
	}
	parseMCPStdioEnvNetwork(mcpConfig, config)
	return nil
}

// parseMCPHTTP populates config from an HTTP MCP server definition.
func parseMCPHTTP(mcpConfig map[string]any, toolName string, config *RegistryMCPServerConfig) error {
	url, hasURL := mcpConfig["url"]
	if !hasURL {
		return fmt.Errorf(
			"http MCP tool '%s' missing required 'url' field. HTTP MCP servers must specify a URL endpoint. "+
				"Example:\n"+
				"mcp-servers:\n"+
				"  %s:\n"+
				"    type: http\n"+
				"    url: \"https://api.example.com/mcp\"\n"+
				"    headers:\n"+
				"      Authorization: \"Bearer ${{ secrets.API_KEY }}\"",
			toolName, toolName,
		)
	}
	urlStr, ok := url.(string)
	if !ok {
		return fmt.Errorf(
			"url field must be a string, got %T. Example:\n"+
				"mcp-servers:\n"+
				"  %s:\n"+
				"    type: http\n"+
				"    url: \"https://api.example.com/mcp\"\n"+
				"    headers:\n"+
				"      Authorization: \"Bearer ${{ secrets.API_KEY }}\"",
			url, toolName)
	}
	mcpLog.Printf("Tool %s uses HTTP transport with URL: %s", toolName, urlStr)
	config.URL = urlStr
	if headers, hasHeaders := mcpConfig["headers"]; hasHeaders {
		if headersMap, ok := headers.(map[string]any); ok {
			for key, value := range headersMap {
				if valueStr, ok := value.(string); ok {
					config.Headers[key] = valueStr
				}
			}
		}
	}
	return nil
}

// ParseMCPConfig parses MCP configuration from various formats (map or JSON string)
func ParseMCPConfig(toolName string, mcpSection any, toolConfig map[string]any) (RegistryMCPServerConfig, error) {
	mcpLog.Printf("Parsing MCP configuration for tool: %s", toolName)
	config := RegistryMCPServerConfig{
		BaseMCPServerConfig: types.BaseMCPServerConfig{
			Env:     make(map[string]string),
			Headers: make(map[string]string),
		},
		Name: toolName,
	}
	if allowed, hasAllowed := toolConfig["allowed"]; hasAllowed {
		if allowedSlice, ok := allowed.([]any); ok {
			for _, item := range allowedSlice {
				if str, ok := item.(string); ok {
					config.Allowed = append(config.Allowed, str)
				}
			}
		}
	}
	var mcpConfig map[string]any
	switch v := mcpSection.(type) {
	case map[string]any:
		mcpConfig = v
	case string:
		if err := json.Unmarshal([]byte(v), &mcpConfig); err != nil {
			return config, fmt.Errorf("invalid JSON in mcp configuration: %w", err)
		}
	default:
		return config, fmt.Errorf("mcp configuration must be a map or JSON string, got %T. Example:\nmcp-servers:\n  %s:\n    command: \"npx @my/tool\"\n    args: [\"--port\", \"3000\"]", v, toolName)
	}
	mcpType, err := parseMCPType(mcpConfig, toolName)
	if err != nil {
		return config, err
	}
	config.Type = mcpType
	if registry, hasRegistry := mcpConfig["registry"]; hasRegistry {
		if registryStr, ok := registry.(string); ok {
			config.Registry = registryStr
		} else {
			return config, fmt.Errorf("registry field must be a string, got %T. Example:\nmcp-servers:\n  %s:\n    registry: \"https://registry.npmjs.org/@my/tool\"\n    command: \"npx @my/tool\"", registry, toolName)
		}
	}
	mcpLog.Printf("Extracting %s configuration for tool: %s", config.Type, toolName)
	switch config.Type {
	case "stdio":
		if err := parseMCPStdio(mcpConfig, toolName, &config); err != nil {
			return config, err
		}
	case "http":
		if err := parseMCPHTTP(mcpConfig, toolName, &config); err != nil {
			return config, err
		}
	default:
		return config, fmt.Errorf("unsupported MCP type '%s' for tool '%s'. Valid types are: stdio, http. Example:\nmcp-servers:\n  %s:\n    type: stdio\n    command: \"npx @my/tool\"\n    args: [\"--port\", \"3000\"]", config.Type, toolName, toolName)
	}
	return config, nil
}
