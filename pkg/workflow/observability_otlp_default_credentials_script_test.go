//go:build !integration

package workflow

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCheckOTLPDefaultCredentialsScript verifies the runtime guard emitted for
// workflows whose OTLP endpoint comes from the enterprise default environment:
// an unset endpoint is a no-op, and an endpoint without credentials fails.
func TestCheckOTLPDefaultCredentialsScript(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("requires Linux bash script behavior")
	}

	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller should resolve the current test file")

	scriptPath := filepath.Join(filepath.Dir(file), "..", "..", "actions", "setup", "sh", "check_otlp_default_credentials.sh")

	tests := []struct {
		name    string
		env     []string
		wantErr bool
		want    string
	}{
		{
			name: "no endpoint and no headers is a no-op",
			env:  []string{"OTEL_EXPORTER_OTLP_ENDPOINT=", "OTEL_EXPORTER_OTLP_HEADERS="},
			want: "OTLP telemetry is not configured",
		},
		{
			name: "no endpoint with headers is a no-op",
			env:  []string{"OTEL_EXPORTER_OTLP_ENDPOINT=", "OTEL_EXPORTER_OTLP_HEADERS=Authorization=token"},
			want: "OTLP telemetry is not configured",
		},
		{
			name: "endpoint with headers succeeds",
			env:  []string{"OTEL_EXPORTER_OTLP_ENDPOINT=https://traces.example.com:4317", "OTEL_EXPORTER_OTLP_HEADERS=Authorization=token"},
			want: "OTLP telemetry endpoint and credentials are configured.",
		},
		{
			name:    "endpoint without headers fails",
			env:     []string{"OTEL_EXPORTER_OTLP_ENDPOINT=https://traces.example.com:4317", "OTEL_EXPORTER_OTLP_HEADERS="},
			wantErr: true,
			want:    "::error::",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("bash", scriptPath)
			cmd.Env = append(filteredEnv(
				"OTEL_EXPORTER_OTLP_ENDPOINT=",
				"OTEL_EXPORTER_OTLP_HEADERS=",
			), tt.env...)

			out, err := cmd.CombinedOutput()
			if tt.wantErr {
				require.Error(t, err, "script should fail, output:\n%s", out)
			} else {
				require.NoError(t, err, "script should succeed, output:\n%s", out)
			}
			assert.Contains(t, string(out), tt.want)
		})
	}
}
