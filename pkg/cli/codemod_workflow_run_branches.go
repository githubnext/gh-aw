package cli

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/logger"
)

var workflowRunBranchesCodemodLog = logger.New("cli:codemod_workflow_run_branches")

var resolveCurrentRepoDefaultBranchFn = func() (string, error) {
	repoSlug := getRepositorySlugFromRemote()
	if repoSlug == "" {
		return "", errors.New("could not determine repository slug from git remote")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return getRepoDefaultBranch(ctx, repoSlug)
}

// getWorkflowRunBranchesCodemod adds default branch restrictions for bare workflow_run triggers.
func getWorkflowRunBranchesCodemod() Codemod {
	return Codemod{
		ID:           "workflow-run-branches-default",
		Name:         "Add workflow_run branch restrictions",
		Description:  "Adds default branch restriction to on.workflow_run when branches are missing (falls back to [main, master])",
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

			branches := []string{"main", "master"}
			defaultBranch, err := resolveCurrentRepoDefaultBranchFn()
			if err != nil {
				workflowRunBranchesCodemodLog.Printf("Could not resolve repository default branch via GitHub API, falling back to [main, master]: %v", err)
			} else if strings.TrimSpace(defaultBranch) != "" {
				branches = []string{strings.TrimSpace(defaultBranch)}
			}
			branches = normalizeWorkflowRunBranches(branches)

			newContent, applied, err := applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
				return addWorkflowRunBranches(lines, branches)
			})
			if applied {
				workflowRunBranchesCodemodLog.Printf("Added branch restrictions to on.workflow_run: %v", branches)
			}
			return newContent, applied, err
		},
	}
}

func addWorkflowRunBranches(lines []string, branches []string) ([]string, bool) {
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
	entries := make([]string, 0, len(branches)+1)
	entries = append(entries, branchIndent+"branches:")
	for _, branch := range branches {
		entries = append(entries, branchIndent+"  - "+branch)
	}

	result := make([]string, 0, len(lines)+len(entries))
	result = append(result, lines[:workflowRunEnd]...)
	result = append(result, entries...)
	result = append(result, lines[workflowRunEnd:]...)
	return result, true
}

func normalizeWorkflowRunBranches(branches []string) []string {
	normalized := make([]string, 0, len(branches))
	seen := make(map[string]struct{})
	for _, branch := range branches {
		trimmed := strings.TrimSpace(branch)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}

	if len(normalized) == 0 {
		return []string{"main", "master"}
	}

	return normalized
}
