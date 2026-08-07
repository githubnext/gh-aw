package workflow

import "github.com/github/gh-aw/pkg/logger"

var runPhaseLog = logger.New("workflow:run_phase")

const (
	runPhaseAgent     = "agent"
	runPhaseDetection = "detection"
	runPhaseEvals     = "evals"
)

func workflowRunPhase(workflowData *WorkflowData) string {
	if workflowData == nil {
		return runPhaseAgent
	}
	if workflowData.IsEvalsRun {
		runPhaseLog.Print("Run phase resolved to evals")
		return runPhaseEvals
	}
	if workflowData.IsDetectionRun {
		runPhaseLog.Print("Run phase resolved to detection")
		return runPhaseDetection
	}
	return runPhaseAgent
}

func isDetectionRun(workflowData *WorkflowData) bool {
	if workflowData == nil {
		return false
	}
	// Keep legacy detection inference for compatibility with tests and older call paths
	// that still use SafeOutputs==nil as the signal for detection jobs.
	return workflowData.IsDetectionRun || (workflowData.SafeOutputs == nil && !workflowData.IsEvalsRun)
}
