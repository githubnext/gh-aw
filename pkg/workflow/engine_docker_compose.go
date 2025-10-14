package workflow

import (
	"fmt"
	"sort"
	"strings"
)

// generateEngineDockerCompose generates Docker Compose configuration for containerized agent execution
// This creates a 3-container setup: agent, squid-proxy, and proxy-init
func generateEngineDockerCompose(engineID string, engineVersion string, envVars map[string]string,
	allowedDomains []string, agentCommand []string, workflowData *WorkflowData) string {

	// Derive network name for this engine
	networkName := "gh-aw-engine-net"

	compose := `version: '3.8'

services:
  # Agent container - runs the AI CLI (Claude Code, Codex, etc.)
  agent:
    image: ghcr.io/githubnext/gh-aw-agent-base:latest
    container_name: gh-aw-agent
    stdin_open: true
    tty: true
    working_dir: /github/workspace
    volumes:
      # Mount GitHub Actions workspace
      - $PWD:/github/workspace:rw
      # Mount MCP configuration (read-only)
      - ./mcp-config:/tmp/gh-aw/mcp-config:ro
      # Mount prompt files (read-only)
      - ./prompts:/tmp/gh-aw/aw-prompts:ro
      # Mount log directory (write access)
      - ./logs:/tmp/gh-aw/logs:rw
      # Mount safe outputs directory (read-write)
      - ./safe-outputs:/tmp/gh-aw/safe-outputs:rw
      # Mount Claude settings if present
      - ./.claude:/tmp/gh-aw/.claude:ro
    environment:
      # Proxy configuration - all traffic goes through localhost:3128
      - HTTP_PROXY=http://localhost:3128
      - HTTPS_PROXY=http://localhost:3128
      - http_proxy=http://localhost:3128
      - https_proxy=http://localhost:3128
      - NO_PROXY=localhost,127.0.0.1
      - no_proxy=localhost,127.0.0.1`

	// Add engine-specific environment variables in sorted order
	if len(envVars) > 0 {
		keys := make([]string, 0, len(envVars))
		for key := range envVars {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			value := envVars[key]
			compose += fmt.Sprintf("\n      - %s=%s", key, value)
		}
	}

	// Add command if specified
	if len(agentCommand) > 0 {
		compose += "\n    command: "
		compose += formatDockerComposeCommand(agentCommand)
	}

	compose += `
    networks:
      - ` + networkName + `
    depends_on:
      # Wait for proxy-init to complete setup
      proxy-init:
        condition: service_completed_successfully
      # Wait for Squid to be healthy
      squid-proxy:
        condition: service_healthy

  # Squid proxy container - provides HTTP/HTTPS proxy with domain filtering
  squid-proxy:
    image: ubuntu/squid:latest
    container_name: gh-aw-squid-proxy
    # Share network namespace with agent container
    # This allows Squid to intercept agent's traffic via iptables rules
    network_mode: "service:agent"
    volumes:
      # Mount Squid TPROXY configuration (read-only)
      - ./squid-tproxy.conf:/etc/squid/squid.conf:ro
      # Mount allowed domains file (read-only)
      - ./allowed_domains.txt:/etc/squid/allowed_domains.txt:ro
      # Persistent volume for Squid logs
      - squid-logs:/var/log/squid
    healthcheck:
      # Check if Squid is running and responding
      test: ["CMD", "squid", "-k", "check"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 10s
    cap_add:
      # Required to bind to ports 3128 and 3129
      - NET_BIND_SERVICE
    depends_on:
      # Squid needs the agent container to create the network namespace first
      - agent

  # Proxy-init container - sets up iptables rules for transparent proxy
  proxy-init:
    image: ghcr.io/githubnext/gh-aw-proxy-init:latest
    container_name: gh-aw-proxy-init
    # Share network namespace with agent container
    # This allows proxy-init to configure iptables that affect agent's traffic
    network_mode: "service:agent"
    cap_add:
      # Required for iptables and ip route commands
      - NET_ADMIN
    depends_on:
      # proxy-init needs agent and squid to be started first
      - agent
      - squid-proxy

# Volumes for persistent data
volumes:
  squid-logs:
    driver: local

# Network configuration
networks:
  ` + networkName + `:
    driver: bridge
`

	return compose
}

// formatDockerComposeCommand formats a command array for Docker Compose YAML
// Handles proper quoting and escaping of command arguments
func formatDockerComposeCommand(command []string) string {
	if len(command) == 0 {
		return "[]"
	}

	var parts []string
	for _, cmd := range command {
		// Quote strings that contain spaces, special characters, or are empty
		if strings.Contains(cmd, " ") || strings.Contains(cmd, "$") ||
			strings.Contains(cmd, "\"") || strings.Contains(cmd, "'") ||
			strings.Contains(cmd, "\n") || cmd == "" {
			// Escape existing double quotes
			escaped := strings.ReplaceAll(cmd, `"`, `\"`)
			parts = append(parts, fmt.Sprintf(`"%s"`, escaped))
		} else {
			parts = append(parts, fmt.Sprintf(`"%s"`, cmd))
		}
	}

	return "[" + strings.Join(parts, ", ") + "]"
}
