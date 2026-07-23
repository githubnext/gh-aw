package workflow

import (
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"

	"github.com/github/gh-aw/pkg/logger"
)

var compilerWorkflowHelpersLog = logger.New("workflow:compiler_workflow_helpers")

// ContainsCheckout returns true if the given custom steps contain an actions/checkout step
func ContainsCheckout(customSteps string) bool {
	_, found := findFirstCheckoutStepIndex(customSteps)
	return found
}

// findFirstCheckoutStepIndex returns the zero-based index of the first
// actions/checkout step in customSteps and true, or (0, false) if none is found.
//
// It accepts both the wrapped form (with a "steps:" key) and the bare sequence
// form (a YAML list) that older call sites may produce.  If the YAML cannot be
// parsed in either form the function returns (0, false): the caller should not
// attempt checkout-step insertion when the step list is unparseable.
func findFirstCheckoutStepIndex(customSteps string) (int, bool) {
	if customSteps == "" {
		return 0, false
	}

	// Try the wrapped form first: "steps:\n  - ...\n"
	var wrapper struct {
		Steps []map[string]any `yaml:"steps"`
	}
	if err := yaml.Unmarshal([]byte(customSteps), &wrapper); err == nil {
		if len(wrapper.Steps) > 0 {
			for i, step := range wrapper.Steps {
				uses, ok := step["uses"].(string)
				if ok && isCheckoutActionReference(uses) {
					compilerWorkflowHelpersLog.Print("Detected actions/checkout in custom steps")
					return i, true
				}
			}
			return 0, false
		}
	}

	// Fall back to the bare sequence form: "- uses: ...\n"
	var steps []map[string]any
	if err := yaml.Unmarshal([]byte(customSteps), &steps); err != nil {
		// Malformed YAML: we cannot safely determine a checkout step index.
		return 0, false
	}

	for i, step := range steps {
		uses, ok := step["uses"].(string)
		if ok && isCheckoutActionReference(uses) {
			compilerWorkflowHelpersLog.Print("Detected actions/checkout in custom steps")
			return i, true
		}
	}

	return 0, false
}

func isCheckoutActionReference(uses string) bool {
	trimmed := strings.TrimSpace(strings.Trim(uses, `"'`))
	return strings.EqualFold(trimmed, "actions/checkout") ||
		strings.HasPrefix(strings.ToLower(trimmed), "actions/checkout@")
}

// GetWorkflowIDFromPath extracts the workflow ID from a markdown file path.
// The workflow ID is the filename without the .md extension.
// Example: "/path/to/ai-moderator.md" -> "ai-moderator"
func GetWorkflowIDFromPath(markdownPath string) string {
	return strings.TrimSuffix(filepath.Base(markdownPath), ".md")
}
