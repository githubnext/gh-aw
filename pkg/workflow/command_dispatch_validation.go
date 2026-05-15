package workflow

import "fmt"

// validateCommandWorkflowDispatchInputs rejects required workflow_dispatch inputs when
// slash_command or label_command triggers are configured.
func validateCommandWorkflowDispatchInputs(workflowData *WorkflowData) error {
	if workflowData == nil || workflowData.RawFrontmatter == nil {
		return nil
	}

	hasCommandTrigger := len(workflowData.Command) > 0 || len(workflowData.LabelCommand) > 0
	if !hasCommandTrigger {
		return nil
	}

	onMap, ok := workflowData.RawFrontmatter["on"].(map[string]any)
	if !ok {
		return nil
	}

	workflowDispatchMap, ok := onMap["workflow_dispatch"].(map[string]any)
	if !ok {
		return nil
	}

	inputsMap, ok := workflowDispatchMap["inputs"].(map[string]any)
	if !ok {
		return nil
	}

	for inputName, inputDef := range inputsMap {
		inputDefMap, ok := inputDef.(map[string]any)
		if !ok {
			continue
		}

		required, ok := inputDefMap["required"].(bool)
		if ok && required {
			return fmt.Errorf(
				"on.workflow_dispatch.inputs.%s.required: true is not allowed with slash_command or label_command; set required: false",
				inputName,
			)
		}
	}

	return nil
}
