//go:build !integration

package cli

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/github/gh-aw/pkg/workflow"
)

func TestParseRunURL(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		expectedRunID    int64
		expectedOwner    string
		expectedRepo     string
		expectedHostname string
		shouldErr        bool
	}{
		{
			name:             "Numeric run ID",
			input:            "1234567890",
			expectedRunID:    1234567890,
			expectedOwner:    "",
			expectedRepo:     "",
			expectedHostname: "",
			shouldErr:        false,
		},
		{
			name:             "Run URL with /actions/runs/",
			input:            "https://github.com/owner/repo/actions/runs/12345678",
			expectedRunID:    12345678,
			expectedOwner:    "owner",
			expectedRepo:     "repo",
			expectedHostname: "github.com",
			shouldErr:        false,
		},
		{
			name:             "Job URL",
			input:            "https://github.com/owner/repo/actions/runs/12345678/job/98765432",
			expectedRunID:    12345678,
			expectedOwner:    "owner",
			expectedRepo:     "repo",
			expectedHostname: "github.com",
			shouldErr:        false,
		},
		{
			name:             "Workflow run URL without /actions/",
			input:            "https://github.com/owner/repo/runs/12345678",
			expectedRunID:    12345678,
			expectedOwner:    "owner",
			expectedRepo:     "repo",
			expectedHostname: "github.com",
			shouldErr:        false,
		},
		{
			name:             "GitHub Enterprise URL",
			input:            "https://github.example.com/owner/repo/actions/runs/12345678",
			expectedRunID:    12345678,
			expectedOwner:    "owner",
			expectedRepo:     "repo",
			expectedHostname: "github.example.com",
			shouldErr:        false,
		},
		{
			name:             "GitHub Enterprise URL without /actions/",
			input:            "https://ghe.company.com/myorg/myrepo/runs/99999",
			expectedRunID:    99999,
			expectedOwner:    "myorg",
			expectedRepo:     "myrepo",
			expectedHostname: "ghe.company.com",
			shouldErr:        false,
		},
		{
			name:      "Invalid URL format",
			input:     "https://github.com/owner/repo/actions",
			shouldErr: true,
		},
		{
			name:      "Invalid string",
			input:     "not-a-number",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runID, owner, repo, hostname, err := parser.ParseRunURL(tt.input)

			if tt.shouldErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if runID != tt.expectedRunID {
					t.Errorf("Expected run ID %d, got %d", tt.expectedRunID, runID)
				}
				if owner != tt.expectedOwner {
					t.Errorf("Expected owner '%s', got '%s'", tt.expectedOwner, owner)
				}
				if repo != tt.expectedRepo {
					t.Errorf("Expected repo '%s', got '%s'", tt.expectedRepo, repo)
				}
				if hostname != tt.expectedHostname {
					t.Errorf("Expected hostname '%s', got '%s'", tt.expectedHostname, hostname)
				}
			}
		})
	}
}

