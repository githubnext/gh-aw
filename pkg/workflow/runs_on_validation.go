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
//   - validateRunsOnValue() - Validates the supported runs-on YAML value shapes
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
)

var runsOnValidationLog = newValidationLogger("runs_on")

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

func validateRunsOnValue(value any) error {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case string:
		return nil
	case []any:
		for _, label := range v {
			if _, ok := label.(string); !ok {
				return fmt.Errorf("invalid runs-on array entry type %T: expected string", label)
			}
		}
		return nil
	case map[string]any:
		for key, value := range v {
			switch key {
			case "group":
				if _, ok := value.(string); !ok {
					return fmt.Errorf("invalid runs-on.group type %T: expected string", value)
				}
			case "labels":
				labels, ok := value.([]any)
				if !ok {
					return fmt.Errorf("invalid runs-on.labels type %T: expected array of strings", value)
				}
				for _, label := range labels {
					if _, ok := label.(string); !ok {
						return fmt.Errorf("invalid runs-on.labels entry type %T: expected string", label)
					}
				}
			default:
				return fmt.Errorf("invalid runs-on object key %q: expected only group or labels", key)
			}
		}
		return nil
	default:
		return fmt.Errorf("invalid runs-on type %T: expected string, array of strings, or object", value)
	}
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
		labels = parseStringSliceAny(v, nil)
	case map[string]any:
		if labelsVal, ok := v["labels"]; ok {
			labels = parseStringSliceAny(labelsVal, nil)
		}
	}

	return labels
}
