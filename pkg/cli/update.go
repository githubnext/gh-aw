package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/workflow"
)

// UpdateWorkflows updates workflows from installed packages
func UpdateWorkflows(workflowName string, staged bool, verbose bool, workflowDir string) error {
	if verbose {
		if workflowName != "" {
			fmt.Printf("Updating workflow: %s\n", workflowName)
		} else {
			fmt.Printf("Updating all workflows from packages\n")
		}
		if staged {
			fmt.Printf("Running in staged mode - showing what would be updated without applying changes\n")
		}
	}

	// Validate and set default for workflow directory
	if workflowDir == "" {
		workflowDir = ".github/workflows"
	} else {
		// Ensure the path is relative
		if filepath.IsAbs(workflowDir) {
			return fmt.Errorf("workflow-dir must be a relative path, got: %s", workflowDir)
		}
		// Clean the path to avoid issues with ".." or other problematic elements
		workflowDir = filepath.Clean(workflowDir)
	}

	// Find git root for consistent behavior
	gitRoot, err := findGitRoot()
	if err != nil {
		return fmt.Errorf("update command requires being in a git repository: %w", err)
	}

	workflowsDir := filepath.Join(gitRoot, workflowDir)
	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		return fmt.Errorf("the %s directory does not exist in git root (%s)", workflowDir, gitRoot)
	}

	// Check both local and global packages
	locations := []bool{true, false} // local first, then global
	var allPackages []Package
	var updatedPackages []Package

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
			fmt.Printf("Checking %s packages for updates...\n", locationName)
		}

		// Find all installed packages
		packages, err := findInstalledPackages(packagesDir)
		if err != nil {
			if verbose {
				fmt.Printf("Warning: Failed to scan %s packages: %v\n", locationName, err)
			}
			continue
		}

		allPackages = append(allPackages, packages...)

		// Check each package for updates
		for _, pkg := range packages {
			hasUpdate, err := checkPackageForUpdates(pkg, verbose)
			if err != nil {
				if verbose {
					fmt.Printf("Warning: Failed to check updates for package %s: %v\n", pkg.Name, err)
				}
				continue
			}
			if hasUpdate {
				updatedPackages = append(updatedPackages, pkg)
			}
		}
	}

	if len(allPackages) == 0 {
		fmt.Println("No packages installed. Use '" + constants.CLIExtensionPrefix + " install <org/repo>' to install packages.")
		return nil
	}

	if len(updatedPackages) == 0 {
		fmt.Println("All packages are up to date.")
		return nil
	}

	// Filter packages by workflow if specified
	if workflowName != "" {
		filteredPackages := filterPackagesByWorkflow(updatedPackages, workflowName)
		if len(filteredPackages) == 0 {
			fmt.Printf("No packages contain workflow '%s' or no updates available for that workflow.\n", workflowName)
			return nil
		}
		updatedPackages = filteredPackages
	}

	if staged {
		// Show what would be updated
		fmt.Printf("The following packages would be updated:\n")
		for _, pkg := range updatedPackages {
			fmt.Printf("  - %s\n", pkg.Name)
			if verbose {
				shortCurrent := pkg.CommitSHA
				if len(shortCurrent) > 8 {
					shortCurrent = shortCurrent[:8]
				} else if shortCurrent == "" {
					shortCurrent = "unknown"
				}
				fmt.Printf("    Current: %s\n", shortCurrent)
				if latestSHA, err := getLatestCommitSHA(pkg.Name); err == nil {
					shortLatest := latestSHA
					if len(shortLatest) > 8 {
						shortLatest = shortLatest[:8]
					}
					fmt.Printf("    Latest:  %s\n", shortLatest)
				}
			}
		}
		fmt.Printf("\nRun without --staged to apply these updates.\n")
		return nil
	}

	// Perform the updates
	fmt.Printf("Updating %d package(s)...\n", len(updatedPackages))

	var recompiledWorkflows []string

	for i, pkg := range updatedPackages {
		fmt.Printf("Updating package %d/%d: %s\n", i+1, len(updatedPackages), pkg.Name)

		// Re-install the package to get latest version
		if err := InstallPackage(pkg.Name, isLocalPackage(pkg.Path), verbose); err != nil {
			fmt.Printf("Warning: Failed to update package %s: %v\n", pkg.Name, err)
			continue
		}

		// Find workflows from this package that are installed in the workflow directory
		installedWorkflows, err := findInstalledWorkflowsFromPackage(pkg, workflowsDir, workflowName)
		if err != nil {
			if verbose {
				fmt.Printf("Warning: Failed to find installed workflows from package %s: %v\n", pkg.Name, err)
			}
			continue
		}

		// Track which workflows we need to recompile
		recompiledWorkflows = append(recompiledWorkflows, installedWorkflows...)

		fmt.Printf("Updated package: %s\n", pkg.Name)
	}

	// Recompile affected workflows
	if len(recompiledWorkflows) > 0 {
		fmt.Printf("\nRecompiling %d affected workflow(s)...\n", len(recompiledWorkflows))

		// Create compiler
		compiler := workflow.NewCompiler(verbose, "", GetVersion())
		compiler.SetSkipValidation(false) // Enable validation for updates

		for _, workflowFile := range recompiledWorkflows {
			if verbose {
				fmt.Printf("Recompiling: %s\n", workflowFile)
			}
			if err := CompileWorkflowWithValidation(compiler, workflowFile, verbose); err != nil {
				fmt.Printf("Warning: Failed to recompile workflow %s: %v\n", workflowFile, err)
			}
		}

		fmt.Printf("Successfully recompiled %d workflow(s)\n", len(recompiledWorkflows))
	}

	fmt.Printf("Update completed successfully!\n")
	return nil
}