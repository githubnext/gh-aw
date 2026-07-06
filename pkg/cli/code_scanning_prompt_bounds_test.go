//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListCodeScanningAlertsPromptBounds(t *testing.T) {
	paths := []string{
		filepath.Join("..", "..", ".github", "workflows", "github-mcp-structural-analysis.md"),
		filepath.Join("..", "..", ".github", "aw", "github-mcp-server.md"),
		filepath.Join("..", "..", ".github", "skills", "github-mcp-server", "SKILL.md"),
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read %s: %v", path, err)
			}
			text := string(content)
			if !strings.Contains(text, "list_code_scanning_alerts") {
				t.Fatalf("%s must reference list_code_scanning_alerts", path)
			}
			if !strings.Contains(text, "state: open") {
				t.Fatalf("%s must include state: open guard", path)
			}
			if !strings.Contains(text, "severity: critical,high") {
				t.Fatalf("%s must include severity: critical,high guard", path)
			}
		})
	}
}
