package workflowcontract

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAvengerSandboxMountsOmitNpmBinary guards against reintroducing the startup crash
// fixed by removing the "/usr/local/bin/npm:/usr/local/bin/npm:ro" bind mount from
// avenger.md's sandbox configuration: the sandbox previously rejected npm as a symlink
// bind mount, crashing before Codex execution. It also asserts that the node runtime and
// its global node_modules directory remain mounted, since the shared nodePathSetupCommand
// (pkg/workflow/copilot_engine_execution.go) relies on that mount to resolve NODE_PATH
// when the npm binary itself is unavailable.
func TestAvengerSandboxMountsOmitNpmBinary(t *testing.T) {
	t.Parallel()
	content, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "avenger.md"))
	if err != nil {
		t.Fatalf("failed to read avenger workflow source: %v", err)
	}
	text := string(content)

	if strings.Contains(text, "/usr/local/bin/npm:/usr/local/bin/npm") {
		t.Error("avenger.md sandbox mounts must not bind-mount /usr/local/bin/npm; the sandbox rejects it as a symlink mount and fails startup")
	}

	for _, mount := range []string{
		`"/usr/local/bin/node:/usr/local/bin/node:ro"`,
		`"/usr/local/lib/node_modules:/usr/local/lib/node_modules:ro"`,
	} {
		if !strings.Contains(text, mount) {
			t.Errorf("avenger.md sandbox mounts must retain %s so NODE_PATH can still resolve global node_modules without npm mounted", mount)
		}
	}
}
