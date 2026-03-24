package workflow

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
)

var frontmatterMetadataLog = logger.New("workflow:frontmatter_extraction_metadata")

// extractFeatures extracts the features field from frontmatter
// Returns a map of feature flags and configuration options (supports boolean flags and string values)
func (c *Compiler) extractFeatures(frontmatter map[string]any) map[string]any {
	frontmatterMetadataLog.Print("Extracting features from frontmatter")
	value, exists := frontmatter["features"]
	if !exists {
		frontmatterMetadataLog.Print("No features field found in frontmatter")
		return nil
	}

	// Features should be an object with any values (boolean or string)
	if featuresMap, ok := value.(map[string]any); ok {
		result := make(map[string]any)
		// Accept any value type (boolean, string, etc.)
		maps.Copy(result, featuresMap)
		frontmatterMetadataLog.Printf("Extracted %d features", len(result))
		return result
	}

	frontmatterMetadataLog.Print("Features field is not a map")
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
		desc := strings.TrimSpace(strValue)
		frontmatterMetadataLog.Printf("Extracted description: %d characters", len(desc))
		return desc
	}

	frontmatterMetadataLog.Printf("Description field is not a string: type=%T", value)
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

	frontmatterMetadataLog.Print("Extracting and validating tracker-id")

	// Convert the value to string
	strValue, ok := value.(string)
	if !ok {
		frontmatterMetadataLog.Printf("Invalid tracker-id type: %T", value)
		return "", fmt.Errorf("tracker-id must be a string, got %T. Example: tracker-id: \"my-tracker-123\"", value)
	}

	trackerID := strings.TrimSpace(strValue)

	// Validate minimum length
	if len(trackerID) < 8 {
		frontmatterMetadataLog.Printf("tracker-id too short: %d characters", len(trackerID))
		return "", fmt.Errorf("tracker-id must be at least 8 characters long (got %d)", len(trackerID))
	}

	// Validate that it's a valid identifier (alphanumeric, hyphens, underscores)
	for i, char := range trackerID {
		if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') &&
			(char < '0' || char > '9') && char != '-' && char != '_' {
			frontmatterMetadataLog.Printf("Invalid character in tracker-id at position %d", i+1)
			return "", fmt.Errorf("tracker-id contains invalid character at position %d: '%c' (only alphanumeric, hyphens, and underscores allowed)", i+1, char)
		}
	}

	frontmatterMetadataLog.Printf("Successfully validated tracker-id: %s", trackerID)
	return trackerID, nil
}

// buildSourceURL converts a source string (owner/repo/path@ref) to a GitHub URL
// For enterprise deployments, the URL will use the GitHub server URL from the workflow context
func buildSourceURL(source string) string {
	frontmatterMetadataLog.Printf("Building source URL from: %s", source)
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
		frontmatterMetadataLog.Printf("Invalid source path format: %s (expected owner/repo/path)", pathPart)
		return ""
	}

	owner := pathComponents[0]
	repo := pathComponents[1]
	filePath := pathComponents[2]

	url := fmt.Sprintf("${{ github.server_url }}/%s/%s/tree/%s/%s", owner, repo, refPart, filePath)
	frontmatterMetadataLog.Printf("Built source URL: %s/%s tree %s", owner, repo, refPart)
	// Use github.server_url for enterprise GitHub deployments
	return url
}

