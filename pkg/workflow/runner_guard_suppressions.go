package workflow

import "strings"

const runnerGuardIgnorePrefix = "# runner-guard:ignore"

// workflowStepRef locates a structural step entry inside a YAML document.
type workflowStepRef struct {
	line int
	key  string
}

// preserveRunnerGuardStepSuppressions copies scoped runner-guard comments from
// workflow frontmatter to the corresponding generated workflow steps.
//
// Directives are matched to the step they annotate, either as a standalone
// comment on the line above the step or as an inline trailing comment on the
// step's first line. Steps are identified by name when they have one and by
// their first line's content otherwise, so unnamed `- uses:`/`- run:` steps are
// supported too. Lines inside block scalar payloads (`run: |`, `run: >`) are
// never treated as structural YAML, so runner-guard comments and `- name:`
// lines embedded in scripts are ignored.
func preserveRunnerGuardStepSuppressions(workflowYAML, frontmatterYAML string) string {
	frontmatterLines := strings.Split(frontmatterYAML, "\n")
	frontmatterSteps, frontmatterStructural := indexWorkflowSteps(frontmatterLines)
	suppressions := collectRunnerGuardSuppressions(frontmatterLines, frontmatterStructural, frontmatterSteps)
	if len(suppressions) == 0 {
		return workflowYAML
	}

	lines := strings.Split(workflowYAML, "\n")
	generatedSteps, _ := indexWorkflowSteps(lines)
	frontmatterCounts := countWorkflowStepKeys(frontmatterSteps)
	generatedCounts := countWorkflowStepKeys(generatedSteps)

	directivesByLine := make(map[int]string, len(suppressions))
	for _, step := range generatedSteps {
		directive, ok := suppressions[step.key]
		if !ok || frontmatterCounts[step.key] != 1 || generatedCounts[step.key] != 1 {
			continue
		}
		directivesByLine[step.line] = directive
	}
	if len(directivesByLine) == 0 {
		return workflowYAML
	}

	output := make([]string, 0, len(lines)+len(directivesByLine))
	for i, line := range lines {
		if directive, ok := directivesByLine[i]; ok {
			indent := line[:len(line)-len(strings.TrimLeft(line, " "))]
			output = append(output, indent+directive)
		}
		output = append(output, line)
	}
	return strings.Join(output, "\n")
}

// collectRunnerGuardSuppressions maps each annotated step key to its directive.
// Keys annotated more than once are ambiguous and are dropped entirely so that a
// directive is never silently attributed to the wrong step.
func collectRunnerGuardSuppressions(lines []string, structural []bool, steps []workflowStepRef) map[string]string {
	stepKeys := make(map[int]string, len(steps))
	for _, step := range steps {
		stepKeys[step.line] = step.key
	}

	suppressions := make(map[string]string)
	ambiguous := make(map[string]bool)
	pending := ""
	for i, line := range lines {
		if !structural[i] {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, runnerGuardIgnorePrefix) {
			pending = trimmed
			continue
		}
		key, isStep := stepKeys[i]
		if !isStep {
			pending = ""
			continue
		}
		directive := pending
		pending = ""
		if inline := inlineRunnerGuardDirective(line); inline != "" {
			directive = inline
		}
		if directive == "" {
			continue
		}
		if _, seen := suppressions[key]; seen {
			ambiguous[key] = true
			continue
		}
		suppressions[key] = directive
	}
	for key := range ambiguous {
		delete(suppressions, key)
	}
	return suppressions
}

// indexWorkflowSteps returns the structural step entries of a YAML document
// along with a mask marking which lines are structural (i.e. not part of a
// block scalar payload).
func indexWorkflowSteps(lines []string) ([]workflowStepRef, []bool) {
	structural := make([]bool, len(lines))
	var steps []workflowStepRef
	var state yamlBlockScalarState
	for i, line := range lines {
		if state.update(line) {
			continue
		}
		structural[i] = true
		if key, ok := workflowStepKey(line); ok {
			steps = append(steps, workflowStepRef{line: i, key: key})
		}
	}
	return steps, structural
}

func countWorkflowStepKeys(steps []workflowStepRef) map[string]int {
	counts := make(map[string]int, len(steps))
	for _, step := range steps {
		counts[step.key]++
	}
	return counts
}

// workflowStepKey derives a matching key for a sequence entry. Named steps are
// keyed by their name; other entries are keyed by their first line's content
// with any trailing runner-guard directive removed.
func workflowStepKey(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "- ") {
		return "", false
	}
	if idx := strings.Index(trimmed, runnerGuardIgnorePrefix); idx > 0 {
		trimmed = strings.TrimSpace(trimmed[:idx])
	}
	if name := workflowStepName(trimmed); name != "" {
		return "name:" + name, true
	}
	return "step:" + trimmed, true
}

// inlineRunnerGuardDirective returns a runner-guard directive written as a
// trailing comment on line, or an empty string when there is none.
func inlineRunnerGuardDirective(line string) string {
	idx := strings.Index(line, runnerGuardIgnorePrefix)
	if idx <= 0 || strings.TrimSpace(line[:idx]) == "" {
		return ""
	}
	return strings.TrimSpace(line[idx:])
}

func workflowStepName(line string) string {
	const prefix = "- name:"
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, prefix) {
		return ""
	}
	return strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, prefix)), `"'`)
}
