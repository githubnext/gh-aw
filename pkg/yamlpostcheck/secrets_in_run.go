// This file implements the RGS-008 post-generation YAML fixer.
//
// # RGS-008 — Secrets Directly Interpolated in run: Blocks
//
// Severity: High
// Reference: https://github.com/Vigilant-LLC/runner-guard
//
// # Problem
//
// When a secret is interpolated directly into a run: block using
// ${{ secrets.NAME }} syntax, the secret value is embedded as plaintext
// in the shell script source before execution.  This exposes the secret to:
//   - Error messages and debug output
//   - Shell history and /proc filesystem
//   - Log masking bypass via side channels
//   - Amplified impact of any expression injection vulnerability
//
// The same risk applies to ${{ github.token }} and ${{ env.GITHUB_TOKEN }}
// when they appear inside run: blocks.
//
// # Fix
//
// SecretsInRunChecker moves these expressions from the run: script body into
// the step's env: map, where GitHub Actions masks the value before it reaches
// the runner shell.  References in the script are replaced with the
// corresponding environment variable using shell syntax ($VAR_NAME).
//
// Before (insecure):
//
//   - name: Call API
//     run: |
//     curl -H "Authorization: Bearer ${{ secrets.API_TOKEN }}" https://example.com
//
// After (secure):
//
//   - name: Call API
//     env:
//     API_TOKEN: ${{ secrets.API_TOKEN }}
//     run: |
//     curl -H "Authorization: Bearer $API_TOKEN" https://example.com
package yamlpostcheck

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var secretsInRunLog = logger.New("yamlpostcheck:secrets_in_run")

// secretsInRunPatterns lists the expression patterns that SecretsInRunChecker
// recognises and fixes.  Each entry pairs a compiled regex (which must contain
// exactly one named capture group "name" to extract the candidate env var name)
// with the fixed env var name to use when the regex contains no capture group.
type secretsInRunPattern struct {
	// re is the compiled regex.  When it contains a "name" capture group the
	// captured text is uppercased to derive the env var name.
	re *regexp.Regexp

	// fixedEnvVar is used when re contains no "name" capture group.
	fixedEnvVar string
}

var secretsExpressionPatterns = []secretsInRunPattern{
	{
		// ${{ secrets.NAME }} — any valid identifier after "secrets."
		re: regexp.MustCompile(`\$\{\{\s*secrets\.(?P<name>[A-Za-z_][A-Za-z0-9_]*)\s*\}\}`),
	},
	{
		// ${{ github.token }}
		re:          regexp.MustCompile(`\$\{\{\s*github\.token\s*\}\}`),
		fixedEnvVar: "GITHUB_TOKEN",
	},
	{
		// ${{ env.GITHUB_TOKEN }}
		re:          regexp.MustCompile(`\$\{\{\s*env\.GITHUB_TOKEN\s*\}\}`),
		fixedEnvVar: "GITHUB_TOKEN",
	},
}

// SecretsInRunChecker is a Checker that detects and fixes secret expressions
// (${{ secrets.* }}, ${{ github.token }}, ${{ env.GITHUB_TOKEN }}) interpolated
// directly inside run: blocks (RGS-008).
//
// For each affected step the checker:
//  1. Collects every distinct secret expression found in the run: content.
//  2. Derives a safe environment variable name for each expression.
//  3. Inserts (or extends) the step's env: map with the required mappings.
//  4. Replaces every occurrence of the expression in the run: content with
//     the corresponding shell variable reference ($VAR_NAME).
//
// The checker never overwrites an existing env: entry that already maps the
// same variable name to the same expression value.  If a name collision is
// detected with a different value, a unique suffixed name is chosen instead.
type SecretsInRunChecker struct {
	log *logger.Logger
}

// NewSecretsInRunChecker returns a new SecretsInRunChecker ready for use.
func NewSecretsInRunChecker() *SecretsInRunChecker {
	return &SecretsInRunChecker{log: secretsInRunLog}
}

// Name implements Checker.
func (c *SecretsInRunChecker) Name() string {
	return "rgs008-secrets-in-run"
}

