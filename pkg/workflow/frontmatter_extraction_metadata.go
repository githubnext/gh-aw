package workflow

import (
	"fmt"
	"math"
	"strings"
)

// extractFeatures extracts the features field from frontmatter
// Returns a map of feature flags and configuration options (supports boolean flags and string values)
func (c *Compiler) extractFeatures(frontmatter map[string]any) map[string]any {
	value, exists := frontmatter["features"]
	if !exists {
		return nil
	}

	// Features should be an object with any values (boolean or string)
	if featuresMap, ok := value.(map[string]any); ok {
		result := make(map[string]any)
		for key, val := range featuresMap {
			// Accept any value type (boolean, string, etc.)
			result[key] = val
		}
		if log.Enabled() {
			frontmatterLog.Printf("Extracted %d features", len(result))
		}
		return result
	}

	return nil
}

// extractDescription extracts the description field from frontmatter
func (c *Compiler) extractDescription(frontmatter map[string]any) string {
	value, exists := frontmatter["description"]
	if !exists {
		return ""
	}

	// Convert the value to string
	if strValue, ok := value.(string); ok {
		return strings.TrimSpace(strValue)
	}

	return ""
}

// extractSource extracts the source field from frontmatter
func (c *Compiler) extractSource(frontmatter map[string]any) string {
	value, exists := frontmatter["source"]
	if !exists {
		return ""
	}

	// Convert the value to string
	if strValue, ok := value.(string); ok {
		return strings.TrimSpace(strValue)
	}

	return ""
}

// extractTrackerID extracts and validates the tracker-id field from frontmatter
func (c *Compiler) extractTrackerID(frontmatter map[string]any) (string, error) {
	value, exists := frontmatter["tracker-id"]
	if !exists {
		return "", nil
	}

	// Convert the value to string
	strValue, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("tracker-id must be a string, got %T. Example: tracker-id: \"my-tracker-123\"", value)
	}

	trackerID := strings.TrimSpace(strValue)

	// Validate minimum length
	if len(trackerID) < 8 {
		return "", fmt.Errorf("tracker-id must be at least 8 characters long (got %d)", len(trackerID))
	}

	// Validate that it's a valid identifier (alphanumeric, hyphens, underscores)
	for i, char := range trackerID {
		if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') &&
			(char < '0' || char > '9') && char != '-' && char != '_' {
			return "", fmt.Errorf("tracker-id contains invalid character at position %d: '%c' (only alphanumeric, hyphens, and underscores allowed)", i+1, char)
		}
	}

	return trackerID, nil
}

// buildSourceURL converts a source string (owner/repo/path@ref) to a GitHub URL
// For enterprise deployments, the URL will use the GitHub server URL from the workflow context
func buildSourceURL(source string) string {
	if source == "" {
		return ""
	}

	// Parse the source string: owner/repo/path@ref
	parts := strings.Split(source, "@")
	if len(parts) == 0 {
		return ""
	}

	pathPart := parts[0] // "owner/repo/path"
	refPart := "main"    // default ref
	if len(parts) > 1 {
		refPart = parts[1]
	}

	// Build GitHub URL using server URL from GitHub Actions context
	// The pathPart is "owner/repo/workflows/file.md", we need to convert it to
	// "${GITHUB_SERVER_URL}/owner/repo/tree/ref/workflows/file.md"
	pathComponents := strings.SplitN(pathPart, "/", 3)
	if len(pathComponents) < 3 {
		return ""
	}

	owner := pathComponents[0]
	repo := pathComponents[1]
	filePath := pathComponents[2]

	// Use github.server_url for enterprise GitHub deployments
	return fmt.Sprintf("${{ github.server_url }}/%s/%s/tree/%s/%s", owner, repo, refPart, filePath)
}

// safeUintToInt safely converts uint to int, returning 0 if overflow would occur
func safeUintToInt(u uint) int {
	if u > math.MaxInt {
		return 0 // Return 0 (engine default) if value would overflow
	}
	return int(u)
}

// safeUint64ToInt safely converts uint64 to int, returning 0 if overflow would occur
func safeUint64ToInt(u uint64) int {
	if u > math.MaxInt {
		return 0 // Return 0 (engine default) if value would overflow
	}
	return int(u)
}

// extractToolsTimeout extracts the timeout setting from tools
// Returns 0 if not set (engines will use their own defaults)
func (c *Compiler) extractToolsTimeout(tools map[string]any) int {
	if tools == nil {
		return 0 // Use engine defaults
	}

	// Check if timeout is explicitly set in tools
	if timeoutValue, exists := tools["timeout"]; exists {
		// Handle different numeric types with safe conversions to prevent overflow
		switch v := timeoutValue.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case uint:
			return safeUintToInt(v) // Safe conversion to prevent overflow (alert #418)
		case uint64:
			return safeUint64ToInt(v) // Safe conversion to prevent overflow (alert #416)
		case float64:
			return int(v)
		}
	}

	// Default to 0 (use engine defaults)
	return 0
}

// extractToolsStartupTimeout extracts the startup-timeout setting from tools
// Returns 0 if not set (engines will use their own defaults)
func (c *Compiler) extractToolsStartupTimeout(tools map[string]any) int {
	if tools == nil {
		return 0 // Use engine defaults
	}

	// Check if startup-timeout is explicitly set in tools
	if timeoutValue, exists := tools["startup-timeout"]; exists {
		// Handle different numeric types with safe conversions to prevent overflow
		switch v := timeoutValue.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case uint:
			return safeUintToInt(v) // Safe conversion to prevent overflow (alert #417)
		case uint64:
			return safeUint64ToInt(v) // Safe conversion to prevent overflow (alert #415)
		case float64:
			return int(v)
		}
	}

	// Default to 0 (use engine defaults)
	return 0
}

// extractMapFromFrontmatter is a generic helper to extract a map[string]any from frontmatter
// This now uses the structured extraction helper for better error handling
func extractMapFromFrontmatter(frontmatter map[string]any, key string) map[string]any {
	return ExtractMapField(frontmatter, key)
}

// extractToolsFromFrontmatter extracts tools section from frontmatter map
func extractToolsFromFrontmatter(frontmatter map[string]any) map[string]any {
	return ExtractMapField(frontmatter, "tools")
}

// extractMCPServersFromFrontmatter extracts mcp-servers section from frontmatter
func extractMCPServersFromFrontmatter(frontmatter map[string]any) map[string]any {
	return ExtractMapField(frontmatter, "mcp-servers")
}

// extractRuntimesFromFrontmatter extracts runtimes section from frontmatter map
func extractRuntimesFromFrontmatter(frontmatter map[string]any) map[string]any {
	return ExtractMapField(frontmatter, "runtimes")
}
