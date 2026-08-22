package cli

import "github.com/github/gh-aw/pkg/scanfindings"

// ValidationIssue represents a single validation, warning, or audit issue entry.
type ValidationIssue struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Line    int    `json:"line,omitempty"`
	File    string `json:"file,omitempty"`
}

// Severity maps the issue type ("error", "warning", ...) onto the shared
// severity vocabulary used by the scanner integrations.
func (v ValidationIssue) Severity() scanfindings.SeverityLevel {
	return scanfindings.ParseSeverity(v.Type)
}

// ToFinding converts the validation issue to the shared finding representation.
func (v ValidationIssue) ToFinding() scanfindings.Finding {
	return scanfindings.Finding{
		Severity: v.Severity(),
		Message:  v.Message,
		File:     v.File,
		Line:     v.Line,
	}
}
