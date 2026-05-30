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

func TestSlashCommandStrategiesDoNotEmitExperimentalWarning(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name: "centralized strategy does not emit warning",
			content: `---
on:
  slash_command:
    name: triage
    strategy: centralized
---

# Test Workflow
`,
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
		},
		{
			name: "label decentralized strategy does not emit warning",
			content: `---
on:
  label_command:
    name: triage
    strategy: decentralized
---

# Test Workflow
`,
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

			require.NotContains(t, stderrOutput, "Using experimental feature: slash_command.strategy: centralized")
			require.NotContains(t, stderrOutput, "Using experimental feature: label_command.strategy: decentralized")
		})
	}
}
