package workflow

import (
	"fmt"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/parser"
)

// hasMCPServers checks if the workflow has any MCP servers configured
func HasMCPServers(workflowData *WorkflowData) bool {
	if workflowData == nil {
		return false
	}

	// Check for standard MCP tools
	for toolName, toolValue := range workflowData.Tools {
		if toolName == "github" || toolName == "playwright" || toolName == "cache-memory" {
			return true
		}
		// Check for custom MCP tools
		if mcpConfig, ok := toolValue.(map[string]any); ok {
			if hasMcp, _ := hasMCPConfig(mcpConfig); hasMcp {
				return true
			}
		}
	}

	// Check if safe-outputs is enabled (adds safe-outputs MCP server)
	if HasSafeOutputsEnabled(workflowData.SafeOutputs) {
		return true
	}

	return false
}

// generateMCPSetup generates the MCP server configuration setup
func (c *Compiler) generateMCPSetup(yaml *strings.Builder, tools map[string]any, engine CodingAgentEngine, workflowData *WorkflowData) {
	// Collect tools that need MCP server configuration
	var mcpTools []string
	var proxyTools []string

	// Check if workflowData is valid before accessing its fields
	if workflowData == nil {
		return
	}

	workflowTools := workflowData.Tools

	for toolName, toolValue := range workflowTools {
		// Standard MCP tools
		if toolName == "github" || toolName == "playwright" || toolName == "cache-memory" {
			mcpTools = append(mcpTools, toolName)
		} else if mcpConfig, ok := toolValue.(map[string]any); ok {
			// Check if it's explicitly marked as MCP type in the new format
			if hasMcp, _ := hasMCPConfig(mcpConfig); hasMcp {
				mcpTools = append(mcpTools, toolName)

				// Check if this tool needs proxy
				if needsProxySetup, _ := needsProxy(mcpConfig); needsProxySetup {
					proxyTools = append(proxyTools, toolName)
				}
			}
		}
	}

	// Check if safe-outputs is enabled and add to MCP tools
	if HasSafeOutputsEnabled(workflowData.SafeOutputs) {
		mcpTools = append(mcpTools, "safe-outputs")
	}

	// Generate safe-outputs configuration once to avoid duplicate computation
	var safeOutputConfig string
	if HasSafeOutputsEnabled(workflowData.SafeOutputs) {
		safeOutputConfig = c.generateSafeOutputsConfig(workflowData)
	}

	// Sort tools to ensure stable code generation
	sort.Strings(mcpTools)
	sort.Strings(proxyTools)

	// Generate proxy configuration files inline for proxy-enabled tools
	// These files will be used automatically by docker compose when MCP tools run
	if len(proxyTools) > 0 {
		yaml.WriteString("      - name: Setup Proxy Configuration for MCP Network Restrictions\n")
		yaml.WriteString("        run: |\n")
		yaml.WriteString("          echo \"Generating proxy configuration files for MCP tools with network restrictions...\"\n")
		yaml.WriteString("          \n")

		// Generate proxy configurations inline for each proxy-enabled tool
		for _, toolName := range proxyTools {
			if toolConfig, ok := tools[toolName].(map[string]any); ok {
				c.generateInlineProxyConfig(yaml, toolName, toolConfig)
			}
		}

		yaml.WriteString("          echo \"Proxy configuration files generated.\"\n")

		// Pre-pull images and start squid proxy ahead of time to avoid timeouts
		yaml.WriteString("      - name: Pre-pull images and start Squid proxy\n")
		yaml.WriteString("        run: |\n")
		yaml.WriteString("          set -e\n")
		yaml.WriteString("          echo 'Pre-pulling Docker images for proxy-enabled MCP tools...'\n")
		yaml.WriteString("          docker pull ubuntu/squid:latest\n")

		// Pull each tool's container image if specified, and bring up squid service
		for _, toolName := range proxyTools {
			if toolConfig, ok := tools[toolName].(map[string]any); ok {
				if mcpConf, err := getMCPConfig(toolConfig, toolName); err == nil && mcpConf.Container != "" {
					fmt.Fprintf(yaml, "          echo 'Pulling %s for tool %s'\n", mcpConf.Container, toolName)
					fmt.Fprintf(yaml, "          docker pull %s\n", mcpConf.Container)
				}
				fmt.Fprintf(yaml, "          echo 'Starting squid-proxy service for %s'\n", toolName)
				fmt.Fprintf(yaml, "          docker compose -f docker-compose-%s.yml up -d squid-proxy\n", toolName)

				// Enforce that egress from this tool's network can only reach the Squid proxy
				subnetCIDR, squidIP, _ := computeProxyNetworkParams(toolName)
				fmt.Fprintf(yaml, "          echo 'Enforcing egress to proxy for %s (subnet %s, squid %s)'\n", toolName, subnetCIDR, squidIP)
				yaml.WriteString("          if command -v sudo >/dev/null 2>&1; then SUDO=sudo; else SUDO=; fi\n")
				// Accept established/related connections first (position 1)
				yaml.WriteString("          $SUDO iptables -C DOCKER-USER -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT 2>/dev/null || $SUDO iptables -I DOCKER-USER 1 -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT\n")
				// Accept all egress from Squid IP (position 2)
				fmt.Fprintf(yaml, "          $SUDO iptables -C DOCKER-USER -s %s -j ACCEPT 2>/dev/null || $SUDO iptables -I DOCKER-USER 2 -s %s -j ACCEPT\n", squidIP, squidIP)
				// Allow traffic to squid:3128 from the subnet (position 3)
				fmt.Fprintf(yaml, "          $SUDO iptables -C DOCKER-USER -s %s -d %s -p tcp --dport 3128 -j ACCEPT 2>/dev/null || $SUDO iptables -I DOCKER-USER 3 -s %s -d %s -p tcp --dport 3128 -j ACCEPT\n", subnetCIDR, squidIP, subnetCIDR, squidIP)
				// Then reject all other egress from that subnet (append to end)
				fmt.Fprintf(yaml, "          $SUDO iptables -C DOCKER-USER -s %s -j REJECT 2>/dev/null || $SUDO iptables -A DOCKER-USER -s %s -j REJECT\n", subnetCIDR, subnetCIDR)
			}
		}
	}

	// If no MCP tools, no configuration needed
	if len(mcpTools) == 0 {
		return
	}

	// Write safe-outputs MCP server if enabled
	if HasSafeOutputsEnabled(workflowData.SafeOutputs) {
		yaml.WriteString("      - name: Setup Safe Outputs Collector MCP\n")
		yaml.WriteString("        run: |\n")
		yaml.WriteString("          mkdir -p /tmp/gh-aw/safe-outputs\n")

		// Write the safe-outputs configuration to config.json
		if safeOutputConfig != "" {
			yaml.WriteString("          cat > /tmp/gh-aw/safe-outputs/config.json << 'EOF'\n")
			yaml.WriteString("          " + safeOutputConfig + "\n")
			yaml.WriteString("          EOF\n")
		}

		yaml.WriteString("          cat > /tmp/gh-aw/safe-outputs/mcp-server.cjs << 'EOF'\n")
		// Embed the safe-outputs MCP server script
		for _, line := range FormatJavaScriptForYAML(safeOutputsMCPServerScript) {
			yaml.WriteString(line)
		}
		yaml.WriteString("          EOF\n")
		yaml.WriteString("          chmod +x /tmp/gh-aw/safe-outputs/mcp-server.cjs\n")
		yaml.WriteString("          \n")
	}

	// Use the engine's RenderMCPConfig method
	yaml.WriteString("      - name: Setup MCPs\n")
	if HasSafeOutputsEnabled(workflowData.SafeOutputs) {
		if safeOutputConfig != "" {
			// Generate filtered tools JSON
			filteredToolsJSON, err := c.GenerateFilteredToolsJSON(workflowData)
			if err != nil {
				// Log error but continue with empty tools
				filteredToolsJSON = "{}"
			}

			// Add environment variables for JSONL validation
			yaml.WriteString("        env:\n")
			fmt.Fprintf(yaml, "          GITHUB_AW_SAFE_OUTPUTS: ${{ env.GITHUB_AW_SAFE_OUTPUTS }}\n")
			fmt.Fprintf(yaml, "          GITHUB_AW_SAFE_OUTPUTS_CONFIG: %q\n", safeOutputConfig)
			fmt.Fprintf(yaml, "          GITHUB_AW_SAFE_OUTPUTS_TOOLS: %q\n", filteredToolsJSON)
			if workflowData.SafeOutputs != nil && workflowData.SafeOutputs.UploadAssets != nil {
				fmt.Fprintf(yaml, "          GITHUB_AW_ASSETS_BRANCH: %q\n", workflowData.SafeOutputs.UploadAssets.BranchName)
				fmt.Fprintf(yaml, "          GITHUB_AW_ASSETS_MAX_SIZE_KB: %d\n", workflowData.SafeOutputs.UploadAssets.MaxSizeKB)
				fmt.Fprintf(yaml, "          GITHUB_AW_ASSETS_ALLOWED_EXTS: %q\n", strings.Join(workflowData.SafeOutputs.UploadAssets.AllowedExts, ","))
			}
		}
	}
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          mkdir -p /tmp/gh-aw/mcp-config\n")
	engine.RenderMCPConfig(yaml, tools, mcpTools, workflowData)
}

