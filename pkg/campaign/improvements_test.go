package campaign

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyzeImprovements(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "campaign-improvements-test")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(tmpDir)

	// Create some test campaign specs
	specs := []CampaignSpec{
		{
			ID:   "test-campaign-1",
			Name: "Test Campaign 1",
			Governance: &CampaignGovernancePolicy{
				MaxNewItemsPerRun:       5,
				MaxDiscoveryItemsPerRun: 50,
			},
		},
		{
			ID:    "test-campaign-2",
			Name:  "Test Campaign 2",
			State: "completed",
		},
	}

	analysis, err := AnalyzeImprovements(tmpDir, specs)
	require.NoError(t, err, "AnalyzeImprovements should not fail")
	assert.NotNil(t, analysis, "Analysis should not be nil")

	// Verify basic structure
	assert.Equal(t, 5, analysis.TotalImprovements, "Should have 5 improvements")
	assert.Equal(t, 5, len(analysis.Improvements), "Should return 5 improvement entries")

	// Verify all improvements are present
	improvementNames := make(map[string]bool)
	for _, imp := range analysis.Improvements {
		improvementNames[imp.Name] = true
		assert.NotEmpty(t, imp.Name, "Improvement name should not be empty")
		assert.NotEmpty(t, imp.Priority, "Improvement priority should not be empty")
		assert.NotEmpty(t, imp.Description, "Improvement description should not be empty")
	}

	assert.True(t, improvementNames["Summarized Campaign Reports"], "Should have Summarized Campaign Reports")
	assert.True(t, improvementNames["Campaign Learning System"], "Should have Campaign Learning System")
	assert.True(t, improvementNames["Enhanced Metrics Integration"], "Should have Enhanced Metrics Integration")
	assert.True(t, improvementNames["Campaign Retrospectives"], "Should have Campaign Retrospectives")
	assert.True(t, improvementNames["Cross-Campaign Analytics"], "Should have Cross-Campaign Analytics")

	// Verify counts sum to total
	assert.Equal(t, analysis.TotalImprovements,
		analysis.ImplementedCount+analysis.PartialCount+analysis.NotImplementedCount,
		"Counts should sum to total")

	// Verify progress is within valid range
	assert.GreaterOrEqual(t, analysis.ImplementationProgress, 0.0, "Progress should be >= 0")
	assert.LessOrEqual(t, analysis.ImplementationProgress, 100.0, "Progress should be <= 100")
}

func TestAnalyzeImprovementsWithEmptyCampaigns(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "campaign-improvements-empty-test")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(tmpDir)

	// Analyze with no campaigns
	analysis, err := AnalyzeImprovements(tmpDir, []CampaignSpec{})
	require.NoError(t, err, "AnalyzeImprovements should not fail with empty campaigns")
	assert.NotNil(t, analysis, "Analysis should not be nil")
	assert.Equal(t, 5, analysis.TotalImprovements, "Should still have 5 improvements")
}

func TestCheckFileContains(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "campaign-file-contains-test")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.go")
	content := `package test

// This file contains metrics and snapshot functionality
func GetMetrics() {}
func SaveSnapshot() {}
`
	err = os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err, "Failed to write test file")

	// Test successful match
	assert.True(t, checkFileContains(tmpDir, "test.go", "metrics", "snapshot"),
		"Should find both keywords")

	// Test case insensitivity
	assert.True(t, checkFileContains(tmpDir, "test.go", "METRICS", "SNAPSHOT"),
		"Should be case insensitive")

	// Test missing keyword
	assert.False(t, checkFileContains(tmpDir, "test.go", "metrics", "missing"),
		"Should not match when keyword is missing")

	// Test non-existent file
	assert.False(t, checkFileContains(tmpDir, "nonexistent.go", "anything"),
		"Should return false for non-existent file")
}

func TestImprovementPriorities(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "campaign-priorities-test")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(tmpDir)

	analysis, err := AnalyzeImprovements(tmpDir, []CampaignSpec{})
	require.NoError(t, err, "AnalyzeImprovements should not fail")

	// Verify priorities match the documentation
	priorityMap := make(map[int]string)
	for _, imp := range analysis.Improvements {
		priorityMap[imp.ID] = imp.Priority
	}

	assert.Equal(t, "High", priorityMap[1], "Summarized Campaign Reports should be High priority")
	assert.Equal(t, "Medium", priorityMap[2], "Campaign Learning System should be Medium priority")
	assert.Equal(t, "High", priorityMap[3], "Enhanced Metrics Integration should be High priority")
	assert.Equal(t, "Medium", priorityMap[4], "Campaign Retrospectives should be Medium priority")
	assert.Equal(t, "Low", priorityMap[5], "Cross-Campaign Analytics should be Low priority")
}