// Check implements Checker.
//
// It walks the workflow tree looking for run: fields in all steps across all
// jobs.  Steps whose run: content contains any of the target secret expression
// patterns are mutated in place: the expressions are moved to the step's env:
// map and replaced with plain shell variable references.
func (c *SecretsInRunChecker) Check(tree map[string]any) (Result, error) {
	c.log.Print("Starting RGS-008 secrets-in-run check")

	var result Result

	jobs, ok := tree["jobs"]
	if !ok {
		c.log.Print("No jobs key found in workflow tree – nothing to check")
		return result, nil
	}

	jobsMap, ok := jobs.(map[string]any)
	if !ok {
		c.log.Printf("jobs key has unexpected type %T – skipping", jobs)
		return result, nil
	}

	c.log.Printf("Scanning %d job(s) for secrets in run: blocks", len(jobsMap))

	for jobName, jobVal := range jobsMap {
		job, ok := jobVal.(map[string]any)
		if !ok {
			continue
		}

		stepsVal, ok := job["steps"]
		if !ok {
			continue
		}

		steps, ok := stepsVal.([]any)
		if !ok {
			continue
		}

		c.log.Printf("Job %q: scanning %d step(s)", jobName, len(steps))

		for stepIdx, stepVal := range steps {
			step, ok := stepVal.(map[string]any)
			if !ok {
				continue
			}

			stepFixed, stepFixes := c.fixStep(jobName, stepIdx, step)
			if stepFixed {
				result.Changed = true
				result.Fixes = append(result.Fixes, stepFixes...)
				// Replace the original step map with the updated one.
				steps[stepIdx] = step
			}
		}
	}

	c.log.Printf("RGS-008 check complete: changed=%v total_fixes=%d", result.Changed, len(result.Fixes))
	return result, nil
}

// fixStep inspects a single step map and moves any secret expressions out of
// the run: field into the env: field.  It mutates step in place and returns
// whether any change was made along with human-readable fix descriptions.
func (c *SecretsInRunChecker) fixStep(jobName string, stepIdx int, step map[string]any) (changed bool, fixes []string) {
	runVal, ok := step["run"]
	if !ok {
		return false, nil
	}

	runScript, ok := runVal.(string)
	if !ok {
		return false, nil
	}

	// Quick pre-check: does the run: field contain any of the target patterns?
	if !anyPatternMatches(runScript) {
		return false, nil
	}

	stepLabel := stepLabelFor(step, jobName, stepIdx)
	c.log.Printf("Step %s: found secret expressions in run: block – applying fix", stepLabel)

	// Collect or create the env: map for this step.
	envMap := getOrCreateEnvMap(step)

	// Iterate over all patterns and process all occurrences in the run: script.
	updatedScript := runScript
	for _, pat := range secretsExpressionPatterns {
		matches := pat.re.FindAllString(updatedScript, -1)
		if len(matches) == 0 {
			continue
		}

		// Deduplicate matches: multiple occurrences of the same expression get
		// the same env var name and are all replaced in one pass.
		seen := make(map[string]string) // expression → env var name
		for _, expr := range matches {
			if _, already := seen[expr]; already {
				continue
			}

			envVarName := c.deriveEnvVarName(pat, expr, envMap)
			seen[expr] = envVarName

			// Add the mapping to the env: map (skip if already present with the
			// identical value — idempotent behaviour).
			existing, alreadySet := envMap[envVarName]
			if alreadySet {
				if existing == expr {
					c.log.Printf("  env var %s already maps to %q – reusing", envVarName, expr)
				} else {
					// Logged in deriveEnvVarName; the name has already been
					// disambiguated so this branch should not normally be reached.
					c.log.Printf("  env var %s collision resolved to %q", envVarName, expr)
					envMap[envVarName] = expr
				}
			} else {
				c.log.Printf("  adding env: %s = %s", envVarName, expr)
				envMap[envVarName] = expr
			}

			// Replace all occurrences of the expression in the script.
			shellRef := "$" + envVarName
			before := updatedScript
			updatedScript = strings.ReplaceAll(updatedScript, expr, shellRef)
			if updatedScript != before {
				fixMsg := fmt.Sprintf(
					"step %s: moved %s to env.%s and replaced with %s",
					stepLabel, expr, envVarName, shellRef,
				)
				fixes = append(fixes, fixMsg)
				changed = true
			}
		}
	}

	if changed {
		step["run"] = updatedScript
		step["env"] = envMap
		c.log.Printf("Step %s: applied %d fix(es)", stepLabel, len(fixes))
	}

	return changed, fixes
}

