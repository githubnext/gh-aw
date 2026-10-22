//go:build !integration

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// modelResolutionReplayDir holds sanitized artifacts from real runs:
//   - 39499460077 (Smoke Copilot Sub Agents) and 39508150750 (the same workflow on main
//     after #70389): mini-whoami/gpt-5-mini and mini54-whoami/gpt-5.4-mini fail closed.
//   - 39499464895 (Smoke Copilot SDK Routed Claude): mini-whoami/gpt-5.4-mini fails;
//     no main record; router classifier traffic is present.
//   - 39508154330 (Smoke Copilot SDK Routed, successful): native session continuation
//     events only; it has no failed resolution record of its own.
//
// Each failing run keeps its session.format header, model.resolution, agent.execution,
// execution.result, and session.collection events, its wrapper agent_execution.json and
// error-summary-only agent-session.jsonl, its awf-reflect.json reduced to provider,
// configured state, model IDs, and routing endpoints, and its agent-stdio.log reduced to
// the harness model-resolution and sub-agent lines copied from the agent job log with
// timestamps removed. Native events from 39508154330
// keep their envelopes with data reduced to identifiers.
var modelResolutionReplayDir = filepath.Join("testdata", "model_resolution_startup")

func readModelResolutionReplayFile(t *testing.T, run, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(modelResolutionReplayDir, run, name))
	require.NoError(t, err)
	return string(data)
}

// writeModelResolutionReplayRun lays out a run's artifacts as audit downloads them.
// Overrides replace files by run-relative path; an empty override omits the file.
func writeModelResolutionReplayRun(t *testing.T, runDir, run string, overrides map[string]string) string {
	t.Helper()
	files := map[string]string{
		"usage/aw_session.jsonl":            readModelResolutionReplayFile(t, run, "aw_session.jsonl"),
		"sandbox/firewall/awf-reflect.json": readModelResolutionReplayFile(t, run, "awf-reflect.json"),
		"agent-session.jsonl":               readModelResolutionReplayFile(t, run, "agent-session.jsonl"),
		"agent_execution.json":              readModelResolutionReplayFile(t, run, "agent_execution.json"),
		"agent-stdio.log":                   readModelResolutionReplayFile(t, run, "agent-stdio.log"),
	}
	maps.Copy(files, overrides)
	for relative, content := range files {
		if content == "" {
			continue
		}
		writeModelAvailabilityEvidence(t, runDir, relative, content)
	}
	return runDir
}

func modelAvailabilityFindingsByCode(findings []AuditFinding) map[AuditFindingCode][]AuditFinding {
	byCode := make(map[AuditFindingCode][]AuditFinding)
	for _, finding := range findings {
		switch finding.Code {
		case AuditFindingModelUnavailable, AuditFindingModelAvailabilityUnknown, AuditFindingModelResolutionFailed:
			byCode[finding.Code] = append(byCode[finding.Code], finding)
		}
	}
	return byCode
}

func assertTwoRoleConfirmedFinding(t *testing.T, findings []AuditFinding) {
	t.Helper()
	require.Len(t, findings, 1)
	finding := findings[0]
	assert.Equal(t, AuditFindingModelUnavailable, finding.Code)
	assert.EqualValues(t, "high", finding.Severity)
	assert.Equal(t, "Model Unavailable", finding.Title)
	for _, text := range []string{
		`engine copilot role(s) "mini-whoami" configured model "gpt-5-mini" (provider copilot, recorded cause model_unavailable)`,
		`engine copilot role(s) "mini54-whoami" configured model "gpt-5.4-mini" (provider copilot, recorded cause model_unavailable)`,
		"models_fetch_complete=true; the configured copilot endpoint advertises 36 model(s), but not \"gpt-5-mini\"",
		"advertises 36 model(s), but not \"gpt-5.4-mini\"",
		"categories: model_not_supported_error",
		"no native agent session activity",
		"agent-stdio.log records a harness refusal or model-specific resolution error for each listed role",
	} {
		assert.Contains(t, finding.Description, text)
	}
	assert.NotContains(t, finding.Description, "haiku-whoami", "successful records must not be flagged")
	assert.NotContains(t, finding.Description, `role(s) "main"`, "successful records must not be flagged")
}

