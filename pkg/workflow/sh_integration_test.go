package workflow

import (
	"strings"
	"testing"
)

// TestWritePromptTextToYAML_IntegrationWithCompiler verifies that WritePromptTextToYAML
// correctly handles large prompt text that would be used in actual workflow compilation.
// This test simulates what would happen if an embedded prompt file was very large.
func TestWritePromptTextToYAML_IntegrationWithCompiler(t *testing.T) {
	// Create a realistic scenario: a very long help text or documentation
	// that might be included as prompt instructions
	section := strings.Repeat("This is an important instruction line that provides guidance to the AI agent on how to perform its task correctly. ", 10)

	// Create 200 lines to ensure we exceed 20KB
	lines := make([]string, 200)
	for i := range lines {
		lines[i] = section
	}
	largePromptText := strings.Join(lines, "\n")

	// Calculate total size
	totalSize := len(largePromptText)
	if totalSize < 20000 {
		t.Fatalf("Test setup error: prompt text should be at least 20000 bytes, got %d", totalSize)
	}

	var yaml strings.Builder
	indent := "          " // Standard indent used in workflow generation

	// Call the function as it would be called in real compilation
	WritePromptTextToYAML(&yaml, largePromptText, indent)

	result := yaml.String()

	// Verify multiple heredoc blocks were created
	heredocCount := strings.Count(result, "cat >> $GITHUB_AW_PROMPT << 'EOF'")
	if heredocCount < 2 {
		t.Errorf("Expected multiple heredoc blocks for large text (%d bytes), got %d", totalSize, heredocCount)
	}

	// Verify we didn't exceed 5 chunks
	if heredocCount > 5 {
		t.Errorf("Expected at most 5 heredoc blocks (max limit), got %d", heredocCount)
	}

	// Verify each heredoc is closed
	eofCount := strings.Count(result, indent+"EOF")
	if eofCount != heredocCount {
		t.Errorf("Expected %d EOF markers to match %d heredoc blocks, got %d", heredocCount, heredocCount, eofCount)
	}

	// Verify the content is preserved (check first and last sections)
	firstSection := section[:100]
	lastSection := section[len(section)-100:]
	if !strings.Contains(result, firstSection) {
		t.Error("Expected to find beginning of original text in output")
	}
	if !strings.Contains(result, lastSection) {
		t.Error("Expected to find end of original text in output")
	}

	// Verify the YAML structure is valid (basic check)
	if !strings.Contains(result, "cat >> $GITHUB_AW_PROMPT << 'EOF'") {
		t.Error("Expected proper heredoc syntax in output")
	}

	t.Logf("Successfully chunked %d bytes into %d heredoc blocks", totalSize, heredocCount)
}

// TestWritePromptTextToYAML_RealWorldSizeSimulation simulates various real-world scenarios
// to ensure chunking works correctly across different text sizes.
func TestWritePromptTextToYAML_RealWorldSizeSimulation(t *testing.T) {
	tests := []struct {
		name           string
		textSize       int // approximate size in bytes
		linesCount     int // number of lines
		expectedChunks int // expected number of chunks
		maxChunks      int // should not exceed this
	}{
		{
			name:           "small prompt (< 1KB)",
			textSize:       500,
			linesCount:     10,
			expectedChunks: 1,
			maxChunks:      1,
		},
		{
			name:           "medium prompt (~10KB)",
			textSize:       10000,
			linesCount:     100,
			expectedChunks: 1,
			maxChunks:      1,
		},
		{
			name:           "large prompt (~25KB)",
			textSize:       25000,
			linesCount:     250,
			expectedChunks: 2,
			maxChunks:      2,
		},
		{
			name:           "very large prompt (~50KB)",
			textSize:       50000,
			linesCount:     500,
			expectedChunks: 3,
			maxChunks:      3,
		},
		{
			name:           "extremely large prompt (~120KB)",
			textSize:       120000,
			linesCount:     1200,
			expectedChunks: 5,
			maxChunks:      5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create text of approximately the desired size
			lineSize := tt.textSize / tt.linesCount
			line := strings.Repeat("x", lineSize)
			lines := make([]string, tt.linesCount)
			for i := range lines {
				lines[i] = line
			}
			text := strings.Join(lines, "\n")

			var yaml strings.Builder
			indent := "          "

			WritePromptTextToYAML(&yaml, text, indent)

			result := yaml.String()
			heredocCount := strings.Count(result, "cat >> $GITHUB_AW_PROMPT << 'EOF'")

			if heredocCount < tt.expectedChunks {
				t.Errorf("Expected at least %d chunks for %s, got %d", tt.expectedChunks, tt.name, heredocCount)
			}

			if heredocCount > tt.maxChunks {
				t.Errorf("Expected at most %d chunks for %s, got %d", tt.maxChunks, tt.name, heredocCount)
			}

			eofCount := strings.Count(result, indent+"EOF")
			if eofCount != heredocCount {
				t.Errorf("EOF count (%d) doesn't match heredoc count (%d) for %s", eofCount, heredocCount, tt.name)
			}

			t.Logf("%s: %d bytes chunked into %d blocks", tt.name, len(text), heredocCount)
		})
	}
}
