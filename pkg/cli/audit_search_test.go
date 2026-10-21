//go:build !integration

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/github/gh-aw/pkg/agentdrain"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const auditSearchTestHeader = `{"type":"session.format","data":{"version":1},"provenance":{"component":"collector","phase":"conclusion"}}` + "\n"

func auditSearchTestEvent(kind, text string, index int) string {
	data, _ := json.Marshal(map[string]string{"content": text})
	return fmt.Sprintf(`{"id":"event-%d","type":%q,"timestamp":"2026-10-21T00:00:00Z","data":%s,"provenance":{"component":"agent","phase":"agent","path":"agent-session.jsonl","index":%d}}`+"\n", index, kind, data, index)
}

func writeAuditSearchTestSession(t *testing.T, dir, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "usage"), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "usage", "aw_session.jsonl"), []byte(auditSearchTestHeader+content), 0600))
}

func TestAuditSearchTokens(t *testing.T) {
	t.Parallel()
	tokens := auditSearchTokens("pkg/cli/audit.go github.issue_read API.EXAMPLE.COM ERR_AUTH_401 Échec")
	for _, term := range []string{"pkg/cli/audit.go", "pkg", "cli", "audit", "go", "github.issue_read", "issue", "read", "api.example.com", "err_auth_401", "auth", "401", "échec"} {
		assert.Contains(t, tokens, term)
	}
	assert.Empty(t, auditSearchTokens("... /// ___"))
}

func TestAuditSearchBM25(t *testing.T) {
	t.Parallel()
	docs := []auditSearchDocument{{text: "needle"}, {text: "hay hay"}}
	ranked, err := rankAuditEvidence(context.Background(), docs, "needle")
	require.NoError(t, err)
	require.Len(t, ranked, 1)
	assert.Equal(t, 0, ranked[0].index)
	want := math.Log(2) * 2.2 / (1 + 1.2*(0.25+0.75/1.5))
	assert.InDelta(t, want, ranked[0].score, 0.000001)

	repeated := append(slices.Clone(docs), docs[0], docs[0], docs[0])
	repeatedRanks, err := rankAuditEvidence(context.Background(), repeated, "needle needle")
	require.NoError(t, err)
	assert.Equal(t, ranked, repeatedRanks, "exact repeats and repeated query words must not distort relevance")
}

func TestAuditSearchUnifiedSession(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	content := auditSearchTestEvent("assistant.message", "Inspect the failing service", 0) +
		auditSearchTestEvent("tool.execution_complete", "Request api.example.com failed ERR_AUTH_401", 1) +
		auditSearchTestEvent("assistant.message", "Retry using the configured credentials", 2)
	writeAuditSearchTestSession(t, dir, content)
	// Neither the raw agent log nor summaries may be used as a replacement corpus.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "agent-session.jsonl"), []byte("raw-only-secret"), 0600))
	search, err := searchAuditSession(context.Background(), dir, 42, "ERR_AUTH_401 api.example.com", 5)
	require.NoError(t, err)
	assert.Equal(t, int64(42), search.RunID)
	assert.Equal(t, "usage/aw_session.jsonl", search.Source)
	assert.Equal(t, 3, search.IndexedEvents)
	require.Len(t, search.Matches, 1)
	match := search.Matches[0]
	assert.Positive(t, match.Score)
	assert.Equal(t, "agent/agent/tool.execution_complete", match.Stage)
	assert.Equal(t, 3, match.Evidence.Reference.Line)
	assert.Equal(t, "event-1", match.Evidence.Reference.ID)
	assert.Equal(t, "agent-session.jsonl", match.Evidence.Reference.Path)
	require.NotNil(t, match.Evidence.Reference.Index)
	assert.Equal(t, 1, *match.Evidence.Reference.Index)
	assert.JSONEq(t, `"2026-10-21T00:00:00Z"`, string(match.Evidence.Reference.Timestamp))
	require.Len(t, match.Context, 2)
	assert.Equal(t, 2, match.Context[0].Reference.Line)
	assert.Equal(t, 4, match.Context[1].Reference.Line)
	assert.Contains(t, match.Evidence.Text, "ERR_AUTH_401")
	assert.NotContains(t, match.Evidence.Text, "raw-only-secret")
	missing, err := searchAuditSession(context.Background(), dir, 42, "raw-only-secret", 5)
	require.NoError(t, err)
	assert.Empty(t, missing.Matches)
	assert.Zero(t, missing.MatchingEvents)
}

