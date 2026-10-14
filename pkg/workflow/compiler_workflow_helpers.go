package workflow

import (
	"path"
	"path/filepath"
	"strconv"
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

// customCheckoutHasRootGoMod checks the first checkout, where runtime setup is inserted.
func customCheckoutHasRootGoMod(customSteps string) bool {
	return checkoutStepsHaveRootGoMod(customSteps, true)
}

func checkoutStepsHaveRootGoMod(checkoutSteps string, firstCheckoutOnly bool) bool {
	var wrapper struct {
		Steps []struct {
			Uses string `yaml:"uses"`
			If   any    `yaml:"if"`
			With struct {
				Path           string `yaml:"path"`
				Repository     string `yaml:"repository"`
				Ref            string `yaml:"ref"`
				SparseCheckout string `yaml:"sparse-checkout"`
				ConeMode       string `yaml:"sparse-checkout-cone-mode"`
			} `yaml:"with"`
		} `yaml:"steps"`
	}
	if err := yaml.Unmarshal([]byte(checkoutSteps), &wrapper); err != nil || len(wrapper.Steps) == 0 {
		if err := yaml.Unmarshal([]byte(checkoutSteps), &wrapper.Steps); err != nil {
			return false
		}
	}
	hasRootGoMod := false
	for _, step := range wrapper.Steps {
		if !isCheckoutActionReference(step.Uses) {
			continue
		}
		atRoot := step.With.Path == "" || filepath.Clean(step.With.Path) == "."
		if atRoot || firstCheckoutOnly {
			repository := strings.Join(strings.Fields(step.With.Repository), "")
			hasRootGoMod = atRoot &&
				(repository == "" || repository == "${{github.repository}}") &&
				step.If == nil && step.With.Ref == "" &&
				sparseCheckoutIncludesRootGoMod(step.With.SparseCheckout, step.With.ConeMode)
		}
		if firstCheckoutOnly {
			break
		}
	}
	return hasRootGoMod
}

func sparseCheckoutIncludesRootGoMod(patterns, coneMode string) bool {
	if coneMode == "" {
		return true
	}
	cone, err := strconv.ParseBool(coneMode)
	if err != nil {
		return false
	}
	if cone {
		return true
	}
	if strings.Contains(patterns, "${{") {
		return false
	}
	if patterns == "" {
		return true
	}
	included := false
	for pattern := range strings.SplitSeq(patterns, "\n") {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" || strings.HasPrefix(pattern, "#") {
			continue
		}
		// Git supports character classes that path.Match cannot interpret.
		if strings.Contains(pattern, "[:") || strings.Contains(pattern, "[!") {
			return false
		}
		excluded := strings.HasPrefix(pattern, "!")
		pattern = strings.TrimPrefix(strings.TrimPrefix(pattern, "!"), "/")
		for strings.HasPrefix(pattern, "**/") {
			pattern = strings.TrimPrefix(pattern, "**/")
		}
		matches, err := path.Match(pattern, "go.mod")
		if err != nil {
			return false
		}
		if matches {
			included = !excluded
		}
	}
	return included
}

// findLastCheckoutStepIndex returns the zero-based index of the last
// actions/checkout step in customSteps and true, or (0, false) if none is found.
// It uses the same parsing strategy as findFirstCheckoutStepIndex.
func findLastCheckoutStepIndex(customSteps string) (int, bool) {
	if customSteps == "" {
		return 0, false
	}
	lastIndex := -1
	found := false

	// Try the wrapped form first: "steps:\n  - ...\n"
	var wrapper struct {
		Steps []map[string]any `yaml:"steps"`
	}
	if err := yaml.Unmarshal([]byte(customSteps), &wrapper); err == nil {
		if len(wrapper.Steps) > 0 {
			for i, step := range wrapper.Steps {
				uses, ok := step["uses"].(string)
				if ok && isCheckoutActionReference(uses) {
					lastIndex = i
					found = true
				}
			}
			return lastIndex, found
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
			lastIndex = i
			found = true
		}
	}
	return lastIndex, found
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
