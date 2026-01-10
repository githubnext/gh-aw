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

//go:embed prompts/workflow_execution_instructions.md
var workflowExecutionInstructionsTemplate string

//go:embed prompts/project_update_instructions.md
var projectUpdateInstructionsTemplate string

//go:embed prompts/closing_instructions.md
var closingInstructionsTemplate string

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

	// ExecutionSequence is the optional workflow execution configuration.
	ExecutionSequence []WorkflowExecutionStep

	// MaxConcurrentWorkflows limits parallel workflow execution.
	MaxConcurrentWorkflows int

	// TimeoutMinutes is the workflow execution timeout.
	TimeoutMinutes int
}

// renderTemplate renders a template string with the given data.
func renderTemplate(tmplStr string, data CampaignPromptData) (string, error) {
	// Create custom template functions for Handlebars-style conditionals
	funcMap := template.FuncMap{
		"if": func(condition bool) bool {
			return condition
		},
		"add1": func(i int) int {
			return i + 1
		},
		"gt": func(a, b int) bool {
			return a > b
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

// RenderWorkflowExecutionInstructions renders the workflow execution instructions with the given data.
func RenderWorkflowExecutionInstructions(data CampaignPromptData) string {
	result, err := renderTemplate(workflowExecutionInstructionsTemplate, data)
	if err != nil {
		templateLog.Printf("Failed to render workflow execution instructions: %v", err)
		return ""
	}
	return strings.TrimSpace(result)
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
