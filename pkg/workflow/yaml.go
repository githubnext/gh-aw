package workflow

import (
	"regexp"
)

// unquoteYAMLKey removes quotes from a YAML key at the start of a line.
// This is necessary because yaml.Marshal adds quotes around reserved words like "on".
// The function only replaces the quoted key if it appears at the start of a line
// (optionally preceded by whitespace) to avoid replacing quoted strings in values.
func unquoteYAMLKey(yamlStr string, key string) string {
	// Create a regex pattern that matches the quoted key at the start of a line
	// Pattern: (start of line or newline) + (optional whitespace) + quoted key + colon
	pattern := `(^|\n)([ \t]*)"` + regexp.QuoteMeta(key) + `":`
	
	// Replacement: keep the line start and whitespace, but remove quotes from the key
	// Need to use ReplaceAllStringFunc to properly construct the replacement
	re := regexp.MustCompile(pattern)
	return re.ReplaceAllStringFunc(yamlStr, func(match string) string {
		// Find the submatch groups
		submatches := re.FindStringSubmatch(match)
		if len(submatches) >= 3 {
			// submatches[0] is the full match
			// submatches[1] is the line start (^ or \n)
			// submatches[2] is the whitespace
			return submatches[1] + submatches[2] + key + ":"
		}
		return match
	})
}
