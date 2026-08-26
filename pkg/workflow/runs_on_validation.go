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
	"errors"
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
	runsOnValidationLog.Printf("Validating runs-on configuration")

	type runnerField struct {
		name  string
		value any
	}

	runsOnFields := []runnerField{
		{name: "runs-on", value: frontmatter["runs-on"]},
		{name: "runs-on-slim", value: frontmatter["runs-on-slim"]},
	}
	if safeOutputs, ok := frontmatter["safe-outputs"].(map[string]any); ok {
		runsOnFields = append(runsOnFields, runnerField{name: "safe-outputs.runs-on", value: safeOutputs["runs-on"]})
		if threatDetection, ok := safeOutputs["threat-detection"].(map[string]any); ok {
			runsOnFields = append(runsOnFields, runnerField{name: "safe-outputs.threat-detection.runs-on", value: threatDetection["runs-on"]})
		}
		if jobs, ok := safeOutputs["jobs"].(map[string]any); ok {
			for jobName, jobValue := range jobs {
				if job, ok := jobValue.(map[string]any); ok {
					runsOnFields = append(runsOnFields, runnerField{
						name:  fmt.Sprintf("safe-outputs.jobs.%s.runs-on", jobName),
						value: job["runs-on"],
					})
				}
			}
		}
	}

	// The apple-container sandbox runtime is the one case where a macOS runner is
	// required rather than forbidden, and only for the agent job's own runs-on.
	// Every other runner field (runs-on-slim and the safe-output job runners) keeps
	// the blanket macOS rejection because those jobs still need Linux containers.
	//
	// Only this workflow's own frontmatter is visible here; runs-on and the sandbox
	// runtime can also arrive from an import. That is safe in both directions:
	// validateSandboxConfig re-validates the merged runner for every apple-container
	// workflow, so an omitted local runs-on is deferred rather than rejected here,
	// and a runtime that arrives only from an import still gets its host requirement
	// enforced after merging.
	appleContainer := frontmatterSelectsAppleContainer(frontmatter)

	for _, field := range runsOnFields {
		labels := extractRunnerLabels(field.value)
		if appleContainer && field.name == "runs-on" {
			if isEmptyRunsOnValue(field.value) {
				// Defer to the post-merge check, which can see an imported runs-on.
				continue
			}
			if err := validateAppleContainerRunnerLabels(labels, true); err != nil {
				runsOnValidationLog.Printf("apple-container runs-on validation failed: %v", err)
				return err
			}
			continue
		}
		for _, label := range labels {
			lower := strings.ToLower(label)
			if strings.HasPrefix(lower, "macos-") || strings.EqualFold(lower, "macos") {
				return formatCompilerError(markdownPath, "error",
					fmt.Sprintf("%s includes unsupported runner '%s'.\n\n"+
						"Agentic workflows require Linux containers and container jobs. Use a Linux runner label or runner-group configuration instead.\n\n"+
						"Example: runs-on: [self-hosted, linux, x64]\n\n"+
						"The only exception is sandbox.agent.runtime: apple-container, which requires %s declared in this workflow's own frontmatter.\n\n"+
						"See %s for details.",
						field.name, label, appleContainerRunnerExample, macOSRunnerFAQURL), nil)
			}
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
				return fmt.Errorf("runs-on array entry has type %T, expected a string label. Example: runs-on: [ubuntu-latest, self-hosted]", label)
			}
		}
		return nil
	case map[string]any:
		if len(v) == 0 {
			return errors.New("runs-on object is empty. Expected an object with 'group' or 'labels'. Example: runs-on:\n  group: my-runner-group")
		}
		for key, value := range v {
			switch key {
			case "group":
				if _, ok := value.(string); !ok {
					return fmt.Errorf("runs-on.group has type %T, expected a string name. Example: runs-on:\n  group: my-runner-group", value)
				}
			case "labels":
				labels, ok := value.([]any)
				if !ok {
					return fmt.Errorf("runs-on.labels has type %T, expected an array of string labels. Example: runs-on:\n  labels: [self-hosted, linux]", value)
				}
				for _, label := range labels {
					if _, ok := label.(string); !ok {
						return fmt.Errorf("runs-on.labels entry has type %T, expected a string label. Example: runs-on:\n  labels: [self-hosted, linux]", label)
					}
				}
			default:
				return fmt.Errorf("runs-on object key '%s' is not supported, expected 'group' or 'labels'. Example: runs-on:\n  group: my-runner-group\n  labels: [self-hosted]", key)
			}
		}
		return nil
	default:
		return fmt.Errorf("runs-on has type %T, expected a string, array of strings, or object. Example: runs-on: ubuntu-latest", value)
	}
}

func isEmptyRunsOnValue(value any) bool {
	switch v := value.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(v) == ""
	case []any:
		if len(v) == 0 {
			return true
		}
		for _, label := range v {
			label, ok := label.(string)
			if !ok || strings.TrimSpace(label) != "" {
				return false
			}
		}
		return true
	case map[string]any:
		if len(v) == 0 {
			return true
		}

		group, hasGroup := v["group"].(string)
		labels, hasLabels := v["labels"].([]any)
		if !hasGroup && !hasLabels {
			return false
		}
		return strings.TrimSpace(group) == "" && len(labels) == 0
	default:
		return false
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
