package workflow

import "github.com/github/gh-aw/pkg/logger"

var frontmatterExtractionSecurityLog = logger.New("workflow:frontmatter_extraction_security")

// extractNetworkPermissions extracts network permissions from frontmatter
func (c *Compiler) extractNetworkPermissions(frontmatter map[string]any) *NetworkPermissions {
	frontmatterExtractionSecurityLog.Print("Extracting network permissions from frontmatter")

	if network, exists := frontmatter["network"]; exists {
		// Handle string format: "defaults"
		if networkStr, ok := network.(string); ok {
			frontmatterExtractionSecurityLog.Printf("Network permissions string format: %s", networkStr)
			if networkStr == "defaults" {
				return &NetworkPermissions{
					Allowed:           []string{"defaults"},
					ExplicitlyDefined: true,
				}
			}
			// Unknown string format, return nil
			frontmatterExtractionSecurityLog.Printf("Unknown network string format: %s", networkStr)
			return nil
		}

		// Handle object format: { allowed: [...], blocked: [...] } or {}
		if networkObj, ok := network.(map[string]any); ok {
			frontmatterExtractionSecurityLog.Printf("Network permissions object format with %d fields", len(networkObj))
			permissions := &NetworkPermissions{
				ExplicitlyDefined: true,
			}

			// Extract allowed domains if present
			if allowed, hasAllowed := networkObj["allowed"]; hasAllowed {
				if allowedSlice, ok := allowed.([]any); ok {
					for _, domain := range allowedSlice {
						if domainStr, ok := domain.(string); ok {
							permissions.Allowed = append(permissions.Allowed, domainStr)
						}
					}
					frontmatterExtractionSecurityLog.Printf("Extracted %d allowed domains", len(permissions.Allowed))
				}
			}

			if allowedInput, hasAllowedInput := networkObj["allowed-input"]; hasAllowedInput {
				if allowedInputBool, ok := allowedInput.(bool); ok {
					permissions.AllowedInput = allowedInputBool
				}
			}

			// Extract blocked domains if present
			if blocked, hasBlocked := networkObj["blocked"]; hasBlocked {
				if blockedSlice, ok := blocked.([]any); ok {
					for _, domain := range blockedSlice {
						if domainStr, ok := domain.(string); ok {
							permissions.Blocked = append(permissions.Blocked, domainStr)
						}
					}
					frontmatterExtractionSecurityLog.Printf("Extracted %d blocked domains", len(permissions.Blocked))
				}
			}

			// Empty object {} means no network access (empty allowed list)
			return permissions
		}
	}
	frontmatterExtractionSecurityLog.Print("No network permissions found in frontmatter")
	return nil
}

// extractSandboxConfig extracts sandbox configuration from front matter
func (c *Compiler) extractSandboxConfig(frontmatter map[string]any) *SandboxConfig {
	frontmatterExtractionSecurityLog.Print("Extracting sandbox configuration from frontmatter")

	sandbox, exists := frontmatter["sandbox"]
	if !exists {
		frontmatterExtractionSecurityLog.Print("No sandbox configuration found")
		return nil
	}

	// Handle boolean format: sandbox: false (NO LONGER SUPPORTED)
	// This format has been removed - only sandbox.agent: false is supported
	if _, ok := sandbox.(bool); ok {
		frontmatterExtractionSecurityLog.Print("Top-level sandbox: false is no longer supported")
		// Return nil to trigger schema validation error
		return nil
	}

	// Handle legacy string format: "default" or "awf" (legacy srt/sandbox-runtime are auto-migrated)
	if sandboxStr, ok := sandbox.(string); ok {
		frontmatterExtractionSecurityLog.Printf("Sandbox string format: type=%s", sandboxStr)
		sandboxType := SandboxType(sandboxStr)
		if isSupportedSandboxType(sandboxType) {
			return &SandboxConfig{
				Type: sandboxType,
			}
		}
		// Unknown string format, return nil
		frontmatterExtractionSecurityLog.Printf("Unsupported sandbox type: %s", sandboxStr)
		return nil
	}

	// Handle object format
	sandboxObj, ok := sandbox.(map[string]any)
	if !ok {
		return nil
	}

	frontmatterExtractionSecurityLog.Printf("Sandbox object format with %d fields", len(sandboxObj))

	return c.extractSandboxObjectConfig(sandboxObj)
}

