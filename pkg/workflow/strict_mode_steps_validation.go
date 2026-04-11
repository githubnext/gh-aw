// This file contains strict mode validation for secrets in custom steps.
//
// It validates that secrets expressions are not used in custom steps (steps and
// post-steps injected in the agent job). In strict mode, secrets in step-level
// env: bindings are allowed (controlled binding, masked by GitHub Actions),
// while secrets in other fields (run, with, etc.) are treated as errors.
// In non-strict mode a warning is emitted instead.
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

// validateStepsSecrets checks the "pre-steps", "steps", and "post-steps" frontmatter sections
// for secrets expressions (e.g. ${{ secrets.MY_SECRET }}).
//
// In strict mode, secrets in step-level env: bindings are allowed (controlled,
// masked binding), while secrets in other fields (run, with, etc.) are errors.
// In non-strict mode a warning is emitted for all secrets.
func (c *Compiler) validateStepsSecrets(frontmatter map[string]any) error {
	for _, sectionName := range []string{"pre-steps", "steps", "post-steps"} {
		if err := c.validateStepsSectionSecrets(frontmatter, sectionName); err != nil {
			return err
		}
	}
	return nil
}

// validateStepsSectionSecrets inspects a single steps section (named by sectionName)
// inside frontmatter for any secrets.* expressions.
//
// In strict mode, secrets in step-level env: bindings are allowed because they are
// controlled bindings that are automatically masked by GitHub Actions. Secrets in
// other step fields (run, with, etc.) are still treated as errors.
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

	// Separate secrets found in step-level env: bindings (safe, controlled)
	// from secrets found in other fields (unsafe, potential leak).
	var unsafeSecretRefs []string
	var envSecretRefs []string
	for _, step := range steps {
		unsafe, envOnly := classifyStepSecrets(step)
		unsafeSecretRefs = append(unsafeSecretRefs, unsafe...)
		envSecretRefs = append(envSecretRefs, envOnly...)
	}

	// Filter out the built-in GITHUB_TOKEN: it is already present in every runner
	// environment and is not a user-defined secret that could be accidentally leaked.
	unsafeSecretRefs = filterBuiltinTokens(unsafeSecretRefs)
	envSecretRefs = filterBuiltinTokens(envSecretRefs)

	allSecretRefs := append(unsafeSecretRefs, envSecretRefs...)

	if len(allSecretRefs) == 0 {
		strictModeValidationLog.Printf("No secrets found in %s section", sectionName)
		return nil
	}

	strictModeValidationLog.Printf("Found %d secret expression(s) in %s section: %d unsafe, %d in env bindings",
		len(allSecretRefs), sectionName, len(unsafeSecretRefs), len(envSecretRefs))

	if c.strictMode {
		// In strict mode, secrets in step-level env: bindings are allowed
		// (controlled binding, masked by GitHub Actions). Only block secrets
		// found in other fields (run, with, etc.).
		if len(unsafeSecretRefs) == 0 {
			strictModeValidationLog.Printf("All secrets in %s section are in env bindings (allowed in strict mode)", sectionName)
			return nil
		}

		unsafeSecretRefs = sliceutil.Deduplicate(unsafeSecretRefs)
		return fmt.Errorf(
			"strict mode: secrets expressions detected in '%s' section may be leaked to the agent job. Found: %s. "+
				"Operations requiring secrets must be moved to a separate job outside the agent job, "+
				"or use step-level env: bindings instead",
			sectionName, strings.Join(unsafeSecretRefs, ", "),
		)
	}

	// Non-strict mode: emit a warning for all secrets.
	allSecretRefs = sliceutil.Deduplicate(allSecretRefs)
	warningMsg := fmt.Sprintf(
		"Warning: secrets expressions detected in '%s' section may be leaked to the agent job. Found: %s. "+
			"Consider moving operations requiring secrets to a separate job outside the agent job.",
		sectionName, strings.Join(allSecretRefs, ", "),
	)
	fmt.Fprintln(os.Stderr, console.FormatWarningMessage(warningMsg))
	c.IncrementWarningCount()

	return nil
}

// classifyStepSecrets separates secrets found in a step into two categories:
// - unsafeRefs: secrets found in fields other than "env" (e.g. run, with)
// - envRefs: secrets found in step-level env: bindings (controlled, masked)
//
// Only secrets in well-formed env: mappings (map[string]any) are classified as
// envRefs. Malformed env values (string, slice, etc.) are treated as unsafe to
// prevent strict-mode bypass via invalid YAML like `env: "${{ secrets.TOKEN }}"`.
func classifyStepSecrets(step any) (unsafeRefs, envRefs []string) {
	stepMap, ok := step.(map[string]any)
	if !ok {
		// Non-map steps: all secrets are considered unsafe.
		return extractSecretsFromStepValue(step), nil
	}
	for key, val := range stepMap {
		refs := extractSecretsFromStepValue(val)
		if key == "env" {
			if _, isMap := val.(map[string]any); isMap {
				envRefs = append(envRefs, refs...)
			} else {
				// Malformed env (string, slice, etc.): treat as unsafe.
				unsafeRefs = append(unsafeRefs, refs...)
			}
		} else {
			unsafeRefs = append(unsafeRefs, refs...)
		}
	}
	return unsafeRefs, envRefs
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
