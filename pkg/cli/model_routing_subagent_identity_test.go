package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func modelRoutingGoldenVariantDir(t *testing.T, caseName string, legacy bool) string {
	t.Helper()
	runDir := filepath.Join(t.TempDir(), caseName)
	copyModelRoutingGoldenFixture(t, filepath.Join("testdata", "model_routing_golden", caseName), runDir)
	if legacy {
		require.NoError(t, os.Remove(filepath.Join(runDir, "usage", "aw_session.jsonl")))
	}
	return runDir
}

func modelRoutingSubagentCostsByName(t *testing.T, costs []ModelRoutingAgentCost) map[string]ModelRoutingAgentCost {
	t.Helper()
	byName := make(map[string]ModelRoutingAgentCost, len(costs))
	for _, cost := range costs {
		_, duplicate := byName[cost.AgentName]
		require.False(t, duplicate, "duplicate sub-agent cost row %q", cost.AgentName)
		byName[cost.AgentName] = cost
	}
	return byName
}

func modelRoutingSubagentCostTotals(costs []ModelRoutingAgentCost) (int, float64) {
	requests, aic := 0, 0.0
	for _, cost := range costs {
		requests += cost.Requests
		aic += cost.AIC
	}
	return requests, aic
}

func TestModelRoutingSubagentCostsBuiltinDefinitions(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		t.Run(map[bool]string{false: "downloaded", true: "legacy"}[legacy], func(t *testing.T) {
			routing := analyzeModelRouting(modelRoutingGoldenVariantDir(t, "copilot-cli-builtin-subagents", legacy))
			require.NotNil(t, routing)
			rows := modelRoutingSubagentCostsByName(t, routing.SubagentCosts)
			require.Len(t, rows, 5)
			expected := map[string]struct {
				displayName string
				model       string
				effort      string
				requests    int
				aic         float64
			}{
				"code-review":     {"router-code-reviewer", "gpt-5.6-sol", "medium", 8, 29.6816},
				"explore":         {"cost-table-explorer", "gpt-5.6-luna", "low", 3, 0.304423},
				"general-purpose": {"python-file-counter", "gpt-5.6-sol", "medium", 2, 8.2792},
				"research":        {"readme-researcher", "gpt-5.6-luna", "medium", 2, 0.542708},
				"task":            {"router-test-runner", "gpt-5.6-luna", "medium", 5, 0.757534},
			}
			for name, want := range expected {
				row, ok := rows[name]
				require.True(t, ok, "missing %s row in %+v", name, routing.SubagentCosts)
				assert.Equal(t, "subagent", row.AgentType, name)
				assert.Equal(t, []string{want.displayName}, row.DisplayNames, name)
				assert.Equal(t, []string{want.model}, row.Models, name)
				assert.Equal(t, want.effort, row.Effort, name)
				assert.Equal(t, 1, row.InstanceCount, name)
				assert.Equal(t, 1, row.CompletedCount, name)
				assert.Equal(t, want.requests, row.Requests, name)
				assert.InDelta(t, want.aic, row.AIC, 0.000001, name)
			}
			requests, aic := modelRoutingSubagentCostTotals(routing.SubagentCosts)
			assert.Equal(t, 20, requests)
			assert.InDelta(t, 39.565465, aic, 0.000001)
		})
	}
}

func TestModelRoutingSubagentCostsSDKServedModels(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		t.Run(map[bool]string{false: "downloaded", true: "legacy"}[legacy], func(t *testing.T) {
			routing := analyzeModelRouting(modelRoutingGoldenVariantDir(t, "copilot-sdk-claude-subagent-completions", legacy))
			require.NotNil(t, routing)
			rows := modelRoutingSubagentCostsByName(t, routing.SubagentCosts)
			require.Len(t, rows, 2)
			assert.Equal(t, []string{"claude-haiku-4.5"}, rows["file-summarizer"].Models)
			assert.Equal(t, []string{"gpt-5.6-luna"}, rows["quick-checker"].Models)
			for name, row := range rows {
				assert.Equal(t, []string{name}, row.DisplayNames, name)
				for _, model := range row.Models {
					assert.NotContains(t, model, "/", "served model %q keeps its provider prefix", model)
				}
			}
			requests, aic := modelRoutingSubagentCostTotals(routing.SubagentCosts)
			assert.Equal(t, 5, requests)
			assert.InDelta(t, 4.18791, aic, 0.000001)
		})
	}
}

