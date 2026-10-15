package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// Run-scoped artifacts may belong to a previous attempt or an incomplete upload.
// Always refresh the entire artifact set so later collections see new evidence.
func auditClusterUsageDirectory(outputDir string) (string, func(), error) {
	parent := filepath.Dir(outputDir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return "", nil, err
	}
	dir, err := os.MkdirTemp(parent, ".audit-cluster-usage-")
	if err != nil {
		return "", nil, err
	}
	return dir, func() { _ = os.RemoveAll(dir) }, nil
}

// Only artifact directories selected by the usage filter are evidence. ReadDir
// sorts names, so multiple reusable-workflow artifacts have stable aggregation.
func auditClusterUsageArtifacts(outputDir string) ([]string, error) {
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return nil, err
	}
	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() && artifactMatchesFilter(entry.Name(), []string{"usage"}) {
			dirs = append(dirs, filepath.Join(outputDir, entry.Name()))
		}
	}
	return dirs, nil
}

func validateAuditClusterUsageAttempt(dir string, request auditClusterRequest) error {
	if request.metadata == nil || request.metadata.Attempt <= 1 {
		return nil
	}
	raw, err := os.ReadFile(filepath.Join(dir, "aw_info.json"))
	if err != nil {
		return fmt.Errorf("usage attempt provenance unavailable: %w", err)
	}
	var info struct {
		RunID      int64  `json:"run_id"`
		RunAttempt string `json:"run_attempt"`
	}
	if err := json.Unmarshal(raw, &info); err != nil {
		return fmt.Errorf("invalid usage attempt provenance: %w", err)
	}
	attempt, err := strconv.Atoi(info.RunAttempt)
	if err != nil || attempt != request.metadata.Attempt || info.RunID != request.runID {
		return fmt.Errorf("usage provenance does not match run %d attempt %d", request.runID, request.metadata.Attempt)
	}
	return nil
}

func collectAuditClusterUsageArtifacts(outputDir string, request auditClusterRequest, verbose bool, result *AuditClusterRun) {
	dirs, err := auditClusterUsageArtifacts(outputDir)
	if err != nil {
		result.Evidence = append(result.Evidence, "usage artifacts: "+err.Error())
		return
	}
	for _, dir := range dirs {
		if err := validateAuditClusterUsageAttempt(dir, request); err != nil {
			result.Evidence = append(result.Evidence, filepath.Base(dir)+": "+err.Error())
			continue
		}
		collectAuditClusterUsageArtifact(dir, verbose, result)
	}
}

func collectAuditClusterUsageArtifact(dir string, verbose bool, result *AuditClusterRun) {
	summary, summaryErr := loadUsageActivitySummary(dir)
	tokens, tokensErr := analyzeTokenUsage(dir, verbose)
	if summaryErr != nil {
		result.Evidence = append(result.Evidence, filepath.Base(dir)+" usage activity: "+summaryErr.Error())
	}
	if tokensErr != nil {
		result.Evidence = append(result.Evidence, filepath.Base(dir)+" usage tokens: "+tokensErr.Error())
	}
	if summary == nil && tokens == nil {
		result.Evidence = append(result.Evidence, filepath.Base(dir)+": no supported usage summary or token records found")
	}
	if summary != nil || tokens != nil {
		result.UsageState = "available"
	}
	if summary != nil {
		applyAuditClusterUsageActivity(dir, summary, result)
	}
	if tokens != nil {
		count := tokens.TotalTokens()
		if count > 0 {
			addAuditClusterMeasurement(&result.Tokens, &count)
		}
		if tokens.AICFound {
			if result.AIC == nil {
				result.AIC = new(float64)
			}
			*result.AIC += tokens.TotalAIC
		}
	}
}

func applyAuditClusterUsageActivity(dir string, summary *usageActivitySummary, result *AuditClusterRun) {
	if summary.Session != nil {
		turns, failed := measuredAuditClusterSession(dir)
		addAuditClusterMeasurement(&result.Turns, turns)
		addAuditClusterMeasurement(&result.FailedToolExecutions, failed)
		if summary.Session.FailedToolExecutions > 0 {
			result.Signatures = append(result.Signatures, "usage:failed-tool-executions")
		}
	}
	if summary.Gateway != nil {
		for _, tool := range summary.Gateway.Tools {
			if tool.FailedCalls > 0 {
				result.Signatures = append(result.Signatures, fmt.Sprintf("usage:tool:%s/%s:failure", tool.ServerName, tool.ToolName))
			}
		}
	}
}

func addAuditClusterMeasurement(total **int, measured *int) {
	if measured == nil {
		return
	}
	if *total == nil {
		*total = new(int)
	}
	**total += *measured
}

// Recover presence from the same schema-validated artifact used above: absent
// session fields must not become measured zeroes.
func measuredAuditClusterSession(dir string) (*int, *int) {
	data, err := os.ReadFile(filepath.Join(dir, "activity", "summary.json"))
	if err != nil {
		return nil, nil
	}
	var activity struct {
		Schema  string `json:"schema"`
		Session *struct {
			Turns  *int `json:"turns"`
			Failed *int `json:"failed_tool_executions"`
		} `json:"session"`
	}
	if json.Unmarshal(data, &activity) == nil && activity.Schema == usageActivitySummarySchema && activity.Session != nil {
		return activity.Session.Turns, activity.Session.Failed
	}
	return nil, nil
}
