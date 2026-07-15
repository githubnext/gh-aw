package workflow

// effectiveStrictMode computes the effective strict mode for a workflow.
// Priority: CLI flag (c.strictMode) > frontmatter strict field > default (true).
// This should be used when emitting metadata/env vars to correctly reflect the
// workflow's strictness as inferred from the source (frontmatter).
func (c *Compiler) effectiveStrictMode(frontmatter map[string]any) bool {
	if c.strictMode {
		// CLI flag takes precedence
		return true
	}
	if strictVal, exists := frontmatter["strict"]; exists {
		if strictBool, ok := strictVal.(bool); ok {
			return strictBool
		}
	}
	// Default: strict mode is on when no explicit setting
	return true
}

// effectiveSafeUpdate returns true when safe update mode should be enforced for
// the given workflow. Safe update mode is equivalent to strict mode: it is
// enabled whenever strict mode is active (CLI --strict flag, frontmatter
// strict: true, or the default). It can be disabled via the CLI --approve flag
// to approve all changes.
func (c *Compiler) effectiveSafeUpdate(data *WorkflowData) bool {
	if c.approve {
		return false
	}
	return c.effectiveStrictMode(data.RawFrontmatter)
}