func assertLegacyNoSessionExploreRow(t *testing.T, row ModelRoutingAgentCost) {
	t.Helper()
	assert.Equal(t, "subagent", row.AgentType)
	assert.Equal(t, []string{"awf-routing", "ghaw-issues"}, row.DisplayNames)
	assert.Equal(t, 2, row.CompletedCount)
	assert.Equal(t, 15, row.Requests)
	assert.Equal(t, 190104, row.InputTokens)
	assert.Equal(t, 8080, row.OutputTokens)
	assert.Equal(t, 153853, row.CacheReadTokens)
	assert.InDelta(t, 50.1184, row.AIC, 0.000001)
	assert.Equal(t, []string{"claude-opus-5"}, row.Models)
}

func TestModelRoutingSubagentCostsGroupByDefinition(t *testing.T) {
	// The downloaded variant's unified lifecycle events lack a top-level agentId;
	// the native agent-session.jsonl toolCallId bridge reconciles each instance,
	// so both variants count every sub-agent once and recover its effort.
	for _, legacy := range []bool{false, true} {
		t.Run(map[bool]string{false: "downloaded", true: "legacy"}[legacy], func(t *testing.T) {
			routing := analyzeModelRouting(modelRoutingGoldenVariantDir(t, "legacy-no-session-routing", legacy))
			require.NotNil(t, routing)
			rows := modelRoutingSubagentCostsByName(t, routing.SubagentCosts)
			require.Len(t, rows, 2)
			explore, ok := rows["explore"]
			require.True(t, ok, "missing explore row in %+v", routing.SubagentCosts)
			assertLegacyNoSessionExploreRow(t, explore)
			assert.Equal(t, 2, explore.InstanceCount)
			assert.Equal(t, "low", explore.Effort)
			assert.Equal(t, []string{"subagent-research"}, rows["research"].DisplayNames)
			assert.Equal(t, 1, rows["research"].InstanceCount)
			assert.Equal(t, "xhigh", rows["research"].Effort)
		})
	}
}

func TestModelRoutingLogsSubagentCostsGroupByDefinition(t *testing.T) {
	runs := make([]ProcessedRun, 0, 2)
	for _, caseName := range []string{"legacy-no-session-routing", "copilot-cli-builtin-subagents"} {
		runDir := modelRoutingGoldenVariantDir(t, caseName, false)
		runs = append(runs, ProcessedRun{Run: WorkflowRun{LogsPath: runDir}, ModelRouting: analyzeModelRouting(runDir)})
	}
	summary := buildModelRoutingLogsSummary(runs)
	require.NotNil(t, summary)
	var explore []ModelRoutingAgentCost
	for _, row := range summary.SubagentCosts {
		assert.NotContains(t, []string{"ghaw-issues", "awf-routing", "cost-table-explorer"}, row.AgentName)
		if row.AgentName == "explore" {
			explore = append(explore, row)
		}
	}
	// Rows are grouped by definition and effort; the recovered low effort merges
	// the legacy run's explore instances with the built-in explore/low row.
	require.Len(t, explore, 1, "explore rows with the same effort merge: %+v", explore)
	assert.Equal(t, "low", explore[0].Effort)
	assert.Equal(t, []string{"awf-routing", "cost-table-explorer", "ghaw-issues"}, explore[0].DisplayNames)
	assert.ElementsMatch(t, []string{"claude-opus-5", "gpt-5.6-luna"}, explore[0].Models)
	assert.Equal(t, 3, explore[0].InstanceCount)
	assert.Equal(t, 3, explore[0].CompletedCount)
	assert.Equal(t, 18, explore[0].Requests)
}