func TestAuditModelAvailabilityCopilotSDKTwoDeclaredSubagents(t *testing.T) {
	runDir := writeModelResolutionReplayRun(t, t.TempDir(), "39499460077", nil)
	findings := generateModelAvailabilityFindings(WorkflowRun{Conclusion: "failure", LogsPath: runDir})
	assertTwoRoleConfirmedFinding(t, findings)

	data, _ := buildLocalAuditData(ProcessedRun{Run: WorkflowRun{DatabaseID: 39499460077, Conclusion: "failure", LogsPath: runDir}}, LogMetrics{}, nil)
	byCode := modelAvailabilityFindingsByCode(data.KeyFindings)
	assert.Len(t, byCode[AuditFindingModelUnavailable], 1)
	assert.Empty(t, byCode[AuditFindingModelAvailabilityUnknown])
	assert.Empty(t, byCode[AuditFindingModelResolutionFailed])
}

func TestAuditModelAvailabilityRoutedCopilotSDKWithoutMainRecord(t *testing.T) {
	runDir := writeModelResolutionReplayRun(t, t.TempDir(), "39499464895", nil)
	session := readModelResolutionReplayFile(t, "39499464895", "aw_session.jsonl")
	require.Contains(t, session, `"purpose":"routing_classification"`, "fixture must retain router classifier traffic")
	require.NotContains(t, session, `"role":"main"`)

	findings := generateModelAvailabilityFindings(WorkflowRun{Conclusion: "failure", LogsPath: runDir})
	require.Len(t, findings, 1)
	assert.Equal(t, AuditFindingModelUnavailable, findings[0].Code)
	assert.Contains(t, findings[0].Description, `engine copilot role(s) "mini-whoami" configured model "gpt-5.4-mini" (provider copilot, recorded cause model_unavailable)`)
	assert.Contains(t, findings[0].Description, `but not "gpt-5.4-mini"`)
	assert.NotContains(t, findings[0].Description, "main")
}

func TestAuditModelAvailabilityCopilotSubagentsAfter70389(t *testing.T) {
	runDir := writeModelResolutionReplayRun(t, t.TempDir(), "39508150750", nil)
	var wrapper struct {
		State string `json:"state"`
	}
	require.NoError(t, json.Unmarshal([]byte(readModelResolutionReplayFile(t, "39508150750", "agent_execution.json")), &wrapper))
	require.Equal(t, "started", wrapper.State)
	require.Contains(t, readModelResolutionReplayFile(t, "39508150750", "agent-session.jsonl"), `"type":"agent.execution"`)

	require.Contains(t, readModelResolutionReplayFile(t, "39508150750", "agent-stdio.log"), "copilot sub-agent 'mini54-whoami': ")

	findings := generateModelAvailabilityFindings(WorkflowRun{Conclusion: "failure", LogsPath: runDir})
	assertTwoRoleConfirmedFinding(t, findings)
}

// TestAuditModelAvailabilityRepositoryAgentFailureOnSuccessIsAdvisory is a deliberate
// negative-control replay: the native continuation events of successful run 39508154330
// combined with an unchanged failed record from run 39499460077. Run 39508154330 did not
// itself record this failure.
func TestAuditModelAvailabilityRepositoryAgentFailureOnSuccessIsAdvisory(t *testing.T) {
	var failedRecord string
	for line := range strings.SplitSeq(readModelResolutionReplayFile(t, "39499460077", "aw_session.jsonl"), "\n") {
		if strings.Contains(line, `"role":"mini-whoami"`) {
			failedRecord = line
		}
	}
	require.NotEmpty(t, failedRecord)
	continuation := strings.TrimSuffix(readModelResolutionReplayFile(t, "39508154330", "aw_session.jsonl"), "\n")
	require.Contains(t, continuation, `"type":"session.start"`)
	session := continuation + "\n" + failedRecord + "\n"

	for _, conclusion := range []string{"success", "failure"} {
		t.Run(conclusion, func(t *testing.T) {
			runDir := writeModelResolutionReplayRun(t, t.TempDir(), "39499460077", map[string]string{
				"usage/aw_session.jsonl": session,
				// A continued run that later fails for an unrelated reason.
				"agent-stdio.log": "Error: unrelated tool failure after the agent started\n",
			})
			findings := generateModelAvailabilityFindings(WorkflowRun{Conclusion: conclusion, LogsPath: runDir})
			require.Len(t, findings, 1)
			assert.Equal(t, AuditFindingModelResolutionFailed, findings[0].Code)
			assert.EqualValues(t, "low", findings[0].Severity)
			assert.Contains(t, findings[0].Description, `role(s) "mini-whoami" configured model "gpt-5-mini"`)
			assert.Contains(t, findings[0].Description, "not treated as")
			assert.Contains(t, findings[0].Description, "optional repository agent")

			data, _ := buildLocalAuditData(ProcessedRun{Run: WorkflowRun{Conclusion: conclusion, Status: "completed", LogsPath: runDir}}, LogMetrics{}, nil)
			byCode := modelAvailabilityFindingsByCode(data.KeyFindings)
			assert.Empty(t, byCode[AuditFindingModelUnavailable])
			assert.Empty(t, byCode[AuditFindingModelAvailabilityUnknown])
			assert.Len(t, byCode[AuditFindingModelResolutionFailed], 1)
			for _, recommendation := range data.Recommendations {
				assert.NotContains(t, recommendation.Action, "Replace the unavailable model pin")
			}
		})
	}
}

