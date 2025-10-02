package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/parser"
)

// InstallImports installs all imports from workflows
// If workflowName is provided, only install imports for that workflow
func InstallImports(workflowName string, verbose bool) error {
	if verbose {
		if workflowName != "" {
			fmt.Fprintln(os.Stderr, console.FormatProgressMessage(fmt.Sprintf("Installing imports for workflow: %s", workflowName)))
		} else {
			fmt.Fprintln(os.Stderr, console.FormatProgressMessage("Installing imports for all workflows"))
		}
	}

	// Get workflows directory
	workflowsDir := filepath.Join(".github", "workflows")
	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		return fmt.Errorf("workflows directory not found: %s", workflowsDir)
	}

	// Collect all imports from workflows
	imports, err := collectImportsFromWorkflows(workflowsDir, workflowName, verbose)
	if err != nil {
		return fmt.Errorf("failed to collect imports: %w", err)
	}

	if len(imports) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No imports found in workflows"))
		return nil
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Found %d unique import(s)", len(imports))))
	}

	// Get or create imports directory
	importsDir, err := parser.GetImportsDir()
	if err != nil {
		return fmt.Errorf("failed to get imports directory: %w", err)
	}

	if err := os.MkdirAll(importsDir, 0755); err != nil {
		return fmt.Errorf("failed to create imports directory: %w", err)
	}

	// Get lock file path
	lockFilePath, err := parser.GetImportLockFilePath()
	if err != nil {
		return fmt.Errorf("failed to get lock file path: %w", err)
	}

	// Ensure .aw directory exists
	awDir := filepath.Dir(lockFilePath)
	if err := os.MkdirAll(awDir, 0755); err != nil {
		return fmt.Errorf("failed to create .aw directory: %w", err)
	}

	// Read existing lock file
	lock, err := parser.ReadImportLockFile(lockFilePath)
	if err != nil {
		return fmt.Errorf("failed to read lock file: %w", err)
	}

	// Process each import
	for _, importSpec := range imports {
		if err := installSingleImport(importSpec, importsDir, lock, verbose); err != nil {
			return fmt.Errorf("failed to install import %s: %w", importSpec.String(), err)
		}
	}

	// Write updated lock file
	if err := parser.WriteImportLockFile(lockFilePath, lock); err != nil {
		return fmt.Errorf("failed to write lock file: %w", err)
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Successfully installed all imports"))
	return nil
}

// collectImportsFromWorkflows collects all unique imports from workflow files
func collectImportsFromWorkflows(workflowsDir, workflowName string, verbose bool) ([]*parser.ImportSpec, error) {
	var allImports []*parser.ImportSpec
	seen := make(map[string]bool)

	// Determine which files to process
	var filesToProcess []string
	if workflowName != "" {
		// Process specific workflow
		workflowFile := filepath.Join(workflowsDir, workflowName)
		if !strings.HasSuffix(workflowFile, ".md") {
			workflowFile += ".md"
		}
		if _, err := os.Stat(workflowFile); os.IsNotExist(err) {
			return nil, fmt.Errorf("workflow file not found: %s", workflowFile)
		}
		filesToProcess = append(filesToProcess, workflowFile)
	} else {
		// Process all markdown files in workflows directory
		err := filepath.Walk(workflowsDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
				filesToProcess = append(filesToProcess, path)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to walk workflows directory: %w", err)
		}
	}

	// Extract imports from each file
	for _, filePath := range filesToProcess {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Checking %s", filePath)))
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", filePath, err)
		}

		// Parse frontmatter
		result, err := parser.ExtractFrontmatterFromContent(string(content))
		if err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to parse frontmatter in %s: %v", filePath, err)))
			}
			continue
		}

		// Check for imports field
		importsValue, hasImports := result.Frontmatter["imports"]
		if !hasImports {
			continue
		}

		// Parse imports
		imports, err := parser.ParseImports(importsValue)
		if err != nil {
			return nil, fmt.Errorf("failed to parse imports in %s: %w", filePath, err)
		}

		// Add to collection (deduplicate)
		for _, imp := range imports {
			key := imp.String()
			if !seen[key] {
				seen[key] = true
				allImports = append(allImports, imp)

				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  Found import: %s", imp.String())))
				}
			}
		}
	}

	return allImports, nil
}

