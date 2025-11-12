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

// extractNpxPackages extracts npx package names from workflow data
func extractNpxPackages(workflowData *WorkflowData) []string {
	return collectPackagesFromWorkflow(workflowData, extractNpxFromCommands, "npx")
}

// extractNpxFromCommands extracts npx package names from command strings
func extractNpxFromCommands(commands string) []string {
	extractor := PackageExtractor{
		CommandNames:       []string{"npx"},
		RequiredSubcommand: "",
		TrimSuffixes:       "&|;",
	}
	return extractor.ExtractPackages(commands)
}