func TestModelRoutingSubagentCostsWithoutSessionSubagentEvents(t *testing.T) {
	t.Run("fixtures without subagent events are unchanged", func(t *testing.T) {
		for _, caseName := range []string{
			"claude-awf-selected-messages", "claude-awf-steering-notices",
			"codex-routed-provider-unavailable", "copilot-routed-gpt-responses",
		} {
			for _, legacy := range []bool{false, true} {
				runDir := modelRoutingGoldenVariantDir(t, caseName, legacy)
				routing := analyzeModelRouting(runDir)
				usage, err := analyzeTokenUsage(runDir, false)
				require.NoError(t, err)
				actual := marshalModelRoutingGolden(t, projectModelRoutingGoldenAudit(routing, usage))
				expectedPath := filepath.Join("testdata", "model_routing_golden", caseName, "expected.json")
				if legacyPath := filepath.Join("testdata", "model_routing_golden", caseName, "expected.legacy.json"); legacy {
					if _, err := os.Stat(legacyPath); err == nil {
						expectedPath = legacyPath
					}
				}
				expected, err := os.ReadFile(expectedPath)
				require.NoError(t, err)
				var expectedJSON, actualJSON bytes.Buffer
				require.NoError(t, json.Compact(&expectedJSON, expected))
				require.NoError(t, json.Compact(&actualJSON, actual))
				assert.Equal(t, expectedJSON.String(), actualJSON.String(), "%s legacy=%t", caseName, legacy)
				if routing != nil {
					assert.Empty(t, routing.SubagentCosts, caseName)
				}
			}
		}
	})

	t.Run("display name without agent definition keeps display-name rows", func(t *testing.T) {
		content := subagentSessionHeader +
			subagentSessionRecord(`{"type":"session.init","data":{"sourceEngine":"copilot","sessionId":"usage"}}`) +
			subagentSessionRecord(`{"type":"subagent.started","agentId":"quick-1","data":{"agentDisplayName":"quick-checker","model":"small"}}`) +
			subagentSessionRecord(`{"type":"subagent.completed","agentId":"quick-1","data":{}}`) +
			subagentSessionRecord(`{"type":"session.shutdown","data":{"agentMetrics":{"main":{"totalNanoAiu":1599000000,"modelMetrics":{"gpt-5.6-luna":{"requests":{"count":10},"usage":{"inputTokens":100,"outputTokens":20}}}},"quick-1":{"agentDisplayName":"quick-checker","totalNanoAiu":1144000000,"modelMetrics":{"small":{"requests":{"count":4},"usage":{"inputTokens":40,"outputTokens":8}}}}}}}`)
		_, _, agents, found, err := parseSessionSubagentModelsDetailed(strings.NewReader(content), true)
		require.NoError(t, err)
		require.True(t, found)
		require.Len(t, agents, 2)
		assert.Equal(t, "quick-checker", agents[1].AgentName)
		assert.Nil(t, agents[1].DisplayNames)
		routing := &ModelRoutingSummary{}
		applyAgentUsageToModelRouting(routing, agents)
		require.Len(t, routing.SubagentCosts, 1)
		assert.Equal(t, "quick-checker", routing.SubagentCosts[0].AgentName)
		assert.Empty(t, routing.SubagentCosts[0].DisplayNames)
		assert.Equal(t, 4, routing.SubagentCosts[0].Requests)
	})

	t.Run("pi agent names have no display names", func(t *testing.T) {
		usageEvent := subagentSessionRecord(`{"type":"pi.subagent_event","data":{"agentName":"reader","event":{"message":{"model":"gpt-5.4-mini","usage":{"input":10,"output":2}}}}}`)
		content := subagentSessionHeader +
			subagentSessionRecord(`{"type":"session.init","data":{"sourceEngine":"pi","sessionId":"parent"}}`) +
			subagentSessionRecord(`{"type":"pi.subagent_dispatch","data":{"agentName":"reader","requestedModel":"small","resolvedModel":"gpt-5.4-mini"}}`) +
			usageEvent +
			subagentSessionRecord(`{"type":"pi.subagent_result","data":{"agentName":"reader","outcome":"completed"}}`)
		_, _, agents, found, err := parseSessionSubagentModelsDetailed(strings.NewReader(content), true)
		require.NoError(t, err)
		require.True(t, found)
		require.Len(t, agents, 1)
		assert.Equal(t, "reader", agents[0].AgentName)
		assert.Nil(t, agents[0].DisplayNames)
		assert.Equal(t, 1, agents[0].CompletedCount)

		for _, legacy := range []bool{false, true} {
			runDir := modelRoutingGoldenVariantDir(t, "pi-claude-two-subagents", legacy)
			usage, err := analyzeTokenUsage(runDir, false)
			require.NoError(t, err)
			require.NotNil(t, usage)
			names := make([]string, 0, len(usage.AgentUsage))
			for _, agent := range usage.AgentUsage {
				assert.Nil(t, agent.DisplayNames, agent.AgentName)
				names = append(names, agent.AgentName)
			}
			want := []string{"file-summarizer", "main", "quick-checker"}
			if legacy {
				// The legacy fallback lacks the events needed to attribute the main agent.
				want = []string{"file-summarizer", "quick-checker"}
			}
			assert.ElementsMatch(t, want, names)
		}
	})
}

