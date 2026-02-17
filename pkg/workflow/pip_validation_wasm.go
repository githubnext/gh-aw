//go:build js || wasm

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
