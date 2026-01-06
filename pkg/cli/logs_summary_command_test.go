package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadLogsDataFromFile tests loading LogsData from a summary.json file
func TestLoadLogsDataFromFile(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	summaryPath := filepath.Join(tmpDir, "summary.json")

	// Create sample logs data
	expectedData := LogsData{
		Summary: LogsSummary{
			TotalRuns:         3,
			TotalDuration:     "1h30m",
			TotalTokens:       15000,
			TotalCost:         2.50,
			TotalTurns:        25,
			TotalErrors:       2,
			TotalWarnings:     5,
			TotalMissingTools: 1,
		},
		Runs: []RunData{
			{
				DatabaseID:       12345,
				WorkflowName:     "Test Workflow",
				Status:           "completed",
				Duration:         "30m",
				TokenUsage:       5000,
				EstimatedCost:    0.75,
				ErrorCount:       0,
				WarningCount:     2,
				MissingToolCount: 0,
				CreatedAt:        time.Now(),
				LogsPath:         "/tmp/logs/run-12345",
			},
		},
		LogsLocation: tmpDir,
	}

	// Write the data to file
	data, err := json.MarshalIndent(expectedData, "", "  ")
	require.NoError(t, err, "Failed to marshal test data")

	err = os.WriteFile(summaryPath, data, 0644)
	require.NoError(t, err, "Failed to write test file")

	// Load the data back
	loadedData, err := loadLogsDataFromFile(summaryPath)
	require.NoError(t, err, "Failed to load logs data")

	// Verify key fields
	assert.Equal(t, expectedData.Summary.TotalRuns, loadedData.Summary.TotalRuns)
	assert.Equal(t, expectedData.Summary.TotalTokens, loadedData.Summary.TotalTokens)
	assert.Equal(t, expectedData.Summary.TotalCost, loadedData.Summary.TotalCost)
	assert.Equal(t, len(expectedData.Runs), len(loadedData.Runs))
}

// TestGenerateMarkdownFromLogsData tests markdown generation
func TestGenerateMarkdownFromLogsData(t *testing.T) {
	// Create sample logs data
	logsData := LogsData{
		Summary: LogsSummary{
			TotalRuns:         5,
			TotalDuration:     "2h15m",
			TotalTokens:       25000,
			TotalCost:         5.75,
			TotalTurns:        42,
			TotalErrors:       3,
			TotalWarnings:     8,
			TotalMissingTools: 2,
		},
		FirewallLog: &FirewallLogSummary{
			TotalRequests:    150,
			AllowedRequests:  140,
			DeniedRequests:   10,
			RequestsByDomain: map[string]DomainRequestStats{
				"api.github.com": {
					Allowed: 50,
					Denied:  0,
				},
				"example.com": {
					Allowed: 40,
					Denied:  5,
				},
				"blocked.com": {
					Allowed: 0,
					Denied:  5,
				},
			},
		},
		ErrorsAndWarnings: []ErrorSummary{
			{
				Type:    "Error",
				Message: "Connection timeout",
				Count:   3,
				Engine:  "copilot",
			},
			{
				Type:    "Warning",
				Message: "Rate limit approaching",
				Count:   8,
				Engine:  "copilot",
			},
		},
		MissingTools: []MissingToolSummary{
			{
				Tool:              "web_search",
				Count:             2,
				WorkflowsDisplay:  "test-workflow",
			},
		},
	}

	config := LogsSummaryConfig{
		WorkflowName: "test-workflow",
		FirewallOnly: true,
	}

	// Generate markdown
	markdown := generateMarkdownFromLogsData(logsData, config)

	// Verify markdown content
	assert.Contains(t, markdown, "# Workflow Execution Summary")
	assert.Contains(t, markdown, "**Workflow:** test-workflow")
	assert.Contains(t, markdown, "Firewall: enabled")
	assert.Contains(t, markdown, "## Summary")
	assert.Contains(t, markdown, "| Total Runs | 5 |")
	assert.Contains(t, markdown, "| Total Tokens | 25000 |")
	assert.Contains(t, markdown, "$5.7500")
	assert.Contains(t, markdown, "## 🔥 Firewall Analysis")
	assert.Contains(t, markdown, "**Total Requests:** 150")
	assert.Contains(t, markdown, "✅ Allowed: 140")
	assert.Contains(t, markdown, "❌ Denied: 10")
	assert.Contains(t, markdown, "### Top Domains")
	assert.Contains(t, markdown, "api.github.com")
	assert.Contains(t, markdown, "## ⚠️ Errors and Warnings")
	assert.Contains(t, markdown, "Connection timeout")
	assert.Contains(t, markdown, "## 🛠️ Missing Tools")
	assert.Contains(t, markdown, "web_search")
}

