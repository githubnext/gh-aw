//go:build !integration

package workflow

import (
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateTemplateInjection_BothPaths tests that validateTemplateInjection detects
// unsafe expressions in run: blocks consistently regardless of whether schema validation
// is enabled (parsedWorkflow != nil) or disabled (parsedWorkflow == nil).
func TestValidateTemplateInjection_BothPaths(t *testing.T) {
	// YAML with an unsafe GitHub Actions expression directly inside a run: block.
	// This is a template injection risk because ${{ github.event.issue.title }}
	// can contain attacker-controlled content that gets passed to the shell.
	unsafeYAML := `
name: unsafe
on: issues
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Unsafe step
        run: echo "${{ github.event.issue.title }}"
`

	// YAML with an unsafe expression only in an env: block (safe pattern: the
	// value is assigned to a shell variable, not interpolated directly into shell).
	safeYAML := `
name: safe
on: issues
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Safe step
        env:
          TITLE: ${{ github.event.issue.title }}
        run: echo "$TITLE"
`

	tmpDir := testutil.TempDir(t, "template-injection-test")
	markdownPath := filepath.Join(tmpDir, "test.md")
	lockFile := stringutil.MarkdownToLockFile(markdownPath)

	compiler := NewCompiler()

	parseYAML := func(t *testing.T, content string) map[string]any {
		t.Helper()
		var out map[string]any
		require.NoError(t, yaml.Unmarshal([]byte(content), &out), "should parse test YAML")
		return out
	}

	t.Run("Path A - schema enabled - unsafe expression detected", func(t *testing.T) {
		// Path A: parsedWorkflow != nil (pre-parsed for schema validation).
		err := compiler.validateTemplateInjection(unsafeYAML, lockFile, markdownPath, parseYAML(t, unsafeYAML))
		require.Error(t, err, "should detect template injection with pre-parsed YAML")
		assert.Contains(t, err.Error(), "github.event", "error should mention the unsafe context")
	})

	t.Run("Path A - schema enabled - safe expression passes", func(t *testing.T) {
		err := compiler.validateTemplateInjection(safeYAML, lockFile, markdownPath, parseYAML(t, safeYAML))
		assert.NoError(t, err, "safe expression in env: block should not be flagged")
	})

	t.Run("Path B - schema disabled - unsafe expression detected", func(t *testing.T) {
		// Path B: parsedWorkflow == nil (schema validation skipped).
		err := compiler.validateTemplateInjection(unsafeYAML, lockFile, markdownPath, nil)
		require.Error(t, err, "should detect template injection via text scan fallback")
		assert.Contains(t, err.Error(), "github.event", "error should mention the unsafe context")
	})

	t.Run("Path B - schema disabled - safe expression passes", func(t *testing.T) {
		err := compiler.validateTemplateInjection(safeYAML, lockFile, markdownPath, nil)
		assert.NoError(t, err, "safe expression in env: block should not be flagged")
	})

	t.Run("both paths agree on unsafe YAML", func(t *testing.T) {
		errA := compiler.validateTemplateInjection(unsafeYAML, lockFile, markdownPath, parseYAML(t, unsafeYAML))
		errB := compiler.validateTemplateInjection(unsafeYAML, lockFile, markdownPath, nil)

		require.Error(t, errA, "Path A must report an error")
		require.Error(t, errB, "Path B must report an error")
		assert.Contains(t, errA.Error(), "github.event", "Path A error should identify the context")
		assert.Contains(t, errB.Error(), "github.event", "Path B error should identify the context")
	})

	t.Run("both paths agree on safe YAML", func(t *testing.T) {
		errA := compiler.validateTemplateInjection(safeYAML, lockFile, markdownPath, parseYAML(t, safeYAML))
		errB := compiler.validateTemplateInjection(safeYAML, lockFile, markdownPath, nil)

		assert.NoError(t, errA, "Path A should not flag safe YAML")
		assert.NoError(t, errB, "Path B should not flag safe YAML")
	})
}