func TestIsPermissionError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "Nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "Authentication required error",
			err:      errors.New("authentication required"),
			expected: true,
		},
		{
			name:     "Exit status 4 error",
			err:      errors.New("exit status 4"),
			expected: true,
		},
		{
			name:     "GitHub CLI authentication error",
			err:      errors.New("GitHub CLI authentication required"),
			expected: true,
		},
		{
			name:     "Permission denied error",
			err:      errors.New("permission denied"),
			expected: true,
		},
		{
			name:     "GH_TOKEN error",
			err:      errors.New("GH_TOKEN environment variable not set"),
			expected: true,
		},
		{
			name:     "Other error",
			err:      errors.New("some other error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPermissionError(tt.err)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestBuildAuditData(t *testing.T) {
	// Create test data
	run := WorkflowRun{
		DatabaseID:    123456,
		WorkflowName:  "Test Workflow",
		Status:        "completed",
		Conclusion:    "success",
		CreatedAt:     time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
		StartedAt:     time.Date(2024, 1, 1, 10, 0, 30, 0, time.UTC),
		UpdatedAt:     time.Date(2024, 1, 1, 10, 5, 0, 0, time.UTC),
		Duration:      4*time.Minute + 30*time.Second,
		Event:         "push",
		HeadBranch:    "main",
		URL:           "https://github.com/org/repo/actions/runs/123456",
		TokenUsage:    1500,
		EstimatedCost: 0.025,
		Turns:         5,
		ErrorCount:    2,
		WarningCount:  1,
		LogsPath:      testutil.TempDir(t, "test-*"),
	}

	metrics := LogMetrics{
		TokenUsage:    1500,
		EstimatedCost: 0.025,
		Turns:         5,
		ToolCalls: []workflow.ToolCallInfo{
			{
				Name:          "github_issue_read",
				CallCount:     3,
				MaxInputSize:  256,
				MaxOutputSize: 1024,
				MaxDuration:   2 * time.Second,
			},
		},
	}

	missingTools := []MissingToolReport{
		{
			Tool:         "missing_tool",
			Reason:       "Tool not available",
			Alternatives: "use alternative_tool instead",
			Timestamp:    "2024-01-01T10:00:00Z",
		},
	}

	mcpFailures := []MCPFailureReport{
		{
			ServerName: "test-server",
			Status:     "failed",
		},
	}

	processedRun := ProcessedRun{
		Run:          run,
		MissingTools: missingTools,
		MCPFailures:  mcpFailures,
	}

	// Build audit data
	auditData := buildAuditData(processedRun, metrics, nil)

	// Verify overview
	if auditData.Overview.RunID != 123456 {
		t.Errorf("Expected run ID 123456, got %d", auditData.Overview.RunID)
	}
	if auditData.Overview.WorkflowName != "Test Workflow" {
		t.Errorf("Expected workflow name 'Test Workflow', got %s", auditData.Overview.WorkflowName)
	}
	if auditData.Overview.Status != "completed" {
		t.Errorf("Expected status 'completed', got %s", auditData.Overview.Status)
	}
	// LogsPath should be converted to relative path
	if auditData.Overview.LogsPath == "" {
		t.Error("Expected logs path to be set")
	}
	// Verify that LogsPath is relative (doesn't start with /)
	if filepath.IsAbs(auditData.Overview.LogsPath) && auditData.Overview.LogsPath != run.LogsPath {
		t.Errorf("Expected logs path to be relative or match original, got '%s'", auditData.Overview.LogsPath)
	}

	// Verify metrics
	if auditData.Metrics.TokenUsage != 1500 {
		t.Errorf("Expected token usage 1500, got %d", auditData.Metrics.TokenUsage)
	}
	if auditData.Metrics.EstimatedCost != 0.025 {
		t.Errorf("Expected estimated cost 0.025, got %f", auditData.Metrics.EstimatedCost)
	}
	if auditData.Metrics.ErrorCount != 2 {
		t.Errorf("Expected error count 2, got %d", auditData.Metrics.ErrorCount)
	}
	if auditData.Metrics.WarningCount != 1 {
		t.Errorf("Expected warning count 1, got %d", auditData.Metrics.WarningCount)
	}

	// Note: Error and warning extraction was removed from buildAuditData
	// The error/warning counts in metrics are preserved but individual error/warning
	// extraction via pattern matching is no longer performed
	// if len(auditData.Errors) != 2 {
	// 	t.Errorf("Expected 2 errors, got %d", len(auditData.Errors))
	// }
	// if len(auditData.Warnings) != 1 {
	// 	t.Errorf("Expected 1 warning, got %d", len(auditData.Warnings))
	// }

	// Verify tool usage
	if len(auditData.ToolUsage) != 1 {
		t.Errorf("Expected 1 tool usage entry, got %d", len(auditData.ToolUsage))
	}

	// Verify missing tools
	if len(auditData.MissingTools) != 1 {
		t.Errorf("Expected 1 missing tool, got %d", len(auditData.MissingTools))
	}

	// Verify MCP failures
	if len(auditData.MCPFailures) != 1 {
		t.Errorf("Expected 1 MCP failure, got %d", len(auditData.MCPFailures))
	}
}

func TestDescribeFile(t *testing.T) {
	tests := []struct {
		filename    string
		description string
	}{
		{"aw_info.json", "Engine configuration and workflow metadata"},
		{"safe_output.jsonl", "Safe outputs from workflow execution"},
		{"agent_output.json", "Validated safe outputs"},
		{"aw.patch", "Git patch of changes made during execution"},
		{"agent-stdio.log", "Agent standard output/error logs"},
		{"log.md", "Human-readable agent session summary"},
		{"firewall.md", "Firewall log analysis report"},
		{"run_summary.json", "Cached summary of workflow run analysis"},
		{"prompt.txt", "Input prompt for AI agent"},
		{"random.log", "Log file"},
		{"unknown.txt", "Text file"},
		{"data.json", "JSON data file"},
		{"output.jsonl", "JSON Lines data file"},
		{"changes.patch", "Git patch file"},
		{"notes.md", "Markdown documentation"},
		{"agent_output", "Directory containing log files"},
		{"firewall-logs", "Directory containing log files"},
		{"squid-logs", "Directory containing log files"},
		{"aw-prompts", "Directory containing AI prompts"},
		{"somedir/", "Directory"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := describeFile(tt.filename)
			if result != tt.description {
				t.Errorf("Expected description '%s', got '%s'", tt.description, result)
			}
		})
	}
}

