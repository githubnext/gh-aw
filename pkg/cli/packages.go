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
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/parser"
)

// Package represents an installed agentic workflow package
type Package struct {
	Name      string
	Path      string
	Workflows []string
	CommitSHA string
}

// WorkflowSourceInfo contains information about where a workflow was found
type WorkflowSourceInfo struct {
	IsPackage          bool
	PackagePath        string
	QualifiedName      string
	NeedsQualifiedName bool
	SourcePath         string
}

// InstallPackage installs agentic workflows from a GitHub repository
func InstallPackage(repoSpec string, local bool, verbose bool) error {
	if verbose {
		fmt.Printf("Installing package: %s\n", repoSpec)
	}

	// Parse repository specification
	repo, version, err := parseRepoSpec(repoSpec)
	if err != nil {
		return fmt.Errorf("invalid repository specification: %w", err)
	}

	if verbose {
		fmt.Printf("Repository: %s\n", repo)
		if version != "" {
			fmt.Printf("Version: %s\n", version)
		} else {
			fmt.Printf("Version: main (default)\n")
		}
	}

	// Get packages directory based on local flag
	packagesDir, err := getPackagesDir(local)
	if err != nil {
		return fmt.Errorf("failed to determine packages directory: %w", err)
	}

	if verbose {
		if local {
			fmt.Printf("Installing to local packages directory: %s\n", packagesDir)
		} else {
			fmt.Printf("Installing to global packages directory: %s\n", packagesDir)
		}
	}

	// Create packages directory
	if err := os.MkdirAll(packagesDir, 0755); err != nil {
		return fmt.Errorf("failed to create packages directory: %w", err)
	}

	// Create target directory for this repository
	targetDir := filepath.Join(packagesDir, repo)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Download workflows from the repository
	if err := downloadWorkflows(repo, version, targetDir, verbose); err != nil {
		return fmt.Errorf("failed to download workflows: %w", err)
	}

	fmt.Printf("✅ Package %s installed successfully!\n", repo)

	// List available workflows
	if verbose {
		workflows, err := findWorkflowsInPackage(targetDir)
		if err != nil {
			fmt.Printf("Warning: Failed to list workflows in package: %v\n", err)
		} else if len(workflows) > 0 {
			fmt.Printf("Available workflows:\n")
			for _, workflow := range workflows {
				fmt.Printf("  - %s\n", workflow)
			}
		}
	}

	return nil
}

// UninstallPackage removes an installed package
func UninstallPackage(repoSpec string, local bool, verbose bool) error {
	if verbose {
		fmt.Printf("Uninstalling package: %s\n", repoSpec)
	}

	// Parse repository specification
	repo, _, err := parseRepoSpec(repoSpec)
	if err != nil {
		return fmt.Errorf("invalid repository specification: %w", err)
	}

	// Get packages directory based on local flag
	packagesDir, err := getPackagesDir(local)
	if err != nil {
		return fmt.Errorf("failed to determine packages directory: %w", err)
	}

	if verbose {
		if local {
			fmt.Printf("Uninstalling from local packages directory: %s\n", packagesDir)
		} else {
			fmt.Printf("Uninstalling from global packages directory: %s\n", packagesDir)
		}
	}

	// Check if package exists
	targetDir := filepath.Join(packagesDir, repo)

	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		fmt.Printf("Package %s is not installed.\n", repo)
		return nil
	}

	// Remove the package directory
	if err := os.RemoveAll(targetDir); err != nil {
		return fmt.Errorf("failed to remove package directory: %w", err)
	}

	fmt.Printf("✅ Package %s uninstalled successfully!\n", repo)

	return nil
}

