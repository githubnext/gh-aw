package cli

// ValidationIssue represents a single validation, warning, or audit issue entry.
type ValidationIssue struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Line    int    `json:"line,omitempty"`
	File    string `json:"file,omitempty"`
}
