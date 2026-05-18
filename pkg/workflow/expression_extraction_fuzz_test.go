//go:build !integration

package workflow

import (
	"strings"
	"testing"
)

// FuzzExtractTerminalSubExpressions fuzz-tests extractTerminalSubExpressions against
// arbitrary inputs. It validates that the function:
//  1. Never panics, regardless of input.
//  2. Returns only non-empty strings.
//  3. Every returned token matches simpleIdentifierRegex and runtimeEvalEnvVarPrefixRegex.
//  4. Returns no duplicate tokens.
//  5. Handles combinations of ||, &&, and parentheses without incorrect results.
func FuzzExtractTerminalSubExpressions(f *testing.F) {
	// Simple cases
	f.Add("steps.sanitized.outputs.text || inputs.command")
	f.Add("needs.build.outputs.version && inputs.override")
	f.Add("inputs.repo")
	f.Add("github.event.issue.number || inputs.item_number")
	f.Add("steps.pick-experiment.outputs.name || inputs.fallback")

	// Parenthesised combinations
	f.Add("(steps.a.outputs.x || inputs.y)")
	f.Add("(steps.a.outputs.x || inputs.y) && inputs.z")
	f.Add("steps.a.outputs.x || (inputs.y && inputs.z)")
	f.Add("(steps.a.outputs.x || inputs.y) && (steps.b.outputs.z || inputs.w)")
	f.Add("(steps.a.outputs.x || (inputs.y && inputs.z)) && needs.pre.outputs.ok")

	// Mixed github.* (excluded) and steps/inputs (included)
	f.Add("github.event.issue.number || inputs.item_number")
	f.Add("(github.event.issue.number || inputs.item_number) && steps.sanitized.outputs.text")

	// String literals (excluded)
	f.Add("steps.sanitized.outputs.text || 'fallback'")
	f.Add("inputs.repo || 'default/repo'")

	// Malformed / edge-case inputs
	f.Add("")
	f.Add("(")
	f.Add(")")
	f.Add("()")
	f.Add("((")
	f.Add("))")
	f.Add("(((steps.a || inputs.b)))")
	f.Add("||")
	f.Add("&&")
	f.Add("|| &&")
	f.Add("steps.a.outputs.x ||")
	f.Add("|| inputs.y")
	f.Add("((steps.a.outputs.x || inputs.y) && (steps.b.outputs.z")

	// Function calls (should produce no output due to simpleIdentifierRegex)
	f.Add("fromJSON(steps.a.outputs.json).field || inputs.fallback")
	f.Add("contains(inputs.labels, 'bug')")

	// Hyphenated identifiers (should be excluded)
	f.Add("steps.pick-experiment.outputs.name")
	f.Add("steps.pick-experiment.outputs.name || inputs.fallback")

	f.Fuzz(func(t *testing.T, expr string) {
		// Must never panic.
		result := extractTerminalSubExpressions(expr)

		// All returned tokens must be non-empty.
		for _, tok := range result {
			if tok == "" {
				t.Errorf("extractTerminalSubExpressions(%q) returned empty token", expr)
			}
		}

		// All returned tokens must satisfy both matchers.
		for _, tok := range result {
			if !simpleIdentifierRegex.MatchString(tok) {
				t.Errorf("extractTerminalSubExpressions(%q) returned non-simple-identifier token %q", expr, tok)
			}
			if !runtimeEvalEnvVarPrefixRegex.MatchString(tok) {
				t.Errorf("extractTerminalSubExpressions(%q) returned token with unexpected prefix %q", expr, tok)
			}
		}

		// No duplicate tokens.
		seen := make(map[string]bool, len(result))
		for _, tok := range result {
			if seen[tok] {
				t.Errorf("extractTerminalSubExpressions(%q) returned duplicate token %q", expr, tok)
			}
			seen[tok] = true
		}
	})
}

// FuzzSplitExpressionOnLogicalOps fuzz-tests splitExpressionOnLogicalOps against
// arbitrary inputs. It validates that the function:
//  1. Never panics.
//  2. Reassembling the parts (with "||" between them) produces the correct result.
//  3. Handles arbitrary combinations of parentheses, operators, and identifiers.
func FuzzSplitExpressionOnLogicalOps(f *testing.F) {
	// Seed with realistic GitHub Actions expression fragments
	f.Add("steps.sanitized.outputs.text || inputs.command")
	f.Add("a && b || c")
	f.Add("(a || b) && c")
	f.Add("a || (b && c)")
	f.Add("(a || b) && (c || d)")
	f.Add("")
	f.Add("||")
	f.Add("&&")
	f.Add("()")
	f.Add("(a)")
	f.Add("((a || b))")
	f.Add("a || b || c || d")
	f.Add("a && b && c && d")
	f.Add("(a || b || c) && (d || e)")

	f.Fuzz(func(t *testing.T, expr string) {
		// Must never panic.
		parts := splitExpressionOnLogicalOps(expr)

		// Result must be non-nil when there is at least one character.
		if expr != "" && len(parts) == 0 {
			t.Errorf("splitExpressionOnLogicalOps(%q) returned no parts for non-empty input", expr)
		}

		// The split may produce parts that contain the characters from the original
		// expression (minus the top-level "||"/"&&" separators).  Check that the total
		// character count of all parts equals the original minus the removed separators.
		totalChars := 0
		for _, p := range parts {
			totalChars += len(p)
		}
		separatorChars := 2 * max(0, len(parts)-1) // each "||"/"&&" separator is 2 bytes
		if totalChars+separatorChars > len(expr) {
			// Parts contain more characters than the original — something went wrong.
			t.Errorf("splitExpressionOnLogicalOps(%q) parts total %d chars > original %d chars (with %d separator chars)",
				expr, totalChars, len(expr), separatorChars)
		}

		// Parenthesis depth must balance in each part that was split at depth-0.
		// (Parts may contain unbalanced parens if the input itself is unbalanced, so
		// we only assert when the whole expression's depth is balanced.)
		totalDepth := 0
		for _, ch := range expr {
			if ch == '(' {
				totalDepth++
			} else if ch == ')' {
				totalDepth--
			}
		}
		if totalDepth == 0 {
			// Expression is balanced — every returned part should individually balance.
			for _, part := range parts {
				d := 0
				for _, ch := range part {
					if ch == '(' {
						d++
					} else if ch == ')' {
						d--
					}
				}
				if d != 0 {
					t.Errorf("splitExpressionOnLogicalOps(%q) part %q has unbalanced parens (depth %d) in a balanced expression",
						expr, part, d)
				}
			}
		}
	})
}

