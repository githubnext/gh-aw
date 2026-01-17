package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/require"
)

func TestStrictMode_CustomJobWritePermissionsRejected(t *testing.T) {
	tmpDir := testutil.TempDir(t, "strict-custom-job-perms")

	workflowPath := filepath.Join(tmpDir, "custom-job-write.md")
	content := `---
on:
  workflow_dispatch:

permissions:
  contents: read
  issues: read
  pull-requests: read

jobs:
  custom:
    runs-on: ubuntu-latest
    permissions:
      issues: write
    steps:
      - run: echo "hello"
---

# Test`

	require.NoError(t, os.WriteFile(workflowPath, []byte(content), 0644))

	compiler := NewCompiler(false, "", "test")
	compiler.SetStrictMode(true)

	err := compiler.CompileWorkflow(workflowPath)
	require.Error(t, err)
	require.Contains(t, err.Error(), "strict mode")
	require.True(t, strings.Contains(err.Error(), "custom job") || strings.Contains(err.Error(), "custom"))
}

func TestStrictMode_CustomJobReadPermissionsAllowed(t *testing.T) {
	tmpDir := testutil.TempDir(t, "strict-custom-job-perms-allow")

	workflowPath := filepath.Join(tmpDir, "custom-job-read.md")
	content := `---
on:
  workflow_dispatch:

permissions:
  contents: read
  issues: read
  pull-requests: read

jobs:
  custom:
    runs-on: ubuntu-latest
    permissions:
      issues: read
    steps:
      - run: echo "hello"
---

# Test`

	require.NoError(t, os.WriteFile(workflowPath, []byte(content), 0644))

	compiler := NewCompiler(false, "", "test")
	compiler.SetStrictMode(true)
	compiler.SetNoEmit(true)

	require.NoError(t, compiler.CompileWorkflow(workflowPath))
}
