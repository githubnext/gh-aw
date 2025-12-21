package cli

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var agentsUninstallLog = logger.New("cli:agents_uninstall")

// uninstallAgentsInteractive provides an interactive interface to uninstall agents
func uninstallAgentsInteractive(verbose bool, keepOrphans bool) error {
	agentsUninstallLog.Print("Starting interactive agent uninstall")

	// Get list of installed agents
	installedAgents, err := scanInstalledWorkflows(verbose)
	if err != nil {
		return fmt.Errorf("failed to scan installed workflows: %w", err)
	}

	if len(installedAgents) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("No agents installed"))
		return nil
	}

	// Show selection UI
	selectedAgents, err := showUninstallSelectionUI(installedAgents)
	if err != nil {
		return fmt.Errorf("agent selection failed: %w", err)
	}

	if len(selectedAgents) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No agents selected for removal"))
		return nil
	}

	// Confirm removal
	var confirmed bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(fmt.Sprintf("Remove %d agent(s)?", len(selectedAgents))).
				Description("This will delete the workflow files and their compiled versions").
				Value(&confirmed),
		),
	)

	if err := form.Run(); err != nil {
		return fmt.Errorf("confirmation failed: %w", err)
	}

	if !confirmed {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Operation cancelled"))
		return nil
	}

	// Remove selected agents
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Removing %d agent(s)...", len(selectedAgents))))
	fmt.Fprintln(os.Stderr, "")

	for _, agent := range selectedAgents {
		if err := RemoveWorkflows(agent, keepOrphans); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Failed to remove %s: %v", agent, err)))
		} else {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Removed %s", agent)))
		}
	}

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Agent removal complete"))

	return nil
}

// showUninstallSelectionUI displays an interactive UI for selecting agents to uninstall
func showUninstallSelectionUI(agents []AgentInfo) ([]string, error) {
	agentsUninstallLog.Printf("Showing uninstall selection UI for %d agents", len(agents))

	// Build options for multi-select
	var options []huh.Option[string]
	
	// Group by category
	categoryAgents := make(map[string][]AgentInfo)
	for _, agent := range agents {
		category := agent.Category
		if category == "" {
			category = "Other"
		}
		categoryAgents[category] = append(categoryAgents[category], agent)
	}

	// Define category order
	categoryOrder := []string{"Triage", "Analysis", "Research", "Planning", "Scheduled", "Coding", "Documentation", "Other"}

	// Build options in category order
	for _, category := range categoryOrder {
		categoryAgentList, exists := categoryAgents[category]
		if !exists || len(categoryAgentList) == 0 {
			continue
		}

		// Add agents in this category (no separator since Disabled is not available)
		for _, agent := range categoryAgentList {
			label := fmt.Sprintf("[%s] %s (%s)", category, agent.Name, agent.Status)
			if agent.Description != "" {
				label += " - " + agent.Description
			}
			options = append(options, huh.NewOption(label, agent.Name))
		}
	}

	if len(options) == 0 {
		return nil, fmt.Errorf("no agents available for removal")
	}

	// Create multi-select form
	var selectedNames []string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select agents to remove").
				Description("Use space to select, enter to confirm").
				Options(options...).
				Value(&selectedNames).
				Height(15),
		),
	)

	// Run the form
	if err := form.Run(); err != nil {
		return nil, fmt.Errorf("form error: %w", err)
	}

	agentsUninstallLog.Printf("Selected %d agents for removal", len(selectedNames))
	return selectedNames, nil
}

// uninstallAgentsDirect removes agents directly without interactive UI
func uninstallAgentsDirect(workflowNames []string, verbose bool, keepOrphans bool) error {
	agentsUninstallLog.Printf("Removing agents directly: %v", workflowNames)

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Removing %d agent(s)...", len(workflowNames))))

	for _, name := range workflowNames {
		if err := RemoveWorkflows(name, keepOrphans); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Failed to remove %s: %v", name, err)))
		} else {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Removed %s", name)))
		}
	}

	return nil
}