func TestAuditModelAvailabilitySessionMissingReflection(t *testing.T) {
	// Derived from run 39499460077 by omitting its awf-reflect.json.
	runDir := writeModelResolutionReplayRun(t, t.TempDir(), "39499460077", map[string]string{"sandbox/firewall/awf-reflect.json": ""})
	findings := generateModelAvailabilityFindings(WorkflowRun{Conclusion: "failure", LogsPath: runDir})
	require.Len(t, findings, 1)
	assert.Equal(t, AuditFindingModelAvailabilityUnknown, findings[0].Code)
	assert.Equal(t, "Model Availability Unverified", findings[0].Title)
	assert.Contains(t, findings[0].Description, `role(s) "mini-whoami" configured model "gpt-5-mini"`)
	assert.Contains(t, findings[0].Description, `role(s) "mini54-whoami" configured model "gpt-5.4-mini"`)
	assert.Contains(t, findings[0].Description, "awf-reflect.json is unavailable")
}

func TestAuditModelAvailabilitySessionIncompleteReflection(t *testing.T) {
	// Derived from run 39499460077 by changing only models_fetch_complete to false.
	reflection := readModelResolutionReplayFile(t, "39499460077", "awf-reflect.json")
	require.Contains(t, reflection, `"models_fetch_complete": true`)
	reflection = strings.Replace(reflection, `"models_fetch_complete": true`, `"models_fetch_complete": false`, 1)
	runDir := writeModelResolutionReplayRun(t, t.TempDir(), "39499460077", map[string]string{"sandbox/firewall/awf-reflect.json": reflection})
	findings := generateModelAvailabilityFindings(WorkflowRun{Conclusion: "failure", LogsPath: runDir})
	require.Len(t, findings, 1)
	assert.Equal(t, AuditFindingModelAvailabilityUnknown, findings[0].Code)
	assert.Equal(t, "Model Availability Unverified", findings[0].Title)
	assert.Contains(t, findings[0].Description, "Model discovery did not complete")
}

func TestAuditModelAvailabilityGroupsRolesSharingModel(t *testing.T) {
	// Derived from run 39499460077: the recorded mini54-whoami envelope and fields are
	// kept, with its configured model changed to the model mini-whoami failed on.
	session := readModelResolutionReplayFile(t, "39499460077", "aw_session.jsonl")
	original := `"role":"mini54-whoami","configured_model":"gpt-5.4-mini"`
	require.Contains(t, session, original)
	session = strings.Replace(session, original, `"role":"mini54-whoami","configured_model":"gpt-5-mini"`, 1)
	runDir := writeModelResolutionReplayRun(t, t.TempDir(), "39499460077", map[string]string{"usage/aw_session.jsonl": session})

	findings := generateModelAvailabilityFindings(WorkflowRun{Conclusion: "failure", LogsPath: runDir})
	require.Len(t, findings, 1)
	assert.Equal(t, AuditFindingModelUnavailable, findings[0].Code)
	assert.Contains(t, findings[0].Description, `engine copilot role(s) "mini-whoami", "mini54-whoami" configured model "gpt-5-mini"`)
	assert.Equal(t, 1, strings.Count(findings[0].Description, `configured model "gpt-5-mini"`))
	assert.NotContains(t, findings[0].Description, "gpt-5.4-mini")
}

