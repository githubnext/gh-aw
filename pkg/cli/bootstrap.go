package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var bootstrapLog = logger.New("cli:bootstrap")

type BootstrapOptions struct {
	Ctx              context.Context
	Repo             string
	Dir              string
	CreateRepo       bool
	Visibility       string
	RequireOwnerType string
	Yes              bool
	PlanOnly         bool
	EngineOverride   string
	Sources          []string
	Force            bool
	NoCompile        bool
	Verbose          bool
}

type bootstrapPlan struct {
	Repo               string
	Dir                string
	RepoExists         bool
	CreateRepo         bool
	CloneRepo          bool
	AttachedCheckout   bool
	InitNeeded         bool
	InitMissingMarkers []string
	ResolvedSources    []string
	SkippedSources     []string
	CompileAfterAdd    bool
	OwnerType          string
	BootstrapProfile   *resolvedBootstrapProfile
	ProfilePlanLines   []string
	ProfileNeedsAction bool
	NeedsMutation      bool
	PlanLines          []string
}

type bootstrapRuntime struct {
	setupRepositoryRuntime
	confirmAction    func(string, string, string) (bool, error)
	initRepo         func(InitOptions) error
	addWorkflows     func(context.Context, []string, AddOptions) (*AddWorkflowsResult, error)
	compileWorkflows func(context.Context, CompileConfig) ([]*workflow.WorkflowData, error)
	resolveProfile   func(context.Context, []string) (*resolvedBootstrapProfile, error)
	profileNeedsPlan func(context.Context, string, *resolvedBootstrapProfile, []string, bool) (bool, []string, error)
	executeProfile   func(context.Context, bootstrapProfileRunConfig) error
}

const (
	bootstrapAddWorkflowsRetryHint = "repository initialization completed; re-run bootstrap to retry workflow addition"
	bootstrapMCPConfigPath         = ".github/mcp.json"
	bootstrapCopilotSetupPath      = ".github/workflows/copilot-setup-steps.yml"
	bootstrapAgenticSkillPath      = ".github/skills/agentic-workflows/SKILL.md"
	bootstrapAgenticAgentPath      = ".github/agents/agentic-workflows.md"
)

func defaultBootstrapRuntime() bootstrapRuntime {
	setupRuntime := defaultSetupRepositoryRuntime()
	return bootstrapRuntime{
		setupRepositoryRuntime: setupRuntime,
		confirmAction:          console.ConfirmAction,
		initRepo:               InitRepository,
		addWorkflows:           AddWorkflows,
		compileWorkflows:       CompileWorkflows,
		resolveProfile:         resolveBootstrapProfileFromSources,
		profileNeedsPlan:       buildBootstrapProfilePlan,
		executeProfile:         executeBootstrapProfile,
	}
}

func RunBootstrap(opts BootstrapOptions) error {
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to determine current directory: %w", err)
	}
	return runBootstrapWithRuntime(normalizeBootstrapOptions(opts), defaultBootstrapRuntime(), originalDir)
}

