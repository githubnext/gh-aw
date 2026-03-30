package parser

import (
	"fmt"
	"regexp"
	"sort"
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
		log.Printf("Translating minimum constraint message: got=%s want=%s", m[1], m[2])
		return fmt.Sprintf("must be at least %s (got %s)", m[2], m[1])
	}
	if m := maxConstraintPattern.FindStringSubmatch(message); len(m) == 3 {
		log.Printf("Translating maximum constraint message: got=%s want=%s", m[1], m[2])
		return fmt.Sprintf("must be at most %s (got %s)", m[2], m[1])
	}
	return message
}

// cleanJSONSchemaErrorMessage removes unhelpful prefixes from jsonschema validation errors
func cleanJSONSchemaErrorMessage(errorMsg string) string {
	log.Printf("Cleaning JSON schema error message (%d chars)", len(errorMsg))
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

	log.Printf("Simplifying oneOf error message (%d lines)", len(strings.Split(message, "\n")))
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
	if _, after, ok := strings.Cut(line, ": got "); ok {
		afterGot := after
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
	log.Printf("Finding frontmatter bounds in %d lines", len(lines))
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
		log.Print("No frontmatter opening delimiter found")
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
		log.Print("No frontmatter closing delimiter found")
		return -1, -1, ""
	}
	log.Printf("Found frontmatter bounds: start=%d end=%d", startIdx, endIdx)

	// Extract frontmatter content between the markers
	frontmatterLines := lines[startIdx+1 : endIdx]
	frontmatterContent = strings.Join(frontmatterLines, "\n")

	return startIdx, endIdx, frontmatterContent
}

// knownFieldValidValues maps well-known JSON schema paths to a human-readable description
// of the valid values / children for that field. Used to append helpful hints when an
// additionalProperties error occurs on these fields so users quickly know what is allowed.
//
// The permissions scope list mirrors the properties defined in main_workflow_schema.json
// under permissions.oneOf[1].properties. Update this list when the schema changes.
var knownFieldValidValues = map[string]string{
	// This list mirrors permissions.oneOf[1].properties in main_workflow_schema.json.
	// Update both when the schema changes.
	"/permissions": "Valid permission scopes: actions, all, attestations, checks, contents, deployments, discussions, id-token, issues, metadata, models, organization-projects, packages, pages, pull-requests, repository-projects, security-events, statuses, vulnerability-alerts",
}

// knownFieldScopes maps well-known JSON schema paths to a slice of valid scope names.
// This enables spell-check ("Did you mean?") suggestions for unknown-property errors.
//
// The permissions scope list mirrors permissions.oneOf[1].properties in main_workflow_schema.json.
// Update both when the schema changes.
var knownFieldScopes = map[string][]string{
	"/permissions": {
		"actions", "all", "attestations", "checks", "contents", "deployments",
		"discussions", "id-token", "issues", "metadata", "models",
		"organization-projects", "packages", "pages", "pull-requests",
		"repository-projects", "security-events", "statuses", "vulnerability-alerts",
	},
}

// knownFieldDocs maps well-known JSON schema paths to documentation URLs.
var knownFieldDocs = map[string]string{
	"/permissions": "https://docs.github.com/en/actions/writing-workflows/choosing-what-your-workflow-does/controlling-permissions-for-github_token",
}

// unknownPropertyPattern extracts the property name(s) from a rewritten "Unknown property(ies):" message.
var unknownPropertyPattern = regexp.MustCompile(`(?i)^Unknown propert(?:y|ies): (.+)$`)