func TestAuditModelAvailabilityPreservesOtherResolutionCauses(t *testing.T) {
	// Derived from run 39499464895: the recorded mini-whoami failure with its cause
	// changed to the contract-valid catalog_unavailable. A legacy Pi startup line is
	// appended to the run's startup log and must not override the recorded cause.
	session := readModelResolutionReplayFile(t, "39499464895", "aw_session.jsonl")
	require.Contains(t, session, `"status":"failed","cause":"model_unavailable"`)
	session = strings.Replace(session, `"status":"failed","cause":"model_unavailable"`, `"status":"failed","cause":"catalog_unavailable"`, 1)
	runDir := writeModelResolutionReplayRun(t, t.TempDir(), "39499464895", map[string]string{
		"usage/aw_session.jsonl": session,
		"agent-stdio.log": readModelResolutionReplayFile(t, "39499464895", "agent-stdio.log") +
			`[gh-aw/pi-models-json] fatal: Pi model "gpt-5.4-mini" is not advertised by the configured copilot proxy endpoint; choose a model listed by AWF /reflect` + "\n",
	})

	findings := generateModelAvailabilityFindings(WorkflowRun{Conclusion: "failure", LogsPath: runDir})
	require.Len(t, findings, 1)
	assert.Equal(t, AuditFindingModelAvailabilityUnknown, findings[0].Code)
	assert.Contains(t, findings[0].Description, "recorded cause catalog_unavailable")
	assert.Contains(t, findings[0].Description, "not model_unavailable")
	assert.NotContains(t, findings[0].Description, "Pi startup")
	assert.NotContains(t, findings[0].Description, "recorded cause model_unavailable")
}

func TestAuditModelAvailabilityClassifiesUnsupportedEndpoint(t *testing.T) {
	// Derived from run 39499464895: the recorded mini-whoami failure with its cause changed
	// to unsupported_endpoint, the cause Codex sub-agents record for a live model without
	// /responses.
	session := readModelResolutionReplayFile(t, "39499464895", "aw_session.jsonl")
	require.Contains(t, session, `"status":"failed","cause":"model_unavailable"`)
	session = strings.Replace(session, `"status":"failed","cause":"model_unavailable"`, `"status":"failed","cause":"unsupported_endpoint"`, 1)
	runDir := writeModelResolutionReplayRun(t, t.TempDir(), "39499464895", map[string]string{"usage/aw_session.jsonl": session})

	findings := generateModelAvailabilityFindings(WorkflowRun{Conclusion: "failure", LogsPath: runDir})
	require.Len(t, findings, 1)
	assert.Equal(t, AuditFindingModelEndpointUnsupported, findings[0].Code)
	assert.Equal(t, "Model Endpoint Unsupported", findings[0].Title)
	assert.Contains(t, findings[0].Description, "recorded cause unsupported_endpoint")
	assert.Contains(t, findings[0].Description, "the model is live, but the engine cannot call its endpoint; choose a compatible model")
	assert.NotContains(t, findings[0].Description, "model absence cannot be confirmed")
	assert.Contains(t, findings[0].Impact, "Choose a model that supports the engine's endpoint")

	recommendations := appendModelAvailabilityRecommendations(nil, findings)
	require.Len(t, recommendations, 1)
	assert.Contains(t, recommendations[0].Action, "Choose a model that supports the endpoint the engine calls")
	assert.NotContains(t, recommendations[0].Action, "Inspect same-run model discovery")
}

