package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSandboxMountsImportedFromSharedWorkflows(t *testing.T) {
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0o755))

	sharedOne := `---
sandbox:
  agent:
    mounts:
      - /tmp/a:/tmp/a:ro
      - /tmp/b:/tmp/b:ro
---

# shared one
`
	sharedTwo := `---
sandbox:
  agent:
    mounts:
      - /tmp/b:/tmp/b:ro
      - /tmp/c:/tmp/c:ro
---

# shared two
`
	mainWorkflow := `---
on: issues
imports:
  - ./shared-one.md
  - ./shared-two.md
sandbox:
  agent:
    mounts:
      - /tmp/c:/tmp/c:ro
      - /tmp/d:/tmp/d:ro
---

# main
`

	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "shared-one.md"), []byte(sharedOne), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "shared-two.md"), []byte(sharedTwo), 0o644))
	mainPath := filepath.Join(workflowsDir, "main.md")
	require.NoError(t, os.WriteFile(mainPath, []byte(mainWorkflow), 0o644))

	compiler := NewCompiler()
	workflowData, err := compiler.ParseWorkflowFile(mainPath)
	require.NoError(t, err)
	require.NotNil(t, workflowData.SandboxConfig)
	require.NotNil(t, workflowData.SandboxConfig.Agent)

	assert.ElementsMatch(t,
		[]string{
			"/tmp/a:/tmp/a:ro",
			"/tmp/b:/tmp/b:ro",
			"/tmp/c:/tmp/c:ro",
			"/tmp/d:/tmp/d:ro",
		},
		workflowData.SandboxConfig.Agent.Mounts,
	)
	assert.Len(t, workflowData.SandboxConfig.Agent.Mounts, 4)
}

