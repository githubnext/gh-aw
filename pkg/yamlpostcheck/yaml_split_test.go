//go:build !integration

package yamlpostcheck

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitYAMLHeader_WithHeader(t *testing.T) {
	content := "# gh-aw-metadata: {}\n# gh-aw-manifest: {}\n#\nname: \"My Workflow\"\njobs: {}\n"
	header, body := SplitYAMLHeader(content)
	assert.Equal(t, "# gh-aw-metadata: {}\n# gh-aw-manifest: {}\n#", header)
	assert.Equal(t, "name: \"My Workflow\"\njobs: {}\n", body)
}

func TestSplitYAMLHeader_NoHeader(t *testing.T) {
	content := "name: \"My Workflow\"\njobs: {}\n"
	header, body := SplitYAMLHeader(content)
	assert.Empty(t, header)
	assert.Equal(t, content, body)
}

func TestSplitYAMLHeader_AllComments(t *testing.T) {
	content := "# just a comment\n# another comment\n"
	header, body := SplitYAMLHeader(content)
	assert.Equal(t, content, header)
	assert.Empty(t, body)
}

func TestJoinYAMLHeaderBody_BothNonEmpty(t *testing.T) {
	header := "# header comment"
	body := "name: foo\n"
	result := JoinYAMLHeaderBody(header, body)
	assert.Equal(t, "# header comment\nname: foo\n", result)
}

func TestJoinYAMLHeaderBody_EmptyHeader(t *testing.T) {
	result := JoinYAMLHeaderBody("", "name: foo\n")
	assert.Equal(t, "name: foo\n", result)
}

func TestJoinYAMLHeaderBody_EmptyBody(t *testing.T) {
	result := JoinYAMLHeaderBody("# header", "")
	assert.Equal(t, "# header", result)
}

func TestJoinYAMLHeaderBody_HeaderEndsWithNewline(t *testing.T) {
	header := "# header\n"
	body := "name: foo\n"
	result := JoinYAMLHeaderBody(header, body)
	// Should not double the newline.
	assert.Equal(t, "# header\nname: foo\n", result)
}

func TestRunOnYAML_NoChanges(t *testing.T) {
	s := New()
	yamlContent := "# gh-aw-manifest: {}\nname: \"My Workflow\"\njobs:\n  agent:\n    steps:\n      - name: safe\n        uses: actions/checkout@v4\n"
	result, changed, fixes, warnings, err := s.RunOnYAML(yamlContent)
	require.NoError(t, err, "should not error on workflow without issues")
	assert.False(t, changed, "safe workflow should not be changed")
	assert.Empty(t, fixes, "no fixes expected")
	assert.Empty(t, warnings, "no warnings expected")
	// The result should be bit-for-bit identical: no re-serialisation when unchanged.
	assert.YAMLEq(t, yamlContent, result, "YAML should be unchanged")
}

func TestRunOnYAML_WithSecretInRun(t *testing.T) {
	s := New()
	yamlContent := `# gh-aw-manifest: {}
name: "My Workflow"
jobs:
  agent:
    steps:
      - name: Call API
        run: |
          curl -H "Authorization: Bearer ${{ secrets.API_TOKEN }}" https://example.com
`
	result, changed, fixes, warnings, err := s.RunOnYAML(yamlContent)
	require.NoError(t, err, "should not error")
	assert.True(t, changed, "workflow with secret in run: should be changed")
	assert.NotEmpty(t, fixes, "should have fixes")
	assert.Empty(t, warnings, "should have no warnings")

	// Header should be preserved.
	assert.Contains(t, result, "# gh-aw-manifest: {}", "header should be preserved")

	// The env: block should map API_TOKEN to the original secret expression.
	assert.Contains(t, result, "API_TOKEN", "env var name should appear in output")
	assert.Contains(t, result, "${{ secrets.API_TOKEN }}", "env: block should contain the expression")

	// The shell variable reference should appear (in the run: block, replacing the expression).
	assert.Contains(t, result, "$API_TOKEN", "run: should reference the env var")
}

func TestRunOnYAML_EmptyYAML(t *testing.T) {
	s := New()
	result, changed, fixes, _, err := s.RunOnYAML("")
	require.NoError(t, err, "empty YAML should not error")
	assert.False(t, changed, "empty YAML should not change")
	assert.Empty(t, fixes)
	assert.Empty(t, result)
}