func TestAuditModelAvailabilityLegacyFallbackRequiresNoFailedResolution(t *testing.T) {
	// A session whose only resolution record succeeded leaves the Pi log as the evidence.
	session := `{"type":"session.format","data":{"version":1},"provenance":{"component":"collector","phase":"conclusion","path":"usage/aw_session.jsonl","index":0}}
{"type":"model.resolution","data":{"engine":"pi","role":"main","configured_model":"claude-opus-5","resolved_model":"claude-opus-5","provider":"copilot","status":"exact","cause":null},"provenance":{"component":"model_resolution","phase":"agent","path":"agent/model-resolution.json","index":0}}
`
	runDir := t.TempDir()
	writeModelAvailabilityEvidence(t, runDir, "usage/aw_session.jsonl", session)
	writeModelAvailabilityEvidence(t, runDir, "agent-stdio.log", modelUnavailableStartupLog)
	writeModelAvailabilityEvidence(t, runDir, "sandbox/firewall/awf-reflect.json",
		`{"models_fetch_complete":true,"endpoints":[{"provider":"copilot","configured":true,"models":["claude-opus-5.5"]}]}`)
	findings := generateModelAvailabilityFindings(WorkflowRun{Conclusion: "failure", LogsPath: runDir})
	require.Len(t, findings, 1)
	assert.Equal(t, AuditFindingModelUnavailable, findings[0].Code)
	assert.Contains(t, findings[0].Description, "Pi startup rejected model")
}

func TestAuditModelAvailabilityIgnoresInvalidResolutionSessions(t *testing.T) {
	failed := `{"type":"model.resolution","data":{"engine":"copilot","role":"mini-whoami","configured_model":"gpt-5-mini","resolved_model":null,"provider":"copilot","status":"failed","cause":"model_unavailable"},"provenance":{"component":"model_resolution","phase":"agent","path":"agent/model-resolution.json","index":2}}`
	for name, session := range map[string]string{
		"missing header":     failed + "\n",
		"unsupported format": `{"type":"session.format","data":{"version":2},"provenance":{"component":"collector"}}` + "\n" + failed + "\n",
		"wrong provenance":   `{"type":"session.format","data":{"version":1},"provenance":{"component":"collector"}}` + "\n" + strings.Replace(failed, `"component":"model_resolution"`, `"component":"agent"`, 1) + "\n",
		"unsafe role":        `{"type":"session.format","data":{"version":1},"provenance":{"component":"collector"}}` + "\n" + strings.Replace(failed, `"mini-whoami"`, `"mini\nsensitive-payload"`, 1) + "\n",
		"resolved model":     `{"type":"session.format","data":{"version":1},"provenance":{"component":"collector"}}` + "\n" + strings.Replace(failed, `"resolved_model":null`, `"resolved_model":"gpt-5-mini"`, 1) + "\n",
	} {
		t.Run(name, func(t *testing.T) {
			runDir := t.TempDir()
			writeModelAvailabilityEvidence(t, runDir, "usage/aw_session.jsonl", session)
			assert.Empty(t, generateModelAvailabilityFindings(WorkflowRun{Conclusion: "failure", LogsPath: runDir}))
		})
	}
}

// TestAuditModelAvailabilityRepositoryAgentWarningThenUnrelatedStartupFailure is derived
// from run 39499464895: its recorded mini-whoami failure is kept, but the startup log is
// replaced with the harness's optional repository-agent warning followed by an unrelated
// provider-listener readiness failure, and the agent error categories are cleared because
// the harness suppresses model-not-supported errors for repository agents.
func TestAuditModelAvailabilityRepositoryAgentWarningThenUnrelatedStartupFailure(t *testing.T) {
	session := readModelResolutionReplayFile(t, "39499464895", "aw_session.jsonl")
	require.Contains(t, session, `"categories":["model_not_supported_error"]`)
	session = strings.Replace(session, `"categories":["model_not_supported_error"]`, `"categories":[]`, 1)
	runDir := writeModelResolutionReplayRun(t, t.TempDir(), "39499464895", map[string]string{
		"usage/aw_session.jsonl": session,
		"agent-stdio.log": "[copilot-harness] warning: repository agent 'mini-whoami' (.github/agents/mini-whoami.agent.md) declares model 'gpt-5.4-mini', which could not be resolved (cause: model_unavailable); the SDK session gives it the session model.\n" +
			"[copilot-harness] copilot-sdk driver mode: provider listener readiness probe failed for http://api-proxy:10002: connect ECONNREFUSED\n",
	})

	findings := generateModelAvailabilityFindings(WorkflowRun{Conclusion: "failure", LogsPath: runDir})
	require.Len(t, findings, 1)
	assert.Equal(t, AuditFindingModelResolutionFailed, findings[0].Code)
	assert.EqualValues(t, "low", findings[0].Severity)
	assert.Contains(t, findings[0].Description, `role(s) "mini-whoami" configured model "gpt-5.4-mini"`)
	assert.Contains(t, findings[0].Description, "no harness refusal or model-specific execution error")

	data, _ := buildLocalAuditData(ProcessedRun{Run: WorkflowRun{Conclusion: "failure", Status: "completed", LogsPath: runDir}}, LogMetrics{}, nil)
	for _, recommendation := range data.Recommendations {
		assert.NotContains(t, recommendation.Action, "Replace the unavailable model pin")
	}
}

