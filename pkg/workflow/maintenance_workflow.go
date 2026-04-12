package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
)

var maintenanceLog = logger.New("workflow:maintenance_workflow")

// generateInstallCLISteps generates YAML steps to install or build the gh-aw CLI.
// In dev mode: builds from source using Setup Go + Build gh-aw (./gh-aw binary available)
// In release mode: installs the released CLI via the setup-cli action (gh aw available)
// In action mode: installs the released CLI via the gh-aw-actions/setup-cli action (gh aw available)
// When resolver is non-nil, attempts to resolve the setup-cli action to a SHA-pinned reference.
func generateInstallCLISteps(actionMode ActionMode, version string, actionTag string, resolver ActionSHAResolver) string {
	if actionMode == ActionModeDev {
		return `      - name: Setup Go
        uses: ` + GetActionPin("actions/setup-go") + `
        with:
          go-version-file: go.mod
          cache: true

      - name: Build gh-aw
        run: make build

`
	}

	cliTag := actionTag
	if cliTag == "" {
		cliTag = version
	}

	// Action mode: use setup-cli action from external gh-aw-actions repository
	if actionMode == ActionModeAction {
		actionRepo := GitHubActionsOrgRepo + "/setup-cli"
		ref := resolveActionRef(actionRepo, cliTag, resolver)
		return `      - name: Install gh-aw
        uses: ` + ref + `
        with:
          version: ` + cliTag + `

`
	}

	// Release mode: use setup-cli action (consistent with copilot-setup-steps.yml)
	actionRepo := GitHubOrgRepo + "/actions/setup-cli"
	ref := resolveActionRef(actionRepo, cliTag, resolver)
	return `      - name: Install gh-aw
        uses: ` + ref + `
        with:
          version: ` + cliTag + `

`
}

// resolveActionRef attempts to resolve an action repo@tag to a SHA-pinned reference
// using the provided resolver. If the resolver is nil or resolution fails, it returns
// the tag-based reference (repo@tag).
func resolveActionRef(actionRepo, tag string, resolver ActionSHAResolver) string {
	if resolver != nil && tag != "" && tag != "dev" {
		sha, err := resolver.ResolveSHA(actionRepo, tag)
		if err != nil {
			maintenanceLog.Printf("Failed to resolve SHA for %s@%s: %v, falling back to tag reference", actionRepo, tag, err)
		} else if sha != "" {
			return formatActionReference(actionRepo, sha, tag)
		}
	}
	return actionRepo + "@" + tag
}

// getCLICmdPrefix returns the CLI command prefix based on action mode.
// In dev mode: "./gh-aw" (local binary built from source)
// In release mode: "gh aw" (installed via gh extension)
func getCLICmdPrefix(actionMode ActionMode) string {
	if actionMode == ActionModeDev {
		return "./gh-aw"
	}
	return "gh aw"
}

// generateMaintenanceCron generates a cron schedule based on the minimum expires value in days
// Schedule runs at minimum required frequency to check expirations at appropriate intervals
// Returns cron expression and description.
func generateMaintenanceCron(minExpiresDays int) (string, string) {
	// Use a pseudo-random but deterministic minute (37) to avoid load spikes at :00
	minute := 37

	// Determine frequency based on minimum expires value (in days)
	// Run at least as often as the shortest expiration would need
	if minExpiresDays <= 1 {
		// For 1 day or less, run every 2 hours
		return fmt.Sprintf("%d */2 * * *", minute), "Every 2 hours"
	} else if minExpiresDays == 2 {
		// For 2 days, run every 6 hours
		return fmt.Sprintf("%d */6 * * *", minute), "Every 6 hours"
	} else if minExpiresDays <= 4 {
		// For 3-4 days, run every 12 hours
		return fmt.Sprintf("%d */12 * * *", minute), "Every 12 hours"
	}

	// For more than 4 days, run daily
	return fmt.Sprintf("%d %d * * *", minute, 0), "Daily"
}

