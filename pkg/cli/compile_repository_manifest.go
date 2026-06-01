package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/logger"
)

var compileRepositoryManifestLog = logger.New("cli:compile_repository_manifest")

var findGitRootForManifestValidation = gitutil.FindGitRoot

func validateRepositoryManifestForCompilation(config CompileConfig, stats *CompilationStats, validationResults *[]ValidationResult) error {
	compileRepositoryManifestLog.Print("Validating repository manifest for compilation")

	gitRoot, err := findGitRootForManifestValidation()
	if err != nil {
		if errors.Is(err, gitutil.ErrNotGitRepository) {
			compileRepositoryManifestLog.Print("Not in a git repository, skipping manifest validation")
			return nil
		}
		return fmt.Errorf("failed to find git root for manifest validation: %w", err)
	}

	manifestPath, err := findLocalRepositoryPackageManifest(gitRoot)
	if err != nil {
		return err
	}
	if manifestPath == "" {
		compileRepositoryManifestLog.Printf("No repository manifest found in %s", gitRoot)
		return nil
	}

	compileRepositoryManifestLog.Printf("Found repository manifest at %s", manifestPath)
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read Agentic Workflow manifest %q: %w", manifestPath, err)
	}

	_, warnings, parseErr := parseRepositoryPackageManifest(manifestPath, content)
	if parseErr == nil {
		parseErr = validateLocalRepositoryPackageContents(manifestPath)
	}
	compileRepositoryManifestLog.Printf("Manifest parse result: warnings=%d, error=%v", len(warnings), parseErr)

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
		manifestContent, err := os.ReadFile(manifestPath)
		if err != nil {
			return fmt.Errorf("failed to read Agentic Workflow manifest %q: %w", manifestPath, err)
		}
		manifest, _, err := parseRepositoryPackageManifest(manifestPath, manifestContent)
		if err != nil {
			return err
		}

		includeInstallablePaths, _, _ := splitManifestIncludePaths(manifest.Includes)
		includeInstallablePaths = append(includeInstallablePaths, manifest.Files...)
		installationSources := normalizePackageInstallablePaths(includeInstallablePaths, "")
		if len(installationSources) == 0 {
			installationSources, err = scanLocalRepositoryPackageInstallablePaths(filepath.Dir(manifestPath))
			if err != nil {
				return err
			}
		}

		return validateManifestInstallableWorkflowPrivacy(manifestPath, installationSources, func(sourcePath string) ([]byte, error) {
			content, err := os.ReadFile(filepath.Join(filepath.Dir(manifestPath), filepath.FromSlash(sourcePath)))
			if err != nil {
				return nil, fmt.Errorf("failed to read workflow %q: %w", sourcePath, err)
			}
			return content, nil
		})
	} else if os.IsNotExist(err) {
		return fmt.Errorf("invalid Agentic Workflow manifest %q: missing required README.md", manifestPath)
	} else {
		return fmt.Errorf("failed to read package README %q: %w", readmePath, err)
	}
}

func scanLocalRepositoryPackageInstallablePaths(packageDir string) ([]string, error) {
	var collected []string
	seen := make(map[string]struct{})

	for _, sourceDir := range packageSourceDirectories {
		sourcePath := filepath.Join(packageDir, filepath.FromSlash(sourceDir))
		err := filepath.WalkDir(sourcePath, func(currentPath string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}

			relativePath, err := filepath.Rel(packageDir, currentPath)
			if err != nil {
				return err
			}
			relativePath = filepath.ToSlash(relativePath)
			if !isSupportedPackageInstallablePath(relativePath) {
				return nil
			}
			if _, exists := seen[relativePath]; exists {
				return nil
			}
			seen[relativePath] = struct{}{}
			collected = append(collected, relativePath)
			return nil
		})
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("failed to scan %q: %w", sourcePath, err)
		}
	}

	return collected, nil
}