// ListPackages lists all installed packages
func ListPackages(local bool, verbose bool) error {
	if verbose {
		fmt.Printf("Listing installed packages...\n")
	}

	packagesDir, err := getPackagesDir(local)
	if err != nil {
		return fmt.Errorf("failed to determine packages directory: %w", err)
	}

	if verbose {
		if local {
			fmt.Printf("Looking in local packages directory: %s\n", packagesDir)
		} else {
			fmt.Printf("Looking in global packages directory: %s\n", packagesDir)
		}
	}

	if _, err := os.Stat(packagesDir); os.IsNotExist(err) {
		if local {
			fmt.Println("No local packages directory found.")
		} else {
			fmt.Println("No global packages directory found.")
		}
		fmt.Println("Use '" + constants.CLIExtensionPrefix + " install <org/repo>' to install packages.")
		return nil
	}

	packages, err := findInstalledPackages(packagesDir)
	if err != nil {
		return fmt.Errorf("failed to scan packages: %w", err)
	}

	if len(packages) == 0 {
		if local {
			fmt.Println("No local packages installed.")
		} else {
			fmt.Println("No global packages installed.")
		}
		fmt.Println("Use '" + constants.CLIExtensionPrefix + " install <org/repo>' to install packages.")
		return nil
	}

	if local {
		fmt.Printf("Local packages installed:\n")
	} else {
		fmt.Printf("Global packages installed:\n")
	}
	for _, pkg := range packages {
		fmt.Printf("  %s\n", pkg.Name)
		if verbose && len(pkg.Workflows) > 0 {
			fmt.Printf("    Workflows:\n")
			for _, workflow := range pkg.Workflows {
				fmt.Printf("      - %s\n", workflow)
			}
		}
		if verbose && pkg.CommitSHA != "" {
			shortSHA := pkg.CommitSHA
			if len(shortSHA) > 8 {
				shortSHA = shortSHA[:8]
			}
			fmt.Printf("    Commit: %s\n", shortSHA)
		}
	}

	return nil
}

// parseRepoSpec parses repository specification like "org/repo@version" or "org/repo@branch" or "org/repo@commit"
func parseRepoSpec(repoSpec string) (repo, version string, err error) {
	parts := strings.SplitN(repoSpec, "@", 2)
	repo = parts[0]

	// Validate repository format (org/repo)
	repoParts := strings.Split(repo, "/")
	if len(repoParts) != 2 || repoParts[0] == "" || repoParts[1] == "" {
		return "", "", fmt.Errorf("repository must be in format 'org/repo'")
	}

	if len(parts) == 2 {
		version = parts[1]
	}

	return repo, version, nil
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

	// Prepare clone arguments
	cloneArgs := []string{"repo", "clone", repo, tempDir}
	if version != "" && version != "main" {
		cloneArgs = append(cloneArgs, "--", "--branch", version)
	}

	if verbose {
		fmt.Printf("Cloning repository: gh %s\n", strings.Join(cloneArgs, " "))
	}

	// Clone the repository
	_, stdErr, err := gh.Exec(cloneArgs...)
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w (stderr: %s)", err, stdErr.String())
	}

	// Get the current commit SHA from the cloned repository
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = tempDir
	commitBytes, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get commit SHA: %w", err)
	}
	commitSHA := strings.TrimSpace(string(commitBytes))

	if verbose {
		fmt.Printf("Repository commit SHA: %s\n", commitSHA)
	}

	// Check if workflows directory exists
	workflowsDir := filepath.Join(tempDir, "workflows")
	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		return fmt.Errorf("workflows directory not found in repository %s", repo)
	}

	// Copy all .md files from workflows directory to target
	if err := copyMarkdownFiles(workflowsDir, targetDir, verbose); err != nil {
		return err
	}

	// Store the commit SHA in a metadata file
	metadataFile := filepath.Join(targetDir, ".aw-metadata")
	metadataContent := fmt.Sprintf("commit_sha=%s\n", commitSHA)
	if err := os.WriteFile(metadataFile, []byte(metadataContent), 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	if verbose {
		fmt.Printf("Stored commit SHA in metadata file: %s\n", metadataFile)
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
		if !strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
			return nil
		}

		// Calculate relative path from source directory
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("failed to calculate relative path: %w", err)
		}

		// Create target path
		targetPath := filepath.Join(targetDir, relPath)

		// Create target directory if needed
		targetDirPath := filepath.Dir(targetPath)
		if err := os.MkdirAll(targetDirPath, 0755); err != nil {
			return fmt.Errorf("failed to create target directory: %w", err)
		}

		// Copy file
		sourceFile, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open source file: %w", err)
		}
		defer sourceFile.Close()

		targetFile, err := os.Create(targetPath)
		if err != nil {
			return fmt.Errorf("failed to create target file: %w", err)
		}
		defer targetFile.Close()

		_, err = sourceFile.WriteTo(targetFile)
		if err != nil {
			return fmt.Errorf("failed to copy file: %w", err)
		}

		if verbose {
			fmt.Printf("Copied: %s\n", relPath)
		}

		return nil
	})
}

