//go:build !integration

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/agentdrain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditSearchAutomaticGrouping(t *testing.T) {
	t.Parallel()
	var crowded strings.Builder
	for i := range 6 {
		crowded.WriteString(auditSearchTestEvent("tool.execution_complete", fmt.Sprintf("timeout request %d", i), i))
	}
	for _, test := range []struct {
		name    string
		content string
		limit   int
		grouped bool
	}{
		{"single", auditSearchTestEvent("tool.execution_complete", "timeout request 1", 0), 5, false},
		{"fits", auditSearchTestEvent("tool.execution_complete", "timeout request 1", 0) +
			auditSearchTestEvent("tool.execution_complete", "timeout request 2", 1), 5, false},
		{"crowded", crowded.String(), 5, true},
		{"unrelated", auditSearchTestEvent("tool.execution_complete", "timeout on first service", 0) +
			auditSearchTestEvent("assistant.message", "timeout on second service", 1), 1, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			docs, err := parseAuditSearchSession(context.Background(), strings.NewReader(auditSearchTestHeader+test.content))
			require.NoError(t, err)
			raw, err := rankAuditEvidence(context.Background(), docs, "timeout")
			require.NoError(t, err)
			clusters, err := groupAuditEvidence(context.Background(), docs, raw, "timeout", test.limit)
			require.NoError(t, err)
			after, err := rankAuditEvidence(context.Background(), docs, "timeout")
			require.NoError(t, err)
			assert.Equal(t, raw, after, "Drain3 must never change BM25 scores or ranks")
			search := selectAuditEvidence(docs, clusters, raw, 42, "usage/aw_session.jsonl", "timeout", test.limit)
			require.NotEmpty(t, search.Matches)
			first := search.Matches[0]
			assert.Equal(t, docs[raw[0].index].ref, first.Evidence.Reference, "the best BM25 result must remain first")
			assert.Equal(t, test.grouped, first.ClusterID != 0)
			if test.grouped {
				assert.Equal(t, len(docs), first.MatchingOccurrences)
				assert.NotEmpty(t, first.Template)
			} else {
				assert.Empty(t, first.Template)
				assert.Zero(t, first.AnomalyScore)
			}
		})
	}
}

func TestAuditSearchExactDuplicatesWithoutMining(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	var content strings.Builder
	for i := range 8 {
		content.WriteString(auditSearchTestEvent("tool.execution_complete", "timeout", i))
	}
	writeAuditSearchTestSession(t, dir, content.String())
	search, err := searchAuditSession(context.Background(), dir, 42, "timeout", 5)
	require.NoError(t, err)
	require.Len(t, search.Matches, 1)
	match := search.Matches[0]
	assert.Zero(t, match.ClusterID)
	assert.Equal(t, 8, match.Occurrences)
	assert.Equal(t, 8, match.MatchingOccurrences)
	assert.Equal(t, 8, search.MatchingEvents)
	assert.Len(t, match.References, 3)
	assert.Empty(t, match.Template)
	encoded, err := json.Marshal(match)
	require.NoError(t, err)
	for _, key := range []string{"cluster_id", "template", "anomaly_score"} {
		assert.NotContains(t, string(encoded), key)
	}
}