func TestRenderJSON(t *testing.T) {
	// Create test audit data
	auditData := AuditData{
		Overview: OverviewData{
			RunID:        123456,
			WorkflowName: "Test Workflow",
			Status:       "completed",
			Conclusion:   "success",
			Event:        "push",
			Branch:       "main",
			URL:          "https://github.com/org/repo/actions/runs/123456",
		},
		Metrics: MetricsData{
			TokenUsage:    1500,
			EstimatedCost: 0.025,
			Turns:         5,
			ErrorCount:    1,
			WarningCount:  1,
		},
		Jobs: []JobData{
			{
				Name:       "test-job",
				Status:     "completed",
				Conclusion: "success",
				Duration:   "2m30s",
			},
		},
		DownloadedFiles: []FileInfo{
			{
				Path:        "aw_info.json",
				Size:        1024,
				Description: "Engine configuration and workflow metadata",
			},
		},
		MissingTools: []MissingToolReport{
			{
				Tool:   "missing_tool",
				Reason: "Tool not available",
			},
		},
		Errors: []ErrorInfo{
			{
				File:    "agent.log",
				Line:    42,
				Type:    "error",
				Message: "Test error",
			},
		},
		Warnings: []ErrorInfo{
			{
				File:    "agent.log",
				Line:    50,
				Type:    "warning",
				Message: "Test warning",
			},
		},
	}

	// Render to JSON
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := renderJSON(auditData)
	w.Close()

	// Read the output
	var buf strings.Builder
	io.Copy(&buf, r)
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("renderJSON failed: %v", err)
	}

	jsonOutput := buf.String()

	// Verify it's valid JSON
	var parsed AuditData
	if err := json.Unmarshal([]byte(jsonOutput), &parsed); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Verify key fields
	if parsed.Overview.RunID != 123456 {
		t.Errorf("Expected run ID 123456, got %d", parsed.Overview.RunID)
	}
	if parsed.Metrics.TokenUsage != 1500 {
		t.Errorf("Expected token usage 1500, got %d", parsed.Metrics.TokenUsage)
	}
	if len(parsed.Jobs) != 1 {
		t.Errorf("Expected 1 job, got %d", len(parsed.Jobs))
	}
	if len(parsed.DownloadedFiles) != 1 {
		t.Errorf("Expected 1 downloaded file, got %d", len(parsed.DownloadedFiles))
	}
	if len(parsed.MissingTools) != 1 {
		t.Errorf("Expected 1 missing tool, got %d", len(parsed.MissingTools))
	}
	if len(parsed.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(parsed.Errors))
	}
	if len(parsed.Warnings) != 1 {
		t.Errorf("Expected 1 warning, got %d", len(parsed.Warnings))
	}
}

