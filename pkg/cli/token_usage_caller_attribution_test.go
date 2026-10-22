//go:build !integration

package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// piCallerLifecycle returns Pi parent-session lifecycle records for two named
// sub-agents that each made one request with the same model and token quartet.
func piCallerLifecycle() []string {
	lines := []string{
		`{"type":"session.init","data":{"sourceEngine":"pi","sessionId":"pi-parent"},"timestamp":"2026-10-21T03:49:50Z","provenance":{"component":"agent","phase":"agent"}}`,
	}
	for _, agent := range []struct{ name, start, request string }{
		{"a-reader", "2026-10-21T03:49:50.100Z", "2026-10-21T03:49:50.300Z"},
		{"z-reader", "2026-10-21T03:49:50.150Z", "2026-10-21T03:49:50.350Z"},
	} {
		lines = append(lines,
			fmt.Sprintf(`{"type":"subagent.started","agentId":"%[1]s-invocation","data":{"invocationId":"%[1]s-invocation","agentName":"%[1]s","model":"small","resolvedModel":"claude-haiku-4.5"},"timestamp":"%[2]s","provenance":{"component":"agent","phase":"agent"}}`, agent.name, agent.start),
			fmt.Sprintf(`{"type":"subagent.request","agentId":"%[1]s-invocation","timestamp":"%[2]s","data":{"invocationId":"%[1]s-invocation","agentName":"%[1]s","model":"claude-haiku-4-5-20251001","inputTokens":10,"outputTokens":1,"cacheReadTokens":0,"cacheWriteTokens":0},"provenance":{"component":"agent","phase":"agent"}}`, agent.name, agent.request),
			fmt.Sprintf(`{"type":"subagent.completed","agentId":"%[1]s-invocation","data":{"invocationId":"%[1]s-invocation","agentName":"%[1]s","outcome":"completed"},"timestamp":"2026-10-21T03:49:52Z","provenance":{"component":"agent","phase":"agent"}}`, agent.name),
		)
	}
	return lines
}

// callerTokenUsageEvent is the collector projection of an AWF v0.28.70
// token-usage record; caller is "" (no caller fields), "main" or an agent name.
func callerTokenUsageEvent(requestID, timestamp, caller string, aic float64) string {
	callerFields := ""
	switch caller {
	case "":
	case "main":
		callerFields = `"purpose":"agent","callerSource":"caller_header",`
	default:
		callerFields = fmt.Sprintf(`"purpose":"subagent","agentName":%q,"callerSource":"caller_header",`, caller)
	}
	return fmt.Sprintf(`{"type":"firewall.token_usage","timestamp":%q,"data":{"provider":"anthropic","model":"claude-haiku-4-5-20251001",%s"path":"/v1/messages?beta=true","xInitiator":"agent","requestId":%q,"status":200,"aic":%v,"usage":{"inputTokens":10,"outputTokens":1,"cacheReadInputTokens":0,"cacheCreationInputTokens":0}},"provenance":{"component":"firewall","phase":"agent"}}`, timestamp, callerFields, requestID, aic)
}

func writeCallerSession(t *testing.T, labelled bool) string {
	t.Helper()
	label := func(caller string) string {
		if labelled {
			return caller
		}
		return ""
	}
	lines := append([]string{strings.TrimSpace(subagentSessionHeader)}, piCallerLifecycle()...)
	lines = append(lines,
		`{"type":"firewall.token_usage","timestamp":"2026-10-21T03:49:50.000Z","data":{"provider":"copilot","model":"gpt-5.4-mini","purpose":"routing_classification","callerSource":"awf","path":"/chat/completions","requestId":"classifier-1","status":200,"aic":0.05,"usage":{"inputTokens":50,"outputTokens":5}},"provenance":{"component":"firewall","phase":"agent"}}`,
		// Main, a-reader and z-reader share the served model, token quartet and an
		// overlapping window; observation order deliberately disagrees with callers.
		callerTokenUsageEvent("main-1", "2026-10-21T03:49:50.400Z", label("main"), 0.1),
		callerTokenUsageEvent("z-1", "2026-10-21T03:49:50.410Z", label("z-reader"), 0.2),
		callerTokenUsageEvent("a-1", "2026-10-21T03:49:50.420Z", label("a-reader"), 0.4),
	)
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "usage"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "usage", "aw_session.jsonl"), []byte(strings.Join(lines, "\n")+"\n"), 0o600))
	return dir
}

