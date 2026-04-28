// Package yamlpostcheck provides a post-generation YAML validation and fixing
// engine for compiled GitHub Actions workflows.
//
// After a workflow is compiled to YAML, this package runs a built-in suite of
// checkers that detect and automatically fix common unsafe patterns in the
// generated YAML tree.  Each checker receives a mutable map[string]any tree
// (as produced by yaml.Unmarshal), optionally transforms it in place, and
// returns a Result that describes what was found and changed.
//
// # Design
//
// Checkers are registered with a Suite and executed in registration order.
// The suite aggregates results across all checkers and re-serialises the tree
// when any checker reports a mutation.
//
// # Built-in Checkers
//
//   - SecretsInRunChecker – RGS-008: moves ${{ secrets.* }}, ${{ github.token }},
//     and ${{ env.GITHUB_TOKEN }} expressions out of run: blocks and into the
//     step's env: map, preventing plaintext secret exposure in shell scripts.
//
// # Usage
//
//	suite := yamlpostcheck.New()
//	changed, fixes, warnings, err := suite.Run(tree)
//	if err != nil {
//	    // handle hard error
//	}
//	if changed {
//	    // re-serialise tree to YAML
//	}
package yamlpostcheck

// Checker is implemented by every post-generation YAML check.
//
// Each checker receives the parsed workflow tree (a map[string]any produced by
// yaml.Unmarshal) and may mutate it in place to fix detected issues.
// The tree is guaranteed to be non-nil when Check is called.
//
// Returning a non-nil error causes the Suite to surface the error to the
// caller without executing any remaining checkers.
type Checker interface {
	// Name returns a short, stable identifier for this checker.
	// It is used in log messages and fix descriptions.
	// Example: "rgs008-secrets-in-run"
	Name() string

	// Check analyses the YAML tree and optionally mutates it to fix issues.
	// It returns a Result summarising what was found and/or changed.
	Check(tree map[string]any) (Result, error)
}

// Result summarises the outcome of a single Checker.Check call.
type Result struct {
	// Changed is true when the checker mutated the YAML tree.
	Changed bool

	// Fixes is a human-readable list of changes applied to the tree.
	// Populated only when Changed is true.
	Fixes []string

	// Warnings is a human-readable list of non-fatal observations about the
	// workflow that the caller may surface to the user.
	Warnings []string
}
