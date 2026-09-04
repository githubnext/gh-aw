package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeGraderRunWorkflow(t *testing.T, content string) string {
	t.Helper()
	root := t.TempDir()
	workflowDir := filepath.Join(root, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(workflowDir, "test.md"), []byte(content), 0o600))
	previous, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(root))
	t.Cleanup(func() { require.NoError(t, os.Chdir(previous)) })
	return "test"
}

func TestRunGraderFromStdin(t *testing.T) {
	workflowID := writeGraderRunWorkflow(t, "---\ngraders: {}\n---\n")
	var output bytes.Buffer
	err := runGrader(context.Background(), graderRunConfig{
		Workflow: workflowID,
		GraderID: "loops",
		Input: bytes.NewBufferString(`{
			"toolCalls":[
				{"name":"view","arguments":{"path":"a"}},
				{"name":"view","arguments":{"path":"a"}}
			],
			"tokenUsageEntries":[],
			"retryEvents":[],
			"artifacts":[]
		}`),
		Output: &output,
	})
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"id":"loops",
		"name":"Loops",
		"value":1,
		"unit":"count",
		"passed":true,
		"status":"pass",
		"source":"builtin",
		"implementation":{"id":"gh-aw/graders","version":1}
	}`, output.String())
}

func TestRunCustomGraderFromStdin(t *testing.T) {
	workflowID := writeGraderRunWorkflow(t, `---
graders:
  custom-score:
    script: |
      return { value: trace.score, message: "computed" }
    unit: ratio
    direction: higher_is_better
    threshold: 0.5
---
`)
	var output bytes.Buffer
	err := runGrader(context.Background(), graderRunConfig{
		Workflow: workflowID,
		GraderID: "custom-score",
		Input:    bytes.NewBufferString(`{"score":0.75}`),
		Output:   &output,
	})
	require.NoError(t, err)
	assert.Contains(t, output.String(), `"value": 0.75`)
	assert.Contains(t, output.String(), `"message": "computed"`)
}

func TestReadGraderPayloadValidation(t *testing.T) {
	_, err := readGraderPayload(bytes.NewBufferString("not-json"), "standard input")
	require.ErrorContains(t, err, "not valid JSON")

	_, err = parseGraderRunID("0")
	require.ErrorContains(t, err, "positive integer")
}
