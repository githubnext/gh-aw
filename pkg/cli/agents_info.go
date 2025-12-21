package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var agentsInfoLog = logger.New("cli:agents_info")

// showAgentInfo displays detailed information about a specific agent
func showAgentInfo(workflowName string, verbose bool, jsonOutput bool) error {
	agentsInfoLog.Printf("Showing info for agent: %s", workflowName)

	// Normalize workflow name (remove .md if present)
	workflowName = strings.TrimSuffix(workflowName, ".md")

	// Find the workflow file
	workflowsDir := getWorkflowsDir()
	workflowPath := filepath.Join(workflowsDir, workflowName+".md")

	// Check if workflow exists
	if _, err := os.Stat(workflowPath); os.IsNotExist(err) {
		// Try to find it in subdirectories
		found := false
		err := filepath.Walk(workflowsDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(path, workflowName+".md") {
				workflowPath = path
				found = true
				return filepath.SkipAll
			}
			return nil
		})
		if err != nil || !found {
			return fmt.Errorf("workflow '%s' not found in %s", workflowName, workflowsDir)
		}
	}

	// Parse agent info
	agent, err := parseAgentInfo(workflowPath, verbose)
	if err != nil {
		return fmt.Errorf("failed to parse workflow: %w", err)
	}

	// Output format
	if jsonOutput {
		return displayAgentInfoJSON(agent)
	}

	return displayAgentInfoFormatted(agent)
}

// displayAgentInfoJSON outputs agent info as JSON
func displayAgentInfoJSON(agent AgentInfo) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(agent)
}

// displayAgentInfoFormatted outputs agent info in a formatted, human-readable way
func displayAgentInfoFormatted(agent AgentInfo) error {
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("═══════════════════════════════════════════════════════"))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Agent: %s", agent.Name)))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("═══════════════════════════════════════════════════════"))
	fmt.Fprintln(os.Stderr, "")

	if agent.Description != "" {
		fmt.Fprintf(os.Stderr, "Description: %s\n", agent.Description)
		fmt.Fprintln(os.Stderr, "")
	}

	fmt.Fprintf(os.Stderr, "Category:    %s\n", agent.Category)
	fmt.Fprintf(os.Stderr, "Status:      %s\n", agent.Status)
	fmt.Fprintf(os.Stderr, "Trigger:     %s\n", agent.Trigger)
	
	if agent.Source != "" && agent.Source != "local" {
		fmt.Fprintf(os.Stderr, "Source:      %s\n", agent.Source)
	}
	
	if len(agent.SafeOutputs) > 0 {
		fmt.Fprintf(os.Stderr, "Safe Outputs: %s\n", strings.Join(agent.SafeOutputs, ", "))
	}

	fmt.Fprintf(os.Stderr, "File Path:   %s\n", agent.FilePath)

	// Show lock file status
	lockFile := strings.TrimSuffix(agent.FilePath, ".md") + ".lock.yml"
	if _, err := os.Stat(lockFile); err == nil {
		fmt.Fprintf(os.Stderr, "Lock File:   %s\n", lockFile)
	} else {
		fmt.Fprintln(os.Stderr, "Lock File:   Not compiled")
	}

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("═══════════════════════════════════════════════════════"))
	fmt.Fprintln(os.Stderr, "")

	return nil
}
