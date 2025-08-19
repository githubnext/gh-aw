package parser

import (
	"errors"
	"testing"

	"github.com/goccy/go-yaml"
)

func TestExtractYAMLError(t *testing.T) {
	tests := []struct {
		name                 string
		err                  error
		frontmatterStartLine int
		expectedLine         int
		expectedColumn       int
		expectedMessage      string
	}{
		{
			name:                 "yaml line error",
			err:                  errors.New("yaml: line 7: mapping values are not allowed in this context"),
			frontmatterStartLine: 1,
			expectedLine:         8, // 7 + 1
			expectedColumn:       0, // No column info provided in string format
			expectedMessage:      "mapping values are not allowed in this context",
		},
		{
			name:                 "yaml line error with frontmatter offset",
			err:                  errors.New("yaml: line 3: found character that cannot start any token"),
			frontmatterStartLine: 5,
			expectedLine:         8, // 3 + 5
			expectedColumn:       0, // No column info provided in string format
			expectedMessage:      "found character that cannot start any token",
		},
		{
			name:                 "non-yaml error",
			err:                  errors.New("some other error"),
			frontmatterStartLine: 1,
			expectedLine:         0,
			expectedColumn:       0,
			expectedMessage:      "some other error",
		},
		{
			name:                 "yaml error with different message format",
			err:                  errors.New("yaml: line 15: found unexpected end of stream"),
			frontmatterStartLine: 2,
			expectedLine:         17, // 15 + 2
			expectedColumn:       0,  // No column info provided in string format
			expectedMessage:      "found unexpected end of stream",
		},
		{
			name:                 "yaml error with indentation issue",
			err:                  errors.New("yaml: line 4: bad indentation of a mapping entry"),
			frontmatterStartLine: 1,
			expectedLine:         5, // 4 + 1
			expectedColumn:       0, // No column info provided in string format
			expectedMessage:      "bad indentation of a mapping entry",
		},
		{
			name:                 "yaml error with duplicate key",
			err:                  errors.New("yaml: line 6: found duplicate key"),
			frontmatterStartLine: 3,
			expectedLine:         9, // 6 + 3
			expectedColumn:       0, // No column info provided in string format
			expectedMessage:      "found duplicate key",
		},
		{
			name:                 "yaml error with complex format",
			err:                  errors.New("yaml: line 12: did not find expected ',' or ']'"),
			frontmatterStartLine: 0,
			expectedLine:         12, // 12 + 0
			expectedColumn:       0,  // No column info provided in string format
			expectedMessage:      "did not find expected ',' or ']'",
		},
		{
			name:                 "yaml unmarshal error multiline",
			err:                  errors.New("yaml: unmarshal errors:\n  line 4: mapping key \"permissions\" already defined at line 2"),
			frontmatterStartLine: 1,
			expectedLine:         5, // 4 + 1
			expectedColumn:       0, // No column info provided in string format
			expectedMessage:      "mapping key \"permissions\" already defined at line 2",
		},
		{
			name:                 "yaml error with flow mapping",
			err:                  errors.New("yaml: line 8: did not find expected ',' or '}'"),
			frontmatterStartLine: 1,
			expectedLine:         9, // 8 + 1
			expectedColumn:       0, // No column info provided in string format
			expectedMessage:      "did not find expected ',' or '}'",
		},
		{
			name:                 "yaml error with invalid character",
			err:                  errors.New("yaml: line 5: found character that cannot start any token"),
			frontmatterStartLine: 0,
			expectedLine:         5, // 5 + 0
			expectedColumn:       0, // No column info provided in string format
			expectedMessage:      "found character that cannot start any token",
		},
		{
			name:                 "yaml error with unmarshal type issue",
			err:                  errors.New("yaml: line 3: cannot unmarshal !!str `yes_please` into bool"),
			frontmatterStartLine: 2,
			expectedLine:         5, // 3 + 2
			expectedColumn:       0, // No column info provided in string format
			expectedMessage:      "cannot unmarshal !!str `yes_please` into bool",
		},
		{
			name:                 "yaml complex unmarshal error with nested line info",
			err:                  errors.New("yaml: unmarshal errors:\n  line 7: found unexpected end of stream\n  line 9: mapping values are not allowed in this context"),
			frontmatterStartLine: 1,
			expectedLine:         8, // First line 7 + 1
			expectedColumn:       0, // No column info provided in string format
			expectedMessage:      "found unexpected end of stream",
		},
		{
			name:                 "yaml error with column information greater than 1",
			err:                  errors.New("yaml: line 5: column 12: invalid character at position"),
			frontmatterStartLine: 1,
			expectedLine:         6, // 5 + 1
			expectedColumn:       12,
			expectedMessage:      "invalid character at position",
		},
		{
			name:                 "yaml error with high column number",
			err:                  errors.New("yaml: line 3: column 45: unexpected token found"),
			frontmatterStartLine: 2,
			expectedLine:         5, // 3 + 2
			expectedColumn:       45,
			expectedMessage:      "unexpected token found",
		},
		{
			name:                 "yaml error with column 1 explicitly specified",
			err:                  errors.New("yaml: line 8: column 1: mapping values not allowed in this context"),
			frontmatterStartLine: 0,
			expectedLine:         8, // 8 + 0
			expectedColumn:       1,
			expectedMessage:      "mapping values not allowed in this context",
		},
		{
			name:                 "yaml error with medium column position",
			err:                  errors.New("yaml: line 2: column 23: found character that cannot start any token"),
			frontmatterStartLine: 3,
			expectedLine:         5, // 2 + 3
			expectedColumn:       23,
			expectedMessage:      "found character that cannot start any token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line, column, message := ExtractYAMLError(tt.err, tt.frontmatterStartLine)

			if line != tt.expectedLine {
				t.Errorf("Expected line %d, got %d", tt.expectedLine, line)
			}
			if column != tt.expectedColumn {
				t.Errorf("Expected column %d, got %d", tt.expectedColumn, column)
			}
			if message != tt.expectedMessage {
				t.Errorf("Expected message '%s', got '%s'", tt.expectedMessage, message)
			}
		})
	}
}

