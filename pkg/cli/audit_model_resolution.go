package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// Model resolution records are runner-written before the engine starts (T-UAS-074).
// Identifiers are validated before they are quoted into findings.
var (
	modelResolutionEnginePattern   = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)
	modelResolutionRolePattern     = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9 ._:@/-]{0,127}$`)
	modelResolutionModelPattern    = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._/?=+:@-]{0,199}$`)
	modelResolutionProviderPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)
	modelResolutionCausePattern    = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)
)

const modelResolutionCauseModelUnavailable = "model_unavailable"

// sessionModelResolutionFailure is one failed model.resolution record. The record
// does not persist whether the role was a declared sub-agent or an optional
// repository agent, so it cannot by itself establish why a run failed.
type sessionModelResolutionFailure struct {
	Engine          string
	Role            string
	ConfiguredModel string
	Provider        string
	Cause           string
}

type sessionAgentExecutionSummary struct {
	Categories []string `json:"categories"`
	ExitCode   *int     `json:"exitCode"`
}

// sessionModelResolutionEvidence holds failed resolution records plus the startup
// and continuation evidence needed to decide whether they explain the run outcome.
type sessionModelResolutionEvidence struct {
	Failures []sessionModelResolutionFailure
	// NativeAgentActivity is true when the unified session contains native agent
	// session.start, assistant, or tool events. Wrapper state, router traffic, and
	// harness-written agent records are not native activity.
	NativeAgentActivity bool
	AgentExecution      *sessionAgentExecutionSummary
}

type sessionModelResolutionEvent struct {
	Type       string          `json:"type"`
	Data       json.RawMessage `json:"data"`
	Provenance struct {
		Component string `json:"component"`
		Phase     string `json:"phase"`
		Path      string `json:"path"`
	} `json:"provenance"`
}

type sessionModelResolutionPayload struct {
	Engine          string           `json:"engine"`
	Role            string           `json:"role"`
	ConfiguredModel string           `json:"configured_model"`
	ResolvedModel   json.RawMessage  `json:"resolved_model"`
	Provider        *json.RawMessage `json:"provider"`
	Status          string           `json:"status"`
	Cause           *string          `json:"cause"`
}

func readSessionModelResolutionEvidence(runDir string) (*sessionModelResolutionEvidence, error) {
	if runDir == "" {
		return nil, nil
	}
	for _, relative := range []string{"usage/aw_session.jsonl", "aw_session.jsonl"} {
		path := filepath.Join(runDir, filepath.FromSlash(relative))
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("%s: %w", relative, err)
		}
		if err := validateSubagentSessionSource(path); err != nil {
			return nil, fmt.Errorf("%s: %w", relative, err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("%s: unified session is not a regular file", relative)
		}
		evidence, err := parseSessionModelResolutionFile(path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", relative, err)
		}
		return evidence, nil
	}
	return nil, nil
}

func parseSessionModelResolutionFile(path string) (evidence *sessionModelResolutionEvidence, err error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			evidence, err = nil, closeErr
		}
	}()
	return parseSessionModelResolutionEvidence(file)
}

