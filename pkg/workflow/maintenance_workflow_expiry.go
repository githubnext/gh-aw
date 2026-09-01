package workflow

import (
	"os"
	"path/filepath"
)

// DisableDefaultActionFailureExpiryMarkersIfUnenforced disables implicit
// action-failure expiry markers when no maintenance workflow exists to enforce them.
// The generated maintenance workflow always includes the global expiry sweeper.
func DisableDefaultActionFailureExpiryMarkersIfUnenforced(workflowDataList []*WorkflowData, workflowDir string) {
	if _, err := os.Stat(filepath.Join(workflowDir, "agentics-maintenance.yml")); os.IsNotExist(err) {
		disableDefaultActionFailureExpiryMarkers(workflowDataList, workflowDir)
	}
}
