package campaign

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var templateLog = logger.New("campaign:template")

// findGitRoot finds the root directory of the git repository
func findGitRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not in a git repository or git command failed: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// loadTemplate loads a template file from .github/aw/ directory
func loadTemplate(filename string) (string, error) {
	gitRoot, err := findGitRoot()
	if err != nil {
		return "", fmt.Errorf("failed to find git root: %w", err)
	}

	templatePath := filepath.Join(gitRoot, ".github", "aw", filename)
	templateLog.Printf("Loading template from: %s", templatePath)

	content, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to read template file %s: %w", filename, err)
	}

	return string(content), nil
}

// CampaignPromptData holds data for rendering campaign orchestrator prompts.
type CampaignPromptData struct {
	// CampaignID is the unique identifier for this campaign.
	CampaignID string

	// CampaignName is the human-readable name of this campaign.
	CampaignName string

	// Objective is the campaign objective statement.
	Objective string

	// KPIs is the KPI definition list for this campaign.
	KPIs []CampaignKPI

	// ProjectURL is the GitHub Project URL
	ProjectURL string

	// CursorGlob is a glob for locating the durable cursor/checkpoint file in repo-memory.
	CursorGlob string

	// MetricsGlob is a glob for locating the metrics snapshot directory in repo-memory.
	MetricsGlob string

	// MaxDiscoveryItemsPerRun caps how many candidate items may be scanned during discovery.
	MaxDiscoveryItemsPerRun int

	// MaxDiscoveryPagesPerRun caps how many pages may be fetched during discovery.
	MaxDiscoveryPagesPerRun int

	// MaxProjectUpdatesPerRun caps how many project update writes may happen per run.
	MaxProjectUpdatesPerRun int

	// MaxProjectCommentsPerRun caps how many comments may be written per run.
	MaxProjectCommentsPerRun int

	// Workflows is the list of worker workflow IDs associated with this campaign.
	Workflows []string
}

// renderTemplate renders a template string with the given data.
func renderTemplate(tmplStr string, data CampaignPromptData) (string, error) {
	// Create custom template functions for Handlebars-style conditionals
	funcMap := template.FuncMap{
		"if": func(condition bool) bool {
			return condition
		},
	}

	// Parse template with custom delimiters to match Handlebars style
	tmpl, err := template.New("prompt").
		Delims("{{", "}}").
		Funcs(funcMap).
		Parse(tmplStr)
	if err != nil {
		templateLog.Printf("Failed to parse template: %v", err)
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		templateLog.Printf("Failed to execute template: %v", err)
		return "", err
	}

	return buf.String(), nil
}

// RenderWorkflowExecution renders the workflow execution instructions with the given data.
func RenderWorkflowExecution(data CampaignPromptData) string {
	tmplStr, err := loadTemplate("execute-campaign-workflow.md")
	if err != nil {
		templateLog.Printf("Failed to load workflow execution template: %v", err)
		return ""
	}

	result, err := renderTemplate(tmplStr, data)
	if err != nil {
		templateLog.Printf("Failed to render workflow execution instructions: %v", err)
		return ""
	}
	return strings.TrimSpace(result)
}

// RenderOrchestratorInstructions renders the orchestrator instructions with the given data.
func RenderOrchestratorInstructions(data CampaignPromptData) string {
	tmplStr, err := loadTemplate("orchestrate-campaign.md")
	if err != nil {
		templateLog.Printf("Failed to load orchestrator instructions template: %v", err)
		// Fallback to a simple version if template loading fails
		return "Each time this orchestrator runs, generate a concise status report for this campaign."
	}

	result, err := renderTemplate(tmplStr, data)
	if err != nil {
		templateLog.Printf("Failed to render orchestrator instructions: %v", err)
		// Fallback to a simple version if template rendering fails
		return "Each time this orchestrator runs, generate a concise status report for this campaign."
	}
	return strings.TrimSpace(result)
}

// RenderProjectUpdateInstructions renders the project update instructions with the given data
func RenderProjectUpdateInstructions(data CampaignPromptData) string {
	tmplStr, err := loadTemplate("update-campaign-project-contract.md")
	if err != nil {
		templateLog.Printf("Failed to load project update instructions template: %v", err)
		return ""
	}

	result, err := renderTemplate(tmplStr, data)
	if err != nil {
		templateLog.Printf("Failed to render project update instructions: %v", err)
		return ""
	}
	return strings.TrimSpace(result)
}

// RenderClosingInstructions renders the closing instructions
func RenderClosingInstructions() string {
	tmplStr, err := loadTemplate("close-agentic-campaign.md")
	if err != nil {
		templateLog.Printf("Failed to load closing instructions template: %v", err)
		return "Use these details to coordinate workers and track progress."
	}

	result, err := renderTemplate(tmplStr, CampaignPromptData{})
	if err != nil {
		templateLog.Printf("Failed to render closing instructions: %v", err)
		return "Use these details to coordinate workers and track progress."
	}
	return strings.TrimSpace(result)
}
