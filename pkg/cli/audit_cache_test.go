//go:build !integration

package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteAuditDataCreatesReusableAuditFile(t *testing.T) {
	runDir := t.TempDir()
	run := WorkflowRun{DatabaseID: 42, Status: "completed", Conclusion: "success"}
	auditData := AuditData{Overview: buildAuditOverview(run, nil)}

	require.NoError(t, writeAuditData(runDir, auditData))

	path := filepath.Join(runDir, auditFileName)
	assert.Equal(t, path, existingAuditPath(runDir))
	cached, ok := loadCachedAuditData(runDir, run)
	require.True(t, ok)
	assert.Equal(t, auditData.Overview, cached.Overview)
}

func TestLoadCachedAuditDataRequiresMatchingRunState(t *testing.T) {
	runDir := t.TempDir()
	run := WorkflowRun{DatabaseID: 42, Status: "in_progress", Conclusion: ""}
	require.NoError(t, writeAuditData(runDir, AuditData{Overview: buildAuditOverview(run, nil)}))

	tests := []struct {
		name string
		run  WorkflowRun
		ok   bool
	}{
		{name: "same state", run: run, ok: true},
		{name: "status changed", run: WorkflowRun{DatabaseID: 42, Status: "completed"}, ok: false},
		{name: "conclusion changed", run: WorkflowRun{DatabaseID: 42, Status: "in_progress", Conclusion: "success"}, ok: false},
		{name: "run changed", run: WorkflowRun{DatabaseID: 43, Status: "in_progress"}, ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := loadCachedAuditData(runDir, tt.run)
			assert.Equal(t, tt.ok, ok)
		})
	}
}

func TestWriteLogsAuditFilesUsesCacheUntilRunStateChanges(t *testing.T) {
	runDir := t.TempDir()
	run := WorkflowRun{
		DatabaseID:   42,
		Status:       "completed",
		Conclusion:   "success",
		WorkflowName: "test",
		LogsPath:     runDir,
	}
	cached := AuditData{
		Overview:    buildAuditOverview(run, nil),
		KeyFindings: []AuditFinding{{Title: "cache marker"}},
	}
	require.NoError(t, writeAuditData(runDir, cached))

	require.NoError(t, writeLogsAuditFiles([]ProcessedRun{{Run: run}}, false))
	data, err := os.ReadFile(auditPath(runDir))
	require.NoError(t, err)
	var unchanged AuditData
	require.NoError(t, json.Unmarshal(data, &unchanged))
	require.Len(t, unchanged.KeyFindings, 1)
	assert.Equal(t, "cache marker", unchanged.KeyFindings[0].Title)

	run.Conclusion = "failure"
	require.NoError(t, writeLogsAuditFiles([]ProcessedRun{{Run: run}}, false))
	data, err = os.ReadFile(auditPath(runDir))
	require.NoError(t, err)
	var refreshed AuditData
	require.NoError(t, json.Unmarshal(data, &refreshed))
	assert.Equal(t, "failure", refreshed.Overview.Conclusion)
	for _, finding := range refreshed.KeyFindings {
		assert.NotEqual(t, "cache marker", finding.Title)
	}
}

func TestBuildLogsDataLinksExistingAudit(t *testing.T) {
	runDir := t.TempDir()
	require.NoError(t, writeAuditData(runDir, AuditData{}))
	run := ProcessedRun{Run: WorkflowRun{DatabaseID: 42, LogsPath: runDir}}

	data := buildLogsData([]ProcessedRun{run}, t.TempDir(), nil)

	require.Len(t, data.Runs, 1)
	assert.Equal(t, auditPath(runDir), data.Runs[0].AuditPath)
}
