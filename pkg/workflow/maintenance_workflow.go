package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var maintenanceLog = logger.New("workflow:maintenance_workflow")

// GenerateMaintenanceWorkflow generates the agentics-maintenance.yml workflow
// if any workflows use the expires field for discussions
func GenerateMaintenanceWorkflow(workflowDataList []*WorkflowData, workflowDir string, verbose bool) error {
	maintenanceLog.Print("Checking if maintenance workflow is needed")

	// Check if any workflow uses expires field
	hasExpires := false
	for _, workflowData := range workflowDataList {
		if workflowData.SafeOutputs != nil && workflowData.SafeOutputs.CreateDiscussions != nil {
			if workflowData.SafeOutputs.CreateDiscussions.Expires > 0 {
				hasExpires = true
				maintenanceLog.Printf("Workflow %s has expires field set to %d days", workflowData.Name, workflowData.SafeOutputs.CreateDiscussions.Expires)
				break
			}
		}
	}

	if !hasExpires {
		maintenanceLog.Print("No workflows use expires field, skipping maintenance workflow generation")
		return nil
	}

	maintenanceLog.Print("Generating maintenance workflow for expired discussions")

	// Create the maintenance workflow content using strings.Builder
	var yaml strings.Builder
	
	yaml.WriteString(`name: Agentics Maintenance

on:
  schedule:
    - cron: "0 0 * * *"  # Daily at midnight UTC
  workflow_dispatch:

permissions:
  contents: read
  discussions: write

jobs:
  close-expired-discussions:
    runs-on: ubuntu-latest
    steps:
      - name: Close expired discussions
        uses: actions/github-script@60a0d83039c74a4aee543508d2ffcb1c3799cdea # v7.0.1
        with:
          script: |
`)
	
	// Add the JavaScript script with proper indentation
	script := getMaintenanceScript()
	WriteJavaScriptToYAML(&yaml, script)

	content := yaml.String()

	// Write the maintenance workflow file
	maintenanceFile := filepath.Join(workflowDir, "agentics-maintenance.yml")
	maintenanceLog.Printf("Writing maintenance workflow to %s", maintenanceFile)

	if err := os.WriteFile(maintenanceFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write maintenance workflow: %w", err)
	}

	maintenanceLog.Print("Maintenance workflow generated successfully")
	return nil
}

// getMaintenanceScript returns the embedded JavaScript for the maintenance workflow
func getMaintenanceScript() string {
	return getCloseExpiredDiscussionsScript()
}
