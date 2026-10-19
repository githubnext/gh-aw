package cli

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func episodeAccountingAWFRecord(id string, delta, total float64) string {
	return fmt.Sprintf(`{"_schema":"token-usage/v0.28.7","event":"token_usage","request_id":%q,"provider":"unknown","ai_credits_this_response":%g,"ai_credits_total":%g}`, id, delta, total)
}

func retrieveEpisodeAccountingArtifacts(t *testing.T, archives []map[string]string) LogsData {
	t.Helper()
	fixture := newEpisodeRetrievalFixture(t)
	fixture.add("org/one", 2, "canonical", "", 0)
	opts := episodeRetrievalTestOptions(t)
	request := func(ctx context.Context, host, endpoint string) ([]byte, error) {
		if endpoint == "repos/org/one/actions/runs/2/artifacts?per_page=100&page=1" {
			artifacts := make([]episodeArtifact, len(archives))
			for i := range archives {
				artifacts[i] = episodeArtifact{ID: int64(21 + i), Name: fmt.Sprintf("invocation-%d-usage", i)}
			}
			return json.Marshal(map[string]any{"artifacts": artifacts})
		}
		if strings.HasPrefix(endpoint, "repos/org/one/actions/artifacts/") {
			id, err := strconv.Atoi(strings.Split(endpoint, "/")[5])
			require.NoError(t, err)
			require.GreaterOrEqual(t, id, 21)
			require.Less(t, id-21, len(archives))
			var buffer bytes.Buffer
			writer := zip.NewWriter(&buffer)
			info, err := writer.Create("aw_info.json")
			require.NoError(t, err)
			require.NoError(t, json.NewEncoder(info).Encode(AwInfo{
				RunID: 2, RunAttempt: "1", Context: fixture.contexts[2],
			}))
			for name, content := range archives[id-21] {
				file, err := writer.Create(name)
				require.NoError(t, err)
				_, err = file.Write([]byte(content))
				require.NoError(t, err)
			}
			require.NoError(t, writer.Close())
			return buffer.Bytes(), nil
		}
		return fixture.request(ctx, host, endpoint)
	}
	data, err := retrieveEpisodeLogs(context.Background(), opts, request)
	require.NoError(t, err)
	require.Len(t, data.Runs, 1)
	return data
}

func TestEpisodeAccountingMultipleValidatedUsageArtifacts(t *testing.T) {
	record := func(id string, input int, delta, total float64) string {
		return fmt.Sprintf(`{"event":"token_usage","request_id":%q,"model":"gpt-4.1","input_tokens":%d,"output_tokens":2,"ai_credits_this_response":%g,"ai_credits_total":%g}`+"\n", id, input, delta, total)
	}
	first := record("first", 10, 0.25, 0.25)
	for _, tc := range []struct {
		name   string
		second string
	}{
		{name: "independent counters", second: record("second", 30, 0.75, 0.75)},
		{name: "overlapping streams", second: first + record("second", 30, 0.75, 1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := retrieveEpisodeAccountingArtifacts(t, []map[string]string{
				{"agent/token_usage.jsonl": first},
				{"agent/token_usage.jsonl": tc.second},
				{"agent/token_usage.jsonl": first},
			})
			run := data.Runs[0]
			require.NotNil(t, run.TokenUsageSummary)
			require.Equal(t, 44, data.Summary.TotalTokens)
			require.Equal(t, 2, run.TokenUsageSummary.TotalRequests)
			require.Equal(t, 40, run.TokenUsageSummary.TotalInputTokens)
			require.InDelta(t, 1, run.TokenUsageSummary.TotalAIC, 1e-9)
			require.InDelta(t, 1, run.TokenUsageSummary.ByModel["gpt-4.1"].AIC, 1e-9)
			require.Contains(t, strings.Join(run.TokenUsageSummary.Warnings, "\n"), "duplicate token usage")

			files, warnings := findUsageJSONLFiles(run.LogsPath)
			require.Empty(t, warnings)
			require.Len(t, files, 3)
			offline, err := analyzeTokenUsageAICOnly(run.LogsPath, false)
			require.NoError(t, err)
			require.NotNil(t, offline)
			require.True(t, offline.AICFound)
			require.InDelta(t, 1, offline.TotalAIC, 1e-9)

			stale := filepath.Join(filepath.Dir(run.LogsPath), "999-usage")
			require.NoError(t, os.MkdirAll(stale, 0o700))
			require.NoError(t, os.WriteFile(filepath.Join(stale, ".episode-complete"), []byte("v1\n"), 0o600))
			require.NoError(t, os.WriteFile(filepath.Join(stale, "token_usage.jsonl"), []byte(record("stale", 900, 99, 99)), 0o600))
			offline, err = analyzeTokenUsageAICOnly(run.LogsPath, false)
			require.NoError(t, err)
			require.InDelta(t, 1, offline.TotalAIC, 1e-9, "offline accounting must retain the validated inventory")
		})
	}
}

func TestEpisodeAccountingUsageFallbackPerArtifact(t *testing.T) {
	data := retrieveEpisodeAccountingArtifacts(t, []map[string]string{
		{"agent/token_usage.jsonl": episodeAccountingAWFRecord("first", 0.25, 0.25)},
		{"agent_usage.json": `{"input_tokens":30,"output_tokens":2,"ai_credits":0.75,"primary_model":"gpt-4.1"}`},
	})
	require.NotNil(t, data.Runs[0].TokenUsageSummary)
	require.InDelta(t, 1, data.Runs[0].TokenUsageSummary.TotalAIC, 1e-9)
	offline, err := analyzeTokenUsageAICOnly(data.Runs[0].LogsPath, false)
	require.NoError(t, err)
	require.NotNil(t, offline)
	require.InDelta(t, 1, offline.TotalAIC, 1e-9)
}

