//go:build !integration

package cli

import (
	"testing"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testFactoryLog = logger.New("cli:codemod_factory")

// baseFieldRemovalConfig returns a minimal valid fieldRemovalCodemodConfig for testing.
func baseFieldRemovalConfig() fieldRemovalCodemodConfig {
	return fieldRemovalCodemodConfig{
		ID:           "test-removal",
		Name:         "Remove test field",
		Description:  "Removes the test field for testing purposes",
		IntroducedIn: "1.0.0",
		ParentKey:    "parent",
		FieldKey:     "child",
		LogMsg:       "Applied test field removal",
		Log:          testFactoryLog,
	}
}

func TestNewFieldRemovalCodemod_Metadata(t *testing.T) {
	cfg := baseFieldRemovalConfig()
	codemod := newFieldRemovalCodemod(cfg)

	assert.Equal(t, cfg.ID, codemod.ID, "ID should match config")
	assert.Equal(t, cfg.Name, codemod.Name, "Name should match config")
	assert.Equal(t, cfg.Description, codemod.Description, "Description should match config")
	assert.Equal(t, cfg.IntroducedIn, codemod.IntroducedIn, "IntroducedIn should match config")
	require.NotNil(t, codemod.Apply, "Apply function should not be nil")
}

func TestNewFieldRemovalCodemod_ParentKeyMissing(t *testing.T) {
	codemod := newFieldRemovalCodemod(baseFieldRemovalConfig())

	content := `---
on: workflow_dispatch
other: value
---

# Test`

	frontmatter := map[string]any{
		"on":    "workflow_dispatch",
		"other": "value",
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err, "Apply should not return an error when parent key is missing")
	assert.False(t, applied, "Codemod should not report changes when parent key is missing")
	assert.Equal(t, content, result, "Content should remain unchanged")
}

func TestNewFieldRemovalCodemod_ParentKeyWrongType(t *testing.T) {
	codemod := newFieldRemovalCodemod(baseFieldRemovalConfig())

	content := `---
on: workflow_dispatch
parent: simple_string
---

# Test`

	frontmatter := map[string]any{
		"on":     "workflow_dispatch",
		"parent": "simple_string",
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err, "Apply should not return an error when parent is not a map")
	assert.False(t, applied, "Codemod should not report changes when parent is not a map")
	assert.Equal(t, content, result, "Content should remain unchanged")
}

func TestNewFieldRemovalCodemod_FieldKeyMissing(t *testing.T) {
	codemod := newFieldRemovalCodemod(baseFieldRemovalConfig())

	content := `---
on: workflow_dispatch
parent:
  other: value
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"parent": map[string]any{
			"other": "value",
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err, "Apply should not return an error when child field is missing")
	assert.False(t, applied, "Codemod should not report changes when child field is missing")
	assert.Equal(t, content, result, "Content should remain unchanged")
}

func TestNewFieldRemovalCodemod_SuccessfulRemoval(t *testing.T) {
	codemod := newFieldRemovalCodemod(baseFieldRemovalConfig())

	content := `---
on: workflow_dispatch
parent:
  child: true
  sibling: value
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"parent": map[string]any{
			"child":   true,
			"sibling": "value",
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err, "Apply should not return an error on successful removal")
	assert.True(t, applied, "Codemod should report that changes were applied")
	assert.NotContains(t, result, "child:", "Result should not contain the removed field")
	assert.Contains(t, result, "sibling: value", "Result should preserve sibling fields")
}

func TestNewFieldRemovalCodemod_PostTransformInvoked(t *testing.T) {
	var postTransformCalled bool
	var capturedFieldValue any

	cfg := baseFieldRemovalConfig()
	cfg.PostTransform = func(lines []string, frontmatter map[string]any, fieldValue any) []string {
		postTransformCalled = true
		capturedFieldValue = fieldValue
		// Append a marker line so we can verify the hook ran
		return append(lines, "# post-transform-marker")
	}

	codemod := newFieldRemovalCodemod(cfg)

	content := `---
on: workflow_dispatch
parent:
  child: sentinel
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"parent": map[string]any{
			"child": "sentinel",
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err, "Apply should not return an error")
	assert.True(t, applied, "Codemod should report changes")
	assert.True(t, postTransformCalled, "PostTransform hook should have been called")
	assert.Equal(t, "sentinel", capturedFieldValue, "PostTransform should receive the removed field's value")
	assert.Contains(t, result, "# post-transform-marker", "Result should contain the output of the PostTransform hook")
}

func TestNewFieldRemovalCodemod_PostTransformNotCalledWhenFieldAbsent(t *testing.T) {
	var postTransformCalled bool

	cfg := baseFieldRemovalConfig()
	cfg.PostTransform = func(lines []string, frontmatter map[string]any, fieldValue any) []string {
		postTransformCalled = true
		return lines
	}

	codemod := newFieldRemovalCodemod(cfg)

	content := `---
on: workflow_dispatch
parent:
  other: value
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"parent": map[string]any{
			"other": "value",
		},
	}

	_, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err, "Apply should not return an error")
	assert.False(t, applied, "Codemod should not report changes")
	assert.False(t, postTransformCalled, "PostTransform should not be called when field is absent")
}

func TestNewFieldRemovalCodemod_TableDriven(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		frontmatter map[string]any
		wantApplied bool
		wantContent string // expected substring in result when applied, or full match when not applied
	}{
		{
			name:    "parent key absent",
			content: "---\non: workflow_dispatch\n---\n\n# Test",
			frontmatter: map[string]any{
				"on": "workflow_dispatch",
			},
			wantApplied: false,
		},
		{
			name:    "parent not a map",
			content: "---\non: workflow_dispatch\nparent: scalar\n---\n\n# Test",
			frontmatter: map[string]any{
				"on":     "workflow_dispatch",
				"parent": "scalar",
			},
			wantApplied: false,
		},
		{
			name:    "child field absent",
			content: "---\non: workflow_dispatch\nparent:\n  other: val\n---\n\n# Test",
			frontmatter: map[string]any{
				"on":     "workflow_dispatch",
				"parent": map[string]any{"other": "val"},
			},
			wantApplied: false,
		},
		{
			name:    "child field present",
			content: "---\non: workflow_dispatch\nparent:\n  child: yes\n  other: val\n---\n\n# Test",
			frontmatter: map[string]any{
				"on":     "workflow_dispatch",
				"parent": map[string]any{"child": true, "other": "val"},
			},
			wantApplied: true,
			wantContent: "other: val",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codemod := newFieldRemovalCodemod(baseFieldRemovalConfig())

			result, applied, err := codemod.Apply(tt.content, tt.frontmatter)

			require.NoError(t, err, "Apply should not return an error")
			assert.Equal(t, tt.wantApplied, applied, "Applied flag should match expectation")

			if tt.wantApplied {
				assert.Contains(t, result, tt.wantContent, "Result should contain expected content")
				assert.NotContains(t, result, "child:", "Result should not contain removed field")
			} else {
				assert.Equal(t, tt.content, result, "Content should be unchanged when not applied")
			}
		})
	}
}