func TestAuditModelAvailabilityOnlyRolesWithStopEvidenceAreAttributed(t *testing.T) {
	// Derived from run 39499460077 by removing mini54-whoami's harness refusal line.
	var lines []string
	for line := range strings.SplitSeq(readModelResolutionReplayFile(t, "39499460077", "agent-stdio.log"), "\n") {
		if !strings.Contains(line, "sub-agent 'mini54-whoami': ") {
			lines = append(lines, line)
		}
	}
	runDir := writeModelResolutionReplayRun(t, t.TempDir(), "39499460077", map[string]string{"agent-stdio.log": strings.Join(lines, "\n")})

	byCode := modelAvailabilityFindingsByCode(generateModelAvailabilityFindings(WorkflowRun{Conclusion: "failure", LogsPath: runDir}))
	require.Len(t, byCode[AuditFindingModelUnavailable], 1)
	assert.Contains(t, byCode[AuditFindingModelUnavailable][0].Description, `role(s) "mini-whoami"`)
	assert.NotContains(t, byCode[AuditFindingModelUnavailable][0].Description, "mini54-whoami")
	require.Len(t, byCode[AuditFindingModelResolutionFailed], 1)
	assert.Contains(t, byCode[AuditFindingModelResolutionFailed][0].Description, `role(s) "mini54-whoami"`)
	assert.NotContains(t, byCode[AuditFindingModelResolutionFailed][0].Description, `"mini-whoami"`)
}

func TestAuditModelAvailabilitySessionInitCountsAsNativeActivity(t *testing.T) {
	// Derived from run 39499460077 by adding an initialization-only native trace, as
	// Claude and Codex emit before failing ahead of their first assistant or tool event.
	session := readModelResolutionReplayFile(t, "39499460077", "aw_session.jsonl") +
		`{"type":"session.init","data":{"model":"gpt-5.3-codex"},"provenance":{"component":"agent","phase":"agent","path":"agent-session.jsonl","index":0}}` + "\n"
	runDir := writeModelResolutionReplayRun(t, t.TempDir(), "39499460077", map[string]string{"usage/aw_session.jsonl": session})

	findings := generateModelAvailabilityFindings(WorkflowRun{Conclusion: "failure", LogsPath: runDir})
	require.Len(t, findings, 1)
	assert.Equal(t, AuditFindingModelResolutionFailed, findings[0].Code)
	assert.Contains(t, findings[0].Description, "session.init")
}

func TestAuditModelAvailabilityAcceptsMultiParameterModelSelectors(t *testing.T) {
	// Derived from run 39499460077 with mini-whoami's configured model replaced by a
	// supported multi-parameter selector in both the record and the startup log.
	const model = "openai/gpt-6.1-sol?effort=high&temperature=0.2"
	session := strings.Replace(readModelResolutionReplayFile(t, "39499460077", "aw_session.jsonl"),
		`"role":"mini-whoami","configured_model":"gpt-5-mini"`, `"role":"mini-whoami","configured_model":"`+model+`"`, 1)
	stdio := readModelResolutionReplayFile(t, "39499460077", "agent-stdio.log")
	stdio = strings.ReplaceAll(stdio, "gpt-5-mini", model)
	runDir := writeModelResolutionReplayRun(t, t.TempDir(), "39499460077", map[string]string{
		"usage/aw_session.jsonl": session,
		"agent-stdio.log":        stdio,
	})

	evidence, err := readSessionModelResolutionEvidence(runDir)
	require.NoError(t, err)
	require.Len(t, evidence.Failures, 2)
	assert.Equal(t, model, evidence.Failures[0].ConfiguredModel)

	findings := generateModelAvailabilityFindings(WorkflowRun{Conclusion: "failure", LogsPath: runDir})
	var descriptions []string
	for _, finding := range findings {
		assert.NotEqual(t, AuditFindingModelResolutionFailed, finding.Code, "both roles have harness refusals")
		descriptions = append(descriptions, finding.Description)
	}
	assert.Contains(t, strings.Join(descriptions, "\n"), `role(s) "mini-whoami" configured model "`+model+`"`)
}