func (c *Compiler) extractSandboxObjectConfig(sandboxObj map[string]any) *SandboxConfig {
	config := &SandboxConfig{}
	if agentVal, hasAgent := sandboxObj["agent"]; hasAgent {
		frontmatterExtractionSecurityLog.Print("Extracting agent sandbox configuration")
		config.Agent = c.extractAgentSandboxConfig(agentVal)
	}
	if mcpVal, hasMCP := sandboxObj["mcp"]; hasMCP {
		frontmatterExtractionSecurityLog.Print("Extracting MCP gateway configuration")
		config.MCP = c.extractMCPGatewayConfig(mcpVal)
	}
	if config.Agent != nil {
		frontmatterExtractionSecurityLog.Print("Sandbox configured with new format (agent)")
		return config
	}
	if typeVal, hasType := sandboxObj["type"]; hasType {
		if typeStr, ok := typeVal.(string); ok {
			config.Type = SandboxType(typeStr)
		}
	}
	if configVal, hasConfig := sandboxObj["config"]; hasConfig {
		config.Config = c.extractSRTConfig(configVal)
	}
	return config
}

// extractAgentSandboxConfig extracts agent sandbox configuration
func (c *Compiler) extractAgentSandboxConfig(agentVal any) *AgentSandboxConfig {
	if config, handled := extractAgentSandboxPrimitive(agentVal); handled {
		return config
	}

	agentObj, ok := agentVal.(map[string]any)
	if !ok {
		return nil
	}

	agentConfig := &AgentSandboxConfig{NetworkIsolation: true}
	c.extractAgentSandboxIdentity(agentConfig, agentObj)
	c.extractAgentSandboxExecution(agentConfig, agentObj)
	c.extractAgentSandboxRuntime(agentConfig, agentObj)
	c.extractAgentSandboxModelFallback(agentConfig, agentObj)
	return agentConfig
}

func extractAgentSandboxPrimitive(agentVal any) (*AgentSandboxConfig, bool) {
	if agentBool, ok := agentVal.(bool); ok {
		if !agentBool {
			frontmatterExtractionSecurityLog.Print("Agent sandbox explicitly disabled with agent: false")
			return &AgentSandboxConfig{
				Disabled: true,
			}, true
		}
		frontmatterExtractionSecurityLog.Print("Agent: true specified but has no effect, treating as unconfigured")
		return nil, true
	}
	if agentStr, ok := agentVal.(string); ok {
		agentType := SandboxType(agentStr)
		if isSupportedSandboxType(agentType) {
			return &AgentSandboxConfig{
				Type: agentType,
			}, true
		}
		return nil, true
	}
	return nil, false
}

func (c *Compiler) extractAgentSandboxIdentity(agentConfig *AgentSandboxConfig, agentObj map[string]any) {
	if idVal, hasID := agentObj["id"]; hasID {
		if idStr, ok := idVal.(string); ok {
			agentConfig.ID = idStr
		}
	}
	if typeVal, hasType := agentObj["type"]; hasType {
		if typeStr, ok := typeVal.(string); ok {
			agentConfig.Type = SandboxType(typeStr)
		}
	}
	if versionVal, hasVersion := agentObj["version"]; hasVersion {
		if versionStr, ok := versionVal.(string); ok {
			agentConfig.Version = versionStr
		}
	}
	if platformVal, hasPlatform := agentObj["platform"]; hasPlatform {
		if platformStr, ok := platformVal.(string); ok {
			agentConfig.Platform = platformStr
		}
	}
}

