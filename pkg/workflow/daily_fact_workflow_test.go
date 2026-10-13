//go:build !integration

package workflow

import (
	"os"
	"strings"
	"testing"
)

func TestDailyFactWorkflowDoesNotUseMemPalace(t *testing.T) {
	if _, err := os.Stat("../../.github/workflows/shared/mcp/mempalace.md"); !os.IsNotExist(err) {
		t.Fatalf("expected removed shared MemPalace workflow to be absent, got %v", err)
	}

	for _, path := range []string{
		"../../.github/workflows/daily-fact.md",
		"../../.github/workflows/daily-fact.lock.yml",
		"../../.github/workflows/requirements.txt",
	} {
		t.Run(path, func(t *testing.T) {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read Daily Fact workflow: %v", err)
			}
			contentLower := strings.ToLower(string(content))
			for _, removed := range []string{
				"mempalace",
				"cache-memory/palace",
				"8765",
				"palace memory",
			} {
				if strings.Contains(contentLower, removed) {
					t.Errorf("Daily Fact workflow still references removed MemPalace dependency %q", removed)
				}
			}
		})
	}
}
