package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/github/gh-aw/pkg/constants"
)

func auditPath(runOutputDir string) string {
	return filepath.Join(runOutputDir, auditFileName)
}

func existingAuditPath(runOutputDir string) string {
	path := auditPath(runOutputDir)
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return ""
}

func loadCachedAuditData(runOutputDir string, run WorkflowRun) (AuditData, bool) {
	data, err := os.ReadFile(auditPath(runOutputDir))
	if err != nil {
		return AuditData{}, false
	}
	var auditData AuditData
	if err := json.Unmarshal(data, &auditData); err != nil {
		return AuditData{}, false
	}
	if auditData.Overview.RunID != run.DatabaseID ||
		auditData.Overview.Status != run.Status ||
		auditData.Overview.Conclusion != run.Conclusion {
		return AuditData{}, false
	}
	return auditData, true
}

func writeAuditData(runOutputDir string, auditData AuditData) error {
	data, err := json.MarshalIndent(auditData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal audit data: %w", err)
	}
	if err := os.MkdirAll(runOutputDir, constants.DirPermSensitive); err != nil {
		return fmt.Errorf("failed to create audit output directory: %w", err)
	}
	if err := os.WriteFile(auditPath(runOutputDir), data, constants.FilePermPublic); err != nil {
		return fmt.Errorf("failed to write audit file: %w", err)
	}
	return nil
}

func writeLogsAuditFiles(processedRuns []ProcessedRun, verbose bool) error {
	for _, processedRun := range processedRuns {
		runOutputDir := processedRun.Run.LogsPath
		if _, ok := loadCachedAuditData(runOutputDir, processedRun.Run); ok {
			continue
		}
		metrics := LogMetrics{}
		if summary, ok := loadRunSummary(runOutputDir, verbose); ok {
			metrics = summary.Metrics
		}
		auditData, _ := buildLocalAuditData(processedRun, metrics, processedRun.MCPToolUsage)
		auditData.Comparison = buildAuditComparisonForProcessedRuns(processedRun, processedRuns)
		if err := writeAuditData(runOutputDir, auditData); err != nil {
			return fmt.Errorf("run %d: %w", processedRun.Run.DatabaseID, err)
		}
	}
	return nil
}