func modelIdentityResolverForDeclarations(t *testing.T, declarations string) *modelIdentityResolver {
	t.Helper()
	runDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(runDir, "aw_info.json"), []byte(`{"sub_agent_models":`+declarations+`}`), 0o600))
	return newModelIdentityResolver(runDir)
}

func TestModelIdentityResolverConcreteDeclarations(t *testing.T) {
	const staticHaiku = `{"name":"file-summarizer","model":"claude-haiku-4.5","patterns":["github-copilot/claude-haiku-4.5"]}`
	// An experiment declaration records the selected variant as its model and
	// the patterns of every variant.
	const experimentHaiku = `{"name":"file-summarizer","model":"claude-haiku-4.5","patterns":["github-copilot/claude-haiku-4.5","github-copilot/gpt-5.6-luna"]}`
	const experimentSmall = `{"name":"quick-checker","model":"small","patterns":["github-copilot/*haiku*","github-copilot/gpt-5*mini*","github-copilot/claude-haiku-4.5"]}`
	haikuUsage := func() []AgentUsageBreakdown {
		return []AgentUsageBreakdown{{
			AgentName: "file-summarizer", AgentType: "subagent", Requests: 3,
			Models: []AgentModelUsage{{Model: "claude-haiku-4.5", Requests: 3}},
		}}
	}

	t.Run("experiment-selected concrete id", func(t *testing.T) {
		resolver := modelIdentityResolverForDeclarations(t, `[`+experimentHaiku+`]`)
		assert.False(t, isModelIdentityAlias(resolver, "claude-haiku-4.5"))
		assert.True(t, resolver.matches("claude-haiku-4.5", "claude-haiku-4.5", ""))
		assert.False(t, resolver.matches("claude-haiku-4.5", "gpt-5.6-luna", ""), "a concrete id matches only its own model")
		agents := resolveAgentUsageModels(haikuUsage(), resolver, "gpt-5.6-luna")
		assert.Equal(t, []string{"claude-haiku-4.5"}, agents[0].ServedModels)
	})

	t.Run("experiment-selected alias", func(t *testing.T) {
		resolver := modelIdentityResolverForDeclarations(t, `[`+experimentSmall+`]`)
		assert.True(t, isModelIdentityAlias(resolver, "small"))
		assert.True(t, resolver.matches("small", "gpt-5.4-mini", ""))
		agents := resolveAgentUsageModels([]AgentUsageBreakdown{{
			AgentName: "quick-checker", AgentType: "subagent", Requests: 2, ResolvedModels: []string{"gpt-5.4-mini"},
			Models: []AgentModelUsage{{Model: "small", Requests: 2}},
		}}, resolver)
		require.Len(t, agents[0].Models, 1)
		assert.Equal(t, "gpt-5.4-mini", agents[0].Models[0].Model)
		assert.NotContains(t, agents[0].ServedModels, "small")
	})

	t.Run("same id declared twice", func(t *testing.T) {
		for name, declarations := range map[string]string{
			"static then experiment": `[` + staticHaiku + `,` + experimentHaiku + `]`,
			"experiment then static": `[` + experimentHaiku + `,` + staticHaiku + `]`,
		} {
			resolver := modelIdentityResolverForDeclarations(t, declarations)
			assert.False(t, isModelIdentityAlias(resolver, "claude-haiku-4.5"), name)
			assert.False(t, resolver.matches("claude-haiku-4.5", "gpt-5.6-luna", ""), name)
			agents := resolveAgentUsageModels(haikuUsage(), resolver, "gpt-5.6-luna")
			assert.Equal(t, []string{"claude-haiku-4.5"}, agents[0].ServedModels, name)
		}
	})
}

