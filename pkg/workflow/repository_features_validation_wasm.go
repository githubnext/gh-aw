//go:build js || wasm

// This file provides WASM/JS no-op stubs for repository features validation functions.
// The canonical (non-WASM) implementations live in repository_features_validation.go.
// If any function signatures change in repository_features_validation.go, this file must be updated to match.

package workflow

type RepositoryFeatures struct {
	HasDiscussions bool
	HasIssues      bool
}

func (c *Compiler) validateRepositoryFeatures(workflowData *WorkflowData) error {
	return nil
}
