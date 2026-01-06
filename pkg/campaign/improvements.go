package campaign

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var improvementsLog = logger.New("campaign:improvements")

// ImprovementStatus represents the implementation status of a campaign improvement.
type ImprovementStatus string

const (
	// ImprovementImplemented indicates the improvement is fully implemented
	ImprovementImplemented ImprovementStatus = "Implemented"
	// ImprovementPartial indicates the improvement is partially implemented
	ImprovementPartial ImprovementStatus = "Partial"
	// ImprovementNotImplemented indicates the improvement is not yet implemented
	ImprovementNotImplemented ImprovementStatus = "Not Implemented"
)

// Improvement represents a campaign improvement from the improvements guide.
type Improvement struct {
	ID          int               `json:"id" console:"header:#"`
	Name        string            `json:"name" console:"header:Improvement"`
	Priority    string            `json:"priority" console:"header:Priority"`
	Status      ImprovementStatus `json:"status" console:"header:Status"`
	Description string            `json:"description,omitempty" console:"header:Description,maxlen:60,omitempty"`
	Evidence    []string          `json:"evidence,omitempty" console:"-"`
}

// ImprovementAnalysis represents the full analysis of campaign improvements.
type ImprovementAnalysis struct {
	TotalImprovements      int           `json:"total_improvements" console:"header:Total"`
	ImplementedCount       int           `json:"implemented_count" console:"header:Implemented"`
	PartialCount           int           `json:"partial_count" console:"header:Partial"`
	NotImplementedCount    int           `json:"not_implemented_count" console:"header:Not Implemented"`
	ImplementationProgress float64       `json:"implementation_progress" console:"header:Progress %"`
	Improvements           []Improvement `json:"improvements" console:"-"`
}

// AnalyzeImprovements analyzes the current campaign implementation to determine
// which improvements from the improvements guide have been implemented.
func AnalyzeImprovements(repoRoot string, specs []CampaignSpec) (*ImprovementAnalysis, error) {
	improvementsLog.Printf("Analyzing campaign improvements for %d campaigns in %s", len(specs), repoRoot)

	// Define all known improvements from the improvements guide
	improvements := []Improvement{
		{
			ID:          1,
			Name:        "Summarized Campaign Reports",
			Priority:    "High",
			Description: "Generate human-readable progress summaries with aggregated metrics and Epic issue updates",
		},
		{
			ID:          2,
			Name:        "Campaign Learning System",
			Priority:    "Medium",
			Description: "Capture and share learnings across runs and between campaigns",
		},
		{
			ID:          3,
			Name:        "Enhanced Metrics Integration",
			Priority:    "High",
			Description: "Enable orchestrators to read and act on historical metrics for decision-making",
		},
		{
			ID:          4,
			Name:        "Campaign Retrospectives",
			Priority:    "Medium",
			Description: "Add campaign completion workflow with retrospective reports",
		},
		{
			ID:          5,
			Name:        "Cross-Campaign Analytics",
			Priority:    "Low",
			Description: "Aggregate metrics across campaigns for portfolio-level visibility",
		},
	}

	// Analyze each improvement
	for i := range improvements {
		status, evidence := analyzeImprovementStatus(&improvements[i], repoRoot, specs)
		improvements[i].Status = status
		improvements[i].Evidence = evidence
	}

	// Calculate summary statistics
	var implemented, partial, notImplemented int
	for _, imp := range improvements {
		switch imp.Status {
		case ImprovementImplemented:
			implemented++
		case ImprovementPartial:
			partial++
		case ImprovementNotImplemented:
			notImplemented++
		}
	}

	// Calculate implementation progress (implemented + 0.5*partial) / total * 100
	progress := (float64(implemented) + float64(partial)*0.5) / float64(len(improvements)) * 100

	analysis := &ImprovementAnalysis{
		TotalImprovements:      len(improvements),
		ImplementedCount:       implemented,
		PartialCount:           partial,
		NotImplementedCount:    notImplemented,
		ImplementationProgress: progress,
		Improvements:           improvements,
	}

	improvementsLog.Printf("Analysis complete: %d implemented, %d partial, %d not implemented (%.1f%% progress)",
		implemented, partial, notImplemented, progress)

	return analysis, nil
}