// appendKnownFieldValidValuesHint appends a "Valid values: …" hint, "Did you mean?" suggestions,
// and a documentation link to message when the jsonPath matches a well-known field and the
// message is an unknown-property error.
// It returns the message unchanged for unknown paths or non-additional-properties messages.
func appendKnownFieldValidValuesHint(message string, jsonPath string) string {
	// Use truncated prefix "unknown propert" to match both singular ("Unknown property")
	// and plural ("Unknown properties") forms produced by rewriteAdditionalPropertiesError.
	if !strings.Contains(strings.ToLower(message), "unknown propert") {
		return message
	}
	log.Printf("Appending known field hint for path: %s", jsonPath)

	// Find the best matching known path: exact match first, then the longest matching parent.
	hint, hintOK := knownFieldValidValues[jsonPath]
	scopes := knownFieldScopes[jsonPath]
	docsURL := knownFieldDocs[jsonPath]
	if !hintOK {
		// Select the longest matching parent path deterministically to avoid
		// random map iteration order when multiple known paths share a common prefix.
		bestPath := ""
		bestLen := 0
		for path := range knownFieldValidValues {
			if strings.HasPrefix(jsonPath, path+"/") {
				if l := len(path); l > bestLen {
					bestLen = l
					bestPath = path
				}
			}
		}
		if bestPath != "" {
			hint = knownFieldValidValues[bestPath]
			scopes = knownFieldScopes[bestPath]
			docsURL = knownFieldDocs[bestPath]
			hintOK = true
		}
	}
	if !hintOK {
		return message
	}

	result := message + " (" + hint + ")"

	// Add "Did you mean?" suggestions when the unknown property name is close to a valid scope.
	if len(scopes) > 0 {
		// unknownPropertyPattern has exactly one capture group, so a successful match
		// returns [fullMatch, captureGroup1], giving len(m) == 2.
		if m := unknownPropertyPattern.FindStringSubmatch(message); len(m) == 2 {
			unknownProps := strings.Split(m[1], ", ")
			var allSuggestions []string
			for _, prop := range unknownProps {
				prop = strings.TrimSpace(prop)
				if prop == "" {
					continue
				}
				// maxClosestMatches is defined in schema_suggestions.go in the same package.
				closest := FindClosestMatches(prop, scopes, maxClosestMatches)
				allSuggestions = append(allSuggestions, closest...)
			}
			// Deduplicate suggestions
			seen := make(map[string]bool)
			var unique []string
			for _, s := range allSuggestions {
				if !seen[s] {
					seen[s] = true
					unique = append(unique, s)
				}
			}
			if len(unique) == 1 {
				result = fmt.Sprintf("%s. Did you mean '%s'?", result, unique[0])
			} else if len(unique) > 1 {
				result = fmt.Sprintf("%s. Did you mean: %s?", result, strings.Join(unique, ", "))
			}
		}
	}

	// Append documentation link on the same line to avoid breaking bullet-list formatting
	// when this message is embedded in "Multiple schema validation failures:" output.
	if docsURL != "" {
		result = fmt.Sprintf("%s See: %s", result, docsURL)
	}

	return result
}

// rewriteAdditionalPropertiesError rewrites "additional properties not allowed" errors to be more user-friendly
func rewriteAdditionalPropertiesError(message string) string {
	// Check if this is an "additional properties not allowed" error
	if strings.Contains(strings.ToLower(message), "additional propert") && strings.Contains(strings.ToLower(message), "not allowed") {
		// Extract property names from the message using regex
		re := regexp.MustCompile(`additional propert(?:y|ies) (.+?) not allowed`)
		match := re.FindStringSubmatch(message)

		if len(match) >= 2 {
			properties := normalizeAdditionalPropertyList(match[1])
			log.Printf("Rewriting additional properties error: %s", properties)

			if strings.Contains(properties, ",") {
				return "Unknown properties: " + properties
			} else {
				return "Unknown property: " + properties
			}
		}
	}

	return message
}

// normalizeAdditionalPropertyList strips quotes, trims whitespace, and sorts the
// comma-separated property names so that diagnostics are deterministic regardless
// of the order in which the schema validator emits them.
func normalizeAdditionalPropertyList(raw string) string {
	raw = strings.ReplaceAll(raw, "'", "")
	parts := strings.Split(raw, ",")
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	sort.Strings(cleaned)
	return strings.Join(cleaned, ", ")
}
