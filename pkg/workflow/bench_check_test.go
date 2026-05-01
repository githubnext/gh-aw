//go:build !integration

package workflow

import (
"fmt"
"os"
"path/filepath"
"testing"
)

func TestValidationState(t *testing.T) {
tmpDir, err := os.MkdirTemp("", "bench-validate")
if err != nil {
t.Fatal(err)
}
defer os.RemoveAll(tmpDir)

testContent := `---
on: pull_request
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
tools:
  github:
    mode: remote
    toolsets: [default]
  bash: ["git status"]
strict: true
timeout-minutes: 10
---

# Validation Benchmark

Test validation performance.
`

testFile := filepath.Join(tmpDir, "validate.md")
if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
t.Fatal(err)
}

compiler := NewCompiler(WithNoEmit(true))
compiler.SetStrictMode(true)
compiler.SetQuiet(true)

workflowData, err := compiler.ParseWorkflowFile(testFile)
if err != nil {
t.Fatal(err)
}

fmt.Fprintf(os.Stderr, "SafeOutputs: %v\n", workflowData.SafeOutputs)
fmt.Fprintf(os.Stderr, "NetworkPermissions: %v\n", workflowData.NetworkPermissions)
fmt.Fprintf(os.Stderr, "CachedPermissions: %v\n", workflowData.CachedPermissions != nil)
fmt.Fprintf(os.Stderr, "CachedParsedToolsets: %v\n", workflowData.CachedParsedToolsets)
fmt.Fprintf(os.Stderr, "CachedConcurrencyGroupExprSet: %v\n", workflowData.CachedConcurrencyGroupExprSet)
fmt.Fprintf(os.Stderr, "Concurrency: %q\n", workflowData.Concurrency)
fmt.Fprintf(os.Stderr, "Features: %v\n", workflowData.Features)
}
