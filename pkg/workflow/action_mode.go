package workflow

// ActionMode defines how JavaScript is embedded in workflow steps
type ActionMode string

const (
	// ActionModeInline embeds JavaScript inline using actions/github-script (current behavior)
	ActionModeInline ActionMode = "inline"

	// ActionModeDev references custom actions using local paths (development mode)
	ActionModeDev ActionMode = "dev"
)

// String returns the string representation of the action mode
func (m ActionMode) String() string {
	return string(m)
}

// IsValid checks if the action mode is valid
func (m ActionMode) IsValid() bool {
	return m == ActionModeInline || m == ActionModeDev
}
