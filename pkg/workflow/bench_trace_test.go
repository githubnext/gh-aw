//go:build !integration

package workflow

import (
"os"
"path/filepath"
"runtime"
"testing"
)

func TestValidationAllocTrace(t *testing.T) {
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

// Warm up
if err := compiler.validateWorkflowData(workflowData, testFile); err != nil {
t.Fatal(err)
}

// Measure allocs for a single call
var m1, m2 runtime.MemStats
runtime.GC()
runtime.ReadMemStats(&m1)
if err := compiler.validateWorkflowData(workflowData, testFile); err != nil {
t.Fatal(err)
}
runtime.ReadMemStats(&m2)
t.Logf("Allocs: %d, Bytes: %d", m2.Mallocs-m1.Mallocs, m2.TotalAlloc-m1.TotalAlloc)
}