// installSingleImport installs a single import by cloning the repository
func installSingleImport(importSpec *parser.ImportSpec, importsDir string, lock *parser.ImportLockFile, verbose bool) error {
	// Check if already installed with same SHA in lock file
	existingEntry := lock.FindEntry(importSpec)

	// Get target directory for this import
	targetDir := importSpec.GetLocalCachePath(importsDir)

	// Resolve commit SHA
	commitSHA, err := resolveImportVersion(importSpec, verbose)
	if err != nil {
		return fmt.Errorf("failed to resolve version: %w", err)
	}

	// Check if already installed with correct SHA
	if existingEntry != nil && existingEntry.CommitSHA == commitSHA {
		if _, err := os.Stat(targetDir); err == nil {
			// Verify the imported file exists
			importedFilePath := filepath.Join(targetDir, importSpec.Path)
			if _, err := os.Stat(importedFilePath); err == nil {
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Import %s already installed with correct version", importSpec.String())))
				}
				return nil
			}
		}
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatProgressMessage(fmt.Sprintf("Installing %s (%s)", importSpec.String(), commitSHA[:8])))
	}

	// Remove existing directory if present
	if _, err := os.Stat(targetDir); err == nil {
		if err := os.RemoveAll(targetDir); err != nil {
			return fmt.Errorf("failed to remove existing import directory: %w", err)
		}
	}

	// Create parent directory
	if err := os.MkdirAll(filepath.Dir(targetDir), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Clone repository (shallow clone at specific commit)
	repoURL := fmt.Sprintf("https://github.com/%s/%s.git", importSpec.Org, importSpec.Repo)

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Cloning from %s", repoURL)))
	}

	// Use git clone with depth 1 at specific commit
	cmd := exec.Command("git", "clone", "--depth", "1", "--branch", importSpec.Version, repoURL, targetDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If branch clone fails, try full clone and checkout
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Shallow clone failed, trying full clone..."))
		}

		cmd = exec.Command("git", "clone", repoURL, targetDir)
		output, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to clone repository: %w (output: %s)", err, string(output))
		}

		// Checkout specific version
		cmd = exec.Command("git", "-C", targetDir, "checkout", commitSHA)
		output, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to checkout version %s: %w (output: %s)", commitSHA, err, string(output))
		}
	}

	// Verify the imported file exists
	importedFilePath := filepath.Join(targetDir, importSpec.Path)
	if _, err := os.Stat(importedFilePath); os.IsNotExist(err) {
		return fmt.Errorf("imported file not found: %s", importSpec.Path)
	}

	// Collect transitive files (from @includes in the imported file)
	transitiveFiles, err := collectTransitiveFiles(importedFilePath, targetDir, verbose)
	if err != nil {
		return fmt.Errorf("failed to collect transitive files: %w", err)
	}

	// Update lock entry
	entry := parser.CreateImportLockEntry(importSpec, commitSHA, transitiveFiles)
	lock.AddOrUpdateEntry(entry)

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Installed %s", importSpec.String())))
	}

	return nil
}

// resolveImportVersion resolves a version string to a commit SHA
func resolveImportVersion(importSpec *parser.ImportSpec, verbose bool) (string, error) {
	repoSlug := importSpec.RepoSlug()
	version := importSpec.Version

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Resolving version %s for %s", version, repoSlug)))
	}

	// Use git ls-remote to resolve the version to a commit SHA
	// This works without authentication and is more reliable
	repoURL := fmt.Sprintf("https://github.com/%s.git", repoSlug)

	// Try to resolve as a branch or tag
	cmd := exec.Command("git", "ls-remote", repoURL, version)
	output, err := cmd.CombinedOutput()
	if verbose && err != nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("git ls-remote error: %v, output: %s", err, string(output))))
	}
	if err == nil && len(output) > 0 {
		// Parse output: SHA \t refs/...
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue
			}
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				sha := parts[0]
				if len(sha) >= 40 {
					if verbose {
						fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Resolved %s to %s", version, sha[:8])))
					}
					return sha[:40], nil
				}
			}
		}
	}

	// Try as a tag ref
	cmd = exec.Command("git", "ls-remote", repoURL, fmt.Sprintf("refs/tags/%s", version))
	output, err = cmd.Output()
	if err == nil && len(output) > 0 {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue
			}
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				sha := parts[0]
				if len(sha) >= 40 {
					if verbose {
						fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Resolved %s to %s", version, sha[:8])))
					}
					return sha[:40], nil
				}
			}
		}
	}

	// Try as a branch ref
	cmd = exec.Command("git", "ls-remote", repoURL, fmt.Sprintf("refs/heads/%s", version))
	output, err = cmd.Output()
	if err == nil && len(output) > 0 {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue
			}
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				sha := parts[0]
				if len(sha) >= 40 {
					if verbose {
						fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Resolved %s to %s", version, sha[:8])))
					}
					return sha[:40], nil
				}
			}
		}
	}

	// If we still can't resolve, return the version as-is (might be a full SHA)
	if len(version) >= 40 && importSpec.IsCommitSHA() {
		return version[:40], nil
	}

	return "", fmt.Errorf("failed to resolve version %s to commit SHA", version)
}

// collectTransitiveFiles collects all files referenced by @include directives
func collectTransitiveFiles(filePath, baseDir string, verbose bool) ([]string, error) {
	var files []string
	seen := make(map[string]bool)

	var collectRecursive func(path string) error
	collectRecursive = func(path string) error {
		if seen[path] {
			return nil
		}
		seen[path] = true

		// Read file
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Look for @include directives
		lines := strings.Split(string(content), "\n")

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "@include") {
				// Extract include path
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					includePath := parts[1]

					// Handle section references
					if strings.Contains(includePath, "#") {
						parts := strings.SplitN(includePath, "#", 2)
						includePath = parts[0]
					}

					// Resolve relative to base directory
					fullPath := filepath.Join(baseDir, includePath)
					relPath, err := filepath.Rel(baseDir, fullPath)
					if err == nil && !strings.HasPrefix(relPath, "..") {
						files = append(files, relPath)
						// Recursively collect from included file
						if err := collectRecursive(fullPath); err != nil && verbose {
							fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to process include %s: %v", includePath, err)))
						}
					}
				}
			}
		}

		return nil
	}

	if err := collectRecursive(filePath); err != nil {
		return nil, err
	}

	return files, nil
}
