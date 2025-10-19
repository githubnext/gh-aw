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
		if toolName == "github" || toolName == "playwright" || toolName == "cache-memory" || toolName == "agentic-workflows" {
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
		if toolName == "github" || toolName == "playwright" || toolName == "cache-memory" || toolName == "agentic-workflows" {
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
		safeOutputConfig = generateSafeOutputsConfig(workflowData)
	}

	// Sort tools to ensure stable code generation
	sort.Strings(mcpTools)
	sort.Strings(proxyTools)

	// Collect all Docker images that will be used and generate download step
	dockerImages := collectDockerImages(tools)
	generateDownloadDockerImagesStep(yaml, dockerImages)

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

		// Start squid proxy ahead of time to avoid timeouts
		// Note: Docker images are now pre-downloaded in the "Downloading container images" step
		yaml.WriteString("      - name: Start Squid proxy\n")
		yaml.WriteString("        run: |\n")
		yaml.WriteString("          set -e\n")
		yaml.WriteString("          echo 'Starting squid-proxy services for proxy-enabled MCP tools...'\n")

		// Bring up squid service for each proxy-enabled tool
		for _, toolName := range proxyTools {
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

	// If no MCP tools, no configuration needed
	if len(mcpTools) == 0 {
		return
	}

	// Install gh-aw extension if agentic-workflows tool is enabled
	hasAgenticWorkflows := false
	for _, toolName := range mcpTools {
		if toolName == "agentic-workflows" {
			hasAgenticWorkflows = true
			break
		}
	}

	if hasAgenticWorkflows {
		yaml.WriteString("      - name: Install gh-aw extension\n")
		yaml.WriteString("        env:\n")
		yaml.WriteString("          GH_TOKEN: ${{ github.token }}\n")
		yaml.WriteString("        run: |\n")
		yaml.WriteString("          echo \"Installing gh-aw extension...\"\n")
		yaml.WriteString("          gh extension install githubnext/gh-aw\n")
		yaml.WriteString("          gh aw --version\n")
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

// hasGitHubTool checks if the tools map contains a GitHub tool
func hasGitHubTool(tools map[string]any) bool {
	_, exists := tools["github"]
	return exists
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

// getGitHubAllowedTools extracts the allowed tools list from GitHub tool configuration
// Returns the list of allowed tools, or nil if no allowed list is specified (which means all tools are allowed)
func getGitHubAllowedTools(githubTool any) []string {
	if toolConfig, ok := githubTool.(map[string]any); ok {
		if allowedSetting, exists := toolConfig["allowed"]; exists {
			// Handle array format
			switch v := allowedSetting.(type) {
			case []any:
				// Convert array to string slice
				tools := make([]string, 0, len(v))
				for _, item := range v {
					if str, ok := item.(string); ok {
						tools = append(tools, str)
					}
				}
				return tools
			case []string:
				return v
			}
		}
	}
	return nil
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
