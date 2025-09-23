package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/githubnext/gh-aw/pkg/constants"
)

// InstallPackage installs agentic workflows from a GitHub repository
func InstallPackage(repoSpec string, local bool, verbose bool) error {
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

	// Get packages directory based on local flag
	packagesDir, err := getPackagesDir(local)
	if err != nil {
		return fmt.Errorf("failed to determine packages directory: %w", err)
	}

	if verbose {
		if local {
			fmt.Fprintf(os.Stderr, "Installing to local packages directory: %s\n", packagesDir)
		} else {
			fmt.Fprintf(os.Stderr, "Installing to global packages directory: %s\n", packagesDir)
		}
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

// UninstallPackage removes an installed package
func UninstallPackage(repoSpec string, local bool, verbose bool) error {
	if verbose {
		fmt.Fprintf(os.Stderr, "Uninstalling package: %s\n", repoSpec)
	}

	// Parse repository specification (only org/repo part, ignore version)
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
			fmt.Fprintf(os.Stderr, "Uninstalling from local packages directory: %s\n", packagesDir)
		} else {
			fmt.Fprintf(os.Stderr, "Uninstalling from global packages directory: %s\n", packagesDir)
		}
	}

	// Check if package exists
	targetDir := filepath.Join(packagesDir, repo)

	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Package %s is not installed.\n", repo)
		return nil
	}

	// Remove the package directory
	if err := os.RemoveAll(targetDir); err != nil {
		return fmt.Errorf("failed to remove package directory: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Successfully uninstalled package: %s\n", repo)
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

	// Find all installed packages
	packages, err := findInstalledPackages(packagesDir)
	if err != nil {
		return fmt.Errorf("failed to scan packages: %w", err)
	}

	if len(packages) == 0 {
		fmt.Println("No packages installed.")
		fmt.Println("Use '" + constants.CLIExtensionPrefix + " install <org/repo>' to install packages.")
		return nil
	}

	for _, pkg := range packages {
		count := len(pkg.Workflows)
		if pkg.CommitSHA != "" {
			// Truncate commit SHA to first 8 characters for display
			shortSHA := pkg.CommitSHA
			if len(shortSHA) > 8 {
				shortSHA = shortSHA[:8]
			}
			if count == 1 {
				fmt.Printf("%s@%s (%d agentic workflow)\n", pkg.Name, shortSHA, count)
			} else {
				fmt.Printf("%s@%s (%d agentic workflows)\n", pkg.Name, shortSHA, count)
			}
		} else {
			if count == 1 {
				fmt.Printf("%s (%d agentic workflow)\n", pkg.Name, count)
			} else {
				fmt.Printf("%s (%d agentic workflows)\n", pkg.Name, count)
			}
		}

		if verbose {
			fmt.Printf("  Location: %s\n", pkg.Path)
			fmt.Printf("  Workflows:\n")
			for _, workflow := range pkg.Workflows {
				fmt.Printf("    - %s\n", workflow)
			}
			fmt.Println()
		}
	}

	return nil
}
