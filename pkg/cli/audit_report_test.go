package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/githubnext/gh-aw/pkg/testutil"
	"github.com/githubnext/gh-aw/pkg/workflow"
)

// createTestProcessedRun creates a test ProcessedRun with customizable parameters
func createTestProcessedRun(opts ...func(*ProcessedRun)) ProcessedRun {
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
		ErrorCount:    0,
		WarningCount:  0,
		LogsPath:      "/tmp/test-logs",
	}

	processedRun := ProcessedRun{
		Run: run,
	}

	for _, opt := range opts {
		opt(&processedRun)
	}

	return processedRun
}

func TestGenerateFindings(t *testing.T) {
	tests := []struct {
		name          string
		processedRun  ProcessedRun
		metrics       MetricsData
		errors        []ErrorInfo
		warnings      []ErrorInfo
		expectedCount int
		checkFindings func(t *testing.T, findings []Finding)
	}{
		{
			name: "successful workflow with no issues",
			processedRun: func() ProcessedRun {
				pr := createTestProcessedRun()
				pr.Run.Conclusion = "success"
				return pr
			}(),
			metrics: MetricsData{
				TokenUsage:    1000,
				EstimatedCost: 0.01,
				Turns:         3,
				ErrorCount:    0,
				WarningCount:  0,
			},
			errors:        []ErrorInfo{},
			warnings:      []ErrorInfo{},
			expectedCount: 1, // Should have success finding
			checkFindings: func(t *testing.T, findings []Finding) {
				found := false
				for _, f := range findings {
					if f.Category == "success" && f.Severity == "info" {
						found = true
						break
					}
				}
				if !found {
					t.Error("Expected success finding not found")
				}
			},
		},
		{
			name: "failed workflow",
			processedRun: func() ProcessedRun {
				pr := createTestProcessedRun()
				pr.Run.Conclusion = "failure"
				return pr
			}(),
			metrics: MetricsData{
				TokenUsage:    1000,
				EstimatedCost: 0.01,
				Turns:         3,
				ErrorCount:    2,
				WarningCount:  0,
			},
			errors:        []ErrorInfo{{Type: "error", Message: "Test error"}},
			warnings:      []ErrorInfo{},
			expectedCount: 1, // Should have failure finding
			checkFindings: func(t *testing.T, findings []Finding) {
				found := false
				for _, f := range findings {
					if f.Category == "error" && f.Severity == "critical" {
						found = true
						if !strings.Contains(f.Title, "Failed") {
							t.Errorf("Expected failure title, got: %s", f.Title)
						}
						break
					}
				}
				if !found {
					t.Error("Expected failure finding not found")
				}
			},
		},
		{
			name: "timed out workflow",
			processedRun: func() ProcessedRun {
				pr := createTestProcessedRun()
				pr.Run.Conclusion = "timed_out"
				return pr
			}(),
			metrics: MetricsData{
				Turns: 20,
			},
			errors:        []ErrorInfo{},
			warnings:      []ErrorInfo{},
			expectedCount: 1, // Timeout finding
			checkFindings: func(t *testing.T, findings []Finding) {
				found := false
				for _, f := range findings {
					if f.Category == "performance" && strings.Contains(f.Title, "Timeout") {
						found = true
						break
					}
				}
				if !found {
					t.Error("Expected timeout finding not found")
				}
			},
		},
		{
			name: "high cost workflow",
			processedRun: func() ProcessedRun {
				return createTestProcessedRun()
			}(),
			metrics: MetricsData{
				EstimatedCost: 1.50, // > 1.0 threshold
				Turns:         5,
			},
			errors:        []ErrorInfo{},
			warnings:      []ErrorInfo{},
			expectedCount: 1, // High cost finding
			checkFindings: func(t *testing.T, findings []Finding) {
				found := false
				for _, f := range findings {
					if f.Category == "cost" && f.Severity == "high" {
						found = true
						break
					}
				}
				if !found {
					t.Error("Expected high cost finding not found")
				}
			},
		},
		{
			name: "moderate cost workflow",
			processedRun: func() ProcessedRun {
				return createTestProcessedRun()
			}(),
			metrics: MetricsData{
				EstimatedCost: 0.75, // Between 0.5 and 1.0
				Turns:         5,
			},
			errors:        []ErrorInfo{},
			warnings:      []ErrorInfo{},
			expectedCount: 1, // Moderate cost finding
			checkFindings: func(t *testing.T, findings []Finding) {
				found := false
				for _, f := range findings {
					if f.Category == "cost" && f.Severity == "medium" {
						found = true
						break
					}
				}
				if !found {
					t.Error("Expected moderate cost finding not found")
				}
			},
		},
		{
			name: "high token usage",
			processedRun: func() ProcessedRun {
				return createTestProcessedRun()
			}(),
			metrics: MetricsData{
				TokenUsage: 60000, // > 50000 threshold
				Turns:      5,
			},
			errors:   []ErrorInfo{},
			warnings: []ErrorInfo{},
			checkFindings: func(t *testing.T, findings []Finding) {
				found := false
				for _, f := range findings {
					if f.Category == "performance" && strings.Contains(f.Title, "Token Usage") {
						found = true
						break
					}
				}
				if !found {
					t.Error("Expected high token usage finding not found")
				}
			},
		},
		{
			name: "many iterations",
			processedRun: func() ProcessedRun {
				return createTestProcessedRun()
			}(),
			metrics: MetricsData{
				Turns: 15, // > 10 threshold
			},
			errors:   []ErrorInfo{},
			warnings: []ErrorInfo{},
			checkFindings: func(t *testing.T, findings []Finding) {
				found := false
				for _, f := range findings {
					if f.Category == "performance" && strings.Contains(f.Title, "Iterations") {
						found = true
						break
					}
				}
				if !found {
					t.Error("Expected many iterations finding not found")
				}
			},
		},
		{
			name: "multiple errors",
			processedRun: func() ProcessedRun {
				return createTestProcessedRun()
			}(),
			metrics: MetricsData{
				Turns:      5,
				ErrorCount: 10,
			},
			errors: []ErrorInfo{
				{Type: "error", Message: "Error 1"},
				{Type: "error", Message: "Error 2"},
				{Type: "error", Message: "Error 3"},
				{Type: "error", Message: "Error 4"},
				{Type: "error", Message: "Error 5"},
				{Type: "error", Message: "Error 6"},
			},
			warnings: []ErrorInfo{},
			checkFindings: func(t *testing.T, findings []Finding) {
				found := false
				for _, f := range findings {
					if f.Category == "error" && strings.Contains(f.Title, "Multiple Errors") {
						found = true
						break
					}
				}
				if !found {
					t.Error("Expected multiple errors finding not found")
				}
			},
		},
		{
			name: "MCP server failures",
			processedRun: func() ProcessedRun {
				pr := createTestProcessedRun()
				pr.MCPFailures = []MCPFailureReport{
					{ServerName: "test-server", Status: "failed"},
				}
				return pr
			}(),
			metrics: MetricsData{
				Turns: 5,
			},
			errors:   []ErrorInfo{},
			warnings: []ErrorInfo{},
			checkFindings: func(t *testing.T, findings []Finding) {
				found := false
				for _, f := range findings {
					if f.Category == "tooling" && strings.Contains(f.Title, "MCP Server") {
						found = true
						break
					}
				}
				if !found {
					t.Error("Expected MCP server failures finding not found")
				}
			},
		},
		{
			name: "missing tools",
			processedRun: func() ProcessedRun {
				pr := createTestProcessedRun()
				pr.MissingTools = []MissingToolReport{
					{Tool: "tool1", Reason: "Not available"},
					{Tool: "tool2", Reason: "Not configured"},
				}
				return pr
			}(),
			metrics: MetricsData{
				Turns: 5,
			},
			errors:   []ErrorInfo{},
			warnings: []ErrorInfo{},
			checkFindings: func(t *testing.T, findings []Finding) {
				found := false
				for _, f := range findings {
					if f.Category == "tooling" && strings.Contains(f.Title, "Tools Not Available") {
						found = true
						break
					}
				}
				if !found {
					t.Error("Expected missing tools finding not found")
				}
			},
		},
		{
			name: "firewall blocked requests",
			processedRun: func() ProcessedRun {
				pr := createTestProcessedRun()
				pr.FirewallAnalysis = &FirewallAnalysis{
					TotalRequests:   10,
					BlockedRequests: 5,
					AllowedRequests: 5,
				}
				return pr
			}(),
			metrics: MetricsData{
				Turns: 5,
			},
			errors:   []ErrorInfo{},
			warnings: []ErrorInfo{},
			checkFindings: func(t *testing.T, findings []Finding) {
				found := false
				for _, f := range findings {
					if f.Category == "network" && strings.Contains(f.Title, "Blocked") {
						found = true
						break
					}
				}
				if !found {
					t.Error("Expected blocked network requests finding not found")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := generateFindings(tt.processedRun, tt.metrics, tt.errors, tt.warnings)

			if tt.expectedCount > 0 && len(findings) < tt.expectedCount {
				t.Errorf("Expected at least %d findings, got %d", tt.expectedCount, len(findings))
			}

			if tt.checkFindings != nil {
				tt.checkFindings(t, findings)
			}
		})
	}
}

func TestGenerateRecommendations(t *testing.T) {
	tests := []struct {
		name                 string
		processedRun         ProcessedRun
		metrics              MetricsData
		findings             []Finding
		expectedMinCount     int
		checkRecommendations func(t *testing.T, recs []Recommendation)
	}{
		{
			name: "failed workflow generates review recommendation",
			processedRun: func() ProcessedRun {
				pr := createTestProcessedRun()
				pr.Run.Conclusion = "failure"
				return pr
			}(),
			metrics:          MetricsData{},
			findings:         []Finding{},
			expectedMinCount: 1,
			checkRecommendations: func(t *testing.T, recs []Recommendation) {
				found := false
				for _, r := range recs {
					if strings.Contains(r.Action, "error logs") && r.Priority == "high" {
						found = true
						break
					}
				}
				if !found {
					t.Error("Expected high priority review recommendation for failed workflow")
				}
			},
		},
		{
			name:         "critical findings generate review recommendation",
			processedRun: createTestProcessedRun(),
			metrics:      MetricsData{},
			findings: []Finding{
				{Category: "error", Severity: "critical", Title: "Test Critical"},
			},
			expectedMinCount: 1,
			checkRecommendations: func(t *testing.T, recs []Recommendation) {
				found := false
				for _, r := range recs {
					if strings.Contains(r.Action, "error logs") && r.Priority == "high" {
						found = true
						break
					}
				}
				if !found {
					t.Error("Expected high priority review recommendation for critical findings")
				}
			},
		},
		{
			name:         "high cost findings generate optimization recommendation",
			processedRun: createTestProcessedRun(),
			metrics:      MetricsData{EstimatedCost: 1.5},
			findings: []Finding{
				{Category: "cost", Severity: "high", Title: "High Cost"},
			},
			expectedMinCount: 1,
			checkRecommendations: func(t *testing.T, recs []Recommendation) {
				found := false
				for _, r := range recs {
					if strings.Contains(r.Action, "Optimize") || strings.Contains(r.Action, "prompt") {
						found = true
						break
					}
				}
				if !found {
					t.Error("Expected optimization recommendation for high cost findings")
				}
			},
		},
		{
			name: "missing tools generate add tools recommendation",
			processedRun: func() ProcessedRun {
				pr := createTestProcessedRun()
				pr.MissingTools = []MissingToolReport{
					{Tool: "missing_tool", Reason: "Not available"},
				}
				return pr
			}(),
			metrics:          MetricsData{},
			findings:         []Finding{},
			expectedMinCount: 1,
			checkRecommendations: func(t *testing.T, recs []Recommendation) {
				found := false
				for _, r := range recs {
					if strings.Contains(r.Action, "missing tools") || strings.Contains(r.Action, "Add") {
						found = true
						break
					}
				}
				if !found {
					t.Error("Expected add tools recommendation for missing tools")
				}
			},
		},
		{
			name: "MCP failures generate fix recommendation",
			processedRun: func() ProcessedRun {
				pr := createTestProcessedRun()
				pr.MCPFailures = []MCPFailureReport{
					{ServerName: "test-server", Status: "failed"},
				}
				return pr
			}(),
			metrics:          MetricsData{},
			findings:         []Finding{},
			expectedMinCount: 1,
			checkRecommendations: func(t *testing.T, recs []Recommendation) {
				found := false
				for _, r := range recs {
					if strings.Contains(r.Action, "MCP") || strings.Contains(r.Action, "server") {
						found = true
						break
					}
				}
				if !found {
					t.Error("Expected MCP fix recommendation for MCP failures")
				}
			},
		},
		{
			name: "many firewall blocks generate network review recommendation",
			processedRun: func() ProcessedRun {
				pr := createTestProcessedRun()
				pr.FirewallAnalysis = &FirewallAnalysis{
					BlockedRequests: 15, // > 10 threshold
				}
				return pr
			}(),
			metrics:          MetricsData{},
			findings:         []Finding{},
			expectedMinCount: 1,
			checkRecommendations: func(t *testing.T, recs []Recommendation) {
				found := false
				for _, r := range recs {
					if strings.Contains(r.Action, "network") || strings.Contains(r.Action, "Review") {
						found = true
						break
					}
				}
				if !found {
					t.Error("Expected network review recommendation for firewall blocks")
				}
			},
		},
		{
			name: "successful workflow with no issues gets monitoring recommendation",
			processedRun: func() ProcessedRun {
				pr := createTestProcessedRun()
				pr.Run.Conclusion = "success"
				return pr
			}(),
			metrics:          MetricsData{},
			findings:         []Finding{},
			expectedMinCount: 1,
			checkRecommendations: func(t *testing.T, recs []Recommendation) {
				found := false
				for _, r := range recs {
					if r.Priority == "low" && strings.Contains(r.Action, "Monitor") {
						found = true
						break
					}
				}
				if !found {
					t.Error("Expected low priority monitoring recommendation for successful workflow")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recs := generateRecommendations(tt.processedRun, tt.metrics, tt.findings)

			if len(recs) < tt.expectedMinCount {
				t.Errorf("Expected at least %d recommendations, got %d", tt.expectedMinCount, len(recs))
			}

			if tt.checkRecommendations != nil {
				tt.checkRecommendations(t, recs)
			}
		})
	}
}

func TestGenerateFailureAnalysis(t *testing.T) {
	tests := []struct {
		name          string
		processedRun  ProcessedRun
		errors        []ErrorInfo
		checkAnalysis func(t *testing.T, analysis *FailureAnalysis)
	}{
		{
			name: "basic failure analysis",
			processedRun: func() ProcessedRun {
				pr := createTestProcessedRun()
				pr.Run.Conclusion = "failure"
				return pr
			}(),
			errors: []ErrorInfo{},
			checkAnalysis: func(t *testing.T, analysis *FailureAnalysis) {
				if analysis.PrimaryFailure != "failure" {
					t.Errorf("Expected primary failure 'failure', got %s", analysis.PrimaryFailure)
				}
			},
		},
		{
			name: "failure with failed jobs",
			processedRun: func() ProcessedRun {
				pr := createTestProcessedRun()
				pr.Run.Conclusion = "failure"
				pr.JobDetails = []JobInfoWithDuration{
					{JobInfo: JobInfo{Name: "build", Conclusion: "success"}},
					{JobInfo: JobInfo{Name: "test", Conclusion: "failure"}},
					{JobInfo: JobInfo{Name: "deploy", Conclusion: "cancelled"}},
				}
				return pr
			}(),
			errors: []ErrorInfo{},
			checkAnalysis: func(t *testing.T, analysis *FailureAnalysis) {
				if len(analysis.FailedJobs) != 2 {
					t.Errorf("Expected 2 failed jobs, got %d", len(analysis.FailedJobs))
				}
			},
		},
		{
			name: "failure with single error",
			processedRun: func() ProcessedRun {
				pr := createTestProcessedRun()
				pr.Run.Conclusion = "failure"
				return pr
			}(),
			errors: []ErrorInfo{
				{Type: "error", Message: "Build failed"},
			},
			checkAnalysis: func(t *testing.T, analysis *FailureAnalysis) {
				if analysis.ErrorSummary != "Build failed" {
					t.Errorf("Expected single error message in summary, got: %s", analysis.ErrorSummary)
				}
			},
		},
		{
			name: "failure with multiple errors",
			processedRun: func() ProcessedRun {
				pr := createTestProcessedRun()
				pr.Run.Conclusion = "failure"
				return pr
			}(),
			errors: []ErrorInfo{
				{Type: "error", Message: "First error"},
				{Type: "error", Message: "Second error"},
				{Type: "error", Message: "Third error"},
			},
			checkAnalysis: func(t *testing.T, analysis *FailureAnalysis) {
				if !strings.Contains(analysis.ErrorSummary, "3 errors") {
					t.Errorf("Expected '3 errors' in summary, got: %s", analysis.ErrorSummary)
				}
				if !strings.Contains(analysis.ErrorSummary, "First error") {
					t.Errorf("Expected first error in summary, got: %s", analysis.ErrorSummary)
				}
			},
		},
		{
			name: "failure with MCP server failure root cause",
			processedRun: func() ProcessedRun {
				pr := createTestProcessedRun()
				pr.Run.Conclusion = "failure"
				pr.MCPFailures = []MCPFailureReport{
					{ServerName: "github-mcp", Status: "failed"},
				}
				return pr
			}(),
			errors: []ErrorInfo{},
			checkAnalysis: func(t *testing.T, analysis *FailureAnalysis) {
				if !strings.Contains(analysis.RootCause, "MCP server failure") {
					t.Errorf("Expected MCP server failure root cause, got: %s", analysis.RootCause)
				}
			},
		},
		{
			name: "failure with timeout error pattern",
			processedRun: func() ProcessedRun {
				pr := createTestProcessedRun()
				pr.Run.Conclusion = "failure"
				return pr
			}(),
			errors: []ErrorInfo{
				{Type: "error", Message: "Connection timeout after 30s"},
			},
			checkAnalysis: func(t *testing.T, analysis *FailureAnalysis) {
				if analysis.RootCause != "Operation timeout" {
					t.Errorf("Expected 'Operation timeout' root cause, got: %s", analysis.RootCause)
				}
			},
		},
		{
			name: "failure with permission error pattern",
			processedRun: func() ProcessedRun {
				pr := createTestProcessedRun()
				pr.Run.Conclusion = "failure"
				return pr
			}(),
			errors: []ErrorInfo{
				{Type: "error", Message: "Permission blocked: cannot access file"},
			},
			checkAnalysis: func(t *testing.T, analysis *FailureAnalysis) {
				if analysis.RootCause != "Permission denied" {
					t.Errorf("Expected 'Permission denied' root cause, got: %s", analysis.RootCause)
				}
			},
		},
		{
			name: "failure with not found error pattern",
			processedRun: func() ProcessedRun {
				pr := createTestProcessedRun()
				pr.Run.Conclusion = "failure"
				return pr
			}(),
			errors: []ErrorInfo{
				{Type: "error", Message: "File not found: test.txt"},
			},
			checkAnalysis: func(t *testing.T, analysis *FailureAnalysis) {
				if analysis.RootCause != "Resource not found" {
					t.Errorf("Expected 'Resource not found' root cause, got: %s", analysis.RootCause)
				}
			},
		},
		{
			name: "failure with authentication error pattern",
			processedRun: func() ProcessedRun {
				pr := createTestProcessedRun()
				pr.Run.Conclusion = "failure"
				return pr
			}(),
			errors: []ErrorInfo{
				{Type: "error", Message: "Authentication failed for user"},
			},
			checkAnalysis: func(t *testing.T, analysis *FailureAnalysis) {
				if analysis.RootCause != "Authentication failure" {
					t.Errorf("Expected 'Authentication failure' root cause, got: %s", analysis.RootCause)
				}
			},
		},
		{
			name: "unknown failure with no errors",
			processedRun: func() ProcessedRun {
				pr := createTestProcessedRun()
				pr.Run.Conclusion = ""
				return pr
			}(),
			errors: []ErrorInfo{},
			checkAnalysis: func(t *testing.T, analysis *FailureAnalysis) {
				if analysis.PrimaryFailure != "unknown" {
					t.Errorf("Expected 'unknown' primary failure, got: %s", analysis.PrimaryFailure)
				}
				if !strings.Contains(analysis.ErrorSummary, "No specific errors") {
					t.Errorf("Expected 'No specific errors' in summary, got: %s", analysis.ErrorSummary)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysis := generateFailureAnalysis(tt.processedRun, tt.errors)

			if analysis == nil {
				t.Fatal("Expected non-nil analysis")
			}

			if tt.checkAnalysis != nil {
				tt.checkAnalysis(t, analysis)
			}
		})
	}
}

func TestGeneratePerformanceMetrics(t *testing.T) {
	tests := []struct {
		name         string
		processedRun ProcessedRun
		metrics      MetricsData
		toolUsage    []ToolUsageInfo
		checkMetrics func(t *testing.T, pm *PerformanceMetrics)
	}{
		{
			name: "tokens per minute calculation",
			processedRun: func() ProcessedRun {
				pr := createTestProcessedRun()
				pr.Run.Duration = 2 * time.Minute
				return pr
			}(),
			metrics: MetricsData{
				TokenUsage: 1000,
			},
			toolUsage: []ToolUsageInfo{},
			checkMetrics: func(t *testing.T, pm *PerformanceMetrics) {
				if pm.TokensPerMinute != 500 {
					t.Errorf("Expected 500 tokens/minute, got %f", pm.TokensPerMinute)
				}
			},
		},
		{
			name: "cost efficiency - excellent",
			processedRun: func() ProcessedRun {
				pr := createTestProcessedRun()
				pr.Run.Duration = 10 * time.Minute
				return pr
			}(),
			metrics: MetricsData{
				EstimatedCost: 0.05, // $0.005/min < $0.01 threshold
			},
			toolUsage: []ToolUsageInfo{},
			checkMetrics: func(t *testing.T, pm *PerformanceMetrics) {
				if pm.CostEfficiency != "excellent" {
					t.Errorf("Expected 'excellent' cost efficiency, got %s", pm.CostEfficiency)
				}
			},
		},
		{
			name: "cost efficiency - good",
			processedRun: func() ProcessedRun {
				pr := createTestProcessedRun()
				pr.Run.Duration = 10 * time.Minute
				return pr
			}(),
			metrics: MetricsData{
				EstimatedCost: 0.25, // $0.025/min
			},
			toolUsage: []ToolUsageInfo{},
			checkMetrics: func(t *testing.T, pm *PerformanceMetrics) {
				if pm.CostEfficiency != "good" {
					t.Errorf("Expected 'good' cost efficiency, got %s", pm.CostEfficiency)
				}
			},
		},
		{
			name: "cost efficiency - moderate",
			processedRun: func() ProcessedRun {
				pr := createTestProcessedRun()
				pr.Run.Duration = 10 * time.Minute
				return pr
			}(),
			metrics: MetricsData{
				EstimatedCost: 0.75, // $0.075/min
			},
			toolUsage: []ToolUsageInfo{},
			checkMetrics: func(t *testing.T, pm *PerformanceMetrics) {
				if pm.CostEfficiency != "moderate" {
					t.Errorf("Expected 'moderate' cost efficiency, got %s", pm.CostEfficiency)
				}
			},
		},
		{
			name: "cost efficiency - poor",
			processedRun: func() ProcessedRun {
				pr := createTestProcessedRun()
				pr.Run.Duration = 10 * time.Minute
				return pr
			}(),
			metrics: MetricsData{
				EstimatedCost: 1.50, // $0.15/min
			},
			toolUsage: []ToolUsageInfo{},
			checkMetrics: func(t *testing.T, pm *PerformanceMetrics) {
				if pm.CostEfficiency != "poor" {
					t.Errorf("Expected 'poor' cost efficiency, got %s", pm.CostEfficiency)
				}
			},
		},
		{
			name:         "most used tool",
			processedRun: createTestProcessedRun(),
			metrics:      MetricsData{},
			toolUsage: []ToolUsageInfo{
				{Name: "bash", CallCount: 5},
				{Name: "github_issue_read", CallCount: 10},
				{Name: "file_edit", CallCount: 3},
			},
			checkMetrics: func(t *testing.T, pm *PerformanceMetrics) {
				if !strings.Contains(pm.MostUsedTool, "github_issue_read") {
					t.Errorf("Expected 'github_issue_read' as most used tool, got %s", pm.MostUsedTool)
				}
				if !strings.Contains(pm.MostUsedTool, "10 calls") {
					t.Errorf("Expected '10 calls' in most used tool, got %s", pm.MostUsedTool)
				}
			},
		},
		{
			name:         "average tool duration",
			processedRun: createTestProcessedRun(),
			metrics:      MetricsData{},
			toolUsage: []ToolUsageInfo{
				{Name: "bash", CallCount: 5, MaxDuration: "1s"},
				{Name: "github_issue_read", CallCount: 10, MaxDuration: "3s"},
			},
			checkMetrics: func(t *testing.T, pm *PerformanceMetrics) {
				if pm.AvgToolDuration == "" {
					t.Error("Expected non-empty average tool duration")
				}
			},
		},
		{
			name: "network requests from firewall",
			processedRun: func() ProcessedRun {
				pr := createTestProcessedRun()
				pr.FirewallAnalysis = &FirewallAnalysis{
					TotalRequests: 50,
				}
				return pr
			}(),
			metrics:   MetricsData{},
			toolUsage: []ToolUsageInfo{},
			checkMetrics: func(t *testing.T, pm *PerformanceMetrics) {
				if pm.NetworkRequests != 50 {
					t.Errorf("Expected 50 network requests, got %d", pm.NetworkRequests)
				}
			},
		},
		{
			name: "zero duration doesn't calculate tokens per minute",
			processedRun: func() ProcessedRun {
				pr := createTestProcessedRun()
				pr.Run.Duration = 0
				return pr
			}(),
			metrics: MetricsData{
				TokenUsage: 1000,
			},
			toolUsage: []ToolUsageInfo{},
			checkMetrics: func(t *testing.T, pm *PerformanceMetrics) {
				if pm.TokensPerMinute != 0 {
					t.Errorf("Expected 0 tokens/minute for zero duration, got %f", pm.TokensPerMinute)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := generatePerformanceMetrics(tt.processedRun, tt.metrics, tt.toolUsage)

			if pm == nil {
				t.Fatal("Expected non-nil performance metrics")
			}

			if tt.checkMetrics != nil {
				tt.checkMetrics(t, pm)
			}
		})
	}
}

func TestBuildAuditDataComplete(t *testing.T) {
	// Create a comprehensive test with all data filled in
	tmpDir := testutil.TempDir(t, "audit-data-test-*")

	// Create test files in the temp directory
	testFiles := map[string]string{
		"aw_info.json": `{"engine":"copilot"}`,
		"output.log":   "Test log content",
	}
	for filename, content := range testFiles {
		err := os.WriteFile(tmpDir+"/"+filename, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	processedRun := ProcessedRun{
		Run: WorkflowRun{
			DatabaseID:    12345,
			WorkflowName:  "Complete Test Workflow",
			Status:        "completed",
			Conclusion:    "failure",
			CreatedAt:     time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
			StartedAt:     time.Date(2024, 1, 1, 10, 0, 30, 0, time.UTC),
			UpdatedAt:     time.Date(2024, 1, 1, 10, 10, 0, 0, time.UTC),
			Duration:      9*time.Minute + 30*time.Second,
			Event:         "pull_request",
			HeadBranch:    "feature-branch",
			URL:           "https://github.com/test/repo/actions/runs/12345",
			TokenUsage:    25000,
			EstimatedCost: 0.75,
			Turns:         8,
			ErrorCount:    3,
			WarningCount:  2,
			LogsPath:      tmpDir,
		},
		JobDetails: []JobInfoWithDuration{
			{JobInfo: JobInfo{Name: "build", Status: "completed", Conclusion: "success"}, Duration: 2 * time.Minute},
			{JobInfo: JobInfo{Name: "test", Status: "completed", Conclusion: "failure"}, Duration: 5 * time.Minute},
		},
		MissingTools: []MissingToolReport{
			{Tool: "special_tool", Reason: "Not configured"},
		},
		MCPFailures: []MCPFailureReport{
			{ServerName: "test-mcp", Status: "connection_error"},
		},
		FirewallAnalysis: &FirewallAnalysis{
			DomainBuckets: DomainBuckets{
				AllowedDomains: []string{"api.github.com"},
				BlockedDomains: []string{"blocked.example.com"},
			},
			TotalRequests:   15,
			AllowedRequests: 10,
			BlockedRequests: 5,
			RequestsByDomain: map[string]DomainRequestStats{
				"api.github.com":      {Allowed: 10, Blocked: 0},
				"blocked.example.com": {Allowed: 0, Blocked: 5},
			},
		},
		RedactedDomainsAnalysis: &RedactedDomainsAnalysis{
			TotalDomains: 2,
			Domains:      []string{"secret.example.com", "internal.test.com"},
		},
	}

	metrics := workflow.LogMetrics{
		TokenUsage:    25000,
		EstimatedCost: 0.75,
		Turns:         8,
		ToolCalls: []workflow.ToolCallInfo{
			{Name: "bash", CallCount: 15, MaxInputSize: 500, MaxOutputSize: 2000, MaxDuration: 5 * time.Second},
			{Name: "github_issue_read", CallCount: 8, MaxInputSize: 100, MaxOutputSize: 5000, MaxDuration: 2 * time.Second},
		},
	}

	// Build audit data
	auditData := buildAuditData(processedRun, metrics)

	// Verify overview
	t.Run("Overview", func(t *testing.T) {
		if auditData.Overview.RunID != 12345 {
			t.Errorf("Expected RunID 12345, got %d", auditData.Overview.RunID)
		}
		if auditData.Overview.WorkflowName != "Complete Test Workflow" {
			t.Errorf("Expected workflow name 'Complete Test Workflow', got %s", auditData.Overview.WorkflowName)
		}
		if auditData.Overview.Status != "completed" {
			t.Errorf("Expected status 'completed', got %s", auditData.Overview.Status)
		}
		if auditData.Overview.Conclusion != "failure" {
			t.Errorf("Expected conclusion 'failure', got %s", auditData.Overview.Conclusion)
		}
	})

	// Verify metrics
	t.Run("Metrics", func(t *testing.T) {
		if auditData.Metrics.TokenUsage != 25000 {
			t.Errorf("Expected token usage 25000, got %d", auditData.Metrics.TokenUsage)
		}
		if auditData.Metrics.ErrorCount != 3 {
			t.Errorf("Expected error count 3, got %d", auditData.Metrics.ErrorCount)
		}
		if auditData.Metrics.WarningCount != 2 {
			t.Errorf("Expected warning count 2, got %d", auditData.Metrics.WarningCount)
		}
	})

	// Verify jobs
	t.Run("Jobs", func(t *testing.T) {
		if len(auditData.Jobs) != 2 {
			t.Errorf("Expected 2 jobs, got %d", len(auditData.Jobs))
		}
	})

	// Verify tool usage
	t.Run("ToolUsage", func(t *testing.T) {
		if len(auditData.ToolUsage) != 2 {
			t.Errorf("Expected 2 tool usage entries, got %d", len(auditData.ToolUsage))
		}
	})

	// Verify findings are generated
	t.Run("Findings", func(t *testing.T) {
		if len(auditData.KeyFindings) == 0 {
			t.Error("Expected at least one finding")
		}
		// Should have failure finding since conclusion is "failure"
		hasFailureFinding := false
		for _, f := range auditData.KeyFindings {
			if f.Severity == "critical" && f.Category == "error" {
				hasFailureFinding = true
				break
			}
		}
		if !hasFailureFinding {
			t.Error("Expected failure finding for failed workflow")
		}
	})

	// Verify recommendations are generated
	t.Run("Recommendations", func(t *testing.T) {
		if len(auditData.Recommendations) == 0 {
			t.Error("Expected at least one recommendation")
		}
	})

	// Verify failure analysis is generated
	t.Run("FailureAnalysis", func(t *testing.T) {
		if auditData.FailureAnalysis == nil {
			t.Error("Expected failure analysis for failed workflow")
		}
	})

	// Verify performance metrics are generated
	t.Run("PerformanceMetrics", func(t *testing.T) {
		if auditData.PerformanceMetrics == nil {
			t.Error("Expected performance metrics")
		}
	})

	// Verify firewall analysis is passed through
	t.Run("FirewallAnalysis", func(t *testing.T) {
		if auditData.FirewallAnalysis == nil {
			t.Error("Expected firewall analysis")
		}
		if auditData.FirewallAnalysis.TotalRequests != 15 {
			t.Errorf("Expected 15 total requests, got %d", auditData.FirewallAnalysis.TotalRequests)
		}
	})

	// Verify redacted domains are passed through
	t.Run("RedactedDomainsAnalysis", func(t *testing.T) {
		if auditData.RedactedDomainsAnalysis == nil {
			t.Error("Expected redacted domains analysis")
		}
		if auditData.RedactedDomainsAnalysis.TotalDomains != 2 {
			t.Errorf("Expected 2 redacted domains, got %d", auditData.RedactedDomainsAnalysis.TotalDomains)
		}
	})

	// Verify MCP failures are passed through
	t.Run("MCPFailures", func(t *testing.T) {
		if len(auditData.MCPFailures) != 1 {
			t.Errorf("Expected 1 MCP failure, got %d", len(auditData.MCPFailures))
		}
	})

	// Verify missing tools are passed through
	t.Run("MissingTools", func(t *testing.T) {
		if len(auditData.MissingTools) != 1 {
			t.Errorf("Expected 1 missing tool, got %d", len(auditData.MissingTools))
		}
	})
}

func TestBuildAuditDataMinimal(t *testing.T) {
	// Test with minimal/empty data
	processedRun := ProcessedRun{
		Run: WorkflowRun{
			DatabaseID:   1,
			WorkflowName: "Minimal",
			Status:       "completed",
			Conclusion:   "success",
			LogsPath:     "/nonexistent",
		},
	}

	metrics := workflow.LogMetrics{}

	auditData := buildAuditData(processedRun, metrics)

	// Should still produce valid data
	if auditData.Overview.RunID != 1 {
		t.Errorf("Expected RunID 1, got %d", auditData.Overview.RunID)
	}

	// Empty slices should be nil or empty, not cause panics
	// We just want to ensure no panics occur accessing these fields
	_ = auditData.Jobs
}

func TestRenderJSONComplete(t *testing.T) {
	auditData := AuditData{
		Overview: OverviewData{
			RunID:        99999,
			WorkflowName: "JSON Test",
			Status:       "completed",
			Conclusion:   "success",
			Event:        "push",
			Branch:       "main",
			URL:          "https://github.com/test/repo/actions/runs/99999",
		},
		Metrics: MetricsData{
			TokenUsage:    5000,
			EstimatedCost: 0.10,
			Turns:         4,
			ErrorCount:    1,
			WarningCount:  2,
		},
		KeyFindings: []Finding{
			{Category: "success", Severity: "info", Title: "Test Finding", Description: "Test description"},
		},
		Recommendations: []Recommendation{
			{Priority: "low", Action: "Monitor", Reason: "Test reason"},
		},
		Jobs: []JobData{
			{Name: "test-job", Status: "completed", Conclusion: "success", Duration: "1m30s"},
		},
		DownloadedFiles: []FileInfo{
			{Path: "test.log", Size: 1024, Description: "Test log"},
		},
		Errors: []ErrorInfo{
			{Type: "error", Message: "Test error"},
		},
		Warnings: []ErrorInfo{
			{Type: "warning", Message: "Test warning 1"},
			{Type: "warning", Message: "Test warning 2"},
		},
		ToolUsage: []ToolUsageInfo{
			{Name: "bash", CallCount: 5, MaxInputSize: 100, MaxOutputSize: 500},
		},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := renderJSON(auditData)
	w.Close()

	// Read output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("renderJSON failed: %v", err)
	}

	jsonOutput := buf.String()

	// Verify valid JSON
	var parsed AuditData
	if err := json.Unmarshal([]byte(jsonOutput), &parsed); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Verify all fields
	if parsed.Overview.RunID != 99999 {
		t.Errorf("Expected RunID 99999, got %d", parsed.Overview.RunID)
	}
	if len(parsed.KeyFindings) != 1 {
		t.Errorf("Expected 1 finding, got %d", len(parsed.KeyFindings))
	}
	if len(parsed.Recommendations) != 1 {
		t.Errorf("Expected 1 recommendation, got %d", len(parsed.Recommendations))
	}
	if len(parsed.Jobs) != 1 {
		t.Errorf("Expected 1 job, got %d", len(parsed.Jobs))
	}
	if len(parsed.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(parsed.Errors))
	}
	if len(parsed.Warnings) != 2 {
		t.Errorf("Expected 2 warnings, got %d", len(parsed.Warnings))
	}
}

func TestToolUsageAggregation(t *testing.T) {
	// Test that tool usage is properly aggregated with prettified names
	processedRun := ProcessedRun{
		Run: WorkflowRun{
			DatabaseID:   1,
			WorkflowName: "Tool Test",
			Status:       "completed",
			Conclusion:   "success",
			LogsPath:     "/tmp/test",
		},
	}

	// Simulate multiple calls to the same tool with different raw names
	metrics := workflow.LogMetrics{
		ToolCalls: []workflow.ToolCallInfo{
			{Name: "github_mcp_server_issue_read", CallCount: 5, MaxInputSize: 100, MaxOutputSize: 500, MaxDuration: 2 * time.Second},
			{Name: "github_mcp_server_issue_read", CallCount: 3, MaxInputSize: 200, MaxOutputSize: 800, MaxDuration: 3 * time.Second},
			{Name: "bash", CallCount: 10, MaxInputSize: 50, MaxOutputSize: 100, MaxDuration: 1 * time.Second},
		},
	}

	auditData := buildAuditData(processedRun, metrics)

	// Tool usage should be aggregated
	// The exact aggregation depends on workflow.PrettifyToolName behavior
	if len(auditData.ToolUsage) == 0 {
		t.Error("Expected tool usage data")
	}

	// Check that bash is present
	bashFound := false
	for _, tool := range auditData.ToolUsage {
		if strings.Contains(strings.ToLower(tool.Name), "bash") {
			bashFound = true
			if tool.CallCount != 10 {
				t.Errorf("Expected bash call count 10, got %d", tool.CallCount)
			}
		}
	}
	if !bashFound {
		t.Error("Expected bash in tool usage")
	}
}

func TestExtractDownloadedFilesEmpty(t *testing.T) {
	// Test with nonexistent directory
	files := extractDownloadedFiles("/nonexistent/path")
	if len(files) != 0 {
		t.Errorf("Expected empty files for nonexistent path, got %d files", len(files))
	}

	// Test with empty directory
	tmpDir := testutil.TempDir(t, "empty-dir-*")
	files = extractDownloadedFiles(tmpDir)
	if len(files) != 0 {
		t.Errorf("Expected empty files for empty directory, got %d files", len(files))
	}
}

func TestFindingSeverityOrdering(t *testing.T) {
	// Test that findings are generated with proper severity levels
	processedRun := ProcessedRun{
		Run: WorkflowRun{
			DatabaseID:   1,
			WorkflowName: "Severity Test",
			Status:       "completed",
			Conclusion:   "failure",
			Duration:     5 * time.Minute,
		},
		MCPFailures: []MCPFailureReport{
			{ServerName: "test-mcp", Status: "failed"},
		},
	}

	metrics := MetricsData{
		ErrorCount:    5,
		EstimatedCost: 2.0, // High cost
		Turns:         15,  // Many turns
	}

	errors := []ErrorInfo{
		{Type: "error", Message: "Error 1"},
		{Type: "error", Message: "Error 2"},
		{Type: "error", Message: "Error 3"},
		{Type: "error", Message: "Error 4"},
		{Type: "error", Message: "Error 5"},
		{Type: "error", Message: "Error 6"},
	}

	findings := generateFindings(processedRun, metrics, errors, []ErrorInfo{})

	// Should have critical, high, and medium findings
	severityCounts := make(map[string]int)
	for _, f := range findings {
		severityCounts[f.Severity]++
	}

	if severityCounts["critical"] == 0 {
		t.Error("Expected at least one critical finding for failed workflow")
	}
	if severityCounts["high"] == 0 {
		t.Error("Expected at least one high severity finding")
	}
}

func TestRecommendationPriorityOrdering(t *testing.T) {
	// Test that recommendations are generated with proper priorities
	processedRun := ProcessedRun{
		Run: WorkflowRun{
			DatabaseID:   1,
			WorkflowName: "Priority Test",
			Status:       "completed",
			Conclusion:   "failure",
		},
		MCPFailures: []MCPFailureReport{
			{ServerName: "test-mcp", Status: "failed"},
		},
		MissingTools: []MissingToolReport{
			{Tool: "missing", Reason: "Not available"},
		},
		FirewallAnalysis: &FirewallAnalysis{
			BlockedRequests: 20, // Many blocked requests
		},
	}

	metrics := MetricsData{
		EstimatedCost: 1.5,
	}

	findings := []Finding{
		{Category: "error", Severity: "critical", Title: "Critical"},
		{Category: "cost", Severity: "high", Title: "High Cost"},
	}

	recs := generateRecommendations(processedRun, metrics, findings)

	// Should have high priority recommendations
	priorityCounts := make(map[string]int)
	for _, r := range recs {
		priorityCounts[r.Priority]++
	}

	if priorityCounts["high"] == 0 {
		t.Error("Expected at least one high priority recommendation")
	}
}

func TestDescribeFileAdditionalPatterns(t *testing.T) {
	// Test file description for additional file patterns not covered in audit_test.go
	tests := []struct {
		filename    string
		description string
	}{
		{"unknown_file", ""}, // Unknown file with no extension
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
