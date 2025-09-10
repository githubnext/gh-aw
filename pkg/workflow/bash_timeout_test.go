package workflow

import (
	"testing"
)

func TestBashToolTimeout(t *testing.T) {
	engine := NewClaudeEngine()

	tests := []struct {
		name     string
		tools    map[string]any
		expected string
	}{
		{
			name: "bash with timeout only",
			tools: map[string]any{
				"bash": map[string]any{
					"timeout": 30,
				},
			},
			expected: "Bash,BashOutput,ExitPlanMode,Glob,Grep,KillBash,LS,NotebookRead,Read,Task,TodoWrite",
		},
		{
			name: "bash with timeout and commands",
			tools: map[string]any{
				"bash": map[string]any{
					"timeout":  60,
					"commands": []any{"echo", "ls"},
				},
			},
			expected: "Bash(echo),Bash(ls),BashOutput,ExitPlanMode,Glob,Grep,KillBash,LS,NotebookRead,Read,Task,TodoWrite",
		},
		{
			name: "bash with timeout and allowed field for commands",
			tools: map[string]any{
				"bash": map[string]any{
					"timeout": 45,
					"allowed": []any{"git", "npm"},
				},
			},
			expected: "Bash(git),Bash(npm),BashOutput,ExitPlanMode,Glob,Grep,KillBash,LS,NotebookRead,Read,Task,TodoWrite",
		},
		{
			name: "bash with commands field should override allowed field",
			tools: map[string]any{
				"bash": map[string]any{
					"timeout":  30,
					"commands": []any{"echo"},
					"allowed":  []any{"git"},
				},
			},
			expected: "Bash(echo),BashOutput,ExitPlanMode,Glob,Grep,KillBash,LS,NotebookRead,Read,Task,TodoWrite",
		},
		{
			name: "bash with string timeout should work",
			tools: map[string]any{
				"bash": map[string]any{
					"timeout": "90",
				},
			},
			expected: "Bash,BashOutput,ExitPlanMode,Glob,Grep,KillBash,LS,NotebookRead,Read,Task,TodoWrite",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.computeAllowedClaudeToolsString(tt.tools, nil)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestExpandNeutralToolsWithBashTimeout(t *testing.T) {
	engine := NewClaudeEngine()

	tests := []struct {
		name     string
		input    map[string]any
		expected map[string]any
	}{
		{
			name: "bash tool with timeout only",
			input: map[string]any{
				"bash": map[string]any{
					"timeout": 30,
				},
			},
			expected: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Bash": nil, // All commands allowed when no commands specified
					},
					"timeout": map[string]any{
						"bash": 30,
					},
				},
			},
		},
		{
			name: "bash tool with timeout and commands",
			input: map[string]any{
				"bash": map[string]any{
					"timeout":  60,
					"commands": []any{"echo", "ls"},
				},
			},
			expected: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Bash": []any{"echo", "ls"},
					},
					"timeout": map[string]any{
						"bash": 60,
					},
				},
			},
		},
		{
			name: "mixed tools with bash timeout",
			input: map[string]any{
				"bash": map[string]any{
					"timeout": 45,
					"allowed": []any{"git"},
				},
				"web-fetch": nil,
			},
			expected: map[string]any{
				"claude": map[string]any{
					"allowed": map[string]any{
						"Bash":     []any{"git"},
						"WebFetch": nil,
					},
					"timeout": map[string]any{
						"bash": 45,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.expandNeutralToolsToClaudeTools(tt.input)

			// Check claude section
			claudeResult, hasClaudeResult := result["claude"]
			claudeExpected, hasClaudeExpected := tt.expected["claude"]

			if hasClaudeExpected != hasClaudeResult {
				t.Errorf("Claude section presence mismatch. Expected: %v, Got: %v", hasClaudeExpected, hasClaudeResult)
				return
			}

			if hasClaudeExpected {
				claudeResultMap, ok1 := claudeResult.(map[string]any)
				claudeExpectedMap, ok2 := claudeExpected.(map[string]any)

				if !ok1 || !ok2 {
					t.Errorf("Claude section type mismatch")
					return
				}

				// Check allowed section
				_, hasAllowedResult := claudeResultMap["allowed"]
				_, hasAllowedExpected := claudeExpectedMap["allowed"]

				if hasAllowedExpected != hasAllowedResult {
					t.Errorf("Claude allowed section presence mismatch. Expected: %v, Got: %v", hasAllowedExpected, hasAllowedResult)
					return
				}

				// Check timeout section
				timeoutResult, hasTimeoutResult := claudeResultMap["timeout"]
				timeoutExpected, hasTimeoutExpected := claudeExpectedMap["timeout"]

				if hasTimeoutExpected != hasTimeoutResult {
					t.Errorf("Claude timeout section presence mismatch. Expected: %v, Got: %v", hasTimeoutExpected, hasTimeoutResult)
					return
				}

				if hasTimeoutExpected {
					timeoutResultMap, ok1 := timeoutResult.(map[string]any)
					timeoutExpectedMap, ok2 := timeoutExpected.(map[string]any)

					if !ok1 || !ok2 {
						t.Errorf("Claude timeout section type mismatch")
						return
					}

					// Check bash timeout value
					bashTimeoutResult, hasBashTimeoutResult := timeoutResultMap["bash"]
					bashTimeoutExpected, hasBashTimeoutExpected := timeoutExpectedMap["bash"]

					if hasBashTimeoutExpected != hasBashTimeoutResult {
						t.Errorf("Bash timeout presence mismatch. Expected: %v, Got: %v", hasBashTimeoutExpected, hasBashTimeoutResult)
						return
					}

					if hasBashTimeoutExpected && bashTimeoutResult != bashTimeoutExpected {
						t.Errorf("Bash timeout value mismatch. Expected: %v, Got: %v", bashTimeoutExpected, bashTimeoutResult)
					}
				}
			}
		})
	}
}

