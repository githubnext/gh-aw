package cli

import (
	"strconv"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/logger"
)

var trialTypesLog = logger.New("cli:trial_types")

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

// extractSafeOutputErrors extracts the "errors" array (if any) from a safe-outputs
// artifact map, returning the rejected safe-output messages as strings.
func extractSafeOutputErrors(safeOutputs map[string]any) []string {
	if safeOutputs == nil {
		return nil
	}
	rawErrors, ok := safeOutputs["errors"]
	if !ok {
		return nil
	}
	errorsSlice, ok := rawErrors.([]any)
	if !ok {
		return nil
	}
	var messages []string
	for _, e := range errorsSlice {
		if msg, ok := e.(string); ok && msg != "" {
			messages = append(messages, msg)
		}
	}
	if len(messages) > 0 {
		trialTypesLog.Printf("Extracted %d rejected safe-output message(s)", len(messages))
	}
	return messages
}

// aggregateTrialResults aggregates a set of per-workflow trial results into an
// overall success flag, the total count of rejected safe-output messages across
// all workflows, and the first rejected message encountered (in result order).
func aggregateTrialResults(results []WorkflowTrialResult) (overallSuccess bool, totalRejected int, firstErrorMessage string) {
	overallSuccess = true
	for _, result := range results {
		if !result.Success {
			overallSuccess = false
			totalRejected += len(result.SafeOutputErrors)
			if firstErrorMessage == "" && len(result.SafeOutputErrors) > 0 {
				firstErrorMessage = result.SafeOutputErrors[0]
			}
		}
	}
	trialTypesLog.Printf("Aggregated %d trial result(s): success=%v totalRejected=%d", len(results), overallSuccess, totalRejected)
	return overallSuccess, totalRejected, firstErrorMessage
}

// sanitizeControlChars replaces ASCII control characters (including escape
// sequences) in a string with their Go-escaped representation. Rejected
// safe-output messages may embed agent-controlled content, so this prevents
// terminal/log control-sequence injection when the messages are printed to
// stderr or embedded in a returned error.
func sanitizeControlChars(s string) string {
	if s == "" {
		return s
	}
	var needsEscaping bool
	for _, r := range s {
		if isControlRune(r) {
			needsEscaping = true
			break
		}
	}
	if !needsEscaping {
		return s
	}
	var b strings.Builder
	for _, r := range s {
		if isControlRune(r) {
			b.WriteString(strconv.QuoteRune(r))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// isControlRune reports whether r is a C0 or C1 control character (including
// DEL), which may be interpreted as terminal/log control or escape sequences.
func isControlRune(r rune) bool {
	return r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f)
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
