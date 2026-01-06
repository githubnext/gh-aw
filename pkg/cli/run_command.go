package cli

// This file now serves as a compatibility bridge.
// The main functionality has been refactored into:
// - run_workflow_validation.go: Input validation, workflow checks (IsRunnable, validateWorkflowInputs, validateRemoteWorkflow)
// - run_workflow_tracking.go: Workflow run tracking and polling (WorkflowRunInfo, getLatestWorkflowRunWithRetry)
// - run_workflow_execution.go: Main execution logic (RunWorkflowOnGitHub, RunWorkflowsOnGitHub)
//
// Public API functions are now in their respective modules:
// - IsRunnable (run_workflow_validation.go)
// - RunWorkflowOnGitHub (run_workflow_execution.go)
// - RunWorkflowsOnGitHub (run_workflow_execution.go)
