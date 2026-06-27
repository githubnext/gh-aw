//go:build !integration

package workflow

import (
	"os"
	"strings"
	"testing"
)

func TestDailyByokOllamaWorkflowHasReadinessAndAllowlist(t *testing.T) {
	sourceContent, err := os.ReadFile("../../.github/workflows/daily-byok-ollama-test.md")
	if err != nil {
		t.Fatalf("failed to read workflow source: %v", err)
	}
	source := string(sourceContent)

	if !strings.Contains(source, "- name: Verify Ollama BYOK readiness") {
		t.Fatalf("expected source workflow to include an explicit Ollama readiness step")
	}
	if !strings.Contains(source, "ollama list | grep -Fq \"$OLLAMA_MODEL\"") {
		t.Fatalf("expected source workflow to verify the required Ollama model is available")
	}
	if !strings.Contains(source, "curl -sf http://localhost:11434/v1/models") {
		t.Fatalf("expected source workflow to probe /v1/models before the agent runs")
	}

	lockContent, err := os.ReadFile("../../.github/workflows/daily-byok-ollama-test.lock.yml")
	if err != nil {
		t.Fatalf("failed to read compiled workflow: %v", err)
	}
	lock := string(lockContent)

	if !strings.Contains(lock, "host.docker.internal:11434") {
		t.Fatalf("expected compiled workflow firewall allow-list to include host.docker.internal:11434")
	}
}
