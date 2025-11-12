// Package workflow provides generic package extraction utilities for agentic workflows.
//
// # Package Extraction Framework
//
// This file provides a generic framework for extracting package names from command strings.
// The PackageExtractor type can be configured to handle different package managers
// (npm, pip, uv, go, etc.) with minimal code duplication.
//
// # Usage Example
//
//	extractor := PackageExtractor{
//	    CommandNames:       []string{"pip", "pip3"},
//	    RequiredSubcommand: "install",
//	    TrimSuffixes:       "&|;",
//	}
//	packages := extractor.ExtractPackages("pip install requests")
//	// Returns: []string{"requests"}
//
// For package-specific extraction, see npm.go, pip.go, and dependabot.go.
// For validation, see validation.go.
package workflow

import (
	"strings"
)

// PackageExtractor provides a configurable framework for extracting package names
// from command-line strings. It can be configured to handle different package
// managers (npm, pip, uv, go) by setting the appropriate command names and options.
type PackageExtractor struct {
	// CommandNames is the list of command names to look for (e.g., ["pip", "pip3"])
	CommandNames []string

	// RequiredSubcommand is the subcommand that must follow the command name
	// (e.g., "install" for pip). If empty, the package name is expected immediately
	// after the command name (e.g., "npx <package>").
	RequiredSubcommand string

	// TrimSuffixes is a string of characters to trim from the end of package names
	// (e.g., "&|;" for shell operators)
	TrimSuffixes string
}

// ExtractPackages extracts package names from command strings using the configured
// extraction rules. It processes multi-line command strings and returns all found
// package names.
//
// The extraction process:
//  1. Split commands by newlines
//  2. Split each line into words
//  3. Find command name matches
//  4. If RequiredSubcommand is set, look for that subcommand
//  5. Skip flags (words starting with -)
//  6. Extract package name and trim configured suffixes
//  7. Return first package found per command invocation
//
// Example usage:
//
//	extractor := PackageExtractor{
//	    CommandNames: []string{"pip", "pip3"},
//	    RequiredSubcommand: "install",
//	    TrimSuffixes: "&|;",
//	}
//	packages := extractor.ExtractPackages("pip install requests==2.28.0")
//	// Returns: []string{"requests==2.28.0"}
func (pe *PackageExtractor) ExtractPackages(commands string) []string {
	var packages []string
	lines := strings.Split(commands, "\n")

	for _, line := range lines {
		words := strings.Fields(line)
		for i, word := range words {
			// Check if this word matches one of our command names
			if !pe.isCommandName(word) {
				continue
			}

			// If we have a required subcommand, find it first
			if pe.RequiredSubcommand != "" {
				pkg := pe.extractWithSubcommand(words, i)
				if pkg != "" {
					packages = append(packages, pkg)
				}
			} else {
				// No subcommand required - package comes directly after command
				pkg := pe.extractDirectPackage(words, i)
				if pkg != "" {
					packages = append(packages, pkg)
				}
			}
		}
	}

	return packages
}

// isCommandName checks if the given word matches any of the configured command names
func (pe *PackageExtractor) isCommandName(word string) bool {
	for _, cmdName := range pe.CommandNames {
		if word == cmdName {
			return true
		}
	}
	return false
}

// extractWithSubcommand extracts a package name when a required subcommand must be present
// (e.g., "pip install <package>")
func (pe *PackageExtractor) extractWithSubcommand(words []string, commandIndex int) string {
	// Look for the required subcommand after the command name
	for j := commandIndex + 1; j < len(words); j++ {
		if words[j] == pe.RequiredSubcommand {
			// Found the subcommand - now find the package name
			return pe.findPackageName(words, j+1)
		}
	}
	return ""
}

// extractDirectPackage extracts a package name that comes directly after the command
// (e.g., "npx <package>")
func (pe *PackageExtractor) extractDirectPackage(words []string, commandIndex int) string {
	if commandIndex+1 >= len(words) {
		return ""
	}
	return pe.findPackageName(words, commandIndex+1)
}

// findPackageName finds and processes the package name starting at the given index.
// It skips flags (words starting with -) and returns the first non-flag word,
// trimming configured suffixes.
//
// This method is exported to allow special-case extraction patterns (like uv)
// to reuse the package finding logic.
func (pe *PackageExtractor) FindPackageName(words []string, startIndex int) string {
	return pe.findPackageName(words, startIndex)
}

// findPackageName is the internal implementation of FindPackageName
func (pe *PackageExtractor) findPackageName(words []string, startIndex int) string {
	for i := startIndex; i < len(words); i++ {
		pkg := words[i]
		// Skip flags (start with - or --)
		if strings.HasPrefix(pkg, "-") {
			continue
		}
		// Trim configured suffixes (e.g., shell operators)
		if pe.TrimSuffixes != "" {
			pkg = strings.TrimRight(pkg, pe.TrimSuffixes)
		}
		return pkg
	}
	return ""
}