func getGitHubDockerImageVersion(githubTool any) string {
	githubDockerImageVersion := constants.DefaultGitHubMCPServerVersion // Default Docker image version
	// Extract version setting from tool properties
	if toolConfig, ok := githubTool.(map[string]any); ok {
		if versionSetting, exists := toolConfig["version"]; exists {
			if stringValue, ok := versionSetting.(string); ok {
				githubDockerImageVersion = stringValue
			}
		}
	}
	return githubDockerImageVersion
}

// getGitHubType extracts the mode from GitHub tool configuration (local or remote)
func getGitHubType(githubTool any) string {
	if toolConfig, ok := githubTool.(map[string]any); ok {
		if modeSetting, exists := toolConfig["mode"]; exists {
			if stringValue, ok := modeSetting.(string); ok {
				return stringValue
			}
		}
	}
	return "local" // default to local (Docker)
}

// getGitHubToken extracts the custom github-token from GitHub tool configuration
func getGitHubToken(githubTool any) string {
	if toolConfig, ok := githubTool.(map[string]any); ok {
		if tokenSetting, exists := toolConfig["github-token"]; exists {
			if stringValue, ok := tokenSetting.(string); ok {
				return stringValue
			}
		}
	}
	return ""
}

// getGitHubReadOnly checks if read-only mode is enabled for GitHub tool
func getGitHubReadOnly(githubTool any) bool {
	if toolConfig, ok := githubTool.(map[string]any); ok {
		if readOnlySetting, exists := toolConfig["read-only"]; exists {
			if boolValue, ok := readOnlySetting.(bool); ok {
				return boolValue
			}
		}
	}
	return false
}

