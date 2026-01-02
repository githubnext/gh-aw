package parser

import (
	"fmt"
	"regexp"
	"strings"
)

// cleanJSONSchemaErrorMessage removes unhelpful prefixes from jsonschema validation errors
func cleanJSONSchemaErrorMessage(errorMsg string) string {
	// Split the error message into lines
	lines := strings.Split(errorMsg, "\n")

	var cleanedLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip the "jsonschema validation failed" line entirely
		if strings.HasPrefix(line, "jsonschema validation failed") {
			continue
		}

		// Remove the unhelpful "- at '': " prefix from error descriptions
		line = strings.TrimPrefix(line, "- at '': ")

		// Keep non-empty lines that have actual content
		if line != "" {
			cleanedLines = append(cleanedLines, line)
		}
	}

	// Join the cleaned lines back together
	result := strings.Join(cleanedLines, "\n")

	// If we have no meaningful content left, return a generic message
	if strings.TrimSpace(result) == "" {
		return "schema validation failed"
	}

	return result
}

// findFrontmatterBounds finds the start and end indices of frontmatter in file lines
// Returns: startIdx (-1 if not found), endIdx (-1 if not found), frontmatterContent
func findFrontmatterBounds(lines []string) (startIdx int, endIdx int, frontmatterContent string) {
	startIdx = -1
	endIdx = -1

	// Look for the opening "---"
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			startIdx = i
			break
		}
		// Skip empty lines and comments at the beginning
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			// Found non-empty, non-comment line before "---" - no frontmatter
			return -1, -1, ""
		}
	}

	if startIdx == -1 {
		return -1, -1, ""
	}

	// Look for the closing "---"
	for i := startIdx + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "---" {
			endIdx = i
			break
		}
	}

	if endIdx == -1 {
		// No closing "---" found
		return -1, -1, ""
	}

	// Extract frontmatter content between the markers
	frontmatterLines := lines[startIdx+1 : endIdx]
	frontmatterContent = strings.Join(frontmatterLines, "\n")

	return startIdx, endIdx, frontmatterContent
}

// rewriteAdditionalPropertiesError rewrites "additional properties not allowed" errors to be more user-friendly
func rewriteAdditionalPropertiesError(message string) string {
	// Check if this is an "additional properties not allowed" error
	if strings.Contains(strings.ToLower(message), "additional propert") && strings.Contains(strings.ToLower(message), "not allowed") {
		// Extract property names from the message using regex
		re := regexp.MustCompile(`additional propert(?:y|ies) (.+?) not allowed`)
		match := re.FindStringSubmatch(message)

		if len(match) >= 2 {
			properties := match[1]
			// Clean up the property list and make it more readable
			properties = strings.ReplaceAll(properties, "'", "")

			if strings.Contains(properties, ",") {
				return fmt.Sprintf("Unknown properties: %s", properties)
			} else {
				return fmt.Sprintf("Unknown property: %s", properties)
			}
		}
	}

	return message
}
