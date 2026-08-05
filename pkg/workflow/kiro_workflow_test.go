//go:build !integration

package workflow

import (
	"os"
	"strings"
	"testing"
)

func TestKiroWorkflowConfiguresContainerRuntime(t *testing.T) {
	for _, path := range []string{
		"../../.github/workflows/shared/kiro.md",
		"../../.github/workflows/smoke-kiro.lock.yml",
	} {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read Kiro workflow file %s: %v", path, err)
		}
		config := string(content)

		if strings.Contains(config, `MCP_GATEWAY_HOST_DOMAIN || "localhost"`) {
			t.Errorf("expected %s to avoid the host-only MCP gateway domain", path)
		}
		if !strings.Contains(config, `MCP_GATEWAY_DOMAIN || "host.docker.internal"`) {
			t.Errorf("expected %s to use the container MCP gateway domain", path)
		}
		if !strings.Contains(config, "PATH: `${binDir}:${process.env.PATH || \"\"}`") {
			t.Errorf("expected %s to expose Kiro's sibling binaries on PATH", path)
		}
	}
}
