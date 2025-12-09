package workflow

import (
	"strings"
	"testing"
)

// TestSafeInputsStepCodeGenerationStability verifies that the MCP setup step code generation
// for safe-inputs produces stable, deterministic output when called multiple times.
// This test ensures that tools are sorted before generating cat commands.
func TestSafeInputsStepCodeGenerationStability(t *testing.T) {
	// Create a config with multiple tools to ensure sorting is tested
	safeInputsConfig := &SafeInputsConfig{
		Tools: map[string]*SafeInputToolConfig{
			"zebra-shell": {
				Name:        "zebra-shell",
				Description: "A shell tool that starts with Z",
				Run:         "echo zebra",
			},
			"alpha-js": {
				Name:        "alpha-js",
				Description: "A JS tool that starts with A",
				Script:      "return 'alpha';",
			},
			"middle-shell": {
				Name:        "middle-shell",
				Description: "A shell tool in the middle",
				Run:         "echo middle",
			},
			"beta-js": {
				Name:        "beta-js",
				Description: "A JS tool that starts with B",
				Script:      "return 'beta';",
			},
		},
	}

	workflowData := &WorkflowData{
		SafeInputs: safeInputsConfig,
		Tools:      make(map[string]any),
		Features: map[string]bool{
			"safe-inputs": true, // Feature flag is optional now
		},
	}

	// Generate MCP setup code multiple times using the actual compiler method
	iterations := 10
	outputs := make([]string, iterations)
	compiler := &Compiler{}

	// Create a mock engine that does nothing for MCP config
	mockEngine := &CustomEngine{}

	for i := 0; i < iterations; i++ {
		var yaml strings.Builder
		compiler.generateMCPSetup(&yaml, workflowData.Tools, mockEngine, workflowData)
		outputs[i] = yaml.String()
	}

	// All iterations should produce identical output
	for i := 1; i < iterations; i++ {
		if outputs[i] != outputs[0] {
			t.Errorf("generateMCPSetup produced different output on iteration %d", i+1)
			// Find first difference for debugging
			for j := 0; j < len(outputs[0]) && j < len(outputs[i]); j++ {
				if outputs[0][j] != outputs[i][j] {
					start := j - 100
					if start < 0 {
						start = 0
					}
					end := j + 100
					if end > len(outputs[0]) {
						end = len(outputs[0])
					}
					if end > len(outputs[i]) {
						end = len(outputs[i])
					}
					t.Errorf("First difference at position %d:\n  Expected: %q\n  Got: %q", j, outputs[0][start:end], outputs[i][start:end])
					break
				}
			}
		}
	}

	// Verify tools appear in sorted order in the output
	// All tools are sorted alphabetically regardless of type (JavaScript or shell):
	// alpha-js, beta-js, middle-shell, zebra-shell
	alphaPos := strings.Index(outputs[0], "alpha-js")
	betaPos := strings.Index(outputs[0], "beta-js")
	middlePos := strings.Index(outputs[0], "middle-shell")
	zebraPos := strings.Index(outputs[0], "zebra-shell")

	if alphaPos == -1 || betaPos == -1 || middlePos == -1 || zebraPos == -1 {
		t.Error("Output should contain all tool names")
	}

	// Verify alphabetical sorting: alpha < beta < middle < zebra
	if !(alphaPos < betaPos && betaPos < middlePos && middlePos < zebraPos) {
		t.Errorf("Tools should be sorted alphabetically in step code: alpha(%d) < beta(%d) < middle(%d) < zebra(%d)",
			alphaPos, betaPos, middlePos, zebraPos)
	}
}