// TestGenerateMarkdownWithoutOptionalSections tests markdown generation with minimal data
func TestGenerateMarkdownWithoutOptionalSections(t *testing.T) {
	// Create minimal logs data (no firewall, errors, or missing tools)
	logsData := LogsData{
		Summary: LogsSummary{
			TotalRuns:         1,
			TotalDuration:     "15m",
			TotalTokens:       1000,
			TotalCost:         0.25,
			TotalTurns:        5,
			TotalErrors:       0,
			TotalWarnings:     0,
			TotalMissingTools: 0,
		},
	}

	config := LogsSummaryConfig{}

	// Generate markdown
	markdown := generateMarkdownFromLogsData(logsData, config)

	// Verify only summary section is present
	assert.Contains(t, markdown, "# Workflow Execution Summary")
	assert.Contains(t, markdown, "## Summary")
	assert.NotContains(t, markdown, "## 🔥 Firewall Analysis")
	assert.NotContains(t, markdown, "## ⚠️ Errors and Warnings")
	assert.NotContains(t, markdown, "## 🛠️ Missing Tools")
	assert.Contains(t, markdown, "_Generated by [GitHub Agentic Workflows]")
}

// TestGenerateMarkdownFirewallAnalysisTopDomains tests that only top 10 domains are shown
func TestGenerateMarkdownFirewallAnalysisTopDomains(t *testing.T) {
	// Create logs data with many domains
	requestsByDomain := make(map[string]DomainRequestStats)
	for i := 1; i <= 15; i++ {
		domainName := "domain" + string(rune('a'+i-1)) + ".com"
		requestsByDomain[domainName] = DomainRequestStats{
			Allowed: i * 10, // Different counts to test sorting
			Denied:  0,
		}
	}

	logsData := LogsData{
		Summary: LogsSummary{
			TotalRuns: 1,
		},
		FirewallLog: &FirewallLogSummary{
			TotalRequests:    150,
			AllowedRequests:  150,
			DeniedRequests:   0,
			RequestsByDomain: requestsByDomain,
		},
	}

	config := LogsSummaryConfig{}

	// Generate markdown
	markdown := generateMarkdownFromLogsData(logsData, config)

	// Count the number of domain rows in the table
	// The markdown table has headers + separator + data rows
	lines := strings.Split(markdown, "\n")
	domainTableStart := false
	domainRowCount := 0

	for _, line := range lines {
		if strings.Contains(line, "### Top Domains") {
			domainTableStart = true
			continue
		}
		if domainTableStart && strings.HasPrefix(line, "| domain") {
			domainRowCount++
		}
		if domainTableStart && line == "" {
			break
		}
	}

	// Should have at most 10 domains
	assert.LessOrEqual(t, domainRowCount, 10, "Should show at most 10 domains")
}

// TestGenerateMarkdownErrorTruncation tests that long error messages are truncated
func TestGenerateMarkdownErrorTruncation(t *testing.T) {
	// Create an error with a very long message
	longMessage := strings.Repeat("This is a very long error message. ", 10) // > 80 chars

	logsData := LogsData{
		Summary: LogsSummary{
			TotalRuns: 1,
		},
		ErrorsAndWarnings: []ErrorSummary{
			{
				Type:    "Error",
				Message: longMessage,
				Count:   1,
				Engine:  "copilot",
			},
		},
	}

	config := LogsSummaryConfig{}

	// Generate markdown
	markdown := generateMarkdownFromLogsData(logsData, config)

	// Find the error message line
	lines := strings.Split(markdown, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "| This is a very long") {
			// Message should be truncated and end with "..."
			assert.Contains(t, line, "...")
			// Should not contain the full message
			assert.NotContains(t, line, longMessage)
			return
		}
	}

	t.Error("Expected to find truncated error message in markdown")
}
