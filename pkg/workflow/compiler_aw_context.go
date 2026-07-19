package workflow

import (
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var awContextLog = logger.New("workflow:compiler_aw_context")

// AwContextInputName is the name of the internal aw_context workflow_dispatch input.
// It is managed internally by the agentic workflow system and should not be surfaced to users.
const AwContextInputName = "aw_context"

// NetworkAllowedInputName is the optional workflow_call input that extends the compiled
// network allowlist at runtime for reusable workflows.
const NetworkAllowedInputName = "network_allowed"

// awContextInputDescription is the description for the aw_context workflow_dispatch input.
// It signals to users that this input is managed internally by the agentic workflow system.
const awContextInputDescription = "Agent caller context (used internally by Agentic Workflows)."

const networkAllowedInputDescription = "Additional allowed network domains or ecosystem identifiers to union with network.allowed (comma-separated, for example: \"rust\" or \"python,github.com\")."

// injectAwContextIntoOnYAML adds the aw_context input to internal workflow triggers
// in the given on-section YAML string.
//
// The injection is string-based to preserve existing YAML comments and formatting.
// It handles these triggers independently:
//   - workflow_dispatch
//   - workflow_call
//
// For each trigger it supports two cases:
//   - Bare trigger line (no sub-keys): adds an inputs: block with aw_context
//   - Trigger with an existing inputs: sub-key: adds aw_context inside inputs
//
// The function is idempotent: calling it twice produces the same result.
func injectAwContextIntoOnYAML(onSection string) string {
	updated := injectInputIntoTrigger(onSection, "workflow_dispatch", AwContextInputName, buildAwContextInputLines)
	updated = injectInputIntoTrigger(updated, "workflow_call", AwContextInputName, buildAwContextInputLines)
	return updated
}

func injectNetworkAllowedIntoOnYAML(onSection string, network *NetworkPermissions) string {
	if network == nil || !network.AllowedInput {
		return onSection
	}
	return injectInputIntoTrigger(onSection, "workflow_call", NetworkAllowedInputName, buildNetworkAllowedInputLines)
}

func injectInputIntoTrigger(onSection string, triggerName string, inputName string, buildInputLines func(int) []string) string {
	if !strings.Contains(onSection, triggerName) {
		awContextLog.Printf("No %s trigger found, skipping %s injection", triggerName, inputName)
		return onSection
	}
	awContextLog.Printf("Injecting %s input into %s trigger", inputName, triggerName)

	lines := strings.Split(onSection, "\n")
	triggerLineIdx, triggerIndent := findBareTriggerLine(lines, triggerName)
	if triggerLineIdx == -1 {
		awContextLog.Printf("No bare %s: line found, skipping %s injection", triggerName, inputName)
		return onSection
	}
	awContextLog.Printf("Found %s at line %d (indent=%d), injecting %s", triggerName, triggerLineIdx, triggerIndent, inputName)

	inputsLineIdx := findTriggerInputsLine(lines, triggerLineIdx, triggerIndent)
	if inputsLineIdx != -1 && inputAlreadyInjected(lines, inputsLineIdx, inputName) {
		awContextLog.Printf("%s already injected into %s, skipping", inputName, triggerName)
		return onSection
	}

	inputLines := buildInputLines(triggerIndent)
	result := injectInputLinesIntoTrigger(lines, triggerLineIdx, triggerIndent, inputsLineIdx, inputLines)
	return strings.Join(result, "\n")
}

func findBareTriggerLine(lines []string, triggerName string) (int, int) {
	for i, line := range lines {
		stripped := strings.TrimLeft(line, " \t")
		rest, found := strings.CutPrefix(stripped, triggerName+":")
		if found {
			rest = strings.TrimSpace(rest)
			if rest == "" || rest == "null" || rest == "~" {
				return i, len(line) - len(stripped)
			}
		}
	}
	return -1, 0
}

func findTriggerInputsLine(lines []string, triggerLineIdx int, triggerIndent int) int {
	for i := triggerLineIdx + 1; i < len(lines); i++ {
		stripped := strings.TrimLeft(lines[i], " \t")
		if stripped == "" || strings.HasPrefix(stripped, "#") {
			continue
		}
		lineIndent := len(lines[i]) - len(stripped)
		if lineIndent <= triggerIndent {
			break
		}
		if strings.HasPrefix(stripped, "inputs:") {
			return i
		}
		break
	}
	return -1
}

func inputAlreadyInjected(lines []string, inputsLineIdx int, inputName string) bool {
	inputsIndent := len(lines[inputsLineIdx]) - len(strings.TrimLeft(lines[inputsLineIdx], " \t"))
	for i := inputsLineIdx + 1; i < len(lines); i++ {
		stripped := strings.TrimLeft(lines[i], " \t")
		if stripped == "" || strings.HasPrefix(stripped, "#") {
			continue
		}
		lineIndent := len(lines[i]) - len(stripped)
		if lineIndent <= inputsIndent {
			break
		}
		if strings.HasPrefix(stripped, inputName+":") {
			return true
		}
	}
	return false
}

func injectInputLinesIntoTrigger(lines []string, triggerLineIdx int, triggerIndent int, inputsLineIdx int, inputLines []string) []string {
	result := make([]string, 0, safeAllocationCapacity(len(lines), len(inputLines), 1))
	for i, line := range lines {
		// When the trigger line contains an explicit null/~ value,
		// replace it with a bare trigger so sub-keys can follow.
		if i == triggerLineIdx && (strings.HasSuffix(strings.TrimSpace(line), " null") ||
			strings.HasSuffix(strings.TrimSpace(line), " ~")) {
			stripped := strings.TrimLeft(line, " \t")
			line = strings.Repeat(" ", triggerIndent) + strings.SplitN(stripped, ":", 2)[0] + ":"
		}
		result = append(result, line)

		if inputsLineIdx != -1 && i == inputsLineIdx {
			result = append(result, inputLines...)
		} else if inputsLineIdx == -1 && i == triggerLineIdx {
			// Trigger is bare — add inputs: + the requested internal input.
			result = append(result, strings.Repeat(" ", triggerIndent+2)+"inputs:")
			result = append(result, inputLines...)
		}
	}
	return result
}

// buildAwContextInputLines returns the indented YAML lines for the aw_context input
// definition, sized relative to the workflow_dispatch: line's indentation.
func buildAwContextInputLines(wdIndent int) []string {
	awIndent := strings.Repeat(" ", wdIndent+4)   // under inputs:
	propIndent := strings.Repeat(" ", wdIndent+6) // properties of aw_context
	return []string{
		awIndent + AwContextInputName + ":",
		propIndent + "default: \"\"",
		propIndent + "description: " + strconv.Quote(awContextInputDescription),
		propIndent + "required: false",
		propIndent + "type: string",
	}
}

func buildNetworkAllowedInputLines(wdIndent int) []string {
	inputIndent := strings.Repeat(" ", wdIndent+4)
	propIndent := strings.Repeat(" ", wdIndent+6)
	return []string{
		inputIndent + NetworkAllowedInputName + ":",
		propIndent + "default: \"\"",
		propIndent + "description: " + strconv.Quote(networkAllowedInputDescription),
		propIndent + "required: false",
		propIndent + "type: string",
	}
}
