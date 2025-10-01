package workflow

import (
"fmt"
"os"
"path/filepath"
"strings"
"testing"
)

func TestDebugOnField(t *testing.T) {
testContent := `---
on:
  issues:
    types: [opened]
  pull_request:
    types: [opened]
  reaction: eyes
permissions:
  contents: read
  issues: write
tools:
  github:
    allowed: [get_issue]
---

# Test On Quote

This workflow tests that the "on" keyword is not quoted.
`

tmpDir, err := os.MkdirTemp("", "debug-test")
if err != nil {
t.Fatal(err)
}
defer os.RemoveAll(tmpDir)

testFile := filepath.Join(tmpDir, "test.md")
if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
t.Fatal(err)
}

compiler := NewCompiler(false, "", "test")
workflowData, err := compiler.parseWorkflowFile(testFile)
if err != nil {
t.Fatalf("Error: %v", err)
}

fmt.Printf("workflowData.On:\n%s\n\n", workflowData.On)

if strings.Contains(workflowData.On, `"on":`) {
t.Error("ERROR: Contains quoted 'on'")
} else if strings.Contains(workflowData.On, "on:") {
fmt.Println("SUCCESS: Contains unquoted 'on'")
}
}