func TestEpisodeAccountingActivityArchivePreservesResults(t *testing.T) {
	activity := `{
		"schema":"usage-activity-summary/v1",
		"session":{"turns":7},
		"firewall":{"total_requests":10,"allowed_requests":8,"blocked_requests":2},
		"gateway":{
			"tools":[{"server_name":"github","tool_name":"issue_read","call_count":2}],
			"servers":[{"server_name":"github","tool_call_count":2}],
			"tool_calls":[{"tool_call_id":"call-1","server_name":"github","tool_name":"issue_read","outcome":"success"}]
		},
		"steering":{"total_events":3,"event_counts":{"token_steering":3}},
		"skills":{"items":[{"name":"documentation","invocation_count":2,"failed_count":1}]},
		"safe_outputs":{"total_items":1,"items":[{"type":"create_issue","number":12,"repo":"org/one"}]},
		"working_set":{"measurement_state":"measured","rebuild_factor":2,"cumulative_input_tokens":200,"peak_input_tokens":100,"invocations":2},
		"ledger":{"transactions_added":3},
		"friction":{"measurement_state":"measured","total_events":1,"cost":{"aic":0.1}}
	}`
	for _, withTokens := range []bool{false, true} {
		t.Run(fmt.Sprintf("with_tokens_%t", withTokens), func(t *testing.T) {
			files := map[string]string{"activity/summary.json": activity}
			if withTokens {
				files["agent/token_usage.jsonl"] = episodeAccountingAWFRecord("first", 0.25, 0.25)
			}
			data := retrieveEpisodeAccountingArtifacts(t, []map[string]string{files})
			run := data.Runs[0]
			require.Equal(t, 7, run.Turns)
			require.Equal(t, 1, run.SafeItemsCount)
			require.NotNil(t, run.TokenUsageSummary, "steering-only activity must retain the initialized token summary")
			require.Equal(t, 3, run.TokenUsageSummary.TotalSteeringEvents)
			require.Equal(t, 3, run.TokenUsageSummary.SteeringEventCounts["token_steering"])
			processed, err := processEpisodeArtifacts(WorkflowRun{DatabaseID: 2}, run.LogsPath, &AwInfo{Context: run.AwContext})
			require.NoError(t, err)
			require.NotNil(t, processed.FirewallAnalysis)
			require.Equal(t, 10, processed.FirewallAnalysis.TotalRequests)
			require.NotNil(t, processed.MCPToolUsage)
			require.Len(t, processed.MCPToolUsage.ToolCalls, 1)
			require.Len(t, processed.SkillActivations, 1)
			require.Equal(t, 2, processed.SkillActivations[0].InvocationCount)
			require.Len(t, run.SafeOutputs, 1)
			require.Equal(t, 12, run.SafeOutputs[0].Number)
			require.NotNil(t, run.WorkingSet)
			require.EqualValues(t, 200, run.WorkingSet.CumulativeInputTokens)
			require.NotNil(t, run.Ledger)
			require.NotNil(t, run.Ledger.TransactionsAdded)
			require.Equal(t, 3, *run.Ledger.TransactionsAdded)
			require.NotNil(t, run.Friction)
			require.InDelta(t, 0.1, run.Friction.Cost.AIC, 1e-9)
		})
	}
}

func TestEpisodeAccountingActivityInventoryPreservesErrors(t *testing.T) {
	for _, failure := range []string{"read", "parse", "schema"} {
		t.Run(failure, func(t *testing.T) {
			root := t.TempDir()
			collection := filepath.Join(root, "collection-test")
			require.NoError(t, os.Mkdir(collection, 0o700))
			inventory, err := json.Marshal([]string{filepath.Join("..", "21-usage"), filepath.Join("..", "22-usage")})
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(filepath.Join(collection, ".episode-artifacts.json"), inventory, 0o600))
			for _, id := range []string{"21", "22"} {
				dir := filepath.Join(root, id+"-usage")
				require.NoError(t, os.MkdirAll(filepath.Join(dir, "activity"), 0o700))
				require.NoError(t, os.WriteFile(filepath.Join(dir, ".episode-complete"), []byte("v1\n"), 0o600))
			}
			first := filepath.Join(root, "21-usage", "activity", "summary.json")
			switch failure {
			case "read":
				require.NoError(t, os.Mkdir(first, 0o700))
			case "parse":
				require.NoError(t, os.WriteFile(first, []byte("{"), 0o600))
			case "schema":
				require.NoError(t, os.WriteFile(first, []byte(`{"schema":"unsupported"}`), 0o600))
			}
			require.NoError(t, os.WriteFile(
				filepath.Join(root, "22-usage", "activity", "summary.json"),
				[]byte(`{"schema":"usage-activity-summary/v1","session":{"turns":7}}`), 0o600,
			))

			summary, err := loadUsageActivitySummary(collection)
			require.Error(t, err, "a later valid summary must not hide an earlier artifact error")
			require.Nil(t, summary)
			require.Contains(t, err.Error(), first)
			require.Contains(t, err.Error(), failure)
		})
	}
}
