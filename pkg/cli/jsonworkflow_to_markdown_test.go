//go:build !integration

package cli

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertJSONWorkflowToMarkdown_Basic(t *testing.T) {
	wf := &JSONWorkflow{
		ID:           "my-workflow",
		Name:         "My Workflow",
		Description:  "Does something useful",
		Instructions: "Step 1: Do A\nStep 2: Do B",
		Engine:       "copilot",
	}
	gen, err := ConvertJSONWorkflowToMarkdown(wf, ConvertOptions{})
	require.NoError(t, err)
	assert.Equal(t, "my-workflow", gen.Filename)
	assert.Contains(t, gen.Markdown, "description: Does something useful")
	assert.Contains(t, gen.Markdown, "engine: copilot")
	assert.Contains(t, gen.Markdown, "# My Workflow")
	assert.Contains(t, gen.Markdown, "Step 1: Do A")
	assert.Empty(t, gen.Warnings)
}

func TestConvertJSONWorkflowToMarkdown_FallbackToName(t *testing.T) {
	wf := &JSONWorkflow{
		Name:         "Weekly Research",
		Instructions: "Do research",
	}
	gen, err := ConvertJSONWorkflowToMarkdown(wf, ConvertOptions{})
	require.NoError(t, err)
	assert.Equal(t, "weekly-research", gen.Filename)
}

func TestConvertJSONWorkflowToMarkdown_NameOverride(t *testing.T) {
	wf := &JSONWorkflow{
		ID:   "original-id",
		Name: "Original Name",
	}
	gen, err := ConvertJSONWorkflowToMarkdown(wf, ConvertOptions{NameOverride: "custom-name"})
	require.NoError(t, err)
	assert.Equal(t, "custom-name", gen.Filename)
}

func TestConvertJSONWorkflowToMarkdown_NoIDOrName(t *testing.T) {
	wf := &JSONWorkflow{
		Instructions: "Just instructions",
	}
	gen, err := ConvertJSONWorkflowToMarkdown(wf, ConvertOptions{})
	require.NoError(t, err)
	assert.Equal(t, "imported-workflow", gen.Filename)
}

func TestConvertJSONWorkflowToMarkdown_Tags(t *testing.T) {
	wf := &JSONWorkflow{
		ID:   "tagged",
		Tags: []string{"automation", "ci"},
	}
	gen, err := ConvertJSONWorkflowToMarkdown(wf, ConvertOptions{})
	require.NoError(t, err)
	assert.Contains(t, gen.Markdown, "tags:")
	assert.Contains(t, gen.Markdown, "- automation")
	assert.Contains(t, gen.Markdown, "- ci")
}

func TestConvertJSONWorkflowToMarkdown_OnField(t *testing.T) {
	wf := &JSONWorkflow{
		ID: "triggered",
		On: map[string]any{"push": nil, "pull_request": nil},
	}
	gen, err := ConvertJSONWorkflowToMarkdown(wf, ConvertOptions{})
	require.NoError(t, err)
	assert.Contains(t, gen.Markdown, "on:")
}

func TestConvertJSONWorkflowToMarkdown_ExtraFieldsPreserved(t *testing.T) {
	raw := `{"id":"extra-wf","name":"Extra WF","unknown_field":"some_value","another":42}`
	var wf JSONWorkflow
	require.NoError(t, json.Unmarshal([]byte(raw), &wf))

	gen, err := ConvertJSONWorkflowToMarkdown(&wf, ConvertOptions{})
	require.NoError(t, err)

	// Unknown fields must appear as comments in the markdown.
	assert.Contains(t, gen.Markdown, "# Unsupported fields", "expected comment header for unsupported fields")
	assert.Contains(t, gen.Markdown, "# ", "expected comment lines")
	// Warnings must be reported for each unknown field.
	assert.Len(t, gen.Warnings, 2, "expected one warning per unknown field")
}

func TestConvertJSONWorkflowToMarkdown_NilInput(t *testing.T) {
	_, err := ConvertJSONWorkflowToMarkdown(nil, ConvertOptions{})
	require.Error(t, err)
}

func TestConvertJSONWorkflowToMarkdown_FrontmatterValid(t *testing.T) {
	wf := &JSONWorkflow{
		ID:           "valid-fm",
		Description:  "description with: colon",
		Instructions: "body",
	}
	gen, err := ConvertJSONWorkflowToMarkdown(wf, ConvertOptions{})
	require.NoError(t, err)
	// Colons in description values must be quoted.
	assert.Contains(t, gen.Markdown, `"description with: colon"`)
}

func TestConvertJSONWorkflowToMarkdown_NewlineInDescription(t *testing.T) {
	wf := &JSONWorkflow{
		ID:          "newline-desc",
		Description: "line one\nline two",
	}
	gen, err := ConvertJSONWorkflowToMarkdown(wf, ConvertOptions{})
	require.NoError(t, err)
	// Newlines must be escaped, not embedded literally, so the frontmatter stays valid.
	assert.Contains(t, gen.Markdown, `"line one\nline two"`)
	assert.NotContains(t, gen.Markdown, "line one\nline two")
}

func TestYamlQuoteString_BackslashN(t *testing.T) {
	// A literal backslash followed by 'n' (not a newline) must survive round-trip
	// as '\\n' inside a double-quoted YAML scalar.
	result := yamlQuoteString(`has\nbackslash`)
	// No quoting needed for a plain backslash-n, but if quoted it must be \\n.
	// Either way the result must not collapse the two characters into a single newline.
	assert.NotContains(t, result, "\n")
	assert.Contains(t, result, `\n`)
}

func TestJSONWorkflow_UnmarshalJSON_CapturesExtra(t *testing.T) {
	raw := `{"id":"w","name":"N","unknown_key":"val","nested":{"a":1}}`
	var wf JSONWorkflow
	require.NoError(t, json.Unmarshal([]byte(raw), &wf))
	assert.Equal(t, "w", wf.ID)
	assert.Equal(t, "N", wf.Name)
	assert.Contains(t, wf.Extra, "unknown_key")
	assert.Contains(t, wf.Extra, "nested")
}

func TestToKebabCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"My Workflow", "my-workflow"},
		{"my_workflow", "my-workflow"},
		{"My Workflow Name!", "my-workflow-name"},
		{"already-kebab", "already-kebab"},
		{"  spaces  ", "spaces"},
		{"Mixed_Case And Underscores", "mixed-case-and-underscores"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.want, toKebabCase(tc.input))
		})
	}
}

func TestGenericURLWorkflowName(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://example.com/my-workflow.md", "my-workflow"},
		{"https://example.com/workflows/daily-report.json", "daily-report"},
		{"https://example.com/", "imported-workflow"},
		{"https://example.com/workflow.yaml", "workflow"},
		// url.Parse treats bare strings as relative paths; "not-a-url" has no extension.
		{"not-a-url", "not-a-url"},
		// Spaces and mixed-case should be kebab-cased.
		{"https://example.com/My%20Workflow.json", "my-workflow"},
		{"https://example.com/My_Workflow.md", "my-workflow"},
		{"https://example.com/Weekly Report.json", "weekly-report"},
	}
	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			assert.Equal(t, tc.want, genericURLWorkflowName(tc.url))
		})
	}
}
