package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActivationOutputsCodemod(t *testing.T) {
	codemod := getActivationOutputsCodemod()

	tests := []struct {
		name        string
		content     string
		expected    string
		shouldApply bool
	}{
		{
			name: "transform text output",
			content: `---
engine: copilot
---

Analyze this content: "${{ needs.activation.outputs.text }}"
`,
			expected: `---
engine: copilot
---

Analyze this content: "${{ steps.sanitized.outputs.text }}"
`,
			shouldApply: true,
		},
		{
			name: "transform title output",
			content: `---
engine: copilot
---

Title: "${{ needs.activation.outputs.title }}"
`,
			expected: `---
engine: copilot
---

Title: "${{ steps.sanitized.outputs.title }}"
`,
			shouldApply: true,
		},
		{
			name: "transform body output",
			content: `---
engine: copilot
---

Body: "${{ needs.activation.outputs.body }}"
`,
			expected: `---
engine: copilot
---

Body: "${{ steps.sanitized.outputs.body }}"
`,
			shouldApply: true,
		},
		{
			name: "transform multiple outputs",
			content: `---
engine: copilot
---

Text: "${{ needs.activation.outputs.text }}"
Title: "${{ needs.activation.outputs.title }}"
Body: "${{ needs.activation.outputs.body }}"
`,
			expected: `---
engine: copilot
---

Text: "${{ steps.sanitized.outputs.text }}"
Title: "${{ steps.sanitized.outputs.title }}"
Body: "${{ steps.sanitized.outputs.body }}"
`,
			shouldApply: true,
		},
		{
			name: "do not transform partial matches",
			content: `---
engine: copilot
---

Custom: "${{ needs.activation.outputs.text_custom }}"
`,
			expected: `---
engine: copilot
---

Custom: "${{ needs.activation.outputs.text_custom }}"
`,
			shouldApply: false,
		},
		{
			name: "do not transform other activation outputs",
			content: `---
engine: copilot
---

ID: "${{ needs.activation.outputs.comment_id }}"
Repo: "${{ needs.activation.outputs.comment_repo }}"
`,
			expected: `---
engine: copilot
---

ID: "${{ needs.activation.outputs.comment_id }}"
Repo: "${{ needs.activation.outputs.comment_repo }}"
`,
			shouldApply: false,
		},
		{
			name: "transform with operators",
			content: `---
engine: copilot
---

Text: "${{ needs.activation.outputs.text || 'default' }}"
`,
			expected: `---
engine: copilot
---

Text: "${{ steps.sanitized.outputs.text || 'default' }}"
`,
			shouldApply: true,
		},
		{
			name: "transform partial match followed by valid match",
			content: `---
engine: copilot
---

Custom: "${{ needs.activation.outputs.text_custom }}"
Valid: "${{ needs.activation.outputs.text }}"
`,
			expected: `---
engine: copilot
---

Custom: "${{ needs.activation.outputs.text_custom }}"
Valid: "${{ steps.sanitized.outputs.text }}"
`,
			shouldApply: true,
		},
		{
			name: "no transformation needed",
			content: `---
engine: copilot
---

Already updated: "${{ steps.sanitized.outputs.text }}"
`,
			expected: `---
engine: copilot
---

Already updated: "${{ steps.sanitized.outputs.text }}"
`,
			shouldApply: false,
		},
		{
			name: "no activation outputs",
			content: `---
engine: copilot
---

Some other content: "${{ github.event.issue.number }}"
`,
			expected: `---
engine: copilot
---

Some other content: "${{ github.event.issue.number }}"
`,
			shouldApply: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, applied, err := codemod.Apply(tt.content, nil)

			require.NoError(t, err, "Codemod should not return an error")
			assert.Equal(t, tt.expected, result, "Content should match expected output")
			assert.Equal(t, tt.shouldApply, applied, "Applied status should match expected")
		})
	}
}

func TestActivationOutputsCodemodMetadata(t *testing.T) {
	codemod := getActivationOutputsCodemod()

	assert.Equal(t, "activation-outputs-to-sanitized-step", codemod.ID)
	assert.Equal(t, "Transform activation outputs to sanitized step", codemod.Name)
	assert.Contains(t, codemod.Description, "needs.activation.outputs")
	assert.Contains(t, codemod.Description, "steps.sanitized.outputs")
	assert.NotEmpty(t, codemod.IntroducedIn)
}
