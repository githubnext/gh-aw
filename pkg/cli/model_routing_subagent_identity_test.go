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
	// The downloaded variant counts each instance twice because its lifecycle
	// events lack a top-level agentId; that separate bug is pinned here.
	instanceCounts := map[bool]int{false: 4, true: 2}
	for _, legacy := range []bool{false, true} {
		t.Run(map[bool]string{false: "downloaded", true: "legacy"}[legacy], func(t *testing.T) {
			routing := analyzeModelRouting(modelRoutingGoldenVariantDir(t, "legacy-no-session-routing", legacy))
			require.NotNil(t, routing)
			rows := modelRoutingSubagentCostsByName(t, routing.SubagentCosts)
			require.Len(t, rows, 2)
			explore, ok := rows["explore"]
			require.True(t, ok, "missing explore row in %+v", routing.SubagentCosts)
			assertLegacyNoSessionExploreRow(t, explore)
			assert.Equal(t, instanceCounts[legacy], explore.InstanceCount)
			if legacy {
				assert.Equal(t, "low", explore.Effort)
			}
			assert.Equal(t, []string{"subagent-research"}, rows["research"].DisplayNames)
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
	require.Len(t, explore, 2, "explore rows with different efforts stay separate: %+v", explore)
	byEffort := map[string]ModelRoutingAgentCost{explore[0].Effort: explore[0], explore[1].Effort: explore[1]}
	assertLegacyNoSessionExploreRow(t, byEffort[""])
	assert.Equal(t, 4, byEffort[""].InstanceCount)
	builtin, ok := byEffort["low"]
	require.True(t, ok)
	assert.Equal(t, []string{"cost-table-explorer"}, builtin.DisplayNames)
	assert.Equal(t, []string{"gpt-5.6-luna"}, builtin.Models)
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
