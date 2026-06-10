package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var workflowDispatchRequiredLog = logger.New("cli:codemod_workflow_dispatch_required")

// getWorkflowDispatchRequiredFalseCodemod creates a codemod that rewrites
// on.workflow_dispatch.inputs.*.required: true → required: false when on.slash_command
// or on.label_command is also present in the same workflow.
//
// Auto-dispatched triggers (slash_command, label_command) cannot supply manual inputs
// to a workflow_dispatch trigger, so required: true inputs can never be satisfied.
// The safe fix is to set required: false and handle missing values with a default.
func getWorkflowDispatchRequiredFalseCodemod() Codemod {
	return Codemod{
		ID:           "workflow-dispatch-required-false-with-slash-command",
		Name:         "Set workflow_dispatch inputs required: false for command triggers",
		Description:  "When on.slash_command or on.label_command is present, rewrites workflow_dispatch.inputs.*.required: true to required: false because auto-dispatched triggers cannot enforce required manual inputs.",
		IntroducedIn: "1.5.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			onValue, hasOn := frontmatter["on"]
			if !hasOn {
				return content, false, nil
			}
			onMap, ok := onValue.(map[string]any)
			if !ok {
				return content, false, nil
			}

			_, hasSlashCommand := onMap["slash_command"]
			_, hasLabelCommand := onMap["label_command"]
			if !hasSlashCommand && !hasLabelCommand {
				return content, false, nil
			}

			wdMap, ok := onMap["workflow_dispatch"].(map[string]any)
			if !ok {
				return content, false, nil
			}
			inputsMap, ok := wdMap["inputs"].(map[string]any)
			if !ok {
				return content, false, nil
			}

			// Only apply when at least one input has required: true
			hasRequiredTrue := false
			for _, inputDef := range inputsMap {
				inputDefMap, ok := inputDef.(map[string]any)
				if !ok {
					continue
				}
				if req, ok := inputDefMap["required"].(bool); ok && req {
					hasRequiredTrue = true
					break
				}
			}
			if !hasRequiredTrue {
				return content, false, nil
			}

			newContent, applied, err := applyFrontmatterLineTransform(content, rewriteWorkflowDispatchRequiredFalse)
			if applied {
				workflowDispatchRequiredLog.Print("Applied workflow-dispatch-required-false-with-slash-command codemod")
			}
			return newContent, applied, err
		},
	}
}

// rewriteWorkflowDispatchRequiredFalse is the line-level transform that walks the YAML
// frontmatter and replaces every "required: true" inside
// on.workflow_dispatch.inputs.<name> with "required: false".
func rewriteWorkflowDispatchRequiredFalse(lines []string) ([]string, bool) {
	result := make([]string, 0, len(lines))
	modified := false

	inOn := false
	onIndent := ""
	inWD := false
	wdIndent := ""
	inInputs := false
	inputsIndent := ""
	inInputEntry := false
	inputEntryIndent := ""

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		indent := getIndentation(line)

		// Blank lines and comments do not affect nesting state.
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			result = append(result, line)
			continue
		}

		// Exit deeper states first (order matters: deepest → shallowest).
		if inInputEntry && len(indent) <= len(inputEntryIndent) {
			inInputEntry = false
		}
		if inInputs && len(indent) <= len(inputsIndent) {
			inInputs = false
			inInputEntry = false
		}
		if inWD && len(indent) <= len(wdIndent) {
			inWD = false
			inInputs = false
			inInputEntry = false
		}
		if inOn && len(indent) <= len(onIndent) {
			inOn = false
			inWD = false
			inInputs = false
			inInputEntry = false
		}

		// Enter on: block.
		if isTopLevelKey(line) && strings.HasPrefix(trimmed, "on:") {
			inOn = true
			onIndent = indent
			result = append(result, line)
			continue
		}

		// Enter workflow_dispatch: within on:.
		if inOn && !inWD && strings.HasPrefix(trimmed, "workflow_dispatch:") {
			inWD = true
			wdIndent = indent
			result = append(result, line)
			continue
		}

		// Enter inputs: within workflow_dispatch: (allow trailing comments).
		if inWD && !inInputs && strings.HasPrefix(trimmed, "inputs:") {
			remainder := strings.TrimSpace(strings.TrimPrefix(trimmed, "inputs:"))
			if remainder == "" || strings.HasPrefix(remainder, "#") {
				inInputs = true
				inputsIndent = indent
				result = append(result, line)
				continue
			}
		}

		// Handle inline inputs maps (for example: "inputs: { pr_number: { required: true } }").
		if inWD && strings.HasPrefix(trimmed, "inputs:") && strings.Contains(trimmed, "required: true") {
			newLine := strings.ReplaceAll(line, "required: true", "required: false")
			if newLine != line {
				result = append(result, newLine)
				modified = true
				workflowDispatchRequiredLog.Print("Rewrote inline workflow_dispatch input required: true to required: false")
				continue
			}
		}

		// Enter an individual input entry and handle inline input maps
		// (for example: "pr_number: { required: true }").
		if inInputs && !inInputEntry && len(indent) > len(inputsIndent) {
			if strings.Contains(trimmed, "{") && strings.Contains(trimmed, "required: true") {
				newLine := strings.ReplaceAll(line, "required: true", "required: false")
				if newLine != line {
					result = append(result, newLine)
					modified = true
					workflowDispatchRequiredLog.Print("Rewrote inline input entry required: true to required: false")
					continue
				}
			}
			inInputEntry = true
			inputEntryIndent = indent
			result = append(result, line)
			continue
		}

		// Within an input entry's properties: rewrite "required: true" → "required: false".
		if inInputEntry && len(indent) > len(inputEntryIndent) {
			if strings.HasPrefix(trimmed, "required: true") {
				newLine := strings.Replace(line, "required: true", "required: false", 1)
				if newLine != line {
					result = append(result, newLine)
					modified = true
					workflowDispatchRequiredLog.Print("Rewrote workflow_dispatch input required: true to required: false")
					continue
				}
			}
		}

		result = append(result, line)
	}

	return result, modified
}
