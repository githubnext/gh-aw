package workflow

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogParserSnapshots(t *testing.T) {
	testDataDir := "test_data"

	tests := []struct {
		name         string
		engine       string
		logFile      string
		expectedFile string
	}{
		{
			name:         "Claude log parsing",
			engine:       "claude",
			logFile:      "sample_claude_log.txt",
			expectedFile: "expected_claude_baseline.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Read the sample log file
			logPath := filepath.Join(testDataDir, tt.logFile)
			logContent, err := os.ReadFile(logPath)
			if err != nil {
				t.Fatalf("Failed to read log file %s: %v", logPath, err)
			}

			// Get the JavaScript parser script
			scriptName := fmt.Sprintf("parse_%s_log", tt.engine)
			jsScript := GetLogParserScript(scriptName)

			if jsScript == "" {
				t.Fatalf("Failed to get log parser script for %s: script is empty", tt.engine)
			}

			// Generate markdown using our JavaScript parser
			markdown, err := runJSLogParser(jsScript, string(logContent))
			if err != nil {
				t.Fatalf("Failed to run JavaScript log parser: %v", err)
			}

			// Read the expected baseline
			expectedPath := filepath.Join(testDataDir, tt.expectedFile)
			expectedContent, err := os.ReadFile(expectedPath)
			if err != nil {
				t.Fatalf("Failed to read expected baseline %s: %v", expectedPath, err)
			}

			// Compare the results
			if markdown != string(expectedContent) {
				// Update the baseline file for manual inspection
				t.Logf("Output differs from baseline. Updating %s for manual inspection", expectedPath)
				if err := os.WriteFile(expectedPath, []byte(markdown), 0644); err != nil {
					t.Errorf("Failed to update baseline file: %v", err)
				}

				// Fail the test so user can inspect changes
				t.Errorf("Generated markdown differs from baseline.\n"+
					"Expected file: %s\n"+
					"Generated length: %d\n"+
					"Expected length: %d\n"+
					"The baseline file has been updated with the new output for manual inspection.\n"+
					"Please review the changes and commit if they are correct.",
					expectedPath, len(markdown), len(expectedContent))
			}
		})
	}
}

// runJSLogParser executes the JavaScript log parser and returns the markdown output
func runJSLogParser(jsScript, logContent string) (string, error) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "log_parser_test")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Write the log content to a temporary file
	logFile := filepath.Join(tempDir, "test.log")
	if err := os.WriteFile(logFile, []byte(logContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write log file: %v", err)
	}

	// Create a Node.js script that will run our parser
	nodeScript := fmt.Sprintf(`
const fs = require('fs');

// Mock @actions/core for testing
const core = {
	summary: {
		addRaw: function(content) {
			this._content = content;
			return this;
		},
		write: function() {
			console.log(this._content);
		},
		_content: ''
	},
	setFailed: function(message) {
		console.error('FAILED:', message);
		// Don't exit in test - just mark the failure
		this._failed = true;
		this._failureMessage = message;
	},
	info: function(message) {
		// Mock info function for testing
	}
};

// Set up environment
process.env.GH_AW_AGENT_OUTPUT = '%s';

// Override require to provide our mock
const originalRequire = require;
require = function(name) {
	if (name === '@actions/core') {
		return core;
	}
	return originalRequire.apply(this, arguments);
};

// Execute the parser script
%s
`, logFile, jsScript)

	// Write the Node.js script
	nodeFile := filepath.Join(tempDir, "test.js")
	if err := os.WriteFile(nodeFile, []byte(nodeScript), 0644); err != nil {
		return "", fmt.Errorf("failed to write node script: %v", err)
	}

	// Execute the Node.js script
	cmd := exec.Command("node", "test.js")
	cmd.Dir = tempDir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to execute node script: %v\nOutput: %s", err, string(output))
	}

	return strings.TrimSpace(string(output)), nil
}

func TestLogParserScriptRetrieval(t *testing.T) {
	tests := []string{
		"parse_claude_log",
		"parse_codex_log",
	}

	for _, scriptName := range tests {
		t.Run(scriptName, func(t *testing.T) {
			script := GetLogParserScript(scriptName)
			if script == "" {
				t.Fatalf("Failed to get script %s: script is empty", scriptName)
			}

			if len(script) == 0 {
				t.Errorf("Script %s is empty", scriptName)
			}

			// Basic validation that it contains expected functions
			expectedFunctions := []string{
				"function main(",
				"function parse",
			}

			for _, expected := range expectedFunctions {
				if !strings.Contains(script, expected) {
					t.Errorf("Script %s missing expected content: %s", scriptName, expected)
				}
			}
		})
	}
}