// findInstalledPackages finds all installed packages
func findInstalledPackages(packagesDir string) ([]Package, error) {
	var packages []Package

	// Walk through the packages directory
	err := filepath.Walk(packagesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if not a directory
		if !info.IsDir() {
			return nil
		}

		// Check if this directory contains workflow files
		if path != packagesDir {
			relPath, err := filepath.Rel(packagesDir, path)
			if err != nil {
				return err
			}

			// Only process top-level package directories (org/repo format)
			pathParts := strings.Split(filepath.ToSlash(relPath), "/")
			if len(pathParts) == 2 {
				workflows, err := findWorkflowsInPackage(path)
				if err != nil {
					return err
				}

				if len(workflows) > 0 {
					// Read commit SHA from metadata file
					commitSHA := readCommitSHAFromMetadata(path)

					packages = append(packages, Package{
						Name:      relPath,
						Path:      path,
						Workflows: workflows,
						CommitSHA: commitSHA,
					})
				}
			}
		}

		return nil
	})

	return packages, err
}

// findWorkflowsInPackage finds all workflow files in a package directory
func findWorkflowsInPackage(packageDir string) ([]string, error) {
	var workflows []string

	err := filepath.Walk(packageDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
			relPath, err := filepath.Rel(packageDir, path)
			if err != nil {
				return err
			}
			workflows = append(workflows, relPath)
		}

		return nil
	})

	return workflows, err
}

// findWorkflowInPackages searches for a workflow in installed packages
func findWorkflowInPackages(workflowPath string, verbose bool) ([]byte, *WorkflowSourceInfo, error) {
	// Check both local and global packages
	locations := []bool{true, false} // local first, then global

	// Remove .md extension if present for searching
	workflowName := strings.TrimSuffix(workflowPath, ".md")

	for _, local := range locations {
		packagesDir, err := getPackagesDir(local)
		if err != nil {
			if verbose {
				fmt.Printf("Warning: Failed to get packages directory (local=%v): %v\n", local, err)
			}
			continue
		}

		locationName := "global"
		if local {
			locationName = "local"
		}

		if _, err := os.Stat(packagesDir); os.IsNotExist(err) {
			if verbose {
				fmt.Printf("No %s packages directory found at %s\n", locationName, packagesDir)
			}
			continue
		}

		if verbose {
			fmt.Printf("Searching %s packages in %s...\n", locationName, packagesDir)
		}

		// Try qualified search (org/repo/workflow) first
		if content, sourceInfo, err := findQualifiedWorkflowInPackages(workflowName, packagesDir, verbose); err == nil {
			return content, sourceInfo, nil
		}

		// Try unqualified search if qualified fails
		if content, sourceInfo, err := findUnqualifiedWorkflowInPackages(workflowName, packagesDir, verbose); err == nil {
			return content, sourceInfo, nil
		}
	}

	return nil, nil, fmt.Errorf("workflow not found in components and no packages installed")
}

// checkPackageForUpdates checks if a package has updates available
func checkPackageForUpdates(pkg Package, verbose bool) (bool, error) {
	if pkg.CommitSHA == "" {
		// No commit SHA stored, assume it needs update
		return true, nil
	}

	// Get latest commit SHA from remote
	latestSHA, err := getLatestCommitSHA(pkg.Name)
	if err != nil {
		return false, fmt.Errorf("failed to get latest commit for %s: %w", pkg.Name, err)
	}

	hasUpdate := latestSHA != pkg.CommitSHA
	if verbose && hasUpdate {
		shortCurrent := pkg.CommitSHA
		if len(shortCurrent) > 8 {
			shortCurrent = shortCurrent[:8]
		}
		shortLatest := latestSHA
		if len(shortLatest) > 8 {
			shortLatest = shortLatest[:8]
		}
		fmt.Printf("Package %s has updates: %s -> %s\n", pkg.Name, shortCurrent, shortLatest)
	} else if verbose {
		shortCurrent := pkg.CommitSHA
		if len(shortCurrent) > 8 {
			shortCurrent = shortCurrent[:8]
		}
		fmt.Printf("Package %s is up to date: %s\n", pkg.Name, shortCurrent)
	}

	return hasUpdate, nil
}