// TestJavaScriptFileChunking validates that large JavaScript files are split into multiple steps
func TestJavaScriptFileChunking(t *testing.T) {
	// Create a medium-sized content that won't exceed limit alone but will when combined
	mediumContent := strings.Repeat("console.log('This is a line of JavaScript code that will be repeated many times');\n", 150)

	tests := []struct {
		name          string
		files         []JavaScriptFileWrite
		expectedSteps int // Exact number of steps expected
	}{
		{
			name: "single small file",
			files: []JavaScriptFileWrite{
				{
					Filename:   "small.cjs",
					Content:    "console.log('small');",
					EOFMarker:  "EOF_SMALL",
					TargetPath: "/tmp/test/small.cjs",
				},
			},
			expectedSteps: 1,
		},
		{
			name: "multiple files that fit in one step",
			files: []JavaScriptFileWrite{
				{
					Filename:   "file1.cjs",
					Content:    "console.log('file1');",
					EOFMarker:  "EOF_1",
					TargetPath: "/tmp/test/file1.cjs",
				},
				{
					Filename:   "file2.cjs",
					Content:    "console.log('file2');",
					EOFMarker:  "EOF_2",
					TargetPath: "/tmp/test/file2.cjs",
				},
			},
			expectedSteps: 1,
		},
		{
			name: "multiple medium files requiring split into 2 steps",
			files: []JavaScriptFileWrite{
				{
					Filename:   "medium1.cjs",
					Content:    mediumContent,
					EOFMarker:  "EOF_MEDIUM1",
					TargetPath: "/tmp/test/medium1.cjs",
				},
				{
					Filename:   "medium2.cjs",
					Content:    mediumContent,
					EOFMarker:  "EOF_MEDIUM2",
					TargetPath: "/tmp/test/medium2.cjs",
				},
			},
			expectedSteps: 2,
		},
		{
			name: "multiple medium files requiring split into 3 steps",
			files: []JavaScriptFileWrite{
				{
					Filename:   "m1.cjs",
					Content:    mediumContent,
					EOFMarker:  "EOF_M1",
					TargetPath: "/tmp/test/m1.cjs",
				},
				{
					Filename:   "m2.cjs",
					Content:    mediumContent,
					EOFMarker:  "EOF_M2",
					TargetPath: "/tmp/test/m2.cjs",
				},
				{
					Filename:   "m3.cjs",
					Content:    mediumContent,
					EOFMarker:  "EOF_M3",
					TargetPath: "/tmp/test/m3.cjs",
				},
			},
			expectedSteps: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yaml strings.Builder
			writeJavaScriptFilesInChunks(&yaml, tt.files, "Test Step")
			output := yaml.String()

			// Count the number of steps generated
			stepCount := strings.Count(output, "- name: Test Step")

			if stepCount != tt.expectedSteps {
				t.Errorf("Expected exactly %d step(s), got %d", tt.expectedSteps, stepCount)
			}

			// Verify all files are written
			for _, file := range tt.files {
				if !strings.Contains(output, file.TargetPath) {
					t.Errorf("Output should contain target path %s", file.TargetPath)
				}
				if !strings.Contains(output, file.EOFMarker) {
					t.Errorf("Output should contain EOF marker %s", file.EOFMarker)
				}
			}

			// Verify each step has proper structure
			if !strings.Contains(output, "- name:") {
				t.Error("Output should contain step name")
			}
			if !strings.Contains(output, "run: |") {
				t.Error("Output should contain run command")
			}
		})
	}
}

// TestJavaScriptFileChunkingPreservesOrder validates that file order is preserved across chunks
func TestJavaScriptFileChunkingPreservesOrder(t *testing.T) {
	files := []JavaScriptFileWrite{
		{Filename: "a.cjs", Content: "console.log('a');", EOFMarker: "EOF_A", TargetPath: "/tmp/a.cjs"},
		{Filename: "b.cjs", Content: "console.log('b');", EOFMarker: "EOF_B", TargetPath: "/tmp/b.cjs"},
		{Filename: "c.cjs", Content: "console.log('c');", EOFMarker: "EOF_C", TargetPath: "/tmp/c.cjs"},
	}

	var yaml strings.Builder
	writeJavaScriptFilesInChunks(&yaml, files, "Test")
	output := yaml.String()

	// Find positions of each file
	posA := strings.Index(output, "/tmp/a.cjs")
	posB := strings.Index(output, "/tmp/b.cjs")
	posC := strings.Index(output, "/tmp/c.cjs")

	if posA == -1 || posB == -1 || posC == -1 {
		t.Error("All files should be present in output")
	}

	// Verify order is preserved
	if !(posA < posB && posB < posC) {
		t.Errorf("File order should be preserved: A(%d) < B(%d) < C(%d)", posA, posB, posC)
	}
}
