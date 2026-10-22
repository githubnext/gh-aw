//go:build !integration

package workflow

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaskOTLPHeadersScript(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("requires Linux bash script behavior")
	}

	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller should resolve the current test file")

	scriptPath := filepath.Join(filepath.Dir(file), "..", "..", "actions", "setup", "sh", "mask_otlp_headers.sh")

	tests := []struct {
		name    string
		env     []string
		want    []string
		wantNot []string
	}{
		{
			name: "multi endpoint headers complete successfully",
			env: []string{
				"OTEL_EXPORTER_OTLP_HEADERS=Authorization=primary-token",
				"GH_AW_OTLP_ALL_HEADERS=Authorization=primary-token,Authorization=secondary-token",
			},
			want: []string{
				"::add-mask::Authorization=primary-token",
				"::add-mask::Authorization=primary-token,Authorization=secondary-token",
				"::add-mask::primary-token",
				"::add-mask::secondary-token",
			},
		},
		{
			name: "bearer token masks raw token",
			env: []string{
				"OTEL_EXPORTER_OTLP_HEADERS=Authorization=Bearer raw-token",
			},
			want: []string{
				"::add-mask::Authorization=Bearer raw-token",
				"::add-mask::Bearer raw-token",
				"::add-mask::raw-token",
			},
		},
		{
			name: "short values are not emitted as masks",
			env: []string{
				"OTEL_EXPORTER_OTLP_HEADERS=Authorization=4,Api-Key=1234,Edge=abcd,Trace=abc,Short=abcdefg,Token=abcdefgh",
			},
			want: []string{
				"::add-mask::Authorization=4,Api-Key=1234,Edge=abcd,Trace=abc,Short=abcdefg,Token=abcdefgh",
				"::add-mask::abcdefgh",
			},
			wantNot: []string{
				"::add-mask::4",
				"::add-mask::1234",
				"::add-mask::abcd",
				"::add-mask::abc",
				"::add-mask::abcdefg",
			},
		},
		{
			name: "boolean placeholder preserves true job outputs",
			env: []string{
				"OTEL_EXPORTER_OTLP_HEADERS=X-No-Auth=true",
			},
			want: []string{"::add-mask::X-No-Auth=true"},
			wantNot: []string{
				"::add-mask::true",
			},
		},
		{
			name: "boolean and numeric placeholders are not individual masks",
			env: []string{
				"OTEL_EXPORTER_OTLP_HEADERS=X-Flag=false,X-Enabled=TRUE,X-Disabled=FALSE,X-Id=1234567890123456",
				"GH_AW_OTLP_ALL_HEADERS=X-No-Auth=true,X-Id=12345678,Api-Key=real-credential",
			},
			want: []string{
				"::add-mask::X-Flag=false,X-Enabled=TRUE,X-Disabled=FALSE,X-Id=1234567890123456",
				"::add-mask::X-No-Auth=true,X-Id=12345678,Api-Key=real-credential",
				"::add-mask::real-credential",
			},
			wantNot: []string{
				"::add-mask::true",
				"::add-mask::false",
				"::add-mask::TRUE",
				"::add-mask::FALSE",
				"::add-mask::12345678",
				"::add-mask::1234567890123456",
			},
		},
		{
			name: "short bearer credential retains raw token mask",
			env: []string{
				"OTEL_EXPORTER_OTLP_HEADERS=Authorization=Bearer " + "abcd",
			},
			want: []string{
				"::add-mask::Authorization=Bearer " + "abcd",
				"::add-mask::Bearer " + "abcd",
				"::add-mask::abcd",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("bash", scriptPath)
			cmd.Env = append(filteredEnv(
				"OTEL_EXPORTER_OTLP_HEADERS=",
				"GH_AW_OTLP_ALL_HEADERS=",
			), tt.env...)

			out, err := cmd.CombinedOutput()
			require.NoError(t, err, "mask script should succeed, output:\n%s", out)

			output := string(out)
			normalizedOutput := "\n" + strings.TrimSpace(output) + "\n"
			for _, want := range tt.want {
				assert.Contains(t, normalizedOutput, "\n"+want+"\n")
			}
			for _, wantNot := range tt.wantNot {
				assert.NotContains(t, normalizedOutput, "\n"+wantNot+"\n")
			}
		})
	}
}

func filteredEnv(excludedPrefixes ...string) []string {
	env := make([]string, 0, len(os.Environ()))
	for _, entry := range os.Environ() {
		excluded := false
		for _, prefix := range excludedPrefixes {
			if strings.HasPrefix(entry, prefix) {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}
		env = append(env, entry)
	}
	return env
}

func TestMaskOTLPAttributesScript(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("requires Linux bash script behavior")
	}

	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller should resolve the current test file")

	scriptPath := filepath.Join(filepath.Dir(file), "..", "..", "actions", "setup", "sh", "mask_otlp_attributes.sh")

	tests := []struct {
		name    string
		env     []string
		want    []string
		wantNot []string
	}{
		{
			name: "masks each attribute value individually",
			env: []string{
				`GH_AW_OTLP_ATTRIBUTES={"langfuse.session.id":"my-session","langfuse.user.id":"my-user"}`,
			},
			want: []string{
				"::add-mask::my-session",
				"::add-mask::my-user",
			},
		},
		{
			name: "short attribute values are not emitted as masks",
			env: []string{
				`GH_AW_OTLP_ATTRIBUTES={"a":"1","b":"176","c":"abc","d":"abcd"}`,
			},
			want: []string{"::add-mask::abcd"},
			wantNot: []string{
				"::add-mask::1",
				"::add-mask::176",
				"::add-mask::abc",
			},
		},
		{
			name: "empty variable is a no-op",
			env:  []string{"GH_AW_OTLP_ATTRIBUTES="},
			want: []string{},
		},
		{
			name: "invalid JSON is a no-op",
			env:  []string{"GH_AW_OTLP_ATTRIBUTES=not-json"},
			want: []string{},
		},
		{
			name: "empty attribute values are skipped",
			env: []string{
				`GH_AW_OTLP_ATTRIBUTES={"my.attr":""}`,
			},
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("bash", scriptPath)
			cmd.Env = append(filteredEnv("GH_AW_OTLP_ATTRIBUTES="), tt.env...)

			out, err := cmd.CombinedOutput()
			require.NoError(t, err, "mask script should succeed, output:\n%s", out)

			output := string(out)
			normalizedOutput := "\n" + strings.TrimSpace(output) + "\n"
			for _, want := range tt.want {
				assert.Contains(t, output, want)
			}
			for _, wantNot := range tt.wantNot {
				assert.NotContains(t, normalizedOutput, "\n"+wantNot+"\n")
			}
			if len(tt.want) == 0 {
				assert.Empty(t, strings.TrimSpace(output), "expected no output for this case")
			}
		})
	}
}
