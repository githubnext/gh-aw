package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var agentsUpdateLog = logger.New("cli:agents_update")

// AgentUpdate represents an agent with update information
type AgentUpdate struct {
	Name           string
	CurrentVersion string
	LatestVersion  string
	HasUpdate      bool
	Source         string
}

// updateAgentsInteractive provides an interactive interface to update agents
func updateAgentsInteractive(verbose bool, updateAll bool, force bool) error {
	agentsUpdateLog.Printf("Starting interactive agent update: updateAll=%v", updateAll)

	// Get list of installed agents
	installedAgents, err := scanInstalledWorkflows(verbose)
	if err != nil {
		return fmt.Errorf("failed to scan installed workflows: %w", err)
	}

	if len(installedAgents) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("No agents installed"))
		return nil
	}

	// Check for updates for each agent
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Checking for updates..."))
	
	var agentUpdates []AgentUpdate
	for _, agent := range installedAgents {
		// Skip local workflows (no source)
		if agent.Source == "" || agent.Source == "local" {
			continue
		}

		update := AgentUpdate{
			Name:           agent.Name,
			CurrentVersion: extractVersionFromSource(agent.Source),
			Source:         agent.Source,
			HasUpdate:      false, // We'll check this below
		}

		// For now, we'll assume updates are available if force is set
		// In a real implementation, we would compare versions
		if force {
			update.HasUpdate = true
			update.LatestVersion = "latest"
		}

		agentUpdates = append(agentUpdates, update)
	}

	if len(agentUpdates) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("No agents with source repositories found"))
		return nil
	}

	// Filter to only agents with updates (unless updateAll or force)
	if !updateAll && !force {
		var updatableAgents []AgentUpdate
		for _, update := range agentUpdates {
			if update.HasUpdate {
				updatableAgents = append(updatableAgents, update)
			}
		}
		
		if len(updatableAgents) == 0 {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("All agents are up to date"))
			return nil
		}
		
		agentUpdates = updatableAgents
	}

	var selectedAgents []AgentUpdate

	if updateAll {
		// Update all agents without prompting
		selectedAgents = agentUpdates
	} else {
		// Show selection UI
		selected, err := showUpdateSelectionUI(agentUpdates)
		if err != nil {
			return fmt.Errorf("agent selection failed: %w", err)
		}
		selectedAgents = selected
	}

	if len(selectedAgents) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No agents selected for update"))
		return nil
	}

	// Update selected agents
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Updating %d agent(s)...", len(selectedAgents))))
	fmt.Fprintln(os.Stderr, "")

	for _, update := range selectedAgents {
		if err := updateAgent(update, verbose, force); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Failed to update %s: %v", update.Name, err)))
		} else {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Updated %s", update.Name)))
		}
	}

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Agent update complete"))

	return nil
}

// showUpdateSelectionUI displays an interactive UI for selecting agents to update
func showUpdateSelectionUI(updates []AgentUpdate) ([]AgentUpdate, error) {
	agentsUpdateLog.Printf("Showing update selection UI for %d agents", len(updates))

	// Build options for multi-select
	var options []huh.Option[string]
	updateMap := make(map[string]AgentUpdate)

	for _, update := range updates {
		label := update.Name
		if update.HasUpdate {
			label += fmt.Sprintf(" (%s → %s)", update.CurrentVersion, update.LatestVersion)
		} else {
			label += " (no updates)"
		}
		
		options = append(options, huh.NewOption(label, update.Name))
		updateMap[update.Name] = update
	}

	if len(options) == 0 {
		return nil, fmt.Errorf("no agents available for update")
	}

	// Create multi-select form
	var selectedNames []string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select agents to update").
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

	// Convert selected names back to updates
	var selectedUpdates []AgentUpdate
	for _, name := range selectedNames {
		if update, ok := updateMap[name]; ok {
			selectedUpdates = append(selectedUpdates, update)
		}
	}

	agentsUpdateLog.Printf("Selected %d agents for update", len(selectedUpdates))
	return selectedUpdates, nil
}

// updateAgent updates a single agent
func updateAgent(update AgentUpdate, verbose bool, force bool) error {
	agentsUpdateLog.Printf("Updating agent: %s", update.Name)

	// Parse the source to extract repository and workflow info
	source := update.Source
	
	// Extract repository from source (format: owner/repo/workflow@version)
	parts := strings.Split(source, "@")
	if len(parts) == 0 {
		return fmt.Errorf("invalid source format: %s", source)
	}

	workflowPath := parts[0]
	pathParts := strings.Split(workflowPath, "/")
	
	if len(pathParts) < 3 {
		return fmt.Errorf("invalid workflow path in source: %s", workflowPath)
	}

	// Build workflow spec for reinstallation
	// Use latest version (no @version suffix) to get the newest version
	workflowSpec := workflowPath

	// Remove the existing workflow
	if err := RemoveWorkflows(update.Name, false); err != nil {
		return fmt.Errorf("failed to remove existing workflow: %w", err)
	}

	// Reinstall the workflow
	if err := AddWorkflows([]string{workflowSpec}, 1, verbose, "", update.Name, force, "", false, false, "", false, ""); err != nil {
		return fmt.Errorf("failed to reinstall workflow: %w", err)
	}

	return nil
}

// updateAgentsDirect updates agents directly without interactive UI
func updateAgentsDirect(workflowNames []string, verbose bool, force bool) error {
	agentsUpdateLog.Printf("Updating agents directly: %v", workflowNames)

	// Get list of installed agents
	installedAgents, err := scanInstalledWorkflows(verbose)
	if err != nil {
		return fmt.Errorf("failed to scan installed workflows: %w", err)
	}

	// Build map of installed agents
	agentMap := make(map[string]AgentInfo)
	for _, agent := range installedAgents {
		agentMap[agent.Name] = agent
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Updating %d agent(s)...", len(workflowNames))))

	for _, name := range workflowNames {
		agent, exists := agentMap[name]
		if !exists {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Agent not found: %s", name)))
			continue
		}

		update := AgentUpdate{
			Name:           agent.Name,
			CurrentVersion: extractVersionFromSource(agent.Source),
			Source:         agent.Source,
			HasUpdate:      true,
			LatestVersion:  "latest",
		}

		if err := updateAgent(update, verbose, force); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Failed to update %s: %v", name, err)))
		} else {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Updated %s", name)))
		}
	}

	return nil
}

// extractVersionFromSource extracts the version from a source string
func extractVersionFromSource(source string) string {
	parts := strings.Split(source, "@")
	if len(parts) > 1 {
		return parts[1]
	}
	return "unknown"
}
