package workflow

import (
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"
)

// extractStringValue extracts a string value from the frontmatter map
func extractStringValue(frontmatter map[string]any, key string) string {
	value, exists := frontmatter[key]
	if !exists {
		return ""
	}

	if strValue, ok := value.(string); ok {
		return strValue
	}

	return ""
}

// parseIntValue safely parses various numeric types to int
// This is a common utility used across multiple parsing functions
func parseIntValue(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case uint64:
		return int(v), true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

// addCustomSafeOutputEnvVars adds custom environment variables to safe output job steps
func (c *Compiler) addCustomSafeOutputEnvVars(steps *[]string, data *WorkflowData) {
	if data.SafeOutputs != nil && len(data.SafeOutputs.Env) > 0 {
		for key, value := range data.SafeOutputs.Env {
			*steps = append(*steps, fmt.Sprintf("          %s: %s\n", key, value))
		}
	}
}

// addSafeOutputGitHubToken adds github-token to the with section of github-script actions
// Uses precedence: safe-outputs global github-token > top-level github-token > default
func (c *Compiler) addSafeOutputGitHubToken(steps *[]string, data *WorkflowData) {
	var safeOutputsToken string
	if data.SafeOutputs != nil {
		safeOutputsToken = data.SafeOutputs.GitHubToken
	}
	effectiveToken := getEffectiveGitHubToken(safeOutputsToken, data.GitHubToken)
	*steps = append(*steps, fmt.Sprintf("          github-token: %s\n", effectiveToken))
}

// addSafeOutputGitHubTokenForConfig adds github-token to the with section, preferring per-config token over global
// Uses precedence: config token > safe-outputs global github-token > top-level github-token > default
func (c *Compiler) addSafeOutputGitHubTokenForConfig(steps *[]string, data *WorkflowData, configToken string) {
	var safeOutputsToken string
	if data.SafeOutputs != nil {
		safeOutputsToken = data.SafeOutputs.GitHubToken
	}
	// Get effective token using double precedence: config > safe-outputs, then > top-level > default
	effectiveToken := getEffectiveGitHubToken(configToken, getEffectiveGitHubToken(safeOutputsToken, data.GitHubToken))
	*steps = append(*steps, fmt.Sprintf("          github-token: %s\n", effectiveToken))
}

// addSafeOutputCopilotGitHubTokenForConfig adds github-token to the with section for Copilot-related operations
// Uses precedence: config token > safe-outputs global github-token > top-level github-token > GH_AW_COPILOT_TOKEN > GH_AW_GITHUB_TOKEN > GITHUB_TOKEN
func (c *Compiler) addSafeOutputCopilotGitHubTokenForConfig(steps *[]string, data *WorkflowData, configToken string) {
	var safeOutputsToken string
	if data.SafeOutputs != nil {
		safeOutputsToken = data.SafeOutputs.GitHubToken
	}
	// Get effective token using double precedence: config > safe-outputs, then > top-level > Copilot default
	effectiveToken := getEffectiveCopilotGitHubToken(configToken, getEffectiveCopilotGitHubToken(safeOutputsToken, data.GitHubToken))
	*steps = append(*steps, fmt.Sprintf("          github-token: %s\n", effectiveToken))
}

// filterMapKeys creates a new map excluding the specified keys
func filterMapKeys(original map[string]any, excludeKeys ...string) map[string]any {
	excludeSet := make(map[string]bool)
	for _, key := range excludeKeys {
		excludeSet[key] = true
	}

	result := make(map[string]any)
	for key, value := range original {
		if !excludeSet[key] {
			result[key] = value
		}
	}
	return result
}

// extractYAMLValue extracts a scalar value from the frontmatter map
func (c *Compiler) extractYAMLValue(frontmatter map[string]any, key string) string {
	if value, exists := frontmatter[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
		if num, ok := value.(int); ok {
			return fmt.Sprintf("%d", num)
		}
		if num, ok := value.(int64); ok {
			return fmt.Sprintf("%d", num)
		}
		if num, ok := value.(uint64); ok {
			return fmt.Sprintf("%d", num)
		}
		if float, ok := value.(float64); ok {
			return fmt.Sprintf("%.0f", float)
		}
	}
	return ""
}

// indentYAMLLines adds indentation to all lines of a multi-line YAML string except the first
func (c *Compiler) indentYAMLLines(yamlContent, indent string) string {
	if yamlContent == "" {
		return yamlContent
	}

	lines := strings.Split(yamlContent, "\n")
	if len(lines) <= 1 {
		return yamlContent
	}

	// First line doesn't get additional indentation
	result := lines[0]
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != "" {
			result += "\n" + indent + lines[i]
		} else {
			result += "\n" + lines[i]
		}
	}

	return result
}