// GenerateMaintenanceWorkflow generates the agentics-maintenance.yml workflow
// if any workflows use the expires field for discussions or issues.
// When repoConfig is non-nil and repoConfig.MaintenanceDisabled is true the
// maintenance workflow is deleted and the function returns immediately.
func GenerateMaintenanceWorkflow(workflowDataList []*WorkflowData, workflowDir string, version string, actionMode ActionMode, actionTag string, verbose bool, repoConfig *RepoConfig) error {
	maintenanceLog.Print("Checking if maintenance workflow is needed")

	// Respect explicit opt-out from aw.json: maintenance: false
	if repoConfig != nil && repoConfig.MaintenanceDisabled {
		maintenanceLog.Print("Maintenance disabled via repo config, skipping generation")

		// Warn if any workflow uses expires — those features rely on maintenance
		// and will silently become no-ops when it is disabled.
		for _, workflowData := range workflowDataList {
			if workflowData.SafeOutputs == nil {
				continue
			}
			usesExpires := (workflowData.SafeOutputs.CreateDiscussions != nil && workflowData.SafeOutputs.CreateDiscussions.Expires > 0) ||
				(workflowData.SafeOutputs.CreateIssues != nil && workflowData.SafeOutputs.CreateIssues.Expires > 0) ||
				(workflowData.SafeOutputs.CreatePullRequests != nil && workflowData.SafeOutputs.CreatePullRequests.Expires > 0)
			if usesExpires {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(
					fmt.Sprintf("Workflow '%s' uses the 'expires' field but maintenance is disabled in aw.json. "+
						"Expiration will not run until maintenance is re-enabled.", workflowData.Name)))
			}
		}

		maintenanceFile := filepath.Join(workflowDir, "agentics-maintenance.yml")
		if _, err := os.Stat(maintenanceFile); err == nil {
			maintenanceLog.Printf("Deleting existing maintenance workflow: %s", maintenanceFile)
			if err := os.Remove(maintenanceFile); err != nil {
				return fmt.Errorf("failed to delete maintenance workflow: %w", err)
			}
		}
		return nil
	}

	// Determine the runs-on value to use for all maintenance jobs.
	const defaultRunsOn = "ubuntu-slim"
	var configuredRunsOn RunsOnValue
	if repoConfig != nil && repoConfig.Maintenance != nil {
		configuredRunsOn = repoConfig.Maintenance.RunsOn
	}
	runsOnValue := FormatRunsOn(configuredRunsOn, defaultRunsOn)

	// Check if any workflow uses expires field for discussions, issues, or pull requests
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
			// Check for expired pull requests
			if workflowData.SafeOutputs.CreatePullRequests != nil {
				if workflowData.SafeOutputs.CreatePullRequests.Expires > 0 {
					hasExpires = true
					expires := workflowData.SafeOutputs.CreatePullRequests.Expires
					maintenanceLog.Printf("Workflow %s has expires field set to %d hours for pull requests", workflowData.Name, expires)
					if minExpires == 0 || expires < minExpires {
						minExpires = expires
					}
				}
			}
		}
	}

	if !hasExpires {
		maintenanceLog.Print("No workflows use expires field, skipping maintenance workflow generation")

		// Delete existing maintenance workflow file if it exists (no expires means no need for maintenance)
		maintenanceFile := filepath.Join(workflowDir, "agentics-maintenance.yml")
		if _, err := os.Stat(maintenanceFile); err == nil {
			maintenanceLog.Printf("Deleting existing maintenance workflow: %s", maintenanceFile)
			if err := os.Remove(maintenanceFile); err != nil {
				return fmt.Errorf("failed to delete maintenance workflow: %w", err)
			}
			maintenanceLog.Print("Maintenance workflow deleted successfully")
		}

		return nil
	}

	maintenanceLog.Printf("Generating maintenance workflow for expired discussions, issues, and pull requests (minimum expires: %d hours)", minExpires)

	// Convert hours to days for cron schedule generation
	minExpiresDays := minExpires / 24
	if minExpires%24 > 0 {
		minExpiresDays++ // Round up partial days
	}

	// Generate cron schedule based on minimum expires value
	cronSchedule, scheduleDesc := generateMaintenanceCron(minExpiresDays)
	maintenanceLog.Printf("Maintenance schedule: %s (%s)", cronSchedule, scheduleDesc)

	// Create the maintenance workflow content using strings.Builder
	var yaml strings.Builder

	// Add workflow header with logo and instructions
	customInstructions := `Alternative regeneration methods:
  make recompile

Or use the gh-aw CLI directly:
  ./gh-aw compile --validate --verbose

The workflow is generated when any workflow uses the 'expires' field
in create-discussions, create-issues, or create-pull-request safe-outputs configuration.
Schedule frequency is automatically determined by the shortest expiration time.`

	header := GenerateWorkflowHeader("", "pkg/workflow/maintenance_workflow.go", customInstructions)
	yaml.WriteString(header)

	yaml.WriteString(`name: Agentic Maintenance

on:
  schedule:
    - cron: "` + cronSchedule + `"  # ` + scheduleDesc + ` (based on minimum expires: ` + strconv.Itoa(minExpiresDays) + ` days)
  workflow_dispatch:
    inputs:
      operation:
        description: 'Optional maintenance operation to run'
        required: false
        type: choice
        default: ''
        options:
          - ''
          - 'disable'
          - 'enable'
          - 'update'
          - 'upgrade'
          - 'safe_outputs'
          - 'create_labels'
          - 'validate'
      run_url:
        description: 'Run URL or run ID to replay safe outputs from (e.g. https://github.com/owner/repo/actions/runs/12345 or 12345). Required when operation is safe_outputs.'
        required: false
        type: string
        default: ''

permissions: {}

jobs:
  close-expired-entities:
    if: ${{ !github.event.repository.fork && (github.event_name != 'workflow_dispatch' || github.event.inputs.operation == '') }}
    runs-on: ` + runsOnValue + `
    permissions:
      discussions: write
      issues: write
      pull-requests: write
    steps:
`)

	// Get the setup action reference (local or remote based on mode)
	// Use the first available WorkflowData's ActionResolver to enable SHA pinning
	var resolver ActionSHAResolver
	if len(workflowDataList) > 0 && workflowDataList[0].ActionResolver != nil {
		resolver = workflowDataList[0].ActionResolver
	}
	setupActionRef := ResolveSetupActionReference(actionMode, version, actionTag, resolver)

	// Add checkout step only in dev/script mode (for local action paths)
	if actionMode == ActionModeDev || actionMode == ActionModeScript {
		yaml.WriteString("      - name: Checkout actions folder\n")
		yaml.WriteString("        uses: " + GetActionPin("actions/checkout") + "\n")
		yaml.WriteString("        with:\n")
		yaml.WriteString("          sparse-checkout: |\n")
		yaml.WriteString("            actions\n")
		yaml.WriteString("          persist-credentials: false\n\n")
	}

	// Add setup step with the resolved action reference
	yaml.WriteString(`      - name: Setup Scripts
        uses: ` + setupActionRef + `
        with:
          destination: ${{ runner.temp }}/gh-aw/actions

      - name: Close expired discussions
        uses: ` + GetActionPin("actions/github-script") + `
        with:
          script: |
`)

	// Add the close expired discussions script using require()
	yaml.WriteString(`            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/close_expired_discussions.cjs');
            await main();

      - name: Close expired issues
        uses: ` + GetActionPin("actions/github-script") + `
        with:
          script: |
`)

	// Add the close expired issues script using require()
	yaml.WriteString(`            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/close_expired_issues.cjs');
            await main();

      - name: Close expired pull requests
        uses: ` + GetActionPin("actions/github-script") + `
        with:
          script: |
`)

	// Add the close expired pull requests script using require()
	yaml.WriteString(`            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/close_expired_pull_requests.cjs');
            await main();
`)

	// Add unified run_operation job for all dispatch operations except safe_outputs, create_labels, and validate
	yaml.WriteString(`
  run_operation:
    if: ${{ github.event_name == 'workflow_dispatch' && github.event.inputs.operation != '' && github.event.inputs.operation != 'safe_outputs' && github.event.inputs.operation != 'create_labels' && github.event.inputs.operation != 'validate' && !github.event.repository.fork }}
    runs-on: ` + runsOnValue + `
    permissions:
      actions: write
      contents: write
      pull-requests: write
    steps:
      - name: Checkout repository
        uses: ` + GetActionPin("actions/checkout") + `
        with:
          persist-credentials: false

      - name: Setup Scripts
        uses: ` + setupActionRef + `
        with:
          destination: ${{ runner.temp }}/gh-aw/actions

      - name: Check admin/maintainer permissions
        uses: ` + GetActionPin("actions/github-script") + `
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/check_team_member.cjs');
            await main();

`)

	yaml.WriteString(generateInstallCLISteps(actionMode, version, actionTag, resolver))
	yaml.WriteString(`      - name: Run operation
        uses: ` + GetActionPin("actions/github-script") + `
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          GH_AW_OPERATION: ${{ github.event.inputs.operation }}
          GH_AW_CMD_PREFIX: ` + getCLICmdPrefix(actionMode) + `
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/run_operation_update_upgrade.cjs');
            await main();
`)

	// Add apply_safe_outputs job for workflow_dispatch with operation == 'safe_outputs'
	yaml.WriteString(`
  apply_safe_outputs:
    if: ${{ github.event_name == 'workflow_dispatch' && github.event.inputs.operation == 'safe_outputs' && !github.event.repository.fork }}
    runs-on: ` + runsOnValue + `
    permissions:
      actions: read
      contents: write
      discussions: write
      issues: write
      pull-requests: write
    steps:
      - name: Checkout actions folder
        uses: ` + GetActionPin("actions/checkout") + `
        with:
          sparse-checkout: |
            actions
          persist-credentials: false

      - name: Setup Scripts
        uses: ` + setupActionRef + `
        with:
          destination: ${{ runner.temp }}/gh-aw/actions

      - name: Check admin/maintainer permissions
        uses: ` + GetActionPin("actions/github-script") + `
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/check_team_member.cjs');
            await main();

      - name: Apply Safe Outputs
        uses: ` + GetActionPin("actions/github-script") + `
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          GH_AW_RUN_URL: ${{ github.event.inputs.run_url }}
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/apply_safe_outputs_replay.cjs');
            await main();
`)

	// Add create_labels job for workflow_dispatch with operation == 'create_labels'
	yaml.WriteString(`
  create_labels:
    if: ${{ github.event_name == 'workflow_dispatch' && github.event.inputs.operation == 'create_labels' && !github.event.repository.fork }}
    runs-on: ` + runsOnValue + `
    permissions:
      contents: read
      issues: write
    steps:
      - name: Checkout repository
        uses: ` + GetActionPin("actions/checkout") + `
        with:
          persist-credentials: false

      - name: Setup Scripts
        uses: ` + setupActionRef + `
        with:
          destination: ${{ runner.temp }}/gh-aw/actions

      - name: Check admin/maintainer permissions
        uses: ` + GetActionPin("actions/github-script") + `
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/check_team_member.cjs');
            await main();

`)

	yaml.WriteString(generateInstallCLISteps(actionMode, version, actionTag, resolver))
	yaml.WriteString(`      - name: Create missing labels
        uses: ` + GetActionPin("actions/github-script") + `
        env:
          GH_AW_CMD_PREFIX: ` + getCLICmdPrefix(actionMode) + `
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/create_labels.cjs');
            await main();
`)

	// Add validate_workflows job for workflow_dispatch with operation == 'validate'
	yaml.WriteString(`
  validate_workflows:
    if: ${{ github.event_name == 'workflow_dispatch' && github.event.inputs.operation == 'validate' && !github.event.repository.fork }}
    runs-on: ` + runsOnValue + `
    permissions:
      contents: read
      issues: write
    steps:
      - name: Checkout repository
        uses: ` + GetActionPin("actions/checkout") + `
        with:
          persist-credentials: false

      - name: Setup Scripts
        uses: ` + setupActionRef + `
        with:
          destination: ${{ runner.temp }}/gh-aw/actions

      - name: Check admin/maintainer permissions
        uses: ` + GetActionPin("actions/github-script") + `
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/check_team_member.cjs');
            await main();

`)

	yaml.WriteString(generateInstallCLISteps(actionMode, version, actionTag, resolver))

	yaml.WriteString(`      - name: Validate workflows and file issue on findings
        uses: ` + GetActionPin("actions/github-script") + `
        env:
          GH_AW_CMD_PREFIX: ` + getCLICmdPrefix(actionMode) + `
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/run_validate_workflows.cjs');
            await main();
`)

	// Add compile-workflows and zizmor-scan jobs only in dev mode
	// These jobs are specific to the gh-aw repository and require go.mod, make build, etc.
	// User repositories won't have these dependencies, so we skip them in release mode
	if actionMode == ActionModeDev {
		// Add compile-workflows job
		yaml.WriteString(`
  compile-workflows:
    if: ${{ !github.event.repository.fork && (github.event_name != 'workflow_dispatch' || github.event.inputs.operation == '') }}
    runs-on: ` + runsOnValue + `
    permissions:
      contents: read
      issues: write
    steps:
`)

		// Dev mode: checkout entire repository (no sparse checkout, but no credentials)
		yaml.WriteString(`      - name: Checkout repository
        uses: actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683 # v4.2.2
        with:
          persist-credentials: false

`)

		yaml.WriteString(generateInstallCLISteps(actionMode, version, actionTag, resolver))
		yaml.WriteString(`      - name: Compile workflows
        run: |
          ` + getCLICmdPrefix(actionMode) + ` compile --validate --verbose
          echo "✓ All workflows compiled successfully"

      - name: Setup Scripts
        uses: ` + setupActionRef + `
        with:
          destination: ${{ runner.temp }}/gh-aw/actions

      - name: Check for out-of-sync workflows and create issue if needed
        uses: ` + GetActionPin("actions/github-script") + `
        with:
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/check_workflow_recompile_needed.cjs');
            await main();

  zizmor-scan:
    if: ${{ !github.event.repository.fork && (github.event_name != 'workflow_dispatch' || github.event.inputs.operation == '') }}
    runs-on: ` + runsOnValue + `
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

  secret-validation:
    if: ${{ !github.event.repository.fork && (github.event_name != 'workflow_dispatch' || github.event.inputs.operation == '') }}
    runs-on: ` + runsOnValue + `
    permissions:
      contents: read
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

		yaml.WriteString(`      - name: Setup Node.js
        uses: actions/setup-node@39370e3970a6d050c480ffad4ff0ed4d3fdee5af # v4.1.0
        with:
          node-version: '22'

      - name: Setup Scripts
        uses: ` + setupActionRef + `
        with:
          destination: ${{ runner.temp }}/gh-aw/actions

      - name: Validate Secrets
        uses: ` + GetActionPin("actions/github-script") + `
        env:
          # GitHub tokens
          GH_AW_GITHUB_TOKEN: ${{ secrets.GH_AW_GITHUB_TOKEN }}
          GH_AW_GITHUB_MCP_SERVER_TOKEN: ${{ secrets.GH_AW_GITHUB_MCP_SERVER_TOKEN }}
          GH_AW_PROJECT_GITHUB_TOKEN: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}
          GH_AW_COPILOT_TOKEN: ${{ secrets.GH_AW_COPILOT_TOKEN }}
          # AI Engine API keys
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
          BRAVE_API_KEY: ${{ secrets.BRAVE_API_KEY }}
          # Integration tokens
          NOTION_API_TOKEN: ${{ secrets.NOTION_API_TOKEN }}
        with:
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/validate_secrets.cjs');
            await main();

      - name: Upload secret validation report
        if: always()
        uses: ` + GetActionPin("actions/upload-artifact") + `
        with:
          name: secret-validation-report
          path: secret-validation-report.md
          retention-days: 30
          if-no-files-found: warn
`)
	}

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