// TestExtractYAMLErrorWithGoccyErrors tests extraction from actual goccy/go-yaml errors
func TestExtractYAMLErrorWithGoccyErrors(t *testing.T) {
	tests := []struct {
		name                 string
		yamlContent          string
		frontmatterStartLine int
		expectedMinLine      int // Use min line since exact line may vary
		expectedMinColumn    int // Use min column since exact column may vary
		expectValidLocation  bool
	}{
		{
			name:                 "goccy invalid syntax",
			yamlContent:          "invalid: yaml: content",
			frontmatterStartLine: 1,
			expectedMinLine:      2, // Should be > frontmatterStartLine
			expectedMinColumn:    5, // Should have a valid column
			expectValidLocation:  true,
		},
		{
			name:                 "goccy indentation error",
			yamlContent:          "name: test\n  invalid_indentation: here",
			frontmatterStartLine: 2,
			expectedMinLine:      3, // Should be > frontmatterStartLine
			expectedMinColumn:    1, // Should have a valid column
			expectValidLocation:  true,
		},
		{
			name:                 "goccy duplicate key",
			yamlContent:          "name: test\nname: duplicate",
			frontmatterStartLine: 0,
			expectedMinLine:      1, // Should be > frontmatterStartLine
			expectedMinColumn:    1, // Should have a valid column
			expectValidLocation:  true,
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

			line, column, message := ExtractYAMLError(err, tt.frontmatterStartLine)

			if tt.expectValidLocation {
				if line < tt.expectedMinLine {
					t.Errorf("Expected line >= %d, got %d", tt.expectedMinLine, line)
				}
				if column < tt.expectedMinColumn {
					t.Errorf("Expected column >= %d, got %d", tt.expectedMinColumn, column)
				}
				if message == "" {
					t.Errorf("Expected non-empty message")
				}
			} else {
				if line != 0 || column != 0 {
					t.Errorf("Expected no location (0,0) when location unknown, got (%d,%d)", line, column)
				}
			}

			t.Logf("YAML: %q -> Line: %d, Column: %d, Message: %s", tt.yamlContent, line, column, message)
		})
	}
}

// TestExtractYAMLErrorUnknownLocation tests that 0,0 is returned when location is unknown
func TestExtractYAMLErrorUnknownLocation(t *testing.T) {
	tests := []struct {
		name                 string
		err                  error
		frontmatterStartLine int
		expectedLine         int
		expectedColumn       int
		expectedMessage      string
	}{
		{
			name:                 "non-yaml error without location",
			err:                  errors.New("generic error without location info"),
			frontmatterStartLine: 1,
			expectedLine:         0,
			expectedColumn:       0,
			expectedMessage:      "generic error without location info",
		},
		{
			name:                 "malformed yaml error string",
			err:                  errors.New("yaml: some error without line info"),
			frontmatterStartLine: 1,
			expectedLine:         0,
			expectedColumn:       0,
			expectedMessage:      "yaml: some error without line info",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line, column, message := ExtractYAMLError(tt.err, tt.frontmatterStartLine)

			if line != tt.expectedLine {
				t.Errorf("Expected line %d, got %d", tt.expectedLine, line)
			}
			if column != tt.expectedColumn {
				t.Errorf("Expected column %d, got %d", tt.expectedColumn, column)
			}
			if message != tt.expectedMessage {
				t.Errorf("Expected message '%s', got '%s'", tt.expectedMessage, message)
			}
		})
	}
}