// extractTopLevelYAMLSection extracts a top-level YAML section from frontmatter
func (c *Compiler) extractTopLevelYAMLSection(frontmatter map[string]any, key string) string {
	value, exists := frontmatter[key]
	if !exists {
		return ""
	}

	// Convert the value back to YAML format with field ordering
	var yamlBytes []byte
	var err error

	// Check if value is a map that we should order alphabetically
	if valueMap, ok := value.(map[string]any); ok {
		// Use OrderMapFields for alphabetical sorting (empty priority list = all alphabetical)
		orderedValue := OrderMapFields(valueMap, []string{})
		// Wrap the ordered value with the key using MapSlice
		wrappedData := yaml.MapSlice{{Key: key, Value: orderedValue}}
		yamlBytes, err = yaml.MarshalWithOptions(wrappedData,
			yaml.Indent(2),                        // Use 2-space indentation
			yaml.UseLiteralStyleIfMultiline(true), // Use literal block scalars for multiline strings
		)
		if err != nil {
			return ""
		}
	} else {
		// Use standard marshaling for non-map types
		yamlBytes, err = yaml.Marshal(map[string]any{key: value})
		if err != nil {
			return ""
		}
	}

	yamlStr := string(yamlBytes)
	// Remove the trailing newline
	yamlStr = strings.TrimSuffix(yamlStr, "\n")

	// Clean up quoted keys - replace "key": with key: at the start of a line
	// Don't unquote "on" key as it's a YAML boolean keyword and must remain quoted
	if key != "on" {
		yamlStr = UnquoteYAMLKey(yamlStr, key)
	}

	// Special handling for "on" section - comment out draft and fork fields from pull_request
	if key == "on" {
		yamlStr = c.commentOutProcessedFieldsInOnSection(yamlStr)
	}

	return yamlStr
}