// getGitHubToolsets extracts the toolsets configuration from GitHub tool
func getGitHubToolsets(githubTool any) string {
	if toolConfig, ok := githubTool.(map[string]any); ok {
		if toolsetsSetting, exists := toolConfig["toolset"]; exists {
			// Handle array format only
			switch v := toolsetsSetting.(type) {
			case []any:
				// Convert array to comma-separated string
				toolsets := make([]string, 0, len(v))
				for _, item := range v {
					if str, ok := item.(string); ok {
						toolsets = append(toolsets, str)
					}
				}
				return strings.Join(toolsets, ",")
			case []string:
				return strings.Join(v, ",")
			}
		}
	}
	return ""
}

func getPlaywrightDockerImageVersion(playwrightTool any) string {
	playwrightDockerImageVersion := "latest" // Default Playwright Docker image version
	// Extract version setting from tool properties
	if toolConfig, ok := playwrightTool.(map[string]any); ok {
		if versionSetting, exists := toolConfig["version"]; exists {
			if stringValue, ok := versionSetting.(string); ok {
				playwrightDockerImageVersion = stringValue
			}
		}
	}
	return playwrightDockerImageVersion
}

// generatePlaywrightAllowedDomains extracts domain list from Playwright tool configuration with bundle resolution
// Uses the same domain bundle resolution as top-level network configuration, defaulting to localhost only
func generatePlaywrightAllowedDomains(playwrightTool any) []string {
	// Default to localhost with all port variations (same as Copilot agent default)
	allowedDomains := constants.DefaultAllowedDomains

	// Extract allowed_domains from Playwright tool configuration
	if toolConfig, ok := playwrightTool.(map[string]any); ok {
		if domainsConfig, exists := toolConfig["allowed_domains"]; exists {
			// Create a mock NetworkPermissions structure to use the same domain resolution logic
			playwrightNetwork := &NetworkPermissions{}

			switch domains := domainsConfig.(type) {
			case []string:
				playwrightNetwork.Allowed = domains
			case []any:
				// Convert []any to []string
				allowedDomainsSlice := make([]string, len(domains))
				for i, domain := range domains {
					if domainStr, ok := domain.(string); ok {
						allowedDomainsSlice[i] = domainStr
					}
				}
				playwrightNetwork.Allowed = allowedDomainsSlice
			case string:
				// Single domain as string
				playwrightNetwork.Allowed = []string{domains}
			}

			// Use the same domain bundle resolution as the top-level network configuration
			resolvedDomains := GetAllowedDomains(playwrightNetwork)

			// Ensure localhost domains are always included
			allowedDomains = parser.EnsureLocalhostDomains(resolvedDomains)
		}
	}

	return allowedDomains
}

// PlaywrightDockerArgs represents the common Docker arguments for Playwright container
type PlaywrightDockerArgs struct {
	ImageVersion   string
	AllowedDomains []string
}

// generatePlaywrightDockerArgs creates the common Docker arguments for Playwright MCP server
func generatePlaywrightDockerArgs(playwrightTool any) PlaywrightDockerArgs {
	return PlaywrightDockerArgs{
		ImageVersion:   getPlaywrightDockerImageVersion(playwrightTool),
		AllowedDomains: generatePlaywrightAllowedDomains(playwrightTool),
	}
}
