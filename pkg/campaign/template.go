package campaign

import (
	"bytes"
	_ "embed"
	"strings"
	"text/template"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var templateLog = logger.New("campaign:template")

//go:embed prompts/orchestrator_instructions.md
var orchestratorInstructionsTemplate string

//go:embed prompts/project_update_instructions.md
var projectUpdateInstructionsTemplate string

//go:embed prompts/closing_instructions.md
var closingInstructionsTemplate string

// CampaignPromptData holds data for rendering campaign orchestrator prompts.
type CampaignPromptData struct {
	// Objective is the campaign objective statement.
	Objective string

	// KPIs is the KPI definition list for this campaign.
	KPIs []CampaignKPI

	// ProjectURL is the GitHub Project URL
	ProjectURL string

	// TrackerLabel is the label used to associate issues/PRs with this campaign.
	TrackerLabel string

	// CursorGlob is a glob for locating the durable cursor/checkpoint file in repo-memory.
	CursorGlob string

	// MetricsGlob is a glob for locating the metrics snapshot directory in repo-memory.
	MetricsGlob string

	// MaxDiscoveryItemsPerRun caps how many candidate items may be scanned during discovery.
	MaxDiscoveryItemsPerRun int

	// MaxDiscoveryPagesPerRun caps how many pages may be fetched during discovery.
	MaxDiscoveryPagesPerRun int
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

// RenderOrchestratorInstructions renders the orchestrator instructions with the given data.
func RenderOrchestratorInstructions(data CampaignPromptData) string {
	result, err := renderTemplate(orchestratorInstructionsTemplate, data)
	if err != nil {
		templateLog.Printf("Failed to render orchestrator instructions: %v", err)
		// Fallback to a simple version if template rendering fails
		return "Each time this orchestrator runs, generate a concise status report for this campaign."
	}
	return strings.TrimSpace(result)
}

// RenderProjectUpdateInstructions renders the project update instructions with the given data
func RenderProjectUpdateInstructions(data CampaignPromptData) string {
	result, err := renderTemplate(projectUpdateInstructionsTemplate, data)
	if err != nil {
		templateLog.Printf("Failed to render project update instructions: %v", err)
		return ""
	}
	return strings.TrimSpace(result)
}

// RenderClosingInstructions renders the closing instructions
func RenderClosingInstructions() string {
	result, err := renderTemplate(closingInstructionsTemplate, CampaignPromptData{})
	if err != nil {
		templateLog.Printf("Failed to render closing instructions: %v", err)
		return "Use these details to coordinate workers and track progress."
	}
	return strings.TrimSpace(result)
}
