//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseWorkflowString_BasicParsing(t *testing.T) {
	markdown := `---
name: hello-world
description: A simple hello world workflow
on:
  workflow_dispatch:
engine: copilot
---

# Mission

Say hello to the world!
`

	compiler := NewCompiler(
		WithNoEmit(true),
		WithSkipValidation(true),
	)

	wd, err := compiler.ParseWorkflowString(markdown, "workflow.md")
	require.NoError(t, err)
	assert.NotNil(t, wd)
	assert.Equal(t, "hello-world", wd.Name)
}

func TestParseWorkflowString_MissingFrontmatter(t *testing.T) {
	markdown := `# Just a heading

No frontmatter here.
`

	compiler := NewCompiler(
		WithNoEmit(true),
		WithSkipValidation(true),
	)

	_, err := compiler.ParseWorkflowString(markdown, "workflow.md")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "frontmatter")
}

func TestParseWorkflowString_InvalidFrontmatterYAML(t *testing.T) {
	markdown := `---
name: [invalid yaml
on: {{{
---

# Broken
`

	compiler := NewCompiler(
		WithNoEmit(true),
		WithSkipValidation(true),
	)

	_, err := compiler.ParseWorkflowString(markdown, "workflow.md")
	require.Error(t, err)
}

func TestParseWorkflowString_SharedWorkflowDetection(t *testing.T) {
	// Shared workflows have no 'on' trigger field
	markdown := `---
name: shared-tools
description: shared component
tools:
  bash: ["echo"]
---

# Shared tools component
`

	compiler := NewCompiler(
		WithNoEmit(true),
		WithSkipValidation(true),
	)

	_, err := compiler.ParseWorkflowString(markdown, "shared/tools.md")
	require.Error(t, err)

	var sharedErr *SharedWorkflowError
	assert.ErrorAs(t, err, &sharedErr, "expected SharedWorkflowError")
}

func TestParseWorkflowString_VirtualPathBehavior(t *testing.T) {
	markdown := `---
name: path-test
on: push
engine: copilot
---

# Path test
`

	compiler := NewCompiler(
		WithNoEmit(true),
		WithSkipValidation(true),
	)

	// The virtual path should be cleaned and used for workflow ID derivation
	wd, err := compiler.ParseWorkflowString(markdown, "some/nested/../workflow.md")
	require.NoError(t, err)
	assert.NotNil(t, wd)
}

func TestCompileToYAML_BasicCompilation(t *testing.T) {
	markdown := `---
name: compile-test
on:
  workflow_dispatch:
engine: copilot
---

# Mission

Greet the user warmly.
`

	compiler := NewCompiler(
		WithNoEmit(true),
		WithSkipValidation(true),
	)

	wd, err := compiler.ParseWorkflowString(markdown, "workflow.md")
	require.NoError(t, err)

	yaml, err := compiler.CompileToYAML(wd, "workflow.md")
	require.NoError(t, err)
	assert.NotEmpty(t, yaml)

	// The generated YAML should contain standard GitHub Actions structure
	assert.Contains(t, yaml, "name:")
	assert.Contains(t, yaml, "on:")
	assert.Contains(t, yaml, "jobs:")
}

func TestCompileToYAML_OutputContainsWorkflowName(t *testing.T) {
	markdown := `---
name: my-unique-workflow
on: push
engine: copilot
---

# Do something
`

	compiler := NewCompiler(
		WithNoEmit(true),
		WithSkipValidation(true),
	)

	wd, err := compiler.ParseWorkflowString(markdown, "workflow.md")
	require.NoError(t, err)

	yaml, err := compiler.CompileToYAML(wd, "workflow.md")
	require.NoError(t, err)
	assert.Contains(t, yaml, "my-unique-workflow")
}

func TestParseWorkflowString_EmptyContent(t *testing.T) {
	compiler := NewCompiler(
		WithNoEmit(true),
		WithSkipValidation(true),
	)

	_, err := compiler.ParseWorkflowString("", "workflow.md")
	require.Error(t, err)
}

