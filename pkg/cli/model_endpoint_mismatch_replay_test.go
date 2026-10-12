//go:build !integration

package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// modelEndpointMismatchReplayDir holds redacted evidence from run 38105198559
// (routing-test-pr67487-mismatch). actions/setup/js/copilot_model_endpoint_mismatch_replay.test.cjs
// checks that the real harness and collector produce these files.
var modelEndpointMismatchReplayDir = filepath.Join("testdata", "model_endpoint_mismatch_startup")

// writeModelEndpointMismatchReplayRun lays out the run's artifacts as audit downloads them.
func writeModelEndpointMismatchReplayRun(t *testing.T, session []byte) string {
	t.Helper()
	runDir := t.TempDir()
	reflect, err := os.ReadFile(filepath.Join(modelEndpointMismatchReplayDir, "awf-reflect.json"))
	require.NoError(t, err)
	record, err := os.ReadFile(filepath.Join(modelEndpointMismatchReplayDir, "model-endpoint-mismatch.json"))
	require.NoError(t, err)
	for relative, content := range map[string][]byte{
		"sandbox/firewall/awf-reflect.json":  reflect,
		"agent/model-endpoint-mismatch.json": record,
		"usage/aw_session.jsonl":             session,
	} {
		path := filepath.Join(runDir, relative)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
		require.NoError(t, os.WriteFile(path, content, 0o600))
	}
	return runDir
}

func TestModelEndpointMismatchReplayStartupRun(t *testing.T) {
	record, err := os.ReadFile(filepath.Join(modelEndpointMismatchReplayDir, "model-endpoint-mismatch.json"))
	require.NoError(t, err)
	session, err := os.ReadFile(filepath.Join(modelEndpointMismatchReplayDir, "aw_session.jsonl"))
	require.NoError(t, err)
	runDir := writeModelEndpointMismatchReplayRun(t, session)

	data, _ := buildLocalAuditData(ProcessedRun{Run: WorkflowRun{LogsPath: runDir}}, LogMetrics{}, nil)
	encoded, err := json.Marshal(data)
	require.NoError(t, err)
	var report map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(encoded, &report))
	var mismatches []json.RawMessage
	require.NoError(t, json.Unmarshal(report["model_endpoint_mismatches"], &mismatches))
	require.Len(t, mismatches, 1)
	assert.JSONEq(t, string(record), string(mismatches[0]))

	var expected ModelEndpointMismatch
	require.NoError(t, json.Unmarshal(record, &expected))
	output := captureStderrForGuardPolicyReportTest(func() { renderConsole(data, runDir) })
	for _, text := range []string{
		"model_endpoint_mismatch: phase=startup configured_model=gpt-5.6-luna resolved_model=gpt-5.6-luna wire_api=completions wire_api_source=override supported_endpoints=/responses, ws:/responses",
		"detail: " + expected.Detail,
		"fix: " + expected.Fix,
	} {
		assert.Contains(t, output, text)
	}
}

func TestModelEndpointMismatchReplayNegativeControl(t *testing.T) {
	runDir := writeModelEndpointMismatchReplayRun(t, nil)
	require.NoError(t, os.Remove(filepath.Join(runDir, "agent", "model-endpoint-mismatch.json")))

	data, _ := buildLocalAuditData(ProcessedRun{Run: WorkflowRun{LogsPath: runDir}}, LogMetrics{}, nil)
	assert.Empty(t, data.ModelEndpointMismatches)
	encoded, err := json.Marshal(data)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), `"model_endpoint_mismatches"`)
	output := captureStderrForGuardPolicyReportTest(func() { renderConsole(data, runDir) })
	assert.NotContains(t, output, "model_endpoint_mismatch:")
}