// extractToolsTimeout extracts the timeout setting from tools
// Returns 0 if not set (engines will use their own defaults)
// Returns error if timeout is explicitly set but invalid (< 1)
func (c *Compiler) extractToolsTimeout(tools map[string]any) (int, error) {
	if tools == nil {
		return 0, nil // Use engine defaults
	}

	// Check if timeout is explicitly set in tools
	if timeoutValue, exists := tools["timeout"]; exists {
		frontmatterMetadataLog.Printf("Extracting tools.timeout value: type=%T", timeoutValue)
		var timeout int
		// Handle different numeric types with safe conversions to prevent overflow
		switch v := timeoutValue.(type) {
		case int:
			timeout = v
		case int64:
			timeout = int(v)
		case uint:
			timeout = safeUintToInt(v) // Safe conversion to prevent overflow (alert #418)
		case uint64:
			timeout = safeUint64ToInt(v) // Safe conversion to prevent overflow (alert #416)
		case float64:
			timeout = int(v)
		default:
			frontmatterMetadataLog.Printf("Invalid tools.timeout type: %T", timeoutValue)
			return 0, fmt.Errorf("tools.timeout must be an integer, got %T", timeoutValue)
		}

		// Validate minimum value per schema constraint
		if timeout < 1 {
			frontmatterMetadataLog.Printf("Invalid tools.timeout value: %d (must be >= 1)", timeout)
			return 0, fmt.Errorf("tools.timeout must be at least 1 second, got %d. Example:\ntools:\n  timeout: 60", timeout)
		}

		frontmatterMetadataLog.Printf("Extracted tools.timeout: %d seconds", timeout)
		return timeout, nil
	}

	// Default to 0 (use engine defaults)
	return 0, nil
}

