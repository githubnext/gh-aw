//go:build !integration

package workflow

import (
	"strings"
	"testing"
)

// TestMultilineStringHandling tests that multiline strings in with parameters
// are correctly serialized with proper YAML indentation
func TestMultilineStringHandling(t *testing.T) {
	testCases := []struct {
		name             string
		stepMap          map[string]any
		expectedYAML     string
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name: "multiline script in with parameters",
			stepMap: map[string]any{
				"name": "Test Script",
				"uses": "actions/github-script@v7",
				"with": map[string]any{
					"script": `const fs = require('fs');
const data = {
  key: "value",
  number: 42
};
console.log(data);`,
					"timeout": "30000",
				},
			},
			shouldContain: []string{
				"name: Test Script",
				"uses: actions/github-script@v7",
				"with:",
				"script: |-", // goccy/go-yaml uses |- (literal strip scalar)
				"  const fs = require('fs');",
				"  const data = {",
				"  console.log(data);",
				"timeout: \"30000\"", // goccy/go-yaml quotes numeric strings
			},
			shouldNotContain: []string{
				"script: const fs = require('fs');\\nconst data", // Should not have raw newlines
				"\\n", // Should not have literal \n in output
			},
		},
		{
			name: "simple single-line with parameters",
			stepMap: map[string]any{
				"name": "Simple Test",
				"uses": "actions/setup-node@v4",
				"with": map[string]any{
					"node-version": "18",
					"cache":        "npm",
				},
			},
			shouldContain: []string{
				"name: Simple Test",
				"uses: actions/setup-node@v4",
				"with:",
				"node-version: \"18\"", // goccy/go-yaml quotes numeric strings
				"cache: npm",
			},
		},
		{
			name: "multiline run command",
			stepMap: map[string]any{
				"name": "Multi-line run",
				"run": `echo "Starting build"
npm install
npm run build
echo "Build complete"`,
			},
			shouldContain: []string{
				"name: Multi-line run",
				"run: |-", // goccy/go-yaml uses |- (literal strip scalar)
				"  echo \"Starting build\"",
				"  npm install",
				"  npm run build",
				"  echo \"Build complete\"",
			},
			shouldNotContain: []string{
				"run: echo \"Starting build\"\\n", // Should not have raw newlines
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertStepToYAML(tt.stepMap)
			if err != nil {
				t.Fatalf("convertStepToYAML failed: %v", err)
			}

			t.Logf("Generated YAML:\n%s", result)

			// Check that required strings are present
			for _, required := range tt.shouldContain {
				if !strings.Contains(result, required) {
					t.Errorf("Expected YAML to contain '%s', but it didn't.\nGenerated:\n%s", required, result)
				}
			}

			// Check that forbidden strings are not present
			for _, forbidden := range tt.shouldNotContain {
				if strings.Contains(result, forbidden) {
					t.Errorf("Expected YAML to NOT contain '%s', but it did.\nGenerated:\n%s", forbidden, result)
				}
			}
		})
	}
}