// analyzeImprovementStatus analyzes a specific improvement to determine its implementation status.
func analyzeImprovementStatus(improvement *Improvement, repoRoot string, specs []CampaignSpec) (ImprovementStatus, []string) {
	improvementsLog.Printf("Analyzing improvement #%d: %s", improvement.ID, improvement.Name)

	var evidence []string

	switch improvement.ID {
	case 1: // Summarized Campaign Reports
		status := analyzeSummarizedReports(repoRoot, specs, &evidence)
		return status, evidence
	case 2: // Campaign Learning System
		status := analyzeLearningSystem(repoRoot, specs, &evidence)
		return status, evidence
	case 3: // Enhanced Metrics Integration
		status := analyzeMetricsIntegration(repoRoot, specs, &evidence)
		return status, evidence
	case 4: // Campaign Retrospectives
		status := analyzeRetrospectives(repoRoot, specs, &evidence)
		return status, evidence
	case 5: // Cross-Campaign Analytics
		status := analyzeCrossCampaignAnalytics(repoRoot, specs, &evidence)
		return status, evidence
	}

	return ImprovementNotImplemented, evidence
}

// analyzeSummarizedReports checks for summarized campaign report implementation.
func analyzeSummarizedReports(repoRoot string, specs []CampaignSpec, evidence *[]string) ImprovementStatus {
	// Check for report generation code
	reportFiles := []string{
		"pkg/campaign/report.go",
		"pkg/campaign/summary_report.go",
	}

	foundReportGen := false
	for _, file := range reportFiles {
		path := filepath.Join(repoRoot, file)
		if _, err := os.Stat(path); err == nil {
			*evidence = append(*evidence, fmt.Sprintf("Found report generation: %s", file))
			foundReportGen = true
		}
	}

	// Check for Epic issue update functionality
	foundEpicUpdate := false
	if checkFileContains(repoRoot, "pkg/campaign/orchestrator.go", "epic", "issue", "comment") {
		*evidence = append(*evidence, "Orchestrator may support Epic issue updates")
		foundEpicUpdate = true
	}

	// Check campaign specs for reporting configuration
	foundReportingConfig := false
	for _, spec := range specs {
		if spec.Governance != nil {
			// Even if there's no explicit reporting field, governance suggests structured reporting
			foundReportingConfig = true
			break
		}
	}

	if foundReportingConfig {
		*evidence = append(*evidence, "Campaign specs have governance configuration")
	}

	// Check for metrics aggregation
	if checkFileContains(repoRoot, "pkg/campaign/status.go", "metrics", "snapshot") {
		*evidence = append(*evidence, "Metrics reading capability exists in status.go")
	}

	if foundReportGen && foundEpicUpdate {
		return ImprovementImplemented
	} else if len(*evidence) > 0 {
		return ImprovementPartial
	}

	return ImprovementNotImplemented
}

// analyzeLearningSystem checks for campaign learning system implementation.
func analyzeLearningSystem(repoRoot string, specs []CampaignSpec, evidence *[]string) ImprovementStatus {
	// Check for learning-related files
	learningFiles := []string{
		"pkg/campaign/learning.go",
		"pkg/campaign/learnings.go",
		"pkg/campaign/insights.go",
	}

	foundLearning := false
	for _, file := range learningFiles {
		path := filepath.Join(repoRoot, file)
		if _, err := os.Stat(path); err == nil {
			*evidence = append(*evidence, fmt.Sprintf("Found learning system: %s", file))
			foundLearning = true
		}
	}

	// Check for learnings in memory paths
	if checkRepoMemoryBranch(repoRoot, "learnings.json") {
		*evidence = append(*evidence, "Found learnings.json in repo-memory")
		foundLearning = true
	}

	if foundLearning {
		return ImprovementImplemented
	}

	return ImprovementNotImplemented
}

