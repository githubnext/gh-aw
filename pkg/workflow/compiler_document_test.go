//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/workflow/actions"
	"github.com/github/gh-aw/pkg/workflow/aw"
	"github.com/github/gh-aw/pkg/workflow/compilerenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const documentWorkflowSource = `---
name: Document workflow
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
tools:
  bash: [echo]
safe-outputs:
  create-issue:
    max: 1
steps:
  - name: Authored step
    run: echo authored
---

# Mission

Find something worth reporting.
`

func TestCompileDocumentToActionsAST(t *testing.T) {
	document, err := aw.Parse(documentWorkflowSource)
	require.NoError(t, err)
	compiler := NewCompiler(WithNoEmit(true), WithSkipValidation(true))
	actual, err := compiler.CompileDocument(document, "document.md")
	require.NoError(t, err)
	assert.Equal(t, "Document workflow", actual.Name)
	assert.NotEmpty(t, actual.Jobs)
	assert.Contains(t, actual.On, "workflow_dispatch")
	jobs := make(map[actions.JobID]*actions.Job)
	for _, job := range actual.Jobs {
		jobs[job.ID] = job
	}
	assert.Contains(t, jobs, actions.JobID("agent"))
	assert.Contains(t, jobs, actions.JobID("safe_outputs"))
	other := NewCompiler(WithNoEmit(true), WithSkipValidation(true))
	data, err := other.ParseWorkflowString(documentWorkflowSource, "document.md")
	require.NoError(t, err)
	content, err := other.CompileToYAML(data, "document.md")
	require.NoError(t, err)
	expected, err := actions.Parse([]byte(content))
	require.NoError(t, err)
	assert.Equal(t, expected, actual, "document compilation must use the complete existing compiler")
	emitted, err := actions.Emit(actual)
	require.NoError(t, err)
	assert.Contains(t, emitted, "echo authored")
	assert.Contains(t, emitted, "Find something worth reporting.")
	assert.NotContains(t, emitted, "\nengine:")
	assert.NotContains(t, emitted, "\ntools:")
	after, err := aw.Emit(document)
	require.NoError(t, err)
	assert.Equal(t, documentWorkflowSource, after)
}

func TestParseWorkflowDocumentEditsAndTypedConfig(t *testing.T) {
	document, err := aw.Parse(documentWorkflowSource)
	require.NoError(t, err)
	document.Frontmatter["name"] = "Edited document"
	document.Markdown = "\n# Edited mission\n\nUse the edited prompt.\n"
	compiler := NewCompiler(WithNoEmit(true), WithSkipValidation(true))
	data, err := compiler.ParseWorkflowDocument(document, "edited.md")
	require.NoError(t, err)
	assert.Equal(t, "Edited document", data.Name)
	assert.Contains(t, data.RawMarkdown, "Use the edited prompt.")
	require.NotNil(t, data.ParsedFrontmatter)
	require.NotNil(t, data.SourceDocument)
	assert.Equal(t, "Edited document", data.ParsedFrontmatter.Name)
	assert.Equal(t, "copilot", data.SourceDocument.Frontmatter["engine"])
	require.NotNil(t, data.SafeOutputs)
	assert.NotSame(t, document, data.SourceDocument)
	assert.Equal(t, document.Frontmatter, data.SourceDocument.Frontmatter)
}

func TestDocumentConfigurationMatchesFileCompiler(t *testing.T) {
	document, err := aw.Parse(documentWorkflowSource)
	require.NoError(t, err)
	for key, command := range map[string]string{
		"pre-steps": "echo pre", "pre-agent-steps": "echo before agent", "post-steps": "echo post",
	} {
		document.Frontmatter[key] = []any{map[string]any{"run": command}}
	}
	document.Frontmatter["services"] = map[string]any{"redis": map[string]any{"image": "redis:7"}}
	document.Frontmatter["jobs"] = map[string]any{
		"prepare": map[string]any{
			"runs-on": "ubuntu-latest", "steps": []any{map[string]any{"run": "echo prepare"}},
		},
	}
	content, err := aw.Emit(document)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "document.md")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	fileCompiler := NewCompiler(WithNoEmit(true), WithSkipValidation(true))
	expected, err := fileCompiler.ParseWorkflowFile(path)
	require.NoError(t, err)
	documentCompiler := NewCompiler(WithNoEmit(true), WithSkipValidation(true))
	actual, err := documentCompiler.ParseWorkflowDocument(document, path)
	require.NoError(t, err)
	assert.NotEmpty(t, actual.PreSteps)
	assert.NotEmpty(t, actual.PreAgentSteps)
	assert.NotEmpty(t, actual.PostSteps)
	assert.NotEmpty(t, actual.Services)
	assert.Contains(t, actual.Jobs, "prepare")
	assert.Equal(t, expected.PreSteps, actual.PreSteps)
	assert.Equal(t, expected.PreAgentSteps, actual.PreAgentSteps)
	assert.Equal(t, expected.PostSteps, actual.PostSteps)
	assert.Equal(t, expected.Services, actual.Services)
	assert.Equal(t, expected.Jobs, actual.Jobs)
	assert.Equal(t, expected.SafeOutputs, actual.SafeOutputs)
}

func TestSourceDocumentRetainedBeforePreprocessing(t *testing.T) {
	source := "---\nname: Source fidelity\non: daily\nengine: copilot\n---\n\n# Mission  \n\nDo work.  \n"
	path := filepath.Join(t.TempDir(), "workflow.md")
	require.NoError(t, os.WriteFile(path, []byte(source), 0o600))
	for _, mode := range []string{"file", "string"} {
		t.Run(mode, func(t *testing.T) {
			compiler := NewCompiler(WithNoEmit(true), WithSkipValidation(true))
			var data *WorkflowData
			var err error
			if mode == "file" {
				data, err = compiler.ParseWorkflowFile(path)
			} else {
				data, err = compiler.ParseWorkflowString(source, path)
			}
			require.NoError(t, err)
			require.NotNil(t, data.SourceDocument)
			assert.Equal(t, "daily", data.SourceDocument.Frontmatter["on"])
			emitted, err := aw.Emit(data.SourceDocument)
			require.NoError(t, err)
			assert.Equal(t, source, emitted)
		})
	}
}

func TestDocumentRelativeImports(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "shared.md"),
		[]byte("---\ntools:\n  bash: [echo]\n---\n\nImported guidance.\n"), 0o600))
	document, err := aw.Parse("---\non: workflow_dispatch\nengine: copilot\nimports: [shared.md]\n---\n\n# Main mission\n")
	require.NoError(t, err)
	compiler := NewCompiler(WithNoEmit(true), WithSkipValidation(true))
	data, err := compiler.ParseWorkflowDocument(document, filepath.Join(dir, "workflow.md"))
	require.NoError(t, err)
	assert.NotEmpty(t, data.ImportedFiles)
	content, err := compiler.CompileToYAML(data, filepath.Join(dir, "workflow.md"))
	require.NoError(t, err)
	assert.Contains(t, content, "{{#runtime-import")
	assert.Contains(t, content, "shared.md")
}

func TestCompileDocumentValidationErrors(t *testing.T) {
	tests := []struct {
		source string
		want   string
	}{
		{"# Plain Markdown\n", "no frontmatter found"},
		{"---\ntools:\n  bash: [echo]\n---\nShared prompt.\n", "shared"},
		{"---\nredirect: owner/repo/workflow.md@main\n---\n", "redirect"},
		{"---\non: workflow_dispatch\nengine: copilot\nunknown-property: true\n---\nPrompt.\n", "unknown-property"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			document, err := aw.Parse(tt.source)
			require.NoError(t, err)
			compiler := NewCompiler(WithNoEmit(true), WithSkipValidation(true))
			_, err = compiler.CompileDocument(document, "invalid.md")
			require.ErrorContains(t, err, tt.want)
		})
	}
	compiler := NewCompiler(WithNoEmit(true), WithSkipValidation(true))
	_, err := compiler.CompileDocument(nil, "invalid.md")
	require.ErrorContains(t, err, "cannot be nil")
}

func TestCompileDocumentPropagatesCompilationFailures(t *testing.T) {
	t.Setenv(compilerenv.RequireSelfHostedRunners, "true")
	document, err := aw.Parse("---\non: workflow_dispatch\nengine: copilot\n---\n\n# Mission\n")
	require.NoError(t, err)
	compiler := NewCompiler(WithNoEmit(true), WithSkipValidation(true))
	_, err = compiler.CompileDocument(document, "invalid-runner.md")
	require.ErrorContains(t, err, "self-hosted runner")
}
