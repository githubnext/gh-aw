package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestParseAndDisplayActionlintOutput(t *testing.T) {
	tests := []struct {
		name           string
		stdout         string
		verbose        bool
		expectedOutput []string
		expectError    bool
		expectedCount  int
	}{
		{
			name: "single error",
			stdout: `[
{"message":"label \"ubuntu-slim\" is unknown. available labels are \"ubuntu-latest\", \"ubuntu-22.04\", \"ubuntu-20.04\", \"windows-latest\", \"windows-2022\", \"windows-2019\", \"macos-latest\", \"macos-13\", \"macos-12\", \"macos-11\". if it is a custom label for self-hosted runner, set list of labels in actionlint.yaml config file","filepath":".github/workflows/test.lock.yml","line":10,"column":14,"kind":"runner-label","snippet":"    runs-on: ubuntu-slim\n             ^~~~~~~~~~~","end_column":24}
]`,
			expectedOutput: []string{
				".github/workflows/test.lock.yml:10:14: error: [runner-label] label \"ubuntu-slim\" is unknown",
			},
			expectError:   false,
			expectedCount: 1,
		},
		{
			name: "multiple errors",
			stdout: `[
{"message":"label \"ubuntu-slim\" is unknown. available labels are \"ubuntu-latest\", \"ubuntu-22.04\", \"ubuntu-20.04\", \"windows-latest\", \"windows-2022\", \"windows-2019\", \"macos-latest\", \"macos-13\", \"macos-12\", \"macos-11\". if it is a custom label for self-hosted runner, set list of labels in actionlint.yaml config file","filepath":".github/workflows/test.lock.yml","line":10,"column":14,"kind":"runner-label","snippet":"    runs-on: ubuntu-slim\n             ^~~~~~~~~~~","end_column":24},
{"message":"shellcheck reported issue in this script: SC2086:info:1:8: Double quote to prevent globbing and word splitting","filepath":".github/workflows/test.lock.yml","line":25,"column":9,"kind":"shellcheck","snippet":"        run: |\n        ^~~~","end_column":12}
]`,
			expectedOutput: []string{
				".github/workflows/test.lock.yml:10:14: error: [runner-label] label \"ubuntu-slim\" is unknown",
				".github/workflows/test.lock.yml:25:9: error: [shellcheck] shellcheck reported issue",
			},
			expectError:   false,
			expectedCount: 2,
		},
		{
			name:           "no errors - empty output",
			stdout:         "",
			expectedOutput: []string{},
			expectError:    false,
			expectedCount:  0,
		},
		{
			name:        "invalid JSON",
			stdout:      `{invalid json}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stderr output
			originalStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			count, err := parseAndDisplayActionlintOutput(tt.stdout, tt.verbose)

			// Restore stderr and get output
			w.Close()
			os.Stderr = originalStderr
			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			// Check error expectation
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check count
			if count != tt.expectedCount {
				t.Errorf("Expected count %d, got %d", tt.expectedCount, count)
			}

			// Check expected output strings are present
			for _, expected := range tt.expectedOutput {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, but it didn't.\nGot: %s", expected, output)
				}
			}
		})
	}
}

func TestGetActionlintVersion(t *testing.T) {
	// Reset the cached version before test
	originalVersion := actionlintVersion
	defer func() { actionlintVersion = originalVersion }()

	tests := []struct {
		name          string
		presetVersion string
		expectCached  bool
	}{
		{
			name:          "first call fetches version",
			presetVersion: "",
			expectCached:  false,
		},
		{
			name:          "second call returns cached version",
			presetVersion: "1.7.9",
			expectCached:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actionlintVersion = tt.presetVersion

			// If we preset a version, this should return immediately
			if tt.expectCached {
				version, err := getActionlintVersion()
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if version != tt.presetVersion {
					t.Errorf("Expected cached version %q, got %q", tt.presetVersion, version)
				}
			}
		})
	}
}
