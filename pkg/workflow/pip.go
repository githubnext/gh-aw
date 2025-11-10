// Package workflow provides Python package extraction for agentic workflows.
//
// # Python Package Extraction
//
// This file extracts Python package names from workflow configurations using pip and uv
// package managers. Extraction functions parse commands and configuration to identify
// packages that will be installed at runtime.
//
// # Extraction Functions
//
//   - extractPipPackages() - Extracts pip packages from workflow configuration
//   - extractPipFromCommands() - Extracts pip packages from command strings
//   - extractUvPackages() - Extracts uv packages from workflow configuration
//   - extractUvFromCommands() - Extracts uv packages from command strings
//
// # When to Add Extraction Here
//
// Add extraction to this file when:
//   - It parses Python/pip ecosystem package names
//   - It identifies packages from shell commands
//   - It extracts packages from workflow steps
//   - It detects uv package manager usage
//
// For package validation functions, see pip_validation.go.
// For general validation, see validation.go.
// For detailed documentation, see specs/validation-architecture.md
package workflow

import (
	"strings"
)

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