// extractToolsStartupTimeout extracts the startup-timeout setting from tools
// Returns 0 if not set (engines will use their own defaults)
// Returns error if startup-timeout is explicitly set but invalid (< 1)
func (c *Compiler) extractToolsStartupTimeout(tools map[string]any) (int, error) {
	if tools == nil {
		return 0, nil // Use engine defaults
	}

	// Check if startup-timeout is explicitly set in tools
	if timeoutValue, exists := tools["startup-timeout"]; exists {
		var timeout int
		// Handle different numeric types with safe conversions to prevent overflow
		switch v := timeoutValue.(type) {
		case int:
			timeout = v
		case int64:
			timeout = int(v)
		case uint:
			timeout = safeUintToInt(v) // Safe conversion to prevent overflow (alert #417)
		case uint64:
			timeout = safeUint64ToInt(v) // Safe conversion to prevent overflow (alert #415)
		case float64:
			timeout = int(v)
		default:
			return 0, fmt.Errorf("tools.startup-timeout must be an integer, got %T", timeoutValue)
		}

		// Validate minimum value per schema constraint
		if timeout < 1 {
			return 0, fmt.Errorf("tools.startup-timeout must be at least 1 second, got %d. Example:\ntools:\n  startup-timeout: 120", timeout)
		}

		return timeout, nil
	}

	// Default to 0 (use engine defaults)
	return 0, nil
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

// extractAPMDependenciesFromFrontmatter extracts APM (Agent Package Manager) dependency
// configuration from frontmatter. Supports two sources:
//   - imports.apm-packages (preferred location)
//   - dependencies (deprecated; emits a deprecation warning when used alone)
//
// It is an error to specify both sources simultaneously.
//
// Each source supports:
//   - Array format: ["org/pkg1", "org/pkg2"]
//   - Object format: {packages: ["org/pkg1", "org/pkg2"], isolated: true, github-app: {...}, github-token: "...", version: "v0.8.0"}
//
// Returns nil if neither source is present or if the resolved source contains no packages.
func extractAPMDependenciesFromFrontmatter(frontmatter map[string]any) (*APMDependenciesInfo, error) {
	hasImportsAPM := false
	var importsAPMValue any
	if importsAny, hasImports := frontmatter["imports"]; hasImports {
		if importsMap, ok := importsAny.(map[string]any); ok {
			if apmAny, hasAPM := importsMap["apm-packages"]; hasAPM {
				hasImportsAPM = true
				importsAPMValue = apmAny
			}
		}
	}

	_, hasDependencies := frontmatter["dependencies"]

	// It is an error to specify both sources simultaneously.
	if hasImportsAPM && hasDependencies {
		return nil, errors.New(
			"cannot use both 'imports.apm-packages' and 'dependencies' simultaneously; " +
				"remove 'dependencies' and use 'imports.apm-packages' exclusively; " +
				"run 'gh aw fix --write' to automatically migrate",
		)
	}

	if hasImportsAPM {
		frontmatterMetadataLog.Print("Extracting APM dependencies from imports.apm-packages")
		return extractAPMDependenciesFromValue(importsAPMValue, "imports.apm-packages")
	}

	// Fall back to top-level dependencies field (deprecated)
	if !hasDependencies {
		return nil, nil
	}

	// Emit deprecation warning for the top-level dependencies field
	fmt.Fprintln(os.Stderr, console.FormatWarningMessage(
		"The top-level 'dependencies' field is deprecated. "+
			"Use 'imports.apm-packages' instead. "+
			"Run 'gh aw fix --write' to automatically migrate.",
	))
	frontmatterMetadataLog.Print("Extracting APM dependencies from deprecated 'dependencies' field")

	return extractAPMDependenciesFromValue(frontmatter["dependencies"], "dependencies")
}

// extractAPMDependenciesFromValue extracts APM dependency configuration from a frontmatter value.
// fieldName is used for error messages.
func extractAPMDependenciesFromValue(value any, fieldName string) (*APMDependenciesInfo, error) {
	var packages []string
	var isolated bool
	var githubApp *GitHubAppConfig
	var githubToken string
	var version string
	var env map[string]string

	switch v := value.(type) {
	case []any:
		// Array format: [pkg1, pkg2]
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				packages = append(packages, s)
			}
		}
	case map[string]any:
		// Object format: {packages: [...], isolated: true, github-app: {...}, github-token: "...", version: "v0.8.0"}
		if pkgsAny, ok := v["packages"]; ok {
			if pkgsArray, ok := pkgsAny.([]any); ok {
				for _, item := range pkgsArray {
					if s, ok := item.(string); ok && s != "" {
						packages = append(packages, s)
					}
				}
			}
		}
		if iso, ok := v["isolated"]; ok {
			if isoBool, ok := iso.(bool); ok {
				isolated = isoBool
			}
		}
		if appAny, ok := v["github-app"]; ok {
			if appMap, ok := appAny.(map[string]any); ok {
				githubApp = parseAppConfig(appMap)
				if githubApp.AppID == "" || githubApp.PrivateKey == "" {
					frontmatterMetadataLog.Printf("%s.github-app missing required app-id or private-key; ignoring", fieldName)
					githubApp = nil
				}
			}
		}
		if tokenAny, ok := v["github-token"]; ok {
			if tokenStr, ok := tokenAny.(string); ok && tokenStr != "" {
				githubToken = tokenStr
				frontmatterMetadataLog.Printf("Extracted %s.github-token: custom token configured", fieldName)
			}
		}
		if versionAny, ok := v["version"]; ok {
			if versionStr, ok := versionAny.(string); ok && versionStr != "" {
				if !isValidVersionTag(versionStr) {
					return nil, fmt.Errorf("%s.version %q is not a valid semver tag (expected format: vX.Y.Z)", fieldName, versionStr)
				}
				version = versionStr
			}
		}
		if envAny, ok := v["env"]; ok {
			if envMap, ok := envAny.(map[string]any); ok && len(envMap) > 0 {
				env = make(map[string]string, len(envMap))
				for k, val := range envMap {
					if s, ok := val.(string); ok {
						env[k] = s
					} else {
						frontmatterMetadataLog.Printf("Skipping non-string env value for key '%s'", k)
					}
				}
			}
		}
	default:
		return nil, nil
	}

	if len(packages) == 0 {
		return nil, nil
	}

	frontmatterMetadataLog.Printf("Extracted %d APM dependency packages from %s (isolated=%v, github-app=%v, github-token=%v, version=%s, env=%d)", len(packages), fieldName, isolated, githubApp != nil, githubToken != "", version, len(env))
	return &APMDependenciesInfo{Packages: packages, Isolated: isolated, GitHubApp: githubApp, GitHubToken: githubToken, Version: version, Env: env}, nil
}