func attributeSubagentSession(t *testing.T, content string) *TokenUsageSummary {
	t.Helper()
	requests, actuals, agents, found, err := parseSessionSubagentModelsDetailed(strings.NewReader(content), true)
	require.NoError(t, err)
	require.True(t, found)
	summary := &TokenUsageSummary{}
	resolveSubagentAttribution(summary, requests, actuals, agents, newModelIdentityResolver(t.TempDir()), nil)
	return summary
}

func subagentRequestRowsByName(t *testing.T, rows []SubagentModelRequest) map[string]SubagentModelRequest {
	t.Helper()
	byName := make(map[string]SubagentModelRequest, len(rows))
	for _, row := range rows {
		_, duplicate := byName[row.AgentName]
		require.False(t, duplicate, "duplicate request row %q", row.AgentName)
		byName[row.AgentName] = row
	}
	return byName
}

func TestSubagentRequestRowsKeepDatedServedModels(t *testing.T) {
	// No AWF token usage: the dated id recorded with requests is the only
	// served evidence, and the zero-request dated key is not promoted.
	content := subagentSessionHeader +
		subagentSessionRecord(`{"type":"session.init","data":{"sourceEngine":"copilot","sessionId":"usage"}}`) +
		subagentSessionRecord(`{"type":"subagent.started","agentId":"quick-1","data":{"agentName":"quick-checker","agentDisplayName":"quick-checker","model":"gpt-5.4-mini"}}`) +
		subagentSessionRecord(`{"type":"subagent.completed","agentId":"quick-1","data":{"agentName":"quick-checker","agentDisplayName":"quick-checker"}}`) +
		subagentSessionRecord(`{"type":"session.shutdown","data":{"agentMetrics":{"quick-1":{"agentName":"quick-checker","agentDisplayName":"quick-checker","totalNanoAiu":1144000000,"modelMetrics":{"gpt-5.4-mini-2026-03-17":{"requests":{"count":2},"usage":{"inputTokens":40,"outputTokens":8}},"gpt-5.4-mini-2025-08-07":{"requests":{"count":0},"usage":{"inputTokens":0,"outputTokens":0}}}}}}}`)
	summary := attributeSubagentSession(t, content)
	require.Len(t, summary.AgentUsage, 1)
	assert.Equal(t, []string{"gpt-5.4-mini-2026-03-17"}, summary.AgentUsage[0].ServedModels)
	require.Len(t, summary.SubagentModelRequests, 1)
	row := summary.SubagentModelRequests[0]
	assert.Equal(t, "gpt-5.4-mini", row.EffectiveModel)
	assert.Equal(t, []string{"gpt-5.4-mini-2026-03-17"}, row.ServedModels)
}

