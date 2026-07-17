package cli

import (
	"errors"
	"fmt"

	"github.com/github/gh-aw/pkg/gitutil"
)

var addFindGitRoot = gitutil.FindGitRoot
var addInitRepository = InitRepository
var addMissingInitMarkers = missingBootstrapInitMarkers

func ensureAddRepositoryInitialized(engineOverride string, verbose bool) error {
	_, err := ensureAddRepositoryInitializedWithDetails(engineOverride, verbose)
	return err
}

func ensureAddRepositoryInitializedWithDetails(engineOverride string, verbose bool) ([]string, error) {
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
		initializedFiles = append(initializedFiles, missingMarkers...)

		addLog.Printf("Repository missing init markers; running init: %v", missingMarkers)
		if err := addInitRepository(InitOptions{
			Verbose:          verbose,
			Engine:           engineOverride,
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

		return nil
	})
	if err != nil {
		return nil, err
	}

	return initializedFiles, nil
}
