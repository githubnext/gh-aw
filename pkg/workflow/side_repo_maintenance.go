package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

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

	header := GenerateWorkflowHeader("", "pkg/workflow/side_repo_maintenance.go", customInstructions)
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
			yaml.WriteString("        uses: " + getActionPin("actions/checkout") + "\n")
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
        uses: ` + getCachedActionPinFromResolver("actions/github-script", resolver) + `
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
        uses: ` + getCachedActionPinFromResolver("actions/github-script", resolver) + `
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
        uses: ` + getCachedActionPinFromResolver("actions/github-script", resolver) + `
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
		yaml.WriteString("        uses: " + getActionPin("actions/checkout") + "\n")
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
        uses: ` + getCachedActionPinFromResolver("actions/github-script", resolver) + `
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/check_team_member.cjs');
            await main();

      - name: Apply Safe Outputs
        uses: ` + getCachedActionPinFromResolver("actions/github-script", resolver) + `
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
        uses: ` + getActionPin("actions/checkout") + `
        with:
          persist-credentials: false

      - name: Setup Scripts
        uses: ` + setupActionRef + `
        with:
          destination: ${{ runner.temp }}/gh-aw/actions

      - name: Check admin/maintainer permissions
        uses: ` + getCachedActionPinFromResolver("actions/github-script", resolver) + `
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
        uses: ` + getCachedActionPinFromResolver("actions/github-script", resolver) + `
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
        uses: ` + getActionPin("actions/checkout") + `
        with:
          persist-credentials: false

      - name: Setup Scripts
        uses: ` + setupActionRef + `
        with:
          destination: ${{ runner.temp }}/gh-aw/actions

      - name: Check admin/maintainer permissions
        uses: ` + getCachedActionPinFromResolver("actions/github-script", resolver) + `
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
        uses: ` + getCachedActionPinFromResolver("actions/github-script", resolver) + `
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
