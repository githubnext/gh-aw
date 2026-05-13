// This file provides validation for pull_request_target trigger security.
//
// # pull_request_target Trigger Validation
//
// The pull_request_target trigger runs workflows in the context of the base
// (target) branch with full write permissions and access to repository secrets.
// Unlike pull_request, it can access secrets from fork PRs, making it extremely
// dangerous when combined with a checkout of PR code.
//
// # Validation Rules
//
//  1. In strict mode: always emit a warning that pull_request_target is a very
//     dangerous trigger, even when checkout: false is set, because the workflow
//     still runs with full write permissions and secret access.
//     Workflows can opt out by setting strict: false in frontmatter.
//
//  2. When checkout is NOT explicitly disabled (checkout: false not set):
//     - In strict mode: return a hard error (extremely insecure).
//     - In non-strict mode: emit a warning.
//
// # References
//
// See: https://securitylab.github.com/resources/github-actions-preventing-pwn-requests/
//
// # When to Add Validation Here
//
// Add validation to this file when:
//   - It validates pull_request_target-specific security requirements.
//   - It enforces checkout restrictions for this trigger type.
//
// For general validation, see validation.go.
// For detailed documentation, see scratchpad/validation-architecture.md

package workflow

import (
	"fmt"
	"os"
	"strings"

	"github.com/goccy/go-yaml"
)

var pullRequestTargetLog = newValidationLogger("pull_request_target")

// validatePullRequestTargetTrigger validates security requirements for pull_request_target triggers.
//
// The pull_request_target trigger runs with full write permissions and repository secret access
// on the base branch. When checkout is not explicitly disabled (checkout: false), the workflow
// may execute untrusted PR code with elevated privileges — a critical security vulnerability
// commonly known as a "pwn request" attack.
//
// In strict mode, a warning is always emitted that pull_request_target is inherently dangerous
// even with checkout disabled, since the workflow still runs with elevated permissions.
// This strict behavior is disabled when the workflow frontmatter explicitly sets strict: false.
func (c *Compiler) validatePullRequestTargetTrigger(workflowData *WorkflowData, markdownPath string) error {
	// Fast path: skip expensive YAML parsing when the On field cannot possibly contain
	// a pull_request_target trigger. This avoids yaml.Unmarshal on every
	// validateWorkflowData call for the common case of non-pull_request_target workflows.
	// The YAML parsing below is the authoritative check — the fast path only provides
	// early exit when the literal string is absent. If the string appears as part of a
	// longer YAML key (e.g. pull_request_target_staging), the YAML parse will correctly
	// find no "pull_request_target" key and return nil, so there are no false positives.
	if !strings.Contains(workflowData.On, "pull_request_target") {
		return nil
	}

	pullRequestTargetLog.Print("Validating pull_request_target trigger security")

	// Parse the On field as YAML to confirm pull_request_target is actually a trigger key.
	var parsedData map[string]any
	if err := yaml.Unmarshal([]byte(workflowData.On), &parsedData); err != nil {
		pullRequestTargetLog.Printf("Could not parse On field as YAML: %v", err)
		return nil
	}

	onData, hasOn := parsedData["on"]
	if !hasOn {
		return nil
	}

	onMap, isMap := onData.(map[string]any)
	if !isMap {
		return nil
	}

	_, hasPRT := onMap["pull_request_target"]
	if !hasPRT {
		return nil
	}

	effectiveStrictMode := c.strictMode
	if workflowData.RawFrontmatter != nil {
		if strictBool, ok := workflowData.RawFrontmatter["strict"].(bool); ok && !strictBool {
			pullRequestTargetLog.Print("Frontmatter strict: false detected, disabling strict mode error for pull_request_target validation")
			effectiveStrictMode = false
		}
	}

	// In strict mode, always emit a warning that pull_request_target is a very dangerous trigger,
	// regardless of whether checkout is disabled. The workflow still runs with full write
	// permissions and has access to all repository secrets.
	if effectiveStrictMode {
		pullRequestTargetLog.Print("Emitting strict mode warning: pull_request_target is a very dangerous trigger")
		warningMsg := "pull_request_target is a very dangerous trigger.\n" +
			"This event runs with full write permissions and access to all repository secrets.\n" +
			"Unlike pull_request, it runs in the context of the target (base) branch, giving\n" +
			"the workflow elevated access even for PRs from untrusted fork contributors.\n" +
			"Even with checkout: false, consider whether pull_request_target is truly necessary.\n" +
			"If you only need to react to PR events without write access, use pull_request instead.\n" +
			"See: https://securitylab.github.com/resources/github-actions-preventing-pwn-requests/"
		fmt.Fprintln(os.Stderr, formatCompilerMessage(markdownPath, "warning", warningMsg))
		c.IncrementWarningCount()
	}

	// If checkout is disabled, the workflow will not execute PR code — no further action needed.
	if workflowData.CheckoutDisabled {
		pullRequestTargetLog.Print("checkout: false is set, skipping insecure-checkout error")
		return nil
	}

	// Checkout is not disabled — the workflow may execute untrusted PR code with elevated privileges.
	pullRequestTargetLog.Print("checkout is NOT disabled, emitting pull_request_target insecure-checkout diagnostic")

	message := "pull_request_target trigger with checkout enabled is extremely insecure.\n\n" +
		"This event runs with full write permissions and access to repository secrets,\n" +
		"but the workflow will check out code from a potentially untrusted PR contributor.\n" +
		"This is a well-known attack vector: a fork PR can inject malicious code that\n" +
		"executes with access to your repository's secrets (\"pwn request\" attack).\n\n" +
		"Suggested fix: Add 'checkout: false' to your workflow frontmatter to prevent\n" +
		"checking out untrusted PR code:\n" +
		"checkout: false\n\n" +
		"See: https://securitylab.github.com/resources/github-actions-preventing-pwn-requests/"

	if effectiveStrictMode {
		return formatCompilerError(markdownPath, "error", message, nil)
	}

	// Non-strict mode: emit a warning so existing workflows continue to compile.
	fmt.Fprintln(os.Stderr, formatCompilerMessage(markdownPath, "warning", message))
	c.IncrementWarningCount()

	return nil
}
