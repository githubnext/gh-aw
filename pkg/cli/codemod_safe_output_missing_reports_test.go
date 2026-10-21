//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func missingReportFrontmatter(t *testing.T, content string) map[string]any {
	t.Helper()
	lines, _, err := parseFrontmatterLines(content)
	require.NoError(t, err)
	var frontmatter map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(strings.Join(lines, "\n")), &frontmatter))
	return frontmatter
}

func TestSafeOutputMissingReportsCodemod(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"tool", "data"} {
		t.Run(kind, func(t *testing.T) {
			codemod := getSafeOutputMissingDataCodemod()
			if kind == "tool" {
				codemod = getSafeOutputMissingToolCodemod()
			}
			target := "report-missing-" + kind
			assert.Equal(t, "safe-output-"+target, codemod.ID)
			assert.NotEmpty(t, codemod.Name)
			assert.NotEmpty(t, codemod.Description)
			assert.Equal(t, "1.5.0", codemod.IntroducedIn)
			require.NotNil(t, codemod.Apply)
			for _, tc := range []struct{ name, input, expected string }{
				{"block", "safe-outputs:\n  missing-" + kind + ":\n    max: 2", "safe-outputs:\n  " + target + ":\n    max: 2"},
				{"false", "safe-outputs:\n    missing-" + kind + ": false # disable", "safe-outputs:\n    " + target + ": false # disable"},
				{"null and comments", "safe-outputs:\n  # explanation\n  missing-" + kind + ": # defaults", "safe-outputs:\n  # explanation\n  " + target + ": # defaults"},
				{"flow", "safe-outputs: {missing-" + kind + ": {max: 2}, noop: {}}", "safe-outputs: {" + target + ": {max: 2}, noop: {}}"},
				{"single quoted key", "'safe-outputs':\n  'missing-" + kind + "': {}", "'safe-outputs':\n  '" + target + "': {}"},
				{"double quoted key", "safe-outputs:\n  \"missing-" + kind + "\": {}", "safe-outputs:\n  \"" + target + "\": {}"},
				{"unicode before flow key", "safe-outputs: {create-issue: {title-prefix: \"你好\"}, missing-" + kind + ": false}", "safe-outputs: {create-issue: {title-prefix: \"你好\"}, " + target + ": false}"},
				{"nested names untouched", "safe-outputs:\n  create-issue:\n    missing-" + kind + ": nested\n  missing-" + kind + ": {}", "safe-outputs:\n  create-issue:\n    missing-" + kind + ": nested\n  " + target + ": {}"},
				{"block scalar untouched", "description: |\n  safe-outputs:\n    missing-" + kind + ": false\nsafe-outputs:\n  missing-" + kind + ": {}", "description: |\n  safe-outputs:\n    missing-" + kind + ": false\nsafe-outputs:\n  " + target + ": {}"},
			} {
				t.Run(tc.name, func(t *testing.T) {
					body := "\n\nDo not rewrite missing-" + kind + ":\n```yaml\nsafe-outputs:\n  missing-" + kind + ": false\n```\n"
					input := "---\n" + tc.input + "\n---" + body
					result, applied, err := codemod.Apply(input, missingReportFrontmatter(t, input))
					require.NoError(t, err)
					require.True(t, applied)
					assert.Equal(t, "---\n"+tc.expected+"\n---"+body, result)
					again, applied, err := codemod.Apply(result, missingReportFrontmatter(t, result))
					require.NoError(t, err)
					assert.False(t, applied)
					assert.Equal(t, result, again)
				})
			}
		})
	}
}

func TestSafeOutputMissingReportsCodemodNoop(t *testing.T) {
	t.Parallel()
	for _, content := range []string{
		"on: push",
		"safe-outputs: null",
		"safe-outputs:\n  report-missing-tool: false\n  report-missing-data: {}",
		"safe-outputs:\n  create-issue:\n    missing-tool: nested",
		"other:\n  safe-outputs:\n    missing-tool: false",
		"description: |\n  safe-outputs:\n    missing-tool: false",
	} {
		content = "---\n" + content + "\n---\n"
		for _, codemod := range []Codemod{getSafeOutputMissingToolCodemod(), getSafeOutputMissingDataCodemod()} {
			result, applied, err := codemod.Apply(content, missingReportFrontmatter(t, content))
			require.NoError(t, err)
			assert.False(t, applied)
			assert.Equal(t, content, result)
		}
	}
}

func TestSafeOutputMissingReportsCodemodPluralAndConflicts(t *testing.T) {
	t.Parallel()
	codemod := getSafeOutputMissingToolCodemod()
	content := "---\nsafe-outputs:\n  missing-tools: {max: 2}\n---\n"
	result, applied, err := codemod.Apply(content, missingReportFrontmatter(t, content))
	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "  report-missing-tool: {max: 2}")
	for _, keys := range []string{
		"  missing-tool: {}\n  report-missing-tool: false",
		"  missing-tools: {}\n  missing-tool: false",
	} {
		content := "---\nsafe-outputs:\n" + keys + "\n---\n"
		result, applied, err := codemod.Apply(content, missingReportFrontmatter(t, content))
		require.ErrorContains(t, err, "conflicting")
		assert.False(t, applied)
		assert.Equal(t, content, result)
	}

}

func TestFixMissingReportsFrontmatter(t *testing.T) {
	t.Parallel()
	file := filepath.Join(t.TempDir(), "missing-reports.md")
	content := "---\non: workflow_dispatch\nsafe-outputs:\n  missing-tools: false # intentionally disabled\n  missing-data: {max: 2}\n---\nKeep missing-tools in the body.\n"
	require.NoError(t, os.WriteFile(file, []byte(content), 0600))
	codemods := []Codemod{getSafeOutputMissingToolCodemod(), getSafeOutputMissingDataCodemod()}
	applied, fixes, err := processWorkflowFileWithInfo(file, codemods, false, false)
	require.NoError(t, err)
	assert.True(t, applied)
	assert.Len(t, fixes, 2)
	unchanged, err := os.ReadFile(file)
	require.NoError(t, err)
	assert.Equal(t, content, string(unchanged), "dry run must not write")
	applied, fixes, err = processWorkflowFileWithInfo(file, codemods, true, false)
	require.NoError(t, err)
	assert.True(t, applied)
	assert.Len(t, fixes, 2)
	migrated, err := os.ReadFile(file)
	require.NoError(t, err)
	assert.Equal(t, "---\non: workflow_dispatch\nsafe-outputs:\n  report-missing-tool: false # intentionally disabled\n  report-missing-data: {max: 2}\n---\nKeep missing-tools in the body.\n", string(migrated))
	require.NoError(t, parser.ValidateMainWorkflowFrontmatterWithSchemaAndLocation(missingReportFrontmatter(t, string(migrated)), file))
	applied, fixes, err = processWorkflowFileWithInfo(file, codemods, true, false)
	require.NoError(t, err)
	assert.False(t, applied)
	assert.Empty(t, fixes)
}
