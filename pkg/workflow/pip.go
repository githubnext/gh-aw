package workflow

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var pipLog = logger.New("workflow:pip")

// validatePythonPackagesWithPip is a generic helper that validates Python packages using pip index.
// It accepts a package list, package type name for error messaging, and pip command to use.
func (c *Compiler) validatePythonPackagesWithPip(packages []string, packageType string, pipCmd string) {
	pipLog.Printf("Validating %d %s packages using %s", len(packages), packageType, pipCmd)

	for _, pkg := range packages {
		// Extract package name without version specifier
		pkgName := pkg
		if eqIndex := strings.Index(pkg, "=="); eqIndex > 0 {
			pkgName = pkg[:eqIndex]
		}

		pipLog.Printf("Validating %s package: %s", packageType, pkgName)

		// Use pip index to check if package exists on PyPI
		cmd := exec.Command(pipCmd, "index", "versions", pkgName)
		output, err := cmd.CombinedOutput()

		if err != nil {
			outputStr := strings.TrimSpace(string(output))
			pipLog.Printf("Package validation failed for %s: %v", pkg, err)
			// Treat all pip validation errors as warnings, not compilation failures
			// The package may be experimental, not yet published, or will be installed at runtime
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("%s package '%s' validation failed - skipping verification. Package may or may not exist on PyPI.", packageType, pkg)))
			if c.verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("  Details: %s", outputStr)))
			}
		} else {
			pipLog.Printf("Package validated successfully: %s", pkg)
			if c.verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("✓ %s package validated: %s", packageType, pkg)))
			}
		}
	}
}

// validatePipPackages validates that pip packages are available on PyPI
func (c *Compiler) validatePipPackages(workflowData *WorkflowData) error {
	packages := extractPipPackages(workflowData)
	if len(packages) == 0 {
		pipLog.Print("No pip packages to validate")
		return nil
	}

	pipLog.Printf("Starting pip package validation for %d packages", len(packages))

	// Check if pip is available
	pipCmd := "pip"
	_, err := exec.LookPath("pip")
	if err != nil {
		// Try pip3 as fallback
		_, err3 := exec.LookPath("pip3")
		if err3 != nil {
			pipLog.Print("pip command not found, skipping validation")
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("pip command not found - skipping pip package validation. Install Python/pip for full validation"))
			return nil
		}
		pipCmd = "pip3"
		pipLog.Print("Using pip3 command for validation")
	}

	c.validatePythonPackagesWithPip(packages, "pip", pipCmd)
	return nil
}

// validateUvPackages validates that uv packages are available
func (c *Compiler) validateUvPackages(workflowData *WorkflowData) error {
	packages := extractUvPackages(workflowData)
	if len(packages) == 0 {
		pipLog.Print("No uv packages to validate")
		return nil
	}

	pipLog.Printf("Starting uv package validation for %d packages", len(packages))

	// Check if uv is available
	_, err := exec.LookPath("uv")
	if err != nil {
		pipLog.Print("uv command not found, falling back to pip validation")
		// uv not available, but we can still validate using pip index
		pipCmd := "pip"
		_, pipErr := exec.LookPath("pip")
		if pipErr != nil {
			// Try pip3 as fallback
			_, pip3Err := exec.LookPath("pip3")
			if pip3Err != nil {
				pipLog.Print("Neither uv nor pip commands found, cannot validate")
				return fmt.Errorf("uv and pip commands not found - cannot validate uv packages. Install uv/pip or disable validation")
			}
			pipCmd = "pip3"
			pipLog.Print("Using pip3 for validation")
		}

		return c.validateUvPackagesWithPip(packages, pipCmd)
	}

	pipLog.Print("Using uv command for validation")

	// Validate with uv
	var errors []string
	for _, pkg := range packages {
		// Extract package name without version specifier
		pkgName := pkg
		if eqIndex := strings.Index(pkg, "=="); eqIndex > 0 {
			pkgName = pkg[:eqIndex]
		}

		// Use uv pip show to check if package exists on PyPI
		cmd := exec.Command("uv", "pip", "show", pkgName, "--no-cache")
		_, err := cmd.CombinedOutput()

		if err != nil {
			// Package not installed, try to check if it's available
			errors = append(errors, fmt.Sprintf("uv package '%s' validation requires network access or local cache", pkg))
		} else if c.verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("✓ uv package validated: %s", pkg)))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("uv package validation requires network access:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

// validateUvPackagesWithPip validates uv packages using pip index
func (c *Compiler) validateUvPackagesWithPip(packages []string, pipCmd string) error {
	c.validatePythonPackagesWithPip(packages, "uv", pipCmd)
	return nil
}

// extractPipPackages extracts pip package names from workflow data
func extractPipPackages(workflowData *WorkflowData) []string {
	return collectPackagesFromWorkflow(workflowData, extractPipFromCommands, "")
}

// extractPipFromCommands extracts pip package names from command strings
func extractPipFromCommands(commands string) []string {
	var packages []string
	lines := strings.Split(commands, "\n")

	for _, line := range lines {
		// Look for "pip install <package>" or "pip3 install <package>" patterns
		words := strings.Fields(line)
		for i, word := range words {
			if (word == "pip" || word == "pip3") && i+1 < len(words) {
				// Look for install command
				for j := i + 1; j < len(words); j++ {
					if words[j] == "install" {
						// Skip flags and find the first package name
						for k := j + 1; k < len(words); k++ {
							pkg := words[k]
							pkg = strings.TrimRight(pkg, "&|;")
							// Skip flags (start with - or --)
							if !strings.HasPrefix(pkg, "-") {
								packages = append(packages, pkg)
								break
							}
						}
						break
					}
				}
			}
		}
	}

	return packages
}

// extractUvPackages extracts uv package names from workflow data
func extractUvPackages(workflowData *WorkflowData) []string {
	return collectPackagesFromWorkflow(workflowData, extractUvFromCommands, "")
}

// extractUvFromCommands extracts uv package names from command strings
func extractUvFromCommands(commands string) []string {
	var packages []string
	lines := strings.Split(commands, "\n")

	for _, line := range lines {
		// Look for "uv pip install <package>" or "uvx <package>" patterns
		words := strings.Fields(line)
		for i, word := range words {
			if word == "uvx" && i+1 < len(words) {
				pkg := words[i+1]
				pkg = strings.TrimRight(pkg, "&|;")
				packages = append(packages, pkg)
			} else if word == "uv" && i+2 < len(words) && words[i+1] == "pip" {
				// Look for install command
				for j := i + 2; j < len(words); j++ {
					if words[j] == "install" {
						// Skip flags and find the first package name
						for k := j + 1; k < len(words); k++ {
							pkg := words[k]
							pkg = strings.TrimRight(pkg, "&|;")
							// Skip flags (start with - or --)
							if !strings.HasPrefix(pkg, "-") {
								packages = append(packages, pkg)
								break
							}
						}
						break
					}
				}
			}
		}
	}

	return packages
}
