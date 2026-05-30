//go:build js || wasm

package workflow

func (c *Compiler) GenerateDependabotManifests(workflowDataList []*WorkflowData, workflowDir string, forceOverwrite bool) error {
	return nil
}