// getLatestCommitSHA gets the latest commit SHA for a repository
func getLatestCommitSHA(repo string) (string, error) {
	// Use GitHub CLI to get the latest commit
	cmd := exec.Command("gh", "api", fmt.Sprintf("repos/%s/commits/HEAD", repo), "--jq", ".sha")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get latest commit: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// filterPackagesByWorkflow filters packages to only those containing the specified workflow
func filterPackagesByWorkflow(packages []Package, workflowName string) []Package {
	var filtered []Package
	for _, pkg := range packages {
		for _, workflow := range pkg.Workflows {
			if workflow == workflowName {
				filtered = append(filtered, pkg)
				break
			}
		}
	}
	return filtered
}

// isLocalPackage determines if a package is installed locally based on its path
func isLocalPackage(packagePath string) bool {
	return strings.Contains(packagePath, ".aw/packages") && !strings.Contains(packagePath, filepath.Join(os.Getenv("HOME"), ".aw/packages"))
}

// findInstalledWorkflowsFromPackage finds workflow files in the workflows directory that came from this package
func findInstalledWorkflowsFromPackage(pkg Package, workflowsDir string, workflowFilter string) ([]string, error) {
	var installedWorkflows []string

	// Look for markdown files in the workflows directory that match package workflows
	for _, workflowName := range pkg.Workflows {
		// Skip if workflow filter is specified and doesn't match
		if workflowFilter != "" && workflowName != workflowFilter {
			continue
		}

		// Build potential path in workflows directory
		workflowFile := filepath.Join(workflowsDir, workflowName)
		if _, err := os.Stat(workflowFile); err == nil {
			installedWorkflows = append(installedWorkflows, workflowFile)
		}
	}

	return installedWorkflows, nil
}

// findQualifiedWorkflowInPackages searches for workflows with qualified names (org/repo/workflow)
func findQualifiedWorkflowInPackages(qualifiedName, packagesDir string, verbose bool) ([]byte, *WorkflowSourceInfo, error) {
	parts := strings.Split(qualifiedName, "/")
	if len(parts) < 3 {
		return nil, nil, fmt.Errorf("qualified name must have at least org/repo/workflow format")
	}

	// Extract org, repo, and workflow path
	org := parts[0]
	repo := parts[1]
	workflowPath := strings.Join(parts[2:], "/")

	// Add .md extension if not present
	if !strings.HasSuffix(workflowPath, ".md") {
		workflowPath += ".md"
	}

	packagePath := filepath.Join(packagesDir, org, repo)
	fullWorkflowPath := filepath.Join(packagePath, workflowPath)

	if verbose {
		fmt.Printf("Looking for qualified workflow: %s\n", fullWorkflowPath)
	}

	if content, err := os.ReadFile(fullWorkflowPath); err == nil {
		if verbose {
			fmt.Printf("Found qualified workflow: %s\n", fullWorkflowPath)
		}
		return content, &WorkflowSourceInfo{
			SourcePath: fullWorkflowPath,
			IsPackage:  true,
		}, nil
	}

	return nil, nil, fmt.Errorf("qualified workflow not found: %s", qualifiedName)
}

// findUnqualifiedWorkflowInPackages searches for workflows by name across all packages
func findUnqualifiedWorkflowInPackages(workflowName, packagesDir string, verbose bool) ([]byte, *WorkflowSourceInfo, error) {
	// Add .md extension if not present
	if !strings.HasSuffix(workflowName, ".md") {
		workflowName += ".md"
	}

	var foundWorkflows []string

	// Walk through all packages to find matching workflows
	err := filepath.Walk(packagesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && info.Name() == workflowName {
			foundWorkflows = append(foundWorkflows, path)
		}

		return nil
	})

	if err != nil {
		return nil, nil, fmt.Errorf("error searching packages: %w", err)
	}

	if len(foundWorkflows) == 0 {
		return nil, nil, fmt.Errorf("workflow not found: %s", workflowName)
	}

	if len(foundWorkflows) > 1 {
		// Multiple workflows found, provide helpful error
		var packageNames []string
		for _, workflowPath := range foundWorkflows {
			relPath, err := filepath.Rel(packagesDir, workflowPath)
			if err == nil {
				// Extract package name (org/repo)
				parts := strings.Split(filepath.ToSlash(relPath), "/")
				if len(parts) >= 2 {
					packageName := strings.Join(parts[:2], "/")
					packageNames = append(packageNames, packageName)
				}
			}
		}

		return nil, nil, fmt.Errorf("multiple workflows named '%s' found in packages: %s. Use qualified name like 'org/repo/%s'",
			workflowName, strings.Join(packageNames, ", "), workflowName)
	}

	// Single workflow found
	content, err := os.ReadFile(foundWorkflows[0])
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read workflow file: %w", err)
	}

	if verbose {
		fmt.Printf("Found unqualified workflow: %s\n", foundWorkflows[0])
	}

	return content, &WorkflowSourceInfo{
		SourcePath: foundWorkflows[0],
		IsPackage:  true,
	}, nil
}

