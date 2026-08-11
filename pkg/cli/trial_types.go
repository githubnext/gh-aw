package cli

import "time"

// WorkflowTrialResult represents the result of running a single workflow trial
type WorkflowTrialResult struct {
	WorkflowName string         `json:"workflow_name"`
	RunID        string         `json:"run_id"`
	SafeOutputs  map[string]any `json:"safe_outputs"`
	//AgentStdioLogs      []string               `json:"agent_stdio_logs,omitempty"`
	AgenticRunInfo      map[string]any `json:"agentic_run_info,omitempty"`
	AdditionalArtifacts map[string]any `json:"additional_artifacts,omitempty"`
	Timestamp           time.Time      `json:"timestamp"`
	// Success reports whether the trial completed without any rejected safe-output
	// messages. It is false when the safe-outputs artifact contains a non-empty
	// "errors" array.
	Success bool `json:"success"`
	// SafeOutputErrors contains the rejected safe-output messages, if any, extracted
	// from the safe-outputs artifact's "errors" array.
	SafeOutputErrors []string `json:"safe_output_errors,omitempty"`
}

// CombinedTrialResult represents the combined results of multiple workflow trials
type CombinedTrialResult struct {
	WorkflowNames []string              `json:"workflow_names"`
	Results       []WorkflowTrialResult `json:"results"`
	Timestamp     time.Time             `json:"timestamp"`
	// Success reports whether all workflow trials completed without any rejected
	// safe-output messages.
	Success bool `json:"success"`
}

// TrialRepoContext groups repository-related configuration for trial execution
type TrialRepoContext struct {
	LogicalRepo string // The repo to simulate execution against
	CloneRepo   string // Alternative to LogicalRepo: clone this repo's contents
	HostRepo    string // The host repository where workflows will be installed
}

// TrialOptions contains all configuration options for running workflow trials
type TrialOptions struct {
	Repos                  TrialRepoContext
	DeleteHostRepo         bool
	ForceDelete            bool
	Quiet                  bool
	DryRun                 bool
	JSONOutput             bool
	TimeoutMinutes         int
	TriggerContext         string
	RepeatCount            int
	AutoMergePRs           bool
	EngineOverride         string
	AppendText             string
	Verbose                bool
	DisableSecurityScanner bool
}