func TestImprovementProgressCalculation(t *testing.T) {
	tests := []struct {
		name             string
		implemented      int
		partial          int
		notImplemented   int
		expectedProgress float64
		expectedTotal    int
	}{
		{
			name:             "all implemented",
			implemented:      5,
			partial:          0,
			notImplemented:   0,
			expectedProgress: 100.0,
			expectedTotal:    5,
		},
		{
			name:             "none implemented",
			implemented:      0,
			partial:          0,
			notImplemented:   5,
			expectedProgress: 0.0,
			expectedTotal:    5,
		},
		{
			name:             "all partial",
			implemented:      0,
			partial:          5,
			notImplemented:   0,
			expectedProgress: 50.0,
			expectedTotal:    5,
		},
		{
			name:             "mixed",
			implemented:      2,
			partial:          2,
			notImplemented:   1,
			expectedProgress: 60.0,
			expectedTotal:    5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Calculate expected progress
			progress := (float64(tt.implemented) + float64(tt.partial)*0.5) / float64(tt.expectedTotal) * 100
			assert.InDelta(t, tt.expectedProgress, progress, 0.1,
				"Progress calculation should match expected value")
		})
	}
}

func TestAnalyzeSummarizedReportsDetection(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "campaign-reports-test")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(tmpDir)

	// Create pkg/campaign directory
	campaignDir := filepath.Join(tmpDir, "pkg", "campaign")
	err = os.MkdirAll(campaignDir, 0755)
	require.NoError(t, err, "Failed to create campaign directory")

	// Test without report files
	specs := []CampaignSpec{{ID: "test", Governance: &CampaignGovernancePolicy{}}}
	var evidence []string
	status := analyzeSummarizedReports(tmpDir, specs, &evidence)
	assert.Equal(t, ImprovementPartial, status, "Should be partial with only governance")

	// Create a report.go file
	reportFile := filepath.Join(campaignDir, "report.go")
	err = os.WriteFile(reportFile, []byte("package campaign\n\nfunc GenerateReport() {}"), 0644)
	require.NoError(t, err, "Failed to write report file")

	// Test with report file
	evidence = []string{}
	status = analyzeSummarizedReports(tmpDir, specs, &evidence)
	assert.NotEqual(t, ImprovementNotImplemented, status, "Should not be not implemented with report file")
	assert.Contains(t, evidence, "Found report generation: pkg/campaign/report.go",
		"Evidence should mention report file")
}

func TestAnalyzeLearningSystemDetection(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "campaign-learning-test")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(tmpDir)

	var evidence []string
	status := analyzeLearningSystem(tmpDir, []CampaignSpec{}, &evidence)
	assert.Equal(t, ImprovementNotImplemented, status, "Should be not implemented without learning files")
	assert.Empty(t, evidence, "Evidence should be empty")

	// Create pkg/campaign directory
	campaignDir := filepath.Join(tmpDir, "pkg", "campaign")
	err = os.MkdirAll(campaignDir, 0755)
	require.NoError(t, err, "Failed to create campaign directory")

	// Create a learning.go file
	learningFile := filepath.Join(campaignDir, "learning.go")
	err = os.WriteFile(learningFile, []byte("package campaign\n\nfunc CaptureLearning() {}"), 0644)
	require.NoError(t, err, "Failed to write learning file")

	// Test with learning file
	evidence = []string{}
	status = analyzeLearningSystem(tmpDir, []CampaignSpec{}, &evidence)
	assert.Equal(t, ImprovementImplemented, status, "Should be implemented with learning file")
	assert.Contains(t, evidence, "Found learning system: pkg/campaign/learning.go",
		"Evidence should mention learning file")
}

func TestAnalyzeCrossCampaignAnalytics(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "campaign-analytics-test")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(tmpDir)

	// Test with single campaign
	var evidence []string
	status := analyzeCrossCampaignAnalytics(tmpDir, []CampaignSpec{{ID: "test"}}, &evidence)
	assert.Equal(t, ImprovementNotImplemented, status, "Should be not implemented with single campaign")

	// Test with multiple campaigns
	evidence = []string{}
	specs := []CampaignSpec{{ID: "test1"}, {ID: "test2"}, {ID: "test3"}}
	status = analyzeCrossCampaignAnalytics(tmpDir, specs, &evidence)
	assert.Equal(t, ImprovementPartial, status, "Should be partial with multiple campaigns")
	assert.Contains(t, evidence, "Repository has 3 campaigns configured",
		"Evidence should mention campaign count")
}
