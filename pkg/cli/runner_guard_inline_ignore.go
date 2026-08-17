package cli

import "strings"

// filterRunnerGuardIgnoredFindings drops findings that have a local inline suppression comment
// in the compiled workflow. runner-guard sometimes reports shell findings on a nearby generated
// setup line rather than the command line, so a forward-looking window is used to catch comments
// that sit immediately before the relevant command in the next run block.
func filterRunnerGuardIgnoredFindings(findings []runnerGuardFinding, gitRoot string) []runnerGuardFinding {
	filtered := make([]runnerGuardFinding, 0, len(findings))
	fileLinesByPath := make(map[string][]string)

	for _, finding := range findings {
		resolvedPath := resolveRunnerGuardFilePath(gitRoot, finding.File)
		lines, ok := fileLinesByPath[resolvedPath]
		if !ok {
			lines = readWorkflowLines(resolvedPath)
			fileLinesByPath[resolvedPath] = lines
		}

		if hasRunnerGuardInlineIgnore(lines, finding.Line, finding.RuleID) {
			runnerGuardLog.Printf("Suppressing %s finding with inline runner-guard ignore in %s", finding.RuleID, finding.File)
			continue
		}
		filtered = append(filtered, finding)
	}

	return filtered
}

func hasRunnerGuardInlineIgnore(lines []string, lineNum int, ruleID string) bool {
	if len(lines) == 0 || ruleID == "" {
		return false
	}
	if lineNum > len(lines) {
		return false
	}

	start := max(lineNum-3, 0)
	end := min(lineNum+20, len(lines))

	marker := "runner-guard:ignore " + ruleID
	for _, line := range lines[start:end] {
		if strings.Contains(line, marker) {
			return true
		}
	}
	return false
}
