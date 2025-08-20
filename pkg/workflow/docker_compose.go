package workflow

import (
	"fmt"
	"hash/crc32"
	"strings"
)

// getStandardProxyArgs returns the standard proxy arguments for all MCP containers
// This defines the standard interface that all proxy-enabled MCP containers should support
func getStandardProxyArgs() []string {
	// We no longer rely on CLI flags like --proxy-url.
	// Leave empty so we don't override container entrypoints.
	return []string{}
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
	// Derive a stable, non-conflicting subnet and network name for this tool
	octet := 100 + (int(crc32.ChecksumIEEE([]byte(toolName))) % 100) // 100-199
	subnet := fmt.Sprintf("172.28.%d.0/24", octet)
	squidIP := fmt.Sprintf("172.28.%d.10", octet)
	networkName := "awproxy-" + toolName

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
    networks:
      ` + networkName + `:
        ipv4_address: ` + squidIP + `

  ` + toolName + `:
    image: ` + containerImage + `
    container_name: ` + toolName + `-mcp
    stdin_open: true
    tty: true
    environment:
      - PROXY_HOST=squid-proxy
      - PROXY_PORT=3128
      - HTTP_PROXY=http://squid-proxy:3128
      - HTTPS_PROXY=http://squid-proxy:3128
    networks:
      - ` + networkName + ``

	// Add environment variables
	for key, value := range envVars {
		if valueStr, ok := value.(string); ok {
			compose += "\n      - " + key + "=" + valueStr
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
	// Only set command if custom args were explicitly provided
	if len(proxyArgs) > 0 {
		compose += `
    command: ` + formatYAMLArray(proxyArgs)
	}

	compose += `
    depends_on:
      squid-proxy:
        condition: service_healthy

volumes:
  squid-logs:

networks:
  ` + networkName + `:
    driver: bridge
    ipam:
      config:
        - subnet: ` + subnet + `
`

	return compose
}

// computeProxyNetworkParams returns the subnet CIDR, squid IP and network name for a given tool
func computeProxyNetworkParams(toolName string) (subnetCIDR string, squidIP string, networkName string) {
	octet := 100 + (int(crc32.ChecksumIEEE([]byte(toolName))) % 100) // 100-199
	subnetCIDR = fmt.Sprintf("172.28.%d.0/24", octet)
	squidIP = fmt.Sprintf("172.28.%d.10", octet)
	networkName = "awproxy-" + toolName
	return
}
