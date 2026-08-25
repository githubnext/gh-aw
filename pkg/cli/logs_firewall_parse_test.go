//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testFirewallLogContent = `1761332531.123 172.30.0.20:35289 blocked.example.com:443 140.82.112.23:443 1.1 CONNECT 403 NONE_NONE:HIER_NONE blocked.example.com:443 "-"
`

// writeFirewallLog creates dir (relative to root) and writes a firewall log file named fileName into it.
func writeFirewallLog(t *testing.T, root string, relDir string, fileName string) string {
	t.Helper()

	dir := filepath.Join(root, filepath.FromSlash(relDir))
	require.NoError(t, os.MkdirAll(dir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, fileName), []byte(testFirewallLogContent), 0644))

	return dir
}

func TestParseFirewallLogsNoLogs(t *testing.T) {
	t.Parallel()

	// Create a temporary directory without any firewall logs
	tempDir := testutil.TempDir(t, "test-*")

	// Run the parser - should not fail, just skip
	require.NoError(t, parseFirewallLogs(tempDir, true), "parseFirewallLogs should not fail when no logs present")

	// Check that firewall.md was NOT created
	firewallMdPath := filepath.Join(tempDir, "firewall.md")
	_, err := os.Stat(firewallMdPath)
	assert.True(t, os.IsNotExist(err), "firewall.md should not be created when no logs are present")
}

func TestParseFirewallLogsNoLogsNonVerbose(t *testing.T) {
	t.Parallel()

	tempDir := testutil.TempDir(t, "test-*")

	require.NoError(t, parseFirewallLogs(tempDir, false), "parseFirewallLogs should not fail when no logs present")

	_, err := os.Stat(filepath.Join(tempDir, "firewall.md"))
	assert.True(t, os.IsNotExist(err), "firewall.md should not be created when no logs are present")
}

func TestFindFirewallLogsDirCandidates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		logDirs []string
		want    string
	}{
		{
			name:    "sandbox squid-logs",
			logDirs: []string{"sandbox/firewall/logs/squid-logs"},
			want:    "sandbox/firewall/logs/squid-logs",
		},
		{
			name:    "sandbox firewall logs",
			logDirs: []string{"sandbox/firewall/logs"},
			want:    "sandbox/firewall/logs",
		},
		{
			name:    "top-level squid-logs",
			logDirs: []string{"squid-logs"},
			want:    "squid-logs",
		},
		{
			name:    "workflow-logs squid-logs",
			logDirs: []string{"workflow-logs/squid-logs"},
			want:    "workflow-logs/squid-logs",
		},
		{
			name:    "sandbox squid-logs wins over other candidates",
			logDirs: []string{"sandbox/firewall/logs/squid-logs", "sandbox/firewall/logs", "squid-logs", "workflow-logs/squid-logs"},
			want:    "sandbox/firewall/logs/squid-logs",
		},
		{
			name:    "sandbox firewall logs wins over top-level squid-logs",
			logDirs: []string{"sandbox/firewall/logs", "squid-logs"},
			want:    "sandbox/firewall/logs",
		},
		{
			name:    "top-level squid-logs wins over workflow-logs",
			logDirs: []string{"squid-logs", "workflow-logs/squid-logs"},
			want:    "squid-logs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tempDir := testutil.TempDir(t, "test-firewall-parse-*")
			for _, dir := range tt.logDirs {
				writeFirewallLog(t, tempDir, dir, "access.log")
			}

			logsDir, err := findFirewallLogsDir(tempDir)
			require.NoError(t, err)
			assert.Equal(t, filepath.Join(tempDir, filepath.FromSlash(tt.want)), logsDir)
		})
	}
}

func TestFindFirewallLogsDirNoMatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		emptyDir string
		file     string
	}{
		{
			name: "no directories at all",
		},
		{
			name:     "candidate directory exists but is empty",
			emptyDir: "sandbox/firewall/logs/squid-logs",
		},
		{
			name:     "candidate directory has no .log files",
			emptyDir: "squid-logs",
			file:     "access.txt",
		},
		{
			name:     "unrelated directory with logs",
			emptyDir: "other-logs",
			file:     "access.log",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tempDir := testutil.TempDir(t, "test-firewall-parse-*")
			if tt.emptyDir != "" {
				if tt.file != "" {
					writeFirewallLog(t, tempDir, tt.emptyDir, tt.file)
				} else {
					require.NoError(t, os.MkdirAll(filepath.Join(tempDir, filepath.FromSlash(tt.emptyDir)), 0755))
				}
			}

			logsDir, err := findFirewallLogsDir(tempDir)
			require.NoError(t, err, "missing or empty firewall log directories are not an error")
			assert.Empty(t, logsDir)
		})
	}
}
