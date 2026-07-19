package workflow

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/setutil"

	"github.com/github/gh-aw/pkg/stringutil"
)

//go:embed assets/side_repo_maintenance_header.md
var sideRepoMaintenanceHeaderTemplate string

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
	// Mutually exclusive with GitHubApp.
	GitHubToken string

	// GitHubApp carries the GitHub App authentication config discovered from the
	// source checkout. When set, each cross-repo maintenance job gets a
	// create-github-app-token mint step and the minted token is used for all
	// github-token: inputs and GH_TOKEN: env vars.
	// Mutually exclusive with GitHubToken.
	GitHubApp *GitHubAppConfig
}

// sideRepoAppTokenStepID is the step ID used for the GitHub App token mint step
// emitted in each cross-repo maintenance job.
const sideRepoAppTokenStepID = "side-repo-app-token"

// sideRepoAppTokenRef is the GitHub Actions expression that references the minted
// token output from the sideRepoAppTokenStepID step.
const sideRepoAppTokenRef = "${{ steps." + sideRepoAppTokenStepID + ".outputs.token }}"

// sideRepoAuth accumulates authentication configuration for a single side-repo target.
// GitHubToken and GitHubApp are mutually exclusive (matching CheckoutConfig).
type sideRepoAuth struct {
	token     string
	githubApp *GitHubAppConfig
}

// collectSideRepoTargets scans all compiled workflow data and returns the unique
// SideRepoTarget entries inferred from checkout blocks with current: true.
// Only checkouts with a static (non-expression) repository string are included.
// When the same repository appears multiple times, a non-empty GitHubToken or a
// non-nil GitHubApp is preferred over an empty auth so that the generated workflow
// uses the custom token rather than falling back to GH_AW_GITHUB_TOKEN.
// The first-seen auth for a given repo is preserved; later occurrences only
// upgrade from "no auth" → "has auth" and never replace an existing auth choice.
func collectSideRepoTargets(workflowDataList []*WorkflowData) []SideRepoTarget {
	maintenanceLog.Printf("Scanning %d workflows for side-repo targets", len(workflowDataList))
	// Use a map to accumulate the best auth seen for each slug.
	// Order slice preserves first-seen repository discovery order for stable output;
	// auth may be upgraded from empty to a non-empty value from later occurrences.
	authByRepo := make(map[string]sideRepoAuth)
	var order []string
	for _, wd := range workflowDataList {
		if wd == nil {
			continue
		}
		for _, checkout := range wd.CheckoutConfigs {
			if !checkout.Current {
				continue
			}
			repo := checkout.Repository
			if repo == "" || strings.Contains(repo, "${{") {
				// Skip empty repositories and expression-based (dynamic) ones.
				continue
			}
			existing, seen := authByRepo[repo]
			if !seen {
				order = append(order, repo)
				authByRepo[repo] = sideRepoAuth{
					token:     checkout.GitHubToken,
					githubApp: checkout.GitHubApp,
				}
			} else if existing.token == "" && existing.githubApp == nil {
				// Upgrade from no-auth to any-auth from a later occurrence.
				if checkout.GitHubToken != "" || checkout.GitHubApp != nil {
					authByRepo[repo] = sideRepoAuth{
						token:     checkout.GitHubToken,
						githubApp: checkout.GitHubApp,
					}
				}
			} else if checkout.GitHubToken != "" || checkout.GitHubApp != nil {
				// A later occurrence provides auth, but an earlier one already set auth.
				// First-seen auth wins; log a notice so users can diagnose unexpected choices.
				maintenanceLog.Printf("Ignoring later auth for %s: first-seen auth (token=%t, app=%t) already recorded",
					repo, existing.token != "", existing.githubApp != nil)
			}
		}
	}
	targets := make([]SideRepoTarget, 0, len(order))
	for _, repo := range order {
		auth := authByRepo[repo]
		targets = append(targets, SideRepoTarget{
			Repository:  repo,
			GitHubToken: auth.token,
			GitHubApp:   auth.githubApp,
		})
	}
	maintenanceLog.Printf("Detected %d side-repo target(s) from checkout configs", len(targets))
	return targets
}

