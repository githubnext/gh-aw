package cli

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow/codemod"
)

var workflowRunBranchesCodemodLog = logger.New("cli:codemod_workflow_run_branches")

type workflowRunBranchesCodemodDeps struct {
	resolveCurrentRepoDefaultBranch func() (string, error)
}

func defaultWorkflowRunBranchesCodemodDeps() workflowRunBranchesCodemodDeps {
	return workflowRunBranchesCodemodDeps{
		resolveCurrentRepoDefaultBranch: func() (string, error) {
			repoSlug := getRepositorySlugFromRemote()
			if repoSlug == "" {
				return "", errors.New("could not determine repository slug from git remote")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			return getRepoDefaultBranch(ctx, repoSlug)
		},
	}
}

// getWorkflowRunBranchesCodemod adds default branch restrictions for bare workflow_run triggers.
func getWorkflowRunBranchesCodemod() Codemod {
	return getWorkflowRunBranchesCodemodWithDeps(defaultWorkflowRunBranchesCodemodDeps())
}

func getWorkflowRunBranchesCodemodWithDeps(deps workflowRunBranchesCodemodDeps) Codemod {
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
			defaultBranch, err := deps.resolveCurrentRepoDefaultBranch()
			if err != nil {
				workflowRunBranchesCodemodLog.Printf("Could not resolve repository default branch via GitHub API, falling back to [main, master]: %v", err)
			} else if strings.TrimSpace(defaultBranch) != "" {
				branches = []string{strings.TrimSpace(defaultBranch)}
			}
			branches = normalizeWorkflowRunBranches(branches)

			newContent, applied, err := codemod.Apply(content, func(f *codemod.Frontmatter) error {
				return f.Set([]string{"on", "workflow_run", "branches"}, branches)
			})
			if applied {
				workflowRunBranchesCodemodLog.Printf("Added branch restrictions to on.workflow_run: %v", branches)
			}
			return newContent, applied, err
		},
	}
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