// analyzeMetricsIntegration checks for enhanced metrics integration.
func analyzeMetricsIntegration(repoRoot string, specs []CampaignSpec, evidence *[]string) ImprovementStatus {
	// Check if metrics are being read
	if checkFileContains(repoRoot, "pkg/campaign/status.go", "FetchMetricsFromRepoMemory") {
		*evidence = append(*evidence, "Metrics fetching capability exists")
	}

	// Check for adaptive logic
	foundAdaptive := false
	if checkFileContains(repoRoot, "pkg/campaign/orchestrator.go", "adaptive", "velocity", "rate") {
		*evidence = append(*evidence, "Orchestrator may use adaptive logic")
		foundAdaptive = true
	}

	// Check for decision-making based on metrics
	if checkFileContains(repoRoot, "pkg/campaign/orchestrator.go", "metrics", "decision") {
		*evidence = append(*evidence, "Decision-making based on metrics detected")
		foundAdaptive = true
	}

	// Check campaign specs for governance with adaptive hints
	hasGovernance := false
	for _, spec := range specs {
		if spec.Governance != nil {
			hasGovernance = true
			if spec.Governance.MaxDiscoveryItemsPerRun > 0 || spec.Governance.MaxNewItemsPerRun > 0 {
				*evidence = append(*evidence, fmt.Sprintf("Campaign '%s' has governance rate limits", spec.ID))
			}
		}
	}

	if hasGovernance {
		*evidence = append(*evidence, "Campaigns have governance policies")
	}

	// Metrics are read but adaptive behavior is not fully implemented
	if len(*evidence) > 0 && !foundAdaptive {
		return ImprovementPartial
	} else if foundAdaptive {
		return ImprovementImplemented
	}

	return ImprovementNotImplemented
}

// analyzeRetrospectives checks for campaign retrospective implementation.
func analyzeRetrospectives(repoRoot string, specs []CampaignSpec, evidence *[]string) ImprovementStatus {
	// Check for retrospective files
	retroFiles := []string{
		"pkg/campaign/retrospective.go",
		"pkg/campaign/completion.go",
	}

	foundRetro := false
	for _, file := range retroFiles {
		path := filepath.Join(repoRoot, file)
		if _, err := os.Stat(path); err == nil {
			*evidence = append(*evidence, fmt.Sprintf("Found retrospective system: %s", file))
			foundRetro = true
		}
	}

	// Check for retrospective.json in memory
	if checkRepoMemoryBranch(repoRoot, "retrospective.json") {
		*evidence = append(*evidence, "Found retrospective.json in repo-memory")
		foundRetro = true
	}

	// Check for completed state campaigns
	hasCompletedCampaigns := false
	for _, spec := range specs {
		if spec.State == "completed" {
			hasCompletedCampaigns = true
			break
		}
	}

	if hasCompletedCampaigns {
		*evidence = append(*evidence, "Some campaigns have completed state")
	}

	if foundRetro {
		return ImprovementImplemented
	}

	return ImprovementNotImplemented
}

// analyzeCrossCampaignAnalytics checks for cross-campaign analytics implementation.
func analyzeCrossCampaignAnalytics(repoRoot string, specs []CampaignSpec, evidence *[]string) ImprovementStatus {
	// Check for analytics files
	analyticsFiles := []string{
		"pkg/campaign/analytics.go",
		"pkg/campaign/dashboard.go",
		"pkg/campaign/portfolio.go",
	}

	foundAnalytics := false
	for _, file := range analyticsFiles {
		path := filepath.Join(repoRoot, file)
		if _, err := os.Stat(path); err == nil {
			*evidence = append(*evidence, fmt.Sprintf("Found analytics system: %s", file))
			foundAnalytics = true
		}
	}

	// Check for aggregate analysis in status command
	if checkFileContains(repoRoot, "pkg/campaign/status.go", "aggregate", "portfolio") {
		*evidence = append(*evidence, "Status command may support aggregation")
		foundAnalytics = true
	}

	// The ability to list multiple campaigns is a basic building block
	if len(specs) > 1 {
		*evidence = append(*evidence, fmt.Sprintf("Repository has %d campaigns configured", len(specs)))
	}

	if foundAnalytics {
		return ImprovementImplemented
	} else if len(specs) > 1 {
		// Having multiple campaigns is a prerequisite but not the full feature
		return ImprovementPartial
	}

	return ImprovementNotImplemented
}

// checkFileContains checks if a file contains all the specified keywords.
func checkFileContains(repoRoot, relPath string, keywords ...string) bool {
	path := filepath.Join(repoRoot, relPath)
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	content := strings.ToLower(string(data))
	for _, keyword := range keywords {
		if !strings.Contains(content, strings.ToLower(keyword)) {
			return false
		}
	}

	return true
}

// checkRepoMemoryBranch checks if a file exists in the memory/campaigns branch.
func checkRepoMemoryBranch(repoRoot string, filename string) bool {
	// Try to list files in memory/campaigns branch
	cmd := exec.Command("git", "ls-tree", "-r", "--name-only", "memory/campaigns")
	cmd.Dir = repoRoot
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.Contains(string(output), filename)
}
