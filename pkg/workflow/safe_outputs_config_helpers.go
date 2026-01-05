package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
)

// ========================================
// Safe Output Configuration Helpers
// ========================================

// formatSafeOutputsRunsOn formats the runs-on value from SafeOutputsConfig for job output
func (c *Compiler) formatSafeOutputsRunsOn(safeOutputs *SafeOutputsConfig) string {
	if safeOutputs == nil || safeOutputs.RunsOn == "" {
		return fmt.Sprintf("runs-on: %s", constants.DefaultActivationJobRunnerImage)
	}

	return fmt.Sprintf("runs-on: %s", safeOutputs.RunsOn)
}

// HasSafeOutputsEnabled checks if any safe-outputs are enabled
func HasSafeOutputsEnabled(safeOutputs *SafeOutputsConfig) bool {
	enabled := hasAnySafeOutputEnabled(safeOutputs)

	if safeOutputsConfigLog.Enabled() {
		safeOutputsConfigLog.Printf("Safe outputs enabled check: %v", enabled)
	}

	return enabled
}

// GetEnabledSafeOutputToolNames returns a list of enabled safe output tool names
// that can be used in the prompt to inform the agent which tools are available
func GetEnabledSafeOutputToolNames(safeOutputs *SafeOutputsConfig) []string {
	tools := getEnabledSafeOutputToolNamesReflection(safeOutputs)

	if safeOutputsConfigLog.Enabled() {
		safeOutputsConfigLog.Printf("Enabled safe output tools: %v", tools)
	}

	return tools
}

// normalizeSafeOutputIdentifier converts dashes to underscores for safe output identifiers.
//
// This is a NORMALIZE function (format standardization pattern). Use this when ensuring
// consistency across the system while remaining resilient to LLM-generated variations.
//
// Safe output identifiers may appear in different formats:
//   - YAML configuration: "create-issue" (dash-separated)
//   - JavaScript code: "create_issue" (underscore-separated)
//   - Internal usage: can vary based on source
//
// This function normalizes all variations to a canonical underscore-separated format,
// ensuring consistent internal representation regardless of input format.
//
// Example inputs and outputs:
//
//	normalizeSafeOutputIdentifier("create-issue")      // returns "create_issue"
//	normalizeSafeOutputIdentifier("create_issue")      // returns "create_issue" (unchanged)
//	normalizeSafeOutputIdentifier("add-comment")       // returns "add_comment"
//
// Note: This function assumes the input is already a valid identifier. It does NOT
// perform character validation or sanitization - it only converts between naming
// conventions. Both dash-separated and underscore-separated formats are valid;
// this function simply standardizes to the internal representation.
//
// See package documentation for guidance on when to use sanitize vs normalize patterns.
func normalizeSafeOutputIdentifier(identifier string) string {
	normalized := strings.ReplaceAll(identifier, "-", "_")
	if safeOutputsConfigLog.Enabled() {
		safeOutputsConfigLog.Printf("Normalized safe output identifier: %s -> %s", identifier, normalized)
	}
	return normalized
}
