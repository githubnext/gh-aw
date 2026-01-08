package workflow

import (
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var mcpUtilsLog = logger.New("workflow:mcp-config-utils")

// rewriteLocalhostToDockerHost rewrites localhost URLs to use host.docker.internal
// This is necessary when MCP servers run on the host machine but are accessed from within
// a Docker container (e.g., when firewall/sandbox is enabled)
func rewriteLocalhostToDockerHost(url string) string {
	// Define the localhost patterns to replace and their docker equivalents
	// Each pattern is a (prefix, replacement) pair
	replacements := []struct {
		prefix      string
		replacement string
	}{
		{"http://localhost", "http://host.docker.internal"},
		{"https://localhost", "https://host.docker.internal"},
		{"http://127.0.0.1", "http://host.docker.internal"},
		{"https://127.0.0.1", "https://host.docker.internal"},
	}

	for _, r := range replacements {
		if strings.HasPrefix(url, r.prefix) {
			newURL := r.replacement + url[len(r.prefix):]
			mcpUtilsLog.Printf("Rewriting localhost URL for Docker access: %s -> %s", url, newURL)
			return newURL
		}
	}

	return url
}

// collectHTTPMCPHeaderSecrets collects all secrets from HTTP MCP tool headers
// Returns a map of environment variable names to their secret expressions
func collectHTTPMCPHeaderSecrets(tools map[string]any) map[string]string {
	allSecrets := make(map[string]string)

	for toolName, toolValue := range tools {
		// Check if this is an MCP tool configuration
		if toolConfig, ok := toolValue.(map[string]any); ok {
			if hasMcp, mcpType := hasMCPConfig(toolConfig); hasMcp && mcpType == "http" {
				// Extract MCP config to get headers
				if mcpConfig, err := getMCPConfig(toolConfig, toolName); err == nil {
					secrets := ExtractSecretsFromMap(mcpConfig.Headers)
					for varName, expr := range secrets {
						allSecrets[varName] = expr
					}
				}
			}
		}
	}

	return allSecrets
}