func TestSubagentAttributionResolvesInstancesBeforeMergingDefinitions(t *testing.T) {
	t.Run("failed and served instances", func(t *testing.T) {
		content := subagentSessionHeader +
			subagentSessionRecord(`{"type":"session.init","data":{"sourceEngine":"copilot","sessionId":"usage"}}`) +
			subagentSessionRecord(`{"type":"subagent.started","agentId":"a-1","data":{"agentName":"explore","agentDisplayName":"scan-a","model":"claude-haiku-4.5"}}`) +
			subagentSessionRecord(`{"type":"subagent.failed","agentId":"a-1","data":{"agentName":"explore","agentDisplayName":"scan-a","error":"HTTP 400"}}`) +
			subagentSessionRecord(`{"type":"subagent.started","agentId":"b-1","data":{"agentName":"explore","agentDisplayName":"scan-b","model":"claude-haiku-4.5"}}`) +
			subagentSessionRecord(`{"type":"subagent.completed","agentId":"b-1","data":{"agentName":"explore","agentDisplayName":"scan-b"}}`) +
			subagentSessionRecord(`{"type":"session.shutdown","data":{"agentMetrics":{"b-1":{"agentName":"explore","agentDisplayName":"scan-b","totalNanoAiu":2000000000,"modelMetrics":{"claude-haiku-4.5":{"requests":{"count":3},"usage":{"inputTokens":30,"outputTokens":6}}}}}}}`)
		summary := attributeSubagentSession(t, content)
		rows := subagentRequestRowsByName(t, summary.SubagentModelRequests)
		require.Len(t, rows, 2)
		failed := rows["scan-a"]
		assert.Empty(t, failed.EffectiveModel)
		assert.Empty(t, failed.ServedModels)
		assert.Equal(t, modelMismatchReasonSubagentFailed, failed.ReasonCode)
		served := rows["scan-b"]
		assert.Equal(t, "claude-haiku-4.5", served.EffectiveModel)
		assert.Equal(t, []string{"claude-haiku-4.5"}, served.ServedModels)

		require.Len(t, summary.AgentUsage, 1)
		explore := summary.AgentUsage[0]
		assert.Equal(t, "explore", explore.AgentName)
		assert.Equal(t, []string{"scan-a", "scan-b"}, explore.DisplayNames)
		assert.Equal(t, 2, explore.InstanceCount)
		assert.Equal(t, 1, explore.CompletedCount)
		assert.Equal(t, 1, explore.FailedCount)
		assert.Equal(t, 3, explore.Requests)
		assert.Equal(t, []string{"claude-haiku-4.5"}, explore.ServedModels)
	})

	t.Run("alias resolved to different models", func(t *testing.T) {
		content := subagentSessionHeader +
			subagentSessionRecord(`{"type":"session.init","data":{"sourceEngine":"copilot","sessionId":"usage"}}`) +
			subagentSessionRecord(`{"type":"subagent.started","agentId":"a-1","data":{"agentName":"explore","agentDisplayName":"scan-a","model":"small","resolvedModel":"gpt-5.4-mini"}}`) +
			subagentSessionRecord(`{"type":"subagent.completed","agentId":"a-1","data":{"agentName":"explore","agentDisplayName":"scan-a"}}`) +
			subagentSessionRecord(`{"type":"subagent.started","agentId":"b-1","data":{"agentName":"explore","agentDisplayName":"scan-b","model":"small","resolvedModel":"claude-haiku-4.5"}}`) +
			subagentSessionRecord(`{"type":"subagent.completed","agentId":"b-1","data":{"agentName":"explore","agentDisplayName":"scan-b"}}`) +
			subagentSessionRecord(`{"type":"session.shutdown","data":{"agentMetrics":{"a-1":{"agentName":"explore","agentDisplayName":"scan-a","totalNanoAiu":1000000000,"modelMetrics":{"small":{"requests":{"count":2},"usage":{"inputTokens":20,"outputTokens":4}}}},"b-1":{"agentName":"explore","agentDisplayName":"scan-b","totalNanoAiu":3000000000,"modelMetrics":{"small":{"requests":{"count":3},"usage":{"inputTokens":30,"outputTokens":6}}}}}}}`)
		summary := attributeSubagentSession(t, content)
		rows := subagentRequestRowsByName(t, summary.SubagentModelRequests)
		require.Len(t, rows, 2)
		assert.Equal(t, "gpt-5.4-mini", rows["scan-a"].EffectiveModel)
		assert.Equal(t, "claude-haiku-4.5", rows["scan-b"].EffectiveModel)

		require.Len(t, summary.AgentUsage, 1)
		explore := summary.AgentUsage[0]
		assert.Equal(t, "explore", explore.AgentName)
		assert.Equal(t, []string{"scan-a", "scan-b"}, explore.DisplayNames)
		require.Len(t, explore.Models, 2)
		assert.Equal(t, "claude-haiku-4.5", explore.Models[0].Model)
		assert.Equal(t, 3, explore.Models[0].Requests)
		assert.Equal(t, "gpt-5.4-mini", explore.Models[1].Model)
		assert.Equal(t, 2, explore.Models[1].Requests)
		assert.Equal(t, 5, explore.Requests)
		assert.NotContains(t, explore.ServedModels, "small")
	})
}
