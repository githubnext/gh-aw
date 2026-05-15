package workflow

import (
	"fmt"
	"strings"
)

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
			var triggerNames []string
			if hasSlashCommand {
				triggerNames = append(triggerNames, "slash_command")
			}
			if hasLabelCommand {
				triggerNames = append(triggerNames, "label_command")
			}
			triggersPhrase := strings.Join(triggerNames, " and ")

			return fmt.Errorf(
				"on.workflow_dispatch.inputs.%s.required: true is not allowed when using %s; set required: false",
				inputName, triggersPhrase,
			)
		}
	}

	return nil
}
