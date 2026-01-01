package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var maintenanceLog = logger.New("workflow:maintenance_workflow")

// generateMaintenanceCron generates a cron schedule based on the minimum expires value in hours
// Schedule runs at minimum required frequency to check expirations at appropriate intervals
// Returns cron expression and description.
func generateMaintenanceCron(minExpiresHours int) (string, string) {
	// Use a pseudo-random but deterministic minute (37) to avoid load spikes at :00
	minute := 37

	// Determine frequency based on minimum expires value (in hours)
	// Run at least as often as the shortest expiration would need
	if minExpiresHours <= 2 {
		// For 2 hours or less, run every hour
		return fmt.Sprintf("%d * * * *", minute), "Every hour"
	} else if minExpiresHours <= 4 {
		// For 3-4 hours, run every 2 hours
		return fmt.Sprintf("%d */2 * * *", minute), "Every 2 hours"
	} else if minExpiresHours <= 12 {
		// For 5-12 hours, run every 4 hours
		return fmt.Sprintf("%d */4 * * *", minute), "Every 4 hours"
	} else if minExpiresHours <= 24 {
		// For 13-24 hours, run every 6 hours
		return fmt.Sprintf("%d */6 * * *", minute), "Every 6 hours"
	} else if minExpiresHours <= 48 {
		// For 25-48 hours, run every 12 hours
		return fmt.Sprintf("%d */12 * * *", minute), "Every 12 hours"
	}

	// For more than 48 hours, run daily
	return fmt.Sprintf("%d %d * * *", minute, 0), "Daily"
}

// GenerateMaintenanceWorkflow generates the agentics-maintenance.yml workflow
// if any workflows use the expires field for discussions
func GenerateMaintenanceWorkflow(workflowDataList []*WorkflowData, workflowDir string, version string, actionMode ActionMode, verbose bool) error {
	maintenanceLog.Print("Checking if maintenance workflow is needed")

	// Check if any workflow uses expires field for discussions or issues
	// and track the minimum expires value to determine schedule frequency
	hasExpires := false
	minExpires := 0 // Track minimum expires value in hours
	for _, workflowData := range workflowDataList {
		if workflowData.SafeOutputs != nil {
			// Check for expired discussions
			if workflowData.SafeOutputs.CreateDiscussions != nil {
				if workflowData.SafeOutputs.CreateDiscussions.Expires > 0 {
					hasExpires = true
					expires := workflowData.SafeOutputs.CreateDiscussions.Expires
					maintenanceLog.Printf("Workflow %s has expires field set to %d hours for discussions", workflowData.Name, expires)
					if minExpires == 0 || expires < minExpires {
						minExpires = expires
					}
				}
			}
			// Check for expired issues
			if workflowData.SafeOutputs.CreateIssues != nil {
				if workflowData.SafeOutputs.CreateIssues.Expires > 0 {
					hasExpires = true
					expires := workflowData.SafeOutputs.CreateIssues.Expires
					maintenanceLog.Printf("Workflow %s has expires field set to %d hours for issues", workflowData.Name, expires)
					if minExpires == 0 || expires < minExpires {
						minExpires = expires
					}
				}
			}
		}
	}

	if !hasExpires {
		maintenanceLog.Print("No workflows use expires field, skipping maintenance workflow generation")
		return nil
	}

	maintenanceLog.Printf("Generating maintenance workflow for expired discussions and issues (minimum expires: %d hours)", minExpires)

	// Generate cron schedule based on minimum expires value
	cronSchedule, scheduleDesc := generateMaintenanceCron(minExpires)
	maintenanceLog.Printf("Maintenance schedule: %s (%s)", cronSchedule, scheduleDesc)

	// Create the maintenance workflow content using strings.Builder
	var yaml strings.Builder

	// Add workflow header with logo and instructions
	customInstructions := `Alternative regeneration methods:
  make recompile

Or use the gh-aw CLI directly:
  ./gh-aw compile --validate --verbose

The workflow is generated when any workflow uses the 'expires' field
in create-discussions or create-issues safe-outputs configuration.
Schedule frequency is automatically determined by the shortest expiration time.`

	header := GenerateWorkflowHeader("", "pkg/workflow/maintenance_workflow.go", customInstructions)
	yaml.WriteString(header)

	yaml.WriteString(`name: Agentics Maintenance

on:
  schedule:
    - cron: "` + cronSchedule + `"  # ` + scheduleDesc + ` (based on minimum expires: ` + fmt.Sprintf("%d", minExpires) + ` hours)
  workflow_dispatch:

permissions: {}

jobs:
  close-expired-discussions:
    runs-on: ubuntu-latest
    permissions:
      discussions: write
    steps:
`)

	// Get the setup action reference (local or remote based on mode)
	setupActionRef := ResolveSetupActionReference(actionMode, version)

	// Add checkout step only in dev mode (for local action paths)
	if actionMode == ActionModeDev {
		yaml.WriteString(`      - name: Checkout actions folder
        uses: ` + GetActionPin("actions/checkout") + `
        with:
          sparse-checkout: |
            actions
          persist-credentials: false

`)
	}

	// Add setup step with the resolved action reference
	yaml.WriteString(`      - name: Setup Scripts
        uses: ` + setupActionRef + `
        with:
          destination: /tmp/gh-aw/actions

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
`)

	// Add checkout step only in dev mode (for local action paths)
	if actionMode == ActionModeDev {
		yaml.WriteString(`      - name: Checkout actions folder
        uses: ` + GetActionPin("actions/checkout") + `
        with:
          sparse-checkout: |
            actions
          persist-credentials: false

`)
	}

	// Add setup step with the resolved action reference
	yaml.WriteString(`      - name: Setup Scripts
        uses: ` + setupActionRef + `
        with:
          destination: /tmp/gh-aw/actions

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
