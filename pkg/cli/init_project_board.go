package cli

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var initProjectBoardLog = logger.New("cli:init_project_board")

//go:embed templates/orchestrator.md
var orchestratorTemplate string

//go:embed templates/issue-template-research.yml
var issueTemplateResearch string

//go:embed templates/issue-template-analysis.yml
var issueTemplateAnalysis string

// ensureProjectBoardOrchestrator creates the orchestrator workflow
func ensureProjectBoardOrchestrator(verbose bool) error {
	initProjectBoardLog.Print("Creating orchestrator workflow")

	workflowsDir := filepath.Join(constants.GetWorkflowDir())
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		initProjectBoardLog.Printf("Failed to create workflows directory: %v", err)
		return fmt.Errorf("failed to create workflows directory: %w", err)
	}

	orchestratorPath := filepath.Join(workflowsDir, "orchestrator.md")

	// Check if file already exists
	if _, err := os.Stat(orchestratorPath); err == nil {
		initProjectBoardLog.Print("Orchestrator workflow already exists, skipping")
		if verbose {
			fmt.Fprintf(os.Stderr, "Orchestrator workflow already exists: %s\n", orchestratorPath)
		}
		return nil
	}

	if err := os.WriteFile(orchestratorPath, []byte(orchestratorTemplate), 0644); err != nil {
		initProjectBoardLog.Printf("Failed to write orchestrator workflow: %v", err)
		return fmt.Errorf("failed to write orchestrator workflow: %w", err)
	}

	initProjectBoardLog.Printf("Created orchestrator workflow at %s", orchestratorPath)
	return nil
}

// ensureIssueTemplates creates issue templates for workflow starters
func ensureIssueTemplates(verbose bool) error {
	initProjectBoardLog.Print("Creating issue templates")

	issueTemplateDir := filepath.Join(".github", "ISSUE_TEMPLATE")
	if err := os.MkdirAll(issueTemplateDir, 0755); err != nil {
		initProjectBoardLog.Printf("Failed to create ISSUE_TEMPLATE directory: %v", err)
		return fmt.Errorf("failed to create ISSUE_TEMPLATE directory: %w", err)
	}

	templates := map[string]string{
		"research.yml": issueTemplateResearch,
		"analysis.yml": issueTemplateAnalysis,
	}

	for filename, content := range templates {
		templatePath := filepath.Join(issueTemplateDir, filename)

		// Check if file already exists
		if _, err := os.Stat(templatePath); err == nil {
			initProjectBoardLog.Printf("Issue template %s already exists, skipping", filename)
			if verbose {
				fmt.Fprintf(os.Stderr, "Issue template already exists: %s\n", templatePath)
			}
			continue
		}

		if err := os.WriteFile(templatePath, []byte(content), 0644); err != nil {
			initProjectBoardLog.Printf("Failed to write issue template %s: %v", filename, err)
			return fmt.Errorf("failed to write issue template %s: %w", filename, err)
		}

		initProjectBoardLog.Printf("Created issue template at %s", templatePath)
	}

	return nil
}