func parseSessionModelResolutionEvidence(input io.Reader) (*sessionModelResolutionEvidence, error) {
	reader := bufio.NewReader(input)
	evidence := &sessionModelResolutionEvidence{}
	seen := make(map[sessionModelResolutionFailure]struct{})
	headerSeen := false
	for lineNumber := 1; ; lineNumber++ {
		line, oversized, readErr := readUnifiedSessionLine(reader)
		if oversized {
			if !headerSeen {
				return nil, fmt.Errorf("oversized unified session header on line %d", lineNumber)
			}
			auditReportLog.Printf("Skipping oversized unified session record on line %d", lineNumber)
		} else if line = bytes.TrimSpace(line); len(line) > 0 {
			if !headerSeen {
				var header unifiedDetectionEvent
				if err := json.Unmarshal(line, &header); err != nil {
					return nil, fmt.Errorf("invalid unified session header on line %d", lineNumber)
				}
				if err := validateUnifiedDetectionHeader(header); err != nil {
					return nil, err
				}
				headerSeen = true
			} else {
				evidence.observe(line, lineNumber, seen)
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			return nil, fmt.Errorf("failed to read unified session: %w", readErr)
		}
	}
	if !headerSeen {
		return nil, errors.New("unified session is missing its leading session.format header")
	}
	return evidence, nil
}

func (evidence *sessionModelResolutionEvidence) observe(line []byte, lineNumber int, seen map[sessionModelResolutionFailure]struct{}) {
	var event sessionModelResolutionEvent
	if err := json.Unmarshal(line, &event); err != nil {
		auditReportLog.Printf("Skipping malformed unified session event on line %d", lineNumber)
		return
	}
	provenance := event.Provenance
	switch {
	case event.Type == "model.resolution":
		if provenance.Component != "model_resolution" || provenance.Phase != "agent" || provenance.Path != "agent/model-resolution.json" {
			return
		}
		failure, failed, valid := decodeFailedModelResolution(event.Data)
		if !valid {
			auditReportLog.Printf("Ignoring invalid model.resolution on line %d", lineNumber)
			return
		}
		if !failed {
			return
		}
		if _, duplicate := seen[failure]; duplicate {
			return
		}
		seen[failure] = struct{}{}
		evidence.Failures = append(evidence.Failures, failure)
	case event.Type == "agent.execution" && provenance.Component == "execution" && provenance.Phase == "agent":
		var summary sessionAgentExecutionSummary
		if json.Unmarshal(event.Data, &summary) == nil && summary.ExitCode != nil {
			evidence.AgentExecution = &summary
		}
	case provenance.Component == "agent" && provenance.Phase == "agent" &&
		(event.Type == "session.start" || strings.HasPrefix(event.Type, "assistant.") || strings.HasPrefix(event.Type, "tool.")):
		evidence.NativeAgentActivity = true
	}
}

// decodeFailedModelResolution validates a model.resolution payload against the
// T-UAS-074 contract. It reports failed=true only for status "failed" records.
func decodeFailedModelResolution(data json.RawMessage) (sessionModelResolutionFailure, bool, bool) {
	var payload sessionModelResolutionPayload
	if !bytes.HasPrefix(bytes.TrimSpace(data), []byte("{")) || json.Unmarshal(data, &payload) != nil {
		return sessionModelResolutionFailure{}, false, false
	}
	if payload.Status != "failed" {
		return sessionModelResolutionFailure{}, false, true
	}
	if !modelResolutionEnginePattern.MatchString(payload.Engine) ||
		!modelResolutionRolePattern.MatchString(payload.Role) ||
		!modelResolutionModelPattern.MatchString(payload.ConfiguredModel) ||
		payload.Cause == nil || !modelResolutionCausePattern.MatchString(*payload.Cause) ||
		!bytes.Equal(bytes.TrimSpace(payload.ResolvedModel), []byte("null")) {
		return sessionModelResolutionFailure{}, false, false
	}
	provider := ""
	if payload.Provider != nil && !bytes.Equal(bytes.TrimSpace(*payload.Provider), []byte("null")) {
		if json.Unmarshal(*payload.Provider, &provider) != nil || !modelResolutionProviderPattern.MatchString(provider) {
			return sessionModelResolutionFailure{}, false, false
		}
	}
	return sessionModelResolutionFailure{
		Engine:          payload.Engine,
		Role:            payload.Role,
		ConfiguredModel: payload.ConfiguredModel,
		Provider:        provider,
		Cause:           *payload.Cause,
	}, true, true
}

// modelResolutionFailureGroup lists the roles that failed on one engine/provider/model/cause.
type modelResolutionFailureGroup struct {
	Engine          string
	Provider        string
	ConfiguredModel string
	Cause           string
	Roles           []string
}

func groupModelResolutionFailures(failures []sessionModelResolutionFailure) []modelResolutionFailureGroup {
	var groups []*modelResolutionFailureGroup
	for _, failure := range failures {
		var group *modelResolutionFailureGroup
		for _, candidate := range groups {
			if candidate.Engine == failure.Engine && candidate.Provider == failure.Provider &&
				candidate.ConfiguredModel == failure.ConfiguredModel && candidate.Cause == failure.Cause {
				group = candidate
				break
			}
		}
		if group == nil {
			group = &modelResolutionFailureGroup{
				Engine: failure.Engine, Provider: failure.Provider,
				ConfiguredModel: failure.ConfiguredModel, Cause: failure.Cause,
			}
			groups = append(groups, group)
		}
		if !slices.Contains(group.Roles, failure.Role) {
			group.Roles = append(group.Roles, failure.Role)
		}
	}
	result := make([]modelResolutionFailureGroup, 0, len(groups))
	for _, group := range groups {
		result = append(result, *group)
	}
	return result
}

func (group modelResolutionFailureGroup) describe() string {
	roles := make([]string, 0, len(group.Roles))
	for _, role := range group.Roles {
		roles = append(roles, fmt.Sprintf("%q", role))
	}
	provider := group.Provider
	if provider == "" {
		provider = "(not recorded)"
	}
	return fmt.Sprintf("engine %s role(s) %s configured model %q (provider %s, recorded cause %s)",
		group.Engine, strings.Join(roles, ", "), group.ConfiguredModel, provider, group.Cause)
}

func describeModelResolutionGroups(groups []modelResolutionFailureGroup) string {
	descriptions := make([]string, 0, len(groups))
	for _, group := range groups {
		descriptions = append(descriptions, group.describe())
	}
	return strings.Join(descriptions, "; ")
}

func countModelResolutionRoles(groups []modelResolutionFailureGroup) int {
	var roles []string
	for _, group := range groups {
		for _, role := range group.Roles {
			if !slices.Contains(roles, role) {
				roles = append(roles, role)
			}
		}
	}
	return len(roles)
}

// modelResolutionContinuationReason explains why failed records are not attributed
// as the run's failure cause. It returns "" only when startup evidence supports a
// fatal startup stop: a failed run, a failed agent execution, and no native agent activity.
func modelResolutionContinuationReason(run WorkflowRun, evidence *sessionModelResolutionEvidence) string {
	switch {
	case run.Conclusion != "failure":
		conclusion := run.Conclusion
		if conclusion == "" {
			conclusion = "unknown"
		}
		return fmt.Sprintf("The run conclusion is %q, so these records are not treated as a failure cause.", conclusion)
	case evidence.NativeAgentActivity:
		return "The unified session contains native agent session activity (session.start, assistant, or tool events), so the run continued past model resolution and these records are not treated as its failure cause."
	case evidence.AgentExecution == nil:
		return "The unified session has no agent execution record, so a startup stop caused by these records cannot be established."
	case *evidence.AgentExecution.ExitCode == 0:
		return "The agent execution record exited with code 0, so these records are not treated as the failure cause."
	default:
		return ""
	}
}

func modelResolutionStartupEvidence(evidence *sessionModelResolutionEvidence, roleCount int) string {
	var categories []string
	for _, category := range evidence.AgentExecution.Categories {
		if modelResolutionCausePattern.MatchString(category) && !slices.Contains(categories, category) {
			categories = append(categories, category)
		}
	}
	text := fmt.Sprintf(" Startup evidence: agent execution exited with code %d", *evidence.AgentExecution.ExitCode)
	if len(categories) > 0 {
		text += " (categories: " + strings.Join(categories, ", ") + ")"
	}
	text += " and the unified session contains no native agent session activity."
	if !slices.Contains(categories, "model_not_supported_error") {
		text += " The agent error summary does not report model_not_supported_error, so attribution relies on the absence of native agent activity."
	}
	if roleCount > 1 {
		text += fmt.Sprintf(" Resolution records do not persist whether a role is a workflow-declared sub-agent (fatal) or an optional repository agent (warning only), so the session cannot identify which of these %d failed roles stopped startup.", roleCount)
	}
	return text
}

func generateSessionModelResolutionFindings(run WorkflowRun, evidence *sessionModelResolutionEvidence) []AuditFinding {
	groups := groupModelResolutionFailures(evidence.Failures)
	if reason := modelResolutionContinuationReason(run, evidence); reason != "" {
		return []AuditFinding{{
			Code:        AuditFindingModelResolutionFailed,
			Category:    "configuration",
			Severity:    "low",
			Title:       "Model Resolution Failure Recorded",
			Description: "The unified session records failed model resolution: " + describeModelResolutionGroups(groups) + ". " + reason + " A failed record can come from an optional repository agent, whose unresolved model is only a warning.",
			Impact:      "Advisory: verify the listed model pins. This record alone does not establish that model resolution stopped the run.",
		}}
	}

	confirmed, unverified := classifyModelResolutionGroups(run, groups)
	startup := modelResolutionStartupEvidence(evidence, countModelResolutionRoles(groups))
	var findings []AuditFinding
	if len(confirmed) > 0 {
		findings = append(findings, AuditFinding{
			Code:        AuditFindingModelUnavailable,
			Category:    "configuration",
			Severity:    "high",
			Title:       "Model Unavailable",
			Description: "Startup model resolution failed before the agent started inference. " + strings.Join(confirmed, " ") + startup,
			Impact:      "The agent could not start inference. This evidence does not establish global model retirement or why provider availability changed.",
		})
	}
	if len(unverified) > 0 {
		findings = append(findings, AuditFinding{
			Code:        AuditFindingModelAvailabilityUnknown,
			Category:    "configuration",
			Severity:    "high",
			Title:       "Model Availability Unverified",
			Description: "Startup model resolution failed before the agent started inference. " + strings.Join(unverified, " ") + startup,
			Impact:      "Inspect model discovery and authentication evidence before changing the model or runtime.",
		})
	}
	return findings
}

// classifyModelResolutionGroups checks each failure group against this run's own
// inventory, keeping the recorded cause when it is not model_unavailable.
func classifyModelResolutionGroups(run WorkflowRun, groups []modelResolutionFailureGroup) (confirmed, unverified []string) {
	var reflection []byte
	var reflectionErr error
	reflectionRead := false
	for _, group := range groups {
		var detail string
		isConfirmed := false
		switch {
		case group.Cause != modelResolutionCauseModelUnavailable:
			detail = fmt.Sprintf("the harness recorded cause %s, not model_unavailable, so this run's inventory is not used to confirm absence", group.Cause)
		case group.Provider == "":
			detail = "no provider was recorded, so no configured endpoint inventory can be matched"
		default:
			if !reflectionRead {
				reflection, reflectionErr = readModelAvailabilityReflection(run.LogsPath)
				reflectionRead = true
			}
			isConfirmed, detail = checkModelAvailability(reflection, reflectionErr, group.Provider, group.ConfiguredModel)
		}
		if isConfirmed {
			confirmed = append(confirmed, group.describe()+": "+detail)
		} else {
			unverified = append(unverified, group.describe()+": "+detail+"; model absence cannot be confirmed from this run's inventory.")
		}
	}
	return confirmed, unverified
}
