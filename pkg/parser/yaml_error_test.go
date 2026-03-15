//go:build !integration

package parser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
)

// TestFormatYAMLError tests the new FormatYAMLError function that uses yaml.FormatError()
func TestFormatYAMLError(t *testing.T) {
	tests := []struct {
		name                  string
		yamlContent           string
		frontmatterLineOffset int
		expectedLineCol       string // Expected [line:col] format in output
		expectSourceContext   bool   // Should contain source code lines with | markers
		expectVisualPointer   bool   // Should contain visual ^ pointer
	}{
		{
			name:                  "invalid mapping with offset 1",
			yamlContent:           "invalid: yaml: syntax",
			frontmatterLineOffset: 1,
			expectedLineCol:       "[1:10]",
			expectSourceContext:   true,
			expectVisualPointer:   true,
		},
		{
			name:                  "invalid mapping with offset 5",
			yamlContent:           "invalid: yaml: syntax",
			frontmatterLineOffset: 5,
			expectedLineCol:       "[5:10]",
			expectSourceContext:   true,
			expectVisualPointer:   true,
		},
		{
			name:                  "indentation error",
			yamlContent:           "name: test\n  invalid_indentation: here",
			frontmatterLineOffset: 3,
			expectedLineCol:       "[3:",
			expectSourceContext:   true,
			expectVisualPointer:   true,
		},
		{
			name:                  "duplicate key",
			yamlContent:           "name: test\nname: duplicate",
			frontmatterLineOffset: 2,
			expectedLineCol:       "[3:1]",
			expectSourceContext:   true,
			expectVisualPointer:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate an actual goccy/go-yaml error
			var result map[string]any
			err := yaml.Unmarshal([]byte(tt.yamlContent), &result)

			if err == nil {
				t.Errorf("Expected YAML parsing to fail for content: %q", tt.yamlContent)
				return
			}

			// Format the error with the new function
			formatted := FormatYAMLError(err, tt.frontmatterLineOffset, tt.yamlContent)

			// Check for expected [line:col] format
			if !strings.Contains(formatted, tt.expectedLineCol) {
				t.Errorf("Expected output to contain '%s', got:\n%s", tt.expectedLineCol, formatted)
			}

			// Check for source context (lines with | markers)
			if tt.expectSourceContext && !strings.Contains(formatted, "|") {
				t.Errorf("Expected output to contain source context with '|' markers, got:\n%s", formatted)
			}

			// Check for visual pointer
			if tt.expectVisualPointer && !strings.Contains(formatted, "^") {
				t.Errorf("Expected output to contain visual pointer '^', got:\n%s", formatted)
			}

			// Verify "already defined at" references also have adjusted line numbers
			if strings.Contains(formatted, "already defined at") {
				if tt.frontmatterLineOffset > 1 && strings.Contains(formatted, "already defined at [1:") {
					t.Errorf("Expected 'already defined at' line numbers to be adjusted, got:\n%s", formatted)
				}
			}

			t.Logf("Formatted error:\n%s", formatted)
		})
	}
}

// TestFormatYAMLErrorAdjustment specifically tests line number adjustment
func TestFormatYAMLErrorAdjustment(t *testing.T) {
	yamlContent := "name: test\nname: duplicate"

	tests := []struct {
		offset             int
		expectedFirstLine  string
		expectedSecondLine string
		expectedDefinedAt  string
	}{
		{
			offset:             1,
			expectedFirstLine:  "   1 |",
			expectedSecondLine: ">  2 |",
			expectedDefinedAt:  "already defined at [1:1]",
		},
		{
			offset:             5,
			expectedFirstLine:  "   5 |",
			expectedSecondLine: ">  6 |",
			expectedDefinedAt:  "already defined at [5:1]",
		},
		{
			offset:             10,
			expectedFirstLine:  "  10 |",
			expectedSecondLine: "> 11 |",
			expectedDefinedAt:  "already defined at [10:1]",
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("offset_%d", tt.offset), func(t *testing.T) {
			var result map[string]any
			err := yaml.Unmarshal([]byte(yamlContent), &result)

			if err == nil {
				t.Errorf("Expected YAML parsing to fail")
				return
			}

			formatted := FormatYAMLError(err, tt.offset, yamlContent)

			// Check first line number
			if !strings.Contains(formatted, tt.expectedFirstLine) {
				t.Errorf("Expected first line number format '%s', got:\n%s", tt.expectedFirstLine, formatted)
			}

			// Check second line number
			if !strings.Contains(formatted, tt.expectedSecondLine) {
				t.Errorf("Expected second line number format '%s', got:\n%s", tt.expectedSecondLine, formatted)
			}

			// Check "already defined at" reference
			if !strings.Contains(formatted, tt.expectedDefinedAt) {
				t.Errorf("Expected 'already defined at' reference '%s', got:\n%s", tt.expectedDefinedAt, formatted)
			}

			t.Logf("Formatted error (offset %d):\n%s", tt.offset, formatted)
		})
	}
}
