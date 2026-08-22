package cli

import "github.com/github/gh-aw/pkg/scanfindings"

// ValidationIssue represents a single validation, warning, or audit issue entry.
type ValidationIssue struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Line    int    `json:"line,omitempty"`
	File    string `json:"file,omitempty"`
}

// ToFinding converts the validation issue to the shared finding representation.
// The caller supplies severity because Type is a diagnostic category, not a
// severity level.
func (v ValidationIssue) ToFinding(severity scanfindings.SeverityLevel) scanfindings.Finding {
	return scanfindings.Finding{
		RuleID:   v.Type,
		Severity: severity,
		Message:  v.Message,
		File:     v.File,
		Line:     v.Line,
	}
}
