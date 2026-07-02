//go:build !integration

package workflow

// contains checks if a string contains a substring.
// Deprecated: prefer strings.Contains or assert.Contains in new tests.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && s[len(s)-len(substr):] == substr) ||
		(len(s) > len(substr) && s[:len(substr)] == substr) ||
		(len(s) > len(substr) && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// toolCallCountsByName converts a slice of ToolCallInfo into a map of tool name to call count.
// Used across engine-level log parsing tests.
func toolCallCountsByName(toolCalls []ToolCallInfo) map[string]int {
	counts := make(map[string]int, len(toolCalls))
	for _, toolCall := range toolCalls {
		counts[toolCall.Name] = toolCall.CallCount
	}

	return counts
}
