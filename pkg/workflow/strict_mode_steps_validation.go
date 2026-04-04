// This file contains strict mode validation for secrets in custom steps.
//
// It validates that secrets expressions are not used in custom steps (steps and
// post-steps injected in the agent job). In strict mode this is an error; in
// non-strict mode a warning is emitted instead.
//
// The goal is to minimise the number of secrets present in the agent job: the
// only secrets that should appear there are those required to configure the
// agentic engine itself.

package workflow

import (
	"fmt"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/sliceutil"
)

// validateStepsSecrets checks both the "steps" and "post-steps" frontmatter sections
// for secrets expressions (e.g. ${{ secrets.MY_SECRET }}).
//
// In strict mode the presence of any such expression is treated as an error.
// In non-strict mode a warning is emitted instead.
func (c *Compiler) validateStepsSecrets(frontmatter map[string]any) error {
	for _, sectionName := range []string{"steps", "post-steps"} {
		if err := c.validateStepsSectionSecrets(frontmatter, sectionName); err != nil {
			return err
		}
	}
	return nil
}

// validateStepsSectionSecrets inspects a single steps section (named by sectionName)
// inside frontmatter for any secrets.* expressions.
func (c *Compiler) validateStepsSectionSecrets(frontmatter map[string]any, sectionName string) error {
	rawValue, exists := frontmatter[sectionName]
	if !exists {
		strictModeValidationLog.Printf("No %s section found, skipping secrets validation", sectionName)
		return nil
	}

	steps, ok := rawValue.([]any)
	if !ok {
		strictModeValidationLog.Printf("%s section is not a list, skipping secrets validation", sectionName)
		return nil
	}

	var secretRefs []string
	for _, step := range steps {
		refs := extractSecretsFromStepValue(step)
		secretRefs = append(secretRefs, refs...)
	}

	// Filter out the built-in GITHUB_TOKEN: it is already present in every runner
	// environment and is not a user-defined secret that could be accidentally leaked.
	secretRefs = filterBuiltinTokens(secretRefs)

	if len(secretRefs) == 0 {
		strictModeValidationLog.Printf("No secrets found in %s section", sectionName)
		return nil
	}

	strictModeValidationLog.Printf("Found %d secret expression(s) in %s section: %v", len(secretRefs), sectionName, secretRefs)

	// Deduplicate for cleaner messages.
	secretRefs = sliceutil.Deduplicate(secretRefs)

	if c.strictMode {
		return fmt.Errorf(
			"strict mode: secrets expressions detected in '%s' section may be leaked to the agent job. Found: %s. "+
				"Operations requiring secrets must be moved to a separate job outside the agent job",
			sectionName, strings.Join(secretRefs, ", "),
		)
	}

	// Non-strict mode: emit a warning.
	warningMsg := fmt.Sprintf(
		"Warning: secrets expressions detected in '%s' section may be leaked to the agent job. Found: %s. "+
			"Consider moving operations requiring secrets to a separate job outside the agent job.",
		sectionName, strings.Join(secretRefs, ", "),
	)
	fmt.Fprintln(os.Stderr, console.FormatWarningMessage(warningMsg))
	c.IncrementWarningCount()

	return nil
}

// extractSecretsFromStepValue recursively walks a step value (which may be a map,
// slice, or primitive) and returns all secrets.* expressions found in string values.
func extractSecretsFromStepValue(value any) []string {
	var refs []string
	switch v := value.(type) {
	case string:
		for _, expr := range ExtractSecretsFromValue(v) {
			refs = append(refs, expr)
		}
	case map[string]any:
		for _, fieldValue := range v {
			refs = append(refs, extractSecretsFromStepValue(fieldValue)...)
		}
	case []any:
		for _, item := range v {
			refs = append(refs, extractSecretsFromStepValue(item)...)
		}
	}
	return refs
}

// filterBuiltinTokens removes secret expressions that reference GitHub's built-in
// GITHUB_TOKEN from the list. GITHUB_TOKEN is automatically provided by the runner
// environment and is not a user-defined secret; it therefore does not represent an
// accidental leak into the agent job.
func filterBuiltinTokens(refs []string) []string {
	out := refs[:0:0]
	for _, ref := range refs {
		if !strings.Contains(ref, "secrets.GITHUB_TOKEN") {
			out = append(out, ref)
		}
	}
	return out
}