func TestAuditCachingBehavior(t *testing.T) {
	// Create a temporary directory for test artifacts
	tempDir := testutil.TempDir(t, "test-*")
	runOutputDir := filepath.Join(tempDir, "run-12345")
	if err := os.MkdirAll(runOutputDir, 0755); err != nil {
		t.Fatalf("Failed to create run directory: %v", err)
	}

	// Create minimal test artifacts
	awInfoPath := filepath.Join(runOutputDir, "aw_info.json")
	awInfoContent := `{"engine_id": "copilot", "workflow_name": "test-workflow"}`
	if err := os.WriteFile(awInfoPath, []byte(awInfoContent), 0644); err != nil {
		t.Fatalf("Failed to create mock aw_info.json: %v", err)
	}

	// Create a test run
	run := WorkflowRun{
		DatabaseID:    12345,
		WorkflowName:  "Test Workflow",
		Status:        "completed",
		Conclusion:    "success",
		CreatedAt:     time.Now(),
		Event:         "push",
		HeadBranch:    "main",
		URL:           "https://github.com/org/repo/actions/runs/12345",
		TokenUsage:    1000,
		EstimatedCost: 0.01,
		Turns:         3,
		ErrorCount:    0,
		WarningCount:  0,
		LogsPath:      runOutputDir,
	}

	metrics := LogMetrics{
		TokenUsage:    1000,
		EstimatedCost: 0.01,
		Turns:         3,
	}

	// Create and save a run summary
	summary := &RunSummary{
		CLIVersion:     GetVersion(),
		RunID:          run.DatabaseID,
		ProcessedAt:    time.Now(),
		Run:            run,
		Metrics:        metrics,
		AccessAnalysis: nil,
		MissingTools:   []MissingToolReport{},
		MCPFailures:    []MCPFailureReport{},
		ArtifactsList:  []string{"aw_info.json"},
		JobDetails:     []JobInfoWithDuration{},
	}

	if err := saveRunSummary(runOutputDir, summary, false); err != nil {
		t.Fatalf("Failed to save run summary: %v", err)
	}

	summaryPath := filepath.Join(runOutputDir, runSummaryFileName)

	// Verify summary file was created
	if _, err := os.Stat(summaryPath); os.IsNotExist(err) {
		t.Fatalf("Run summary file should exist after saveRunSummary")
	}

	// Load the summary back
	loadedSummary, ok := loadRunSummary(runOutputDir, false)
	if !ok {
		t.Fatalf("Failed to load run summary")
	}

	// Verify loaded summary matches
	if loadedSummary.RunID != summary.RunID {
		t.Errorf("Expected run ID %d, got %d", summary.RunID, loadedSummary.RunID)
	}
	if loadedSummary.CLIVersion != summary.CLIVersion {
		t.Errorf("Expected CLI version %s, got %s", summary.CLIVersion, loadedSummary.CLIVersion)
	}
	if loadedSummary.Run.WorkflowName != summary.Run.WorkflowName {
		t.Errorf("Expected workflow name %s, got %s", summary.Run.WorkflowName, loadedSummary.Run.WorkflowName)
	}

	// Verify that downloadRunArtifacts skips download when valid summary exists
	// This is tested by checking that the function returns without error
	// and doesn't attempt to call `gh run download`
	err := downloadRunArtifacts(run.DatabaseID, runOutputDir, false, "", "", "")
	if err != nil {
		t.Errorf("downloadRunArtifacts should skip download when valid summary exists, but got error: %v", err)
	}
}

// SKIPPED: Scripts now use require() pattern and are loaded at runtime from external files
func TestAuditParseFlagBehavior(t *testing.T) {
	t.Skip("Test skipped - log parser scripts now use require() pattern and are loaded at runtime from external files")
}

