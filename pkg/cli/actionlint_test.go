//go:build !integration

package cli

import (
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAndDisplayActionlintOutput(t *testing.T) {
	tests := []struct {
		name           string
		stdout         string
		verbose        bool
		expectedOutput []string
		expectError    bool
		expectedCount  int
		expectedKinds  map[string]int
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
			expectedKinds: map[string]int{"runner-label": 1},
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
			expectedKinds: map[string]int{"runner-label": 1, "shellcheck": 1},
		},
		{
			name:           "no errors - empty output",
			stdout:         "",
			expectedOutput: []string{},
			expectError:    false,
			expectedCount:  0,
			expectedKinds:  map[string]int{},
		},
		{
			name:        "invalid JSON",
			stdout:      `{invalid json}`,
			expectError: true,
		},
		{
			name: "multiple errors from multiple files",
			stdout: `[
{"message":"label \"ubuntu-slim\" is unknown","filepath":".github/workflows/test1.lock.yml","line":10,"column":14,"kind":"runner-label","snippet":"    runs-on: ubuntu-slim\n             ^~~~~~~~~~~","end_column":24},
{"message":"shellcheck reported issue","filepath":".github/workflows/test2.lock.yml","line":25,"column":9,"kind":"shellcheck","snippet":"        run: |\n        ^~~~","end_column":12}
]`,
			expectedOutput: []string{
				".github/workflows/test1.lock.yml:10:14: error: [runner-label]",
				".github/workflows/test2.lock.yml:25:9: error: [shellcheck]",
			},
			expectError:   false,
			expectedCount: 2,
			expectedKinds: map[string]int{"runner-label": 1, "shellcheck": 1},
		},
		{
			name: "errors from three files",
			stdout: `[
{"message":"error 1","filepath":".github/workflows/a.lock.yml","line":10,"column":1,"kind":"error","snippet":"test","end_column":5},
{"message":"error 2","filepath":".github/workflows/b.lock.yml","line":20,"column":1,"kind":"error","snippet":"test","end_column":5},
{"message":"error 3","filepath":".github/workflows/c.lock.yml","line":30,"column":1,"kind":"error","snippet":"test","end_column":5}
]`,
			expectedOutput: []string{
				".github/workflows/a.lock.yml:10:1",
				".github/workflows/b.lock.yml:20:1",
				".github/workflows/c.lock.yml:30:1",
			},
			expectError:   false,
			expectedCount: 3,
			expectedKinds: map[string]int{"error": 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var count int
			var kinds map[string]int
			var err error

			output := testutil.CaptureStderr(t, func() {
				count, kinds, err = parseAndDisplayActionlintOutput(tt.stdout, tt.verbose)
			})

			if tt.expectError {
				require.Error(t, err, "should return error for invalid input")
			} else {
				require.NoError(t, err, "should not return error for valid input")
				assert.Equal(t, tt.expectedCount, count, "error count should match expected")
				if tt.expectedKinds != nil {
					assert.Equal(t, tt.expectedKinds, kinds, "error kinds should match expected")
				}
				for _, expected := range tt.expectedOutput {
					assert.Contains(t, output, expected,
						"output should contain %q", expected)
				}
			}
		})
	}
}

func TestGetActionlintVersion(t *testing.T) {
	original := actionlintVersion
	defer func() { actionlintVersion = original }()

	actionlintVersion = "1.7.9"
	version, err := getActionlintVersion()
	require.NoError(t, err, "should not error when version is cached")
	assert.Equal(t, "1.7.9", version, "should return cached version")
}

