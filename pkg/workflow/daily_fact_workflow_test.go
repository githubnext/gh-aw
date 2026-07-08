//go:build !integration

package workflow

import (
	"os"
	"strings"
	"testing"
)

func TestDailyFactWorkflowUsesHostDockerInternalForMemPalace(t *testing.T) {
	sourceContent, err := os.ReadFile("../../.github/workflows/shared/mcp/mempalace.md")
	if err != nil {
		t.Fatalf("failed to read shared mempalace import: %v", err)
	}
	source := string(sourceContent)

	if strings.Contains(source, "http://localhost:8765/mcp") {
		t.Fatalf("expected shared mempalace import to avoid localhost MCP URLs under network isolation")
	}
	if !strings.Contains(source, "http://host.docker.internal:8765/mcp") {
		t.Fatalf("expected shared mempalace import to use host.docker.internal for the MCP URL")
	}

	lockContent, err := os.ReadFile("../../.github/workflows/daily-fact.lock.yml")
	if err != nil {
		t.Fatalf("failed to read compiled workflow: %v", err)
	}
	lock := string(lockContent)

	if strings.Contains(lock, "http://localhost:8765/mcp") {
		t.Fatalf("expected compiled Daily Fact workflow to avoid localhost MCP URLs")
	}
	if !strings.Contains(lock, "http://host.docker.internal:8765/mcp") {
		t.Fatalf("expected compiled Daily Fact workflow to use host.docker.internal for MemPalace")
	}
}
