package workflow

import (
	"fmt"
	"maps"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/setutil"
	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/types"
)

var mcpCustomLog = logger.New("workflow:mcp-config-custom")

// renderCustomMCPConfigWrapperWithContext generates custom MCP server configuration wrapper with workflow context
// This version includes workflowData to determine if localhost URLs should be rewritten
func renderCustomMCPConfigWrapperWithContext(yaml *strings.Builder, toolName string, toolConfig map[string]any, isLast bool, workflowData *WorkflowData) error {
	mcpCustomLog.Printf("Rendering custom MCP config wrapper with context for tool: %s", toolName)
	fmt.Fprintf(yaml, "              \"%s\": {\n", toolName)

	// Determine if localhost URLs should be rewritten to host.docker.internal
	// This is needed when firewall is enabled (agent is not disabled)
	rewriteLocalhost := shouldRewriteLocalhostToDocker(workflowData)

	// Use the shared MCP config renderer with JSON format
	renderer := MCPConfigRenderer{
		IndentLevel:              "                ",
		Format:                   "json",
		RewriteLocalhostToDocker: rewriteLocalhost,
		GuardPolicies:            deriveWriteSinkGuardPolicyFromWorkflow(workflowData),
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

// renderCustomMCPEnvVars normalizes custom MCP env values for the target output
// format before serialization.
//
// For TOML output, GitHub Actions template expressions are rewritten to direct
// ${VAR} references because Codex config expects shell-style environment
// expansion. For JSON output, Copilot uses escaped \${VAR} passthrough syntax,
// while non-Copilot engines use bash variable substitution to avoid embedding
// secret expressions directly in the generated run block.
func renderCustomMCPEnvVars(env map[string]string, tomlFormat, requiresCopilotFields bool) map[string]string {
	renderedEnv := make(map[string]string, len(env))
	for envKey, envValue := range env {
		if tomlFormat {
			// Replace template expressions with environment variable references for TOML.
			// For TOML, we use direct shell variable syntax without backslash.
			envValue = strings.ReplaceAll(envValue, "${{ secrets.", "${")
			envValue = strings.ReplaceAll(envValue, "${{ env.", "${")
			envValue = strings.ReplaceAll(envValue, "${{ github.workspace }}", "${GITHUB_WORKSPACE}")
			envValue = strings.ReplaceAll(envValue, " }}", "}")
		} else if requiresCopilotFields {
			// For Copilot, replace all template expressions with \${VAR} syntax.
			envValue = ReplaceTemplateExpressionsWithEnvVars(envValue)
		} else {
			// For non-Copilot engines, replace secrets with ${VAR} bash expansion so
			// they are never directly interpolated in the run block (RGS-008). The
			// env vars are injected into the step env block by collectMCPEnvironmentVariables.
			envValue = ReplaceSecretsWithBashVars(envValue)
		}
		renderedEnv[envKey] = envValue
	}

	return renderedEnv
}

// renderSharedMCPConfig generates MCP server configuration for a single tool using shared logic
// This function handles the common logic for rendering MCP configurations across different engines
func renderSharedMCPConfig(yaml *strings.Builder, toolName string, toolConfig map[string]any, renderer MCPConfigRenderer) error {
	mcpCustomLog.Printf("Rendering MCP config for tool: %s, format: %s", toolName, renderer.Format)

	mcpConfig, err := getMCPConfig(toolConfig, toolName)
	if err != nil {
		mcpCustomLog.Printf("Failed to parse MCP config for tool %s: %v", toolName, err)
		return fmt.Errorf("failed to parse MCP config for tool '%s': %w", toolName, err)
	}
	if err := validateMCPGatewayStdioCommand(toolName, mcpConfig); err != nil {
		return err
	}

	headerSecrets := mcpHeaderSecrets(mcpConfig)
	existingProperties, err := existingMCPConfigProperties(toolName, mcpConfig, renderer, headerSecrets)
	if err != nil || len(existingProperties) == 0 {
		return err
	}

	hasTrailingGuardPolicies := renderer.Format == "json" && len(renderer.GuardPolicies) > 0
	renderMCPConfigProperties(yaml, mcpConfig, renderer, existingProperties, headerSecrets, hasTrailingGuardPolicies)
	renderMCPConfigGuardPolicies(yaml, toolName, renderer, hasTrailingGuardPolicies)
	return nil
}

func validateMCPGatewayStdioCommand(toolName string, mcpConfig *parser.RegistryMCPServerConfig) error {
	if mcpConfig.Type != "stdio" || mcpConfig.Command == "" || mcpConfig.Command == "docker" {
		return nil
	}
	return fmt.Errorf(
		"tool '%s' stdio MCP server uses command %q which is not supported by MCP Gateway. "+
			"Stdio servers must be containerized (use 'container' with 'entrypoint'), "+
			"or switch to HTTP transport for servers that run directly on the runner.\n\n"+
			"Example (container):\ntools:\n  %s:\n    container: \"my-registry/my-tool:latest\"\n    entrypoint: \"my-tool\"\n    args: [\"--verbose\"]\n\n"+
			"Example (HTTP — for Python/Node servers installed on the runner):\ntools:\n  %s:\n    type: http\n    url: \"http://localhost:8765/mcp\"",
		toolName, mcpConfig.Command, toolName, toolName,
	)
}

func mcpHeaderSecrets(mcpConfig *parser.RegistryMCPServerConfig) map[string]string {
	if mcpConfig.Type == "http" {
		return ExtractSecretsFromMap(mcpConfig.Headers)
	}
	return nil
}

func existingMCPConfigProperties(toolName string, mcpConfig *parser.RegistryMCPServerConfig, renderer MCPConfigRenderer, headerSecrets map[string]string) ([]string, error) {
	propertyOrder, err := mcpConfigPropertyOrder(toolName, mcpConfig, renderer, headerSecrets)
	if err != nil {
		return nil, err
	}
	var existingProperties []string
	for _, prop := range propertyOrder {
		if mcpConfigPropertyExists(prop, mcpConfig, renderer, headerSecrets) {
			existingProperties = append(existingProperties, prop)
		}
	}
	return existingProperties, nil
}

func mcpConfigPropertyOrder(toolName string, mcpConfig *parser.RegistryMCPServerConfig, renderer MCPConfigRenderer, headerSecrets map[string]string) ([]string, error) {
	switch mcpConfig.Type {
	case "stdio":
		if renderer.Format == "toml" {
			return []string{"container", "entrypoint", "entrypointArgs", "mounts", "command", "args", "env", "proxy-args", "registry"}, nil
		}
		return []string{"type", "container", "entrypoint", "entrypointArgs", "mounts", "command", "args", "tools", "env", "proxy-args", "registry"}, nil
	case "http":
		if renderer.Format == "toml" {
			return []string{"url", "http_headers"}, nil
		}
		if len(headerSecrets) > 0 {
			return []string{"type", "url", "headers", "auth", "tools", "env"}, nil
		}
		return []string{"type", "url", "headers", "auth", "tools"}, nil
	default:
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Custom MCP server '%s' has unsupported type '%s'. Supported types: stdio, http", toolName, mcpConfig.Type)))
		return nil, nil
	}
}

func mcpConfigPropertyExists(prop string, mcpConfig *parser.RegistryMCPServerConfig, renderer MCPConfigRenderer, headerSecrets map[string]string) bool {
	switch prop {
	case "type":
		return true
	case "tools":
		return renderer.RequiresCopilotFields || len(mcpConfig.Allowed) > 0
	case "container":
		return mcpConfig.Container != ""
	case "entrypoint":
		return mcpConfig.Entrypoint != ""
	case "entrypointArgs":
		return len(mcpConfig.EntrypointArgs) > 0
	case "mounts":
		return len(mcpConfig.Mounts) > 0
	case "command":
		return mcpConfig.Command != ""
	case "args":
		return len(mcpConfig.Args) > 0
	case "env":
		return len(mcpConfig.Env) > 0 || len(headerSecrets) > 0
	case "url":
		return mcpConfig.URL != ""
	case "headers", "http_headers":
		return len(mcpConfig.Headers) > 0
	case "auth":
		return mcpConfig.Auth != nil
	case "proxy-args":
		return len(mcpConfig.ProxyArgs) > 0
	case "registry":
		return mcpConfig.Registry != ""
	default:
		return false
	}
}

func renderMCPConfigProperties(yaml *strings.Builder, mcpConfig *parser.RegistryMCPServerConfig, renderer MCPConfigRenderer, properties []string, headerSecrets map[string]string, hasTrailingGuardPolicies bool) {
	for propIndex, property := range properties {
		isLast := (propIndex == len(properties)-1) && !hasTrailingGuardPolicies
		renderMCPConfigProperty(yaml, property, mcpConfig, renderer, headerSecrets, isLast)
	}
}

func renderMCPConfigProperty(yaml *strings.Builder, property string, mcpConfig *parser.RegistryMCPServerConfig, renderer MCPConfigRenderer, headerSecrets map[string]string, isLast bool) {
	switch property {
	case "type":
		renderMCPJSONStringProperty(yaml, renderer.IndentLevel, "type", mcpConfig.Type, isLast)
	case "tools":
		renderMCPToolsProperty(yaml, renderer.IndentLevel, mcpConfig.Allowed, isLast)
	case "container":
		renderMCPStringProperty(yaml, renderer, "container", "container", mcpConfig.Container, isLast)
	case "entrypoint":
		renderMCPStringProperty(yaml, renderer, "entrypoint", "entrypoint", mcpConfig.Entrypoint, isLast)
	case "entrypointArgs":
		renderMCPArrayProperty(yaml, renderer, "entrypointArgs", "entrypointArgs", mcpConfig.EntrypointArgs, true, true, isLast)
	case "mounts":
		renderMCPArrayProperty(yaml, renderer, "mounts", "mounts", mcpConfig.Mounts, true, true, isLast)
	case "command":
		renderMCPStringProperty(yaml, renderer, "command", "command", mcpConfig.Command, isLast)
	case "args":
		renderMCPArrayProperty(yaml, renderer, "args", "args", mcpConfig.Args, false, false, isLast)
	case "env":
		renderMCPEnvProperty(yaml, renderer, mcpConfig.Env, headerSecrets, isLast)
	case "url":
		renderMCPURLProperty(yaml, renderer, mcpConfig.URL, isLast)
	case "http_headers":
		writeTOMLInlineStringMapSection(yaml, renderer.IndentLevel, "http_headers", mcpConfig.Headers)
	case "headers":
		renderMCPHeadersProperty(yaml, renderer, mcpConfig.Headers, headerSecrets, isLast)
	case "auth":
		renderMCPAuthProperty(yaml, renderer.IndentLevel, mcpConfig.Auth, isLast)
	case "proxy-args":
		renderMCPArrayProperty(yaml, renderer, "proxy_args", "proxy-args", mcpConfig.ProxyArgs, false, false, isLast)
	case "registry":
		renderMCPStringProperty(yaml, renderer, "registry", "registry", mcpConfig.Registry, isLast)
	}
}

func renderMCPStringProperty(yaml *strings.Builder, renderer MCPConfigRenderer, tomlName string, jsonName string, value string, isLast bool) {
	if renderer.Format == "toml" {
		fmt.Fprintf(yaml, "%s%s = \"%s\"\n", renderer.IndentLevel, tomlName, value)
		return
	}
	renderMCPJSONStringProperty(yaml, renderer.IndentLevel, jsonName, value, isLast)
}

func renderMCPJSONStringProperty(yaml *strings.Builder, indent string, name string, value string, isLast bool) {
	fmt.Fprintf(yaml, "%s\"%s\": \"%s\"%s\n", indent, name, value, mcpJSONComma(isLast))
}

func renderMCPToolsProperty(yaml *strings.Builder, indent string, allowed []string, isLast bool) {
	fmt.Fprintf(yaml, "%s\"tools\": [\n", indent)
	tools := allowed
	if len(tools) == 0 {
		tools = []string{"*"}
	}
	for toolIndex, tool := range tools {
		fmt.Fprintf(yaml, "%s  \"%s\"%s\n", indent, tool, mcpJSONComma(toolIndex == len(tools)-1))
	}
	fmt.Fprintf(yaml, "%s]%s\n", indent, mcpJSONComma(isLast))
}

func renderMCPArrayProperty(yaml *strings.Builder, renderer MCPConfigRenderer, tomlName string, jsonName string, values []string, tomlInline bool, replaceTemplates bool, isLast bool) {
	if renderer.Format == "toml" {
		renderMCPTOMLArrayProperty(yaml, renderer.IndentLevel, tomlName, values, tomlInline)
		return
	}
	fmt.Fprintf(yaml, "%s\"%s\": [\n", renderer.IndentLevel, jsonName)
	for valueIndex, value := range values {
		if replaceTemplates && renderer.RequiresCopilotFields {
			value = ReplaceTemplateExpressionsWithEnvVars(value)
		}
		fmt.Fprintf(yaml, "%s  \"%s\"%s\n", renderer.IndentLevel, value, mcpJSONComma(valueIndex == len(values)-1))
	}
	fmt.Fprintf(yaml, "%s]%s\n", renderer.IndentLevel, mcpJSONComma(isLast))
}

func renderMCPTOMLArrayProperty(yaml *strings.Builder, indent string, name string, values []string, inline bool) {
	if inline {
		fmt.Fprintf(yaml, "%s%s = [", indent, name)
		for valueIndex, value := range values {
			if valueIndex > 0 {
				yaml.WriteString(", ")
			}
			fmt.Fprintf(yaml, "\"%s\"", value)
		}
		yaml.WriteString("]\n")
		return
	}
	fmt.Fprintf(yaml, "%s%s = [\n", indent, name)
	for _, value := range values {
		fmt.Fprintf(yaml, "%s  \"%s\",\n", indent, value)
	}
	fmt.Fprintf(yaml, "%s]\n", indent)
}

func renderMCPEnvProperty(yaml *strings.Builder, renderer MCPConfigRenderer, env map[string]string, headerSecrets map[string]string, isLast bool) {
	renderedEnv := renderCustomMCPEnvVars(env, renderer.Format == "toml", renderer.RequiresCopilotFields)
	if renderer.Format == "toml" {
		writeTOMLInlineStringMapSection(yaml, renderer.IndentLevel, "env", renderedEnv)
		return
	}
	for varName := range headerSecrets {
		if _, exists := renderedEnv[varName]; !exists {
			renderedEnv[varName] = "\\${" + varName + "}"
		}
	}
	writeJSONStringMapSectionRaw(yaml, renderer.IndentLevel, "env", renderedEnv, !isLast)
}

func renderMCPURLProperty(yaml *strings.Builder, renderer MCPConfigRenderer, urlValue string, isLast bool) {
	if renderer.RewriteLocalhostToDocker {
		urlValue = rewriteLocalhostToDockerHost(urlValue)
	}
	renderMCPStringProperty(yaml, renderer, "url", "url", urlValue, isLast)
}

func renderMCPHeadersProperty(yaml *strings.Builder, renderer MCPConfigRenderer, headers map[string]string, headerSecrets map[string]string, isLast bool) {
	renderedHeaders := make(map[string]string, len(headers))
	for headerKey, headerValue := range headers {
		if len(headerSecrets) > 0 {
			headerValue = ReplaceSecretsWithEnvVars(headerValue, headerSecrets)
		}
		renderedHeaders[headerKey] = headerValue
	}
	writeJSONStringMapSectionRaw(yaml, renderer.IndentLevel, "headers", renderedHeaders, !isLast)
}

func renderMCPAuthProperty(yaml *strings.Builder, indent string, auth *types.MCPAuthConfig, isLast bool) {
	if auth == nil {
		return
	}
	fmt.Fprintf(yaml, "%s\"auth\": {\n", indent)
	if auth.Audience != "" {
		fmt.Fprintf(yaml, "%s  \"type\": \"%s\",\n", indent, auth.Type)
		fmt.Fprintf(yaml, "%s  \"audience\": \"%s\"\n", indent, auth.Audience)
	} else {
		fmt.Fprintf(yaml, "%s  \"type\": \"%s\"\n", indent, auth.Type)
	}
	fmt.Fprintf(yaml, "%s}%s\n", indent, mcpJSONComma(isLast))
}

func renderMCPConfigGuardPolicies(yaml *strings.Builder, toolName string, renderer MCPConfigRenderer, hasTrailingGuardPolicies bool) {
	if hasTrailingGuardPolicies {
		renderGuardPoliciesJSON(yaml, renderer.GuardPolicies, renderer.IndentLevel)
	} else if renderer.Format == "toml" && len(renderer.GuardPolicies) > 0 {
		renderGuardPoliciesToml(yaml, renderer.GuardPolicies, toolName)
	}
}

func mcpJSONComma(isLast bool) string {
	if isLast {
		return ""
	}
	return ","
}

// collectHTTPMCPHeaderSecrets collects all secrets from HTTP MCP tool headers
// Returns a map of environment variable names to their secret expressions
func collectHTTPMCPHeaderSecrets(tools map[string]any) map[string]string {
	allSecrets := make(map[string]string)

	for toolName, toolValue := range tools {
		// Check if this is an MCP tool configuration
		if toolConfig, ok := toolValue.(map[string]any); ok {
			if hasMcp, mcpType := hasMCPConfig(toolConfig); hasMcp && mcpType == "http" {
				// Extract MCP config to get headers
				if mcpConfig, err := getMCPConfig(toolConfig, toolName); err == nil {
					secrets := ExtractSecretsFromMap(mcpConfig.Headers)
					maps.Copy(allSecrets, secrets)
				}
			}
		}
	}

	return allSecrets
}

// getMCPConfig extracts MCP configuration from a tool config and returns a structured MCPServerConfig
func getMCPConfig(toolConfig map[string]any, toolName string) (*parser.RegistryMCPServerConfig, error) {
	mcpCustomLog.Printf("Extracting MCP config for tool: %s", toolName)

	config := MapToolConfig(toolConfig)
	result := &parser.RegistryMCPServerConfig{
		BaseMCPServerConfig: types.BaseMCPServerConfig{
			Env:     make(map[string]string),
			Headers: make(map[string]string),
		},
		Name: toolName,
	}

	if err := validateMCPConfigProperties(toolConfig, toolName); err != nil {
		return nil, err
	}
	if err := inferMCPConfigType(result, config, toolName); err != nil {
		return nil, err
	}
	extractMCPCommonFields(result, config)
	if err := extractMCPTypeSpecificFields(result, config, toolName); err != nil {
		return nil, err
	}
	if allowed, hasAllowed := config.GetStringArray("allowed"); hasAllowed {
		result.Allowed = allowed
	}
	finalizeStdioMCPConfig(result)
	return result, nil
}

func validateMCPConfigProperties(toolConfig map[string]any, toolName string) error {
	knownProperties := map[string]struct{}{
		"type": {}, "mode": {}, "command": {}, "container": {}, "version": {}, "args": {},
		"entrypoint": {}, "entrypointArgs": {}, "mounts": {}, "env": {}, "proxy-args": {},
		"url": {}, "headers": {}, "auth": {}, "registry": {}, "allowed": {}, "toolsets": {},
	}
	for key := range toolConfig {
		if !setutil.Contains(knownProperties, key) {
			mcpCustomLog.Printf("Unknown property '%s' in MCP config for tool '%s'", key, toolName)
			validProps := sliceutil.SortedKeys(knownProperties)
			return fmt.Errorf(
				"unknown property '%s' in MCP configuration for tool '%s'. Valid properties are: %s. "+
					"Example:\n"+
					"mcp-servers:\n"+
					"  %s:\n"+
					"    command: \"npx @my/tool\"\n"+
					"    args: [\"--port\", \"3000\"]",
				key, toolName, strings.Join(validProps, ", "), toolName)
		}
	}
	return nil
}

func inferMCPConfigType(result *parser.RegistryMCPServerConfig, config MapToolConfig, toolName string) error {
	if typeStr, hasType := config.GetString("type"); hasType {
		mcpCustomLog.Printf("MCP type explicitly set to: %s", typeStr)
		if typeStr == "local" {
			result.Type = "stdio"
		} else {
			result.Type = typeStr
		}
		return nil
	}

	mcpCustomLog.Print("No explicit MCP type, inferring from fields")
	if _, hasURL := config.GetString("url"); hasURL {
		result.Type = "http"
		mcpCustomLog.Printf("Inferred MCP type as http (has url field)")
	} else if _, hasCommand := config.GetString("command"); hasCommand {
		result.Type = "stdio"
		mcpCustomLog.Printf("Inferred MCP type as stdio (has command field)")
	} else if _, hasContainer := config.GetString("container"); hasContainer {
		result.Type = "stdio"
		mcpCustomLog.Printf("Inferred MCP type as stdio (has container field)")
	} else {
		return missingMCPTypeError(toolName)
	}
	return nil
}

func missingMCPTypeError(toolName string) error {
	mcpCustomLog.Printf("Unable to determine MCP type for tool '%s': missing type, url, command, or container", toolName)
	return fmt.Errorf(
		"unable to determine MCP type for tool '%s': missing type, url, command, or container. "+
			"Must specify one of: 'type' (stdio/http), 'url' (for HTTP MCP), 'command' (for command-based), or 'container' (for Docker-based). "+
			"Example:\n"+
			"mcp-servers:\n"+
			"  %s:\n"+
			"    command: \"npx @my/tool\"\n"+
			"    args: [\"--port\", \"3000\"]",
		toolName, toolName,
	)
}

func extractMCPCommonFields(result *parser.RegistryMCPServerConfig, config MapToolConfig) {
	if registry, hasRegistry := config.GetString("registry"); hasRegistry {
		result.Registry = registry
	}
}

func extractMCPTypeSpecificFields(result *parser.RegistryMCPServerConfig, config MapToolConfig, toolName string) error {
	mcpCustomLog.Printf("Extracting fields for MCP type: %s", result.Type)
	switch result.Type {
	case "stdio":
		extractStdioMCPFields(result, config)
		return nil
	case "http":
		return extractHTTPMCPFields(result, config, toolName)
	default:
		return unsupportedMCPTypeError(result.Type, toolName)
	}
}

func extractStdioMCPFields(result *parser.RegistryMCPServerConfig, config MapToolConfig) {
	if command, hasCommand := config.GetString("command"); hasCommand {
		result.Command = command
	}
	if container, hasContainer := config.GetString("container"); hasContainer {
		result.Container = container
	}
	if version, hasVersion := config.GetString("version"); hasVersion {
		result.Version = version
	}
	if args, hasArgs := config.GetStringArray("args"); hasArgs {
		result.Args = args
	}
	if entrypoint, hasEntrypoint := config.GetString("entrypoint"); hasEntrypoint {
		result.Entrypoint = entrypoint
	}
	if entrypointArgs, hasEntrypointArgs := config.GetStringArray("entrypointArgs"); hasEntrypointArgs {
		result.EntrypointArgs = entrypointArgs
	}
	if mounts, hasMounts := config.GetStringArray("mounts"); hasMounts {
		result.Mounts = mounts
	}
	if env, hasEnv := config.GetStringMap("env"); hasEnv {
		result.Env = env
	}
	if proxyArgs, hasProxyArgs := config.GetStringArray("proxy-args"); hasProxyArgs {
		result.ProxyArgs = proxyArgs
	}
}

func extractHTTPMCPFields(result *parser.RegistryMCPServerConfig, config MapToolConfig, toolName string) error {
	if url, hasURL := config.GetString("url"); hasURL {
		result.URL = url
	} else {
		return missingHTTPMCPURLError(toolName)
	}
	if headers, hasHeaders := config.GetStringMap("headers"); hasHeaders {
		result.Headers = headers
	}
	if authVal, hasAuth := config.GetAny("auth"); hasAuth {
		result.Auth = parseMCPAuthConfig(authVal)
	}
	return nil
}

func parseMCPAuthConfig(authVal any) *types.MCPAuthConfig {
	if authMap, ok := authVal.(map[string]any); ok {
		authConfig := &types.MCPAuthConfig{}
		if authType, ok := authMap["type"].(string); ok {
			authConfig.Type = authType
		}
		if audience, ok := authMap["audience"].(string); ok {
			authConfig.Audience = audience
		}
		if authConfig.Type != "" {
			return authConfig
		}
	}
	if authCfg, ok := authVal.(*types.MCPAuthConfig); ok {
		return authCfg
	}
	return nil
}

func missingHTTPMCPURLError(toolName string) error {
	mcpCustomLog.Printf("HTTP MCP tool '%s' missing required 'url' field", toolName)
	return fmt.Errorf(
		"http MCP tool '%s' missing required 'url' field. HTTP MCP servers must specify a URL endpoint. "+
			"Example:\n"+
			"mcp-servers:\n"+
			"  %s:\n"+
			"    type: http\n"+
			"    url: \"https://api.example.com/mcp\"\n"+
			"    headers:\n"+
			"      Authorization: \"****** secrets.API_KEY }}\"",
		toolName, toolName,
	)
}

func unsupportedMCPTypeError(mcpType string, toolName string) error {
	mcpCustomLog.Printf("Unsupported MCP type '%s' for tool '%s'", mcpType, toolName)
	return fmt.Errorf(
		"unsupported MCP type '%s' for tool '%s'. Valid types are: stdio, http. "+
			"Example:\n"+
			"mcp-servers:\n"+
			"  %s:\n"+
			"    type: stdio\n"+
			"    command: \"npx @my/tool\"\n"+
			"    args: [\"--port\", \"3000\"]",
		mcpType, toolName, toolName)
}

func finalizeStdioMCPConfig(result *parser.RegistryMCPServerConfig) {
	if result.Type == "stdio" && result.Container == "" && result.Command != "" {
		containerConfig := getWellKnownContainer(result.Command)
		if containerConfig != nil {
			mcpCustomLog.Printf("Auto-assigning container for command '%s': %s", result.Command, containerConfig.Image)
			result.Container = containerConfig.Image
			result.Entrypoint = containerConfig.Entrypoint
			result.EntrypointArgs = result.Args
			result.Args = nil
			result.Command = ""
		}
	}
	if result.Type == "stdio" && result.Container != "" && result.Version != "" {
		result.Container = result.Container + ":" + result.Version
		result.Version = ""
	}
}

// hasMCPConfig checks if a tool configuration has MCP configuration
func hasMCPConfig(toolConfig map[string]any) (bool, string) {
	// Check for direct type field
	if mcpType, hasType := toolConfig["type"]; hasType {
		if typeStr, ok := mcpType.(string); ok && parser.IsMCPType(typeStr) {
			// Normalize "local" to "stdio" for consistency
			if typeStr == "local" {
				return true, "stdio"
			}
			return true, typeStr
		}
	}

	// Infer type from presence of fields (same logic as getMCPConfig)
	if _, hasURL := toolConfig["url"]; hasURL {
		return true, "http"
	} else if _, hasCommand := toolConfig["command"]; hasCommand {
		return true, "stdio"
	} else if _, hasContainer := toolConfig["container"]; hasContainer {
		return true, "stdio"
	}

	return false, ""
}
