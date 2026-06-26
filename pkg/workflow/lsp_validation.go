package workflow

import "fmt"

func (c *Compiler) validateLSPSupport(workflowData *WorkflowData) error {
	if workflowData == nil || len(workflowData.LSP) == 0 {
		return nil
	}

	manager := NewLSPManager(workflowData.LSP)
	if err := manager.Validate(); err != nil {
		return err
	}

	engineID := ResolveEngineID(workflowData)
	if engineID == "" {
		engineID = "copilot"
	}

	if engineID != "copilot" {
		return fmt.Errorf("lsp is currently only supported for engine: copilot (found engine: %s)", engineID)
	}

	return nil
}
