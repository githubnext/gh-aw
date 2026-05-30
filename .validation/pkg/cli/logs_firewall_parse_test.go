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
