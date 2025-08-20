package workflow

import (
	"fmt"
	"strings"
)

// getStandardProxyArgs returns the standard proxy arguments for all MCP containers
// This defines the standard interface that all proxy-enabled MCP containers should support
func getStandardProxyArgs() []string {
	return []string{"--proxy-url", "http://squid-proxy:3128"}
}

// formatYAMLArray formats a string slice as a YAML array
func formatYAMLArray(items []string) string {
	if len(items) == 0 {
		return "[]"
	}
	
	var parts []string
	for _, item := range items {
		parts = append(parts, fmt.Sprintf(`"%s"`, item))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// generateDockerCompose generates the Docker Compose configuration
func generateDockerCompose(containerImage string, envVars map[string]any, toolName string, customProxyArgs []string) string {
	compose := `services:
  squid-proxy:
    image: ubuntu/squid:latest
    container_name: squid-proxy-` + toolName + `
    ports:
      - "3128:3128"
    volumes:
      - ./squid.conf:/etc/squid/squid.conf:ro
      - ./allowed_domains.txt:/etc/squid/allowed_domains.txt:ro
      - squid-logs:/var/log/squid
    healthcheck:
      test: ["CMD", "squid", "-k", "check"]
      interval: 30s
      timeout: 10s
      retries: 3
    restart: unless-stopped

  ` + toolName + `:
    image: ` + containerImage + `
    container_name: ` + toolName + `-mcp
    stdin_open: true
    tty: true
    environment:
      - PROXY_HOST=squid-proxy
      - PROXY_PORT=3128`

	// Add environment variables
	if envVars != nil {
		for key, value := range envVars {
			if valueStr, ok := value.(string); ok {
				compose += "\n      - " + key + "=" + valueStr
			}
		}
	}

	// Set proxy-aware command - use standard proxy args for all containers
	var proxyArgs []string
	if len(customProxyArgs) > 0 {
		// Use user-provided proxy args (for advanced users or non-standard containers)
		proxyArgs = customProxyArgs
	} else {
		// Use standard proxy args for all MCP containers
		proxyArgs = getStandardProxyArgs()
	}
	
	compose += `
    command: ` + formatYAMLArray(proxyArgs)

	compose += `
    depends_on:
      squid-proxy:
        condition: service_healthy

volumes:
  squid-logs:
`

	return compose
}
