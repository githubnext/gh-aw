package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cli/go-gh/v2"
	"github.com/githubnext/gh-aw/pkg/parser"
)

// InstallPackage installs agentic workflows from a GitHub repository
func InstallPackage(repoSpec string, verbose bool) error {
	if verbose {
		fmt.Fprintf(os.Stderr, "Installing package: %s\n", repoSpec)
	}

	// Parse repository specification (org/repo[@version])
	spec, err := parseRepoSpec(repoSpec)
	if err != nil {
		return fmt.Errorf("invalid repository specification: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Repository: %s\n", spec.RepoSlug)
		if spec.Version != "" {
			fmt.Fprintf(os.Stderr, "Version: %s\n", spec.Version)
		} else {
			fmt.Fprintf(os.Stderr, "Version: main (default)\n")
		}
	}

	// Get global packages directory
	packagesDir, err := getPackagesDir()
	if err != nil {
		return fmt.Errorf("failed to determine packages directory: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Installing to global packages directory: %s\n", packagesDir)
	}

	// Create packages directory
	if err := os.MkdirAll(packagesDir, 0755); err != nil {
		return fmt.Errorf("failed to create packages directory: %w", err)
	}

	// Create target directory for this repository
	targetDir := filepath.Join(packagesDir, spec.RepoSlug)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create package directory: %w", err)
	}

	// Check if package already exists
	if _, err := os.Stat(targetDir); err == nil {
		entries, err := os.ReadDir(targetDir)
		if err == nil && len(entries) > 0 {
			fmt.Fprintf(os.Stderr, "Package %s already exists. Updating...\n", spec.RepoSlug)
			// Remove existing content
			if err := os.RemoveAll(targetDir); err != nil {
				return fmt.Errorf("failed to remove existing package: %w", err)
			}
			if err := os.MkdirAll(targetDir, 0755); err != nil {
				return fmt.Errorf("failed to recreate package directory: %w", err)
			}
		}
	}

	// Download workflows from the repository
	if err := downloadWorkflows(spec.RepoSlug, spec.Version, targetDir, verbose); err != nil {
		return fmt.Errorf("failed to download workflows: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Successfully installed package: %s\n", spec.RepoSlug)
	return nil
}

// downloadWorkflows downloads all .md files from the workflows directory of a GitHub repository
func downloadWorkflows(repo, version, targetDir string, verbose bool) error {
	if verbose {
		fmt.Printf("Downloading workflows from %s/workflows...\n", repo)
	}

	// Create a temporary directory for cloning
	tempDir, err := os.MkdirTemp("", "gh-aw-clone-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Prepare clone arguments - handle SHA commits vs branches/tags differently
	var cloneArgs []string
	isSHA := isCommitSHA(version)

	if isSHA {
		// For commit SHAs, we need full clone to reach the specific commit
		cloneArgs = []string{"repo", "clone", repo, tempDir}
	} else {
		// For branches/tags, use shallow clone for efficiency
		cloneArgs = []string{"repo", "clone", repo, tempDir, "--", "--depth", "1"}
		if version != "" && version != "main" {
			cloneArgs = append(cloneArgs, "--branch", version)
		}
	}

	if verbose {
		fmt.Printf("Cloning repository: gh %s\n", strings.Join(cloneArgs, " "))
	}

	// Clone the repository
	_, stdErr, err := gh.Exec(cloneArgs...)
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w (stderr: %s)", err, stdErr.String())
	}

	// If a specific SHA was requested, checkout that commit
	if isSHA {
		cmd := exec.Command("git", "checkout", version)
		cmd.Dir = tempDir
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to checkout commit %s: %w (output: %s)", version, err, string(output))
		}
		if verbose {
			fmt.Printf("Checked out commit: %s\n", version)
		}
	}

	// Get the current commit SHA from the cloned repository
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = tempDir
	commitBytes, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get commit SHA: %w", err)
	}
	commitSHA := strings.TrimSpace(string(commitBytes))

	// Validate that we're at the expected commit if a specific SHA was requested
	if isSHA && commitSHA != version {
		return fmt.Errorf("cloned repository is at commit %s, but expected %s", commitSHA, version)
	}

	if verbose {
		fmt.Printf("Repository commit SHA: %s\n", commitSHA)
	}

	// Copy all .md files from temp directory to target
	if err := copyMarkdownFiles(tempDir, targetDir, verbose); err != nil {
		return err
	}

	// Store the commit SHA in a metadata file for later retrieval
	metadataPath := filepath.Join(targetDir, ".commit-sha")
	if err := os.WriteFile(metadataPath, []byte(commitSHA), 0644); err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "Warning: Failed to write commit SHA metadata: %v\n", err)
		}
	}

	return nil
}

