//go:build !js && !wasm

// This file provides NPM package validation for agentic workflows.
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
//   - Returns ErrNpmNotAvailable (treated as a warning) when npm is not installed
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
// For detailed documentation, see scratchpad/validation-architecture.md

package workflow

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
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

	if err := validateNpxPackageNames(packages); err != nil {
		return err
	}

	// Check if npm is available
	_, err := exec.LookPath("npm")
	if err != nil {
		npmValidationLog.Print("npm command not found, cannot validate npx packages")
		return ErrNpmNotAvailable
	}

	errors := c.validateNpxPackagesOnRegistry(packages)
	if len(errors) > 0 {
		npmValidationLog.Printf("npx package validation failed with %d errors", len(errors))
		return NewValidationError(
			"npx.packages",
			fmt.Sprintf("%d packages not found", len(errors)),
			"npx packages not found on npm registry",
			fmt.Sprintf("Fix package names or verify they exist on npm:\n\n%s\n\nCheck package availability:\n$ npm view <package-name>\n\nSearch for similar packages:\n$ npm search <keyword>", strings.Join(errors, "\n")),
		)
	}

	npmValidationLog.Print("All npx packages validated successfully")
	return nil
}

func validateNpxPackageNames(packages []string) error {
	if err := rejectHyphenPrefixPackages(packages, "npx"); err != nil {
		npmValidationLog.Printf("npx package name validation failed: %v", err)
		return err
	}
	for _, pkg := range packages {
		if err := validateNpmPackageName(pkg); err != nil {
			npmValidationLog.Printf("npm package name validation failed: %v", err)
			return NewValidationError(
				"npx.packages",
				"invalid npm package name",
				err.Error(),
				fmt.Sprintf("npm package names must match @scope/name or name (lowercase alphanumeric, hyphens, dots, underscores).\n\nInvalid name: %q", pkg),
			)
		}
	}
	return nil
}

func (c *Compiler) validateNpxPackagesOnRegistry(packages []string) []string {
	var errors []string
	for _, pkg := range packages {
		npmValidationLog.Printf("Validating npm package: %s", pkg)

		// Use npm view to check if package exists.
		// Pass -- to prevent the package name being interpreted as a flag (argument injection defence).
		cmd := exec.Command("npm", "view", "--", pkg, "name")
		output, err := cmd.CombinedOutput()

		if err != nil {
			npmValidationLog.Printf("Package validation failed for %s: %v", pkg, err)
			errors = append(errors, fmt.Sprintf("npx package '%s' not found on npm registry: %s", pkg, strings.TrimSpace(string(output))))
		} else {
			npmValidationLog.Printf("Package validated successfully: %s", pkg)
			if c.verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("✓ npm package validated: "+pkg))
			}
		}
	}
	return errors
}
