package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/gitutil"
)

var findGitRootForManifestValidation = gitutil.FindGitRoot

func validateRepositoryManifestForCompilation(config CompileConfig, stats *CompilationStats, validationResults *[]ValidationResult) error {
	gitRoot, err := findGitRootForManifestValidation()
	if err != nil {
		if errors.Is(err, gitutil.ErrNotGitRepository) {
			return nil
		}
		return fmt.Errorf("failed to find git root for manifest validation: %w", err)
	}

	manifestPath, err := findLocalRepositoryPackageManifest(gitRoot)
	if err != nil {
		return err
	}
	if manifestPath == "" {
		return nil
	}

	content, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read Agentic Workflow manifest %q: %w", manifestPath, err)
	}

	_, warnings, parseErr := parseRepositoryPackageManifest(manifestPath, content)
	if parseErr == nil {
		parseErr = validateLocalRepositoryPackageContents(manifestPath)
	}

	if len(warnings) > 0 {
		stats.Warnings += len(warnings)
	}

	result := ValidationResult{
		Workflow: filepath.Base(manifestPath),
		Valid:    parseErr == nil,
	}
	for _, warning := range warnings {
		result.Warnings = append(result.Warnings, CompileValidationError{
			Type:    "manifest_warning",
			Message: warning,
		})
	}

	if parseErr != nil {
		result.Errors = append(result.Errors, CompileValidationError{
			Type:    "manifest_error",
			Message: parseErr.Error(),
		})
		*validationResults = append(*validationResults, result)

		if config.JSONOutput {
			return errors.New("compilation failed")
		}
		return parseErr
	}

	if len(result.Warnings) > 0 {
		*validationResults = append(*validationResults, result)
		if !config.JSONOutput {
			for _, warning := range warnings {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(warning))
			}
		}
	}

	return nil
}

func findLocalRepositoryPackageManifest(gitRoot string) (string, error) {
	manifestPath := filepath.Join(gitRoot, repositoryPackageManifestFileName)
	if _, err := os.Stat(manifestPath); err == nil {
		return manifestPath, nil
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to check if Agentic Workflow manifest %q exists: %w", manifestPath, err)
	}

	return "", nil
}

func validateLocalRepositoryPackageContents(manifestPath string) error {
	readmePath := filepath.Join(filepath.Dir(manifestPath), "README.md")
	if _, err := os.Stat(readmePath); err == nil {
		return nil
	} else if os.IsNotExist(err) {
		return fmt.Errorf("invalid Agentic Workflow manifest %q: missing required README.md", manifestPath)
	} else {
		return fmt.Errorf("failed to read package README %q: %w", readmePath, err)
	}
}