// effectiveSideRepoToken returns the GitHub token expression to use for the
// side-repo maintenance workflow. It prefers the token from the checkout config;
// when a GitHub App is configured it returns the minted token reference; when
// neither is set it falls back to a conventional secret name.
func effectiveSideRepoToken(checkout SideRepoTarget) string {
	if checkout.GitHubToken != "" && checkout.GitHubApp != nil {
		maintenanceLog.Printf("SideRepoTarget %s has both GitHubToken and GitHubApp configured; using explicit GitHubToken", checkout.Repository)
	}
	if checkout.GitHubToken != "" {
		return checkout.GitHubToken
	}
	if checkout.GitHubApp != nil {
		return sideRepoAppTokenRef
	}
	return "${{ secrets.GH_AW_GITHUB_TOKEN }}"
}

// sideRepoAppTokenMintStepYAML generates the YAML snippet for a
// create-github-app-token step to be inserted at the top of each cross-repo
// maintenance job. The step ID is sideRepoAppTokenStepID so the minted token is
// referenced via sideRepoAppTokenRef by subsequent steps in the same job.
func sideRepoAppTokenMintStepYAML(app *GitHubAppConfig, targetRepo string) string {
	var c Compiler
	lines := c.buildGitHubAppTokenMintStepWithMeta(
		app,
		nil, // no additional permission scoping; the app's installation grants determine access
		targetRepo,
		targetRepo,
		"Generate GitHub App token",
		sideRepoAppTokenStepID,
	)
	return strings.Join(lines, "")
}

// generateAllSideRepoMaintenanceWorkflowsOptions configures side-repo maintenance workflow generation.
type generateAllSideRepoMaintenanceWorkflowsOptions struct {
	workflowDataList []*WorkflowData
	workflowDir      string
	version          string
	actionMode       ActionMode
	actionTag        string
	runsOnValue      string
	resolver         SHAResolver
	hasExpires       bool
	minExpiresDays   int
}

// generateAllSideRepoMaintenanceWorkflows detects SideRepoOps targets and
// generates a per-target maintenance workflow for each unique static repository.
func generateAllSideRepoMaintenanceWorkflows(
	ctx context.Context,
	opts generateAllSideRepoMaintenanceWorkflowsOptions,
) error {
	workflowDataList := opts.workflowDataList
	workflowDir := opts.workflowDir
	hasExpires := opts.hasExpires
	minExpiresDays := opts.minExpiresDays
	targets := collectSideRepoTargets(workflowDataList)
	maintenanceLog.Printf("Generating maintenance workflows for %d side-repo target(s): hasExpires=%t, minExpiresDays=%d", len(targets), hasExpires, minExpiresDays)

	// Track which side-repo maintenance files we (re-)generate so we can identify
	// and remove stale files from previous runs when target repos are renamed or removed.
	generatedFiles := make(map[string]struct{})

	for _, target := range targets {
		filename, err := generateSideRepoMaintenanceForTarget(ctx, target, opts)
		if err != nil {
			return fmt.Errorf("failed to generate side-repo maintenance workflow for %s: %w", target.Repository, err)
		}
		generatedFiles[filename] = struct{}{}
		fmt.Fprintf(os.Stderr, "  Generated side-repo maintenance workflow: %s\n", filename)
	}

	// Remove stale side-repo maintenance workflows that are no longer referenced.
	return removeStaleSideRepoMaintenanceWorkflows(workflowDir, generatedFiles)
}

