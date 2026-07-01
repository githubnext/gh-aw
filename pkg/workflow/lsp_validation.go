package workflow

import (
	"fmt"

	"github.com/github/gh-aw/pkg/logger"
)

var lspValidationLog = logger.New("workflow:lsp_validation")

func (c *Compiler) validateLSPSupport(workflowData *WorkflowData) error {
	if workflowData == nil || len(workflowData.LSP) == 0 {
		return nil
	}

	lspValidationLog.Printf("validateLSPSupport: validating %d LSP config(s)", len(workflowData.LSP))

	manager := NewLSPManager(workflowData.LSP)
	if err := manager.Validate(); err != nil {
		return err
	}

	engineID := ResolveEngineID(workflowData)
	if engineID == "" {
		engineID = "copilot"
	}

	if engineID != "copilot" {
		lspValidationLog.Printf("validateLSPSupport: rejecting unsupported engine %q for LSP", engineID)
		return fmt.Errorf("lsp is currently only supported for engine: copilot (found engine: %s)", engineID)
	}

	lspValidationLog.Printf("validateLSPSupport: LSP validation passed for engine %q", engineID)
	return nil
}
