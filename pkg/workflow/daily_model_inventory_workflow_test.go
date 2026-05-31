//go:build !integration

package workflow

import (
	"os"
	"strings"
	"testing"
)

func TestDailyModelInventoryWorkflowFetchesReflectInAgentPrompt(t *testing.T) {
	lockContent, err := os.ReadFile("../../.github/workflows/daily-model-inventory.lock.yml")
	if err != nil {
		t.Fatalf("failed to read compiled workflow: %v", err)
	}

	lockContentStr := string(lockContent)

	if strings.Contains(lockContentStr, "- name: Fetch Copilot reflect inventory") {
		t.Fatalf("expected compiled workflow to avoid pre-job Copilot reflect fetch step")
	}

	sourceContent, err := os.ReadFile("../../.github/workflows/daily-model-inventory.md")
	if err != nil {
		t.Fatalf("failed to read workflow source: %v", err)
	}
	sourceContentStr := string(sourceContent)

	if !strings.Contains(sourceContentStr, "### Step 0: Fetch Copilot Models from API Proxy") {
		t.Fatalf("expected workflow prompt to include Copilot reflect fetch Step 0")
	}

	if !strings.Contains(sourceContentStr, "curl -fsS http://api-proxy:10000/reflect") {
		t.Fatalf("expected workflow prompt to fetch Copilot reflect data from api-proxy")
	}

	if !strings.Contains(sourceContentStr, "`billing.multiplier` field from the Copilot reflect endpoint as the primary") {
		t.Fatalf("expected workflow prompt to use billing.multiplier from reflect as the primary source of truth")
	}
}
