package cli

import "github.com/github/gh-aw/pkg/workflow"

// SetVersionInfo sets version information for the workflow package.
func SetVersionInfo(v string) {
	workflow.SetVersion(v)
}

// GetVersion returns the current version.
func GetVersion() string {
	return workflow.GetVersion()
}