func TestBuildAuditDataWithFirewall(t *testing.T) {
	// Create test data with firewall analysis
	run := WorkflowRun{
		DatabaseID:    123456,
		WorkflowName:  "Test Workflow",
		Status:        "completed",
		Conclusion:    "success",
		CreatedAt:     time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
		Event:         "push",
		HeadBranch:    "main",
		URL:           "https://github.com/org/repo/actions/runs/123456",
		TokenUsage:    1500,
		EstimatedCost: 0.025,
		Turns:         5,
		ErrorCount:    0,
		WarningCount:  0,
		LogsPath:      testutil.TempDir(t, "test-*"),
	}

	metrics := LogMetrics{
		TokenUsage:    1500,
		EstimatedCost: 0.025,
		Turns:         5,
	}

	firewallAnalysis := &FirewallAnalysis{
		DomainBuckets: DomainBuckets{
			AllowedDomains: []string{"api.github.com:443", "npmjs.org:443"},
			BlockedDomains: []string{"blocked.example.com:443"},
		},
		TotalRequests:   10,
		AllowedRequests: 7,
		BlockedRequests: 3,
		RequestsByDomain: map[string]DomainRequestStats{
			"api.github.com:443":      {Allowed: 5, Blocked: 0},
			"npmjs.org:443":           {Allowed: 2, Blocked: 0},
			"blocked.example.com:443": {Allowed: 0, Blocked: 3},
		},
	}

	processedRun := ProcessedRun{
		Run:              run,
		FirewallAnalysis: firewallAnalysis,
		MissingTools:     []MissingToolReport{},
		MCPFailures:      []MCPFailureReport{},
	}

	// Build audit data
	auditData := buildAuditData(processedRun, metrics, nil)

	// Verify firewall analysis is included
	if auditData.FirewallAnalysis == nil {
		t.Fatal("Expected firewall analysis to be included in audit data")
	}

	// Verify firewall data is correct
	if auditData.FirewallAnalysis.TotalRequests != 10 {
		t.Errorf("Expected 10 total requests, got %d", auditData.FirewallAnalysis.TotalRequests)
	}
	if auditData.FirewallAnalysis.AllowedRequests != 7 {
		t.Errorf("Expected 7 allowed requests, got %d", auditData.FirewallAnalysis.AllowedRequests)
	}
	if auditData.FirewallAnalysis.BlockedRequests != 3 {
		t.Errorf("Expected 3 denied requests, got %d", auditData.FirewallAnalysis.BlockedRequests)
	}
	if len(auditData.FirewallAnalysis.AllowedDomains) != 2 {
		t.Errorf("Expected 2 allowed domains, got %d", len(auditData.FirewallAnalysis.AllowedDomains))
	}
	if len(auditData.FirewallAnalysis.BlockedDomains) != 1 {
		t.Errorf("Expected 1 blocked domain, got %d", len(auditData.FirewallAnalysis.BlockedDomains))
	}
}

func TestRenderJSONWithFirewall(t *testing.T) {
	// Create test audit data with firewall analysis
	firewallAnalysis := &FirewallAnalysis{
		DomainBuckets: DomainBuckets{
			AllowedDomains: []string{"api.github.com:443"},
			BlockedDomains: []string{"blocked.example.com:443"},
		},
		TotalRequests:   10,
		AllowedRequests: 7,
		BlockedRequests: 3,
		RequestsByDomain: map[string]DomainRequestStats{
			"api.github.com:443":      {Allowed: 7, Blocked: 0},
			"blocked.example.com:443": {Allowed: 0, Blocked: 3},
		},
	}

	auditData := AuditData{
		Overview: OverviewData{
			RunID:        123456,
			WorkflowName: "Test Workflow",
			Status:       "completed",
			Conclusion:   "success",
			Event:        "push",
			Branch:       "main",
			URL:          "https://github.com/org/repo/actions/runs/123456",
		},
		Metrics: MetricsData{
			TokenUsage:    1500,
			EstimatedCost: 0.025,
			Turns:         5,
			ErrorCount:    0,
			WarningCount:  0,
		},
		FirewallAnalysis: firewallAnalysis,
		DownloadedFiles:  []FileInfo{},
		MissingTools:     []MissingToolReport{},
		MCPFailures:      []MCPFailureReport{},
		Errors:           []ErrorInfo{},
		Warnings:         []ErrorInfo{},
		ToolUsage:        []ToolUsageInfo{},
	}

	// Render to JSON
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := renderJSON(auditData)
	w.Close()

	// Read the output
	var buf strings.Builder
	io.Copy(&buf, r)
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("renderJSON failed: %v", err)
	}

	jsonOutput := buf.String()

	// Verify it's valid JSON
	var parsed AuditData
	if err := json.Unmarshal([]byte(jsonOutput), &parsed); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Verify firewall analysis is included
	if parsed.FirewallAnalysis == nil {
		t.Fatal("Expected firewall analysis in JSON output")
	}

	// Verify firewall data is correct
	if parsed.FirewallAnalysis.TotalRequests != 10 {
		t.Errorf("Expected 10 total requests, got %d", parsed.FirewallAnalysis.TotalRequests)
	}
	if parsed.FirewallAnalysis.AllowedRequests != 7 {
		t.Errorf("Expected 7 allowed requests, got %d", parsed.FirewallAnalysis.AllowedRequests)
	}
	if parsed.FirewallAnalysis.BlockedRequests != 3 {
		t.Errorf("Expected 3 denied requests, got %d", parsed.FirewallAnalysis.BlockedRequests)
	}
	if len(parsed.FirewallAnalysis.AllowedDomains) != 1 {
		t.Errorf("Expected 1 allowed domain, got %d", len(parsed.FirewallAnalysis.AllowedDomains))
	}
	if len(parsed.FirewallAnalysis.BlockedDomains) != 1 {
		t.Errorf("Expected 1 blocked domain, got %d", len(parsed.FirewallAnalysis.BlockedDomains))
	}
}