func generateSideRepoMaintenanceForTarget(ctx context.Context, target SideRepoTarget, opts generateAllSideRepoMaintenanceWorkflowsOptions) (string, error) {
	slug := stringutil.SanitizeForFilename(target.Repository)
	filename := "agentics-maintenance-" + slug + ".yml"
	outPath := filepath.Join(opts.workflowDir, filename)

	maintenanceLog.Printf("Generating side-repo maintenance workflow: %s → %s", target.Repository, filename)
	err := generateSideRepoMaintenanceWorkflow(ctx, generateSideRepoMaintenanceWorkflowOptions{
		target:         target,
		outPath:        outPath,
		version:        opts.version,
		actionMode:     opts.actionMode,
		actionTag:      opts.actionTag,
		runsOnValue:    opts.runsOnValue,
		resolver:       opts.resolver,
		hasExpires:     opts.hasExpires,
		minExpiresDays: opts.minExpiresDays,
	})
	return filename, err
}

func removeStaleSideRepoMaintenanceWorkflows(workflowDir string, generatedFiles map[string]struct{}) error {
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
		if setutil.Contains(generatedFiles, name) {
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

// generateSideRepoMaintenanceWorkflowOptions configures generation of a single side-repo
// maintenance workflow.
type generateSideRepoMaintenanceWorkflowOptions struct {
	target         SideRepoTarget
	outPath        string
	version        string
	actionMode     ActionMode
	actionTag      string
	runsOnValue    string
	resolver       SHAResolver
	hasExpires     bool
	minExpiresDays int
}

// generateSideRepoMaintenanceWorkflow generates a workflow_call-based maintenance
// workflow that targets an external repository detected via the SideRepoOps pattern.
// The generated workflow mirrors agentics-maintenance.yml but authenticates against
// the target repository using the token from the checkout config and sets
// GH_AW_TARGET_REPO_SLUG for all cross-repo operations.
func generateSideRepoMaintenanceWorkflow(
	ctx context.Context,
	opts generateSideRepoMaintenanceWorkflowOptions,
) error {
	build := newSideRepoMaintenanceBuild(ctx, opts)
	maintenanceLog.Printf("Building side-repo workflow content: repo=%s, actionMode=%s, hasExpires=%t", build.repoSlug, build.actionMode, build.hasExpires)

	var yaml strings.Builder
	yaml.WriteString(build.header())
	yaml.WriteString(build.onSection())
	if build.hasExpires {
		yaml.WriteString(build.closeExpiredJob())
	}
	yaml.WriteString(build.applySafeOutputsJob())
	yaml.WriteString(build.createLabelsJob())
	yaml.WriteString(build.activityReportJob())
	yaml.WriteString(build.validateWorkflowsJob())

	content := yaml.String()
	maintenanceLog.Printf("Writing side-repo maintenance workflow to %s", build.outPath)
	if err := os.WriteFile(build.outPath, []byte(content), constants.FilePermPublic); err != nil {
		return fmt.Errorf("failed to write side-repo maintenance workflow: %w", err)
	}
	return nil
}

type sideRepoMaintenanceBuild struct {
	ctx            context.Context
	target         SideRepoTarget
	outPath        string
	version        string
	actionMode     ActionMode
	actionTag      string
	runsOnValue    string
	resolver       SHAResolver
	hasExpires     bool
	minExpiresDays int
	token          string
	repoSlug       string
	mintStepYAML   string
	setupActionRef string
	cronSchedule   string
	scheduleDesc   string
}

func newSideRepoMaintenanceBuild(ctx context.Context, opts generateSideRepoMaintenanceWorkflowOptions) sideRepoMaintenanceBuild {
	build := sideRepoMaintenanceBuild{
		ctx: ctx, target: opts.target, outPath: opts.outPath, version: opts.version,
		actionMode: opts.actionMode, actionTag: opts.actionTag, runsOnValue: opts.runsOnValue,
		resolver: opts.resolver, hasExpires: opts.hasExpires, minExpiresDays: opts.minExpiresDays,
		token: effectiveSideRepoToken(opts.target), repoSlug: opts.target.Repository,
	}
	build.setupActionRef = ResolveSetupActionReference(ctx, build.actionMode, build.version, build.actionTag, build.resolver)
	build.mintStepYAML = build.sideRepoMintStepYAML()
	build.cronSchedule, build.scheduleDesc = build.sideRepoCron()
	return build
}

func (b sideRepoMaintenanceBuild) sideRepoMintStepYAML() string {
	if b.target.GitHubApp != nil && b.target.GitHubToken == "" {
		maintenanceLog.Printf("GitHub App auth configured for %s; will emit mint step in cross-repo jobs", b.repoSlug)
		return sideRepoAppTokenMintStepYAML(b.target.GitHubApp, b.target.Repository)
	}
	if b.target.GitHubApp != nil {
		maintenanceLog.Printf("SideRepoTarget %s has both GitHubToken and GitHubApp configured; skipping app token mint step", b.repoSlug)
	}
	return ""
}

func (b sideRepoMaintenanceBuild) sideRepoCron() (string, string) {
	if !b.hasExpires {
		return "", ""
	}
	effectiveDays := b.minExpiresDays
	if effectiveDays == 0 {
		effectiveDays = 5
	}
	return generateSideRepoMaintenanceCron(b.repoSlug, effectiveDays)
}

func (b sideRepoMaintenanceBuild) header() string {
	customInstructions := strings.ReplaceAll(sideRepoMaintenanceHeaderTemplate, "{REPO_SLUG}", b.repoSlug)
	return GenerateWorkflowHeader("", "pkg/workflow/side_repo_maintenance.go", customInstructions)
}

func (b sideRepoMaintenanceBuild) onSection() string {
	onSection := strings.ReplaceAll(sideRepoMaintenanceOnTemplate, "@@REPO_SLUG@@", b.repoSlug)
	if b.hasExpires {
		onSection += strings.NewReplacer(
			"@@CRON_SCHEDULE@@", b.cronSchedule,
			"@@SCHEDULE_DESC@@", b.scheduleDesc,
			"@@MIN_EXPIRES_DAYS@@", strconv.Itoa(b.minExpiresDays),
		).Replace(sideRepoMaintenanceScheduleTemplate)
	}
	return onSection + sideRepoMaintenancePermissionsAndJobs
}

func (b sideRepoMaintenanceBuild) closeExpiredJob() string {
	maintenanceLog.Printf("Including close-expired-entities job for %s (cron=%s)", b.repoSlug, b.cronSchedule)
	return b.replaceCommon(sideRepoMaintenanceCloseExpiredTemplate,
		"@@CONDITION@@", RenderCondition(buildNotForkAndScheduled()),
		"@@CHECKOUT_ACTIONS_STEP@@", b.actionsCheckoutStep(),
		"@@CRON_SCHEDULE@@", b.cronSchedule,
		"@@SCHEDULE_DESC@@", b.scheduleDesc,
	)
}

func (b sideRepoMaintenanceBuild) applySafeOutputsJob() string {
	return b.replaceCommon(sideRepoMaintenanceApplySafeOutputsTemplate,
		"@@CONDITION@@", RenderCondition(buildDispatchOperationCondition("safe_outputs")),
		"@@CHECKOUT_ACTIONS_STEP@@", b.actionsCheckoutStep(),
	)
}

func (b sideRepoMaintenanceBuild) createLabelsJob() string {
	return b.replaceCommon(sideRepoMaintenanceCreateLabelsTemplate,
		"@@CONDITION@@", RenderCondition(buildDispatchOperationCondition("create_labels")),
		"@@INSTALL_CLI_STEPS@@", generateInstallCLISteps(b.ctx, b.actionMode, b.version, b.actionTag, b.resolver),
		"@@CLI_CMD_PREFIX@@", getCLICmdPrefix(b.actionMode),
	)
}

func (b sideRepoMaintenanceBuild) activityReportJob() string {
	return b.replaceCommon(sideRepoMaintenanceActivityReportTemplate,
		"@@CONDITION@@", RenderCondition(buildDispatchOperationCondition("activity_report")),
		"@@INSTALL_CLI_STEPS@@", generateInstallCLISteps(b.ctx, b.actionMode, b.version, b.actionTag, b.resolver),
		"@@CLI_CMD_PREFIX@@", getCLICmdPrefix(b.actionMode),
	)
}

func (b sideRepoMaintenanceBuild) validateWorkflowsJob() string {
	return b.replaceCommon(sideRepoMaintenanceValidateWorkflowsTemplate,
		"@@CONDITION@@", RenderCondition(buildDispatchOperationCondition("validate")),
		"@@FORMATTED_RUNS_ON@@", FormatRunsOn(nil, "ubuntu-latest"),
		"@@INSTALL_CLI_STEPS@@", generateInstallCLISteps(b.ctx, b.actionMode, b.version, b.actionTag, b.resolver),
		"@@CLI_CMD_PREFIX@@", getCLICmdPrefix(b.actionMode),
	)
}

func (b sideRepoMaintenanceBuild) actionsCheckoutStep() string {
	if b.actionMode != ActionModeDev && b.actionMode != ActionModeScript {
		return ""
	}
	return strings.ReplaceAll(sideRepoMaintenanceActionsCheckoutTemplate, "@@CHECKOUT_PIN@@", getActionPin("actions/checkout"))
}

func (b sideRepoMaintenanceBuild) replaceCommon(template string, pairs ...string) string {
	commonPairs := []string{
		"@@RUNS_ON@@", b.runsOnValue,
		"@@MINT_STEP@@", b.mintStepYAML,
		"@@SETUP_ACTION_REF@@", b.setupActionRef,
		"@@GITHUB_SCRIPT_PIN@@", getCachedActionPinFromResolver("actions/github-script", b.resolver),
		"@@CHECKOUT_PIN@@", getActionPin("actions/checkout"),
		"@@CACHE_RESTORE_PIN@@", getActionPin("actions/cache/restore"),
		"@@CACHE_SAVE_PIN@@", getActionPin("actions/cache/save"),
		"@@TOKEN@@", b.token,
		"@@REPO_SLUG@@", b.repoSlug,
	}
	return strings.NewReplacer(append(commonPairs, pairs...)...).Replace(template)
}

const sideRepoMaintenanceOnTemplate = `name: Agentic Maintenance (@@REPO_SLUG@@)

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
          - 'activity_report'
          - 'validate'
      run_url:
        description: 'Run URL or run ID to replay safe outputs from (e.g. https://github.com/owner/repo/actions/runs/12345 or 12345). Required when operation is safe_outputs.'
        required: false
        type: string
        default: ''
  workflow_call:
    inputs:
      operation:
        description: 'Optional maintenance operation to run (safe_outputs, create_labels, activity_report, validate)'
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

const sideRepoMaintenanceScheduleTemplate = `  schedule:
    - cron: "@@CRON_SCHEDULE@@"  # @@SCHEDULE_DESC@@ (based on minimum expires: @@MIN_EXPIRES_DAYS@@ days)
`

const sideRepoMaintenancePermissionsAndJobs = `
permissions: {}

jobs:
`

const sideRepoMaintenanceActionsCheckoutTemplate = `      - name: Checkout actions folder
        uses: @@CHECKOUT_PIN@@
        with:
          sparse-checkout: |
            actions
          clean: false
          persist-credentials: false

`

const sideRepoMaintenanceCloseExpiredTemplate = `  close-expired-entities:
    if: ${{ @@CONDITION@@ }}
    runs-on: @@RUNS_ON@@
    permissions:
      discussions: write
      issues: write
      pull-requests: write
    # Runs on schedule: @@CRON_SCHEDULE@@ (@@SCHEDULE_DESC@@)
    steps:
@@MINT_STEP@@@@CHECKOUT_ACTIONS_STEP@@      - name: Setup Scripts
        uses: @@SETUP_ACTION_REF@@
        with:
          destination: ${{ runner.temp }}/gh-aw/actions

      - name: Close expired discussions
        uses: @@GITHUB_SCRIPT_PIN@@
        env:
          GH_AW_TARGET_REPO_SLUG: "@@REPO_SLUG@@"
        with:
          github-token: @@TOKEN@@
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/close_expired_discussions.cjs');
            await main();

      - name: Close expired issues
        uses: @@GITHUB_SCRIPT_PIN@@
        env:
          GH_AW_TARGET_REPO_SLUG: "@@REPO_SLUG@@"
        with:
          github-token: @@TOKEN@@
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/close_expired_issues.cjs');
            await main();

      - name: Close expired pull requests
        uses: @@GITHUB_SCRIPT_PIN@@
        env:
          GH_AW_TARGET_REPO_SLUG: "@@REPO_SLUG@@"
        with:
          github-token: @@TOKEN@@
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/close_expired_pull_requests.cjs');
            await main();
`

const sideRepoMaintenanceApplySafeOutputsTemplate = `
  apply_safe_outputs:
    if: ${{ @@CONDITION@@ }}
    runs-on: @@RUNS_ON@@
    permissions:
      actions: read
      contents: write
      discussions: write
      issues: write
      pull-requests: write
    outputs:
      run_url: ${{ steps.record.outputs.run_url }}
    steps:
@@MINT_STEP@@@@CHECKOUT_ACTIONS_STEP@@      - name: Setup Scripts
        uses: @@SETUP_ACTION_REF@@
        with:
          destination: ${{ runner.temp }}/gh-aw/actions

      - name: Check admin/maintainer permissions
        uses: @@GITHUB_SCRIPT_PIN@@
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/check_team_member.cjs');
            await main();

      - name: Apply Safe Outputs
        uses: @@GITHUB_SCRIPT_PIN@@
        env:
          GH_TOKEN: @@TOKEN@@
          GH_AW_RUN_URL: ${{ inputs.run_url }}
          GH_AW_TARGET_REPO_SLUG: "@@REPO_SLUG@@"
        with:
          github-token: @@TOKEN@@
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/apply_safe_outputs_replay.cjs');
            await main();

      - name: Record outputs
        id: record
        env:
          GH_AW_RUN_URL: ${{ inputs.run_url }}
        run: echo "run_url=$GH_AW_RUN_URL" >> "$GITHUB_OUTPUT"
`

const sideRepoMaintenanceCreateLabelsTemplate = `
  create_labels:
    if: ${{ @@CONDITION@@ }}
    runs-on: @@RUNS_ON@@
    permissions:
      contents: read
      issues: write
    steps:
@@MINT_STEP@@      - name: Checkout repository
        uses: @@CHECKOUT_PIN@@
        with:
          persist-credentials: false

      - name: Setup Scripts
        uses: @@SETUP_ACTION_REF@@
        with:
          destination: ${{ runner.temp }}/gh-aw/actions

      - name: Check admin/maintainer permissions
        uses: @@GITHUB_SCRIPT_PIN@@
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/check_team_member.cjs');
            await main();

@@INSTALL_CLI_STEPS@@      - name: Create missing labels in target repository
        uses: @@GITHUB_SCRIPT_PIN@@
        env:
          GH_AW_CMD_PREFIX: @@CLI_CMD_PREFIX@@
          GH_AW_TARGET_REPO_SLUG: "@@REPO_SLUG@@"
        with:
          github-token: @@TOKEN@@
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/create_labels.cjs');
            await main();
`

const sideRepoMaintenanceActivityReportTemplate = `
  activity_report:
    if: ${{ @@CONDITION@@ }}
    runs-on: @@RUNS_ON@@
    timeout-minutes: 120
    permissions:
      actions: read
      contents: read
      issues: write
    steps:
@@MINT_STEP@@      - name: Checkout repository
        uses: @@CHECKOUT_PIN@@
        with:
          persist-credentials: false

      - name: Setup Scripts
        uses: @@SETUP_ACTION_REF@@
        with:
          destination: ${{ runner.temp }}/gh-aw/actions

      - name: Check admin/maintainer permissions
        uses: @@GITHUB_SCRIPT_PIN@@
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/check_team_member.cjs');
            await main();

@@INSTALL_CLI_STEPS@@      - name: Restore activity report logs cache
        id: activity_report_logs_cache
        uses: @@CACHE_RESTORE_PIN@@
        with:
          path: ./.cache/gh-aw/activity-report-logs
          key: ${{ runner.os }}-activity-report-logs-@@REPO_SLUG@@-${{ github.ref_name }}-${{ github.run_id }}
          restore-keys: |
            ${{ runner.os }}-activity-report-logs-@@REPO_SLUG@@-
            ${{ runner.os }}-activity-report-logs-
      - name: Download activity report logs in target repository
        timeout-minutes: 20
        shell: bash
        env:
          GH_TOKEN: @@TOKEN@@
          GH_AW_CMD_PREFIX: @@CLI_CMD_PREFIX@@
          GH_AW_TARGET_REPO_SLUG: "@@REPO_SLUG@@"
        run: |
          ${GH_AW_CMD_PREFIX} logs \
            --repo "${GH_AW_TARGET_REPO_SLUG}" \
            --start-date -1w \
            --count 500 \
            --output ./.cache/gh-aw/activity-report-logs \
            --format markdown \
            --report-file ./.cache/gh-aw/activity-report-logs/report.md

      - name: Save activity report logs cache
        if: ${{ always() }}
        uses: @@CACHE_SAVE_PIN@@
        with:
          path: ./.cache/gh-aw/activity-report-logs
          key: ${{ steps.activity_report_logs_cache.outputs.cache-primary-key }}

      - name: Generate activity report issue in target repository
        uses: @@GITHUB_SCRIPT_PIN@@
        with:
          github-token: @@TOKEN@@
          script: |
            const fs = require('node:fs');
            const reportPath = './.cache/gh-aw/activity-report-logs/report.md';
            if (!fs.existsSync(reportPath)) {
              core.warning('Activity report markdown not found at ' + reportPath + '; skipping issue creation.');
              return;
            }
            let reportBody = '';
            try {
              reportBody = fs.readFileSync(reportPath, 'utf8').trim();
            } catch (error) {
              core.warning('Failed to read activity report markdown at ' + reportPath + ': ' + error.message);
              return;
            }
            if (!reportBody) {
              core.warning('Activity report markdown is empty at ' + reportPath + '; skipping issue creation.');
              return;
            }
            const repoSlug = process.env.GH_AW_TARGET_REPO_SLUG || '';
            const [owner, repo] = repoSlug.split('/');
            if (!owner || !repo) {
              core.setFailed('Invalid GH_AW_TARGET_REPO_SLUG: ' + repoSlug);
              return;
            }
            const body = [
              '### Agentic workflow activity report',
              '',
              'Repository: ' + repoSlug,
              'Generated at: ' + new Date().toISOString(),
              '',
              reportBody,
            ].join('\n');
            const createdIssue = await github.rest.issues.create({
              owner,
              repo,
              title: '[aw] agentic status report',
              body,
              labels: ['agentic-workflows'],
            });
            core.info('Created issue #' + createdIssue.data.number + ': ' + createdIssue.data.html_url);
`

const sideRepoMaintenanceValidateWorkflowsTemplate = `
  validate_workflows:
    if: ${{ @@CONDITION@@ }}
    runs-on: @@FORMATTED_RUNS_ON@@
    permissions:
      contents: read
      issues: write
    steps:
      - name: Checkout repository
        uses: @@CHECKOUT_PIN@@
        with:
          persist-credentials: false

      - name: Setup Scripts
        uses: @@SETUP_ACTION_REF@@
        with:
          destination: ${{ runner.temp }}/gh-aw/actions

      - name: Check admin/maintainer permissions
        uses: @@GITHUB_SCRIPT_PIN@@
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/check_team_member.cjs');
            await main();

@@INSTALL_CLI_STEPS@@      - name: Validate workflows and file issue on findings
        uses: @@GITHUB_SCRIPT_PIN@@
        env:
          GH_AW_CMD_PREFIX: @@CLI_CMD_PREFIX@@
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          script: |
            const { setupGlobals } = require('${{ runner.temp }}/gh-aw/actions/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('${{ runner.temp }}/gh-aw/actions/run_validate_workflows.cjs');
            await main();
`
