package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/workflow"
)

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
	NeedsMutation      bool
	PlanLines          []string
}

type bootstrapRuntime struct {
	setupRepositoryRuntime
	confirmAction    func(string, string, string) (bool, error)
	initRepo         func(InitOptions) error
	addWorkflows     func(context.Context, []string, AddOptions) (*AddWorkflowsResult, error)
	compileWorkflows func(context.Context, CompileConfig) ([]*workflow.WorkflowData, error)
}

func defaultBootstrapRuntime() bootstrapRuntime {
	setupRuntime := defaultSetupRepositoryRuntime()
	return bootstrapRuntime{
		setupRepositoryRuntime: setupRuntime,
		confirmAction:          console.ConfirmAction,
		initRepo:               InitRepository,
		addWorkflows:           AddWorkflows,
		compileWorkflows:       CompileWorkflows,
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
		return nil
	}

	if !plan.NeedsMutation {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Bootstrap already satisfied for %s", opts.Repo)))
		return nil
	}

	if !opts.Yes {
		if IsRunningInCI() {
			return errors.New("--yes is required in CI when bootstrap would make changes")
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
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Created %s", plan.Repo)))
	}

	if plan.CloneRepo {
		if err := runtime.cloneRepo(ctx, plan.Repo, plan.Dir); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Cloned %s into %s", plan.Repo, plan.Dir)))
	}

	resolvedSources := resolveDeployWorkflowSpecs(opts.Sources, originalDir)

	if err := withWorkingDir(plan.Dir, func() error {
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
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Initialized repository for agentic workflows"))
		}

		addedWorkflows := false
		if len(resolvedSources) > 0 {
			addOpts := AddOptions{
				Verbose:        opts.Verbose,
				EngineOverride: opts.EngineOverride,
				Force:          opts.Force,
			}
			workflowsToAdd, skippedWorkflows, err := excludeExistingSourcedWorkflows(resolvedSources, addOpts)
			if err != nil {
				return fmt.Errorf("failed to inspect existing workflows: %w", err)
			}
			if len(skippedWorkflows) > 0 {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Skipping already sourced workflows: %s", strings.Join(skippedWorkflows, ", "))))
			}
			if len(workflowsToAdd) > 0 {
				if _, err := runtime.addWorkflows(ctx, workflowsToAdd, addOpts); err != nil {
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

		return nil
	}); err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Bootstrap completed for %s", plan.Repo)))
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

func validateBootstrapOptions(opts BootstrapOptions) error {
	if strings.Count(opts.Repo, "/") != 1 {
		return errors.New("--repo must use the OWNER/REPO format")
	}

	switch opts.Visibility {
	case "private", "public", "internal":
	default:
		return errors.New("--visibility must be one of: private, public, internal")
	}

	switch opts.RequireOwnerType {
	case "any", "org", "user":
	default:
		return errors.New("--require-owner-type must be one of: any, org, user")
	}

	return nil
}

func buildBootstrapPlan(ctx context.Context, opts BootstrapOptions, runtime bootstrapRuntime, originalDir string) (*bootstrapPlan, error) {
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
			if err := withWorkingDir(plan.Dir, func() error {
				var excludeErr error
				workflowsToAdd, skippedWorkflows, excludeErr = excludeExistingSourcedWorkflows(plan.ResolvedSources, addOpts)
				return excludeErr
			}); err != nil {
				return nil, err
			}
			plan.ResolvedSources = workflowsToAdd
			plan.SkippedSources = skippedWorkflows
			plan.CompileAfterAdd = len(workflowsToAdd) > 0 && !opts.NoCompile
		}
	}

	plan.PlanLines = buildBootstrapPlanLines(plan, opts)
	plan.NeedsMutation = plan.CreateRepo || plan.CloneRepo || plan.InitNeeded || len(plan.ResolvedSources) > 0

	if plan.AttachedCheckout && plan.NeedsMutation {
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
		markerPath := filepath.Join(baseDir, filepath.FromSlash(marker))
		if _, err := os.Stat(markerPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				missing = append(missing, marker)
				continue
			}
			return nil, fmt.Errorf("failed to inspect %s: %w", marker, err)
		}
	}
	return missing, nil
}

func expectedBootstrapInitMarkers(engineOverride string) []string {
	markers := []string{
		".gitattributes",
		".vscode/settings.json",
	}
	if engineOverride == "" || engineOverride == "copilot" {
		markers = append(markers,
			".github/skills/agentic-workflows/SKILL.md",
			".github/agents/agentic-workflows.md",
			".github/mcp.json",
			".github/workflows/copilot-setup-steps.yml",
		)
	}
	return markers
}

func buildBootstrapPlanLines(plan *bootstrapPlan, opts BootstrapOptions) []string {
	lines := []string{fmt.Sprintf("Bootstrap plan for %s", plan.Repo)}

	if plan.CreateRepo {
		lines = append(lines, fmt.Sprintf("- create remote repository (%s)", opts.Visibility))
		if plan.CloneRepo {
			lines = append(lines, fmt.Sprintf("- clone into %s", plan.Dir))
		}
	} else if plan.CloneRepo {
		lines = append(lines, fmt.Sprintf("- clone existing repository into %s", plan.Dir))
	} else if plan.AttachedCheckout {
		lines = append(lines, fmt.Sprintf("- attach existing checkout at %s", plan.Dir))
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
		lines = append(lines, fmt.Sprintf("- skip already sourced workflows: %s", strings.Join(plan.SkippedSources, ", ")))
	}

	if plan.OwnerType != "" {
		lines = append(lines, fmt.Sprintf("- verified owner type: %s", normalizeSetupOwnerType(plan.OwnerType)))
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