func TestDisplayActionlintSummary(t *testing.T) {
	tests := []struct {
		name             string
		stats            *ActionlintStats
		expectedContains []string
	}{
		{
			name: "summary with errors and warnings",
			stats: &ActionlintStats{
				TotalWorkflows: 5,
				TotalErrors:    10,
				TotalWarnings:  3,
				ErrorsByKind: map[string]int{
					"runner-label": 5,
					"shellcheck":   5,
				},
			},
			expectedContains: []string{
				"Actionlint Summary",
				"Checked 5 workflow(s)",
				"Found 13 issue(s)",
				"10 error(s), 3 warning(s)",
				"Issues by type:",
				"runner-label: 5",
				"shellcheck: 5",
			},
		},
		{
			name: "summary with only errors",
			stats: &ActionlintStats{
				TotalWorkflows: 3,
				TotalErrors:    7,
				TotalWarnings:  0,
				ErrorsByKind: map[string]int{
					"syntax": 7,
				},
			},
			expectedContains: []string{
				"Actionlint Summary",
				"Checked 3 workflow(s)",
				"Found 7 issue(s)",
				"7 error(s)",
				"Issues by type:",
				"syntax: 7",
			},
		},
		{
			name: "summary with no issues",
			stats: &ActionlintStats{
				TotalWorkflows: 10,
				TotalErrors:    0,
				TotalWarnings:  0,
				ErrorsByKind:   map[string]int{},
			},
			expectedContains: []string{
				"Actionlint Summary",
				"Checked 10 workflow(s)",
				"No issues found",
			},
		},
		{
			name:             "nil stats - no output",
			stats:            nil,
			expectedContains: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalStats := actionlintStats
			defer func() { actionlintStats = originalStats }()
			actionlintStats = tt.stats

			output := testutil.CaptureStderr(t, displayActionlintSummary)

			for _, expected := range tt.expectedContains {
				assert.Contains(t, output, expected,
					"output should contain %q", expected)
			}
		})
	}
}

func TestInitActionlintStats(t *testing.T) {
	originalStats := actionlintStats
	defer func() { actionlintStats = originalStats }()

	initActionlintStats()

	require.NotNil(t, actionlintStats, "actionlintStats should not be nil after initialization")
	assert.Zero(t, actionlintStats.TotalWorkflows, "TotalWorkflows should start at 0")
	assert.Zero(t, actionlintStats.TotalErrors, "TotalErrors should start at 0")
	assert.Zero(t, actionlintStats.TotalWarnings, "TotalWarnings should start at 0")
	assert.NotNil(t, actionlintStats.ErrorsByKind, "ErrorsByKind map should be initialized")
	assert.Empty(t, actionlintStats.ErrorsByKind, "ErrorsByKind should start empty")
}

func TestGetActionlintDocsURL(t *testing.T) {
	tests := []struct {
		name     string
		kind     string
		expected string
	}{
		{
			name:     "empty kind returns base URL",
			kind:     "",
			expected: "https://github.com/rhysd/actionlint/blob/main/docs/checks.md",
		},
		{
			name:     "runner-label kind",
			kind:     "runner-label",
			expected: "https://github.com/rhysd/actionlint/blob/main/docs/checks.md#check-runner-labels",
		},
		{
			name:     "shellcheck kind",
			kind:     "shellcheck",
			expected: "https://github.com/rhysd/actionlint/blob/main/docs/checks.md#check-shellcheck-integ",
		},
		{
			name:     "pyflakes kind",
			kind:     "pyflakes",
			expected: "https://github.com/rhysd/actionlint/blob/main/docs/checks.md#check-pyflakes-integ",
		},
		{
			name:     "expression kind",
			kind:     "expression",
			expected: "https://github.com/rhysd/actionlint/blob/main/docs/checks.md#check-syntax-expression",
		},
		{
			name:     "generic kind with check- prefix",
			kind:     "check-job-deps",
			expected: "https://github.com/rhysd/actionlint/blob/main/docs/checks.md#check-job-deps",
		},
		{
			name:     "generic kind without check- prefix",
			kind:     "job-deps",
			expected: "https://github.com/rhysd/actionlint/blob/main/docs/checks.md#check-job-deps",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getActionlintDocsURL(tt.kind)
			assert.Equal(t, tt.expected, result, "docs URL should match expected for kind %q", tt.kind)
		})
	}
}
