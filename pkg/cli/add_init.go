package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"charm.land/huh/v2"
	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/gitutil"
)

var addFindGitRoot = gitutil.FindGitRoot
var addInitRepository = InitRepository
var addMissingInitMarkers = missingBootstrapInitMarkers
var addMissingAuthoringSupportFiles = missingAddAuthoringSupportFiles
var addConfirmAuthoringSupport = func(ctx context.Context) (bool, error) {
	addAuthoringSupport := true
	form := console.NewConfirmForm(
		huh.NewConfirm().
			Title("Would you also like to add support to use coding agents in this repository to author, debug, update and audit agentic workflows?").
			Affirmative("Yes, add coding agent support").
			Negative("No, add only the workflow").
			Value(&addAuthoringSupport),
	)
	if err := form.RunWithContext(ctx); err != nil {
		return false, fmt.Errorf("coding agent support confirmation failed: %w", err)
	}
	return addAuthoringSupport, nil
}

func missingAddAuthoringSupportFiles(baseDir string, engineOverride string, noGitattributes bool) ([]string, error) {
	var missing []string
	for _, path := range expectedBootstrapInitMarkers(engineOverride) {
		if noGitattributes && path == ".gitattributes" {
			continue
		}
		info, err := os.Stat(filepath.Join(baseDir, filepath.FromSlash(path)))
		if err == nil && info.Mode().IsRegular() {
			continue
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("failed to inspect %s: %w", path, err)
		}
		missing = append(missing, path)
	}
	return missing, nil
}

type addRepositoryInitializationPlan struct {
	enabled bool
	files   []string
}

func confirmAddRepositoryInitialization(ctx context.Context, engineOverride string, noGitattributes bool) (addRepositoryInitializationPlan, error) {
	gitRoot, err := addFindGitRoot()
	if err != nil {
		if errors.Is(err, gitutil.ErrNotGitRepository) {
			return addRepositoryInitializationPlan{}, nil
		}
		return addRepositoryInitializationPlan{}, fmt.Errorf("failed to determine repository root for automatic initialization: %w", err)
	}

	var missingMarkers []string
	if err := withWorkingDir(gitRoot, func() error {
		var inspectErr error
		missingMarkers, inspectErr = addMissingAuthoringSupportFiles(".", engineOverride, noGitattributes)
		return inspectErr
	}); err != nil {
		return addRepositoryInitializationPlan{}, fmt.Errorf("failed to inspect repository initialization state: %w", err)
	}
	if len(missingMarkers) == 0 {
		return addRepositoryInitializationPlan{}, nil
	}

	confirmed, err := addConfirmAuthoringSupport(ctx)
	if err != nil || !confirmed {
		if err == nil {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Coding agent authoring support: skipped"))
		}
		return addRepositoryInitializationPlan{}, err
	}
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Coding agent authoring support: enabled"))
	return addRepositoryInitializationPlan{enabled: true, files: missingMarkers}, nil
}

func applyAddRepositoryInitialization(plan addRepositoryInitializationPlan, engineOverride string, verbose bool, noGitattributes bool) ([]string, error) {
	if !plan.enabled {
		return nil, nil
	}
	return ensureAddRepositoryInitializedWithDetails(engineOverride, verbose, noGitattributes)
}

func confirmAndInitializeAddRepository(ctx context.Context, engineOverride string, verbose bool, noGitattributes bool) ([]string, error) {
	plan, err := confirmAddRepositoryInitialization(ctx, engineOverride, noGitattributes)
	if err != nil {
		return nil, err
	}
	return applyAddRepositoryInitialization(plan, engineOverride, verbose, noGitattributes)
}

func ensureAddRepositoryInitialized(engineOverride string, verbose bool, noGitattributes bool) error {
	_, err := ensureAddRepositoryInitializedWithDetails(engineOverride, verbose, noGitattributes)
	return err
}

func ensureAddRepositoryInitializedWithDetails(engineOverride string, verbose bool, noGitattributes bool) ([]string, error) {
	gitRoot, err := addFindGitRoot()
	if err != nil {
		if errors.Is(err, gitutil.ErrNotGitRepository) {
			addLog.Print("Skipping automatic repository initialization outside a git checkout")
			return nil, nil
		}
		return nil, fmt.Errorf("failed to determine repository root for automatic initialization: %w", err)
	}

	var initializedFiles []string
	err = withWorkingDir(gitRoot, func() error {
		missingMarkers, err := addMissingInitMarkers(".", engineOverride)
		if err != nil {
			return fmt.Errorf("failed to inspect repository initialization state: %w", err)
		}
		if len(missingMarkers) == 0 {
			return nil
		}

		addLog.Printf("Repository missing init markers; running init: %v", missingMarkers)
		if err := addInitRepository(InitOptions{
			Verbose:          verbose,
			Quiet:            true,
			Engine:           engineOverride,
			NoGitattributes:  noGitattributes,
			Skill:            true,
			Agent:            true,
			MCP:              true,
			CodespaceRepos:   []string{},
			CodespaceEnabled: false,
			Completions:      false,
			CreatePR:         false,
		}); err != nil {
			return fmt.Errorf("failed to initialize repository for agentic workflows: %w", err)
		}

		// Record only the files that were actually written by init (some markers,
		// e.g. .gitattributes with --no-gitattributes, may intentionally be skipped).
		// Use absolute paths so callers don't need to resolve against gitRoot.
		for _, marker := range missingMarkers {
			ok, statErr := isBootstrapInitMarkerSatisfied(".", marker)
			if statErr != nil || !ok {
				continue
			}
			absPath, pathErr := filepath.Abs(marker)
			if pathErr != nil {
				return fmt.Errorf("failed to resolve path for initialized file %s: %w", marker, pathErr)
			}
			initializedFiles = append(initializedFiles, absPath)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return initializedFiles, nil
}
