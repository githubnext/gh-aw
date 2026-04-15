package workflow

import (
	"fmt"
	"hash/fnv"
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

// sideRepoCronSeed derives a deterministic 64-bit seed from a repository slug
// using FNV-1a hashing. The seed is used to scatter cron offsets across
// multiple side-repo maintenance workflows so they don't all fire at once.
func sideRepoCronSeed(repoSlug string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(repoSlug))
	return h.Sum64()
}

// generateSideRepoMaintenanceCron generates a scattered cron schedule for a
// side-repo maintenance workflow. The minute (and start hour for sub-daily
// schedules) are derived deterministically from the repository slug so that
// multiple side-repos are spread across the clock face rather than all firing
// at the same moment.
func generateSideRepoMaintenanceCron(repoSlug string, minExpiresDays int) (string, string) {
	seed := sideRepoCronSeed(repoSlug)
	// Derive a deterministic minute in 0-59 from the seed.
	minute := int(seed % 60)

	if minExpiresDays <= 1 {
		// Every 2 hours — vary the starting minute only.
		return fmt.Sprintf("%d */2 * * *", minute), "Every 2 hours"
	} else if minExpiresDays == 2 {
		// Every 6 hours — vary the starting hour within the 6-hour window.
		startHour := int((seed >> 8) % 6)
		return fmt.Sprintf("%d %d,%d,%d,%d * * *", minute, startHour, startHour+6, startHour+12, startHour+18), "Every 6 hours"
	} else if minExpiresDays <= 4 {
		// Every 12 hours — vary the starting hour within the 12-hour window.
		startHour := int((seed >> 8) % 12)
		return fmt.Sprintf("%d %d,%d * * *", minute, startHour, startHour+12), "Every 12 hours"
	}

	// Daily — vary the hour of day (0-23) to spread load.
	hour := int((seed >> 8) % 24)
	return fmt.Sprintf("%d %d * * *", minute, hour), "Daily"
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

	// Get the setup action reference (local or remote based on mode).
	// Use the first available WorkflowData's ActionResolver to enable SHA pinning.
	// Computed early so it is available in the !hasExpires path for side-repo workflows.
	var resolver ActionSHAResolver
	if len(workflowDataList) > 0 && workflowDataList[0].ActionResolver != nil {
		resolver = workflowDataList[0].ActionResolver
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

		// Even without expires, side-repo targets still need maintenance workflows
		// for safe_outputs, create_labels, and validate operations.
		return generateAllSideRepoMaintenanceWorkflows(workflowDataList, workflowDir, version, actionMode, actionTag, runsOnValue, resolver, false, 0)
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
          - 'clean_cache_memories'
          - 'validate'
      run_url:
        description: 'Run URL or run ID to replay safe outputs from (e.g. https://github.com/owner/repo/actions/runs/12345 or 12345). Required when operation is safe_outputs.'
        required: false
        type: string
        default: ''
  workflow_call:
    inputs:
      operation:
        description: 'Optional maintenance operation to run (disable, enable, update, upgrade, safe_outputs, create_labels, clean_cache_memories, validate)'
        required: false
        type: string
        default: ''
      run_url:
        description: 'Run URL or run ID to replay safe outputs from (e.g. https://github.com/owner/repo/actions/runs/12345 or 12345). Required when operation is safe_outputs.'
        required: false
        type: string
        default: ''
    outputs:
      operation_completed:
        description: 'The maintenance operation that was completed (empty when none ran or a scheduled job ran)'
        value: ${{ jobs.run_operation.outputs.operation || inputs.operation }}
      applied_run_url:
        description: 'The run URL that safe outputs were applied from'
        value: ${{ jobs.apply_safe_outputs.outputs.run_url }}

permissions: {}

jobs:
  close-expired-entities:
    if: ${{ ` + RenderCondition(buildNotForkAndScheduled()) + ` }}
    runs-on: ` + runsOnValue + `
    permissions:
      discussions: write
      issues: write
      pull-requests: write
    steps:
`)

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

	// Add cleanup-cache-memory job for scheduled runs and clean_cache_memories operation
	// This job lists all caches starting with "memory-", groups them by key prefix,
	// keeps the latest run ID per group, and deletes the rest.
	cleanupCacheCondition := buildNotForkAndScheduledOrOperation("clean_cache_memories")
	yaml.WriteString(`
  cleanup-cache-memory:
    if: ${{ ` + RenderCondition(cleanupCacheCondition) + ` }}
    runs-on: ` + runsOnValue + `
    permissions:
      actions: write
    steps:
`)

	// Add checkout step only in dev/script mode (for local action paths)
	if actionMode == ActionModeDev || actionMode == ActionModeScript {
		yaml.WriteString("      - name: Checkout actions folder\n")
		yaml.WriteString("        uses: " + GetActionPin("actions/checkout") + "\n")
		yaml.WriteString("        with:\n")
		yaml.WriteString("          sparse-checkout: |\n")
		yaml.WriteString("            actions\n")
		yaml.WriteString("          persist-credentials: false\n\n")
	}

	yaml.WriteString(`      - name: Setup Scripts
        uses: ` + setupActionRef + `
        with:
          destination: ${{ runner.temp }}/gh-aw/actions

      - name: Cleanup outdated cache-memory entries
        uses: ` + GetActionPin("actions/github-script") + `
        with:
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/cleanup_cache_memory.cjs');
            await main();
`)

	// Add unified run_operation job for all dispatch operations except those with dedicated jobs (safe_outputs, create_labels, clean_cache_memories, validate)
	runOperationCondition := buildRunOperationCondition("safe_outputs", "create_labels", "clean_cache_memories", "validate")
	yaml.WriteString(`
  run_operation:
    if: ${{ ` + RenderCondition(runOperationCondition) + ` }}
    runs-on: ` + runsOnValue + `
    permissions:
      actions: write
      contents: write
      pull-requests: write
    outputs:
      operation: ${{ steps.record.outputs.operation }}
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
          GH_AW_OPERATION: ${{ inputs.operation }}
          GH_AW_CMD_PREFIX: ` + getCLICmdPrefix(actionMode) + `
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/run_operation_update_upgrade.cjs');
            await main();

      - name: Record outputs
        id: record
        run: echo "operation=${{ inputs.operation }}" >> "$GITHUB_OUTPUT"
`)

	// Add apply_safe_outputs job for workflow_dispatch with operation == 'safe_outputs'
	yaml.WriteString(`
  apply_safe_outputs:
    if: ${{ ` + RenderCondition(buildDispatchOperationCondition("safe_outputs")) + ` }}
    runs-on: ` + runsOnValue + `
    permissions:
      actions: read
      contents: write
      discussions: write
      issues: write
      pull-requests: write
    outputs:
      run_url: ${{ steps.record.outputs.run_url }}
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
          GH_AW_RUN_URL: ${{ inputs.run_url }}
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/apply_safe_outputs_replay.cjs');
            await main();

      - name: Record outputs
        id: record
        run: echo "run_url=${{ inputs.run_url }}" >> "$GITHUB_OUTPUT"
`)

	// Add create_labels job for workflow_dispatch with operation == 'create_labels'
	yaml.WriteString(`
  create_labels:
    if: ${{ ` + RenderCondition(buildDispatchOperationCondition("create_labels")) + ` }}
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
	// This job uses ubuntu-latest by default (needs full runner for CLI installation).
	validateRunsOnValue := FormatRunsOn(configuredRunsOn, "ubuntu-latest")
	yaml.WriteString(`
  validate_workflows:
    if: ${{ ` + RenderCondition(buildDispatchOperationCondition("validate")) + ` }}
    runs-on: ` + validateRunsOnValue + `
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
    if: ${{ ` + RenderCondition(buildNotForkAndScheduled()) + ` }}
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
          ` + getCLICmdPrefix(actionMode) + ` compile --validate --validate-images --verbose
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

  secret-validation:
    if: ${{ ` + RenderCondition(buildNotForkAndScheduled()) + ` }}
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

	// Generate side-repo maintenance workflows for any SideRepoOps targets detected.
	if err := generateAllSideRepoMaintenanceWorkflows(workflowDataList, workflowDir, version, actionMode, actionTag, runsOnValue, resolver, hasExpires, minExpiresDays); err != nil {
		return err
	}

	return nil
}

// SideRepoTarget represents a target repository inferred from a checkout block
// with current: true in a compiled workflow. It is used to generate a
// side-repo-specific agentics-maintenance workflow.
type SideRepoTarget struct {
	// Repository is the static owner/repo slug of the target (e.g. "my-org/main-repo").
	// Expression-based repositories (containing "${{") are excluded.
	Repository string

	// GitHubToken is the token expression used to authenticate against the target
	// repository, e.g. "${{ secrets.GH_AW_MAIN_REPO_TOKEN }}". Empty when the
	// checkout config does not specify a custom token.
	GitHubToken string
}

// collectSideRepoTargets scans all compiled workflow data and returns the unique
// SideRepoTarget entries inferred from checkout blocks with current: true.
// Only checkouts with a static (non-expression) repository string are included.
// When the same repository appears multiple times, a non-empty GitHubToken is
// preferred over an empty one so that the generated workflow uses the custom
// token rather than falling back to GH_AW_GITHUB_TOKEN.
func collectSideRepoTargets(workflowDataList []*WorkflowData) []SideRepoTarget {
	// Use a map to accumulate the best token seen for each slug.
	// Order slice preserves first-seen repository discovery order for stable output;
	// tokens may be upgraded to non-empty values from later occurrences.
	tokenByRepo := make(map[string]string)
	var order []string
	for _, wd := range workflowDataList {
		for _, checkout := range wd.CheckoutConfigs {
			if !checkout.Current {
				continue
			}
			repo := checkout.Repository
			if repo == "" || strings.Contains(repo, "${{") {
				// Skip empty repositories and expression-based (dynamic) ones.
				continue
			}
			existing, seen := tokenByRepo[repo]
			if !seen {
				order = append(order, repo)
				tokenByRepo[repo] = checkout.GitHubToken
			} else if existing == "" && checkout.GitHubToken != "" {
				// Upgrade to a non-empty token when one is encountered later.
				tokenByRepo[repo] = checkout.GitHubToken
			}
		}
	}
	targets := make([]SideRepoTarget, 0, len(order))
	for _, repo := range order {
		targets = append(targets, SideRepoTarget{
			Repository:  repo,
			GitHubToken: tokenByRepo[repo],
		})
	}
	maintenanceLog.Printf("Detected %d side-repo target(s) from checkout configs", len(targets))
	return targets
}

// sanitizeRepoForFilename converts an "owner/repo" slug into a string safe for
// use as part of a filename, replacing "/" with "-" and any remaining
// non-alphanumeric characters (except "-", "_", ".") with "-".
func sanitizeRepoForFilename(repo string) string {
	var sb strings.Builder
	for _, r := range strings.ReplaceAll(repo, "/", "-") {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			sb.WriteRune(r)
		} else {
			sb.WriteRune('-')
		}
	}
	return sb.String()
}

// effectiveSideRepoToken returns the GitHub token expression to use for the
// side-repo maintenance workflow. It prefers the token from the checkout config;
// when none is set it falls back to a conventional secret name.
func effectiveSideRepoToken(checkout SideRepoTarget) string {
	if checkout.GitHubToken != "" {
		return checkout.GitHubToken
	}
	return "${{ secrets.GH_AW_GITHUB_TOKEN }}"
}

// generateAllSideRepoMaintenanceWorkflows detects SideRepoOps targets and
// generates a per-target maintenance workflow for each unique static repository.
func generateAllSideRepoMaintenanceWorkflows(
	workflowDataList []*WorkflowData,
	workflowDir string,
	version string,
	actionMode ActionMode,
	actionTag string,
	runsOnValue string,
	resolver ActionSHAResolver,
	hasExpires bool,
	minExpiresDays int,
) error {
	targets := collectSideRepoTargets(workflowDataList)

	// Track which side-repo maintenance files we (re-)generate so we can identify
	// and remove stale files from previous runs when target repos are renamed or removed.
	generatedFiles := make(map[string]bool)

	for _, target := range targets {
		slug := sanitizeRepoForFilename(target.Repository)
		filename := "agentics-maintenance-" + slug + ".yml"
		generatedFiles[filename] = true
		outPath := filepath.Join(workflowDir, filename)

		maintenanceLog.Printf("Generating side-repo maintenance workflow: %s → %s", target.Repository, filename)
		if err := generateSideRepoMaintenanceWorkflow(target, outPath, version, actionMode, actionTag, runsOnValue, resolver, hasExpires, minExpiresDays); err != nil {
			return fmt.Errorf("failed to generate side-repo maintenance workflow for %s: %w", target.Repository, err)
		}
		fmt.Fprintf(os.Stderr, "  Generated side-repo maintenance workflow: %s\n", filename)
	}

	// Remove stale side-repo maintenance workflows that are no longer referenced.
	entries, err := os.ReadDir(workflowDir)
	if err != nil {
		return fmt.Errorf("failed to read workflow directory %s for stale side-repo maintenance workflow cleanup: %w", workflowDir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "agentics-maintenance-") || !strings.HasSuffix(name, ".yml") {
			continue
		}
		if generatedFiles[name] {
			continue
		}
		stalePath := filepath.Join(workflowDir, name)
		maintenanceLog.Printf("Removing stale side-repo maintenance workflow: %s", name)
		if err := os.Remove(stalePath); err != nil {
			return fmt.Errorf("failed to remove stale side-repo maintenance workflow %s: %w", stalePath, err)
		}
		fmt.Fprintf(os.Stderr, "  Removed stale side-repo maintenance workflow: %s\n", name)
	}

	return nil
}

// generateSideRepoMaintenanceWorkflow generates a workflow_call-based maintenance
// workflow that targets an external repository detected via the SideRepoOps pattern.
// The generated workflow mirrors agentics-maintenance.yml but authenticates against
// the target repository using the token from the checkout config and sets
// GH_AW_TARGET_REPO_SLUG for all cross-repo operations.
func generateSideRepoMaintenanceWorkflow(
	target SideRepoTarget,
	outPath string,
	version string,
	actionMode ActionMode,
	actionTag string,
	runsOnValue string,
	resolver ActionSHAResolver,
	hasExpires bool,
	minExpiresDays int,
) error {
	token := effectiveSideRepoToken(target)
	repoSlug := target.Repository

	var yaml strings.Builder

	customInstructions := `Alternative regeneration methods:
  make recompile

Or use the gh-aw CLI directly:
  ./gh-aw compile --validate --verbose

This workflow is generated for the SideRepoOps target repository "` + repoSlug + `".
It can be triggered via workflow_dispatch or called via workflow_call to run maintenance
operations (safe-outputs replay, label creation, validation, expired-entity cleanup)
against the target repository.`

	header := GenerateWorkflowHeader("", "pkg/workflow/maintenance_workflow.go", customInstructions)
	yaml.WriteString(header)

	// Pre-compute cron schedule values (needed in both the on: section and the
	// close-expired-entities job comment when hasExpires is true).
	// Uses fuzzy scheduling: minute and hour offsets are derived from the repo
	// slug hash so that multiple side-repo workflows are scattered across the
	// clock face instead of all firing at the same time.
	var cronSchedule, scheduleDesc string
	if hasExpires {
		effectiveDays := minExpiresDays
		if effectiveDays == 0 {
			// minExpiresDays == 0 means expiry < 1 day; use a conservative daily default.
			effectiveDays = 5
		}
		cronSchedule, scheduleDesc = generateSideRepoMaintenanceCron(repoSlug, effectiveDays)
	}

	// Build the `on:` triggers. A schedule trigger is added when at least one
	// workflow uses `expires`, because the close-expired-entities job's condition
	// (`buildNotForkAndScheduled`) also matches scheduled runs.
	onSection := `name: Agentic Maintenance (` + repoSlug + `)

on:
  workflow_dispatch:
    inputs:
      operation:
        description: 'Optional maintenance operation to run'
        required: false
        type: choice
        default: ''
        options:
          - ''
          - 'safe_outputs'
          - 'create_labels'
          - 'validate'
      run_url:
        description: 'Run URL or run ID to replay safe outputs from (e.g. https://github.com/owner/repo/actions/runs/12345 or 12345). Required when operation is safe_outputs.'
        required: false
        type: string
        default: ''
  workflow_call:
    inputs:
      operation:
        description: 'Optional maintenance operation to run (safe_outputs, create_labels, validate)'
        required: false
        type: string
        default: ''
      run_url:
        description: 'Run URL or run ID to replay safe outputs from (e.g. https://github.com/owner/repo/actions/runs/12345 or 12345). Required when operation is safe_outputs.'
        required: false
        type: string
        default: ''
    outputs:
      applied_run_url:
        description: 'The run URL that safe outputs were applied from'
        value: ${{ jobs.apply_safe_outputs.outputs.run_url }}
`
	if hasExpires {
		onSection += `  schedule:
    - cron: "` + cronSchedule + `"  # ` + scheduleDesc + ` (based on minimum expires: ` + strconv.Itoa(minExpiresDays) + ` days)
`
	}
	onSection += `
permissions: {}

jobs:
`
	yaml.WriteString(onSection)

	setupActionRef := ResolveSetupActionReference(actionMode, version, actionTag, resolver)

	// Add close-expired-entities job only when any workflow uses expires.
	if hasExpires {
		closeExpiredCondition := buildNotForkAndScheduled()
		yaml.WriteString(`  close-expired-entities:
    if: ${{ ` + RenderCondition(closeExpiredCondition) + ` }}
    runs-on: ` + runsOnValue + `
    permissions:
      discussions: write
      issues: write
      pull-requests: write
    # Runs on schedule: ` + cronSchedule + ` (` + scheduleDesc + `)
    steps:
`)

		if actionMode == ActionModeDev || actionMode == ActionModeScript {
			yaml.WriteString("      - name: Checkout actions folder\n")
			yaml.WriteString("        uses: " + GetActionPin("actions/checkout") + "\n")
			yaml.WriteString("        with:\n")
			yaml.WriteString("          sparse-checkout: |\n")
			yaml.WriteString("            actions\n")
			yaml.WriteString("          persist-credentials: false\n\n")
		}

		yaml.WriteString(`      - name: Setup Scripts
        uses: ` + setupActionRef + `
        with:
          destination: ${{ runner.temp }}/gh-aw/actions

      - name: Close expired discussions
        uses: ` + GetActionPin("actions/github-script") + `
        env:
          GH_AW_TARGET_REPO_SLUG: "` + repoSlug + `"
        with:
          github-token: ` + token + `
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/close_expired_discussions.cjs');
            await main();

      - name: Close expired issues
        uses: ` + GetActionPin("actions/github-script") + `
        env:
          GH_AW_TARGET_REPO_SLUG: "` + repoSlug + `"
        with:
          github-token: ` + token + `
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/close_expired_issues.cjs');
            await main();

      - name: Close expired pull requests
        uses: ` + GetActionPin("actions/github-script") + `
        env:
          GH_AW_TARGET_REPO_SLUG: "` + repoSlug + `"
        with:
          github-token: ` + token + `
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/close_expired_pull_requests.cjs');
            await main();
`)
	}

	// Add apply_safe_outputs job for workflow_dispatch/workflow_call with operation == 'safe_outputs'
	yaml.WriteString(`
  apply_safe_outputs:
    if: ${{ ` + RenderCondition(buildDispatchOperationCondition("safe_outputs")) + ` }}
    runs-on: ` + runsOnValue + `
    permissions:
      actions: read
      contents: write
      discussions: write
      issues: write
      pull-requests: write
    outputs:
      run_url: ${{ steps.record.outputs.run_url }}
    steps:
`)

	if actionMode == ActionModeDev || actionMode == ActionModeScript {
		yaml.WriteString("      - name: Checkout actions folder\n")
		yaml.WriteString("        uses: " + GetActionPin("actions/checkout") + "\n")
		yaml.WriteString("        with:\n")
		yaml.WriteString("          sparse-checkout: |\n")
		yaml.WriteString("            actions\n")
		yaml.WriteString("          persist-credentials: false\n\n")
	}

	yaml.WriteString(`      - name: Setup Scripts
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
          GH_TOKEN: ` + token + `
          GH_AW_RUN_URL: ${{ inputs.run_url }}
          GH_AW_TARGET_REPO_SLUG: "` + repoSlug + `"
        with:
          github-token: ` + token + `
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/apply_safe_outputs_replay.cjs');
            await main();

      - name: Record outputs
        id: record
        run: echo "run_url=${{ inputs.run_url }}" >> "$GITHUB_OUTPUT"
`)

	// Add create_labels job for workflow_dispatch/workflow_call with operation == 'create_labels'
	yaml.WriteString(`
  create_labels:
    if: ${{ ` + RenderCondition(buildDispatchOperationCondition("create_labels")) + ` }}
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
	yaml.WriteString(`      - name: Create missing labels in target repository
        uses: ` + GetActionPin("actions/github-script") + `
        env:
          GH_AW_CMD_PREFIX: ` + getCLICmdPrefix(actionMode) + `
          GH_AW_TARGET_REPO_SLUG: "` + repoSlug + `"
        with:
          github-token: ` + token + `
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/create_labels.cjs');
            await main();
`)

	// Add validate_workflows job for workflow_dispatch/workflow_call with operation == 'validate'
	validateRunsOnValue := FormatRunsOn(nil, "ubuntu-latest")
	yaml.WriteString(`
  validate_workflows:
    if: ${{ ` + RenderCondition(buildDispatchOperationCondition("validate")) + ` }}
    runs-on: ` + validateRunsOnValue + `
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

	content := yaml.String()
	maintenanceLog.Printf("Writing side-repo maintenance workflow to %s", outPath)
	if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write side-repo maintenance workflow: %w", err)
	}
	return nil
}

// buildNotForkCondition creates a condition to check the repository is not a fork.
func buildNotForkCondition() ConditionNode {
	return &NotNode{
		Child: BuildPropertyAccess("github.event.repository.fork"),
	}
}

// buildNotDispatchOrCallOrEmptyOperation creates a condition that is true when the event
// is not a workflow_dispatch or workflow_call, or the operation input is empty.
// Uses the `inputs.operation` context which works for both workflow_dispatch and workflow_call.
func buildNotDispatchOrCallOrEmptyOperation() ConditionNode {
	return BuildOr(
		BuildAnd(
			BuildNotEquals(
				BuildPropertyAccess("github.event_name"),
				BuildStringLiteral("workflow_dispatch"),
			),
			BuildNotEquals(
				BuildPropertyAccess("github.event_name"),
				BuildStringLiteral("workflow_call"),
			),
		),
		BuildEquals(
			BuildPropertyAccess("inputs.operation"),
			BuildStringLiteral(""),
		),
	)
}

// buildNotForkAndScheduledOrOperation creates a condition for jobs that run on
// schedule (or empty operation) AND when a specific operation is selected.
// Condition: !fork && (not_dispatch_or_call || operation == \'\' || operation == op)
func buildNotForkAndScheduledOrOperation(operation string) ConditionNode {
	return BuildAnd(
		buildNotForkCondition(),
		BuildOr(
			buildNotDispatchOrCallOrEmptyOperation(),
			BuildEquals(
				BuildPropertyAccess("inputs.operation"),
				BuildStringLiteral(operation),
			),
		),
	)
}

// buildNotForkAndScheduled creates a condition for jobs that should run on any
// non-dispatch/call event (e.g. schedule, push) or on workflow_dispatch/workflow_call
// with an empty operation, and never on forks.
// Condition: !fork && ((event_name != \'workflow_dispatch\' && event_name != \'workflow_call\') || operation == \'\')
func buildNotForkAndScheduled() ConditionNode {
	return BuildAnd(
		buildNotForkCondition(),
		buildNotDispatchOrCallOrEmptyOperation(),
	)
}

// buildDispatchOperationCondition creates a condition for jobs that should run
// only when a specific workflow_dispatch or workflow_call operation is selected and not a fork.
// Condition: (dispatch || call) && operation == op && !fork
func buildDispatchOperationCondition(operation string) ConditionNode {
	return BuildAnd(
		BuildAnd(
			BuildOr(
				BuildEventTypeEquals("workflow_dispatch"),
				BuildEventTypeEquals("workflow_call"),
			),
			BuildEquals(
				BuildPropertyAccess("inputs.operation"),
				BuildStringLiteral(operation),
			),
		),
		buildNotForkCondition(),
	)
}

// buildRunOperationCondition creates the condition for the unified run_operation
// job that handles all dispatch/call operations except the ones with dedicated jobs.
// Condition: (dispatch || call) && operation != \'\' && operation != each excluded && !fork.
func buildRunOperationCondition(excludedOperations ...string) ConditionNode {
	// Start with: event is workflow_dispatch or workflow_call AND operation is not empty
	condition := BuildAnd(
		BuildOr(
			BuildEventTypeEquals("workflow_dispatch"),
			BuildEventTypeEquals("workflow_call"),
		),
		BuildNotEquals(
			BuildPropertyAccess("inputs.operation"),
			BuildStringLiteral(""),
		),
	)

	// Exclude each dedicated operation
	for _, op := range excludedOperations {
		condition = BuildAnd(
			condition,
			BuildNotEquals(
				BuildPropertyAccess("inputs.operation"),
				BuildStringLiteral(op),
			),
		)
	}

	// AND not a fork
	return BuildAnd(condition, buildNotForkCondition())
}
