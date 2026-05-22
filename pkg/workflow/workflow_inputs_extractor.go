package workflow

import (
	"os"

	"github.com/github/gh-aw/pkg/parser"
)

func extractInputsFromYAML(workflowPath, trigger string) (map[string]any, error) {
	workflow, err := readWorkflowYAML(workflowPath)
	if err != nil {
		return nil, err
	}

	return extractInputsFromParsedWorkflow(workflow, trigger), nil
}

func extractInputsFromMarkdown(mdPath, trigger string) (map[string]any, error) {
	content, err := os.ReadFile(mdPath) // #nosec G304 -- mdPath is validated via isPathWithinDir in findWorkflowFile
	if err != nil {
		return nil, err
	}

	result, err := parser.ExtractFrontmatterFromContent(string(content))
	if err != nil || result == nil {
		return make(map[string]any), nil
	}

	return extractInputsFromParsedWorkflow(result.Frontmatter, trigger), nil
}

func extractInputsFromParsedWorkflow(workflow map[string]any, trigger string) map[string]any {
	onSection, hasOn := workflow["on"]
	if !hasOn {
		return make(map[string]any)
	}

	onMap, ok := onSection.(map[string]any)
	if !ok {
		return make(map[string]any)
	}

	triggerConfig, hasTriggerConfig := onMap[trigger]
	if !hasTriggerConfig {
		return make(map[string]any)
	}

	triggerMap, ok := triggerConfig.(map[string]any)
	if !ok {
		return make(map[string]any)
	}

	inputs, hasInputs := triggerMap["inputs"]
	if !hasInputs {
		return make(map[string]any)
	}

	inputsMap, ok := inputs.(map[string]any)
	if !ok {
		return make(map[string]any)
	}

	return inputsMap
}
