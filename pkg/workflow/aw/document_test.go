package aw

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocumentRoundTrip(t *testing.T) {
	for _, content := range []string{
		"",
		"# Plain Markdown\n\n  Preserve indentation.  \n",
		"---\nengine: copilot\n---\n\n# Mission\n\n---\n\n```yaml\non: push\n```\n",
		"---\n---",
		"---\n# Shared instructions\n---\n\nPrompt.\n",
		"---\n \t\n---\n",
		"---\nredirect: owner/repo/workflow.md@main\n---",
		"---\nengine: copilot\non: daily\nsafe-outputs:\n  create-issue:\n    max: 1\n---\n\n# Mission\n\nDo work.  \n",
		" \t--- \r\n# Comment\r\non: [push, issues]\r\nengine: {id: copilot}\r\n--- \r\n\r\n# Mission  \r\n",
	} {
		t.Run(content, func(t *testing.T) {
			document, err := Parse(content)
			require.NoError(t, err)
			emitted, err := Emit(document)
			require.NoError(t, err)
			assert.Equal(t, content, emitted)
			expected, err := parser.ExtractFrontmatterFromContent(content)
			require.NoError(t, err)
			assert.Equal(t, expected, document.FrontmatterResult(), "compiler view must preserve parser behavior")
		})
	}
}

func TestDocumentEditsPreserveUntouchedRegions(t *testing.T) {
	content := "---\r\n# Keep this header comment\r\nengine: copilot\r\n---\r\n\r\n# Prompt  \r\n"
	document, err := Parse(content)
	require.NoError(t, err)
	assert.Equal(t, "\r\n# Prompt  \r\n", document.Markdown)
	assert.Equal(t, 2, document.Location.FrontmatterStart)
	assert.Equal(t, 3, document.Location.FieldLines["engine"])
	document.Markdown = "\n# Edited prompt\n"
	emitted, err := Emit(document)
	require.NoError(t, err)
	assert.Equal(t, "---\r\n# Keep this header comment\r\nengine: copilot\r\n---\r\n\n# Edited prompt\n", emitted)
	document, err = Parse(content)
	require.NoError(t, err)
	document.Frontmatter["engine"] = map[string]any{"id": "copilot", "model": "gpt-5.4"}
	emitted, err = Emit(document)
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(emitted, "\r\n# Prompt  \r\n"), "frontmatter edits must not change prompt bytes")
	reparsed, err := Parse(emitted)
	require.NoError(t, err)
	assert.Equal(t, document.Frontmatter, reparsed.Frontmatter)
	assert.Equal(t, document.Markdown, reparsed.Markdown)
}

func TestDocumentProgrammaticConstruction(t *testing.T) {
	document := &Document{
		Frontmatter: map[string]any{
			"on": "workflow_dispatch", "engine": "copilot",
			"tools":        map[string]any{"bash": []any{"echo"}},
			"safe-outputs": map[string]any{"create-issue": map[string]any{"max": 1}},
		},
		Markdown: "\n# Mission\n\nDo work.\n",
	}
	content, err := Emit(document)
	require.NoError(t, err)
	parsed, err := Parse(content)
	require.NoError(t, err)
	assert.Equal(t, document.Markdown, parsed.Markdown)
	assert.Equal(t, "copilot", parsed.Frontmatter["engine"])
	assert.Contains(t, parsed.Frontmatter, "safe-outputs")
	plain, err := Emit(&Document{Markdown: "# Plain\n"})
	require.NoError(t, err)
	assert.Equal(t, "# Plain\n", plain)
}

func TestDocumentParserViewIsIndependent(t *testing.T) {
	source := "---\nengine: copilot\ntools:\n  bash: [echo]\n---\n\n# Prompt\n"
	document, err := Parse(source)
	require.NoError(t, err)
	view := document.FrontmatterResult()
	view.Frontmatter["tools"].(map[string]any)["bash"].([]any)[0] = "changed"
	view.Frontmatter["engine"] = "changed"
	view.FieldLines["engine"] = 100
	view.FrontmatterLines[0] = "changed"
	after, err := Emit(document)
	require.NoError(t, err)
	assert.Equal(t, source, after)
	assert.Equal(t, "copilot", document.Frontmatter["engine"])
	assert.Equal(t, 2, document.Location.FieldLines["engine"])
}

func TestDocumentErrors(t *testing.T) {
	for _, content := range []string{"---\non: push", "---\non: [\n---\nPrompt"} {
		_, err := Parse(content)
		require.Error(t, err)
	}
	_, err := Emit(nil)
	require.ErrorContains(t, err, "cannot be nil")
	_, err = Emit(&Document{Frontmatter: map[string]any{"invalid": make(chan int)}})
	require.Error(t, err)
}

func TestRepositoryMarkdownDocuments(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "..", "..", ".github", "workflows", "*.md"))
	require.NoError(t, err)
	require.NotEmpty(t, files)
	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			source, err := os.ReadFile(file)
			require.NoError(t, err)
			document, err := Parse(string(source))
			require.NoError(t, err)
			emitted, err := Emit(document)
			require.NoError(t, err)
			assert.Equal(t, string(source), emitted, "must preserve every AW field, comment, and prompt byte")
			expected, err := parser.ExtractFrontmatterFromContent(string(source))
			require.NoError(t, err)
			assert.Equal(t, expected, document.FrontmatterResult())
		})
	}
	t.Logf("Round-tripped all %d repository AW Markdown documents", len(files))
}
