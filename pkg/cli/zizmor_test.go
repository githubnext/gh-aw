package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestParseAndDisplayZizmorOutput(t *testing.T) {
	tests := []struct {
		name           string
		stdout         string
		stderr         string
		verbose        bool
		expectedOutput []string
		expectError    bool
	}{
		{
			name: "single file with findings",
			stdout: `[
  {
    "ident": "excessive-permissions",
    "determinations": {
      "severity": "Medium"
    },
    "locations": [
      {
        "symbolic": {
          "key": {
            "Local": {
              "given_path": "./.github/workflows/test.lock.yml"
            }
          }
        }
      }
    ]
  }
]`,
			stderr: " INFO audit: zizmor: 🌈 completed ./.github/workflows/test.lock.yml\n",
			expectedOutput: []string{
				"🌈 zizmor 1 warning in ./.github/workflows/test.lock.yml",
				"  - [Medium] excessive-permissions",
			},
			expectError: false,
		},
		{
			name: "multiple findings in same file",
			stdout: `[
  {
    "ident": "excessive-permissions",
    "determinations": {
      "severity": "Medium"
    },
    "locations": [
      {
        "symbolic": {
          "key": {
            "Local": {
              "given_path": "./.github/workflows/test.lock.yml"
            }
          }
        }
      }
    ]
  },
  {
    "ident": "template-injection",
    "determinations": {
      "severity": "High"
    },
    "locations": [
      {
        "symbolic": {
          "key": {
            "Local": {
              "given_path": "./.github/workflows/test.lock.yml"
            }
          }
        }
      }
    ]
  }
]`,
			stderr: " INFO audit: zizmor: 🌈 completed ./.github/workflows/test.lock.yml\n",
			expectedOutput: []string{
				"🌈 zizmor 2 warnings in ./.github/workflows/test.lock.yml",
				"  - [Medium] excessive-permissions",
				"  - [High] template-injection",
			},
			expectError: false,
		},
		{
			name:           "file with no findings",
			stdout:         "[]",
			stderr:         " INFO audit: zizmor: 🌈 completed ./.github/workflows/clean.lock.yml\n",
			expectedOutput: []string{
				// No output expected for 0 warnings
			},
			expectError: false,
		},
		{
			name: "multiple files",
			stdout: `[
  {
    "ident": "excessive-permissions",
    "determinations": {
      "severity": "Medium"
    },
    "locations": [
      {
        "symbolic": {
          "key": {
            "Local": {
              "given_path": "./.github/workflows/test1.lock.yml"
            }
          }
        }
      }
    ]
  },
  {
    "ident": "template-injection",
    "determinations": {
      "severity": "High"
    },
    "locations": [
      {
        "symbolic": {
          "key": {
            "Local": {
              "given_path": "./.github/workflows/test2.lock.yml"
            }
          }
        }
      }
    ]
  }
]`,
			stderr: " INFO audit: zizmor: 🌈 completed ./.github/workflows/test1.lock.yml\n INFO audit: zizmor: 🌈 completed ./.github/workflows/test2.lock.yml\n",
			expectedOutput: []string{
				"🌈 zizmor 1 warning in ./.github/workflows/test1.lock.yml",
				"  - [Medium] excessive-permissions",
				"🌈 zizmor 1 warning in ./.github/workflows/test2.lock.yml",
				"  - [High] template-injection",
			},
			expectError: false,
		},
		{
			name: "finding with multiple locations in same file counts as one",
			stdout: `[
  {
    "ident": "excessive-permissions",
    "determinations": {
      "severity": "Medium"
    },
    "locations": [
      {
        "symbolic": {
          "key": {
            "Local": {
              "given_path": "./.github/workflows/test.lock.yml"
            }
          }
        }
      },
      {
        "symbolic": {
          "key": {
            "Local": {
              "given_path": "./.github/workflows/test.lock.yml"
            }
          }
        }
      }
    ]
  }
]`,
			stderr: " INFO audit: zizmor: 🌈 completed ./.github/workflows/test.lock.yml\n",
			expectedOutput: []string{
				"🌈 zizmor 1 warning in ./.github/workflows/test.lock.yml",
				"  - [Medium] excessive-permissions",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stderr output
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			warningCount, err := parseAndDisplayZizmorOutput(tt.stdout, tt.stderr, tt.verbose)

			// Restore stderr
			w.Close()
			os.Stderr = oldStderr

			// Read captured output
			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			// Check error expectation
			if tt.expectError && err == nil {
				t.Errorf("Expected an error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Verify warning count is non-negative
			if warningCount < 0 {
				t.Errorf("Warning count should be non-negative, got: %d", warningCount)
			}

			// Check expected output
			for _, expected := range tt.expectedOutput {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, but got:\n%s", expected, output)
				}
			}
		})
	}
}