// copyMarkdownFiles recursively copies markdown files from source to target directory
func copyMarkdownFiles(sourceDir, targetDir string, verbose bool) error {
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if not a markdown file
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".md") {
			return nil
		}

		// Get relative path from source directory
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Create target file path
		targetFile := filepath.Join(targetDir, relPath)

		// Create target directory if needed
		targetFileDir := filepath.Dir(targetFile)
		if err := os.MkdirAll(targetFileDir, 0755); err != nil {
			return fmt.Errorf("failed to create target directory %s: %w", targetFileDir, err)
		}

		// Copy file
		if verbose {
			fmt.Printf("Copying: %s -> %s\n", relPath, targetFile)
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read source file %s: %w", path, err)
		}

		if err := os.WriteFile(targetFile, content, 0644); err != nil {
			return fmt.Errorf("failed to write target file %s: %w", targetFile, err)
		}

		return nil
	})
}

// Package represents an installed package
type Package struct {
	Name      string
	Path      string
	Workflows []string
	CommitSHA string
}

// WorkflowSourceInfo contains information about where a workflow was found
type WorkflowSourceInfo struct {
	PackagePath string
	SourcePath  string
	CommitSHA   string // The actual commit SHA used when the package was installed
}

// findWorkflowInPackageForRepo searches for a workflow in installed packages
func findWorkflowInPackageForRepo(workflow *WorkflowSpec, verbose bool) ([]byte, *WorkflowSourceInfo, error) {

	packagesDir, err := getPackagesDir()
	if err != nil {
		if verbose {
			fmt.Printf("Warning: Failed to get packages directory: %v\n", err)
		}
		return nil, nil, fmt.Errorf("failed to get packages directory: %w", err)
	}

	if _, err := os.Stat(packagesDir); os.IsNotExist(err) {
		if verbose {
			fmt.Printf("No packages directory found at %s\n", packagesDir)
		}
		return nil, nil, fmt.Errorf("no packages directory found")
	}

	// Handle local workflows (starting with "./")
	if strings.HasPrefix(workflow.WorkflowPath, "./") {
		if verbose {
			fmt.Printf("Searching local filesystem for workflow: %s\n", workflow.WorkflowPath)
		}

		// For local workflows, use current directory as packagePath
		packagePath := "."
		workflowFile := workflow.WorkflowPath

		if verbose {
			fmt.Printf("Looking for local workflow: %s\n", workflowFile)
		}

		content, err := os.ReadFile(workflowFile)
		if err != nil {
			return nil, nil, fmt.Errorf("local workflow '%s' not found: %w", workflow.WorkflowPath, err)
		}

		sourceInfo := &WorkflowSourceInfo{
			PackagePath: packagePath,
			SourcePath:  workflowFile,
			CommitSHA:   "", // Local workflows don't have commit SHA
		}

		return content, sourceInfo, nil
	}

	if verbose {
		fmt.Printf("Searching packages in %s for workflow: %s\n", packagesDir, workflow.WorkflowPath)
	}

	// Check if workflow name contains org/repo prefix
	// Fully qualified name: org/repo/workflow_name
	packagePath := filepath.Join(packagesDir, workflow.RepoSlug)
	workflowFile := filepath.Join(packagePath, workflow.WorkflowPath)

	if verbose {
		fmt.Printf("Looking for qualified workflow: %s\n", workflowFile)
	}

	content, err := os.ReadFile(workflowFile)
	if err != nil {
		return nil, nil, fmt.Errorf("workflow '%s' not found in repo '%s'", workflow.WorkflowPath, workflow.RepoSlug)
	}

	// Try to read the commit SHA from metadata file
	var commitSHA string
	metadataPath := filepath.Join(packagePath, ".commit-sha")
	if shaBytes, err := os.ReadFile(metadataPath); err == nil {
		commitSHA = strings.TrimSpace(string(shaBytes))
		if verbose {
			fmt.Printf("Found commit SHA from metadata: %s\n", commitSHA)
		}
	} else if verbose {
		fmt.Printf("Warning: Could not read commit SHA metadata: %v\n", err)
	}

	sourceInfo := &WorkflowSourceInfo{
		PackagePath: packagePath,
		SourcePath:  workflowFile,
		CommitSHA:   commitSHA,
	}

	return content, sourceInfo, nil

}

// collectPackageIncludeDependencies collects dependencies for package-based workflows
func collectPackageIncludeDependencies(content, packagePath string, verbose bool) ([]IncludeDependency, error) {
	var dependencies []IncludeDependency
	seen := make(map[string]bool)

	if verbose {
		fmt.Printf("Collecting package dependencies from: %s\n", packagePath)
	}

	err := collectPackageIncludesRecursive(content, packagePath, &dependencies, seen, verbose)
	return dependencies, err
}

