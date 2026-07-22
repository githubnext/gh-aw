package workflow

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
		return runPhaseEvals
	}
	if workflowData.IsDetectionRun {
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
