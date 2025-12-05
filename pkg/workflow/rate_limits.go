package workflow

import (
	"encoding/json"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var rateLimitsLog = logger.New("workflow:rate_limits")

// extractRateLimitsConfig extracts rate-limits configuration from frontmatter
func (c *Compiler) extractRateLimitsConfig(frontmatter map[string]any) *RateLimitsConfig {
	rateLimitsLog.Print("Extracting rate-limits configuration from frontmatter")

	if rateLimits, exists := frontmatter["rate-limits"]; exists {
		if rateLimitsMap, ok := rateLimits.(map[string]any); ok {
			config := &RateLimitsConfig{}

			// Extract github-api rate limit
			if githubAPI, exists := rateLimitsMap["github-api"]; exists {
				if val, ok := githubAPI.(string); ok {
					config.GitHubAPI = val
					rateLimitsLog.Printf("Extracted github-api rate limit: %s", val)
				}
			}

			// Extract mcp-requests rate limit
			if mcpRequests, exists := rateLimitsMap["mcp-requests"]; exists {
				if val, ok := mcpRequests.(string); ok {
					config.MCPRequests = val
					rateLimitsLog.Printf("Extracted mcp-requests rate limit: %s", val)
				}
			}

			// Extract network-requests rate limit
			if networkRequests, exists := rateLimitsMap["network-requests"]; exists {
				if val, ok := networkRequests.(string); ok {
					config.NetworkRequests = val
					rateLimitsLog.Printf("Extracted network-requests rate limit: %s", val)
				}
			}

			// Extract file-read rate limit
			if fileRead, exists := rateLimitsMap["file-read"]; exists {
				if val, ok := fileRead.(string); ok {
					config.FileRead = val
					rateLimitsLog.Printf("Extracted file-read rate limit: %s", val)
				}
			}

			// Return nil if no rate limits were found
			if config.GitHubAPI == "" && config.MCPRequests == "" &&
				config.NetworkRequests == "" && config.FileRead == "" {
				rateLimitsLog.Print("No rate-limits found in frontmatter")
				return nil
			}

			return config
		}
	}

	rateLimitsLog.Print("No rate-limits configuration found in frontmatter")
	return nil
}

// MergeRateLimits merges rate-limits configurations from imports with top-level config
func (c *Compiler) MergeRateLimits(topConfig *RateLimitsConfig, importedRateLimitsJSON string) (*RateLimitsConfig, error) {
	rateLimitsLog.Print("Merging rate-limits from imports")

	if importedRateLimitsJSON == "" || importedRateLimitsJSON == "{}" {
		rateLimitsLog.Print("No imported rate-limits to merge")
		return topConfig, nil
	}

	// Start with top-level config or create a new one
	result := &RateLimitsConfig{}
	if topConfig != nil {
		result.GitHubAPI = topConfig.GitHubAPI
		result.MCPRequests = topConfig.MCPRequests
		result.NetworkRequests = topConfig.NetworkRequests
		result.FileRead = topConfig.FileRead
		rateLimitsLog.Print("Starting with top-level rate-limits config")
	}

	// Split by newlines to handle multiple JSON objects from different imports
	lines := strings.Split(importedRateLimitsJSON, "\n")
	rateLimitsLog.Printf("Processing %d rate-limits definition lines", len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line == "{}" {
			continue
		}

		// Parse JSON line to rate-limits config
		var importedConfig RateLimitsConfig
		if err := json.Unmarshal([]byte(line), &importedConfig); err != nil {
			rateLimitsLog.Printf("Failed to parse rate-limits: %v", err)
			continue // Skip invalid lines
		}

		// Merge rate limits (imported values override only if top-level is empty)
		// Top-level config takes precedence over imported config
		if result.GitHubAPI == "" && importedConfig.GitHubAPI != "" {
			result.GitHubAPI = importedConfig.GitHubAPI
			rateLimitsLog.Printf("Merged github-api rate limit from import: %s", importedConfig.GitHubAPI)
		}
		if result.MCPRequests == "" && importedConfig.MCPRequests != "" {
			result.MCPRequests = importedConfig.MCPRequests
			rateLimitsLog.Printf("Merged mcp-requests rate limit from import: %s", importedConfig.MCPRequests)
		}
		if result.NetworkRequests == "" && importedConfig.NetworkRequests != "" {
			result.NetworkRequests = importedConfig.NetworkRequests
			rateLimitsLog.Printf("Merged network-requests rate limit from import: %s", importedConfig.NetworkRequests)
		}
		if result.FileRead == "" && importedConfig.FileRead != "" {
			result.FileRead = importedConfig.FileRead
			rateLimitsLog.Printf("Merged file-read rate limit from import: %s", importedConfig.FileRead)
		}
	}

	// Return nil if no rate limits were found
	if result.GitHubAPI == "" && result.MCPRequests == "" &&
		result.NetworkRequests == "" && result.FileRead == "" {
		rateLimitsLog.Print("No rate-limits after merging")
		return nil, nil
	}

	rateLimitsLog.Print("Successfully merged rate-limits configuration")
	return result, nil
}
