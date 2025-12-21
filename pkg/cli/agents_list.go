package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/parser"
)

var agentsListLog = logger.New("cli:agents_list")

// AgentInfo represents metadata about an installed agentic workflow
type AgentInfo struct {
	Name        string   `console:"Name"`
	Description string   `console:"Description"`
	Category    string   `console:"Category"`
	Source      string   `console:"Source"`
	Status      string   `console:"Status"`
	Trigger     string   `console:"Trigger"`
	SafeOutputs []string `console:"-"`
	FilePath    string   `console:"-"`
}

// scanInstalledWorkflows scans .github/workflows for installed agentic workflows
func scanInstalledWorkflows(verbose bool) ([]AgentInfo, error) {
	agentsListLog.Print("Scanning installed workflows")

	workflowsDir := getWorkflowsDir()
	
	// Check if workflows directory exists
	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		agentsListLog.Printf("Workflows directory does not exist: %s", workflowsDir)
		return []AgentInfo{}, nil
	}

	// Find all .md files in the workflows directory
	var workflowFiles []string
	err := filepath.Walk(workflowsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".md") {
			workflowFiles = append(workflowFiles, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan workflows directory: %w", err)
	}

	agentsListLog.Printf("Found %d workflow files", len(workflowFiles))

	// Parse each workflow file to extract metadata
	var agents []AgentInfo
	for _, filePath := range workflowFiles {
		agent, err := parseAgentInfo(filePath, verbose)
		if err != nil {
			agentsListLog.Printf("Failed to parse workflow %s: %v", filePath, err)
			if verbose {
				fmt.Fprintf(os.Stderr, "Warning: Failed to parse workflow %s: %v\n", filePath, err)
			}
			continue
		}
		agents = append(agents, agent)
	}

	agentsListLog.Printf("Successfully parsed %d agents", len(agents))
	return agents, nil
}

// parseAgentInfo extracts metadata from a workflow file
func parseAgentInfo(filePath string, verbose bool) (AgentInfo, error) {
	agentsListLog.Printf("Parsing agent info from: %s", filePath)

	// Read the file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return AgentInfo{}, fmt.Errorf("failed to read file: %w", err)
	}

	// Parse frontmatter using parser package
	result, err := parser.ExtractFrontmatterFromContent(string(content))
	if err != nil {
		return AgentInfo{}, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	fm := result.Frontmatter
	body := result.Markdown

	// Extract workflow name from filename
	baseName := filepath.Base(filePath)
	workflowName := strings.TrimSuffix(baseName, ".md")

	// Extract description from the first H1 or first paragraph
	description := extractAgentDescription(body)

	// Extract category from frontmatter or infer from description
	category := extractCategory(fm, description)

	// Extract source repository
	source := extractSource(fm)

	// Determine status (enabled/disabled) by checking if .lock.yml exists and is enabled
	status := determineWorkflowStatus(filePath)

	// Extract trigger type
	trigger := extractTrigger(fm)

	// Extract safe outputs
	safeOutputs := extractSafeOutputs(fm)

	agent := AgentInfo{
		Name:        workflowName,
		Description: description,
		Category:    category,
		Source:      source,
		Status:      status,
		Trigger:     trigger,
		SafeOutputs: safeOutputs,
		FilePath:    filePath,
	}

	agentsListLog.Printf("Parsed agent: %s (category=%s, status=%s)", agent.Name, agent.Category, agent.Status)
	return agent, nil
}

// extractAgentDescription extracts the description from the workflow body
func extractAgentDescription(body string) string {
	lines := strings.Split(body, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Skip empty lines
		if line == "" {
			continue
		}
		
		// Skip H1 headers
		if strings.HasPrefix(line, "# ") {
			continue
		}
		
		// Return first non-empty, non-H1 line as description
		if line != "" {
			// Limit description length
			if len(line) > 100 {
				return line[:97] + "..."
			}
			return line
		}
	}
	
	return ""
}

// extractCategory extracts or infers the category from frontmatter or description
func extractCategory(fm map[string]any, description string) string {
	// Check if category is explicitly defined in frontmatter
	if cat, ok := fm["category"].(string); ok && cat != "" {
		return cat
	}

	// Infer category from description or workflow patterns
	descLower := strings.ToLower(description)
	
	if strings.Contains(descLower, "triage") || strings.Contains(descLower, "issue") {
		return "Triage"
	}
	if strings.Contains(descLower, "ci") || strings.Contains(descLower, "doctor") {
		return "Analysis"
	}
	if strings.Contains(descLower, "research") || strings.Contains(descLower, "status") {
		return "Research"
	}
	if strings.Contains(descLower, "daily") || strings.Contains(descLower, "weekly") {
		return "Scheduled"
	}
	if strings.Contains(descLower, "code") || strings.Contains(descLower, "pr") || strings.Contains(descLower, "fix") {
		return "Coding"
	}
	if strings.Contains(descLower, "doc") {
		return "Documentation"
	}
	if strings.Contains(descLower, "plan") {
		return "Planning"
	}
	
	return "Other"
}

// extractSource extracts the source repository from frontmatter
func extractSource(fm map[string]any) string {
	if source, ok := fm["source"].(string); ok && source != "" {
		return source
	}
	return "local"
}

// determineWorkflowStatus determines if a workflow is enabled or disabled
func determineWorkflowStatus(filePath string) string {
	// Check if .lock.yml exists
	lockFile := strings.TrimSuffix(filePath, ".md") + ".lock.yml"
	if _, err := os.Stat(lockFile); os.IsNotExist(err) {
		return "not compiled"
	}

	// Read lock file to check if it's disabled
	content, err := os.ReadFile(lockFile)
	if err != nil {
		return "unknown"
	}

	// Check if the workflow has "disabled_workflow: true" or similar
	if strings.Contains(string(content), "# This workflow is disabled") {
		return "disabled"
	}

	return "enabled"
}

// extractTrigger extracts the trigger type from frontmatter
func extractTrigger(fm map[string]any) string {
	if onTrigger, ok := fm["on"]; ok {
		switch v := onTrigger.(type) {
		case string:
			return v
		case map[string]any:
			// Extract trigger types
			var triggers []string
			for key := range v {
				triggers = append(triggers, key)
			}
			if len(triggers) > 0 {
				return strings.Join(triggers, ", ")
			}
		}
	}
	return "manual"
}

// extractSafeOutputs extracts safe outputs from frontmatter
func extractSafeOutputs(fm map[string]any) []string {
	var outputs []string
	
	if safeOutputs, ok := fm["safe-outputs"].(map[string]any); ok {
		for key := range safeOutputs {
			outputs = append(outputs, key)
		}
	}
	
	return outputs
}

// filterWorkflowsBySource filters workflows by source repository
func filterWorkflowsBySource(workflows []AgentInfo, repoFilter string) []AgentInfo {
	var filtered []AgentInfo
	
	for _, workflow := range workflows {
		// Match if source contains the filter string
		if strings.Contains(workflow.Source, repoFilter) {
			filtered = append(filtered, workflow)
		}
	}
	
	return filtered
}

// displayWorkflowsJSON outputs workflows as JSON
func displayWorkflowsJSON(workflows []AgentInfo) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(workflows)
}
