package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

type addCreateOptions struct {
	Repo             string
	Visibility       string
	License          string
	RequireOwnerType string
	EngineOverride   string
	Verbose          bool
	// SkipInit, when true, skips repository initialization in the checkout so
	// that PR-based flows (add-wizard, add --create-pull-request) receive a
	// clean working tree and their preflight check does not fail.
	SkipInit bool
}

func registerAddCreateFlags(cmd *cobra.Command) {
	createFlag := cmd.Flags().String("create", "", "Create or attach a target repository before adding workflows. Formats: OWNER/REPO, REPO (infers owner from current repo), or no value (prompts for repo name in add-wizard)")
	// Support --create without a value: when flag is present but no value given, cobra sets it to NoOptDefVal
	cmd.Flags().Lookup("create").NoOptDefVal = "prompt"
	_ = createFlag // flag is registered, value retrieved via GetString
	cmd.Flags().String("visibility", "private", "Repository visibility for --create: private, public, or internal")
	cmd.Flags().String("license", "", "Repository license for --create (for example: mit, apache-2.0, gpl-3.0)")
	cmd.Flags().String("require-owner-type", "any", "Require the --create repository owner to be org, user, or any")
}

func normalizeAddCreateOptions(opts addCreateOptions) addCreateOptions {
	if opts.Visibility == "" {
		opts.Visibility = "private"
	}
	if opts.RequireOwnerType == "" {
		opts.RequireOwnerType = "any"
	}
	// Skip normalization if repo is "prompt" (will be prompted for in add-wizard)
	if opts.Repo == "prompt" || opts.Repo == "" {
		return opts
	}
	// Infer owner from current repo if only repo name is provided
	if !strings.Contains(opts.Repo, "/") {
		currentRepoSlug := getRepositorySlugFromRemote()
		if currentRepoSlug != "" {
			parts := strings.Split(currentRepoSlug, "/")
			if len(parts) >= 1 {
				owner := parts[0]
				opts.Repo = owner + "/" + opts.Repo
			}
		}
	}
	return opts
}

func validateAddCreateOptions(opts addCreateOptions) error {
	// Repo is required (should have been set by normalization or prompt)
	if opts.Repo == "" {
		return errors.New("--create requires a repository name. Use OWNER/REPO, REPO (to infer owner), or no value in add-wizard (to prompt)")
	}

	// After normalization, repo must be in OWNER/REPO format
	if !isValidOwnerRepoSlug(opts.Repo) {
		return fmt.Errorf("invalid repository format %q after normalization. Expected OWNER/REPO format", opts.Repo)
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

func prepareAddTargetCheckout(ctx context.Context, opts addCreateOptions) (string, error) {
	originalDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to determine current directory before preparing the target repository checkout: %w", err)
	}
	return prepareAddTargetCheckoutWithRuntime(ctx, opts, defaultBootstrapRuntime(), originalDir)
}

func prepareAddTargetCheckoutWithRuntime(ctx context.Context, opts addCreateOptions, runtime bootstrapRuntime, originalDir string) (string, error) {
	opts = normalizeAddCreateOptions(opts)
	if err := validateAddCreateOptions(opts); err != nil {
		return "", err
	}

	runtime = normalizeBootstrapRuntime(runtime)
	if ctx == nil {
		ctx = context.Background()
	}

	plan, err := buildBootstrapPlan(ctx, normalizeBootstrapOptions(BootstrapOptions{
		Ctx:              ctx,
		Repo:             opts.Repo,
		CreateRepo:       true,
		Visibility:       opts.Visibility,
		RequireOwnerType: opts.RequireOwnerType,
		EngineOverride:   opts.EngineOverride,
		Verbose:          opts.Verbose,
	}), runtime, originalDir)
	if err != nil {
		return "", err
	}

	if plan.CreateRepo {
		if err := runtime.createRepo(ctx, plan.Repo, setupRepositoryCreateOptions{
			Visibility: opts.Visibility,
			License:    opts.License,
		}); err != nil {
			return "", err
		}
	}

	if plan.CloneRepo {
		if err := runtime.cloneRepo(ctx, plan.Repo, plan.Dir); err != nil {
			return "", err
		}
	}

	if !opts.SkipInit {
		if err := initializeAddTargetCheckoutIfNeeded(ctx, runtime, plan.Dir, opts); err != nil {
			return "", err
		}
	}

	return strings.TrimSpace(plan.Dir), nil
}

func initializeAddTargetCheckoutIfNeeded(ctx context.Context, runtime bootstrapRuntime, checkoutDir string, opts addCreateOptions) error {
	missingMarkers, err := missingBootstrapInitMarkers(checkoutDir, opts.EngineOverride)
	if err != nil {
		return err
	}
	if len(missingMarkers) == 0 {
		return nil
	}

	return withWorkingDir(checkoutDir, func() error {
		return runtime.initRepo(addCreateInitOptions(ctx, opts))
	})
}

func addCreateInitOptions(ctx context.Context, opts addCreateOptions) InitOptions {
	return InitOptions{
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
	}
}
