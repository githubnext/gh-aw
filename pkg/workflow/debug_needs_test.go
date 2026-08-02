package workflow

import (
	"fmt"
	"testing"
)

func TestDebugOnNeedsCommentOut(t *testing.T) {
	compiler := NewCompiler()

	yamlStr := `"on":
  needs:
    - custom_job
  workflow_dispatch:`

	result := compiler.commentOutProcessedFieldsInOnSection(yamlStr, map[string]any{})
	fmt.Printf("Result:\n%s\n", result)

	if containsStr(result, "  needs:") {
		t.Errorf("BUG: 'needs:' was NOT commented out. Result:\n%s", result)
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsSubstr(s, sub))
}

func containsSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
