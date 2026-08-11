package cli

import (
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var trialTypesLog = logger.New("cli:trial_types")

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
