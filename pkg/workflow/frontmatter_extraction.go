package workflow

import (
	"fmt"
	"os"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/goccy/go-yaml"
)

var frontmatterLog = logger.New("workflow:frontmatter_extraction")

// Note: extractStringValue, parseIntValue, and filterMapKeys have been moved to map_helpers.go
// Note: addCustomSafeOutputEnvVars, addSafeOutputGitHubToken, addSafeOutputGitHubTokenForConfig,
//       and addSafeOutputCopilotGitHubTokenForConfig have been moved to safe_outputs_env.go

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

	frontmatterLog.Printf("Extracting YAML section: %s", key)

	// Convert the value back to YAML format with field ordering
	var yamlBytes []byte
	var err error

	// Check if value is a map that we should order alphabetically
	if valueMap, ok := value.(map[string]any); ok {
		// Use OrderMapFields for alphabetical sorting (empty priority list = all alphabetical)
		orderedValue := OrderMapFields(valueMap, []string{})
		// Wrap the ordered value with the key using MapSlice
		wrappedData := yaml.MapSlice{{Key: key, Value: orderedValue}}
		yamlBytes, err = yaml.MarshalWithOptions(wrappedData, DefaultMarshalOptions...)
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

	// Post-process YAML to ensure cron expressions are quoted
	// The YAML library may drop quotes from cron expressions like "0 14 * * 1-5"
	// which causes validation errors since they start with numbers but contain spaces
	yamlStr = parser.QuoteCronExpressions(yamlStr)

	// Clean up null values - replace `: null` with `:` for cleaner output
	// GitHub Actions treats `workflow_dispatch:` and `workflow_dispatch: null` identically
	yamlStr = CleanYAMLNullValues(yamlStr)

	// Clean up quoted keys - replace "key": with key: at the start of a line
	// Don't unquote "on" key as it's a YAML boolean keyword and must remain quoted
	if key != "on" {
		yamlStr = UnquoteYAMLKey(yamlStr, key)
	}

	// Special handling for "on" section - comment out draft and fork fields from pull_request
	if key == "on" {
		yamlStr = c.commentOutProcessedFieldsInOnSection(yamlStr, frontmatter)
		// Add zizmor ignore comment if workflow_run trigger is present
		yamlStr = c.addZizmorIgnoreForWorkflowRun(yamlStr)
		// Add friendly format comments for schedule cron expressions
		yamlStr = c.addFriendlyScheduleComments(yamlStr, frontmatter)
	}

	return yamlStr
}

