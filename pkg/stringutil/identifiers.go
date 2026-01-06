package stringutil

import "strings"

// NormalizeWorkflowName removes .md and .lock.yml extensions from workflow names.
// This is used to standardize workflow identifiers regardless of the file format.
//
// The function checks for extensions in order of specificity:
// 1. Removes .lock.yml extension (the compiled workflow format)
// 2. Removes .md extension (the markdown source format)
// 3. Returns the name unchanged if no recognized extension is found
//
// This function performs normalization only - it assumes the input is already
// a valid identifier and does NOT perform character validation or sanitization.
//
// Examples:
//
//	NormalizeWorkflowName("weekly-research")           // returns "weekly-research"
//	NormalizeWorkflowName("weekly-research.md")        // returns "weekly-research"
//	NormalizeWorkflowName("weekly-research.lock.yml")  // returns "weekly-research"
//	NormalizeWorkflowName("my.workflow.md")            // returns "my.workflow"
func NormalizeWorkflowName(name string) string {
	// Remove .lock.yml extension first (longer extension)
	if strings.HasSuffix(name, ".lock.yml") {
		return strings.TrimSuffix(name, ".lock.yml")
	}

	// Remove .md extension
	if strings.HasSuffix(name, ".md") {
		return strings.TrimSuffix(name, ".md")
	}

	return name
}

// NormalizeSafeOutputIdentifier converts dashes to underscores for safe output identifiers.
// This standardizes identifier format from the user-facing dash-separated format
// to the internal underscore-separated format used in safe outputs configuration.
//
// Both dash-separated and underscore-separated formats are valid inputs.
// This function simply standardizes to the internal representation.
//
// This function performs normalization only - it assumes the input is already
// a valid identifier and does NOT perform character validation or sanitization.
//
// Examples:
//
//	NormalizeSafeOutputIdentifier("create-issue")      // returns "create_issue"
//	NormalizeSafeOutputIdentifier("create_issue")      // returns "create_issue" (unchanged)
//	NormalizeSafeOutputIdentifier("add-comment")       // returns "add_comment"
//	NormalizeSafeOutputIdentifier("update-pr")         // returns "update_pr"
func NormalizeSafeOutputIdentifier(identifier string) string {
	return strings.ReplaceAll(identifier, "-", "_")
}
