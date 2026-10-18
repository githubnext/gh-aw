package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func TestRunGraderFromUnifiedSession(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node unavailable")
	}
	workflowID := writeGraderRunWorkflow(t, "---\non: workflow_dispatch\n---\nTest.\n")
	provenance := `"provenance":{"component":"agent","phase":"agent","path":"agent-session.jsonl","index":0}`
	events := []string{
		`{"type":"tool.execution_start","data":{"toolName":"view","input":{"path":"a"}},` + provenance + `}`,
		`{"type":"tool.execution_start","data":{"toolName":"view","input":{"path":"a"}},` + provenance + `}`,
	}
	for _, input := range []string{strings.Join(events, "\n"), "[" + strings.Join(events, ",") + "]"} {
		var output bytes.Buffer
		require.NoError(t, runGrader(context.Background(), graderRunConfig{
			Workflow: workflowID, GraderID: "loops", Input: strings.NewReader(input), Output: &output,
		}))
		var result struct {
			ID     string
			Value  float64
			Status string
			Source string
		}
		require.NoError(t, json.Unmarshal(output.Bytes(), &result))
		assert.Equal(t, "loops", result.ID)
		assert.InDelta(t, 1.0, result.Value, 1e-9)
		assert.Equal(t, "pass", result.Status)
		assert.Equal(t, "builtin", result.Source)
	}
}

func TestRunGraderRejectsCustomCode(t *testing.T) {
	for _, entry := range []string{
		"custom:\n    script: return 1",
		"operational-value:\n    run: evaluator.sh",
		"loops:\n    script: return 1",
	} {
		t.Run(entry, func(t *testing.T) {
			workflowID := writeGraderRunWorkflow(t, "---\non: workflow_dispatch\ngraders:\n  "+entry+"\n---\nTest.\n")
			var output bytes.Buffer
			err := runGrader(context.Background(), graderRunConfig{
				Workflow: workflowID, GraderID: "loops", Input: strings.NewReader("[]"), Output: &output,
			})
			require.Error(t, err)
			require.Contains(t, err.Error(), "graders")
			assert.Empty(t, output.String())
		})
	}
}

func TestLoadGraderRunDefinitionResolvesImports(t *testing.T) {
	workflowID := writeGraderRunWorkflow(t, "---\non: workflow_dispatch\nimports:\n  - shared/graders.md\ngraders:\n  loops:\n    threshold: 9\n---\nTest.\n")
	require.NoError(t, os.MkdirAll(".github/workflows/shared", 0o755))
	require.NoError(t, os.WriteFile(".github/workflows/shared/graders.md", []byte("---\ngraders:\n  loops:\n    threshold: 2\n  recurrence-determinism: {}\n---\n"), 0o600))
	loops, err := loadGraderRunDefinition(workflowID, "loops")
	require.NoError(t, err)
	assert.InDelta(t, 9.0, *loops.Threshold, 1e-9)
	trajectory, err := loadGraderRunDefinition(workflowID, "recurrence-determinism")
	require.NoError(t, err)
	assert.Equal(t, "builtin", trajectory.Source)
}

func TestReadGraderPayloadValidation(t *testing.T) {
	for _, input := range []string{"", "null", "{}", `{"toolCalls":[]}`, "[null]", "[1]", `[{"type":"x","data":{}}]`, "{invalid", "{}\ninvalid"} {
		_, err := readGraderPayload(strings.NewReader(input), "test")
		require.Error(t, err, input)
	}
	result, err := readGraderPayload(strings.NewReader("[]"), "test")
	require.NoError(t, err)
	assert.JSONEq(t, "[]", string(result))
}

func TestGraderNodePermissionFlag(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node unavailable")
	}
	flag, err := graderNodePermissionFlag(context.Background(), node)
	require.NoError(t, err)
	assert.Contains(t, []string{"--permission", "--experimental-permission"}, flag)
}

func TestGraderNodePermissionFlagProbeFailure(t *testing.T) {
	_, err := graderNodePermissionFlag(context.Background(), "/nonexistent/node")
	require.ErrorContains(t, err, "failed to detect Node.js permission support")
}
