//go:build js || wasm

// This file provides WASM/JS no-op stubs for Dependabot manifest generation functions.
// The canonical (non-WASM) implementations live in dependabot.go.
// If any function signatures change in dependabot.go, this file must be updated to match.

package workflow

func (c *Compiler) GenerateDependabotManifests(workflowDataList []*WorkflowData, workflowDir string, forceOverwrite bool) error {
	return nil
}
