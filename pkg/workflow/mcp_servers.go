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

	// Collect all Docker images that will be used and generate download step
	dockerImages := collectDockerImages(tools)
	generateDownloadDockerImagesStep(yaml, dockerImages)

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
		// Use effective token with precedence: top-level github-token > default
		effectiveToken := getEffectiveGitHubToken("", workflowData.GitHubToken)

		yaml.WriteString("      - name: Install gh-aw extension\n")
		yaml.WriteString("        env:\n")
		yaml.WriteString(fmt.Sprintf("          GH_TOKEN: %s\n", effectiveToken))
		yaml.WriteString("        run: |\n")
		yaml.WriteString("          echo \"Installing gh-aw extension...\"\n")
		yaml.WriteString("          gh extension install githubnext/gh-aw\n")
		yaml.WriteString("          gh aw --version\n")
	}

	// Write safe-outputs MCP server if enabled
	if HasSafeOutputsEnabled(workflowData.SafeOutputs) {
		yaml.WriteString("      - name: Setup Safe Outputs Collector MCP\n")
		yaml.WriteString("        run: |\n")
		yaml.WriteString("          mkdir -p /tmp/gh-aw/safeoutputs\n")

		// Write the safe-outputs configuration to config.json
		if safeOutputConfig != "" {
			yaml.WriteString("          cat > /tmp/gh-aw/safeoutputs/config.json << 'EOF'\n")
			yaml.WriteString("          " + safeOutputConfig + "\n")
			yaml.WriteString("          EOF\n")
		}

		yaml.WriteString("          cat > /tmp/gh-aw/safeoutputs/mcp-server.cjs << 'EOF'\n")
		// Embed the safe-outputs MCP server script
		for _, line := range FormatJavaScriptForYAML(safeOutputsMCPServerScript) {
			yaml.WriteString(line)
		}
		yaml.WriteString("          EOF\n")
		yaml.WriteString("          chmod +x /tmp/gh-aw/safeoutputs/mcp-server.cjs\n")
		yaml.WriteString("          \n")
	}

	// Use the engine's RenderMCPConfig method
	yaml.WriteString("      - name: Setup MCPs\n")

	// Add env block for environment variables to prevent template injection
	needsEnvBlock := false
	hasGitHub := false
	hasSafeOutputs := false
	hasPlaywright := false
	var playwrightAllowedDomainsSecrets map[string]string
	// Note: hasAgenticWorkflows is already declared earlier in this function

	for _, toolName := range mcpTools {
		if toolName == "github" {
			hasGitHub = true
			needsEnvBlock = true
		}
		if toolName == "safe-outputs" {
			hasSafeOutputs = true
			needsEnvBlock = true
		}
		if toolName == "agentic-workflows" {
			needsEnvBlock = true
		}
		if toolName == "playwright" {
			hasPlaywright = true
			// Extract all expressions from playwright arguments using ExpressionExtractor
			if playwrightTool, ok := tools["playwright"]; ok {
				allowedDomains := generatePlaywrightAllowedDomains(playwrightTool)
				customArgs := getPlaywrightCustomArgs(playwrightTool)
				playwrightAllowedDomainsSecrets = extractExpressionsFromPlaywrightArgs(allowedDomains, customArgs)
				if len(playwrightAllowedDomainsSecrets) > 0 {
					needsEnvBlock = true
				}
			}
		}
	}

	if needsEnvBlock {
		yaml.WriteString("        env:\n")

		// Add GitHub token env var if GitHub tool is present
		if hasGitHub {
			githubTool := tools["github"]
			customGitHubToken := getGitHubToken(githubTool)
			effectiveToken := getEffectiveGitHubToken(customGitHubToken, workflowData.GitHubToken)
			yaml.WriteString("          GITHUB_MCP_SERVER_TOKEN: " + effectiveToken + "\n")
		}

		// Add safe-outputs env vars if present
		if hasSafeOutputs {
			yaml.WriteString("          GH_AW_SAFE_OUTPUTS: ${{ env.GH_AW_SAFE_OUTPUTS }}\n")
			yaml.WriteString("          GH_AW_SAFE_OUTPUTS_CONFIG: ${{ toJSON(env.GH_AW_SAFE_OUTPUTS_CONFIG) }}\n")
			yaml.WriteString("          GH_AW_ASSETS_BRANCH: ${{ env.GH_AW_ASSETS_BRANCH }}\n")
			yaml.WriteString("          GH_AW_ASSETS_MAX_SIZE_KB: ${{ env.GH_AW_ASSETS_MAX_SIZE_KB }}\n")
			yaml.WriteString("          GH_AW_ASSETS_ALLOWED_EXTS: ${{ env.GH_AW_ASSETS_ALLOWED_EXTS }}\n")
		}

		// Add GITHUB_TOKEN for agentic-workflows if present
		if hasAgenticWorkflows {
			yaml.WriteString("          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}\n")
		}

		// Add Playwright expression environment variables if present
		if hasPlaywright && len(playwrightAllowedDomainsSecrets) > 0 {
			// Sort env var names for consistent output
			envVarNames := make([]string, 0, len(playwrightAllowedDomainsSecrets))
			for envVarName := range playwrightAllowedDomainsSecrets {
				envVarNames = append(envVarNames, envVarName)
			}
			sort.Strings(envVarNames)

			for _, envVarName := range envVarNames {
				originalExpr := playwrightAllowedDomainsSecrets[envVarName]
				yaml.WriteString(fmt.Sprintf("          %s: %s\n", envVarName, originalExpr))
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

// hasGitHubTool checks if the GitHub tool is configured (using ParsedTools)
func hasGitHubTool(parsedTools *Tools) bool {
	if parsedTools == nil {
		return false
	}
	return parsedTools.GitHub != nil
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
// Defaults to true for security
func getGitHubReadOnly(githubTool any) bool {
	if toolConfig, ok := githubTool.(map[string]any); ok {
		if readOnlySetting, exists := toolConfig["read-only"]; exists {
			if boolValue, ok := readOnlySetting.(bool); ok {
				return boolValue
			}
		}
	}
	return true // default to read-only for security
}

// getGitHubToolsets extracts the toolsets configuration from GitHub tool
// Defaults to "default" for recommended toolset
func getGitHubToolsets(githubTool any) string {
	if toolConfig, ok := githubTool.(map[string]any); ok {
		if toolsetsSetting, exists := toolConfig["toolsets"]; exists {
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
	return "default" // default to recommended toolset
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

// extractExpressionsFromPlaywrightArgs extracts all GitHub Actions expressions from playwright arguments
// Returns a map of environment variable names to their original expressions
// Uses the same ExpressionExtractor as used for shell script security
func extractExpressionsFromPlaywrightArgs(allowedDomains []string, customArgs []string) map[string]string {
	// Combine all arguments into a single string for extraction
	var allArgs []string
	allArgs = append(allArgs, allowedDomains...)
	allArgs = append(allArgs, customArgs...)

	if len(allArgs) == 0 {
		return make(map[string]string)
	}

	// Join all arguments with a separator that won't appear in expressions
	combined := strings.Join(allArgs, "\n")

	// Use ExpressionExtractor to find all expressions
	extractor := NewExpressionExtractor()
	mappings, err := extractor.ExtractExpressions(combined)
	if err != nil {
		return make(map[string]string)
	}

	// Convert to map of env var name -> original expression
	result := make(map[string]string)
	for _, mapping := range mappings {
		result[mapping.EnvVar] = mapping.Original
	}

	return result
}

// replaceExpressionsInPlaywrightArgs replaces all GitHub Actions expressions with environment variable references
// This prevents any expressions from being exposed in GitHub Actions logs
func replaceExpressionsInPlaywrightArgs(args []string, expressions map[string]string) []string {
	if len(expressions) == 0 {
		return args
	}

	// Create a temporary extractor with the same mappings
	combined := strings.Join(args, "\n")
	extractor := NewExpressionExtractor()
	_, _ = extractor.ExtractExpressions(combined)

	// Replace expressions in the combined string
	replaced := extractor.ReplaceExpressionsWithEnvVars(combined)

	// Split back into individual arguments
	return strings.Split(replaced, "\n")
}
