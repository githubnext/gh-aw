package yamlpostcheck

import (
	"fmt"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/goccy/go-yaml"
)

var suiteLog = logger.New("yamlpostcheck:suite")

// Suite runs a set of Checkers against a parsed YAML workflow tree.
//
// Checkers are executed in registration order.  Each checker may mutate the
// tree; subsequent checkers always see the latest (possibly mutated) state.
// The suite aggregates all fixes, warnings, and the first hard error.
type Suite struct {
	checkers []Checker
}

// New creates a Suite pre-loaded with the default set of built-in checkers.
//
// Built-in checkers are always enabled; they cannot be disabled by callers.
func New() *Suite {
	s := &Suite{}
	// Register built-in checkers in execution order.
	// Add new checkers here as they are introduced.
	s.Register(NewSecretsInRunChecker())
	return s
}

// Register appends a Checker to the suite.
//
// Checkers are executed in the order they are registered.  Register panics if
// checker is nil.
func (s *Suite) Register(c Checker) {
	if c == nil {
		panic("yamlpostcheck: cannot register nil Checker")
	}
	s.checkers = append(s.checkers, c)
	suiteLog.Printf("Registered checker: %s", c.Name())
}

// Run executes all registered checkers against tree in registration order.
//
// tree must not be nil.  Each checker may mutate tree in place; subsequent
// checkers see the updated tree.
//
// Run returns:
//   - changed: true if any checker mutated the tree.
//   - fixes:   human-readable descriptions of all mutations applied.
//   - warnings: human-readable non-fatal observations from all checkers.
//   - err:     the first hard error returned by any checker; remaining
//     checkers are not executed when an error occurs.
func (s *Suite) Run(tree map[string]any) (changed bool, fixes []string, warnings []string, err error) {
	if tree == nil {
		suiteLog.Print("Run called with nil tree – skipping all checkers")
		return false, nil, nil, nil
	}

	suiteLog.Printf("Running %d checker(s) against workflow tree", len(s.checkers))

	for _, c := range s.checkers {
		suiteLog.Printf("Running checker: %s", c.Name())

		result, checkerErr := c.Check(tree)
		if checkerErr != nil {
			suiteLog.Printf("Checker %s returned error: %v", c.Name(), checkerErr)
			return changed, fixes, warnings, fmt.Errorf("checker %s: %w", c.Name(), checkerErr)
		}

		if result.Changed {
			changed = true
			suiteLog.Printf("Checker %s applied %d fix(es)", c.Name(), len(result.Fixes))
		} else {
			suiteLog.Printf("Checker %s: no changes", c.Name())
		}

		fixes = append(fixes, result.Fixes...)
		warnings = append(warnings, result.Warnings...)

		for _, fix := range result.Fixes {
			suiteLog.Printf("  fix: %s", fix)
		}
		for _, w := range result.Warnings {
			suiteLog.Printf("  warning: %s", w)
		}
	}

	suiteLog.Printf("Suite complete: changed=%v fixes=%d warnings=%d", changed, len(fixes), len(warnings))
	return changed, fixes, warnings, nil
}

// RunOnYAML is a convenience method that parses yamlContent, runs all
// registered checkers on the parsed tree, and — when any checker mutated the
// tree — re-serialises the body to produce a fixed YAML string.
//
// The leading comment block (gh-aw-metadata, gh-aw-manifest, logo, etc.) is
// always preserved verbatim; only the YAML body is parsed and potentially
// re-marshalled.
//
// RunOnYAML returns:
//   - result:   the (possibly fixed) YAML string.
//   - changed:  true when at least one checker mutated the tree.
//   - fixes:    human-readable descriptions of all mutations applied.
//   - warnings: human-readable non-fatal observations from all checkers.
//   - err:      the first hard error from any checker, or a parse/marshal error.
func (s *Suite) RunOnYAML(yamlContent string) (result string, changed bool, fixes []string, warnings []string, err error) {
	if yamlContent == "" {
		return "", false, nil, nil, nil
	}

	// Preserve the leading comment block (header) verbatim.
	header, body := SplitYAMLHeader(yamlContent)
	suiteLog.Printf("YAML split: header=%d bytes body=%d bytes", len(header), len(body))

	if body == "" {
		// Nothing to parse / fix — just a comment block.
		suiteLog.Print("YAML body is empty – skipping post-generation checks")
		return yamlContent, false, nil, nil, nil
	}

	// Parse the body into a mutable tree.
	var tree map[string]any
	if parseErr := yaml.Unmarshal([]byte(body), &tree); parseErr != nil {
		// Non-fatal: skip the suite rather than blocking compilation, but surface
		// the skip as a warning so callers are aware the check was not applied.
		skipMsg := fmt.Sprintf("post-generation checks skipped: failed to parse YAML body (%v)", parseErr)
		suiteLog.Printf("post-generation checks skipped: failed to parse YAML body: %v", parseErr)
		return yamlContent, false, nil, []string{skipMsg}, nil
	}
	if tree == nil {
		suiteLog.Print("Parsed YAML body is nil – skipping post-generation checks")
		return yamlContent, false, nil, nil, nil
	}

	// Run all checkers.
	changed, fixes, warnings, err = s.Run(tree)
	if err != nil {
		return yamlContent, changed, fixes, warnings, fmt.Errorf("post-generation YAML check: %w", err)
	}

	if !changed {
		suiteLog.Print("No changes from post-generation checks – returning original YAML")
		return yamlContent, false, fixes, warnings, nil
	}

	// At least one checker mutated the tree — re-serialise the body.
	suiteLog.Print("Re-serialising YAML body after post-generation fixes")
	marshalledBody, marshalErr := yaml.Marshal(tree)
	if marshalErr != nil {
		// Re-serialisation failed: surface as a warning and return the original
		// content.  changed=true here would be misleading since the on-disk YAML
		// was not updated, so we return false but include a warning explaining
		// that the in-memory tree was mutated but could not be serialised.
		marshalWarning := fmt.Sprintf("post-generation fixes applied in memory but could not be serialised (%v); original YAML unchanged", marshalErr)
		suiteLog.Printf("post-generation fixes applied in memory but could not be serialised: %v; original YAML unchanged", marshalErr)
		warnings = append(warnings, marshalWarning)
		return yamlContent, false, fixes, warnings, nil
	}

	result = JoinYAMLHeaderBody(header, string(marshalledBody))
	suiteLog.Printf("Post-generation fix applied: %d bytes → %d bytes", len(yamlContent), len(result))
	return result, true, fixes, warnings, nil
}
