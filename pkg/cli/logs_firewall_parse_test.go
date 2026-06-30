//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
)

func TestParseFirewallLogsNoLogs(t *testing.T) {
	// Create a temporary directory without any firewall logs
	tempDir := testutil.TempDir(t, "test-*")

	// Run the parser - should not fail, just skip
	err := parseFirewallLogs(tempDir, true)
	if err != nil {
		t.Fatalf("parseFirewallLogs should not fail when no logs present: %v", err)
	}

	// Check that firewall.md was NOT created
	firewallMdPath := filepath.Join(tempDir, "firewall.md")
	if _, err := os.Stat(firewallMdPath); !os.IsNotExist(err) {
		t.Errorf("firewall.md should not be created when no logs are present")
	}
}

func TestFindFirewallLogsDirSandboxFallbackToTopLevel(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-firewall-parse-*")

	sandboxLogsDir := filepath.Join(tempDir, "sandbox", "firewall", "logs")
	if err := os.MkdirAll(filepath.Join(sandboxLogsDir, "squid-logs"), 0755); err != nil {
		t.Fatalf("Failed to create firewall log directories: %v", err)
	}

	logContent := `1761332531.123 172.30.0.20:35289 blocked.example.com:443 140.82.112.23:443 1.1 CONNECT 403 NONE_NONE:HIER_NONE blocked.example.com:443 "-"
`
	if err := os.WriteFile(filepath.Join(sandboxLogsDir, "access.log"), []byte(logContent), 0644); err != nil {
		t.Fatalf("Failed to write access.log: %v", err)
	}

	logsDir, err := findFirewallLogsDir(tempDir)
	if err != nil {
		t.Fatalf("findFirewallLogsDir failed: %v", err)
	}
	if logsDir != sandboxLogsDir {
		t.Fatalf("findFirewallLogsDir returned %q, want %q", logsDir, sandboxLogsDir)
	}
}