func TestAuditSearchDrainGrouping(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	var content strings.Builder
	for i := range 12 {
		content.WriteString(auditSearchTestEvent("tool.execution_complete", fmt.Sprintf("request api.example.com failed with status 429 attempt %d", i), i))
	}
	content.WriteString(auditSearchTestEvent("tool.execution_complete", "request api.example.com failed with status 429 attempt 11", 12))
	writeAuditSearchTestSession(t, dir, content.String())
	search, err := searchAuditSession(context.Background(), dir, 42, "429", 5)
	require.NoError(t, err)
	require.Len(t, search.Matches, 1)
	match := search.Matches[0]
	assert.Equal(t, 13, match.Occurrences)
	assert.Equal(t, 13, match.MatchingOccurrences)
	assert.Equal(t, 13, search.MatchingEvents)
	assert.Len(t, match.References, 3)
	assert.Contains(t, match.Template, "<")
	assert.GreaterOrEqual(t, match.AnomalyScore, 0.0)
	assert.LessOrEqual(t, match.AnomalyScore, 1.0)
}

func TestAuditSearchRetainsMaskedParametersAndUnicode(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeAuditSearchTestSession(t, dir, auditSearchTestEvent("tool.execution_complete", "timeout contacting api.example.com request 12345 Échec", 0))
	for _, query := range []string{"12345", "api.example.com", "échec"} {
		search, err := searchAuditSession(context.Background(), dir, 42, query, 5)
		require.NoError(t, err)
		require.Len(t, search.Matches, 1, query)
	}
}

func TestAuditSearchStagesAndNativeProvenance(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	agent := auditSearchTestEvent("tool.execution_complete", "timeout api.example.com", 0)
	detection := strings.ReplaceAll(auditSearchTestEvent("tool.execution_complete", "timeout api.example.com", 1), `"phase":"agent"`, `"phase":"detection"`)
	gateway := `{"type":"mcp.tool_result","data":{"error":"timeout api.example.com"},"provenance":{"component":"mcp","phase":"agent","path":"gateway.jsonl","index":0}}` + "\n"
	native := `{"type":"assistant.message","agentId":"child-1","data":{"content":"timeout\u0020api.example.com"},"provenance":{"component":"agent","phase":"agent","path":"agent-session.jsonl","index":2,"native":{"path":"children/trace.jsonl","index":4,"persistedPath":"sessions/child.jsonl"}}}` + "\n"
	writeAuditSearchTestSession(t, dir, agent+detection+gateway+native)
	search, err := searchAuditSession(context.Background(), dir, 42, "timeout", 5)
	require.NoError(t, err)
	require.Len(t, search.Matches, 4, "cluster ID 1 in different stages must not collapse")
	for _, match := range search.Matches {
		assert.Equal(t, 1, match.Occurrences)
		assert.Equal(t, 1, match.ClusterID)
		if match.Evidence.Reference.AgentID == "child-1" {
			ref := match.Evidence.Reference
			require.NotNil(t, ref.Native)
			assert.Equal(t, "children/trace.jsonl", ref.Native.Path)
			assert.Equal(t, "sessions/child.jsonl", ref.Native.PersistedPath)
			require.NotNil(t, ref.Native.Index)
			assert.Equal(t, 4, *ref.Native.Index)
			assert.Contains(t, match.Evidence.Text, "timeout api.example.com")
		}
	}
	require.NoError(t, os.Rename(filepath.Join(dir, "usage", "aw_session.jsonl"), filepath.Join(dir, "aw_session.jsonl")))
	search, err = searchAuditSession(context.Background(), dir, 42, "timeout", 1)
	require.NoError(t, err)
	assert.Equal(t, "aw_session.jsonl", search.Source)
	assert.True(t, search.Truncated)
}