// commentOutProcessedFieldsInOnSection comments out draft, fork, forks, names, manual-approval, stop-after, skip-if-match, reaction, and lock-for-agent fields in the on section
// These fields are processed separately and should be commented for documentation
// Exception: names fields in sections with __gh_aw_native_label_filter__ marker in frontmatter are NOT commented out
func (c *Compiler) commentOutProcessedFieldsInOnSection(yamlStr string, frontmatter map[string]any) string {
	frontmatterLog.Print("Processing 'on' section to comment out processed fields")

	// Check frontmatter for native label filter markers
	nativeLabelFilterSections := make(map[string]bool)
	if onValue, exists := frontmatter["on"]; exists {
		if onMap, ok := onValue.(map[string]any); ok {
			for _, sectionKey := range []string{"issues", "pull_request", "discussion", "issue_comment"} {
				if sectionValue, hasSec := onMap[sectionKey]; hasSec {
					if sectionMap, ok := sectionValue.(map[string]any); ok {
						if marker, hasMarker := sectionMap["__gh_aw_native_label_filter__"]; hasMarker {
							if useNative, ok := marker.(bool); ok && useNative {
								nativeLabelFilterSections[sectionKey] = true
								frontmatterLog.Printf("Section %s uses native label filtering", sectionKey)
							}
						}
					}
				}
			}
		}
	}

	lines := strings.Split(yamlStr, "\n")
	var result []string
	inPullRequest := false
	inIssues := false
	inDiscussion := false
	inIssueComment := false
	inForksArray := false
	inSkipIfMatch := false
	currentSection := "" // Track which section we're in ("issues", "pull_request", "discussion", or "issue_comment")

	for _, line := range lines {
		// Check if we're entering a pull_request, issues, discussion, or issue_comment section
		if strings.Contains(line, "pull_request:") {
			inPullRequest = true
			inIssues = false
			inDiscussion = false
			inIssueComment = false
			currentSection = "pull_request"
			result = append(result, line)
			continue
		}
		if strings.Contains(line, "issues:") {
			inIssues = true
			inPullRequest = false
			inDiscussion = false
			inIssueComment = false
			currentSection = "issues"
			result = append(result, line)
			continue
		}
		if strings.Contains(line, "discussion:") {
			inDiscussion = true
			inPullRequest = false
			inIssues = false
			inIssueComment = false
			currentSection = "discussion"
			result = append(result, line)
			continue
		}
		if strings.Contains(line, "issue_comment:") {
			inIssueComment = true
			inPullRequest = false
			inIssues = false
			inDiscussion = false
			currentSection = "issue_comment"
			result = append(result, line)
			continue
		}

		// Check if we're leaving the pull_request, issues, discussion, or issue_comment section (new top-level key or end of indent)
		if inPullRequest || inIssues || inDiscussion || inIssueComment {
			// If line is not indented or is a new top-level key, we're out of the section
			if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "    ") && !strings.HasPrefix(line, "\t") {
				inPullRequest = false
				inIssues = false
				inDiscussion = false
				inIssueComment = false
				inForksArray = false
				currentSection = ""
			}
		}

		trimmedLine := strings.TrimSpace(line)

		// Skip marker lines in the YAML output
		if (inPullRequest || inIssues || inDiscussion || inIssueComment) && strings.Contains(trimmedLine, "__gh_aw_native_label_filter__:") {
			// Don't include the marker line in the output
			continue
		}

		// Check if we're entering the forks array
		if inPullRequest && strings.HasPrefix(trimmedLine, "forks:") {
			inForksArray = true
		}

		// Check if we're entering skip-if-match object
		if !inPullRequest && !inIssues && !inDiscussion && !inIssueComment && !inSkipIfMatch {
			// Check both uncommented and commented forms
			if (strings.HasPrefix(trimmedLine, "skip-if-match:") && trimmedLine == "skip-if-match:") ||
				(strings.HasPrefix(trimmedLine, "# skip-if-match:") && strings.Contains(trimmedLine, "pre-activation job")) {
				inSkipIfMatch = true
			}
		}

		// Check if we're leaving skip-if-match object (encountering another top-level field)
		// Skip this check if we just entered skip-if-match on this line
		if inSkipIfMatch && strings.TrimSpace(line) != "" &&
			!strings.HasPrefix(trimmedLine, "skip-if-match:") &&
			!strings.HasPrefix(trimmedLine, "# skip-if-match:") {
			// Get the indentation of the current line
			lineIndent := len(line) - len(strings.TrimLeft(line, " \t"))
			// If this is a field at same level as skip-if-match (2 spaces) and not a comment, we're out of skip-if-match
			if lineIndent == 2 && !strings.HasPrefix(trimmedLine, "#") {
				inSkipIfMatch = false
			}
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

		// Check for top-level fields that should be commented out (not inside pull_request, issues, discussion, or issue_comment)
		if !inPullRequest && !inIssues && !inDiscussion && !inIssueComment {
			if strings.HasPrefix(trimmedLine, "manual-approval:") {
				shouldComment = true
				commentReason = " # Manual approval processed as environment field in activation job"
			} else if strings.HasPrefix(trimmedLine, "stop-after:") {
				shouldComment = true
				commentReason = " # Stop-after processed as stop-time check in pre-activation job"
			} else if strings.HasPrefix(trimmedLine, "skip-if-match:") {
				shouldComment = true
				commentReason = " # Skip-if-match processed as search check in pre-activation job"
			} else if inSkipIfMatch && (strings.HasPrefix(trimmedLine, "query:") || strings.HasPrefix(trimmedLine, "max:")) {
				// Comment out nested fields in skip-if-match object
				shouldComment = true
				commentReason = ""
			} else if strings.HasPrefix(trimmedLine, "reaction:") {
				shouldComment = true
				commentReason = " # Reaction processed as activation job step"
			}
		}

		if !shouldComment && inPullRequest && strings.Contains(trimmedLine, "draft:") {
			shouldComment = true
			commentReason = " # Draft filtering applied via job conditions"
		} else if inPullRequest && strings.HasPrefix(trimmedLine, "forks:") {
			shouldComment = true
			commentReason = " # Fork filtering applied via job conditions"
		} else if inForksArray && strings.HasPrefix(trimmedLine, "-") {
			shouldComment = true
			commentReason = " # Fork filtering applied via job conditions"
		} else if (inPullRequest || inIssues || inDiscussion || inIssueComment) && strings.HasPrefix(trimmedLine, "lock-for-agent:") {
			shouldComment = true
			commentReason = " # Lock-for-agent processed as issue locking in activation job"
		} else if (inPullRequest || inIssues || inDiscussion || inIssueComment) && strings.HasPrefix(trimmedLine, "names:") {
			// Only comment out names if NOT using native label filtering for this section
			if !nativeLabelFilterSections[currentSection] {
				shouldComment = true
				commentReason = " # Label filtering applied via job conditions"
			}
		} else if (inPullRequest || inIssues || inDiscussion || inIssueComment) && line != "" {
			// Check if we're in a names array (after "names:" line)
			// Look back to see if the previous uncommented line was "names:"
			// Only do this if NOT using native label filtering for this section
			if !nativeLabelFilterSections[currentSection] {
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
			} // Close native filter check
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
		frontmatterLog.Print("Expanding 'all: read' permissions to individual scopes")
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
		enabledCount := 0
		for key, val := range featuresMap {
			// Convert value to boolean
			if boolVal, ok := val.(bool); ok {
				result[key] = boolVal
				if boolVal {
					enabledCount++
				}
			}
		}
		if log.Enabled() {
			frontmatterLog.Printf("Extracted %d features (%d enabled)", len(result), enabledCount)
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
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') || char == '-' || char == '_') {
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
	// Check new format: on.slash_command or on.slash_command.name (preferred)
	// Also check legacy format: on.command or on.command.name (deprecated)
	if onValue, exists := frontmatter["on"]; exists {
		if onMap, ok := onValue.(map[string]any); ok {
			var commandValue any
			var hasCommand bool
			var isDeprecated bool

			// Check for slash_command first (preferred)
			if slashCommandValue, hasSlashCommand := onMap["slash_command"]; hasSlashCommand {
				commandValue = slashCommandValue
				hasCommand = true
				isDeprecated = false
			} else if legacyCommandValue, hasLegacyCommand := onMap["command"]; hasLegacyCommand {
				// Fall back to command (deprecated)
				commandValue = legacyCommandValue
				hasCommand = true
				isDeprecated = true
			}

			if hasCommand {
				// Show deprecation warning if using old field name
				if isDeprecated {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage("The 'command:' trigger field is deprecated. Please use 'slash_command:' instead."))
					c.IncrementWarningCount()
				}

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
					Mode:              "defaults",
					ExplicitlyDefined: true,
				}
			}
			// Unknown string format, return nil
			return nil
		}

		// Handle object format: { allowed: [...], firewall: ... } or {}
		if networkObj, ok := network.(map[string]any); ok {
			permissions := &NetworkPermissions{
				ExplicitlyDefined: true,
			}

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

// extractSandboxConfig extracts sandbox configuration from front matter
func (c *Compiler) extractSandboxConfig(frontmatter map[string]any) *SandboxConfig {
	sandbox, exists := frontmatter["sandbox"]
	if !exists {
		return nil
	}

	// Handle legacy string format: "default" or "sandbox-runtime"
	if sandboxStr, ok := sandbox.(string); ok {
		sandboxType := SandboxType(sandboxStr)
		if isSupportedSandboxType(sandboxType) {
			return &SandboxConfig{
				Type: sandboxType,
			}
		}
		// Unknown string format, return nil
		return nil
	}

	// Handle object format
	sandboxObj, ok := sandbox.(map[string]any)
	if !ok {
		return nil
	}

	config := &SandboxConfig{}

	// Check for new format: { agent: ..., mcp: ... }
	if agentVal, hasAgent := sandboxObj["agent"]; hasAgent {
		config.Agent = c.extractAgentSandboxConfig(agentVal)
	}

	if mcpVal, hasMCP := sandboxObj["mcp"]; hasMCP {
		if mcpObj, ok := mcpVal.(map[string]any); ok {
			config.MCP = parseMCPGatewayTool(mcpObj)
		}
	}

	// If we found agent or mcp fields, return the new format config
	if config.Agent != nil || config.MCP != nil {
		return config
	}

	// Handle legacy object format: { type: "...", config: {...} }
	if typeVal, hasType := sandboxObj["type"]; hasType {
		if typeStr, ok := typeVal.(string); ok {
			config.Type = SandboxType(typeStr)
		}
	}

	// Extract config if present (custom SRT config)
	if configVal, hasConfig := sandboxObj["config"]; hasConfig {
		config.Config = c.extractSRTConfig(configVal)
	}

	return config
}

// extractAgentSandboxConfig extracts agent sandbox configuration
func (c *Compiler) extractAgentSandboxConfig(agentVal any) *AgentSandboxConfig {
	// Handle boolean format: false (to disable firewall)
	if agentBool, ok := agentVal.(bool); ok {
		if !agentBool {
			// agent: false means disable firewall
			return &AgentSandboxConfig{
				Disabled: true,
			}
		}
		// agent: true is not a valid configuration
		return nil
	}

	// Handle string format: "awf" or "srt"
	if agentStr, ok := agentVal.(string); ok {
		agentType := SandboxType(agentStr)
		if isSupportedSandboxType(agentType) {
			return &AgentSandboxConfig{
				Type: agentType,
			}
		}
		return nil
	}

	// Handle object format: { id/type: "...", config: {...}, command: "...", args: [...], env: {...} }
	agentObj, ok := agentVal.(map[string]any)
	if !ok {
		return nil
	}

	agentConfig := &AgentSandboxConfig{}

	// Extract ID field (new format)
	if idVal, hasID := agentObj["id"]; hasID {
		if idStr, ok := idVal.(string); ok {
			agentConfig.ID = idStr
		}
	}

	// Extract Type field (legacy format)
	if typeVal, hasType := agentObj["type"]; hasType {
		if typeStr, ok := typeVal.(string); ok {
			agentConfig.Type = SandboxType(typeStr)
		}
	}

	// Extract config for SRT
	if configVal, hasConfig := agentObj["config"]; hasConfig {
		agentConfig.Config = c.extractSRTConfig(configVal)
	}

	// Extract command (custom command to replace AWF binary download)
	if commandVal, hasCommand := agentObj["command"]; hasCommand {
		if commandStr, ok := commandVal.(string); ok {
			agentConfig.Command = commandStr
		}
	}

	// Extract args (additional arguments to append)
	if argsVal, hasArgs := agentObj["args"]; hasArgs {
		if argsSlice, ok := argsVal.([]any); ok {
			for _, arg := range argsSlice {
				if argStr, ok := arg.(string); ok {
					agentConfig.Args = append(agentConfig.Args, argStr)
				}
			}
		}
	}

	// Extract env (environment variables to set on the step)
	if envVal, hasEnv := agentObj["env"]; hasEnv {
		if envObj, ok := envVal.(map[string]any); ok {
			agentConfig.Env = make(map[string]string)
			for key, value := range envObj {
				if valueStr, ok := value.(string); ok {
					agentConfig.Env[key] = valueStr
				}
			}
		}
	}

	// Extract mounts (container mounts for AWF)
	if mountsVal, hasMounts := agentObj["mounts"]; hasMounts {
		if mountsSlice, ok := mountsVal.([]any); ok {
			for _, mount := range mountsSlice {
				if mountStr, ok := mount.(string); ok {
					agentConfig.Mounts = append(agentConfig.Mounts, mountStr)
				}
			}
		}
	}

	return agentConfig
}

// extractSRTConfig extracts Sandbox Runtime configuration from a map
func (c *Compiler) extractSRTConfig(configVal any) *SandboxRuntimeConfig {
	configObj, ok := configVal.(map[string]any)
	if !ok {
		return nil
	}

	srtConfig := &SandboxRuntimeConfig{}

	// Extract network config
	if networkVal, hasNetwork := configObj["network"]; hasNetwork {
		if networkObj, ok := networkVal.(map[string]any); ok {
			netConfig := &SRTNetworkConfig{}

			// Extract allowedDomains
			if allowedDomains, hasAllowed := networkObj["allowedDomains"]; hasAllowed {
				if domainsSlice, ok := allowedDomains.([]any); ok {
					for _, domain := range domainsSlice {
						if domainStr, ok := domain.(string); ok {
							netConfig.AllowedDomains = append(netConfig.AllowedDomains, domainStr)
						}
					}
				}
			}

			// Extract deniedDomains
			if deniedDomains, hasDenied := networkObj["deniedDomains"]; hasDenied {
				if domainsSlice, ok := deniedDomains.([]any); ok {
					for _, domain := range domainsSlice {
						if domainStr, ok := domain.(string); ok {
							netConfig.DeniedDomains = append(netConfig.DeniedDomains, domainStr)
						}
					}
				}
			}

			// Extract allowUnixSockets
			if unixSockets, hasUnixSockets := networkObj["allowUnixSockets"]; hasUnixSockets {
				if socketsSlice, ok := unixSockets.([]any); ok {
					for _, socket := range socketsSlice {
						if socketStr, ok := socket.(string); ok {
							netConfig.AllowUnixSockets = append(netConfig.AllowUnixSockets, socketStr)
						}
					}
				}
			}

			// Extract allowLocalBinding
			if allowLocalBinding, hasAllowLocalBinding := networkObj["allowLocalBinding"]; hasAllowLocalBinding {
				if bindingBool, ok := allowLocalBinding.(bool); ok {
					netConfig.AllowLocalBinding = bindingBool
				}
			}

			// Extract allowAllUnixSockets
			if allowAllUnixSockets, hasAllowAllUnixSockets := networkObj["allowAllUnixSockets"]; hasAllowAllUnixSockets {
				if unixSocketsBool, ok := allowAllUnixSockets.(bool); ok {
					netConfig.AllowAllUnixSockets = unixSocketsBool
				}
			}

			srtConfig.Network = netConfig
		}
	}

	// Extract filesystem config
	if filesystemVal, hasFilesystem := configObj["filesystem"]; hasFilesystem {
		if filesystemObj, ok := filesystemVal.(map[string]any); ok {
			fsConfig := &SRTFilesystemConfig{}

			// Extract denyRead
			if denyRead, hasDenyRead := filesystemObj["denyRead"]; hasDenyRead {
				if pathsSlice, ok := denyRead.([]any); ok {
					fsConfig.DenyRead = []string{}
					for _, path := range pathsSlice {
						if pathStr, ok := path.(string); ok {
							fsConfig.DenyRead = append(fsConfig.DenyRead, pathStr)
						}
					}
				}
			}

			// Extract allowWrite
			if allowWrite, hasAllowWrite := filesystemObj["allowWrite"]; hasAllowWrite {
				if pathsSlice, ok := allowWrite.([]any); ok {
					for _, path := range pathsSlice {
						if pathStr, ok := path.(string); ok {
							fsConfig.AllowWrite = append(fsConfig.AllowWrite, pathStr)
						}
					}
				}
			}

			// Extract denyWrite
			if denyWrite, hasDenyWrite := filesystemObj["denyWrite"]; hasDenyWrite {
				if pathsSlice, ok := denyWrite.([]any); ok {
					fsConfig.DenyWrite = []string{}
					for _, path := range pathsSlice {
						if pathStr, ok := path.(string); ok {
							fsConfig.DenyWrite = append(fsConfig.DenyWrite, pathStr)
						}
					}
				}
			}

			srtConfig.Filesystem = fsConfig
		}
	}

	// Extract ignoreViolations
	if ignoreViolations, hasIgnoreViolations := configObj["ignoreViolations"]; hasIgnoreViolations {
		if violationsObj, ok := ignoreViolations.(map[string]any); ok {
			violations := make(map[string][]string)
			for key, value := range violationsObj {
				if pathsSlice, ok := value.([]any); ok {
					var paths []string
					for _, path := range pathsSlice {
						if pathStr, ok := path.(string); ok {
							paths = append(paths, pathStr)
						}
					}
					violations[key] = paths
				}
			}
			srtConfig.IgnoreViolations = violations
		}
	}

	// Extract enableWeakerNestedSandbox
	if enableWeakerNestedSandbox, hasEnableWeaker := configObj["enableWeakerNestedSandbox"]; hasEnableWeaker {
		if weakerBool, ok := enableWeakerNestedSandbox.(bool); ok {
			srtConfig.EnableWeakerNestedSandbox = weakerBool
		}
	}

	return srtConfig
}

// addZizmorIgnoreForWorkflowRun adds a zizmor ignore comment for workflow_run triggers
// The comment is added after the workflow_run: line to suppress dangerous-triggers warnings
// since the compiler adds proper role and fork validation to secure these triggers
func (c *Compiler) addZizmorIgnoreForWorkflowRun(yamlStr string) string {
	// Check if the YAML contains workflow_run trigger
	if !strings.Contains(yamlStr, "workflow_run:") {
		return yamlStr
	}

	lines := strings.Split(yamlStr, "\n")
	var result []string
	annotationAdded := false // Track if we've already added the annotation

	for _, line := range lines {
		result = append(result, line)

		// Skip if we've already added the annotation (prevents duplicates)
		if annotationAdded {
			continue
		}

		// Check if this is a non-comment workflow_run: key at the correct YAML level
		trimmedLine := strings.TrimSpace(line)

		// Skip if the line is a comment
		if strings.HasPrefix(trimmedLine, "#") {
			continue
		}

		// Match lines that are only 'workflow_run:' (possibly with trailing whitespace or a comment)
		// e.g., 'workflow_run:', 'workflow_run: # comment', '  workflow_run:'
		// But not 'someworkflow_run:', 'workflow_run: value', etc.
		if idx := strings.Index(trimmedLine, "workflow_run:"); idx == 0 {
			after := strings.TrimSpace(trimmedLine[len("workflow_run:"):])
			// Only allow if nothing or only a comment follows
			if after == "" || strings.HasPrefix(after, "#") {
				// Get the indentation of the workflow_run line
				indentation := ""
				if len(line) > len(trimmedLine) {
					indentation = line[:len(line)-len(trimmedLine)]
				}

				// Add zizmor ignore comment with proper indentation
				// The comment explains that the trigger is secured with role and fork validation
				comment := indentation + "  # zizmor: ignore[dangerous-triggers] - workflow_run trigger is secured with role and fork validation"
				result = append(result, comment)
				annotationAdded = true
			}
		}
	}

	return strings.Join(result, "\n")
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
