package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var agentsBrowseLog = logger.New("cli:agents_browse")

// AvailableAgent represents an available agent from a repository
type AvailableAgent struct {
	Name        string
	Description string
	Category    string
	Installed   bool
	WorkflowID  string // The ID used for installation (e.g., "ci-doctor")
}

// browseAndInstallAgents provides an interactive interface to browse and install agents
func browseAndInstallAgents(repository string, verbose bool, force bool) error {
	agentsBrowseLog.Printf("Browsing agents from repository: %s", repository)

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Loading available agents from %s...", repository)))
	}

	// Parse repository spec to extract repo slug and version
	spec, err := parseRepoSpec(repository)
	if err != nil {
		return fmt.Errorf("invalid repository specification: %w", err)
	}

	// Install the repository package to get access to workflows
	repoWithVersion := spec.RepoSlug
	if spec.Version != "" {
		repoWithVersion = fmt.Sprintf("%s@%s", spec.RepoSlug, spec.Version)
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Installing repository %s...", repoWithVersion)))
	}

	if err := InstallPackage(repoWithVersion, verbose); err != nil {
		return fmt.Errorf("failed to install repository: %w", err)
	}

	// List available workflows in the repository
	availableWorkflows, err := listWorkflowsWithMetadata(spec.RepoSlug, verbose)
	if err != nil {
		return fmt.Errorf("failed to list workflows: %w", err)
	}

	if len(availableWorkflows) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("No workflows found in repository %s", repository)))
		return nil
	}

	// Get list of installed workflows
	installedWorkflows, err := scanInstalledWorkflows(verbose)
	if err != nil {
		return fmt.Errorf("failed to scan installed workflows: %w", err)
	}

	// Create a map of installed workflow names for quick lookup
	installedMap := make(map[string]bool)
	for _, installed := range installedWorkflows {
		// Extract the workflow ID from the source if available
		// Otherwise use the name
		if strings.Contains(installed.Source, spec.RepoSlug) {
			installedMap[installed.Name] = true
		}
	}

	// Build list of available agents with installation status
	var availableAgents []AvailableAgent
	for _, wf := range availableWorkflows {
		// Infer category from description
		category := inferCategoryFromDescription(wf.Description)
		
		agent := AvailableAgent{
			Name:        wf.Name,
			Description: wf.Description,
			Category:    category,
			WorkflowID:  wf.ID,
			Installed:   installedMap[wf.ID],
		}
		availableAgents = append(availableAgents, agent)
	}

	// Group agents by category for better organization
	categorizedAgents := groupAgentsByCategory(availableAgents)

	// Show interactive selection interface
	selectedAgents, err := showAgentSelectionUI(categorizedAgents, spec.RepoSlug)
	if err != nil {
		return fmt.Errorf("agent selection failed: %w", err)
	}

	if len(selectedAgents) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No agents selected for installation"))
		return nil
	}

	// Install selected agents
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Installing %d agent(s)...", len(selectedAgents))))
	fmt.Fprintln(os.Stderr, "")

	var workflowSpecs []string
	for _, agent := range selectedAgents {
		workflowSpec := fmt.Sprintf("%s/%s", spec.RepoSlug, agent.WorkflowID)
		if spec.Version != "" {
			workflowSpec += "@" + spec.Version
		}
		workflowSpecs = append(workflowSpecs, workflowSpec)
	}

	// Use the existing add workflow functionality
	return AddWorkflows(workflowSpecs, 1, verbose, "", "", force, "", false, false, "", false, "")
}

// groupAgentsByCategory groups agents by their category
func groupAgentsByCategory(agents []AvailableAgent) map[string][]AvailableAgent {
	categories := make(map[string][]AvailableAgent)
	
	for _, agent := range agents {
		category := agent.Category
		if category == "" {
			category = "Other"
		}
		categories[category] = append(categories[category], agent)
	}
	
	return categories
}

// showAgentSelectionUI displays an interactive UI for selecting agents to install
func showAgentSelectionUI(categorizedAgents map[string][]AvailableAgent, repository string) ([]AvailableAgent, error) {
	agentsBrowseLog.Print("Showing agent selection UI")

	// Build options for the multi-select
	var options []huh.Option[string]
	var optionToAgent = make(map[string]AvailableAgent)

	// Define category order for consistent display
	categoryOrder := []string{
		"Triage & Analysis",
		"Research & Planning",
		"Coding & Development",
		"Other",
	}

	// Map categories to their display order
	categoryMap := map[string]string{
		"Triage":        "Triage & Analysis",
		"Analysis":      "Triage & Analysis",
		"Research":      "Research & Planning",
		"Planning":      "Research & Planning",
		"Scheduled":     "Research & Planning",
		"Coding":        "Coding & Development",
		"Documentation": "Coding & Development",
		"Other":         "Other",
	}

	// Regroup agents by display categories
	displayCategories := make(map[string][]AvailableAgent)
	for category, agents := range categorizedAgents {
		displayCat := categoryMap[category]
		if displayCat == "" {
			displayCat = "Other"
		}
		displayCategories[displayCat] = append(displayCategories[displayCat], agents...)
	}

	// Build options in category order
	for _, displayCat := range categoryOrder {
		agents, exists := displayCategories[displayCat]
		if !exists || len(agents) == 0 {
			continue
		}

		// Add agents in this category (no separator since Disabled is not available)
		for _, agent := range agents {
			label := agent.Name
			if agent.Installed {
				label += " (installed)"
			}
			if agent.Description != "" {
				label += " - " + agent.Description
			}
			// Prepend category to label
			label = fmt.Sprintf("[%s] %s", displayCat, label)

			key := fmt.Sprintf("%s:%s", displayCat, agent.WorkflowID)
			options = append(options, huh.NewOption(label, key))
			optionToAgent[key] = agent
		}
	}

	if len(options) == 0 {
		return nil, fmt.Errorf("no agents available for installation")
	}

	// Create multi-select form
	var selectedKeys []string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title(fmt.Sprintf("Select agents to install from %s", repository)).
				Description("Use space to select, enter to confirm").
				Options(options...).
				Value(&selectedKeys).
				Height(15),
		),
	)

	// Run the form
	if err := form.Run(); err != nil {
		return nil, fmt.Errorf("form error: %w", err)
	}

	// Convert selected keys back to agents
	var selectedAgents []AvailableAgent
	for _, key := range selectedKeys {
		if agent, ok := optionToAgent[key]; ok {
			selectedAgents = append(selectedAgents, agent)
		}
	}

	agentsBrowseLog.Printf("Selected %d agents for installation", len(selectedAgents))
	return selectedAgents, nil
}

// installAgentsDirect installs agents directly without interactive UI
func installAgentsDirect(repository string, workflowNames []string, verbose bool, force bool) error {
	agentsBrowseLog.Printf("Installing agents directly: %v", workflowNames)

	// Parse repository spec
	spec, err := parseRepoSpec(repository)
	if err != nil {
		return fmt.Errorf("invalid repository specification: %w", err)
	}

	// Build workflow specs
	var workflowSpecs []string
	for _, name := range workflowNames {
		workflowSpec := fmt.Sprintf("%s/%s", spec.RepoSlug, name)
		if spec.Version != "" {
			workflowSpec += "@" + spec.Version
		}
		workflowSpecs = append(workflowSpecs, workflowSpec)
	}

	// Use the existing add workflow functionality
	return AddWorkflows(workflowSpecs, 1, verbose, "", "", force, "", false, false, "", false, "")
}

// inferCategoryFromDescription infers the category from a description
func inferCategoryFromDescription(description string) string {
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