func TestExtractStepOutput(t *testing.T) {
	jobLog := `##[group]Run actions/checkout@v4
Checking out repository...
##[endgroup]
##[group]Run ./setup-environment.sh
Setting up environment...
ENVIRONMENT=test
##[endgroup]
##[group]Run npm test
Running tests...
##[error]Test failed: expected 5, got 3
Error: Process completed with exit code 1.
##[endgroup]
##[group]Run cleanup.sh
Cleaning up...
##[endgroup]`

	tests := []struct {
		name        string
		stepNumber  int
		expectError bool
		checkOutput func(t *testing.T, output string)
	}{
		{
			name:        "Extract step 3 (failing step)",
			stepNumber:  3,
			expectError: false,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "npm test") {
					t.Error("Output should contain 'npm test'")
				}
				if !strings.Contains(output, "##[error]Test failed") {
					t.Error("Output should contain error message")
				}
			},
		},
		{
			name:        "Extract step 1",
			stepNumber:  1,
			expectError: false,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "actions/checkout") {
					t.Error("Output should contain 'actions/checkout'")
				}
			},
		},
		{
			name:        "Extract non-existent step",
			stepNumber:  99,
			expectError: true,
			checkOutput: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := extractStepOutput(jobLog, tt.stepNumber)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.checkOutput != nil {
				tt.checkOutput(t, output)
			}
		})
	}
}

func TestFindFirstFailingStep(t *testing.T) {
	tests := []struct {
		name            string
		jobLog          string
		expectedStepNum int
		checkOutput     func(t *testing.T, output string)
	}{
		{
			name: "Find failing step with error marker",
			jobLog: `##[group]Step 1
Success
##[endgroup]
##[group]Step 2
Running...
##[error]Something went wrong
Error details here
##[endgroup]
##[group]Step 3
This runs after failure
##[endgroup]`,
			expectedStepNum: 2,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "##[error]Something went wrong") {
					t.Error("Output should contain error message")
				}
			},
		},
		{
			name: "Find failing step with exit code",
			jobLog: `##[group]Step 1
Success
##[endgroup]
##[group]Step 2
Running command...
exit code 1
##[endgroup]`,
			expectedStepNum: 2,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "exit code 1") {
					t.Error("Output should contain exit code")
				}
			},
		},
		{
			name: "No failing steps",
			jobLog: `##[group]Step 1
Success
##[endgroup]
##[group]Step 2
Also success
##[endgroup]`,
			expectedStepNum: 0,
			checkOutput:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stepNum, output := findFirstFailingStep(tt.jobLog)

			if stepNum != tt.expectedStepNum {
				t.Errorf("Expected step number %d, got %d", tt.expectedStepNum, stepNum)
			}

			if tt.checkOutput != nil && stepNum > 0 {
				tt.checkOutput(t, output)
			}
		})
	}
}
