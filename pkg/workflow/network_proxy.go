package workflow

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed config/squid.conf
var squidConfigContent string

// needsProxy determines if a tool configuration requires proxy setup
func needsProxy(toolConfig map[string]any) (bool, []string) {
	// Check if tool has container configuration (before transformation)
	_, hasContainer := toolConfig["container"]
	if !hasContainer {
		return false, nil
	}

	// Check if it has network permissions
	hasNetPerms, domains := hasNetworkPermissions(toolConfig)

	return hasNetPerms, domains
}

// generateSquidConfig generates the Squid proxy configuration
func generateSquidConfig() string {
	return squidConfigContent
}

// generateAllowedDomainsFile generates the allowed domains file content
func generateAllowedDomainsFile(domains []string) string {
	content := "# Allowed domains for egress traffic\n# Add one domain per line\n"
	for _, domain := range domains {
		content += domain + "\n"
	}
	return content
}

// generateProxyFiles generates Squid proxy configuration files for a tool
// Removed unused generateProxyFiles; inline generation is used instead.

// generateInlineProxyConfig generates proxy configuration files inline in the workflow
func (c *Compiler) generateInlineProxyConfig(yaml *strings.Builder, toolName string, toolConfig map[string]any) {
	needsProxySetup, allowedDomains := needsProxy(toolConfig)
	if !needsProxySetup {
		return
	}

	// Get container image directly from toolConfig before transformation
	containerInterface, hasContainer := toolConfig["container"]
	if !hasContainer {
		if c.verbose {
			fmt.Printf("Proxy-enabled tool '%s' missing container field\n", toolName)
		}
		return
	}

	containerStr, ok := containerInterface.(string)
	if !ok {
		if c.verbose {
			fmt.Printf("Proxy-enabled tool '%s' container field is not a string\n", toolName)
		}
		return
	}

	// Get version if specified
	version := ""
	if versionInterface, hasVersion := toolConfig["version"]; hasVersion {
		if versionStr, ok := versionInterface.(string); ok {
			version = versionStr
		}
	}

	// Build full container image with version
	if version != "" {
		containerStr = containerStr + ":" + version
	}

	// Get environment variables
	envVars := make(map[string]any)
	if envInterface, hasEnv := toolConfig["env"]; hasEnv {
		if envMap, ok := envInterface.(map[string]any); ok {
			for k, v := range envMap {
				envVars[k] = v
			}
		}
	}

	// Get MCP config for proxy args
	mcpConfig, err := getMCPConfig(toolConfig, toolName)
	if err != nil {
		if c.verbose {
			fmt.Printf("Error getting MCP config for %s: %v\n", toolName, err)
		}
		return
	}

	if c.verbose {
		fmt.Printf("Generating inline proxy configuration for tool '%s'\n", toolName)
	}

	// Generate squid.conf inline
	yaml.WriteString("          # Generate Squid proxy configuration\n")
	yaml.WriteString("          cat > squid.conf << 'EOF'\n")
	squidConfigContent := generateSquidConfig()
	for _, line := range strings.Split(squidConfigContent, "\n") {
		fmt.Fprintf(yaml, "          %s\n", line)
	}
	yaml.WriteString("          EOF\n")
	yaml.WriteString("          \n")

	// Generate allowed_domains.txt inline
	yaml.WriteString("          # Generate allowed domains file\n")
	yaml.WriteString("          cat > allowed_domains.txt << 'EOF'\n")
	allowedDomainsContent := generateAllowedDomainsFile(allowedDomains)
	for _, line := range strings.Split(allowedDomainsContent, "\n") {
		fmt.Fprintf(yaml, "          %s\n", line)
	}
	yaml.WriteString("          EOF\n")
	yaml.WriteString("          \n")

	// Extract custom proxy args from MCP config if present
	customProxyArgs := mcpConfig.ProxyArgs

	// Generate docker-compose.yml inline
	fmt.Fprintf(yaml, "          # Generate Docker Compose configuration for %s\n", toolName)
	fmt.Fprintf(yaml, "          cat > docker-compose-%s.yml << 'EOF'\n", toolName)
	dockerComposeContent := generateDockerCompose(containerStr, envVars, toolName, customProxyArgs)
	for _, line := range strings.Split(dockerComposeContent, "\n") {
		fmt.Fprintf(yaml, "          %s\n", line)
	}
	yaml.WriteString("          EOF\n")
	yaml.WriteString("          \n")
}
