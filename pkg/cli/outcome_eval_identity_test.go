//go:build !integration

package cli

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvaluateOutcomesStableActionIdentity(t *testing.T) {
	const actionType = "identity_test_action"
	outcomeEvaluators[actionType] = func(_ context.Context, item CreatedItemReport, repo string) OutcomeReport {
		return OutcomeReport{
			Type: item.Type, Repo: repo, ObjectNumber: item.Number,
			OutcomeEvaluation: outcomeEvidence(OutcomeStatusAccepted, EvidenceStrong, "accepted"),
		}
	}
	t.Cleanup(func() { delete(outcomeEvaluators, actionType) })
	items := []CreatedItemReport{
		{Type: actionType, Number: 1, Timestamp: "2026-10-01T01:00:00Z", Metadata: map[string]any{"change": "first"}},
		{Type: actionType, Number: 1, Timestamp: "2026-10-01T01:00:00Z", Metadata: map[string]any{"change": "second"}},
		{Type: actionType, Number: 2, Timestamp: "2026-10-01T01:00:00Z"},
	}
	first := EvaluateOutcomes(context.Background(), items, "acme/repo", nil)
	require.Len(t, first, 3)
	for i, report := range first {
		assert.NotEmpty(t, report.ItemID)
		for _, other := range first[i+1:] {
			assert.NotEqual(t, report.ItemID, other.ItemID, "same-target actions remain distinct")
		}
	}
	source, err := json.Marshal(items)
	require.NoError(t, err)
	var restored []CreatedItemReport
	require.NoError(t, json.Unmarshal(source, &restored))
	slices.Reverse(restored)
	restored = append([]CreatedItemReport{{Type: "noop"}}, restored...)
	second := EvaluateOutcomes(context.Background(), restored, "acme/repo", nil)
	require.Len(t, second, 3)
	observations := make([]EpisodeOutcomeObservation, 0, 6)
	for i := range first {
		assert.Equal(t, first[i].ItemID, second[len(second)-1-i].ItemID)
		observations = append(observations, EpisodeOutcomeObservation{RunID: 1, RunItemCount: 3, OutcomeReport: first[i]})
		observations = append(observations, EpisodeOutcomeObservation{RunID: 1, RunItemCount: 3, OutcomeReport: second[i]})
	}
	data, err := buildEpisodeOutcomes(episodeLogs(1), observations, map[int64]episodeOutcomeCost{1: episodeCost(6)})
	require.NoError(t, err)
	assert.Equal(t, 3, data.Episodes[0].Summary.Total)
	assert.Equal(t, 3, data.Episodes[0].Summary.Accepted)
	assert.Equal(t, "complete", data.Episodes[0].OutcomeCoverage)
	require.NotNil(t, data.Episodes[0].CostPerAcceptedOutcome)
	assert.InDelta(t, 2.0, *data.Episodes[0].CostPerAcceptedOutcome, 1e-9)

	filtered := EvaluateOutcomes(context.Background(), items[1:2], "acme/repo", nil)
	require.Len(t, filtered, 1)
	assert.Equal(t, first[1].ItemID, filtered[0].ItemID, "filtering unrelated actions cannot change identity")
}

func TestOutcomeItemIDIndistinguishableActions(t *testing.T) {
	t.Parallel()
	item := CreatedItemReport{Type: "add_comment", Number: 1, Timestamp: "2026-10-01T01:00:00Z"}
	occurrences := make(map[[sha256.Size]byte]int)
	first := outcomeItemID(item, occurrences)
	second := outcomeItemID(item, occurrences)
	assert.NotEqual(t, first, second, "identical manifest entries still represent distinct actions")
	again := make(map[[sha256.Size]byte]int)
	assert.Equal(t, first, outcomeItemID(item, again))
	assert.Equal(t, second, outcomeItemID(item, again))
}

func TestOutcomeItemIDCanonicalSourceEncoding(t *testing.T) {
	t.Parallel()
	first := CreatedItemReport{
		Type: "update_issue", Number: 1, Timestamp: "2026-10-01T01:00:00Z",
		Metadata: map[string]any{"z": float64(1), "a": map[string]any{"title": "updated", "number": float64(2)}},
	}
	second := first
	second.Metadata = map[string]any{"a": map[string]any{"number": 2, "title": "updated"}, "z": 1}
	assert.Equal(t, outcomeItemID(first, make(map[[sha256.Size]byte]int)),
		outcomeItemID(second, make(map[[sha256.Size]byte]int)))
	second.Metadata = map[string]any{"invalid": make(chan int)}
	assert.Empty(t, outcomeItemID(second, make(map[[sha256.Size]byte]int)),
		"unrepresentable evidence must not receive a misleading stable identity")
}