// collectPackageIncludesRecursive recursively processes @include directives in package content
func collectPackageIncludesRecursive(content, baseDir string, dependencies *[]IncludeDependency, seen map[string]bool, verbose bool) error {
	includePattern := regexp.MustCompile(`^@include(\?)?\s+(.+)$`)

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if matches := includePattern.FindStringSubmatch(line); matches != nil {
			isOptional := matches[1] == "?"
			includePath := strings.TrimSpace(matches[2])

			// Handle section references (file.md#Section)
			var filePath string
			if strings.Contains(includePath, "#") {
				parts := strings.SplitN(includePath, "#", 2)
				filePath = parts[0]
			} else {
				filePath = includePath
			}

			// Resolve the full source path relative to base directory
			fullSourcePath := filepath.Join(baseDir, filePath)

			// Skip if we've already processed this file
			if seen[fullSourcePath] {
				continue
			}
			seen[fullSourcePath] = true

			// Add dependency
			dep := IncludeDependency{
				SourcePath: fullSourcePath,
				TargetPath: filePath, // Keep relative path for target
				IsOptional: isOptional,
			}
			*dependencies = append(*dependencies, dep)

			if verbose {
				fmt.Printf("Found include dependency: %s -> %s\n", fullSourcePath, filePath)
			}

			// Read the included file and process its includes recursively
			includedContent, err := os.ReadFile(fullSourcePath)
			if err != nil {
				if verbose {
					fmt.Printf("Warning: Could not read include file %s: %v\n", fullSourcePath, err)
				}
				continue
			}

			// Extract markdown content from the included file
			markdownContent, err := parser.ExtractMarkdownContent(string(includedContent))
			if err != nil {
				if verbose {
					fmt.Printf("Warning: Could not extract markdown from %s: %v\n", fullSourcePath, err)
				}
				continue
			}

			// Recursively process includes in the included file
			includedDir := filepath.Dir(fullSourcePath)
			if err := collectPackageIncludesRecursive(markdownContent, includedDir, dependencies, seen, verbose); err != nil {
				if verbose {
					fmt.Printf("Warning: Error processing includes in %s: %v\n", fullSourcePath, err)
				}
			}
		}
	}

	return scanner.Err()
}

// copyIncludeDependenciesFromPackageWithForce copies include dependencies from package filesystem with force option
func copyIncludeDependenciesFromPackageWithForce(dependencies []IncludeDependency, githubWorkflowsDir string, verbose bool, force bool, tracker *FileTracker) error {
	for _, dep := range dependencies {
		// Create the target path in .github/workflows
		targetPath := filepath.Join(githubWorkflowsDir, dep.TargetPath)

		// Create target directory if it doesn't exist
		targetDir := filepath.Dir(targetPath)
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", targetDir, err)
		}

		// Read source content from package
		sourceContent, err := os.ReadFile(dep.SourcePath)
		if err != nil {
			if dep.IsOptional {
				// For optional includes, just show an informational message and skip
				if verbose {
					fmt.Printf("Optional include file not found: %s (you can create this file to configure the workflow)\n", dep.TargetPath)
				}
				continue
			}
			fmt.Printf("Warning: Failed to read include file %s: %v\n", dep.SourcePath, err)
			continue
		}

		// Check if target file already exists
		fileExists := false
		if existingContent, err := os.ReadFile(targetPath); err == nil {
			fileExists = true
			// File exists, compare contents
			if string(existingContent) == string(sourceContent) {
				// Contents are the same, skip
				if verbose {
					fmt.Printf("Include file %s already exists with same content, skipping\n", dep.TargetPath)
				}
				continue
			}

			// Contents are different
			if !force {
				fmt.Printf("Include file %s already exists with different content, skipping (use --force to overwrite)\n", dep.TargetPath)
				continue
			}

			// Force is enabled, overwrite
			fmt.Printf("Overwriting existing include file: %s\n", dep.TargetPath)
		}

		// Track the file based on whether it existed before (if tracker is available)
		if tracker != nil {
			if fileExists {
				tracker.TrackModified(targetPath)
			} else {
				tracker.TrackCreated(targetPath)
			}
		}

		// Write to target
		if err := os.WriteFile(targetPath, sourceContent, 0644); err != nil {
			return fmt.Errorf("failed to write include file %s: %w", targetPath, err)
		}

		if verbose {
			fmt.Printf("Copied include file: %s -> %s\n", dep.SourcePath, targetPath)
		}
	}

	return nil
}

// IncludeDependency represents a file dependency from @include directives
type IncludeDependency struct {
	SourcePath string // Path in the source (local)
	TargetPath string // Relative path where it should be copied in .github/workflows
	IsOptional bool   // Whether this is an optional include (@include?)
}
