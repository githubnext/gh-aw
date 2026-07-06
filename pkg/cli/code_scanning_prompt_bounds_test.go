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
				t.Skipf("%s does not reference list_code_scanning_alerts", path)
			}

			lines := strings.Split(text, "\n")
			hasBoundedReference := false
			for i, line := range lines {
				if !strings.Contains(line, "list_code_scanning_alerts") {
					continue
				}
				start := max(0, i-8)
				end := min(len(lines), i+9)
				window := strings.Join(lines[start:end], "\n")
				if strings.Contains(window, "state: open") && strings.Contains(window, "severity: critical,high") {
					hasBoundedReference = true
					break
				}
			}
			if !hasBoundedReference {
				t.Fatalf("%s must include state: open and severity: critical,high near list_code_scanning_alerts", path)
			}
		})
	}
}
