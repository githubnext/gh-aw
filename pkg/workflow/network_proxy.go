package workflow

import (
	"fmt"
	"strings"
)

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
	return `# Squid configuration for egress traffic control
# This configuration implements a whitelist-based proxy

# Access log and cache configuration
access_log /var/log/squid/access.log squid
cache_log /var/log/squid/cache.log
cache deny all

# Port configuration
http_port 3128

# ACL definitions for allowed domains
acl allowed_domains dstdomain "/etc/squid/allowed_domains.txt"
acl localnet src 10.0.0.0/8
acl localnet src 172.16.0.0/12
acl localnet src 192.168.0.0/16
acl SSL_ports port 443
acl Safe_ports port 80
acl Safe_ports port 443
acl CONNECT method CONNECT

# Access rules
# Deny requests to unknown domains (not in whitelist)
http_access deny !allowed_domains
http_access deny !Safe_ports
http_access deny CONNECT !SSL_ports
http_access allow localnet
http_access deny all

# Disable caching
cache deny all

# DNS settings
dns_nameservers 8.8.8.8 8.8.4.4

# Forwarded headers
forwarded_for delete
via off

# Error page customization
error_directory /usr/share/squid/errors/English

# Logging
logformat combined %>a %[ui %[un [%tl] "%rm %ru HTTP/%rv" %>Hs %<st "%{Referer}>h" "%{User-Agent}>h" %Ss:%Sh
access_log /var/log/squid/access.log combined

# Memory and file descriptor limits
cache_mem 64 MB
maximum_object_size 0 KB
`
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