func TestAuditModelAvailabilityRejectsDuplicateAgentExecution(t *testing.T) {
	// Derived from run 39499460077 by appending a replayed exit-0 agent.execution record.
	session := readModelResolutionReplayFile(t, "39499460077", "aw_session.jsonl") +
		`{"type":"agent.execution","data":{"categories":[],"errorCodes":[],"errorTypes":[],"exitCode":0},"provenance":{"component":"execution","phase":"agent","path":"agent-errors.jsonl","index":1}}` + "\n"
	runDir := writeModelResolutionReplayRun(t, t.TempDir(), "39499460077", map[string]string{"usage/aw_session.jsonl": session})

	_, err := readSessionModelResolutionEvidence(runDir)
	require.ErrorContains(t, err, "multiple agent.execution records")
	assert.Empty(t, generateModelAvailabilityFindings(WorkflowRun{Conclusion: "failure", LogsPath: runDir}))
}

func TestAuditModelAvailabilityBoundsUnifiedSessionSize(t *testing.T) {
	session := readModelResolutionReplayFile(t, "39499460077", "aw_session.jsonl")
	evidence, err := parseSessionModelResolutionEvidence(strings.NewReader(session), int64(len(session)))
	require.NoError(t, err)
	assert.Len(t, evidence.Failures, 2)

	_, err = parseSessionModelResolutionEvidence(strings.NewReader(session), int64(len(session)-1))
	require.ErrorContains(t, err, "exceeds the")
}

func TestAuditModelAvailabilityReadsOnlyCanonicalSessionPath(t *testing.T) {
	runDir := t.TempDir()
	writeModelAvailabilityEvidence(t, runDir, "aw_session.jsonl", readModelResolutionReplayFile(t, "39499460077", "aw_session.jsonl"))
	evidence, err := readSessionModelResolutionEvidence(runDir)
	require.NoError(t, err)
	assert.Nil(t, evidence, "a root-level aw_session.jsonl is not the canonical usage artifact")
}

func installModelResolutionAuditFakeGH(t *testing.T, runID int64) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake gh is not supported on Windows")
	}
	dir := t.TempDir()
	script := fmt.Sprintf(`#!/bin/sh
case "$*" in
  *actions/runs/%[1]d/artifacts*) echo '[]' ;;
  *actions/runs/%[1]d*)
    echo '{"databaseId":%[1]d,"workflowName":"Smoke Copilot Sub Agents","workflowPath":".github/workflows/smoke-copilot-sub-agents.lock.yml","status":"completed","conclusion":"failure"}' ;;
  *actions/workflows/*/runs*) echo '[]' ;;
  "run download "*) echo 'no valid artifacts found' >&2; exit 1 ;;
  "run view "*) echo '{"jobs":[]}' ;;
  *) echo "unexpected gh invocation: $*" >&2; exit 1 ;;
esac
`, runID)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "gh"), []byte(script), 0o755))
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")
}

