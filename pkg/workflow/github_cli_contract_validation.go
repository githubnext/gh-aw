package workflow

import (
	"context"
	"fmt"
	"maps"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/workflow/aw"
	workflowlint "github.com/github/gh-aw/pkg/workflow/lint"
)

func (c *Compiler) validateGitHubCLIContracts(data *WorkflowData, markdownPath string, contents []string) {
	runner, err := workflowlint.New(workflowlint.GitHubCLISyntaxRule(), workflowlint.GitHubProxyDiffRule())
	if err != nil {
		fmt.Fprintln(os.Stderr, formatCompilerMessage(markdownPath, "warning", "GitHub CLI contract linting unavailable; check the workflow with gh aw lint"))
		c.IncrementWarningCount()
		return
	}
	frontmatter := maps.Clone(data.RawFrontmatter)
	if frontmatter == nil {
		frontmatter = make(map[string]any)
	}
	frontmatter["tools"] = data.Tools
	diagnostics, err := runner.Run(context.Background(), workflowlint.Input{Document: &aw.Document{
		Frontmatter: frontmatter, Markdown: strings.Join(contents, "\n\n"),
	}})
	if err != nil {
		fmt.Fprintln(os.Stderr, formatCompilerMessage(markdownPath, "warning", "GitHub CLI contract linting incomplete; check authored steps and prompt commands with gh aw lint"))
		c.IncrementWarningCount()
	}
	seen := make(map[string]struct{})
	for _, diagnostic := range diagnostics {
		key := diagnostic.RuleID + diagnostic.Path + diagnostic.Message
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		fmt.Fprintln(os.Stderr, formatCompilerMessage(markdownPath, "warning", "["+diagnostic.RuleID+"] "+diagnostic.Path+": "+diagnostic.Message))
		c.IncrementWarningCount()
	}
}