func (c *Compiler) extractAgentSandboxExecution(agentConfig *AgentSandboxConfig, agentObj map[string]any) {
	if sudoVal, hasSudo := agentObj["sudo"]; hasSudo {
		if sudoBool, ok := sudoVal.(bool); ok {
			agentConfig.NetworkIsolation = !sudoBool
			if sudoBool {
				agentConfig.SudoExplicitlyEnabled = true
			}
		}
	}
	if configVal, hasConfig := agentObj["config"]; hasConfig {
		agentConfig.Config = c.extractSRTConfig(configVal)
	}
	if commandVal, hasCommand := agentObj["command"]; hasCommand {
		if commandStr, ok := commandVal.(string); ok {
			agentConfig.Command = commandStr
		}
	}
	agentConfig.Args = append(agentConfig.Args, stringSliceFromMap(agentObj, "args")...)
	if env, ok := stringMapFromMap(agentObj, "env"); ok {
		agentConfig.Env = env
	}
	agentConfig.Mounts = append(agentConfig.Mounts, stringSliceFromMap(agentObj, "mounts")...)
}

func (c *Compiler) extractAgentSandboxRuntime(agentConfig *AgentSandboxConfig, agentObj map[string]any) {
	if runtimeVal, hasRuntime := agentObj["runtime"]; hasRuntime {
		if runtimeStr, ok := runtimeVal.(string); ok {
			agentConfig.Runtime = AgentRuntime(runtimeStr)
			frontmatterExtractionSecurityLog.Printf("Extracted sandbox.agent.runtime: %s", runtimeStr)
		}
	}
	if legacyVal, hasLegacy := agentObj["legacy-security"]; hasLegacy {
		if legacyStr, ok := legacyVal.(string); ok && legacyStr == "enable" {
			agentConfig.LegacySecurity = true
			frontmatterExtractionSecurityLog.Print("Extracted sandbox.agent.legacy-security: enable")
		}
	}
}

func (c *Compiler) extractAgentSandboxModelFallback(agentConfig *AgentSandboxConfig, agentObj map[string]any) {
	if mfVal, hasMF := agentObj["model-fallback"]; hasMF {
		switch v := mfVal.(type) {
		case bool:
			value := TemplatableBool("false")
			if v {
				value = TemplatableBool("true")
			}
			agentConfig.ModelFallback = &value
			frontmatterExtractionSecurityLog.Printf("Extracted sandbox.agent.model-fallback")
		case string:
			if isExpression(v) {
				value := TemplatableBool(v)
				agentConfig.ModelFallback = &value
				frontmatterExtractionSecurityLog.Printf("Extracted sandbox.agent.model-fallback")
			}
		}
	}
}

// extractMCPGatewayConfig extracts MCP gateway configuration from frontmatter
// Per MCP Gateway Specification v1.0.0: Only container-based execution is supported.
// Direct command execution is not supported.
func (c *Compiler) extractMCPGatewayConfig(mcpVal any) *MCPGatewayRuntimeConfig {
	// Handle nil or boolean false
	if mcpVal == nil {
		return nil
	}
	if mcpBool, ok := mcpVal.(bool); ok && !mcpBool {
		return nil
	}

	// Handle object format: { container: "...", port: ..., args: [...], env: {...} }
	mcpObj, ok := mcpVal.(map[string]any)
	if !ok {
		frontmatterExtractionSecurityLog.Printf("MCP gateway configuration is not an object: %T", mcpVal)
		return nil
	}

	mcpConfig := &MCPGatewayRuntimeConfig{}
	c.extractMCPGatewayCoreConfig(mcpConfig, mcpObj)
	c.extractMCPGatewayListConfig(mcpConfig, mcpObj)
	c.extractMCPGatewayPayloadConfig(mcpConfig, mcpObj)
	c.extractMCPGatewayTrustedAndKeepaliveConfig(mcpConfig, mcpObj)
	return mcpConfig
}

