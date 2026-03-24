//go:build !integration

package cli

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSummarizeObservabilityPolicyResult(t *testing.T) {
	result := ObservabilityPolicyResult{
		Violations: []ObservabilityPolicyViolation{
			{Action: "warn"},
			{Action: "gate"},
			{Action: "fail"},
		},
	}

	summary := summarizeObservabilityPolicyResult(result)

	assert.Equal(t, "fail", summary.Status, "fail should take precedence in summary status")
	assert.Equal(t, 3, summary.TotalViolations, "all violations should be counted")
	assert.Equal(t, 1, summary.FailViolations, "fail violations should be counted")
	assert.Equal(t, 1, summary.GateViolations, "gate violations should be counted")
	assert.Equal(t, 1, summary.WarnViolations, "warn violations should be counted")
	assert.True(t, summary.Blocking, "fail or gate should mark summary as blocking")
}

func TestRunObservabilityPolicyEval_JSONOutput(t *testing.T) {
	policyPath := writeJSONFixture(t, "policy.json", ObservabilityPolicy{
		SchemaVersion: "1.0.0",
		Rules: []ObservabilityPolicyRule{
			{
				ID:      "warn-control-plane-failure",
				Action:  "warn",
				Message: "GitHub MCP failed during the run",
				Match: ObservabilityPolicyMatch{
					MCPFailureServers: []string{"github"},
				},
			},
		},
	})

	reportPath := writeJSONFixture(t, "report.json", ObservabilityPayload{
		Tooling: &ObservabilityPayloadTooling{
			MCPFailures: []ObservabilityPolicyMCPFailure{{ServerName: "github"}},
		},
	})

	stdout := captureStream(t, true, func() {
		err := RunObservabilityPolicyEval(ObservabilityPolicyEvalConfig{
			PolicyPath: policyPath,
			ReportPath: reportPath,
			JSONOutput: true,
		})
		require.NoError(t, err, "warn-only result should not return an error")
	})

	var evaluation ObservabilityPolicyEvaluation
	require.NoError(t, json.Unmarshal([]byte(stdout), &evaluation), "command should emit valid JSON")
	assert.Equal(t, "warn", evaluation.Summary.Status, "warn-only result should produce warn status")
	assert.Equal(t, 1, evaluation.Summary.WarnViolations, "warn violations should be counted")
	assert.Len(t, evaluation.Violations, 1, "one policy violation should be emitted")
}

func TestRunObservabilityPolicyEval_GateViolationReturnsError(t *testing.T) {
	policyPath := writeJSONFixture(t, "policy.json", ObservabilityPolicy{
		SchemaVersion: "1.0.0",
		Rules: []ObservabilityPolicyRule{
			{
				ID:      "gate-write-mode",
				Action:  "gate",
				Message: "Write-capable runs require approval",
				Match: ObservabilityPolicyMatch{
					ActuationModes: []string{"write_capable"},
				},
			},
		},
	})

	reportPath := writeJSONFixture(t, "report.json", ObservabilityPayload{
		Actuation: &ObservabilityPayloadActuation{Mode: "write_capable"},
	})

	var err error
	_ = captureStream(t, true, func() {
		err = RunObservabilityPolicyEval(ObservabilityPolicyEvalConfig{
			PolicyPath: policyPath,
			ReportPath: reportPath,
			JSONOutput: true,
		})
	})

	assert.ErrorContains(t, err, "requires approval", "gate violations should return a blocking error")
}

func writeJSONFixture(t *testing.T, name string, value any) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, name)
	content, err := json.Marshal(value)
	require.NoError(t, err, "fixture should marshal")
	require.NoError(t, os.WriteFile(path, content, 0o644), "fixture should be written")

	return path
}

func captureStream(t *testing.T, stdout bool, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	require.NoError(t, err, "pipe should be created")

	if stdout {
		old := os.Stdout
		os.Stdout = w
		defer func() {
			os.Stdout = old
		}()
	} else {
		old := os.Stderr
		os.Stderr = w
		defer func() {
			os.Stderr = old
		}()
	}

	fn()
	require.NoError(t, w.Close(), "writer should close cleanly")

	output, readErr := io.ReadAll(r)
	require.NoError(t, readErr, "captured output should be readable")
	return string(output)
}
