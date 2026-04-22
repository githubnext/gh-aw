package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var workflowRunBranchesCodemodLog = logger.New("cli:codemod_workflow_run_branches")

// getWorkflowRunBranchesCodemod adds default branch restrictions for bare workflow_run triggers.
func getWorkflowRunBranchesCodemod() Codemod {
	return Codemod{
		ID:           "workflow-run-branches-default",
		Name:         "Add workflow_run branch restrictions",
		Description:  "Adds default branches [main, master] to on.workflow_run when branches are missing",
		IntroducedIn: "1.0.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			onAny, hasOn := frontmatter["on"]
			if !hasOn {
				return content, false, nil
			}

			onMap, ok := onAny.(map[string]any)
			if !ok {
				return content, false, nil
			}

			workflowRunAny, hasWorkflowRun := onMap["workflow_run"]
			if !hasWorkflowRun {
				return content, false, nil
			}

			workflowRunMap, ok := workflowRunAny.(map[string]any)
			if !ok {
				return content, false, nil
			}

			if _, hasBranches := workflowRunMap["branches"]; hasBranches {
				return content, false, nil
			}

			newContent, applied, err := applyFrontmatterLineTransform(content, addDefaultWorkflowRunBranches)
			if applied {
				workflowRunBranchesCodemodLog.Print("Added default branch restrictions to on.workflow_run")
			}
			return newContent, applied, err
		},
	}
}

func addDefaultWorkflowRunBranches(lines []string) ([]string, bool) {
	onIdx := -1
	onIndent := ""
	onEnd := len(lines)
	for i, line := range lines {
		if isTopLevelKey(line) && strings.HasPrefix(strings.TrimSpace(line), "on:") {
			onIdx = i
			onIndent = getIndentation(line)
			for j := i + 1; j < len(lines); j++ {
				if isTopLevelKey(lines[j]) {
					onEnd = j
					break
				}
			}
			break
		}
	}
	if onIdx == -1 {
		return lines, false
	}

	workflowRunIdx := -1
	workflowRunIndent := ""
	workflowRunEnd := onEnd
	for i := onIdx + 1; i < onEnd; i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		if len(getIndentation(lines[i])) <= len(onIndent) {
			break
		}

		if strings.HasPrefix(trimmed, "workflow_run:") {
			if strings.Contains(trimmed, "{") {
				return lines, false
			}
			workflowRunIdx = i
			workflowRunIndent = getIndentation(lines[i])
			for j := i + 1; j < onEnd; j++ {
				innerTrimmed := strings.TrimSpace(lines[j])
				if innerTrimmed == "" || strings.HasPrefix(innerTrimmed, "#") {
					continue
				}
				if len(getIndentation(lines[j])) <= len(workflowRunIndent) {
					workflowRunEnd = j
					break
				}
			}
			break
		}
	}

	if workflowRunIdx == -1 {
		return lines, false
	}

	for i := workflowRunIdx + 1; i < workflowRunEnd; i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "branches:") {
			return lines, false
		}
	}

	branchIndent := workflowRunIndent + "  "
	entries := []string{
		branchIndent + "branches:",
		branchIndent + "  - main",
		branchIndent + "  - master",
	}

	result := make([]string, 0, len(lines)+len(entries))
	result = append(result, lines[:workflowRunEnd]...)
	result = append(result, entries...)
	result = append(result, lines[workflowRunEnd:]...)
	return result, true
}
