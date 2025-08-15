package workflow

import (
	"testing"
)

func TestClaudeEngine_ParseLogMetrics_Basic(t *testing.T) {
	engine := NewClaudeEngine()

	tests := []struct {
		name          string
		logContent    string
		verbose       bool
		expectNoCrash bool
	}{
		{
			name:          "empty log content",
			logContent:    "",
			verbose:       false,
			expectNoCrash: true,
		},
		{
			name:          "whitespace only",
			logContent:    "   \n\t   \n   ",
			verbose:       false,
			expectNoCrash: true,
		},
		{
			name: "simple log with errors",
			logContent: `Starting process...
Error: Something went wrong
Warning: Deprecated feature
Process completed`,
			verbose:       false,
			expectNoCrash: true,
		},
		{
			name: "verbose mode",
			logContent: `Debug: Starting
Processing...
Debug: Completed`,
			verbose:       true,
			expectNoCrash: true,
		},
		{
			name: "multiline content",
			logContent: `Line 1
Line 2
Line 3
Line 4
Line 5`,
			verbose:       false,
			expectNoCrash: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The main test is that this doesn't crash
			func() {
				defer func() {
					if r := recover(); r != nil {
						if tt.expectNoCrash {
							t.Errorf("ParseLogMetrics crashed unexpectedly: %v", r)
						}
					}
				}()

				metrics := engine.ParseLogMetrics(tt.logContent, tt.verbose)

				// Basic validation - should return valid struct
				if metrics.ErrorCount < 0 {
					t.Errorf("ErrorCount should not be negative, got %d", metrics.ErrorCount)
				}
				if metrics.WarningCount < 0 {
					t.Errorf("WarningCount should not be negative, got %d", metrics.WarningCount)
				}
				if metrics.TokenUsage < 0 {
					t.Errorf("TokenUsage should not be negative, got %d", metrics.TokenUsage)
				}
				if metrics.EstimatedCost < 0 {
					t.Errorf("EstimatedCost should not be negative, got %f", metrics.EstimatedCost)
				}
			}()
		})
	}
}

func TestCodexEngine_ParseLogMetrics_Basic(t *testing.T) {
	engine := NewCodexEngine()

	tests := []struct {
		name          string
		logContent    string
		verbose       bool
		expectNoCrash bool
	}{
		{
			name:          "empty log content",
			logContent:    "",
			verbose:       false,
			expectNoCrash: true,
		},
		{
			name:          "whitespace only",
			logContent:    "   \n\t   \n   ",
			verbose:       false,
			expectNoCrash: true,
		},
		{
			name: "simple log with errors",
			logContent: `Starting process...
Error: Something went wrong
Warning: Deprecated feature
Process completed`,
			verbose:       false,
			expectNoCrash: true,
		},
		{
			name: "verbose mode",
			logContent: `Debug: Starting
Processing...
Debug: Completed`,
			verbose:       true,
			expectNoCrash: true,
		},
		{
			name: "multiline content",
			logContent: `Line 1
Line 2
Line 3
Line 4
Line 5`,
			verbose:       false,
			expectNoCrash: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The main test is that this doesn't crash
			func() {
				defer func() {
					if r := recover(); r != nil {
						if tt.expectNoCrash {
							t.Errorf("ParseLogMetrics crashed unexpectedly: %v", r)
						}
					}
				}()

				metrics := engine.ParseLogMetrics(tt.logContent, tt.verbose)

				// Basic validation - should return valid struct
				if metrics.ErrorCount < 0 {
					t.Errorf("ErrorCount should not be negative, got %d", metrics.ErrorCount)
				}
				if metrics.WarningCount < 0 {
					t.Errorf("WarningCount should not be negative, got %d", metrics.WarningCount)
				}
				if metrics.TokenUsage < 0 {
					t.Errorf("TokenUsage should not be negative, got %d", metrics.TokenUsage)
				}
				// Codex engine doesn't track cost, so it should be 0
				if metrics.EstimatedCost != 0 {
					t.Errorf("Codex engine should have 0 cost, got %f", metrics.EstimatedCost)
				}
			}()
		})
	}
}

func TestCompiler_SetFileTracker_Simple(t *testing.T) {
	// Create compiler
	compiler := NewCompiler(false, "", "test-version")

	// Initial state should have nil tracker
	if compiler.fileTracker != nil {
		t.Errorf("Expected initial fileTracker to be nil")
	}

	// Create mock tracker
	mockTracker := &SimpleMockFileTracker{}

	// Set tracker
	compiler.SetFileTracker(mockTracker)

	// Verify tracker was set
	if compiler.fileTracker != mockTracker {
		t.Errorf("Expected tracker to be set")
	}

	// Set to nil
	compiler.SetFileTracker(nil)

	// Verify tracker is nil
	if compiler.fileTracker != nil {
		t.Errorf("Expected tracker to be nil after setting to nil")
	}
}

// SimpleMockFileTracker is a basic implementation for testing
type SimpleMockFileTracker struct {
	tracked []string
}

func (s *SimpleMockFileTracker) TrackCreated(filePath string) {
	if s.tracked == nil {
		s.tracked = make([]string, 0)
	}
	s.tracked = append(s.tracked, filePath)
}
