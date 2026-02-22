package cli

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var expiresIntegerCodemodLog = logger.New("cli:codemod_expires_integer")

// expiresIntegerValuePattern matches an expires value that is a pure integer (possibly with a trailing comment)
var expiresIntegerValuePattern = regexp.MustCompile(`^(\s*)(\d+)(\s*)(#.*)?$`)

// getExpiresIntegerToStringCodemod creates a codemod for converting integer expires values to day strings.
// Converts e.g. "expires: 7" to "expires: 7d" in all safe-outputs types.
func getExpiresIntegerToStringCodemod() Codemod {
	return Codemod{
		ID:           "expires-integer-to-string",
		Name:         "Convert expires integer to day string",
		Description:  "Converts integer 'expires' values (e.g., 'expires: 7') to day string format (e.g., 'expires: 7d') in safe-outputs types",
		IntroducedIn: "0.13.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			// Check if safe-outputs exists
			safeOutputsValue, hasSafeOutputs := frontmatter["safe-outputs"]
			if !hasSafeOutputs {
				return content, false, nil
			}

			safeOutputsMap, ok := safeOutputsValue.(map[string]any)
			if !ok {
				return content, false, nil
			}

			// Check if any safe-outputs type has an integer expires value
			hasIntegerExpires := false
			for _, outputTypeValue := range safeOutputsMap {
				outputTypeMap, ok := outputTypeValue.(map[string]any)
				if !ok {
					continue
				}
				if expiresValue, hasExpires := outputTypeMap["expires"]; hasExpires {
					switch expiresValue.(type) {
					case int, int64, uint64:
						hasIntegerExpires = true
					}
				}
			}

			if !hasIntegerExpires {
				return content, false, nil
			}

			// Parse frontmatter to get raw lines
			frontmatterLines, markdown, err := parseFrontmatterLines(content)
			if err != nil {
				return content, false, err
			}

			// Convert integer expires values to day strings within safe-outputs blocks
			result, modified := convertExpiresIntegersToDayStrings(frontmatterLines)
			if !modified {
				return content, false, nil
			}

			// Reconstruct the content
			newContent := reconstructContent(result, markdown)
			expiresIntegerCodemodLog.Print("Applied expires integer-to-string migration")
			return newContent, true, nil
		},
	}
}

// convertExpiresIntegersToDayStrings converts integer expires values to day strings within safe-outputs blocks.
// Only affects expires lines nested inside a safe-outputs block.
func convertExpiresIntegersToDayStrings(lines []string) ([]string, bool) {
	var result []string
	var modified bool
	var inSafeOutputsBlock bool
	var safeOutputsIndent string

	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Track if we're in the safe-outputs block
		if strings.HasPrefix(trimmedLine, "safe-outputs:") {
			inSafeOutputsBlock = true
			safeOutputsIndent = getIndentation(line)
			result = append(result, line)
			continue
		}

		// Check if we've left the safe-outputs block
		if inSafeOutputsBlock && len(trimmedLine) > 0 && !strings.HasPrefix(trimmedLine, "#") {
			if hasExitedBlock(line, safeOutputsIndent) {
				inSafeOutputsBlock = false
			}
		}

		// Convert integer expires to day string if inside safe-outputs block
		if inSafeOutputsBlock && strings.HasPrefix(trimmedLine, "expires:") {
			newLine, converted := convertExpiresLineToString(line)
			if converted {
				result = append(result, newLine)
				modified = true
				expiresIntegerCodemodLog.Printf("Converted integer expires to day string on line %d", i+1)
				continue
			}
		}

		result = append(result, line)
	}

	return result, modified
}

// convertExpiresLineToString converts an expires line with an integer value to use a day string.
// For example: "    expires: 7" -> "    expires: 7d"
// Lines that already use a string format (e.g., "expires: 7d", "expires: 24h") are left unchanged.
// Returns the (possibly converted) line and whether a conversion was made.
func convertExpiresLineToString(line string) (string, bool) {
	indent := getIndentation(line)
	trimmedLine := strings.TrimSpace(line)

	// Extract the value part after "expires:"
	valuePart := strings.TrimPrefix(trimmedLine, "expires:")

	// Match an integer value optionally followed by whitespace and a comment
	matches := expiresIntegerValuePattern.FindStringSubmatch(valuePart)
	if matches == nil {
		// Not an integer value (already a string like "7d" or "false")
		return line, false
	}

	intValue := matches[2]   // the digits
	trailingWS := matches[3] // whitespace between value and comment
	comment := matches[4]    // optional trailing comment

	// Build the new line, preserving any trailing comment
	if comment != "" {
		return fmt.Sprintf("%sexpires: %sd%s%s", indent, intValue, trailingWS, comment), true
	}
	return fmt.Sprintf("%sexpires: %sd", indent, intValue), true
}
