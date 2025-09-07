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
	// Check if tool has MCP container configuration
	mcpConfig, err := getMCPConfig(toolConfig, "")
	if err != nil {
		return false, nil
	}

	// Check if it has a container field
	_, hasContainer := mcpConfig["container"]
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

	// Get container image and environment variables from MCP config
	mcpConfig, err := getMCPConfig(toolConfig, toolName)
	if err != nil {
		if c.verbose {
			fmt.Printf("Error getting MCP config for %s: %v\n", toolName, err)
		}
		return
	}

	containerImage, hasContainer := mcpConfig["container"]
	if !hasContainer {
		if c.verbose {
			fmt.Printf("Proxy-enabled tool '%s' missing container configuration\n", toolName)
		}
		return
	}

	containerStr, ok := containerImage.(string)
	if !ok {
		if c.verbose {
			fmt.Printf("Container image must be a string for tool %s\n", toolName)
		}
		return
	}

	var envVars map[string]any
	if env, hasEnv := mcpConfig["env"]; hasEnv {
		if envMap, ok := env.(map[string]any); ok {
			envVars = envMap
		}
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
	var customProxyArgs []string
	if proxyArgsInterface, hasProxyArgs := mcpConfig["proxy_args"]; hasProxyArgs {
		if proxyArgsSlice, ok := proxyArgsInterface.([]any); ok {
			for _, arg := range proxyArgsSlice {
				if argStr, ok := arg.(string); ok {
					customProxyArgs = append(customProxyArgs, argStr)
				}
			}
		}
	}

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
