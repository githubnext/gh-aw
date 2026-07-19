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
	state := rewriteWorkflowDispatchRequiredFalseState{}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		indent := getIndentation(line)

		// Blank lines and comments do not affect nesting state.
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			result = append(result, line)
			continue
		}

		rewriteWorkflowDispatchRequiredFalseExitBlocks(indent, &state)

		// Enter on: block.
		if isTopLevelKey(line) && strings.HasPrefix(trimmed, "on:") {
			state.inOn = true
			state.onIndent = indent
			result = append(result, line)
			continue
		}

		if rewriteWorkflowDispatchRequiredFalseEnterWorkflowDispatch(trimmed, indent, &state) {
			result = append(result, line)
			continue
		}

		if rewriteWorkflowDispatchRequiredFalseEnterInputs(trimmed, indent, &state) {
			result = append(result, line)
			continue
		}

		if newLine, replaced := rewriteWorkflowDispatchRequiredFalseInlineInputs(line, trimmed, &state); replaced {
			result = append(result, newLine)
			modified = true
			continue
		}

		newLine, handled, changed := rewriteWorkflowDispatchRequiredFalseInputEntry(line, trimmed, indent, &state)
		if handled {
			result = append(result, newLine)
			modified = modified || changed
			continue
		}

		if newLine, replaced := rewriteWorkflowDispatchRequiredFalseInputRequired(line, trimmed, indent, &state); replaced {
			result = append(result, newLine)
			modified = true
			continue
		}

		result = append(result, line)
	}

	return result, modified
}

type rewriteWorkflowDispatchRequiredFalseState struct {
	inOn             bool
	onIndent         string
	inWD             bool
	wdIndent         string
	inInputs         bool
	inputsIndent     string
	inInputEntry     bool
	inputEntryIndent string
}

func rewriteWorkflowDispatchRequiredFalseExitBlocks(indent string, state *rewriteWorkflowDispatchRequiredFalseState) {
	// Exit deeper states first (order matters: deepest → shallowest).
	if state.inInputEntry && len(indent) <= len(state.inputEntryIndent) {
		state.inInputEntry = false
	}
	if state.inInputs && len(indent) <= len(state.inputsIndent) {
		state.inInputs = false
		state.inInputEntry = false
	}
	if state.inWD && len(indent) <= len(state.wdIndent) {
		state.inWD = false
		state.inInputs = false
		state.inInputEntry = false
	}
	if state.inOn && len(indent) <= len(state.onIndent) {
		state.inOn = false
		state.inWD = false
		state.inInputs = false
		state.inInputEntry = false
	}
}

func rewriteWorkflowDispatchRequiredFalseEnterWorkflowDispatch(trimmed, indent string, state *rewriteWorkflowDispatchRequiredFalseState) bool {
	// Enter workflow_dispatch: within on:.
	if !state.inOn || state.inWD || !strings.HasPrefix(trimmed, "workflow_dispatch:") {
		return false
	}
	state.inWD = true
	state.wdIndent = indent
	return true
}

func rewriteWorkflowDispatchRequiredFalseEnterInputs(trimmed, indent string, state *rewriteWorkflowDispatchRequiredFalseState) bool {
	// Enter inputs: within workflow_dispatch: (allow trailing comments).
	if !state.inWD || state.inInputs || !strings.HasPrefix(trimmed, "inputs:") {
		return false
	}
	remainder := strings.TrimSpace(strings.TrimPrefix(trimmed, "inputs:"))
	if remainder != "" && !strings.HasPrefix(remainder, "#") {
		return false
	}
	state.inInputs = true
	state.inputsIndent = indent
	return true
}

func rewriteWorkflowDispatchRequiredFalseInlineInputs(line, trimmed string, state *rewriteWorkflowDispatchRequiredFalseState) (string, bool) {
	// Handle inline inputs maps (for example: "inputs: { pr_number: { required: true } }").
	if !state.inWD || !strings.HasPrefix(trimmed, "inputs:") || !strings.Contains(trimmed, "required: true") {
		return "", false
	}
	newLine := strings.ReplaceAll(line, "required: true", "required: false")
	if newLine == line {
		return "", false
	}
	workflowDispatchRequiredLog.Print("Rewrote inline workflow_dispatch input required: true to required: false")
	return newLine, true
}

func rewriteWorkflowDispatchRequiredFalseInputEntry(line, trimmed, indent string, state *rewriteWorkflowDispatchRequiredFalseState) (string, bool, bool) {
	// Enter an individual input entry and handle inline input maps
	// (for example: "pr_number: { required: true }").
	if !state.inInputs || state.inInputEntry || len(indent) <= len(state.inputsIndent) {
		return "", false, false
	}
	if strings.Contains(trimmed, "{") && strings.Contains(trimmed, "required: true") {
		newLine := strings.ReplaceAll(line, "required: true", "required: false")
		if newLine != line {
			workflowDispatchRequiredLog.Print("Rewrote inline input entry required: true to required: false")
			return newLine, true, true
		}
	}
	state.inInputEntry = true
	state.inputEntryIndent = indent
	return line, true, false
}

func rewriteWorkflowDispatchRequiredFalseInputRequired(line, trimmed, indent string, state *rewriteWorkflowDispatchRequiredFalseState) (string, bool) {
	// Within an input entry's properties: rewrite "required: true" → "required: false".
	if !state.inInputEntry || len(indent) <= len(state.inputEntryIndent) || !strings.HasPrefix(trimmed, "required: true") {
		return "", false
	}
	newLine := strings.Replace(line, "required: true", "required: false", 1)
	if newLine == line {
		return "", false
	}
	workflowDispatchRequiredLog.Print("Rewrote workflow_dispatch input required: true to required: false")
	return newLine, true
}