func piCallerCredits(t *testing.T, runDir string) (map[string]float64, map[string]int, *TokenUsageSummary) {
	t.Helper()
	_, _, agents, found, err := readSessionSubagentModelsDetailed(runDir)
	require.NoError(t, err)
	require.True(t, found)
	summary := &TokenUsageSummary{AgentUsage: agents, AICFound: true, TotalAIC: 0.75}
	augmentSubagentModelAttribution(runDir, summary)
	credits := make(map[string]float64)
	requests := make(map[string]int)
	for _, agent := range summary.AgentUsage {
		credits[agent.AgentName] = agent.AIC
		requests[agent.AgentName] = agent.Requests
	}
	return credits, requests, summary
}

func modelRoutingCallerCredits(runDir string) map[string]float64 {
	entries := tokenUsageEntriesForRun(runDir)
	_, _, agents, _, _ := readSessionSubagentModelsDetailed(runDir)
	resolver := newModelIdentityResolver(runDir)
	agentSummary := &TokenUsageSummary{AgentUsage: mergeAgentUsageByDefinition(resolveAgentUsageModels(agents, resolver, tokenUsageEntryModels(entries)...))}
	matchPiAgentUsageCredits(agentSummary, entries, resolver)
	routing := &ModelRoutingSummary{}
	applyAgentUsageToModelRouting(routing, agentSummary.AgentUsage)
	credits := map[string]float64{"main": routing.MainAgentCost.AIC}
	for _, cost := range routing.SubagentCosts {
		credits[cost.AgentName] = cost.AIC
	}
	return credits
}

// TestProxyCallerPrecedesCorrelationAndPreservesFallback is the
// proxy_caller_precedes_correlation_and_preserves_fallback acceptance test.
func TestProxyCallerPrecedesCorrelationAndPreservesFallback(t *testing.T) {
	t.Run("proxy_caller_precedes_correlation_and_preserves_fallback", func(t *testing.T) {
		labelled := writeCallerSession(t, true)
		credits, requests, summary := piCallerCredits(t, labelled)
		assert.InDelta(t, 0.4, credits["a-reader"], 1e-9, "a-reader is credited only with its declared record")
		assert.InDelta(t, 0.2, credits["z-reader"], 1e-9, "z-reader is credited only with its declared record")
		assert.InDelta(t, 0.1, credits["main"], 1e-9, "the explicit main record is main even though it shares a sub-agent model")
		assert.Equal(t, 1, requests["main"])
		assert.Equal(t, 1, requests["a-reader"], "native sub-agent request counts are not double-counted")
		assert.Equal(t, 1, requests["z-reader"])
		assert.Empty(t, summary.Warnings, "explicit attribution reconciles with the non-classifier proxy total")
		assert.Equal(t, credits, modelRoutingCallerCredits(labelled), "audit and logs attribute the same credits")

		// Older records without caller fields keep the existing observation-order
		// correlation: a-reader takes the earliest matching record and the third
		// record, on a sub-agent model, is never inferred as main.
		unlabelled := writeCallerSession(t, false)
		credits, _, summary = piCallerCredits(t, unlabelled)
		assert.InDelta(t, 0.1, credits["a-reader"], 1e-9)
		assert.InDelta(t, 0.2, credits["z-reader"], 1e-9)
		assert.InDelta(t, 0.0, credits["main"], 1e-9)
		assert.NotEmpty(t, summary.Warnings, "heuristic attribution leaves the mismatch visible")
	})
}

