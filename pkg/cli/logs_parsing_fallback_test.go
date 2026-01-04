package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLogFileWithEngine_FallbackParser(t *testing.T) {
	tests := []struct {
		name           string
		logContent     string
		expectedErrors int
		expectedWarns  int
	}{
		{
			name: "GitHub Actions workflow commands",
			logContent: `2024-01-04T10:00:00.000Z Some log line
::error::Failed to connect to database
::warning::Deprecated API used
::error::Invalid configuration
Some other log line
::warning::Memory usage high`,
			expectedErrors: 2,
			expectedWarns:  2,
		},
		{
			name: "Generic error patterns",
			logContent: `2024-01-04T10:00:00.000Z Starting process
ERROR: Connection timeout
Warning: Cache miss detected
Error: Invalid token
WARNING: Resource limit reached`,
			expectedErrors: 2,
			expectedWarns:  2,
		},
		{
			name: "Mixed error formats",
			logContent: `::error::GitHub Actions error
ERROR: Generic error message
::warning::GitHub Actions warning
Warning: Generic warning message`,
			expectedErrors: 2,
			expectedWarns:  2,
		},
		{
			name: "No errors or warnings",
			logContent: `2024-01-04T10:00:00.000Z Starting process
INFO: Process started successfully
DEBUG: Loading configuration
INFO: Configuration loaded`,
			expectedErrors: 0,
			expectedWarns:  0,
		},
		{
			name:           "Empty log file",
			logContent:     "",
			expectedErrors: 0,
			expectedWarns:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary log file
			tempDir := t.TempDir()
			logFile := filepath.Join(tempDir, "test.log")
			err := os.WriteFile(logFile, []byte(tt.logContent), 0644)
			require.NoError(t, err, "Failed to create test log file")

			// Parse the log file without an engine (fallback mode)
			metrics, err := parseLogFileWithEngine(logFile, nil, false, false)
			require.NoError(t, err, "parseLogFileWithEngine should not return an error")

			// Count errors and warnings
			errorCount := 0
			warnCount := 0
			for _, logErr := range metrics.Errors {
				if logErr.Type == "error" {
					errorCount++
				} else if logErr.Type == "warning" {
					warnCount++
				}
			}

			assert.Equal(t, tt.expectedErrors, errorCount, "Error count mismatch")
			assert.Equal(t, tt.expectedWarns, warnCount, "Warning count mismatch")
		})
	}
}

func TestParseLogFileWithEngine_FallbackVsEngineSpecific(t *testing.T) {
	// Test that fallback parser works but engine-specific parser is still preferred
	logContent := `::error::GitHub Actions error
ERROR: Generic error
::warning::GitHub Actions warning`

	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")
	err := os.WriteFile(logFile, []byte(logContent), 0644)
	require.NoError(t, err)

	// Test 1: No engine (fallback parser)
	metricsNoEngine, err := parseLogFileWithEngine(logFile, nil, false, false)
	require.NoError(t, err)
	assert.Greater(t, len(metricsNoEngine.Errors), 0, "Fallback parser should detect errors")

	// Test 2: With engine - the engine will parse using its own logic
	// We're just testing that the code path works, actual parsing depends on engine
	// In this test we don't have a real engine, so we skip this part
	// Real engines are tested in their respective test files
}

func TestParseLogFileWithEngine_NoAwInfoJson(t *testing.T) {
	// Simulate a scenario where aw_info.json is missing
	// This should trigger the fallback parser

	logContent := `Starting workflow
::error::Configuration file not found
::warning::Using default configuration
Processing data
ERROR: Database connection failed
Done`

	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "agent-stdio.log")
	err := os.WriteFile(logFile, []byte(logContent), 0644)
	require.NoError(t, err)

	// Parse without engine (simulating missing aw_info.json)
	metrics, err := parseLogFileWithEngine(logFile, nil, false, false)
	require.NoError(t, err)

	// Verify that errors were detected
	errorCount := 0
	warnCount := 0
	for _, logErr := range metrics.Errors {
		if logErr.Type == "error" {
			errorCount++
		} else if logErr.Type == "warning" {
			warnCount++
		}
	}

	assert.Equal(t, 2, errorCount, "Should detect 2 errors")
	assert.Equal(t, 1, warnCount, "Should detect 1 warning")
}
