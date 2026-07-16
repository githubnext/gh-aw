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
}

func registerAddCreateFlags(cmd *cobra.Command) {
	cmd.Flags().String("create", "", "Create or attach a target repository (OWNER/REPO format) before adding workflows")
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
	return opts
}

func validateAddCreateOptions(opts addCreateOptions) error {
	if !isValidOwnerRepoSlug(opts.Repo) {
		return errors.New("--create must use the OWNER/REPO format. Example: --create github/gh-aw")
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

	if err := initializeAddTargetCheckoutIfNeeded(ctx, runtime, plan.Dir, opts); err != nil {
		return "", err
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
