package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"charm.land/huh/v2"
	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/gitutil"
)

var addFindGitRoot = gitutil.FindGitRoot
var addInitRepository = InitRepository
var addMissingInitMarkers = missingBootstrapInitMarkers
var addConfirmAuthoringSupport = func(ctx context.Context) (bool, error) {
	addAuthoringSupport := true
	form := console.NewConfirmForm(
		huh.NewConfirm().
			Title("Add prompts and skills for coding agents?").
			Description("These help coding agents author, debug, update, and audit agentic workflows in this repository.").
			Affirmative("Yes, add prompts and skills").
			Negative("No, add only the workflow").
			Value(&addAuthoringSupport),
	)
	if err := form.RunWithContext(ctx); err != nil {
		return false, fmt.Errorf("coding agent prompts and skills confirmation failed: %w", err)
	}
	return addAuthoringSupport, nil
}

type addRepositoryInitializationPlan struct {
	enabled          bool
	files            []string
	originalContents map[string][]byte
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
	originalContents := make(map[string][]byte)
	if err := withWorkingDir(gitRoot, func() error {
		var inspectErr error
		missingMarkers, inspectErr = addMissingInitMarkers(".", engineOverride)
		if inspectErr != nil {
			return inspectErr
		}
		if noGitattributes {
			missingMarkers = slices.DeleteFunc(missingMarkers, func(path string) bool {
				return path == ".gitattributes"
			})
		}
		for _, marker := range missingMarkers {
			content, readErr := os.ReadFile(filepath.FromSlash(marker))
			if readErr == nil {
				originalContents[marker] = content
			} else if !errors.Is(readErr, os.ErrNotExist) {
				return fmt.Errorf("failed to read %s: %w", marker, readErr)
			}
		}
		return nil
	}); err != nil {
		return addRepositoryInitializationPlan{}, fmt.Errorf("failed to inspect repository initialization state: %w", err)
	}
	if len(missingMarkers) == 0 {
		return addRepositoryInitializationPlan{}, nil
	}

	confirmed, err := addConfirmAuthoringSupport(ctx)
	if err != nil || !confirmed {
		if err == nil {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Coding agent prompts and skills: skipped"))
		}
		return addRepositoryInitializationPlan{}, err
	}
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Coding agent prompts and skills: enabled"))
	return addRepositoryInitializationPlan{enabled: true, files: missingMarkers, originalContents: originalContents}, nil
}

func applyAddRepositoryInitialization(plan addRepositoryInitializationPlan, engineOverride string, verbose bool, noGitattributes bool) ([]string, map[string][]byte, error) {
	if !plan.enabled {
		return nil, nil, nil
	}
	files, err := ensureAddRepositoryInitializedFromPlan(plan.files, engineOverride, verbose, noGitattributes)
	if err != nil {
		return nil, nil, err
	}
	originalContents := make(map[string][]byte, len(plan.originalContents))
	gitRoot, err := addFindGitRoot()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to determine repository root for initialized files: %w", err)
	}
	for path, content := range plan.originalContents {
		originalContents[filepath.Join(gitRoot, filepath.FromSlash(path))] = content
	}
	return files, originalContents, nil
}

func confirmAndInitializeAddRepository(ctx context.Context, engineOverride string, verbose bool, noGitattributes bool) ([]string, error) {
	plan, err := confirmAddRepositoryInitialization(ctx, engineOverride, noGitattributes)
	if err != nil {
		return nil, err
	}
	files, _, err := applyAddRepositoryInitialization(plan, engineOverride, verbose, noGitattributes)
	return files, err
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
		initializedFiles, err = initializeAddRepositoryFiles(missingMarkers, engineOverride, verbose, noGitattributes)
		return err
	})
	if err != nil {
		return nil, err
	}

	return initializedFiles, nil
}

func ensureAddRepositoryInitializedFromPlan(markers []string, engineOverride string, verbose bool, noGitattributes bool) ([]string, error) {
	gitRoot, err := addFindGitRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to determine repository root for automatic initialization: %w", err)
	}
	var initializedFiles []string
	err = withWorkingDir(gitRoot, func() error {
		var initErr error
		initializedFiles, initErr = initializeAddRepositoryFiles(markers, engineOverride, verbose, noGitattributes)
		return initErr
	})
	return initializedFiles, err
}

func initializeAddRepositoryFiles(markers []string, engineOverride string, verbose bool, noGitattributes bool) ([]string, error) {
	if len(markers) == 0 {
		return nil, nil
	}
	addLog.Printf("Repository missing init markers; running init: %v", markers)
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
		return nil, fmt.Errorf("failed to initialize repository for agentic workflows: %w", err)
	}

	initializedFiles := make([]string, 0, len(markers))
	for _, marker := range markers {
		ok, statErr := isBootstrapInitMarkerSatisfied(".", marker)
		if statErr != nil || !ok {
			continue
		}
		absPath, pathErr := filepath.Abs(marker)
		if pathErr != nil {
			return nil, fmt.Errorf("failed to resolve path for initialized file %s: %w", marker, pathErr)
		}
		initializedFiles = append(initializedFiles, absPath)
	}
	return initializedFiles, nil
}