func TestAuditSearchRejectsInvalidSessions(t *testing.T) {
	t.Parallel()
	tests := map[string]string{
		"empty":                 "",
		"native without header": auditSearchTestEvent("assistant.message", "hello", 0),
		"unsupported header":    strings.Replace(auditSearchTestHeader, `"version":1`, `"version":2`, 1),
		"duplicate header":      auditSearchTestHeader + auditSearchTestHeader,
		"malformed event":       auditSearchTestHeader + "{invalid}\n",
		"missing provenance":    auditSearchTestHeader + `{"type":"assistant.message","data":{}}`,
		"invalid data":          auditSearchTestHeader + `{"type":"assistant.message","data":null,"provenance":{"component":"agent"}}`,
	}
	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := parseAuditSearchSession(context.Background(), strings.NewReader(content))
			require.Error(t, err)
		})
	}
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "agent-session.jsonl"), []byte("native only"), 0600))
	_, _, err := readAuditSearchSession(context.Background(), dir)
	require.ErrorContains(t, err, "unified session unavailable")
	target := filepath.Join(dir, "other.jsonl")
	require.NoError(t, os.WriteFile(target, []byte(auditSearchTestHeader), 0600))
	require.NoError(t, os.Symlink(target, filepath.Join(dir, "aw_session.jsonl")))
	_, _, err = readAuditSearchSession(context.Background(), dir)
	require.ErrorContains(t, err, "symbolic link")
}

func TestAuditSearchInputBoundsAndCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := parseAuditSearchSession(ctx, strings.NewReader(auditSearchTestHeader))
	require.ErrorIs(t, err, context.Canceled)
	_, err = rankAuditEvidence(ctx, []auditSearchDocument{{text: "needle"}}, "needle")
	require.ErrorIs(t, err, context.Canceled)
	input := auditSearchTestHeader + strings.Repeat(auditSearchTestEvent("tool.execution_complete", "retry", 1), maxAuditSearchEvents+1)
	_, err = parseAuditSearchSession(context.Background(), strings.NewReader(input))
	require.ErrorContains(t, err, "20000-event")
	_, err = parseAuditSearchSession(context.Background(), strings.NewReader(auditSearchTestHeader+strings.Repeat(" ", maxUnifiedSessionLineSize+1)+"\n"))
	require.ErrorContains(t, err, "oversized")
	var nested any = "needle"
	for range 66 {
		nested = []any{nested}
	}
	var decoded strings.Builder
	require.ErrorContains(t, flattenAuditSearchData(&decoded, "", nested, 0), "nesting limit")
	stages := make([]auditSearchDocument, 257)
	for i := range stages {
		stages[i].stage = fmt.Sprintf("stage-%d", i)
	}
	_, err = clusterAuditEvidence(context.Background(), stages)
	require.ErrorContains(t, err, "256-stage")
	dir := t.TempDir()
	file, err := os.Create(filepath.Join(dir, "aw_session.jsonl"))
	require.NoError(t, err)
	require.NoError(t, file.Truncate(maxAuditSearchSessionBytes+1))
	require.NoError(t, file.Close())
	_, _, err = readAuditSearchSession(context.Background(), dir)
	require.ErrorContains(t, err, "64 MiB")
}

