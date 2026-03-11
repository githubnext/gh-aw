package cli

import "github.com/github/gh-aw/pkg/workflow"

// Package-level version information
var (
	version = "dev"
)

func init() {
	// Set the default version in the workflow package
	// This allows workflow.NewCompiler() to auto-detect the version
	workflow.SetDefaultVersion(version)
}

// SetVersionInfo sets the version information for the CLI and workflow package
func SetVersionInfo(v string) {
	version = v
	workflow.SetDefaultVersion(v) // Keep workflow package in sync
}

// GetVersion returns the current version
func GetVersion() string {
	return version
}