func (c *Compiler) extractMCPGatewayCoreConfig(mcpConfig *MCPGatewayRuntimeConfig, mcpObj map[string]any) {
	if containerVal, hasContainer := mcpObj["container"]; hasContainer {
		if containerStr, ok := containerVal.(string); ok {
			mcpConfig.Container = containerStr
		}
	}
	if versionVal, hasVersion := mcpObj["version"]; hasVersion {
		if versionStr, ok := versionVal.(string); ok {
			mcpConfig.Version = versionStr
		}
	}
	if entrypointVal, hasEntrypoint := mcpObj["entrypoint"]; hasEntrypoint {
		if entrypointStr, ok := entrypointVal.(string); ok {
			mcpConfig.Entrypoint = entrypointStr
		}
	}
	if port, ok := intFromMap(mcpObj, "port"); ok {
		mcpConfig.Port = port
	}
	if apiKeyVal, hasAPIKey := mcpObj["api-key"]; hasAPIKey {
		if apiKeyStr, ok := apiKeyVal.(string); ok {
			mcpConfig.APIKey = apiKeyStr
		}
	}
	if domainVal, hasDomain := mcpObj["domain"]; hasDomain {
		if domainStr, ok := domainVal.(string); ok {
			mcpConfig.Domain = domainStr
		}
	}
}

func (c *Compiler) extractMCPGatewayListConfig(mcpConfig *MCPGatewayRuntimeConfig, mcpObj map[string]any) {
	mcpConfig.Args = append(mcpConfig.Args, stringSliceFromMap(mcpObj, "args")...)
	mcpConfig.EntrypointArgs = append(mcpConfig.EntrypointArgs, stringSliceFromMap(mcpObj, "entrypointArgs")...)
	if env, ok := stringMapFromMap(mcpObj, "env"); ok {
		mcpConfig.Env = env
	}
	mcpConfig.Mounts = append(mcpConfig.Mounts, stringSliceFromMap(mcpObj, "mounts")...)
}

func (c *Compiler) extractMCPGatewayPayloadConfig(mcpConfig *MCPGatewayRuntimeConfig, mcpObj map[string]any) {
	for _, key := range []string{"payloadDir", "payload-dir"} {
		if payloadDirVal, hasPayloadDir := mcpObj[key]; hasPayloadDir {
			if payloadDirStr, ok := payloadDirVal.(string); ok {
				mcpConfig.PayloadDir = payloadDirStr
				break
			}
		}
	}
	for _, key := range []string{"payloadPathPrefix", "payload-path-prefix"} {
		if payloadPathPrefixVal, hasPayloadPathPrefix := mcpObj[key]; hasPayloadPathPrefix {
			if payloadPathPrefixStr, ok := payloadPathPrefixVal.(string); ok {
				mcpConfig.PayloadPathPrefix = payloadPathPrefixStr
				break
			}
		}
	}
	for _, key := range []string{"payloadSizeThreshold", "payload-size-threshold"} {
		if payloadSizeThreshold, ok := intFromMap(mcpObj, key); ok {
			mcpConfig.PayloadSizeThreshold = payloadSizeThreshold
			if mcpConfig.PayloadSizeThreshold != 0 {
				break
			}
		}
	}
}

func (c *Compiler) extractMCPGatewayTrustedAndKeepaliveConfig(mcpConfig *MCPGatewayRuntimeConfig, mcpObj map[string]any) {
	for _, key := range []string{"trustedBots", "trusted-bots"} {
		if _, hasTrustedBots := mcpObj[key]; hasTrustedBots {
			mcpConfig.TrustedBots = append(mcpConfig.TrustedBots, stringSliceFromMap(mcpObj, key)...)
			if len(mcpConfig.TrustedBots) > 0 {
				break
			}
		}
	}
	for _, key := range []string{"keepaliveInterval", "keepalive-interval"} {
		if keepalive, hasKeepalive := intFromMap(mcpObj, key); hasKeepalive {
			mcpConfig.KeepaliveInterval = keepalive
			break
		}
		if _, hasKeepalive := mcpObj[key]; hasKeepalive {
			break
		}
	}
}