func TestAuditSearchOutputBoundsAndExcerpt(t *testing.T) {
	t.Parallel()
	docs := make([]auditSearchDocument, 20)
	clusters := make(map[string]agentdrain.Cluster)
	ranked := make([]auditSearchRank, 20)
	for i := range docs {
		docs[i] = auditSearchDocument{
			ref: AuditEvidenceRef{Line: i + 2}, stage: "tool_call", clusterID: i + 1,
			text: strings.Repeat("background ", 400) + "needle " + strings.Repeat("tail ", 400),
		}
		clusters[auditEvidenceClusterKey(docs[i])] = agentdrain.Cluster{Size: 1, Template: []string{strings.Repeat("template ", 100)}}
		ranked[i] = auditSearchRank{i, float64(20 - i)}
	}
	search := selectAuditEvidence(docs, clusters, ranked, 42, "usage/aw_session.jsonl", "needle", 20)
	assert.True(t, search.Truncated)
	assert.LessOrEqual(t, search.TextBytes, maxAuditSearchTextBytes)
	assert.NotEmpty(t, search.Matches)
	for _, match := range search.Matches {
		assert.Contains(t, match.Evidence.Text, "needle")
		assert.True(t, match.Evidence.Truncated)
		assert.LessOrEqual(t, len(match.Evidence.Text), auditSearchExcerptBytes)
	}
	excerpt := auditEvidenceExcerpt(auditSearchDocument{text: strings.Repeat("é", 500) + " needle\x1b[31m\n"}, "needle")
	assert.True(t, utf8.ValidString(excerpt.Text))
	assert.Contains(t, excerpt.Text, "needle")
	assert.NotContains(t, excerpt.Text, "\x1b")
	assert.True(t, utf8.ValidString(auditSearchClip("ééé", 5)))
	excerpt = auditEvidenceExcerpt(auditSearchDocument{text: strings.Repeat("Ⱥ", 500) + " needle"}, "needle")
	assert.Contains(t, excerpt.Text, "needle", "Unicode case conversion must not corrupt source offsets")
}

func TestAuditSearchRecallAtFixedBudget(t *testing.T) {
	t.Parallel()
	// Plain first-match search spends the one-result budget on verbose noise.
	var docs []auditSearchDocument
	for range 10 {
		docs = append(docs, auditSearchDocument{text: "timeout " + strings.Repeat("unrelated ", 100)})
	}
	docs = append(docs,
		auditSearchDocument{text: "api.example.com timeout ERR_AUTH_401"},
		auditSearchDocument{text: "github.issue_read timeout rate_limit"},
		auditSearchDocument{text: "pkg/cli/audit.go timeout context_deadline"})
	for _, identifier := range []string{"ERR_AUTH_401", "github.issue_read", "pkg/cli/audit.go"} {
		ranked, err := rankAuditEvidence(context.Background(), docs, "timeout "+identifier)
		require.NoError(t, err)
		require.NotEmpty(t, ranked)
		assert.Contains(t, docs[ranked[0].index].text, identifier, "relevant evidence recall@1 must be 1")
		assert.NotContains(t, docs[0].text, identifier, "first-match baseline recall@1 is 0")
	}
}

func TestAuditSearchOptions(t *testing.T) {
	t.Parallel()
	for _, flags := range [][]string{
		{"--search", "  "}, {"--search", "///"}, {"--search-limit", "2"},
		{"--search", "timeout", "--search-limit", "0"},
		{"--search", "timeout", "--search-limit", "21"},
		{"--search", strings.Repeat("a", 1025)},
	} {
		cmd := NewAuditCommand()
		require.NoError(t, cmd.ParseFlags(flags))
		_, err := getAuditCommandOptions(cmd)
		require.Error(t, err)
	}
	artifacts := []string{"agent"}
	cfg, err := newAuditRunConfig(42, AuditOptions{Hostname: "github.com", Search: "timeout", ArtifactSets: artifacts})
	require.NoError(t, err)
	assert.Equal(t, []string{"agent"}, artifacts, "caller options must not be mutated")
	assert.Contains(t, cfg.artifactFilter, "usage")
	assert.Equal(t, defaultAuditSearchLimit, cfg.auditOptions().SearchLimit)
	assert.Equal(t, "timeout", cfg.auditOptions().Search)
	for _, opts := range []AuditOptions{{Search: "timeout", Group: true}, {Search: "timeout", JobID: 10}} {
		_, err := newAuditRunConfig(42, opts)
		require.Error(t, err)
	}
	for _, args := range [][]string{
		{"42", "43", "--search", "timeout"},
		{"42", "--group", "--search", "timeout"},
		{"https://github.com/owner/repo/actions/runs/42/job/10", "--search", "timeout"},
	} {
		cmd := NewAuditCommand()
		cmd.SetArgs(args)
		require.ErrorContains(t, cmd.Execute(), "--search")
	}
}