// TestRawProxyCallerAttributionWithoutUnifiedTokenUsage covers the legacy raw
// accounting path with real token-usage/v0.28.70 record shapes from Smoke Pi
// Routed run 39540473269, extended with the caller fields AWF writes for an
// x-awf-caller declaration.
func TestRawProxyCallerAttributionWithoutUnifiedTokenUsage(t *testing.T) {
	dir := t.TempDir()
	lifecycle := []string{
		strings.TrimSpace(subagentSessionHeader),
		`{"type":"session.init","data":{"sourceEngine":"pi","sessionId":"pi-parent"},"timestamp":"2026-10-21T03:49:47Z","provenance":{"component":"agent","phase":"agent"}}`,
		`{"type":"subagent.started","agentId":"haiku","data":{"invocationId":"haiku","agentName":"haiku-whoami","model":"claude-haiku-4.5","resolvedModel":"claude-haiku-4-5-20251001"},"timestamp":"2026-10-21T03:49:50.800Z","provenance":{"component":"agent","phase":"agent"}}`,
		`{"type":"subagent.request","agentId":"haiku","timestamp":"2026-10-21T03:49:51.799Z","data":{"invocationId":"haiku","agentName":"haiku-whoami","model":"claude-haiku-4-5-20251001","inputTokens":900,"outputTokens":20,"cacheReadTokens":0,"cacheWriteTokens":0},"provenance":{"component":"agent","phase":"agent"}}`,
		`{"type":"subagent.completed","agentId":"haiku","data":{"invocationId":"haiku","agentName":"haiku-whoami","outcome":"completed"},"timestamp":"2026-10-21T03:49:52Z","provenance":{"component":"agent","phase":"agent"}}`,
	}
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "usage"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "usage", "aw_session.jsonl"), []byte(strings.Join(lifecycle, "\n")+"\n"), 0o600))
	raw := []string{
		`{"_schema":"token-usage/v0.28.70","timestamp":"2026-10-21T03:49:48.000Z","event":"token_usage","request_id":"classifier","purpose":"routing_classification","caller_source":"awf","provider":"copilot","model":"gpt-5.4-mini","path":"/chat/completions","status":200,"streaming":false,"input_tokens":400,"output_tokens":8,"cache_read_tokens":0,"cache_write_tokens":0,"duration_ms":300,"ai_credits_this_response":0.01}`,
		`{"_schema":"token-usage/v0.28.70","timestamp":"2026-10-21T03:49:50.709Z","event":"token_usage","request_id":"e92da47e-0815-427d-bb28-db4d501bcccf","purpose":"agent","caller_source":"caller_header","x_initiator":"agent","provider":"copilot","model":"gpt-5.6-luna","path":"/responses","status":200,"streaming":true,"input_tokens":5000,"output_tokens":120,"cache_read_tokens":0,"cache_write_tokens":0,"duration_ms":900,"ai_credits_this_response":0.3}`,
		`{"_schema":"token-usage/v0.28.70","timestamp":"2026-10-21T03:49:51.799Z","event":"token_usage","request_id":"511ef3bb-e6e6-43a5-9b28-8a6f4082a147","purpose":"subagent","agent_name":"haiku-whoami","caller_source":"caller_header","x_initiator":"agent","provider":"copilot","model":"claude-haiku-4-5-20251001","path":"/v1/messages?beta=true","status":200,"streaming":true,"input_tokens":900,"output_tokens":20,"cache_read_tokens":0,"cache_write_tokens":0,"duration_ms":400,"ai_credits_this_response":0.05}`,
		`{"_schema":"token-usage/v0.28.70","timestamp":"2026-10-21T03:49:58.208Z","event":"token_usage","request_id":"4d023499-e6e6-43a5-9b28-8a6f4082a147","purpose":"subagent","agent_name":"mini-whoami","caller_source":"caller_header","x_initiator":"agent","provider":"copilot","model":"gpt-5.3-codex","path":"/responses","status":200,"streaming":true,"input_tokens":700,"output_tokens":10,"cache_read_tokens":0,"cache_write_tokens":0,"duration_ms":350,"ai_credits_this_response":0.07}`,
	}
	rawDir := filepath.Join(dir, "sandbox", "firewall", "logs", "api-proxy-logs")
	require.NoError(t, os.MkdirAll(rawDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(rawDir, "token-usage.jsonl"), []byte(strings.Join(raw, "\n")+"\n"), 0o600))

	entries := rawCallerAttributedEntries(dir)
	require.Len(t, entries, 4)
	assert.Equal(t, TokenUsageEntry{}.AgentName, entries[0].AgentName, "AWF-owned records carry no agent name")
	assert.Equal(t, "awf", entries[0].CallerSource)
	assert.Equal(t, "haiku-whoami", entries[2].AgentName)

	credits, requests, summary := piCallerCredits(t, dir)
	assert.InDelta(t, 0.05, credits["haiku-whoami"], 1e-9)
	assert.InDelta(t, 0.3, credits["main"], 1e-9, "the classifier is never main")
	assert.Equal(t, 1, requests["main"])
	assert.Contains(t, summary.Warnings, "proxy token usage declares pi sub-agent mini-whoami without a matching sub-agent run; credits were not attributed")
}
