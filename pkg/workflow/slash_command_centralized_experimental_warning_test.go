//go:build integration

package workflow

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/require"
)

func TestSlashCommandCentralizedExperimentalWarning(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		expectWarning bool
	}{
		{
			name: "centralized strategy emits warning",
			content: `---
on:
  slash_command:
    name: triage
    strategy: centralized
---

# Test Workflow
`,
			expectWarning: true,
		},
		{
			name: "inline strategy does not emit warning",
			content: `---
on:
  slash_command:
    name: triage
---

# Test Workflow
`,
			expectWarning: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "slash-command-centralized-warning-test")
			workflowPath := filepath.Join(tmpDir, "test-workflow.md")
			require.NoError(t, os.WriteFile(workflowPath, []byte(tt.content), 0644))

			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			compiler := NewCompiler()
			compiler.SetStrictMode(false)
			err := compiler.CompileWorkflow(workflowPath)

			w.Close()
			os.Stderr = oldStderr

			var buf bytes.Buffer
			_, _ = io.Copy(&buf, r)
			stderrOutput := buf.String()
			require.NoError(t, err)

			expected := "Using experimental feature: slash_command.strategy: centralized"
			if tt.expectWarning {
				require.Contains(t, stderrOutput, expected)
				require.Greater(t, compiler.GetWarningCount(), 0)
			} else {
				require.NotContains(t, stderrOutput, expected)
			}
		})
	}
}
