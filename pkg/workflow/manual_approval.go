package workflow

import (
	"fmt"
)

// extractManualApprovalFromOn extracts the manual-approval value from the on: section
func (c *Compiler) extractManualApprovalFromOn(frontmatter map[string]any) (string, error) {
	onSection, exists := frontmatter["on"]
	if !exists {
		return "", nil
	}

	// Handle different formats of the on: section
	switch on := onSection.(type) {
	case string:
		// Simple string format like "on: push" - no manual-approval possible
		return "", nil
	case map[string]any:
		// Complex object format - look for manual-approval
		if manualApproval, exists := on["manual-approval"]; exists {
			if str, ok := manualApproval.(string); ok {
				return str, nil
			}
			return "", fmt.Errorf("manual-approval value must be a string")
		}
		return "", nil
	default:
		return "", fmt.Errorf("invalid on: section format")
	}
}

// processManualApprovalConfiguration extracts manual-approval configuration from frontmatter
func (c *Compiler) processManualApprovalConfiguration(frontmatter map[string]any, workflowData *WorkflowData) error {
	// Extract manual-approval from the on: section
	manualApproval, err := c.extractManualApprovalFromOn(frontmatter)
	if err != nil {
		return err
	}
	workflowData.ManualApproval = manualApproval

	return nil
}