// TestAuditModelAvailabilityRun39499460077EndToEnd runs the audit command pipeline
// against run 39499460077's replayed artifacts with a fake gh, then again from cache.
func TestAuditModelAvailabilityRun39499460077EndToEnd(t *testing.T) {
	const runID int64 = 39499460077
	installModelResolutionAuditFakeGH(t, runID)
	outputDir := t.TempDir()
	runDir := writeModelResolutionReplayRun(t, filepath.Join(outputDir, fmt.Sprintf("run-%d", runID)), "39499460077", nil)

	audit := func() AuditData {
		t.Helper()
		stdout, _ := captureOutput(t, func() error {
			skipped, err := AuditWorkflowRun(context.Background(), runID, AuditOptions{
				Hostname: "github.com", Owner: "github", Repo: "gh-aw", OutputDir: outputDir, JSONOutput: true,
			})
			assert.False(t, skipped)
			require.NoError(t, err)
			return nil
		})
		var data AuditData
		require.NoError(t, json.Unmarshal([]byte(stdout), &data), stdout)
		return data
	}

	fresh := audit()
	assert.Equal(t, auditCacheSourceFull, fresh.CacheSource)
	assertTwoRoleConfirmedFinding(t, modelAvailabilityFindingsByCode(fresh.KeyFindings)[AuditFindingModelUnavailable])
	assert.Empty(t, modelAvailabilityFindingsByCode(fresh.KeyFindings)[AuditFindingModelAvailabilityUnknown])
	var recommended bool
	for _, recommendation := range fresh.Recommendations {
		recommended = recommended || strings.Contains(recommendation.Action, "Replace the unavailable model pin")
	}
	assert.True(t, recommended)
	assert.FileExists(t, filepath.Join(runDir, runSummaryFileName))
	assert.FileExists(t, filepath.Join(runDir, "audit.json"))

	cached := audit()
	assert.Equal(t, fresh.KeyFindings, cached.KeyFindings, "cached audit must preserve the finding")

	output := captureStderrForGuardPolicyReportTest(func() { renderConsole(cached, runDir) })
	assert.Contains(t, output, "Model Unavailable")
	assert.Contains(t, output, "mini54-whoami")
	assert.Contains(t, output, "debug-playbook.md#model-availability")
}

// TestAuditModelAvailabilityUpgradeRegeneratesOlderCachedAudit seeds run 39499460077's
// output directory with a schema 9 audit.json written before unified-session model
// resolution analysis existed. The upgraded audit must regenerate it, while a report
// at the current schema is still reused as-is.
func TestAuditModelAvailabilityUpgradeRegeneratesOlderCachedAudit(t *testing.T) {
	const runID int64 = 39499460077
	installModelResolutionAuditFakeGH(t, runID)

	for _, tc := range []struct {
		name          string
		schemaVersion int
		regenerated   bool
	}{
		{name: "schema 9 report", schemaVersion: 9, regenerated: true},
		{name: "current schema report", schemaVersion: auditSchemaVersion, regenerated: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			outputDir := t.TempDir()
			runDir := writeModelResolutionReplayRun(t, filepath.Join(outputDir, fmt.Sprintf("run-%d", runID)), "39499460077", nil)
			audit := func() AuditData {
				t.Helper()
				stdout, _ := captureOutput(t, func() error {
					_, err := AuditWorkflowRun(context.Background(), runID, AuditOptions{
						Hostname: "github.com", Owner: "github", Repo: "gh-aw", OutputDir: outputDir, JSONOutput: true,
					})
					require.NoError(t, err)
					return nil
				})
				var data AuditData
				require.NoError(t, json.Unmarshal([]byte(stdout), &data), stdout)
				return data
			}

			// Build a matching report, then replace it with an older-shaped one that has
			// no model resolution findings.
			old := audit()
			old.SchemaVersion = tc.schemaVersion
			old.KeyFindings = []AuditFinding{{Title: "stale cached finding"}}
			old.Recommendations = nil
			encoded, err := json.Marshal(old)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(auditPath(runDir), encoded, 0o600))

			upgraded := audit()
			byCode := modelAvailabilityFindingsByCode(upgraded.KeyFindings)
			if tc.regenerated {
				assertTwoRoleConfirmedFinding(t, byCode[AuditFindingModelUnavailable])
				for _, finding := range upgraded.KeyFindings {
					assert.NotEqual(t, "stale cached finding", finding.Title)
				}
				written, err := os.ReadFile(auditPath(runDir))
				require.NoError(t, err)
				var cached AuditData
				require.NoError(t, json.Unmarshal(written, &cached))
				assert.Equal(t, auditSchemaVersion, cached.SchemaVersion)
				assert.Len(t, modelAvailabilityFindingsByCode(cached.KeyFindings)[AuditFindingModelUnavailable], 1)
			} else {
				assert.Empty(t, byCode[AuditFindingModelUnavailable])
				require.Len(t, upgraded.KeyFindings, 1)
				assert.Equal(t, "stale cached finding", upgraded.KeyFindings[0].Title)
			}
		})
	}
}