func TestAuditSearchOutcomePartitions(t *testing.T) {
	t.Parallel()
	for _, field := range []string{"success", "isError", "exitCode", "status", "result", "decision", "event", "toolName", "error", "errorCode", "numericErrorCode", "resultKind", "resultIsError", "scalarType"} {
		t.Run(field, func(t *testing.T) {
			a, b := map[string]any{}, map[string]any{}
			switch field {
			case "success", "isError":
				a[field], b[field] = true, false
			case "exitCode":
				a[field], b[field] = json.Number("0"), json.Number("1")
			case "error":
				b["error"] = map[string]any{"message": "failed"}
			case "errorCode":
				a["error"], b["error"] = map[string]any{"code": "denied"}, map[string]any{"code": "failure"}
			case "numericErrorCode":
				a["error"], b["error"] = map[string]any{"code": json.Number("-32601")}, map[string]any{"code": json.Number("-32000")}
			case "scalarType":
				a["success"], b["success"] = false, "false"
			case "resultKind":
				a["result"], b["result"] = map[string]any{"kind": "approved"}, map[string]any{"kind": "denied"}
			case "resultIsError":
				a["result"], b["result"] = map[string]any{"isError": false}, map[string]any{"isError": true}
			default:
				a[field], b[field] = "success", "failure"
			}
			docs := []auditSearchDocument{
				{stage: "agent/agent/tool.execution_complete", clusterID: 1, outcome: auditSearchOutcome(a)},
				{stage: "agent/agent/tool.execution_complete", clusterID: 1, outcome: auditSearchOutcome(b)},
			}
			require.NotEqual(t, docs[0].outcome, docs[1].outcome)
			cluster := agentdrain.Cluster{ID: 1, Size: 2, Template: []string{"<*>"}}
			original := map[string]agentdrain.Cluster{auditEvidenceClusterKey(docs[0]): cluster}
			partitions := partitionAuditEvidenceOutcomes(docs, original)
			require.Len(t, partitions, 2)
			assert.NotEqual(t, docs[0].clusterID, docs[1].clusterID)
			for _, group := range partitions {
				assert.Equal(t, 1, group.Size)
			}
		})
	}
}

func TestAuditSearchAutomaticGroupingPreservesOutcomeCounts(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	var content strings.Builder
	for i := range 12 {
		_, err := fmt.Fprintf(&content, `{"type":"workflow.job_result","data":{"jobId":"job-%d","result":%q},"provenance":{"component":"workflow"}}`+"\n",
			i, []string{"success", "failure"}[i%2])
		require.NoError(t, err)
	}
	writeAuditSearchTestSession(t, dir, content.String())
	search, err := searchAuditSession(context.Background(), dir, 42, "workflow.job_result", 5)
	require.NoError(t, err)
	require.Len(t, search.Matches, 2)
	assert.Equal(t, 12, search.MatchingEvents)
	for _, match := range search.Matches {
		assert.Equal(t, 6, match.Occurrences)
		assert.Equal(t, 6, match.MatchingOccurrences)
		for _, ref := range match.References {
			assert.Equal(t, match.Evidence.Reference.Line%2, ref.Line%2, "references must share the representative's outcome")
		}
	}
}

func TestAuditSearchGroupingMustSaveExcerptBytes(t *testing.T) {
	t.Parallel()
	docs := []auditSearchDocument{
		{stage: "tool", clusterID: 1, text: "timeout 1"},
		{stage: "tool", clusterID: 1, text: "timeout 2"},
	}
	ranked := []auditSearchRank{{index: 0}, {index: 1}}
	clusters := map[string]agentdrain.Cluster{
		"tool:1": {Template: []string{strings.Repeat("template ", 100)}},
	}
	assert.Empty(t, auditEvidenceUsefulGroups(docs, ranked, clusters, "timeout"))
	clusters["tool:1"] = agentdrain.Cluster{Template: []string{"timeout"}}
	assert.Contains(t, auditEvidenceUsefulGroups(docs, ranked, clusters, "timeout"), "tool:1")
}

func TestAuditSearchUncrowdedSkipsMiner(t *testing.T) {
	t.Parallel()
	docs := make([]auditSearchDocument, 257)
	for i := range docs {
		docs[i] = auditSearchDocument{stage: fmt.Sprintf("stage-%d", i), text: "timeout"}
	}
	original := slices.Clone(docs)
	clusters, err := groupAuditEvidence(context.Background(), docs, nil, "absent", 5)
	require.NoError(t, err, "no matches must not start the 256-stage Drain3 miner")
	assert.Len(t, clusters, 257)
	assert.Equal(t, original, docs)
	dir := t.TempDir()
	var content strings.Builder
	for i := range 257 {
		content.WriteString(auditSearchTestEvent(fmt.Sprintf("event-%d", i), "timeout", i))
	}
	writeAuditSearchTestSession(t, dir, content.String())
	_, err = searchAuditSession(context.Background(), dir, 42, "absent", 5)
	require.ErrorContains(t, err, "256-stage", "the session limit applies even when Drain3 is skipped")
}