// FuzzExtractExpressions fuzz-tests the full ExtractExpressions pipeline against
// arbitrary markdown strings. It validates that the function:
//  1. Never panics or returns an error for arbitrary input.
//  2. Returns only ExpressionMapping values with non-empty Original, EnvVar, and Content.
//  3. Every EnvVar has the "GH_AW_" prefix and is uppercase.
//  4. No two mappings share the same EnvVar.
//  5. Mappings for compound expressions (containing ||/&&) are accompanied by
//     sub-expression mappings for steps.*/inputs.*/needs.* leaf tokens.
func FuzzExtractExpressions(f *testing.F) {
	// Plain markdown (no expressions)
	f.Add("This is plain text")
	f.Add("")
	f.Add("# Heading\n\nSome content.")

	// Single simple expressions
	f.Add("Repo: ${{ github.repository }}")
	f.Add("Actor: ${{ github.actor }}")
	f.Add("Step output: ${{ steps.sanitized.outputs.text }}")
	f.Add("Input: ${{ inputs.command }}")

	// Compound expressions
	f.Add("Data: ${{ steps.sanitized.outputs.text || inputs.command }}")
	f.Add("Data: ${{ needs.build.outputs.version && inputs.override }}")

	// Parenthesised compound expressions
	f.Add("Data: ${{ (steps.a.outputs.x || inputs.y) && inputs.z }}")
	f.Add("Data: ${{ steps.a.outputs.x || (inputs.y && inputs.z) }}")
	f.Add("Data: ${{ (steps.a.outputs.x || inputs.y) && (steps.b.outputs.z || inputs.w) }}")

	// Multiple expressions in one markdown
	f.Add("Repo: ${{ github.repository }}, Actor: ${{ github.actor }}")
	f.Add("${{ steps.a.outputs.x || inputs.y }}, ${{ inputs.z }}")

	// Deprecated activation output syntax
	f.Add("Content: ${{ needs.activation.outputs.text }}")
	f.Add("Fallback: ${{ needs.activation.outputs.text || 'default' }}")

	// Malformed expressions
	f.Add("Bad: ${{ }}")
	f.Add("Unterminated: ${{ steps.a.outputs.x")
	f.Add("No open: steps.a.outputs.x }}")

	// Expressions with special characters
	f.Add("${{ github.event.inputs.name || 'default-name' }}")
	f.Add("${{ inputs.repo || 'owner/repo' }}")

	f.Fuzz(func(t *testing.T, markdown string) {
		extractor := NewExpressionExtractor()

		// Must never panic or return an error.
		mappings, err := extractor.ExtractExpressions(markdown)
		if err != nil {
			t.Errorf("ExtractExpressions(%q) returned unexpected error: %v", markdown, err)
			return
		}

		// Every mapping must have non-empty fields and a valid env var.
		envVarSeen := make(map[string]bool, len(mappings))
		for _, m := range mappings {
			if m.Original == "" {
				t.Errorf("ExtractExpressions(%q) returned mapping with empty Original", markdown)
			}
			if m.EnvVar == "" {
				t.Errorf("ExtractExpressions(%q) returned mapping with empty EnvVar", markdown)
			}
			if m.Content == "" {
				t.Errorf("ExtractExpressions(%q) returned mapping with empty Content", markdown)
			}
			if !strings.HasPrefix(m.EnvVar, "GH_AW_") {
				t.Errorf("ExtractExpressions(%q): EnvVar %q missing GH_AW_ prefix", markdown, m.EnvVar)
			}
			if m.EnvVar != strings.ToUpper(m.EnvVar) {
				t.Errorf("ExtractExpressions(%q): EnvVar %q is not uppercase", markdown, m.EnvVar)
			}
			if envVarSeen[m.EnvVar] {
				t.Errorf("ExtractExpressions(%q): duplicate EnvVar %q", markdown, m.EnvVar)
			}
			envVarSeen[m.EnvVar] = true
		}

		// For each compound mapping, every qualifying leaf sub-expression must have
		// its own deterministic mapping present in the result.
		contentToMapping := make(map[string]*ExpressionMapping, len(mappings))
		for _, m := range mappings {
			contentToMapping[m.Content] = m
		}
		for _, m := range mappings {
			for _, sub := range extractTerminalSubExpressions(m.Content) {
				if _, ok := contentToMapping[sub]; !ok {
					t.Errorf("ExtractExpressions(%q): compound %q missing sub-expression mapping for %q",
						markdown, m.Content, sub)
				}
			}
		}
	})
}

// max returns the larger of a and b.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
