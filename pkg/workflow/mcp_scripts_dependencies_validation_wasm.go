//go:build js || wasm

package workflow

func (c *Compiler) validateMCPScriptDependencies(workflowData *WorkflowData) error {
	return nil
}
