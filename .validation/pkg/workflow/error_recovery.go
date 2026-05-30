package workflow

import (
	"errors"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var errorRecoveryLog = logger.New("workflow:error_recovery")

// ErrorSeverity classifies how urgently a compilation error should be fixed.
type ErrorSeverity int

const (
	SeverityCritical ErrorSeverity = iota
	SeverityHigh
	SeverityMedium
	SeverityLow
)

// PrioritizedError describes a single user-facing error after severity sorting.
type PrioritizedError struct {
	Message    string
	Severity   ErrorSeverity
	Category   string
	Suggestion string
}

// RecoveryPlan describes the recommended next steps for a set of related errors.
type RecoveryPlan struct {
	Steps []string
}

// PrioritizedErrorReport contains the final prioritized compilation report.
type PrioritizedErrorReport struct {
	TotalCount      int
	DisplayedErrors []PrioritizedError
	HiddenCount     int
	SuppressedCount int
	RecoveryPlan    *RecoveryPlan
}

// ExpandErrorMessages unwraps joined compiler errors into individual display messages.
func ExpandErrorMessages(err error) []string {
	if err == nil {
		return nil
	}

	errorRecoveryLog.Print("Expanding error messages from compilation error")
	var messages []string
	collectErrorMessages(err, &messages)
	if len(messages) == 0 {
		return []string{strings.TrimSpace(err.Error())}
	}

	seen := make(map[string]struct{}, len(messages))
	result := make([]string, 0, len(messages))
	for _, msg := range messages {
		for _, expanded := range expandDisplayMessage(msg) {
			trimmed := strings.TrimSpace(expanded)
			if trimmed == "" {
				continue
			}
			key := normalizeErrorMessage(trimmed)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			result = append(result, trimmed)
		}
	}

	if len(result) == 0 {
		return []string{strings.TrimSpace(err.Error())}
	}

	return result
}

func collectErrorMessages(err error, messages *[]string) {
	if err == nil {
		return
	}

	var multi interface{ Unwrap() []error }
	if errors.As(err, &multi) {
		children := multi.Unwrap()
		if len(children) > 0 {
			for _, child := range children {
				collectErrorMessages(child, messages)
			}
			return
		}
	}

	var single interface{ Unwrap() error }
	if errors.As(err, &single) {
		child := single.Unwrap()
		if child != nil && shouldSuppressWrapperMessage(err.Error()) {
			collectErrorMessages(child, messages)
			return
		}
	}

	*messages = append(*messages, strings.TrimSpace(err.Error()))
}

func shouldSuppressWrapperMessage(message string) bool {
	lower := strings.ToLower(strings.TrimSpace(message))
	return strings.HasPrefix(lower, "found ") && strings.Contains(lower, " errors:")
}

// BuildPrioritizedErrorReportFromMessages classifies, suppresses, and limits messages.
func BuildPrioritizedErrorReportFromMessages(messages []string, showAll bool) PrioritizedErrorReport {
	errorRecoveryLog.Printf("Building prioritized error report: message_count=%d, show_all=%v", len(messages), showAll)
	prioritized, suppressedCount := prioritizeErrorMessages(messages)
	displayed := prioritized
	if !showAll && len(displayed) > 5 {
		errorRecoveryLog.Printf("Truncating displayed errors from %d to top 5 (set show_all=true to see all)", len(displayed))
		displayed = displayed[:5]
	}

	report := PrioritizedErrorReport{
		TotalCount:      len(prioritized),
		DisplayedErrors: displayed,
		HiddenCount:     len(prioritized) - len(displayed),
		SuppressedCount: suppressedCount,
	}
	if len(prioritized) > 1 {
		report.RecoveryPlan = buildRecoveryPlan(prioritized, suppressedCount)
	}

	errorRecoveryLog.Printf("Prioritized report ready: total=%d, displayed=%d, hidden=%d, suppressed=%d, has_plan=%v",
		report.TotalCount, len(report.DisplayedErrors), report.HiddenCount, report.SuppressedCount, report.RecoveryPlan != nil)
	return report
}

func prioritizeErrorMessages(messages []string) ([]PrioritizedError, int) {
	candidates := make([]PrioritizedError, 0, len(messages))
	seen := make(map[string]struct{}, len(messages))
	for _, message := range messages {
		trimmed := strings.TrimSpace(message)
		if trimmed == "" {
			continue
		}
		key := normalizeErrorMessage(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		candidates = append(candidates, classifyErrorMessage(trimmed))
	}

	var prioritized []PrioritizedError
	suppressedCount := 0
	hasCriticalSyntax := false
	for _, candidate := range candidates {
		if candidate.Severity == SeverityCritical && candidate.Category == "syntax" {
			hasCriticalSyntax = true
			break
		}
	}

	for _, candidate := range candidates {
		if shouldSuppressCascadingError(candidate, hasCriticalSyntax) {
			suppressedCount++
			continue
		}
		prioritized = append(prioritized, candidate)
	}

	if hasCriticalSyntax {
		errorRecoveryLog.Printf("Critical syntax errors detected, suppressed %d cascading errors", suppressedCount)
	}

	if len(prioritized) == 0 {
		prioritized = candidates
		suppressedCount = 0
	}

	sort.SliceStable(prioritized, func(i, j int) bool {
		if prioritized[i].Severity != prioritized[j].Severity {
			return prioritized[i].Severity < prioritized[j].Severity
		}
		if prioritized[i].Category != prioritized[j].Category {
			return prioritized[i].Category < prioritized[j].Category
		}
		return prioritized[i].Message < prioritized[j].Message
	})

	return prioritized, suppressedCount
}

func shouldSuppressCascadingError(candidate PrioritizedError, hasCriticalSyntax bool) bool {
	if !hasCriticalSyntax {
		return false
	}

	lower := normalizeErrorMessage(candidate.Message)
	if candidate.Category == "syntax" && candidate.Severity != SeverityCritical {
		return true
	}

	if candidate.Category != "configuration" {
		return false
	}

	return strings.Contains(lower, "missing required") ||
		strings.Contains(lower, "field 'engine'") ||
		strings.Contains(lower, "field 'on'") ||
		strings.Contains(lower, "frontmatter")
}

func classifyValidationSeverity(field string, reason string) (ErrorSeverity, string) {
	lowerField := strings.ToLower(field)
	lowerReason := strings.ToLower(reason)

	switch {
	case lowerField == "engine" || strings.Contains(lowerReason, "invalid engine"):
		return SeverityCritical, "configuration"
	case strings.Contains(lowerField, "network") || strings.Contains(lowerReason, "strict mode"):
		return SeverityHigh, "permissions"
	case strings.Contains(lowerField, "mcp") || strings.Contains(lowerField, "tools"):
		return SeverityHigh, "tools"
	case strings.Contains(lowerField, "event") || strings.Contains(lowerField, "filter"):
		return SeverityMedium, "events"
	case strings.Contains(lowerField, "permission"):
		return SeverityMedium, "permissions"
	case strings.Contains(lowerField, "runtime") || strings.Contains(lowerField, "version"):
		return SeverityMedium, "runtime"
	case strings.Contains(lowerField, "deprecated") || strings.Contains(lowerReason, "deprecated"):
		return SeverityLow, "deprecation"
	default:
		return SeverityMedium, "configuration"
	}
}

func classifyErrorMessage(message string) PrioritizedError {
	lower := normalizeErrorMessage(message)

	switch {
	case strings.Contains(lower, "failed to parse frontmatter"),
		strings.Contains(lower, "failed to parse yaml frontmatter"),
		strings.Contains(lower, "no frontmatter found"),
		strings.Contains(lower, "mapping values are not allowed"),
		strings.Contains(lower, "did not find expected key"):
		return PrioritizedError{
			Message:    message,
			Severity:   SeverityCritical,
			Category:   "syntax",
			Suggestion: "Fix the YAML/frontmatter syntax first, then re-run `gh aw compile`.",
		}
	case strings.Contains(lower, "invalid engine"),
		strings.Contains(lower, "invalid engine value"):
		return PrioritizedError{
			Message:    message,
			Severity:   SeverityCritical,
			Category:   "configuration",
			Suggestion: "Use a supported engine name in frontmatter, for example `engine: copilot`.",
		}
	case strings.Contains(lower, "field 'engine'") && (strings.Contains(lower, "empty") || strings.Contains(lower, "required")):
		return PrioritizedError{
			Message:    message,
			Severity:   SeverityCritical,
			Category:   "configuration",
			Suggestion: "Add an `engine:` value to the workflow frontmatter before fixing lower-priority issues.",
		}
	case strings.Contains(lower, "network.allowed"),
		(strings.Contains(lower, "network") && strings.Contains(lower, "strict mode")):
		return PrioritizedError{
			Message:    message,
			Severity:   SeverityHigh,
			Category:   "permissions",
			Suggestion: "Either enable strict mode for the workflow or remove the unsupported network configuration.",
		}
	case strings.Contains(lower, "mcp"),
		strings.Contains(lower, "tool configuration"),
		strings.Contains(lower, "tools."),
		strings.Contains(lower, "tools/"):
		return PrioritizedError{
			Message:    message,
			Severity:   SeverityHigh,
			Category:   "tools",
			Suggestion: "Check the `tools:` and MCP server configuration for missing required fields or unsupported values.",
		}
	case strings.Contains(lower, "event"),
		strings.Contains(lower, "workflow_dispatch"),
		strings.Contains(lower, "pull-request"),
		strings.Contains(lower, "pull_request"):
		return PrioritizedError{
			Message:    message,
			Severity:   SeverityMedium,
			Category:   "events",
			Suggestion: "Correct the event or filter name, then re-run compilation.",
		}
	case strings.Contains(lower, "permission"):
		return PrioritizedError{
			Message:    message,
			Severity:   SeverityMedium,
			Category:   "permissions",
			Suggestion: "Adjust the permissions block to match the workflow's required scopes.",
		}
	case strings.Contains(lower, "runtime"),
		strings.Contains(lower, "node version"),
		strings.Contains(lower, "python version"),
		strings.Contains(lower, "version conflict"):
		return PrioritizedError{
			Message:    message,
			Severity:   SeverityMedium,
			Category:   "runtime",
			Suggestion: "Resolve the runtime version conflict or choose a supported version.",
		}
	case strings.Contains(lower, "deprecated"),
		strings.Contains(lower, "warning"),
		strings.Contains(lower, "recommend"):
		return PrioritizedError{
			Message:    message,
			Severity:   SeverityLow,
			Category:   "deprecation",
			Suggestion: "Clean this up after the higher-priority errors are fixed.",
		}
	default:
		return PrioritizedError{
			Message:    message,
			Severity:   SeverityMedium,
			Category:   "configuration",
			Suggestion: "Fix this configuration issue and re-run `gh aw compile`.",
		}
	}
}

func buildRecoveryPlan(prioritized []PrioritizedError, suppressedCount int) *RecoveryPlan {
	steps := make([]string, 0, 4)
	if hasSeverity(prioritized, SeverityCritical) {
		steps = append(steps, "Fix the critical syntax and required-configuration errors first.")
	}
	if hasSeverity(prioritized, SeverityCritical) || suppressedCount > 0 {
		steps = append(steps, "Re-run `gh aw compile` after the first fixes to confirm whether cascading errors disappear.")
	}
	if hasSeverity(prioritized, SeverityHigh) {
		steps = append(steps, "Address the remaining high-priority network, tool, or MCP configuration issues next.")
	}
	if hasSeverity(prioritized, SeverityMedium) {
		steps = append(steps, "Resolve the remaining event, permission, and runtime validation errors.")
	}
	if hasSeverity(prioritized, SeverityLow) {
		steps = append(steps, "Clean up any remaining warnings or deprecated fields last.")
	}
	if len(steps) == 0 {
		return nil
	}

	return &RecoveryPlan{Steps: steps}
}

func hasSeverity(prioritized []PrioritizedError, severity ErrorSeverity) bool {
	for _, err := range prioritized {
		if err.Severity == severity {
			return true
		}
	}
	return false
}

func normalizeErrorMessage(message string) string {
	return strings.Join(strings.Fields(strings.ToLower(message)), " ")
}

func expandDisplayMessage(message string) []string {
	const schemaPrefix = "Multiple schema validation failures:"
	if !strings.Contains(message, schemaPrefix) {
		return []string{message}
	}

	prefix, remainder, found := strings.Cut(message, schemaPrefix)
	if !found {
		return []string{message}
	}

	var expanded []string
	prefix = strings.TrimRight(prefix, " ")
	if prefix != "" {
		prefix += " "
	}
	for line := range strings.SplitSeq(remainder, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "- ") {
			continue
		}
		expanded = append(expanded, prefix+strings.TrimPrefix(trimmed, "- "))
	}

	if len(expanded) == 0 {
		return []string{message}
	}

	return expanded
}

// Heading returns a human-friendly severity heading for terminal output.
func (s ErrorSeverity) Heading() string {
	switch s {
	case SeverityCritical:
		return "CRITICAL (fix first)"
	case SeverityHigh:
		return "HIGH PRIORITY"
	case SeverityMedium:
		return "MEDIUM PRIORITY"
	case SeverityLow:
		return "LOW PRIORITY"
	default:
		return "PRIORITY"
	}
}

// Icon returns a terminal-friendly severity icon.
func (s ErrorSeverity) Icon() string {
	switch s {
	case SeverityCritical:
		return "🔴"
	case SeverityHigh:
		return "🟠"
	case SeverityMedium:
		return "🟡"
	case SeverityLow:
		return "🔵"
	default:
		return "•"
	}
}
