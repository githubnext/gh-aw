package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

// InstallPackage installs agentic workflows from a GitHub repository
func InstallPackage(repoSpec string, verbose bool) error {
	if verbose {
		fmt.Fprintf(os.Stderr, "Installing package: %s\n", repoSpec)
	}

	// Parse repository specification (org/repo[@version])
	repo, version, err := parseRepoSpec(repoSpec)
	if err != nil {
		return fmt.Errorf("invalid repository specification: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Repository: %s\n", repo)
		if version != "" {
			fmt.Fprintf(os.Stderr, "Version: %s\n", version)
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
	targetDir := filepath.Join(packagesDir, repo)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create package directory: %w", err)
	}

	// Check if package already exists
	if _, err := os.Stat(targetDir); err == nil {
		entries, err := os.ReadDir(targetDir)
		if err == nil && len(entries) > 0 {
			fmt.Fprintf(os.Stderr, "Package %s already exists. Updating...\n", repo)
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
	if err := downloadWorkflows(repo, version, targetDir, verbose); err != nil {
		return fmt.Errorf("failed to download workflows: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Successfully installed package: %s\n", repo)
	return nil
}
