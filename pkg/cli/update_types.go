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

// actionsLockEntry represents a single action pin entry
type actionsLockEntry struct {
	Repo    string `json:"repo"`
	Version string `json:"version"`
	SHA     string `json:"sha"`
}

// actionsLockFile represents the structure of actions-lock.json
type actionsLockFile struct {
	Entries map[string]actionsLockEntry `json:"entries"`
}
