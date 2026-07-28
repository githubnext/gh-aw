package modelsdev

import "strings"

// modelIDReplacer normalizes separator characters in model IDs so that IDs
// differing only in ".", "_", or "-" compare equal.
var modelIDReplacer = strings.NewReplacer(".", "-", "_", "-")

// NormalizeProvider maps provider aliases (e.g. "github", "copilot", "github_models")
// to their canonical form ("github-copilot") and lower-cases all other values.
func NormalizeProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "github", "copilot", "github_models":
		return "github-copilot"
	default:
		return strings.ToLower(strings.TrimSpace(provider))
	}
}

// NormalizeComparableModelID lower-cases the value and replaces "." and "_" with "-"
// so that model IDs differing only in those separators compare equal.
func NormalizeComparableModelID(value string) string {
	return modelIDReplacer.Replace(strings.ToLower(strings.TrimSpace(value)))
}
