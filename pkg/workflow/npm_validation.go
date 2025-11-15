// Package workflow provides NPM package validation for agentic workflows.
//
// # NPM Package Validation
//
// This file validates NPM package availability on the npm registry for packages
// used with npx (Node Package Execute). Validation ensures that Node.js packages
// specified in workflows exist and can be installed at runtime.
//
// # Validation Functions
//
//   - validateNpxPackages() - Validates npm packages used with npx launcher
//
// # Validation Pattern: External Registry Check
//
// NPM package validation queries the npm registry using the npm CLI:
//   - Uses `npm view <package> name` to check package existence
//   - Returns hard errors if packages don't exist (unlike pip validation)
//   - Requires npm to be installed on the system
//
// # When to Add Validation Here
//
// Add validation to this file when:
//   - It validates Node.js/npm ecosystem packages
//   - It checks npm registry package existence
//   - It validates npx launcher packages
//   - It validates Node.js version compatibility
//
// For package extraction functions, see npm.go.
// For general validation, see validation.go.
// For detailed documentation, see specs/validation-architecture.md
package workflow

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var npmValidationLog = logger.New("workflow:npm_validation")

// validateNpxPackages validates that npx packages are available on npm registry
func (c *Compiler) validateNpxPackages(workflowData *WorkflowData) error {
	packages := extractNpxPackages(workflowData)
	if len(packages) == 0 {
		npmValidationLog.Print("No npx packages to validate")
		return nil
	}

	npmValidationLog.Printf("Validating %d npx packages", len(packages))

	// Check if npm is available
	_, err := exec.LookPath("npm")
	if err != nil {
		npmValidationLog.Print("npm command not found, cannot validate npx packages")
		return fmt.Errorf("npm command not found - cannot validate npx packages. Install Node.js/npm or disable validation")
	}

	var errors []string
	for _, pkg := range packages {
		npmValidationLog.Printf("Validating npm package: %s", pkg)

		// Use npm view to check if package exists
		cmd := exec.Command("npm", "view", pkg, "name")
		output, err := cmd.CombinedOutput()

		if err != nil {
			npmValidationLog.Printf("Package validation failed for %s: %v", pkg, err)
			errors = append(errors, fmt.Sprintf("npx package '%s' not found on npm registry: %s", pkg, strings.TrimSpace(string(output))))
		} else {
			npmValidationLog.Printf("Package validated successfully: %s", pkg)
			if c.verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("✓ npm package validated: %s", pkg)))
			}
		}
	}

	if len(errors) > 0 {
		npmValidationLog.Printf("npx package validation failed with %d errors", len(errors))
		return fmt.Errorf("npx package validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}

	npmValidationLog.Print("All npx packages validated successfully")
	return nil
}

// validatePlaywrightVersion validates that Playwright package version matches the pinned constant
// This ensures consistent Playwright versions across all workflows
func (c *Compiler) validatePlaywrightVersion(workflowData *WorkflowData) error {
	packages := extractNpxPackages(workflowData)
	if len(packages) == 0 {
		return nil
	}

	npmValidationLog.Print("Checking Playwright package versions")

	var warnings []string
	for _, pkg := range packages {
		// Check if this is a Playwright package
		if !strings.HasPrefix(pkg, "@playwright/mcp") {
			continue
		}

		// Parse the package version
		parts := strings.Split(pkg, "@")
		if len(parts) < 3 {
			// No version specified - this is fine, will use default
			continue
		}

		version := parts[len(parts)-1]

		// Check if version matches pinned constant or is "latest"
		if version != constants.DefaultPlaywrightVersion && version != "latest" {
			npmValidationLog.Printf("Playwright version mismatch: %s (expected: %s)", pkg, constants.DefaultPlaywrightVersion)
			warnings = append(warnings, fmt.Sprintf(
				"Playwright package '%s' version does not match pinned version '%s'. Consider updating to '@playwright/mcp@%s' for consistency",
				pkg, constants.DefaultPlaywrightVersion, constants.DefaultPlaywrightVersion,
			))
		} else if version == "latest" {
			npmValidationLog.Printf("Playwright using 'latest' tag, recommend pinning to: %s", constants.DefaultPlaywrightVersion)
			if c.verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf(
					"⚠ Playwright package '%s' uses 'latest' tag. Consider pinning to version %s for reproducibility",
					pkg, constants.DefaultPlaywrightVersion,
				)))
			}
		} else {
			npmValidationLog.Printf("Playwright version validated: %s", pkg)
			if c.verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("✓ Playwright version validated: %s", pkg)))
			}
		}
	}

	if len(warnings) > 0 {
		npmValidationLog.Printf("Playwright version validation completed with %d warnings", len(warnings))
		for _, warning := range warnings {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(warning))
		}
	}

	return nil
}
