package workflow

import "fmt"

// validateCommandWorkflowDispatchInputs rejects required workflow_dispatch inputs when
// slash_command or label_command triggers are configured.
func validateCommandWorkflowDispatchInputs(workflowData *WorkflowData) error {
	if workflowData == nil || workflowData.RawFrontmatter == nil {
		return nil
	}

	hasSlashCommand := len(workflowData.Command) > 0
	hasLabelCommand := len(workflowData.LabelCommand) > 0
	hasSlashOrLabelCommand := hasSlashCommand || hasLabelCommand
	if !hasSlashOrLabelCommand {
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
			triggerName := "slash_command or label_command"
			if hasSlashCommand && hasLabelCommand {
				triggerName = "slash_command and label_command"
			} else if hasSlashCommand {
				triggerName = "slash_command"
			} else if hasLabelCommand {
				triggerName = "label_command"
			}

			return fmt.Errorf(
				"on.workflow_dispatch.inputs.%s.required: true is not allowed when using %s; set required: false",
				inputName, triggerName,
			)
		}
	}

	return nil
}