func TestExtractBashTimeoutEnvVars(t *testing.T) {
	engine := NewClaudeEngine()

	tests := []struct {
		name     string
		tools    map[string]any
		expected map[string]string
	}{
		{
			name:     "no tools",
			tools:    nil,
			expected: map[string]string{},
		},
		{
			name: "bash tool without timeout",
			tools: map[string]any{
				"bash": []any{"echo"},
			},
			expected: map[string]string{},
		},
		{
			name: "bash tool with integer timeout",
			tools: map[string]any{
				"bash": map[string]any{
					"timeout": 30,
				},
			},
			expected: map[string]string{
				"BASH_DEFAULT_TIMEOUT_MS": "30000",
				"BASH_MAX_TIMEOUT_MS":     "30000",
			},
		},
		{
			name: "bash tool with float timeout",
			tools: map[string]any{
				"bash": map[string]any{
					"timeout": 45.5,
				},
			},
			expected: map[string]string{
				"BASH_DEFAULT_TIMEOUT_MS": "45500",
				"BASH_MAX_TIMEOUT_MS":     "45500",
			},
		},
		{
			name: "bash tool with string timeout",
			tools: map[string]any{
				"bash": map[string]any{
					"timeout": "60",
				},
			},
			expected: map[string]string{
				"BASH_DEFAULT_TIMEOUT_MS": "60000",
				"BASH_MAX_TIMEOUT_MS":     "60000",
			},
		},
		{
			name: "bash tool with pre-millisecond timeout string",
			tools: map[string]any{
				"bash": map[string]any{
					"timeout": "120000",
				},
			},
			expected: map[string]string{
				"BASH_DEFAULT_TIMEOUT_MS": "120000000",
				"BASH_MAX_TIMEOUT_MS":     "120000000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.extractBashTimeoutEnvVars(tt.tools)
			
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d env vars, got %d. Expected: %+v, Got: %+v", len(tt.expected), len(result), tt.expected, result)
				return
			}
			
			for expectedKey, expectedValue := range tt.expected {
				if actualValue, exists := result[expectedKey]; !exists {
					t.Errorf("Expected env var %s not found", expectedKey)
				} else if actualValue != expectedValue {
					t.Errorf("Expected %s=%s, got %s=%s", expectedKey, expectedValue, expectedKey, actualValue)
				}
			}
		})
	}
}

func TestValidateBashTimeoutSupport(t *testing.T) {
tests := []struct {
name        string
engineID    string
tools       map[string]any
expectError bool
errorMsg    string
}{
{
name:     "claude engine with bash timeout should pass",
engineID: "claude",
tools: map[string]any{
"bash": map[string]any{
"timeout": 30,
},
},
expectError: false,
},
{
name:     "codex engine without bash timeout should pass",
engineID: "codex",
tools: map[string]any{
"bash": []any{"echo"},
},
expectError: false,
},
{
name:     "codex engine with bash timeout should fail",
engineID: "codex",
tools: map[string]any{
"bash": map[string]any{
"timeout": 30,
},
},
expectError: true,
errorMsg:    "bash tool timeout configuration is not supported by engine 'codex'",
},
{
name:     "custom engine with bash timeout should fail",
engineID: "custom",
tools: map[string]any{
"bash": map[string]any{
"timeout":  45,
"commands": []any{"echo"},
},
},
expectError: true,
errorMsg:    "bash tool timeout configuration is not supported by engine 'custom'",
},
{
name:     "no bash tool should pass for any engine",
engineID: "codex",
tools: map[string]any{
"web-fetch": nil,
},
expectError: false,
},
}

compiler := NewCompiler(false, "", "test")
engines := NewEngineRegistry()

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
engine, err := engines.GetEngine(tt.engineID)
if err != nil {
t.Fatalf("Engine %s not found: %v", tt.engineID, err)
}

err = compiler.validateBashTimeoutSupport(tt.tools, engine)

if tt.expectError {
if err == nil {
t.Errorf("Expected error but got none")
} else if err.Error() != tt.errorMsg {
t.Errorf("Expected error message \"%s\", got \"%s\"", tt.errorMsg, err.Error())
}
} else {
if err != nil {
t.Errorf("Expected no error but got: %v", err)
}
}
})
}
}