func TestAuditSearchReportDoesNotChangeFindingsOrCache(t *testing.T) {
	dir := t.TempDir()
	run := WorkflowRun{DatabaseID: 42, Status: "completed", Conclusion: "failure", LogsPath: dir}
	cached := AuditData{
		CacheSource: auditCacheSourceFull, Overview: buildAuditOverview(run, nil),
		KeyFindings: []AuditFinding{{Title: "deterministic finding"}},
		Metrics:     MetricsData{ErrorCount: 3},
	}

	require.NoError(t, writeAuditData(dir, cached))
	writeAuditSearchTestSession(t, dir, auditSearchTestEvent("tool.execution_complete", "ERR_AUTH_401", 0))
	before, err := os.ReadFile(auditPath(dir))
	require.NoError(t, err)
	opts := AuditOptions{OutputDir: dir, JSONOutput: true, Search: "ERR_AUTH_401"}
	output, _ := captureOutput(t, func() error {
		return renderAuditReport(context.Background(), ProcessedRun{Run: run}, LogMetrics{}, nil, opts)
	})
	var searched AuditData
	require.NoError(t, json.Unmarshal([]byte(output), &searched))
	require.NotNil(t, searched.EvidenceSearch)
	assert.Equal(t, cached.KeyFindings, searched.KeyFindings)
	assert.Equal(t, cached.Metrics, searched.Metrics)
	after, err := os.ReadFile(auditPath(dir))
	require.NoError(t, err)
	assert.Equal(t, before, after)

	opts.Search = "no-matching-evidence"
	output, _ = captureOutput(t, func() error {
		return renderAuditReport(context.Background(), ProcessedRun{Run: run}, LogMetrics{}, nil, opts)
	})
	require.NoError(t, json.Unmarshal([]byte(output), &searched))
	assert.Empty(t, searched.EvidenceSearch.Matches)
	assert.Equal(t, cached.KeyFindings, searched.KeyFindings, "zero hits cannot suppress deterministic findings")
	opts.Search = ""
	output, _ = captureOutput(t, func() error {
		return renderAuditReport(context.Background(), ProcessedRun{Run: run}, LogMetrics{}, nil, opts)
	})
	assert.NotContains(t, output, "evidence_search")
	console := testutil.CaptureStderr(t, func() { renderConsoleEvidenceSearch(searched.EvidenceSearch) })
	assert.Contains(t, console, "retrieval only")
}

func TestAuditSearchConsoleEvidence(t *testing.T) {
	dir := t.TempDir()
	writeAuditSearchTestSession(t, dir, auditSearchTestEvent("tool.execution_complete", "timeout\x1b[31m\napi.example.com", 0))
	search, err := searchAuditSession(context.Background(), dir, 42, "timeout", 5)
	require.NoError(t, err)
	output := testutil.CaptureStderr(t, func() { renderConsoleEvidenceSearch(search) })
	assert.Contains(t, output, "bm25=")
	assert.Contains(t, output, "occurrences=1 matching=1")
	assert.Contains(t, output, "line=2")
	assert.Contains(t, output, "timeout")
	assert.NotContains(t, output, "\x1b")
}

func TestAuditSearchMissingSessionBypassesSummary(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, saveRunSummary(dir, &RunSummary{RunID: 42, CLIVersion: GetVersion()}, false))
	require.NoError(t, markArtifactDownloaded(dir, string(ArtifactSetAll)))
	done, skipped, err := renderCachedAuditIfAvailable(context.Background(), auditRunConfig{
		runID: 42, outputDir: dir, search: "timeout",
	})
	require.NoError(t, err)
	assert.False(t, done)
	assert.False(t, skipped)
	_, err = os.Stat(filepath.Join(dir, downloadedArtifactsMarkerDir, string(ArtifactSetAll)))
	require.ErrorIs(t, err, os.ErrNotExist)
	err = renderPreparedAuditReport(context.Background(), AuditData{}, AuditOptions{OutputDir: dir, Search: "timeout"})
	require.ErrorContains(t, err, "unified session unavailable")
}
