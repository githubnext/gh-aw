package parser

import (
	"fmt"
	"regexp"
	"strings"
)

// atPathPattern matches "- at '/path': " or "at '/path': " prefixes in error messages
var atPathPattern = regexp.MustCompile(`^-?\s*at '([^']*)': (.+)$`)

// minConstraintPattern matches "minimum: got X, want Y" messages from the jsonschema library
var minConstraintPattern = regexp.MustCompile(`^minimum: got (-?\d+(?:\.\d+)?), want (-?\d+(?:\.\d+)?)$`)

// maxConstraintPattern matches "maximum: got X, want Y" messages from the jsonschema library
var maxConstraintPattern = regexp.MustCompile(`^maximum: got (-?\d+(?:\.\d+)?), want (-?\d+(?:\.\d+)?)$`)

// translateSchemaConstraintMessage rewrites jsonschema range-constraint messages into plain English.
//
// Examples:
//   - "minimum: got -45, want 1" → "must be at least 1 (got -45)"
//   - "maximum: got 120, want 60" → "must be at most 60 (got 120)"
func translateSchemaConstraintMessage(message string) string {
	if m := minConstraintPattern.FindStringSubmatch(message); len(m) == 3 {
		return fmt.Sprintf("must be at least %s (got %s)", m[2], m[1])
	}
	if m := maxConstraintPattern.FindStringSubmatch(message); len(m) == 3 {
		return fmt.Sprintf("must be at most %s (got %s)", m[2], m[1])
	}
	return message
}

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

	// Apply oneOf cleanup to the full cleaned message
	return cleanOneOfMessage(result)
}

// cleanOneOfMessage simplifies 'oneOf failed, none matched' error messages by:
// 1. Removing "got X, want Y" type-mismatch lines (from the wrong branch of a oneOf)
// 2. Removing the "oneOf failed, none matched" wrapper line
// 3. Extracting the most meaningful sub-error (e.g., enum constraint violations)
//
// This converts confusing schema jargon like:
//
//	"'oneOf' failed, none matched\n- at '/engine': value must be one of...\n- at '/engine': got string, want object"
//
// into plain language:
//
//	"value must be one of 'claude', 'codex', 'copilot', 'gemini'"
func cleanOneOfMessage(message string) string {
	if !strings.Contains(message, "'oneOf' failed") {
		return message
	}

	lines := strings.Split(message, "\n")
	var meaningful []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// Skip the "oneOf failed" wrapper line — it's schema jargon, not user guidance
		if strings.Contains(trimmed, "'oneOf' failed, none matched") {
			continue
		}
		// Skip "got X, want Y" type-mismatch lines from the wrong oneOf branch
		if isTypeConflictLine(trimmed) {
			continue
		}
		meaningful = append(meaningful, trimmed)
	}

	if len(meaningful) == 0 {
		return message // Return original if we cannot simplify
	}

	// Strip "- at '/path':" prefixes and format each remaining constraint
	var cleaned []string
	for _, line := range meaningful {
		cleaned = append(cleaned, stripAtPathPrefix(line))
	}

	return strings.Join(cleaned, "; ")
}

// isTypeConflictLine returns true for "got X, want Y" lines that arise from the
// wrong branch of a oneOf constraint. These lines are generated when the user's value
// matches one branch's type but not the other, and they are confusing to display.
// Handles both bare "got X, want Y" and embedded "- at '/path': got X, want Y" forms.
func isTypeConflictLine(line string) bool {
	// Direct "got X, want Y" format (bare form)
	if strings.HasPrefix(line, "got ") && strings.Contains(line, ", want ") {
		return true
	}
	// Embedded form: "- at '/path': got X, want Y"
	// Look for ": got " followed by ", want " later in the line
	if idx := strings.Index(line, ": got "); idx >= 0 {
		afterGot := line[idx+len(": got "):]
		return strings.Contains(afterGot, ", want ")
	}
	return false
}

// stripAtPathPrefix removes "- at '/path': " or "at '/path': " prefixes from schema error lines
// and formats nested path references to be more readable.
//
// Examples:
//   - "- at '/engine': value must be one of..." → "value must be one of..."
//   - "- at '/permissions/deployments': value must be..." → "'deployments': value must be..."
func stripAtPathPrefix(line string) string {
	match := atPathPattern.FindStringSubmatch(line)
	if match == nil {
		return line
	}
	path := match[1]
	msg := match[2]

	// For nested paths (e.g., /permissions/deployments), keep the last component
	// so users know which sub-field has the error
	if idx := strings.LastIndex(path, "/"); idx > 0 {
		subField := path[idx+1:]
		return fmt.Sprintf("'%s': %s", subField, msg)
	}

	// For top-level field errors, just return the constraint message
	return msg
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
