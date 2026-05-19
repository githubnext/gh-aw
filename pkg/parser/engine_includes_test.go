//go:build !integration

package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandIncludesForEngines(t *testing.T) {
	tests := []struct {
		name             string
		includeFiles     map[string]string
		mainContent      string
		expectedLen      int
		expectedExact    map[int]string
		expectedContains map[int][]string
	}{
		{
			name: "single string engine",
			includeFiles: map[string]string{
				"include-engine.md": `---
engine: codex
tools:
  github:
    allowed: ["list_issues"]
---

# Include with Engine
`,
			},
			mainContent: `# Main Workflow

@include include-engine.md

Some content here.
`,
			expectedLen:   1,
			expectedExact: map[int]string{0: `"codex"`},
		},
		{
			name: "object format engine",
			includeFiles: map[string]string{
				"include-object-engine.md": `---
engine:
  id: claude
  model: claude-3-5-sonnet-20241022
  max-turns: 5
tools:
  github:
    allowed: ["list_issues"]
---

# Include with Object Engine
`,
			},
			mainContent: `# Main Workflow

@include include-object-engine.md

Some content here.
`,
			expectedLen: 1,
			expectedContains: map[int][]string{
				0: {`"id":"claude"`, `"model":"claude-3-5-sonnet-20241022"`, `"max-turns":5`},
			},
		},
		{
			name: "include without engine",
			includeFiles: map[string]string{
				"include-no-engine.md": `---
tools:
  github:
    allowed: ["list_issues"]
---

# Include without Engine
`,
			},
			mainContent: `# Main Workflow

@include include-no-engine.md

Some content here.
`,
			expectedLen: 0,
		},
		{
			name: "multiple includes",
			includeFiles: map[string]string{
				"include1.md": `---
engine: claude
tools:
  github:
    allowed: ["list_issues"]
---

# First Include
`,
				"include2.md": `---
engine: codex
tools:
  claude:
    allowed: ["Read", "Write"]
---

# Second Include
`,
			},
			mainContent: `# Main Workflow

@include include1.md

Some content here.

@include include2.md

More content.
`,
			expectedLen:   2,
			expectedExact: map[int]string{0: `"claude"`, 1: `"codex"`},
		},
		{
			name: "optional include missing file",
			mainContent: `# Main Workflow

@include? missing-file.md

Some content here.
`,
			expectedLen: 0,
		},
		{
			name: "object engine with command",
			includeFiles: map[string]string{
				"include-command.md": `---
engine:
  id: copilot
  command: /custom/path/to/copilot
  version: "1.0.0"
tools:
  github:
    allowed: ["list_issues"]
---

# Include with Custom Command
`,
			},
			mainContent: `# Main Workflow

@include include-command.md

Some content here.
`,
			expectedLen: 1,
			expectedContains: map[int][]string{
				0: {`"id":"copilot"`, `"command":"/custom/path/to/copilot"`, `"version":"1.0.0"`},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "test-*")
			for fileName, fileContent := range tt.includeFiles {
				includeFile := filepath.Join(tmpDir, fileName)
				err := os.WriteFile(includeFile, []byte(fileContent), 0644)
				require.NoError(t, err, "Should write include file %s", fileName)
			}

			engines, err := ExpandIncludesForEngines(tt.mainContent, tmpDir)
			require.NoError(t, err, "Should expand includes for test case %q", tt.name)

			require.Len(t, engines, tt.expectedLen, "Should return expected number of engines for test case %q", tt.name)

			for idx, expected := range tt.expectedExact {
				assert.Equal(t, expected, engines[idx], "Engine %d should match expected value for test case %q", idx, tt.name)
			}

			for idx, expectedFields := range tt.expectedContains {
				for _, expectedField := range expectedFields {
					assert.Containsf(t, engines[idx], expectedField, "Engine %d should include field %q for test case %q", idx, expectedField, tt.name)
				}
			}
		})
	}
}

func TestExtractEngineFromContent(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name: "string engine",
			content: `---
engine: claude
---
# Test
`,
			expected: `"claude"`,
		},
		{
			name: "object engine",
			content: `---
engine:
  id: codex
  model: gpt-4
---
# Test
`,
			expected: `{"id":"codex","model":"gpt-4"}`,
		},
		{
			name: "no engine",
			content: `---
tools:
  github: {}
---
# Test
`,
			expected: "",
		},
		{
			name: "no frontmatter",
			content: `# Test

Just markdown content.
`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractFrontmatterField(tt.content, "engine", "")
			require.NoError(t, err, "Should extract engine field without error for case %q", tt.name)
			assert.Equal(t, tt.expected, result, "Extracted engine should match expected value for case %q", tt.name)
		})
	}
}
