// This file provides validation for the runs-on field in agentic workflows.
//
// # Runner Type Validation
//
// This file validates that the runs-on field in workflow frontmatter does not
// specify runner types that are incompatible with agentic workflows. Specifically,
// macOS runners are not supported because agentic workflows rely on containers to
// provide a secure sandbox, and GitHub-hosted macOS runners do not support container
// jobs which are required for the Agent Workflow Firewall.
//
// # Validation Functions
//
//   - validateRunsOn() - Validates the runs-on field for unsupported runner types
//   - extractRunnerLabels() - Extracts individual runner labels from runs-on value
//
// # When to Add Validation Here
//
// Add validation to this file when:
//   - Adding new runner type restrictions
//   - Detecting additional unsupported runner configurations
//   - Improving error messages for runner selection

package workflow

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var runsOnValidationLog = logger.New("workflow:runs_on_validation")

// macOSRunnerFAQURL is the URL to the FAQ entry explaining why macOS runners are not supported.
const macOSRunnerFAQURL = "https://github.github.com/gh-aw/reference/faq/#why-are-macos-runners-not-supported"

// validateRunsOn validates that the runs-on field does not specify macOS runners,
// which are not supported in agentic workflows because they do not support
// container jobs required for the Agent Workflow Firewall sandbox.
//
// Returns an error with a FAQ link if a macOS runner is detected, nil otherwise.
func validateRunsOn(frontmatter map[string]any, markdownPath string) error {
	runsOn, exists := frontmatter["runs-on"]
	if !exists {
		return nil
	}

	runsOnValidationLog.Printf("Validating runs-on configuration")

	labels := extractRunnerLabels(runsOn)
	for _, label := range labels {
		lower := strings.ToLower(label)
		if strings.HasPrefix(lower, "macos-") || lower == "macos" {
			return formatCompilerError(markdownPath, "error",
				fmt.Sprintf("runner '%s' is not supported in agentic workflows.\n\n"+
					"macOS runners are not supported because agentic workflows rely on containers "+
					"for the secure Agent Workflow Firewall sandbox, and GitHub-hosted macOS runners "+
					"do not support container jobs.\n\n"+
					"Use 'ubuntu-latest' (default) or another Linux-based runner instead.\n\n"+
					"See %s for details.",
					label, macOSRunnerFAQURL), nil)
		}
	}

	runsOnValidationLog.Printf("runs-on validation passed")
	return nil
}

// extractRunnerLabels extracts individual runner label strings from a runs-on value.
// Handles all supported GitHub Actions runs-on forms:
//   - string: "ubuntu-latest"
//   - array: ["self-hosted", "linux"]
//   - object with labels: {group: "...", labels: ["linux"]}
func extractRunnerLabels(runsOn any) []string {
	var labels []string

	switch v := runsOn.(type) {
	case string:
		labels = append(labels, v)
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				labels = append(labels, s)
			}
		}
	case map[string]any:
		if labelsVal, ok := v["labels"]; ok {
			if labelsArr, ok := labelsVal.([]any); ok {
				for _, item := range labelsArr {
					if s, ok := item.(string); ok {
						labels = append(labels, s)
					}
				}
			}
		}
	}

	return labels
}
