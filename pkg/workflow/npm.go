// Package workflow provides NPM package extraction utilities for agentic workflows.
//
// # NPM Package Extraction
//
// This file provides utilities to extract NPM package names from workflow data
// for packages used with npx (Node Package Execute). The extracted packages
// can be validated by the validation functions in validation.go.
//
// # Extraction Functions
//
//   - extractNpxPackages() - Extracts npm packages used with npx launcher
//   - extractNpxFromCommands() - Parses command strings to find npx packages
//
// For package validation, see validation.go.
// For detailed documentation, see specs/validation-architecture.md
package workflow

import (
	"strings"
)

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
