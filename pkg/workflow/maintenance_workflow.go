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

	// Check if any workflow uses expires field for discussions or issues
	hasExpires := false
	for _, workflowData := range workflowDataList {
		if workflowData.SafeOutputs != nil {
			// Check for expired discussions
			if workflowData.SafeOutputs.CreateDiscussions != nil {
				if workflowData.SafeOutputs.CreateDiscussions.Expires > 0 {
					hasExpires = true
					maintenanceLog.Printf("Workflow %s has expires field set to %d days for discussions", workflowData.Name, workflowData.SafeOutputs.CreateDiscussions.Expires)
					break
				}
			}
			// Check for expired issues
			if workflowData.SafeOutputs.CreateIssues != nil {
				if workflowData.SafeOutputs.CreateIssues.Expires > 0 {
					hasExpires = true
					maintenanceLog.Printf("Workflow %s has expires field set to %d days for issues", workflowData.Name, workflowData.SafeOutputs.CreateIssues.Expires)
					break
				}
			}
		}
	}

	if !hasExpires {
		maintenanceLog.Print("No workflows use expires field, skipping maintenance workflow generation")
		return nil
	}

	maintenanceLog.Print("Generating maintenance workflow for expired discussions and issues")

	// Create the maintenance workflow content using strings.Builder
	var yaml strings.Builder

	// Add workflow header with logo and instructions
	customInstructions := `Alternative regeneration methods:
  make recompile

Or use the gh-aw CLI directly:
  ./gh-aw compile --validate --verbose

The workflow is generated when any workflow uses the 'expires' field
in create-discussions or create-issues safe-outputs configuration.`

	header := GenerateWorkflowHeader("", "pkg/workflow/maintenance_workflow.go", customInstructions)
	yaml.WriteString(header)

	yaml.WriteString(`name: Agentics Maintenance

on:
  schedule:
    - cron: "0 0 * * *"  # Daily at midnight UTC
  workflow_dispatch:

permissions: {}

jobs:
  close-expired-discussions:
    runs-on: ubuntu-latest
    permissions:
      discussions: write
    steps:
      - name: Close expired discussions
        uses: ` + GetActionPin("actions/github-script") + `
        with:
          script: |
`)

	// Add the close expired discussions script using require()
	yaml.WriteString(`            const { setupGlobals } = require('/tmp/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io);
            const { main } = require('/tmp/gh-aw/actions/close_expired_discussions.cjs');
            await main();
`)

	// Add close-expired-issues job
	yaml.WriteString(`
  close-expired-issues:
    runs-on: ubuntu-latest
    permissions:
      issues: write
    steps:
      - name: Close expired issues
        uses: ` + GetActionPin("actions/github-script") + `
        with:
          script: |
`)

	// Add the close expired issues script using require()
	yaml.WriteString(`            const { setupGlobals } = require('/tmp/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io);
            const { main } = require('/tmp/gh-aw/actions/close_expired_issues.cjs');
            await main();
`)

	// Add compile-workflows job
	yaml.WriteString(`
  compile-workflows:
    runs-on: ubuntu-latest
    permissions:
      contents: read
    steps:
      - name: Checkout repository
        uses: actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683 # v4.2.2

      - name: Setup Go
        uses: actions/setup-go@41dfa10bad2bb2ae585af6ee5bb4d7d973ad74ed # v5.1.0
        with:
          go-version-file: go.mod
          cache: true

      - name: Build gh-aw
        run: make build

      - name: Compile workflows
        run: |
          ./gh-aw compile --validate --verbose
          echo "✓ All workflows compiled successfully"

      - name: Check for out-of-sync workflows
        run: |
          if git diff --exit-code .github/workflows/*.lock.yml; then
            echo "✓ All workflow lock files are up to date"
          else
            echo "::error::Some workflow lock files are out of sync. Run 'make recompile' locally."
            echo "::group::Diff of out-of-sync files"
            git diff .github/workflows/*.lock.yml
            echo "::endgroup::"
            exit 1
          fi

  zizmor-scan:
    runs-on: ubuntu-latest
    needs: compile-workflows
    permissions:
      contents: read
    steps:
      - name: Checkout repository
        uses: actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683 # v4.2.2

      - name: Setup Go
        uses: actions/setup-go@41dfa10bad2bb2ae585af6ee5bb4d7d973ad74ed # v5.1.0
        with:
          go-version-file: go.mod
          cache: true

      - name: Build gh-aw
        run: make build

      - name: Run zizmor security scanner
        run: |
          ./gh-aw compile --zizmor --verbose
          echo "✓ Zizmor security scan completed"
`)

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