// readCommitSHAFromMetadata reads the commit SHA from the package metadata file
func readCommitSHAFromMetadata(packageDir string) string {
	metadataFile := filepath.Join(packageDir, ".aw-metadata")
	content, err := os.ReadFile(metadataFile)
	if err != nil {
		return ""
	}

	// Parse the metadata file for commit_sha
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "commit_sha=") {
			return strings.TrimPrefix(line, "commit_sha=")
		}
	}

	return "" // commit_sha not found in metadata
}

// collectPackageIncludeDependencies collects include dependencies from package content
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

// listPackageWorkflows lists all workflows available from installed packages
func listPackageWorkflows(verbose bool) error {
	// Check both local and global packages
	locations := []bool{true, false} // local first, then global
	var allPackages []Package

	for _, local := range locations {
		packagesDir, err := getPackagesDir(local)
		if err != nil {
			if verbose {
				fmt.Printf("Warning: Failed to get packages directory (local=%v): %v\n", local, err)
			}
			continue
		}

		locationName := "global"
		if local {
			locationName = "local"
		}

		if _, err := os.Stat(packagesDir); os.IsNotExist(err) {
			if verbose {
				fmt.Printf("No %s packages directory found at %s\n", locationName, packagesDir)
			}
			continue
		}

		if verbose {
			fmt.Printf("Searching for workflows in %s packages...\n", locationName)
		}

		// Find all installed packages
		packages, err := findInstalledPackages(packagesDir)
		if err != nil {
			if verbose {
				fmt.Printf("Warning: Failed to scan %s packages: %v\n", locationName, err)
			}
			continue
		}

		// Mark packages with their location
		for i := range packages {
			if local {
				packages[i].Name = packages[i].Name + ", local"
			} else {
				packages[i].Name = packages[i].Name + ", global"
			}
		}

		allPackages = append(allPackages, packages...)
	}

	if len(allPackages) == 0 {
		fmt.Println("No workflows or packages found.")
		fmt.Println("Use '" + constants.CLIExtensionPrefix + " install <org/repo>' to install packages.")
		return nil
	}

	fmt.Println("Available workflows from packages:")
	fmt.Println("==================================")

	for _, pkg := range allPackages {
		if verbose {
			fmt.Printf("Package: %s\n", pkg.Name)
		}

		for _, workflow := range pkg.Workflows {
			// Read the workflow file to get its title
			workflowFile := filepath.Join(pkg.Path, workflow+".md")
			workflowName, err := extractWorkflowNameFromFile(workflowFile)
			if err != nil || workflowName == "" {
				fmt.Printf("  %-30s (from %s)\n", workflow, pkg.Name)
			} else {
				fmt.Printf("  %-30s - %s (from %s)\n", workflow, workflowName, pkg.Name)
			}
		}
	}

	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  " + constants.CLIExtensionPrefix + " add <workflow>      - Add workflow from any package")
	fmt.Println("  " + constants.CLIExtensionPrefix + " add <workflow> -n <name> - Add workflow with specific name")
	fmt.Println("  " + constants.CLIExtensionPrefix + " list --packages     - List installed packages")

	return nil
}