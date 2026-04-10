//go:build js || wasm

// This file provides WASM/JS no-op stubs for pip/uv validation functions.
// The canonical (non-WASM) implementations live in pip_validation.go.
// If any function signatures change in pip_validation.go, this file must be updated to match.

package workflow

func (c *Compiler) validatePythonPackagesWithPip(packages []string, packageType string, pipCmd string) {
}

func (c *Compiler) validatePipPackages(workflowData *WorkflowData) error {
	return nil
}

func (c *Compiler) validateUvPackages(workflowData *WorkflowData) error {
	return nil
}

func (c *Compiler) validateUvPackagesWithPip(packages []string, pipCmd string) error {
	return nil
}
