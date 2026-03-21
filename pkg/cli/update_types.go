package cli

// workflowWithSource represents a workflow with its source information
type workflowWithSource struct {
	Name       string
	Path       string
	SourceSpec string // e.g., "owner/repo/path@ref"
}

// updateFailure represents a failed workflow update
type updateFailure struct {
	Name  string
	Error string
}
