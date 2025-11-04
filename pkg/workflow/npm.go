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
// For general validation, see validation.go.
// For detailed documentation, see specs/validation-architecture.md
package workflow

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var npmLog = logger.New("workflow:npm")

// validateNpxPackages validates that npx packages are available on npm registry
func (c *Compiler) validateNpxPackages(workflowData *WorkflowData) error {
	packages := extractNpxPackages(workflowData)
	if len(packages) == 0 {
		npmLog.Print("No npx packages to validate")
		return nil
	}

	npmLog.Printf("Validating %d npx packages", len(packages))

	// Check if npm is available
	_, err := exec.LookPath("npm")
	if err != nil {
		npmLog.Print("npm command not found, cannot validate npx packages")
		return fmt.Errorf("npm command not found - cannot validate npx packages. Install Node.js/npm or disable validation")
	}

	var errors []string
	for _, pkg := range packages {
		npmLog.Printf("Validating npm package: %s", pkg)

		// Use npm view to check if package exists
		cmd := exec.Command("npm", "view", pkg, "name")
		output, err := cmd.CombinedOutput()

		if err != nil {
			npmLog.Printf("Package validation failed for %s: %v", pkg, err)
			errors = append(errors, fmt.Sprintf("npx package '%s' not found on npm registry: %s", pkg, strings.TrimSpace(string(output))))
		} else {
			npmLog.Printf("Package validated successfully: %s", pkg)
			if c.verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("✓ npm package validated: %s", pkg)))
			}
		}
	}

	if len(errors) > 0 {
		npmLog.Printf("npx package validation failed with %d errors", len(errors))
		return fmt.Errorf("npx package validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}

	npmLog.Print("All npx packages validated successfully")
	return nil
}

// extractNpxPackages extracts npx package names from workflow data
func extractNpxPackages(workflowData *WorkflowData) []string {
	return collectPackagesFromWorkflow(workflowData, extractNpxFromCommands, "npx")
}

// extractNpxFromCommands extracts npx package names from command strings
func extractNpxFromCommands(commands string) []string {
	var packages []string
	lines := strings.Split(commands, "\n")

	for _, line := range lines {
		// Look for "npx <package>" pattern
		words := strings.Fields(line)
		for i, word := range words {
			if word == "npx" && i+1 < len(words) {
				// Skip flags and find the first package name
				for j := i + 1; j < len(words); j++ {
					pkg := words[j]
					pkg = strings.TrimRight(pkg, "&|;")
					// Skip flags (start with - or --)
					if !strings.HasPrefix(pkg, "-") {
						packages = append(packages, pkg)
						break
					}
				}
			}
		}
	}

	return packages
}
