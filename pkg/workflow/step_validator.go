package workflow

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var stepValidatorLog = logger.New("workflow:step_validator")

// ghCommandPattern matches "gh " followed by a command or flag
// This pattern looks for "gh" as a word boundary followed by a space and something (command, flag, etc.)
var ghCommandPattern = regexp.MustCompile(`\bgh\s+`)

// ValidateStepsGHToken validates that steps using 'gh' CLI commands have GH_TOKEN set in env
// Returns an error if a step has a 'run' field containing 'gh' commands without GH_TOKEN in env
func ValidateStepsGHToken(steps []any) error {
	stepValidatorLog.Print("Validating steps for GH_TOKEN requirement")

	for i, step := range steps {
		stepMap, ok := step.(map[string]any)
		if !ok {
			continue
		}

		// Check if step has a 'run' field
		runField, hasRun := stepMap["run"]
		if !hasRun {
			continue
		}

		// Convert run field to string
		runStr, ok := runField.(string)
		if !ok {
			continue
		}

		// Check if the run command contains 'gh' CLI commands
		if !ghCommandPattern.MatchString(runStr) {
			continue
		}

		stepValidatorLog.Printf("Step %d contains 'gh' command: %q", i, runStr)

		// Check if the step has env field with GH_TOKEN
		envField, hasEnv := stepMap["env"]
		if !hasEnv {
			return fmt.Errorf("step %d uses 'gh' CLI commands but does not have GH_TOKEN set in env:\n  run: %s\n\nPlease add GH_TOKEN to the step's env field:\n  env:\n    GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}", i, truncateForError(runStr))
		}

		// Check if env is a map and contains GH_TOKEN
		envMap, ok := envField.(map[string]any)
		if !ok {
			return fmt.Errorf("step %d uses 'gh' CLI commands but has invalid env field (expected map):\n  run: %s\n\nPlease add GH_TOKEN to the step's env field:\n  env:\n    GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}", i, truncateForError(runStr))
		}

		if _, hasGHToken := envMap["GH_TOKEN"]; !hasGHToken {
			return fmt.Errorf("step %d uses 'gh' CLI commands but does not have GH_TOKEN set in env:\n  run: %s\n\nPlease add GH_TOKEN to the step's env field:\n  env:\n    GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}", i, truncateForError(runStr))
		}

		stepValidatorLog.Printf("Step %d has GH_TOKEN properly set", i)
	}

	stepValidatorLog.Print("All steps with 'gh' commands have GH_TOKEN set")
	return nil
}

// truncateForError truncates a string for use in error messages
func truncateForError(s string) string {
	const maxLen = 100
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