// commentOutProcessedFieldsInOnSection comments out draft, fork, forks, and names fields in pull_request/issues sections within the YAML string
// These fields are processed separately by applyPullRequestDraftFilter, applyPullRequestForkFilter, and applyLabelFilter and should be commented for documentation
func (c *Compiler) commentOutProcessedFieldsInOnSection(yamlStr string) string {
	lines := strings.Split(yamlStr, "\n")
	var result []string
	inPullRequest := false
	inIssues := false
	inForksArray := false

	for _, line := range lines {
		// Check if we're entering a pull_request or issues section
		if strings.Contains(line, "pull_request:") {
			inPullRequest = true
			inIssues = false
			result = append(result, line)
			continue
		}
		if strings.Contains(line, "issues:") {
			inIssues = true
			inPullRequest = false
			result = append(result, line)
			continue
		}

		// Check if we're leaving the pull_request or issues section (new top-level key or end of indent)
		if inPullRequest || inIssues {
			// If line is not indented or is a new top-level key, we're out of the section
			if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "    ") && !strings.HasPrefix(line, "\t") {
				inPullRequest = false
				inIssues = false
				inForksArray = false
			}
		}

		trimmedLine := strings.TrimSpace(line)

		// Check if we're entering the forks array
		if inPullRequest && strings.HasPrefix(trimmedLine, "forks:") {
			inForksArray = true
		}

		// Check if we're leaving the forks array by encountering another top-level field at the same level
		if inForksArray && inPullRequest && strings.TrimSpace(line) != "" {
			// Get the indentation of the current line
			lineIndent := len(line) - len(strings.TrimLeft(line, " \t"))

			// If this is a non-dash line at the same level as the forks field (4 spaces), we're out of the array
			if lineIndent == 4 && !strings.HasPrefix(trimmedLine, "-") && !strings.HasPrefix(trimmedLine, "forks:") {
				inForksArray = false
			}
		}

		// Determine if we should comment out this line
		shouldComment := false
		var commentReason string

		if inPullRequest && strings.Contains(trimmedLine, "draft:") {
			shouldComment = true
			commentReason = " # Draft filtering applied via job conditions"
		} else if inPullRequest && strings.HasPrefix(trimmedLine, "forks:") {
			shouldComment = true
			commentReason = " # Fork filtering applied via job conditions"
		} else if inForksArray && strings.HasPrefix(trimmedLine, "-") {
			shouldComment = true
			commentReason = " # Fork filtering applied via job conditions"
		} else if (inPullRequest || inIssues) && strings.HasPrefix(trimmedLine, "names:") {
			shouldComment = true
			commentReason = " # Label filtering applied via job conditions"
		} else if (inPullRequest || inIssues) && line != "" {
			// Check if we're in a names array (after "names:" line)
			// Look back to see if the previous uncommented line was "names:"
			if len(result) > 0 {
				for i := len(result) - 1; i >= 0; i-- {
					prevLine := result[i]
					prevTrimmed := strings.TrimSpace(prevLine)

					// Skip empty lines
					if prevTrimmed == "" {
						continue
					}

					// If we find "names:", and current line is an array item, comment it
					if strings.Contains(prevTrimmed, "names:") && strings.Contains(prevTrimmed, "# Label filtering") {
						if strings.HasPrefix(trimmedLine, "-") {
							shouldComment = true
							commentReason = " # Label filtering applied via job conditions"
						}
						break
					}

					// If we find a different field or commented names array item, break
					if !strings.HasPrefix(prevTrimmed, "#") || !strings.Contains(prevTrimmed, "Label filtering") {
						break
					}

					// If it's a commented names array item, continue
					if strings.HasPrefix(prevTrimmed, "# -") && strings.Contains(prevTrimmed, "Label filtering") {
						if strings.HasPrefix(trimmedLine, "-") {
							shouldComment = true
							commentReason = " # Label filtering applied via job conditions"
						}
						continue
					}

					break
				}
			}
		}

		if shouldComment {
			// Preserve the original indentation and comment out the line
			indentation := ""
			trimmed := strings.TrimLeft(line, " \t")
			if len(line) > len(trimmed) {
				indentation = line[:len(line)-len(trimmed)]
			}

			commentedLine := indentation + "# " + trimmed + commentReason
			result = append(result, commentedLine)
		} else {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// extractPermissions extracts permissions from frontmatter using the permission parser
func (c *Compiler) extractPermissions(frontmatter map[string]any) string {
	permissionsValue, exists := frontmatter["permissions"]
	if !exists {
		return ""
	}

	// Check if this is an "all: read" case by using the parser
	parser := NewPermissionsParserFromValue(permissionsValue)

	// If it's "all: read", use the parser to expand it
	if parser.hasAll && parser.allLevel == "read" {
		permissions := parser.ToPermissions()
		yaml := permissions.RenderToYAML()

		// Adjust indentation from 6 spaces to 2 spaces for workflow-level permissions
		// RenderToYAML uses 6 spaces for job-level rendering
		lines := strings.Split(yaml, "\n")
		for i := 1; i < len(lines); i++ {
			if strings.HasPrefix(lines[i], "      ") {
				lines[i] = "  " + lines[i][6:]
			}
		}
		return strings.Join(lines, "\n")
	}

	// For all other cases, use standard extraction
	return c.extractTopLevelYAMLSection(frontmatter, "permissions")
}

// extractIfCondition extracts the if condition from frontmatter, returning just the expression
// without the "if: " prefix
func (c *Compiler) extractIfCondition(frontmatter map[string]any) string {
	value, exists := frontmatter["if"]
	if !exists {
		return ""
	}

	// Convert the value to string - it should be just the expression
	if strValue, ok := value.(string); ok {
		return c.extractExpressionFromIfString(strValue)
	}

	return ""
}

// extractFeatures extracts the features field from frontmatter
// Returns a map of feature flags (feature name -> enabled)
func (c *Compiler) extractFeatures(frontmatter map[string]any) map[string]bool {
	value, exists := frontmatter["features"]
	if !exists {
		return nil
	}

	// Features should be an object with boolean values
	if featuresMap, ok := value.(map[string]any); ok {
		result := make(map[string]bool)
		for key, val := range featuresMap {
			// Convert value to boolean
			if boolVal, ok := val.(bool); ok {
				result[key] = boolVal
			}
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

// extractSafetyPromptSetting extracts the safety-prompt setting from tools
// Returns true by default (safety prompt is enabled by default)
func (c *Compiler) extractSafetyPromptSetting(tools map[string]any) bool {
	if tools == nil {
		return true // Default is enabled
	}

	// Check if safety-prompt is explicitly set in tools
	if safetyPromptValue, exists := tools["safety-prompt"]; exists {
		if boolValue, ok := safetyPromptValue.(bool); ok {
			return boolValue
		}
	}

	// Default to true (enabled)
	return true
}

// extractToolsTimeout extracts the timeout setting from tools
// Returns 0 if not set (engines will use their own defaults)
func (c *Compiler) extractToolsTimeout(tools map[string]any) int {
	if tools == nil {
		return 0 // Use engine defaults
	}

	// Check if timeout is explicitly set in tools
	if timeoutValue, exists := tools["timeout"]; exists {
		// Handle different numeric types
		switch v := timeoutValue.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case uint:
			return int(v)
		case uint64:
			return int(v)
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
		// Handle different numeric types
		switch v := timeoutValue.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case uint:
			return int(v)
		case uint64:
			return int(v)
		case float64:
			return int(v)
		}
	}

	// Default to 0 (use engine defaults)
	return 0
}

// extractExpressionFromIfString extracts the expression part from a string that might
// contain "if: expression" or just "expression", returning just the expression
func (c *Compiler) extractExpressionFromIfString(ifString string) string {
	if ifString == "" {
		return ""
	}

	// Check if the string starts with "if: " and strip it
	if strings.HasPrefix(ifString, "if: ") {
		return strings.TrimSpace(ifString[4:]) // Remove "if: " prefix
	}

	// Return the string as-is (it's just the expression)
	return ifString
}

// extractCommandConfig extracts command configuration from frontmatter including name and events
func (c *Compiler) extractCommandConfig(frontmatter map[string]any) (commandName string, commandEvents []string) {
	// Check new format: on.command or on.command.name
	if onValue, exists := frontmatter["on"]; exists {
		if onMap, ok := onValue.(map[string]any); ok {
			if commandValue, hasCommand := onMap["command"]; hasCommand {
				// Check if command is a string (shorthand format)
				if commandStr, ok := commandValue.(string); ok {
					return commandStr, nil // nil means default (all events)
				}
				// Check if command is a map with a name key (object format)
				if commandMap, ok := commandValue.(map[string]any); ok {
					var name string
					var events []string

					if nameValue, hasName := commandMap["name"]; hasName {
						if nameStr, ok := nameValue.(string); ok {
							name = nameStr
						}
					}

					// Extract events field
					if eventsValue, hasEvents := commandMap["events"]; hasEvents {
						events = ParseCommandEvents(eventsValue)
					}

					return name, events
				}
			}
		}
	}

	return "", nil
}

// extractNetworkPermissions extracts network permissions from frontmatter
func (c *Compiler) extractNetworkPermissions(frontmatter map[string]any) *NetworkPermissions {
	if network, exists := frontmatter["network"]; exists {
		// Handle string format: "defaults"
		if networkStr, ok := network.(string); ok {
			if networkStr == "defaults" {
				return &NetworkPermissions{
					Mode: "defaults",
				}
			}
			// Unknown string format, return nil
			return nil
		}

		// Handle object format: { allowed: [...], firewall: ... } or {}
		if networkObj, ok := network.(map[string]any); ok {
			permissions := &NetworkPermissions{}

			// Extract allowed domains if present
			if allowed, hasAllowed := networkObj["allowed"]; hasAllowed {
				if allowedSlice, ok := allowed.([]any); ok {
					for _, domain := range allowedSlice {
						if domainStr, ok := domain.(string); ok {
							permissions.Allowed = append(permissions.Allowed, domainStr)
						}
					}
				}
			}

			// Extract firewall configuration if present
			if firewall, hasFirewall := networkObj["firewall"]; hasFirewall {
				permissions.Firewall = c.extractFirewallConfig(firewall)
			}

			// Empty object {} means no network access (empty allowed list)
			return permissions
		}
	}
	return nil
}

// extractFirewallConfig extracts firewall configuration from various formats
func (c *Compiler) extractFirewallConfig(firewall any) *FirewallConfig {
	// Handle null/empty object format: firewall: or firewall: {}
	if firewall == nil {
		return &FirewallConfig{
			Enabled: true,
		}
	}

	// Handle boolean format: firewall: true or firewall: false
	if firewallBool, ok := firewall.(bool); ok {
		return &FirewallConfig{
			Enabled: firewallBool,
		}
	}

	// Handle string format: firewall: "disable"
	if firewallStr, ok := firewall.(string); ok {
		if firewallStr == "disable" {
			return &FirewallConfig{
				Enabled: false,
			}
		}
		// Unknown string format, return nil
		return nil
	}

	// Handle object format: firewall: { args: [...], version: "..." }
	if firewallObj, ok := firewall.(map[string]any); ok {
		config := &FirewallConfig{
			Enabled: true, // Default to enabled when object is specified
		}

		// Extract args if present
		if args, hasArgs := firewallObj["args"]; hasArgs {
			if argsSlice, ok := args.([]any); ok {
				for _, arg := range argsSlice {
					if argStr, ok := arg.(string); ok {
						config.Args = append(config.Args, argStr)
					}
				}
			}
		}

		// Extract version if present
		if version, hasVersion := firewallObj["version"]; hasVersion {
			if versionStr, ok := version.(string); ok {
				config.Version = versionStr
			}
		}

		// Extract log-level if present
		if logLevel, hasLogLevel := firewallObj["log-level"]; hasLogLevel {
			if logLevelStr, ok := logLevel.(string); ok {
				config.LogLevel = logLevelStr
			}
		}

		return config
	}

	return nil
}
