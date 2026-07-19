package workflow

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/constants"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/fileutil"
	"github.com/github/gh-aw/pkg/logger"
)

var maintenanceLog = logger.New("workflow:maintenance_workflow")

// generateInstallCLISteps generates YAML steps to install or build the gh-aw CLI.
// In dev mode: builds from source using Setup Go + Build gh-aw (./gh-aw binary available)
// In release mode: installs the released CLI via the setup-cli action (gh aw available)
// In action mode: installs the released CLI via the gh-aw-actions/setup-cli action (gh aw available)
// When resolver is non-nil, attempts to resolve the setup-cli action to a SHA-pinned reference.
func generateInstallCLISteps(ctx context.Context, actionMode ActionMode, version string, actionTag string, resolver SHAResolver) string {
	if actionMode == ActionModeDev {
		return `      - name: Setup Go
        uses: ` + getActionPin("actions/setup-go") + `
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
		ref := resolveActionRef(ctx, actionRepo, cliTag, resolver)
		return `      - name: Install gh-aw
        uses: ` + ref + `
        with:
          version: ` + cliTag + `

`
	}

	// Release mode: use setup-cli action from external gh-aw-actions repository
	actionRepo := GitHubActionsOrgRepo + "/setup-cli"
	ref := resolveActionRef(ctx, actionRepo, cliTag, resolver)
	return `      - name: Install gh-aw
        uses: ` + ref + `
        with:
          version: ` + cliTag + `

`
}

// resolveActionRef attempts to resolve an action repo@tag to a SHA-pinned reference
// using the provided resolver. If the resolver is nil or resolution fails, it returns
// the tag-based reference (repo@tag).
func resolveActionRef(ctx context.Context, actionRepo, tag string, resolver SHAResolver) string {
	if resolver != nil && tag != "" && tag != "dev" {
		sha, err := resolver.ResolveSHA(ctx, actionRepo, tag)
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

// FetchDefaultBranch queries the GitHub API to determine the default branch of the
// given repository slug (owner/repo). Returns "main" as a fallback when the slug is
// empty, not in owner/repo format, or when the API call fails.
func FetchDefaultBranch(slug string) string {
	const fallback = "main"
	if slug == "" || strings.Count(slug, "/") != 1 {
		maintenanceLog.Printf("No valid repository slug, using default branch fallback: %s", fallback)
		return fallback
	}
	maintenanceLog.Printf("Fetching default branch for repository: %s", slug)
	output, err := RunGH("Fetching default branch...", "api", "/repos/"+slug, "--jq", ".default_branch")
	if err != nil {
		maintenanceLog.Printf("Failed to fetch default branch for %s: %v, falling back to %s", slug, err, fallback)
		return fallback
	}
	branch := strings.TrimSpace(string(output))
	if branch == "" {
		maintenanceLog.Printf("Empty default branch response for %s, falling back to %s", slug, fallback)
		return fallback
	}
	maintenanceLog.Printf("Default branch for %s: %s", slug, branch)
	return branch
}

// GenerateMaintenanceWorkflowOptions configures a maintenance workflow generation run.
type GenerateMaintenanceWorkflowOptions struct {
	WorkflowDataList []*WorkflowData
	WorkflowDir      string
	Version          string
	ActionMode       ActionMode
	ActionTag        string
	RepoConfig       *RepoConfig
	RepoSlug         string
}

const defaultNoOpIssueExpirationHours = 24 * 30

func isNoOpReportAsIssueEnabled(reportAsIssue *string) bool {
	return reportAsIssue == nil || !strings.EqualFold(strings.TrimSpace(*reportAsIssue), "false")
}

// GenerateMaintenanceWorkflow generates the agentics-maintenance.yml workflow
// if any workflows use expiring safe outputs or noop issue reporting.
// When opts.RepoConfig is non-nil and opts.RepoConfig.MaintenanceDisabled is true the
// maintenance workflow is deleted and the function returns immediately.
// opts.RepoSlug is the owner/repo slug used to determine the default branch for the push
// trigger; pass an empty string to fall back to "main".
func GenerateMaintenanceWorkflow(ctx context.Context, opts GenerateMaintenanceWorkflowOptions) error {
	maintenanceLog.Print("Checking if maintenance workflow is needed")

	generation := newMaintenanceWorkflowGeneration(ctx, opts)

	// Respect explicit opt-out from aw.json: maintenance: false
	if opts.RepoConfig != nil && opts.RepoConfig.MaintenanceDisabled {
		return generateMaintenanceDisabledAutoUpdate(ctx, opts, generation)
	}

	// Scan workflows for expires fields and track the minimum expires value
	hasExpires, minExpires, triggerReason := scanWorkflowsForExpires(opts.WorkflowDataList)

	if !hasExpires {
		return generateNoExpiresMaintenanceWorkflows(ctx, opts, generation)
	}

	return generateExpiresMaintenanceWorkflow(ctx, opts, generation, minExpires, triggerReason)
}

type maintenanceWorkflowGeneration struct {
	resolver                       SHAResolver
	setupActionRef                 string
	githubScriptPin                string
	runsOnValue                    string
	configuredRunsOn               RunsOnValue
	disableLabelTrigger            bool
	maintenanceConfig              *MaintenanceConfig
	compileGitHubTokenSecret       string
	enableCompileCreatePullRequest bool
}

func newMaintenanceWorkflowGeneration(ctx context.Context, opts GenerateMaintenanceWorkflowOptions) maintenanceWorkflowGeneration {
	resolver := firstWorkflowActionResolver(opts.WorkflowDataList)
	generation := maintenanceWorkflowGeneration{
		resolver:            resolver,
		setupActionRef:      ResolveSetupActionReference(ctx, opts.ActionMode, opts.Version, opts.ActionTag, resolver),
		githubScriptPin:     getCachedActionPinFromResolver("actions/github-script", resolver),
		disableLabelTrigger: true,
	}
	if opts.RepoConfig != nil && opts.RepoConfig.Maintenance != nil {
		generation.maintenanceConfig = opts.RepoConfig.Maintenance
		generation.configuredRunsOn = generation.maintenanceConfig.RunsOn
		generation.disableLabelTrigger = !generation.maintenanceConfig.IsLabelTriggerEnabled()
		if generation.maintenanceConfig.Compile != nil {
			generation.compileGitHubTokenSecret = generation.maintenanceConfig.Compile.CreatePullRequestGitHubToken
			generation.enableCompileCreatePullRequest = strings.TrimSpace(generation.compileGitHubTokenSecret) != ""
		}
	}
	generation.runsOnValue = FormatRunsOn(generation.configuredRunsOn, "ubuntu-slim")
	return generation
}

func firstWorkflowActionResolver(workflowDataList []*WorkflowData) SHAResolver {
	for _, workflowData := range workflowDataList {
		if workflowData != nil && workflowData.ActionResolver != nil {
			return workflowData.ActionResolver
		}
	}
	return nil
}

func generateMaintenanceDisabledAutoUpdate(ctx context.Context, opts GenerateMaintenanceWorkflowOptions, generation maintenanceWorkflowGeneration) error {
	if err := handleMaintenanceDisabled(opts.WorkflowDataList, opts.WorkflowDir); err != nil {
		return err
	}
	return generateMaintenanceAutoUpdate(ctx, opts, generation, opts.RepoConfig.IsAutoUpgradeEnabled())
}

func generateNoExpiresMaintenanceWorkflows(ctx context.Context, opts GenerateMaintenanceWorkflowOptions, generation maintenanceWorkflowGeneration) error {
	maintenanceLog.Print("No workflows use expires field, skipping maintenance workflow generation")
	if err := deleteExistingMaintenanceWorkflow(opts.WorkflowDir); err != nil {
		return err
	}
	if err := generateAllSideRepoMaintenanceWorkflows(ctx, generateAllSideRepoMaintenanceWorkflowsOptions{
		workflowDataList: opts.WorkflowDataList,
		workflowDir:      opts.WorkflowDir,
		version:          opts.Version,
		actionMode:       opts.ActionMode,
		actionTag:        opts.ActionTag,
		runsOnValue:      generation.runsOnValue,
		resolver:         generation.resolver,
		hasExpires:       false,
		minExpiresDays:   0,
	}); err != nil {
		return err
	}
	return generateMaintenanceAutoUpdate(ctx, opts, generation, opts.RepoConfig != nil && opts.RepoConfig.IsAutoUpgradeEnabled())
}

func deleteExistingMaintenanceWorkflow(workflowDir string) error {
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

func generateExpiresMaintenanceWorkflow(ctx context.Context, opts GenerateMaintenanceWorkflowOptions, generation maintenanceWorkflowGeneration, minExpires int, triggerReason string) error {
	maintenanceLog.Printf("Maintenance workflow generation triggered: %s", triggerReason)
	maintenanceLog.Printf("Generating maintenance workflow for expired discussions, issues, and pull requests (minimum expires: %d hours)", minExpires)

	// Convert hours to days for cron schedule generation
	minExpiresDays := minExpires / 24
	if minExpires%24 > 0 {
		minExpiresDays++ // Round up partial days
	}

	// Generate cron schedule based on minimum expires value
	cronSchedule, scheduleDesc := generateMaintenanceCron(minExpiresDays)
	maintenanceLog.Printf("Maintenance schedule: %s (%s)", cronSchedule, scheduleDesc)

	// Fetch the default branch for the push trigger (dev mode only)
	// Resolved here to avoid passing it through multiple layers; empty slug falls back to "main"
	defaultBranch := FetchDefaultBranch(opts.RepoSlug)

	content := buildExpiresMaintenanceWorkflowContent(ctx, opts, generation, minExpiresDays, cronSchedule, scheduleDesc, defaultBranch)

	// Write the maintenance workflow file
	if err := writeMaintenanceWorkflowFile(opts.WorkflowDir, content); err != nil {
		return err
	}

	// Generate side-repo maintenance workflows for any SideRepoOps targets detected.
	if err := generateSideRepoMaintenanceForExpires(ctx, opts, generation, minExpiresDays); err != nil {
		return err
	}

	return generateMaintenanceAutoUpdate(ctx, opts, generation, opts.RepoConfig != nil && opts.RepoConfig.IsAutoUpgradeEnabled())
}

func buildExpiresMaintenanceWorkflowContent(ctx context.Context, opts GenerateMaintenanceWorkflowOptions, generation maintenanceWorkflowGeneration, minExpiresDays int, cronSchedule string, scheduleDesc string, defaultBranch string) string {
	maintenanceLog.Printf(
		"Maintenance compile configuration: createPullRequest=%v tokenSecretConfigured=%v",
		generation.enableCompileCreatePullRequest,
		strings.TrimSpace(generation.compileGitHubTokenSecret) != "",
	)
	return buildMaintenanceWorkflowYAML(ctx, buildMaintenanceWorkflowYAMLOptions{
		cronSchedule:        cronSchedule,
		scheduleDesc:        scheduleDesc,
		minExpiresDays:      minExpiresDays,
		runsOnValue:         generation.runsOnValue,
		actionMode:          opts.ActionMode,
		version:             opts.Version,
		actionTag:           opts.ActionTag,
		resolver:            generation.resolver,
		configuredRunsOn:    generation.configuredRunsOn,
		defaultBranch:       defaultBranch,
		disableLabelTrigger: generation.disableLabelTrigger,
		maintenanceConfig:   generation.maintenanceConfig,
		compileGitHubToken:  getEffectiveMaintenanceGitHubToken(generation.compileGitHubTokenSecret),
		createCompilePR:     generation.enableCompileCreatePullRequest,
		copilotOrgBilling:   allCopilotWorkflowsUseOrgBilling(opts.WorkflowDataList),
	})
}

func writeMaintenanceWorkflowFile(workflowDir string, content string) error {
	maintenanceFile := filepath.Join(workflowDir, "agentics-maintenance.yml")
	maintenanceLog.Printf("Writing maintenance workflow to %s", maintenanceFile)
	if err := fileutil.EnsureParentDir(maintenanceFile, constants.DirPermPublic); err != nil {
		return fmt.Errorf("failed to create maintenance workflow directory: %w", err)
	}
	if err := os.WriteFile(maintenanceFile, []byte(content), constants.FilePermPublic); err != nil {
		return fmt.Errorf("failed to write maintenance workflow: %w", err)
	}
	maintenanceLog.Print("Maintenance workflow generated successfully")
	return nil
}

func generateSideRepoMaintenanceForExpires(ctx context.Context, opts GenerateMaintenanceWorkflowOptions, generation maintenanceWorkflowGeneration, minExpiresDays int) error {
	return generateAllSideRepoMaintenanceWorkflows(ctx, generateAllSideRepoMaintenanceWorkflowsOptions{
		workflowDataList: opts.WorkflowDataList,
		workflowDir:      opts.WorkflowDir,
		version:          opts.Version,
		actionMode:       opts.ActionMode,
		actionTag:        opts.ActionTag,
		runsOnValue:      generation.runsOnValue,
		resolver:         generation.resolver,
		hasExpires:       true,
		minExpiresDays:   minExpiresDays,
	})
}

func generateMaintenanceAutoUpdate(ctx context.Context, opts GenerateMaintenanceWorkflowOptions, generation maintenanceWorkflowGeneration, enabled bool) error {
	return GenerateAutoUpdateWorkflow(GenerateAutoUpdateWorkflowOptions{
		Context:         ctx,
		WorkflowDir:     opts.WorkflowDir,
		Enabled:         enabled,
		RepoSlug:        opts.RepoSlug,
		SetupActionRef:  generation.setupActionRef,
		GitHubScriptPin: generation.githubScriptPin,
		ActionMode:      opts.ActionMode,
		Version:         opts.Version,
		ActionTag:       opts.ActionTag,
		Resolver:        generation.resolver,
	})
}

// handleMaintenanceDisabled handles the case where maintenance is disabled in repo config.
// It warns about workflows that use expires and deletes any existing maintenance workflow.
func handleMaintenanceDisabled(workflowDataList []*WorkflowData, workflowDir string) error {
	maintenanceLog.Print("Maintenance disabled via repo config, skipping generation")

	// Warn if any workflow uses expires — those features rely on maintenance
	// and will silently become no-ops when it is disabled.
	for _, workflowData := range workflowDataList {
		if workflowData == nil || workflowData.SafeOutputs == nil {
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

// allCopilotWorkflowsUseOrgBilling reports whether all Copilot-engine workflows
// in the list have copilot-requests: write set. This indicates org billing mode,
// where the GITHUB_TOKEN is used for Copilot authentication and the
// COPILOT_GITHUB_TOKEN secret is not required.
// Returns false if no Copilot workflows are found (billing mode is indeterminate)
// or if any Copilot workflow does not have copilot-requests: write set.
func allCopilotWorkflowsUseOrgBilling(workflowDataList []*WorkflowData) bool {
	copilotCount := 0
	for _, data := range workflowDataList {
		if data == nil {
			continue
		}
		engineID := ResolveEngineID(data)
		// Default engine (empty string) is Copilot, as is an explicit "copilot" ID.
		if engineID != "" && engineID != string(constants.CopilotEngine) {
			continue
		}
		copilotCount++
		if !hasCopilotRequestsWritePermission(data) {
			return false
		}
	}
	return copilotCount > 0
}

// scanWorkflowsForExpires checks all workflow data for expires fields and returns
// whether any expires fields are set, the minimum expires value in hours, and the
// first reason that triggered maintenance workflow generation.
func scanWorkflowsForExpires(workflowDataList []*WorkflowData) (bool, int, string) {
	state := &expiresScanState{}

	for _, workflowData := range workflowDataList {
		if workflowData == nil || workflowData.SafeOutputs == nil {
			continue
		}
		scanWorkflowForExpires(workflowData, state)
	}

	return state.hasExpires, state.minExpires, state.triggerReason
}

type expiresScanState struct {
	hasExpires    bool
	minExpires    int
	triggerReason string
}

func (s *expiresScanState) record(workflowName string, expires int, reason string, logMessage string) {
	s.hasExpires = true
	if s.triggerReason == "" {
		s.triggerReason = reason
		maintenanceLog.Printf("Maintenance workflow became required: %s", reason)
	}
	maintenanceLog.Printf(logMessage, workflowName, expires)
	if s.minExpires == 0 || expires < s.minExpires {
		s.minExpires = expires
	}
}

func scanWorkflowForExpires(workflowData *WorkflowData, state *expiresScanState) {
	if workflowData.SafeOutputs.CreateDiscussions != nil && workflowData.SafeOutputs.CreateDiscussions.Expires > 0 {
		expires := workflowData.SafeOutputs.CreateDiscussions.Expires
		state.record(workflowData.Name, expires, fmt.Sprintf("workflow %q sets safe_outputs.create_discussions.expires=%dh", workflowData.Name, expires), "Workflow %s has expires field set to %d hours for discussions")
	}
	if workflowData.SafeOutputs.CreateIssues != nil && workflowData.SafeOutputs.CreateIssues.Expires > 0 {
		expires := workflowData.SafeOutputs.CreateIssues.Expires
		state.record(workflowData.Name, expires, fmt.Sprintf("workflow %q sets safe_outputs.create_issues.expires=%dh", workflowData.Name, expires), "Workflow %s has expires field set to %d hours for issues")
	}
	if workflowData.SafeOutputs.CreatePullRequests != nil && workflowData.SafeOutputs.CreatePullRequests.Expires > 0 {
		expires := workflowData.SafeOutputs.CreatePullRequests.Expires
		state.record(workflowData.Name, expires, fmt.Sprintf("workflow %q sets safe_outputs.create_pull_requests.expires=%dh", workflowData.Name, expires), "Workflow %s has expires field set to %d hours for pull requests")
	}
	if workflowData.SafeOutputs.NoOp != nil && isNoOpReportAsIssueEnabled(workflowData.SafeOutputs.NoOp.ReportAsIssue) {
		expires := defaultNoOpIssueExpirationHours
		state.record(workflowData.Name, expires, fmt.Sprintf("workflow %q enables no-op issue reporting (default expiration %dh)", workflowData.Name, expires), "Workflow %s has no-op report-as-issue enabled, using %d-hour no-op issue expiration")
	}
}
