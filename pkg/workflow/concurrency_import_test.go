//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConcurrencyJobDiscriminatorImport(t *testing.T) {
	tempDir := t.TempDir()
	sharedPath := filepath.Join(tempDir, "shared.md")
	require.NoError(t, os.WriteFile(sharedPath, []byte(`---
concurrency:
  job-discriminator: ${{ inputs.shared_id }}
---

Shared concurrency configuration.
`), 0o644))

	tests := []struct {
		name                  string
		mainConcurrency       string
		expectedDiscriminator string
	}{
		{
			name:                  "uses imported discriminator",
			expectedDiscriminator: "${{ inputs.shared_id }}",
		},
		{
			name: "main workflow wins",
			mainConcurrency: `concurrency:
  job-discriminator: ${{ inputs.main_id }}
`,
			expectedDiscriminator: "${{ inputs.main_id }}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mainPath := filepath.Join(tempDir, tt.name+".md")
			content := "---\non: workflow_dispatch\nimports:\n  - shared.md\n" + tt.mainConcurrency + "---\n\nMain workflow.\n"
			require.NoError(t, os.WriteFile(mainPath, []byte(content), 0o644))

			data, err := NewCompiler().ParseWorkflowFile(mainPath)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedDiscriminator, data.ConcurrencyJobDiscriminator)
		})
	}
}