// extractSRTConfig extracts Sandbox Runtime configuration from a map
func (c *Compiler) extractSRTConfig(configVal any) *SandboxRuntimeConfig {
	configObj, ok := configVal.(map[string]any)
	if !ok {
		return nil
	}

	srtConfig := &SandboxRuntimeConfig{}

	if networkVal, hasNetwork := configObj["network"]; hasNetwork {
		if networkObj, ok := networkVal.(map[string]any); ok {
			srtConfig.Network = extractSRTNetworkConfig(networkObj)
		}
	}

	if filesystemVal, hasFilesystem := configObj["filesystem"]; hasFilesystem {
		if filesystemObj, ok := filesystemVal.(map[string]any); ok {
			srtConfig.Filesystem = extractSRTFilesystemConfig(filesystemObj)
		}
	}

	if ignoreViolations, hasIgnoreViolations := configObj["ignoreViolations"]; hasIgnoreViolations {
		if violationsObj, ok := ignoreViolations.(map[string]any); ok {
			srtConfig.IgnoreViolations = stringSliceMapFromAnyMap(violationsObj)
		}
	}

	if enableWeakerNestedSandbox, hasEnableWeaker := configObj["enableWeakerNestedSandbox"]; hasEnableWeaker {
		if weakerBool, ok := enableWeakerNestedSandbox.(bool); ok {
			srtConfig.EnableWeakerNestedSandbox = weakerBool
		}
	}

	return srtConfig
}

func extractSRTNetworkConfig(networkObj map[string]any) *SRTNetworkConfig {
	netConfig := &SRTNetworkConfig{}
	netConfig.AllowedDomains = append(netConfig.AllowedDomains, stringSliceFromMap(networkObj, "allowedDomains")...)
	netConfig.BlockedDomains = append(netConfig.BlockedDomains, stringSliceFromMap(networkObj, "blockedDomains")...)
	netConfig.AllowUnixSockets = append(netConfig.AllowUnixSockets, stringSliceFromMap(networkObj, "allowUnixSockets")...)
	if bindingBool, ok := boolFromMap(networkObj, "allowLocalBinding"); ok {
		netConfig.AllowLocalBinding = bindingBool
	}
	if unixSocketsBool, ok := boolFromMap(networkObj, "allowAllUnixSockets"); ok {
		netConfig.AllowAllUnixSockets = unixSocketsBool
	}
	return netConfig
}

func extractSRTFilesystemConfig(filesystemObj map[string]any) *SRTFilesystemConfig {
	fsConfig := &SRTFilesystemConfig{}
	if paths, ok := stringSliceFromAny(filesystemObj["denyRead"]); ok {
		fsConfig.DenyRead = paths
	}
	fsConfig.AllowWrite = append(fsConfig.AllowWrite, stringSliceFromMap(filesystemObj, "allowWrite")...)
	if paths, ok := stringSliceFromAny(filesystemObj["denyWrite"]); ok {
		fsConfig.DenyWrite = paths
	}
	return fsConfig
}

func stringSliceMapFromAnyMap(values map[string]any) map[string][]string {
	result := make(map[string][]string)
	for key, value := range values {
		values, ok := value.([]any)
		if !ok {
			continue
		}
		var paths []string
		for _, path := range values {
			if pathStr, ok := path.(string); ok {
				paths = append(paths, pathStr)
			}
		}
		result[key] = paths
	}
	return result
}

func stringSliceFromMap(values map[string]any, key string) []string {
	slice, _ := stringSliceFromAny(values[key])
	return slice
}

func stringSliceFromAny(value any) ([]string, bool) {
	values, ok := value.([]any)
	if !ok {
		return nil, false
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if valueStr, ok := value.(string); ok {
			result = append(result, valueStr)
		}
	}
	return result, true
}

func stringMapFromMap(values map[string]any, key string) (map[string]string, bool) {
	valueMap, ok := values[key].(map[string]any)
	if !ok {
		return nil, false
	}
	result := make(map[string]string)
	for key, value := range valueMap {
		if valueStr, ok := value.(string); ok {
			result[key] = valueStr
		}
	}
	return result, true
}

func intFromMap(values map[string]any, key string) (int, bool) {
	switch v := values[key].(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case uint:
		return int(v), true
	case uint64:
		return int(v), true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

func boolFromMap(values map[string]any, key string) (bool, bool) {
	value, ok := values[key].(bool)
	return value, ok
}
