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

func findFirstCheckoutStepIndex(customSteps string) (int, bool) {
	if customSteps == "" {
		return 0, false
	}

	var wrapper struct {
		Steps []map[string]any `yaml:"steps"`
	}
	if err := yaml.Unmarshal([]byte(customSteps), &wrapper); err != nil {
		return 0, false
	}

	for i, step := range wrapper.Steps {
		uses, ok := step["uses"].(string)
		if ok && isCheckoutActionReference(uses) {
			compilerWorkflowHelpersLog.Print("Detected actions/checkout in custom steps")
			return i, true
		}
	}

	return 0, false
}

func isCheckoutActionReference(uses string) bool {
	normalized := strings.ToLower(strings.TrimSpace(strings.Trim(uses, `"'`)))
	return normalized == "actions/checkout" || strings.HasPrefix(normalized, "actions/checkout@")
}

// GetWorkflowIDFromPath extracts the workflow ID from a markdown file path.
// The workflow ID is the filename without the .md extension.
// Example: "/path/to/ai-moderator.md" -> "ai-moderator"
func GetWorkflowIDFromPath(markdownPath string) string {
	return strings.TrimSuffix(filepath.Base(markdownPath), ".md")
}
