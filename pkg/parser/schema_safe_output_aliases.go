package parser

import (
	"fmt"
	"sort"
	"strings"
)

// safeOutputsSchemaPath is the JSON schema path for the safe-outputs section.
const safeOutputsSchemaPath = "/safe-outputs"

// safeOutputAliases maps common agent mistakes to their correct safe-output field names.
// Only includes true concept remappings where the alias remains far from the canonical
// field even after separator normalization (underscore→hyphen). Simple underscore variants
// (e.g. "add_labels" → "add-labels") are handled automatically by the Levenshtein
// separator-normalization in FindClosestMatches and do not need explicit entries here.
var safeOutputAliases = map[string]string{
	// add-comment: MCP tool names and common misphrases that Levenshtein cannot bridge
	// (e.g. "create-issue-comment" vs "add-comment" is distance ~14 after normalization)
	"create-issue-comment": "add-comment",
	"create_issue_comment": "add-comment",
	"add-issue-comment":    "add-comment",
	"add_issue_comment":    "add-comment",
	"post-comment":         "add-comment",
	"post_comment":         "add-comment",
	"create-comment":       "add-comment",
	"create_comment":       "add-comment",
}

// safeOutputAliasSuggestion returns a "Did you mean 'X'?" suggestion when an unknown
// property under /safe-outputs matches a known alias for the correct field name.
// It returns an empty string when the error is not under safe-outputs, is not an
// additional-properties error, or when none of the invalid props match a known alias.
func safeOutputAliasSuggestion(errorMessage, jsonPath string) string {
	if jsonPath != safeOutputsSchemaPath {
		return ""
	}

	lowerError := strings.ToLower(errorMessage)
	if !strings.Contains(lowerError, "additional propert") || !strings.Contains(lowerError, "not allowed") {
		return ""
	}

	invalidProps := extractAdditionalPropertyNames(errorMessage)
	if len(invalidProps) == 0 {
		return ""
	}

	var suggestions []string
	seen := make(map[string]struct{})
	for _, prop := range invalidProps {
		canonical, ok := safeOutputAliases[prop]
		if !ok {
			continue
		}
		if _, already := seen[canonical]; already {
			continue
		}
		seen[canonical] = struct{}{}
		suggestions = append(suggestions, fmt.Sprintf("'%s'", canonical))
	}

	if len(suggestions) == 0 {
		return ""
	}

	sort.Strings(suggestions)

	if len(suggestions) == 1 {
		return fmt.Sprintf("Did you mean %s?", suggestions[0])
	}
	return fmt.Sprintf("Did you mean: %s?", strings.Join(suggestions, ", "))
}