// deriveEnvVarNameMaxRetries is the maximum number of disambiguation attempts
// before deriveEnvVarName gives up and returns the last candidate as-is.
// In practice, env maps contain at most a handful of entries, so this limit
// will never be reached under normal operating conditions.
const deriveEnvVarNameMaxRetries = 100

// deriveEnvVarName returns the environment variable name to use for expr.
//
// Priority:
//  1. If pat contains a "name" capture group, uppercase the captured text.
//  2. Otherwise fall back to pat.fixedEnvVar.
//
// When the derived name collides with an existing entry in envMap that holds a
// different value, a disambiguating suffix (_1, _2, …) is appended until a
// free slot is found.  If no free slot is found within deriveEnvVarNameMaxRetries
// attempts, the last candidate is returned (guaranteed unique within that run).
func (c *SecretsInRunChecker) deriveEnvVarName(pat secretsInRunPattern, expr string, envMap map[string]any) string {
	base := pat.fixedEnvVar

	// Try to extract the name from a capture group.
	if idx := pat.re.SubexpIndex("name"); idx >= 0 {
		match := pat.re.FindStringSubmatch(expr)
		if match != nil && idx < len(match) {
			base = strings.ToUpper(match[idx])
		}
	}

	candidate := base

	// Resolve collisions: if envMap[candidate] already exists with a different
	// value, try candidate_1, candidate_2, … until we find a free slot or a
	// matching slot.  A safety limit prevents unbounded iteration in pathological
	// cases (e.g., an env map pre-populated with TOKEN, TOKEN_1, TOKEN_2, …).
	for i := 1; i <= deriveEnvVarNameMaxRetries; i++ {
		existing, exists := envMap[candidate]
		if !exists {
			// Free slot — use it.
			break
		}
		if existing == expr {
			// Already mapped to the same expression — reuse without changes.
			break
		}
		// Collision with a different value: try next candidate.
		c.log.Printf("  env var %s conflicts (existing=%v, new=%s) – trying %s_%d",
			candidate, existing, expr, base, i)
		candidate = fmt.Sprintf("%s_%d", base, i)
	}

	return candidate
}

// anyPatternMatches returns true if s contains at least one match for any of
// the secretsExpressionPatterns.  This is used as a fast pre-check to avoid
// the per-pattern scanning overhead on steps that contain no secret expressions.
func anyPatternMatches(s string) bool {
	// Quick string check first to avoid regex overhead on the vast majority of steps.
	if !strings.Contains(s, "${{") {
		return false
	}
	for _, pat := range secretsExpressionPatterns {
		if pat.re.MatchString(s) {
			return true
		}
	}
	return false
}

// getOrCreateEnvMap retrieves the existing env: map from step, or creates and
// inserts a new empty map when env: is absent.  The returned map is always
// non-nil and is the live value in step["env"].
func getOrCreateEnvMap(step map[string]any) map[string]any {
	if existing, ok := step["env"]; ok {
		if envMap, ok := existing.(map[string]any); ok {
			return envMap
		}
		// env: exists but has an unexpected type — replace it with a fresh map.
		// Use the package-level suite logger to avoid coupling to the checker logger.
		suiteLog.Printf("step has env: of unexpected type %T – replacing with empty map", step["env"])
	}
	envMap := make(map[string]any)
	step["env"] = envMap
	return envMap
}

// stepLabelFor returns a human-readable identifier for the step used in log
// messages and fix descriptions.  It prefers the step's "name" field when
// present; otherwise falls back to "job/<jobName>/step[<idx>]".
func stepLabelFor(step map[string]any, jobName string, stepIdx int) string {
	if name, ok := step["name"].(string); ok && name != "" {
		return fmt.Sprintf("%q (job %s, index %d)", name, jobName, stepIdx)
	}
	return fmt.Sprintf("job %s step[%d]", jobName, stepIdx)
}
