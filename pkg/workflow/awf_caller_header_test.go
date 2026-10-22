//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUnsupportedEngineDoesNotMislabelSubagents is the
// unsupported_engine_does_not_mislabel_subagents acceptance test. The Copilot
// CLI, Copilot SDK, Claude Code and Codex header mechanisms are provider- or
// process-wide, so a compiled `x-awf-caller: main` would also label their
// sub-agents' requests. None is configured; only Pi declares callers, from each
// Pi process's own provider hook (pi_provider.cjs).
func TestUnsupportedEngineDoesNotMislabelSubagents(t *testing.T) {
	const agent = "\n## agent: `haiku-whoami`\n---\ndescription: Reports its model\nmodel: small\n---\nSay which model you are.\n"
	tests := []struct {
		name   string
		engine string
	}{
		{name: "copilot cli", engine: "engine: copilot\n"},
		{name: "copilot sdk", engine: "engine:\n  id: copilot\n  copilot-sdk: true\n"},
		{name: "claude", engine: "engine: claude\n"},
		{name: "codex", engine: "engine: codex\n"},
		{name: "pi", engine: "engine: pi\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := "---\non: workflow_dispatch\nstrict: false\npermissions:\n  contents: read\n" + tt.engine + "---\n# Work\nDelegate to haiku-whoami.\n" + agent
			source := filepath.Join(t.TempDir(), "workflow.md")
			require.NoError(t, os.WriteFile(source, []byte(content), 0o600))
			compiler := NewCompiler()
			data, err := compiler.ParseWorkflowFile(source)
			require.NoError(t, err)
			generated, err := compiler.CompileToYAML(data, source)
			require.NoError(t, err)
			lower := strings.ToLower(generated)
			assert.NotContains(t, lower, "x-awf-caller", "no process- or provider-wide caller header is compiled")
			for _, variable := range []string{"ANTHROPIC_CUSTOM_HEADERS", "COPILOT_PROVIDER_HEADERS"} {
				assert.NotContains(t, generated, variable, "%s would apply to every request of the process", variable)
			}
			if tt.name == "pi" {
				assert.Contains(t, generated, "pi_provider.cjs", "Pi declares callers from its per-process provider extension")
			}
		})
	}
}
