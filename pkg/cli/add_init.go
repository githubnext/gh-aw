package cli

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/github/gh-aw/pkg/gitutil"
)

var addFindGitRoot = gitutil.FindGitRoot
var addInitRepository = InitRepository
var addMissingInitMarkers = missingBootstrapInitMarkers

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
			if ok, statErr := isBootstrapInitMarkerSatisfied(".", marker); statErr == nil && ok {
				absPath, pathErr := filepath.Abs(marker)
				if pathErr == nil {
					initializedFiles = append(initializedFiles, absPath)
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return initializedFiles, nil
}