func runBootstrapWithRuntime(opts BootstrapOptions, runtime bootstrapRuntime, originalDir string) error {
	runtime = normalizeBootstrapRuntime(runtime)

	if err := validateBootstrapOptions(opts); err != nil {
		return err
	}

	ctx := opts.Ctx
	if ctx == nil {
		ctx = context.Background()
	}

	plan, err := buildBootstrapPlan(ctx, opts, runtime, originalDir)
	if err != nil {
		return err
	}

	printBootstrapPlan(plan)

	if opts.PlanOnly {
		bootstrapLog.Printf("Plan-only mode; skipping mutation for %s", opts.Repo)
		return nil
	}

	if !plan.NeedsMutation {
		bootstrapLog.Printf("Bootstrap already satisfied for %s; no mutation needed", opts.Repo)
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Bootstrap already satisfied for "+opts.Repo))
		return nil
	}

	if !opts.Yes {
		if IsRunningInCI() {
			return errors.New("--yes is required in CI when bootstrap would make changes. Example: gh aw bootstrap --repo OWNER/REPO --yes")
		}
		confirmed, err := runtime.confirmAction(
			fmt.Sprintf("Apply bootstrap changes to %s?", plan.Repo),
			"Apply changes",
			"Cancel",
		)
		if err != nil {
			return fmt.Errorf("failed to confirm bootstrap plan: %w", err)
		}
		if !confirmed {
			return errors.New("bootstrap cancelled")
		}
	}

	if plan.CreateRepo {
		if err := runtime.createRepo(ctx, plan.Repo, opts.Visibility); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created "+plan.Repo))
	}

	if plan.CloneRepo {
		if err := runtime.cloneRepo(ctx, plan.Repo, plan.Dir); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Cloned %s into %s", plan.Repo, plan.Dir)))
	}

	if err := withWorkingDir(plan.Dir, func() error {
		repositoryInitialized := false
		missingMarkers, err := missingBootstrapInitMarkers(".", opts.EngineOverride)
		if err != nil {
			return err
		}
		if len(missingMarkers) > 0 {
			if err := runtime.initRepo(InitOptions{
				Ctx:              ctx,
				Verbose:          opts.Verbose,
				Engine:           opts.EngineOverride,
				Skill:            true,
				Agent:            true,
				MCP:              true,
				CodespaceRepos:   []string{},
				CodespaceEnabled: false,
				Completions:      false,
				CreatePR:         false,
			}); err != nil {
				return fmt.Errorf("failed to initialize repository: %w", err)
			}
			repositoryInitialized = true
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Initialized repository for agentic workflows"))
		}

		addedWorkflows := false
		if len(plan.ResolvedSources) > 0 {
			addOpts := AddOptions{
				Verbose:        opts.Verbose,
				EngineOverride: opts.EngineOverride,
				Force:          opts.Force,
			}
			workflowsToAdd := plan.ResolvedSources
			var skippedWorkflows []string
			// Attached checkouts were already filtered during plan construction.
			if !opts.Force && !plan.AttachedCheckout {
				workflowsToAdd, skippedWorkflows, err = excludeExistingSourcedWorkflows(plan.ResolvedSources, addOpts)
				if err != nil {
					return fmt.Errorf("failed to inspect existing workflows: %w", err)
				}
			}
			if len(skippedWorkflows) > 0 {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Skipping already sourced workflows: "+strings.Join(skippedWorkflows, ", ")))
			}
			if len(workflowsToAdd) > 0 {
				if _, err := runtime.addWorkflows(ctx, workflowsToAdd, addOpts); err != nil {
					if repositoryInitialized {
						return fmt.Errorf("failed to add workflows (%s): %w", bootstrapAddWorkflowsRetryHint, err)
					}
					return fmt.Errorf("failed to add workflows: %w", err)
				}
				addedWorkflows = true
			}
		}

		if addedWorkflows && !opts.NoCompile {
			if _, err := runtime.compileWorkflows(ctx, CompileConfig{
				Verbose:        opts.Verbose,
				EngineOverride: opts.EngineOverride,
			}); err != nil {
				return fmt.Errorf("failed to compile workflows: %w", err)
			}
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Compiled workflows"))
		}

		if plan.BootstrapProfile != nil && runtime.executeProfile != nil {
			if err := runtime.executeProfile(ctx, bootstrapProfileRunConfig{
				Repo:     plan.Repo,
				RepoDir:  plan.Dir,
				Sources:  resolveDeployWorkflowSpecs(opts.Sources, originalDir),
				Profile:  plan.BootstrapProfile,
				Yes:      opts.Yes,
				PlanOnly: opts.PlanOnly,
				Verbose:  opts.Verbose,
				Force:    opts.Force,
			}); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Bootstrap completed for "+plan.Repo))
	return nil
}

func normalizeBootstrapOptions(opts BootstrapOptions) BootstrapOptions {
	if opts.Visibility == "" {
		opts.Visibility = "private"
	}
	if opts.RequireOwnerType == "" {
		opts.RequireOwnerType = "any"
	}
	return opts
}

func normalizeBootstrapRuntime(runtime bootstrapRuntime) bootstrapRuntime {
	defaults := defaultBootstrapRuntime()
	if runtime.confirmAction == nil {
		runtime.confirmAction = defaults.confirmAction
	}
	if runtime.initRepo == nil {
		runtime.initRepo = defaults.initRepo
	}
	if runtime.addWorkflows == nil {
		runtime.addWorkflows = defaults.addWorkflows
	}
	if runtime.compileWorkflows == nil {
		runtime.compileWorkflows = defaults.compileWorkflows
	}
	if runtime.resolveProfile == nil {
		runtime.resolveProfile = defaults.resolveProfile
	}
	if runtime.profileNeedsPlan == nil {
		runtime.profileNeedsPlan = defaults.profileNeedsPlan
	}
	if runtime.executeProfile == nil {
		runtime.executeProfile = defaults.executeProfile
	}
	return runtime
}

func validateBootstrapOptions(opts BootstrapOptions) error {
	if !isValidOwnerRepoSlug(opts.Repo) {
		return errors.New("--repo must use the OWNER/REPO format. Example: --repo github/gh-aw")
	}

	switch opts.Visibility {
	case "private", "public", "internal":
	default:
		return errors.New("--visibility must be one of: private, public, internal. Example: --visibility private")
	}

	switch opts.RequireOwnerType {
	case "any", "org", "user":
	default:
		return errors.New("--require-owner-type must be one of: any, org, user. Example: --require-owner-type org")
	}

	return nil
}

func buildBootstrapPlan(ctx context.Context, opts BootstrapOptions, runtime bootstrapRuntime, originalDir string) (*bootstrapPlan, error) {
	bootstrapLog.Printf("Building bootstrap plan: repo=%s, createRepo=%t, sources=%d", opts.Repo, opts.CreateRepo, len(opts.Sources))
	if err := runtime.checkAuth(ctx); err != nil {
		return nil, fmt.Errorf("failed to verify GitHub CLI authentication: %w", err)
	}

	plan := &bootstrapPlan{
		Repo:            opts.Repo,
		Dir:             resolveSetupCheckoutDir(opts.Repo, opts.Dir),
		ResolvedSources: resolveDeployWorkflowSpecs(opts.Sources, originalDir),
		CompileAfterAdd: len(opts.Sources) > 0 && !opts.NoCompile,
	}

	owner := strings.Split(opts.Repo, "/")[0]
	if opts.RequireOwnerType != "any" {
		ownerType, err := runtime.ownerType(ctx, owner)
		if err != nil {
			return nil, err
		}
		plan.OwnerType = ownerType
		if normalizeSetupOwnerType(ownerType) != opts.RequireOwnerType {
			return nil, fmt.Errorf("owner %s is %s, but --require-owner-type=%s was requested", owner, normalizeSetupOwnerType(ownerType), opts.RequireOwnerType)
		}
	}

	repoExists, err := runtime.repoExists(ctx, opts.Repo)
	if err != nil {
		return nil, err
	}
	plan.RepoExists = repoExists
	if !repoExists {
		if !opts.CreateRepo {
			return nil, fmt.Errorf("repository %s does not exist; rerun with --create-repo to create it", opts.Repo)
		}
		plan.CreateRepo = true
	}

	inspection, err := inspectSetupCheckout(plan.Dir, plan.Repo, runtime.dirOriginRepo)
	if err != nil {
		return nil, err
	}
	plan.CloneRepo = inspection.cloneNeeded
	plan.AttachedCheckout = inspection.attached

	if inspection.attached {
		missingMarkers, err := missingBootstrapInitMarkers(plan.Dir, opts.EngineOverride)
		if err != nil {
			return nil, err
		}
		plan.InitMissingMarkers = missingMarkers
		plan.InitNeeded = len(missingMarkers) > 0

		if len(plan.ResolvedSources) > 0 {
			addOpts := AddOptions{EngineOverride: opts.EngineOverride}
			var workflowsToAdd []string
			var skippedWorkflows []string
			if opts.Force {
				workflowsToAdd = plan.ResolvedSources
			} else {
				if err := withWorkingDir(plan.Dir, func() error {
					var excludeErr error
					workflowsToAdd, skippedWorkflows, excludeErr = excludeExistingSourcedWorkflows(plan.ResolvedSources, addOpts)
					return excludeErr
				}); err != nil {
					return nil, err
				}
			}
			plan.ResolvedSources = workflowsToAdd
			plan.SkippedSources = skippedWorkflows
			plan.CompileAfterAdd = len(workflowsToAdd) > 0 && !opts.NoCompile
		}
	}

	if len(opts.Sources) > 0 && runtime.resolveProfile != nil {
		profile, err := runtime.resolveProfile(ctx, opts.Sources)
		if err != nil {
			return nil, err
		}
		if profile != nil {
			plan.BootstrapProfile = profile
			profileNeedsPlan := runtime.profileNeedsPlan
			if profileNeedsPlan == nil {
				profileNeedsPlan = defaultBootstrapRuntime().profileNeedsPlan
			}
			needsAction, profileLines, err := profileNeedsPlan(ctx, plan.Repo, profile, opts.Sources, plan.RepoExists)
			if err != nil {
				return nil, err
			}
			plan.ProfileNeedsAction = needsAction
			plan.ProfilePlanLines = append(plan.ProfilePlanLines, profileLines...)
		}
	}

	plan.PlanLines = buildBootstrapPlanLines(plan, opts)
	plan.NeedsMutation = plan.CreateRepo || plan.CloneRepo || plan.InitNeeded || len(plan.ResolvedSources) > 0 || plan.ProfileNeedsAction
	bootstrapLog.Printf("Bootstrap plan built: createRepo=%t, cloneRepo=%t, attached=%t, initNeeded=%t, profileNeedsAction=%t, needsMutation=%t",
		plan.CreateRepo, plan.CloneRepo, plan.AttachedCheckout, plan.InitNeeded, plan.ProfileNeedsAction, plan.NeedsMutation)

	if !opts.PlanOnly && plan.AttachedCheckout && plan.NeedsMutation {
		if err := withWorkingDir(plan.Dir, func() error {
			return runtime.checkCleanWorktree(opts.Verbose)
		}); err != nil {
			return nil, err
		}
	}

	return plan, nil
}

func missingBootstrapInitMarkers(baseDir string, engineOverride string) ([]string, error) {
	markers := expectedBootstrapInitMarkers(engineOverride)
	missing := make([]string, 0)
	for _, marker := range markers {
		ok, err := isBootstrapInitMarkerSatisfied(baseDir, marker)
		if err != nil {
			return nil, err
		}
		if !ok {
			missing = append(missing, marker)
		}
	}
	return missing, nil
}

func isBootstrapInitMarkerSatisfied(baseDir string, marker string) (bool, error) {
	markerPath := filepath.Join(baseDir, filepath.FromSlash(marker))
	info, err := os.Stat(markerPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("failed to inspect %s: %w", marker, err)
	}
	if !info.Mode().IsRegular() {
		return false, nil
	}

	switch marker {
	case ".gitattributes":
		content, err := os.ReadFile(markerPath)
		if err != nil {
			return false, fmt.Errorf("failed to inspect %s: %w", marker, err)
		}
		return strings.Contains(string(content), constants.WorkflowsLockYmlGitAttributesEntry), nil
	case bootstrapMCPConfigPath:
		content, err := os.ReadFile(markerPath)
		if err != nil {
			return false, fmt.Errorf("failed to inspect %s: %w", marker, err)
		}
		var config MCPConfig
		if err := json.Unmarshal(content, &config); err != nil {
			return false, nil
		}
		servers := config.MCPServers
		if len(servers) == 0 {
			servers = config.Servers
		}
		server, ok := servers["github-agentic-workflows"]
		if !ok {
			return false, nil
		}
		if strings.TrimSpace(server.Command) != "gh" {
			return false, nil
		}
		return len(server.Args) >= 2 && server.Args[0] == "aw" && server.Args[1] == "mcp-server", nil
	case bootstrapCopilotSetupPath:
		content, err := os.ReadFile(markerPath)
		if err != nil {
			return false, fmt.Errorf("failed to inspect %s: %w", marker, err)
		}
		steps := string(content)
		hasLegacyInstall := strings.Contains(steps, "install-gh-aw.sh") ||
			(strings.Contains(steps, "Install gh-aw extension") && strings.Contains(steps, "curl -fsSL"))
		hasActionInstall := strings.Contains(steps, "actions/setup-cli")
		return hasLegacyInstall || hasActionInstall, nil
	case bootstrapAgenticSkillPath:
		expected, err := buildAgenticWorkflowsSkillContent()
		if err != nil {
			return false, fmt.Errorf("failed to inspect %s: %w", marker, err)
		}
		content, err := os.ReadFile(markerPath)
		if err != nil {
			return false, fmt.Errorf("failed to inspect %s: %w", marker, err)
		}
		return strings.TrimSpace(string(content)) == strings.TrimSpace(expected), nil
	case bootstrapAgenticAgentPath:
		expected, err := buildAgenticWorkflowsAgentContent(baseDir)
		if err != nil {
			return false, fmt.Errorf("failed to inspect %s: %w", marker, err)
		}
		content, err := os.ReadFile(markerPath)
		if err != nil {
			return false, fmt.Errorf("failed to inspect %s: %w", marker, err)
		}
		return strings.TrimSpace(string(content)) == strings.TrimSpace(expected), nil
	default:
		return info.Size() > 0, nil
	}
}

func expectedBootstrapInitMarkers(engineOverride string) []string {
	markers := []string{
		".gitattributes",
		".vscode/settings.json",
	}
	if engineOverride == "" || engineOverride == "copilot" {
		markers = append(markers,
			bootstrapAgenticSkillPath,
			bootstrapAgenticAgentPath,
			bootstrapMCPConfigPath,
			bootstrapCopilotSetupPath,
		)
	}
	return markers
}

func buildBootstrapPlanLines(plan *bootstrapPlan, opts BootstrapOptions) []string {
	lines := []string{"Bootstrap plan for " + plan.Repo}

	if plan.CreateRepo {
		lines = append(lines, fmt.Sprintf("- create remote repository (%s)", opts.Visibility))
		if plan.CloneRepo {
			lines = append(lines, "- clone into "+plan.Dir)
		}
	} else if plan.CloneRepo {
		lines = append(lines, "- clone existing repository into "+plan.Dir)
	} else if plan.AttachedCheckout {
		lines = append(lines, "- attach existing checkout at "+plan.Dir)
	}

	if plan.AttachedCheckout {
		if plan.InitNeeded {
			lines = append(lines, fmt.Sprintf("- initialize repository artifacts (missing: %s)", strings.Join(plan.InitMissingMarkers, ", ")))
		} else {
			lines = append(lines, "- initialization markers already present")
		}
	} else {
		lines = append(lines, "- inspect init markers after clone")
	}

	if len(plan.ResolvedSources) > 0 {
		lines = append(lines, fmt.Sprintf("- add %d workflow/package source(s)", len(plan.ResolvedSources)))
		if plan.CompileAfterAdd {
			lines = append(lines, "- compile workflows after adding sources")
		}
	}
	if len(plan.SkippedSources) > 0 {
		lines = append(lines, "- skip already sourced workflows: "+strings.Join(plan.SkippedSources, ", "))
	}

	if plan.BootstrapProfile != nil {
		lines = append(lines, "- evaluate bootstrap actions from "+plan.BootstrapProfile.PackageID)
		if plan.ProfileNeedsAction {
			lines = append(lines, fmt.Sprintf("- apply bootstrap profile actions (%d action(s))", len(plan.BootstrapProfile.Profile.Config)))
		} else {
			lines = append(lines, "- bootstrap profile actions already satisfied")
		}
		lines = append(lines, plan.ProfilePlanLines...)
	}

	if plan.OwnerType != "" {
		lines = append(lines, "- verified owner type: "+normalizeSetupOwnerType(plan.OwnerType))
	}

	if !plan.CreateRepo && !plan.CloneRepo && !plan.InitNeeded && len(plan.ResolvedSources) == 0 {
		lines = append(lines, "- no changes required")
	}

	return lines
}

func printBootstrapPlan(plan *bootstrapPlan) {
	for _, line := range plan.PlanLines {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(line))
	}
	fmt.Fprintln(os.Stderr, "")
}

func withWorkingDir(dir string, fn func() error) error {
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to read current directory: %w", err)
	}
	if err := os.Chdir(dir); err != nil {
		return fmt.Errorf("failed to change directory to %s: %w", dir, err)
	}
	defer func() {
		_ = os.Chdir(originalDir)
	}()
	return fn()
}
