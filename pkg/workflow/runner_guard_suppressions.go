package workflow

import "strings"

const runnerGuardIgnorePrefix = "# runner-guard:ignore"

// preserveRunnerGuardStepSuppressions copies scoped runner-guard comments from
// workflow frontmatter to the corresponding generated workflow steps.
func preserveRunnerGuardStepSuppressions(workflowYAML, frontmatterYAML string) string {
	suppressions := make(map[string]string)
	frontmatterLines := strings.Split(frontmatterYAML, "\n")
	for i := 0; i+1 < len(frontmatterLines); i++ {
		directive := strings.TrimSpace(frontmatterLines[i])
		if !strings.HasPrefix(directive, runnerGuardIgnorePrefix) {
			continue
		}
		if stepName := workflowStepName(frontmatterLines[i+1]); stepName != "" {
			suppressions[stepName] = directive
		}
	}
	if len(suppressions) == 0 {
		return workflowYAML
	}

	frontmatterNames := countWorkflowStepNames(frontmatterYAML)
	generatedNames := countWorkflowStepNames(workflowYAML)
	var output strings.Builder
	for line := range strings.SplitSeq(workflowYAML, "\n") {
		stepName := workflowStepName(line)
		if directive := suppressions[stepName]; directive != "" && frontmatterNames[stepName] == 1 && generatedNames[stepName] == 1 {
			indent := line[:len(line)-len(strings.TrimLeft(line, " "))]
			output.WriteString(indent)
			output.WriteString(directive)
			output.WriteByte('\n')
		}
		output.WriteString(line)
		output.WriteByte('\n')
	}
	return strings.TrimSuffix(output.String(), "\n")
}

func workflowStepName(line string) string {
	const prefix = "- name:"
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, prefix) {
		return ""
	}
	return strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, prefix)), `"'`)
}

func countWorkflowStepNames(content string) map[string]int {
	counts := make(map[string]int)
	for line := range strings.SplitSeq(content, "\n") {
		if stepName := workflowStepName(line); stepName != "" {
			counts[stepName]++
		}
	}
	return counts
}