func TestParseWorkflowString_FrontmatterOnly(t *testing.T) {
	markdown := `---
name: no-body
on: push
engine: copilot
---
`

	compiler := NewCompiler(
		WithNoEmit(true),
		WithSkipValidation(true),
	)

	// Should parse successfully even without markdown body
	wd, err := compiler.ParseWorkflowString(markdown, "workflow.md")
	require.NoError(t, err)
	assert.NotNil(t, wd)
}

func TestCompileToYAML_EndToEnd(t *testing.T) {
	// Full round trip: markdown string -> parse -> compile -> YAML string
	markdown := `---
name: e2e-test
description: End-to-end string API test
on:
  issues:
    types: [opened]
engine: copilot
---

# Mission

When a new issue is opened, add a welcome comment.
`

	compiler := NewCompiler(
		WithNoEmit(true),
		WithSkipValidation(true),
	)

	wd, err := compiler.ParseWorkflowString(markdown, "workflow.md")
	require.NoError(t, err)

	yaml, err := compiler.CompileToYAML(wd, "workflow.md")
	require.NoError(t, err)

	// Verify the YAML is valid-looking
	// CompileToYAML skips the ASCII art header (wasm/editor mode), so YAML starts with the name
	assert.False(t, strings.Contains(yaml, "Agentic"), "compiled YAML should not contain ASCII art header")
	assert.Contains(t, yaml, "e2e-test")
	assert.Contains(t, yaml, "issues")
}

func TestCompileToYAML_PromptContentInlined(t *testing.T) {
	// Verify that the markdown prompt content is inlined in the compiled YAML
	// when using ParseWorkflowString (the Wasm/browser path).
	// This is the key regression test for the wasm live editor prompt issue.
	markdown := `---
name: hello-world
on:
  workflow_dispatch:
engine: copilot
---

# Mission

Say hello to the world!
`

	compiler := NewCompiler(
		WithNoEmit(true),
		WithSkipValidation(true),
	)

	wd, err := compiler.ParseWorkflowString(markdown, "workflow.md")
	require.NoError(t, err)

	yamlOutput, err := compiler.CompileToYAML(wd, "workflow.md")
	require.NoError(t, err)

	// The prompt content should be inlined in the YAML, not behind a runtime-import macro
	assert.Contains(t, yamlOutput, "# Mission", "compiled YAML should contain the markdown heading")
	assert.Contains(t, yamlOutput, "Say hello to the world!", "compiled YAML should contain the markdown body")
	assert.NotContains(t, yamlOutput, "{{#runtime-import", "compiled YAML from string API should not contain runtime-import macros")
}

func TestCompileToYAML_PromptContentInlinedWithExpressions(t *testing.T) {
	// Verify that GitHub expressions in the markdown prompt are properly handled
	// when inlined (expressions should be extracted and replaced with env vars)
	markdown := `---
name: expr-test
on:
  issues:
    types: [opened]
engine: copilot
---

# Mission

Handle issue ${{ github.event.issue.number }} in repo ${{ github.repository }}.
`

	compiler := NewCompiler(
		WithNoEmit(true),
		WithSkipValidation(true),
	)

	wd, err := compiler.ParseWorkflowString(markdown, "workflow.md")
	require.NoError(t, err)

	yamlOutput, err := compiler.CompileToYAML(wd, "workflow.md")
	require.NoError(t, err)

	// The prompt content should be present (with expressions replaced by env var references)
	assert.Contains(t, yamlOutput, "# Mission", "compiled YAML should contain the markdown heading")
	assert.Contains(t, yamlOutput, "Handle issue", "compiled YAML should contain the prompt text")
	assert.NotContains(t, yamlOutput, "{{#runtime-import", "compiled YAML from string API should not contain runtime-import macros")
}
