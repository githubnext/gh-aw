//go:build !integration

package cli

import (
	"math"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func episodeObservation(runID int64, itemID string, status OutcomeStatus) EpisodeOutcomeObservation {
	return EpisodeOutcomeObservation{
		RunID: runID, RunItemCount: 1,
		OutcomeReport: OutcomeReport{
			ItemID: itemID, Type: "create_issue", Repo: "acme/repo", ObjectNumber: 10,
			CreatedAt: "2026-10-01T01:00:00Z", CheckedAt: "2026-10-02T01:00:00Z",
			OutcomeEvaluation: outcomeEvidence(status, EvidenceStrong, string(status)),
		},
	}
}

func episodeLogs(ids ...int64) episodeOutcomesLogs {
	logs := episodeOutcomesLogs{Episodes: []episodeOutcomeMembership{{EpisodeData: EpisodeData{EpisodeID: "ep-1", RunIDs: ids, TotalRuns: len(ids)}}}}
	for _, id := range ids {
		logs.Runs = append(logs.Runs, RunData{RunID: id})
	}
	return logs
}

func episodeCost(aic float64) episodeOutcomeCost {
	return episodeOutcomeCost{aic: &aic}
}

func TestBuildEpisodeOutcomesLatestAcrossRuns(t *testing.T) {
	t.Parallel()
	first := episodeObservation(1, "item-1", OutcomeStatusPending)
	accepted := episodeObservation(1, "item-1", OutcomeStatusAccepted)
	accepted.CheckedAt = "2026-10-03T01:00:00Z"
	acceptedSecond := episodeObservation(2, "item-1", OutcomeStatusAccepted)
	observations := []EpisodeOutcomeObservation{accepted, acceptedSecond, first, accepted}
	costs := map[int64]episodeOutcomeCost{1: episodeCost(3), 2: episodeCost(5)}
	data, err := buildEpisodeOutcomes(episodeLogs(2, 1), observations, costs)
	require.NoError(t, err)
	require.Len(t, data.Episodes, 1)
	row := data.Episodes[0]
	assert.Equal(t, []int64{1, 2}, row.RunIDs)
	assert.Equal(t, "selected_runs", row.MembershipScope)
	assert.Equal(t, "complete", row.OutcomeCoverage)
	assert.Equal(t, "available", row.CostCoverage)
	assert.Equal(t, 2, row.Summary.Total)
	assert.Equal(t, 2, row.Summary.Accepted)
	assert.Equal(t, []int64{1, 2}, row.OutcomeRunIDs)
	require.NotNil(t, row.ObservedAIC)
	require.NotNil(t, row.CostPerAcceptedOutcome)
	assert.InDelta(t, 8.0, *row.ObservedAIC, 1e-9)
	assert.InDelta(t, 4.0, *row.CostPerAcceptedOutcome, 1e-9)
	assert.Contains(t, row.Qualifications[0], "full-episode membership is not verified")

	slices.Reverse(observations)
	reversed, err := buildEpisodeOutcomes(episodeLogs(2, 1), observations, costs)
	require.NoError(t, err)
	assert.Equal(t, data, reversed)
}

func TestBuildEpisodeOutcomesCostAndEvidence(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name            string
		costs           map[int64]episodeOutcomeCost
		status          OutcomeStatus
		missingEvidence bool
		costCoverage    string
		observed        *float64
		ratio           *float64
	}{
		{name: "zero is observed cost", costs: map[int64]episodeOutcomeCost{1: episodeCost(0)}, status: OutcomeStatusAccepted, costCoverage: "available", observed: new(float64), ratio: new(float64)},
		{name: "unavailable is not zero", status: OutcomeStatusAccepted, costCoverage: "unavailable"},
		{name: "negative cost", costs: map[int64]episodeOutcomeCost{1: episodeCost(-1)}, status: OutcomeStatusAccepted, costCoverage: "unavailable"},
		{name: "nonfinite cost", costs: map[int64]episodeOutcomeCost{1: episodeCost(math.Inf(1))}, status: OutcomeStatusAccepted, costCoverage: "unavailable"},
		{name: "nan cost", costs: map[int64]episodeOutcomeCost{1: episodeCost(math.NaN())}, status: OutcomeStatusAccepted, costCoverage: "unavailable"},
		{name: "no accepted", costs: map[int64]episodeOutcomeCost{1: episodeCost(2)}, status: OutcomeStatusRejected, costCoverage: "available", observed: episodeCost(2).aic},
		{name: "pending", costs: map[int64]episodeOutcomeCost{1: episodeCost(2)}, status: OutcomeStatusPending, costCoverage: "available", observed: episodeCost(2).aic},
		{name: "unknown", costs: map[int64]episodeOutcomeCost{1: episodeCost(2)}, status: OutcomeStatusUnknown, costCoverage: "available", observed: episodeCost(2).aic},
		{name: "error", costs: map[int64]episodeOutcomeCost{1: episodeCost(2)}, status: OutcomeStatusError, costCoverage: "available", observed: episodeCost(2).aic},
		{name: "no observations", costs: map[int64]episodeOutcomeCost{1: episodeCost(2)}, missingEvidence: true, costCoverage: "available", observed: episodeCost(2).aic},
		{name: "usage warnings", costs: map[int64]episodeOutcomeCost{1: {aic: episodeCost(2).aic, warnings: []string{"unpriced usage"}}}, status: OutcomeStatusAccepted, costCoverage: "partial", observed: episodeCost(2).aic},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := []EpisodeOutcomeObservation{episodeObservation(1, "item-1", tc.status)}
			if tc.missingEvidence {
				input = nil
			}
			data, err := buildEpisodeOutcomes(episodeLogs(1), input, tc.costs)
			require.NoError(t, err)
			row := data.Episodes[0]
			assert.Equal(t, tc.costCoverage, row.CostCoverage)
			assert.Equal(t, tc.observed, row.ObservedAIC)
			assert.Equal(t, tc.ratio, row.CostPerAcceptedOutcome)
			if tc.missingEvidence {
				assert.Equal(t, []int64{1}, row.MissingOutcomeRunIDs)
				assert.Equal(t, "partial", row.OutcomeCoverage)
			}
			if tc.status == OutcomeStatusPending || tc.status == OutcomeStatusUnknown || tc.status == OutcomeStatusError {
				assert.Contains(t, row.Qualifications, "Pending, unknown, and error outcomes remain unresolved; accepted counts are an observation snapshot.")
			}
		})
	}
}

func TestBuildEpisodeOutcomesPartialCoverage(t *testing.T) {
	t.Parallel()
	observations := []EpisodeOutcomeObservation{episodeObservation(1, "item-1", OutcomeStatusAccepted)}
	costs := map[int64]episodeOutcomeCost{1: episodeCost(2)}
	data, err := buildEpisodeOutcomes(episodeLogs(1, 2), observations, costs)
	require.NoError(t, err)
	row := data.Episodes[0]
	assert.Equal(t, 1, row.Summary.Accepted)
	assert.Equal(t, "partial", row.CostCoverage)
	assert.Equal(t, "partial", row.OutcomeCoverage)
	assert.Equal(t, []int64{2}, row.MissingOutcomeRunIDs)
	assert.Equal(t, []int64{2}, row.MissingCostRunIDs)
	assert.InDelta(t, 2.0, *row.ObservedAIC, 1e-9)
	assert.Nil(t, row.CostPerAcceptedOutcome)

	logs := episodeLogs(1, 2)
	logs.Runs = logs.Runs[:1]
	costs[2] = episodeCost(3)
	observations = append(observations, episodeObservation(2, "item-1", OutcomeStatusAccepted))
	data, err = buildEpisodeOutcomes(logs, observations, costs)
	require.NoError(t, err)
	assert.Equal(t, []int64{2}, data.Episodes[0].MissingRunIDs)
	assert.Nil(t, data.Episodes[0].CostPerAcceptedOutcome)

	logs = episodeLogs(1)
	logs.Episodes[0].TotalRuns = 2
	data, err = buildEpisodeOutcomes(logs, observations, costs)
	require.NoError(t, err)
	assert.Nil(t, data.Episodes[0].CostPerAcceptedOutcome)
	assert.Equal(t, []int64{2}, data.UnmatchedOutcomeRunIDs)
}

func TestBuildEpisodeOutcomesItemIdentity(t *testing.T) {
	t.Parallel()
	accepted := episodeObservation(1, "item-1", OutcomeStatusAccepted)
	second := episodeObservation(1, "item-2", OutcomeStatusPending)
	accepted.RunItemCount, second.RunItemCount = 2, 2
	observations := []EpisodeOutcomeObservation{accepted, second}
	data, err := buildEpisodeOutcomes(episodeLogs(1), observations, map[int64]episodeOutcomeCost{1: episodeCost(6)})
	require.NoError(t, err)
	assert.Equal(t, 2, data.Episodes[0].Summary.Total, "different actions on the same target remain distinct")
	assert.Equal(t, 1, data.Episodes[0].Summary.Accepted)
	assert.Equal(t, 1, data.Episodes[0].Summary.Pending)
	assert.InDelta(t, 6.0, *data.Episodes[0].CostPerAcceptedOutcome, 1e-9)

	legacy := accepted
	legacy.ItemID = ""
	legacy.CheckedAt = "2026-10-01T01:00:00Z"
	second.OutcomeStatus = OutcomeStatusAccepted
	observations = []EpisodeOutcomeObservation{accepted, second, legacy, legacy}
	data, err = buildEpisodeOutcomes(episodeLogs(1), observations, map[int64]episodeOutcomeCost{1: episodeCost(6)})
	require.NoError(t, err)
	assert.Equal(t, 2, data.Episodes[0].Summary.Accepted, "stale legacy observations neither merge precise items nor add items")
	assert.Equal(t, "complete", data.Episodes[0].OutcomeCoverage)
	require.NotNil(t, data.Episodes[0].CostPerAcceptedOutcome)
	assert.InDelta(t, 3.0, *data.Episodes[0].CostPerAcceptedOutcome, 1e-9)
}

func TestLatestEpisodeOutcomesIdentitySnapshots(t *testing.T) {
	t.Parallel()
	snapshot := func(itemID string, status OutcomeStatus, checkedAt string) EpisodeOutcomeObservation {
		item := episodeObservation(1, itemID, status)
		item.CheckedAt = checkedAt
		item.RunItemCount = 2
		return item
	}
	const (
		oldTime    = "2026-10-01T01:00:00Z"
		modernTime = "2026-10-02T01:00:00Z"
		legacyTime = "2026-10-03T01:00:00Z"
		newTime    = "2026-10-04T01:00:00Z"
	)
	first := snapshot("item-1", OutcomeStatusAccepted, modernTime)
	second := snapshot("item-2", OutcomeStatusAccepted, modernTime)
	firstRejected := snapshot("item-1", OutcomeStatusRejected, newTime)
	secondPending := snapshot("item-2", OutcomeStatusPending, modernTime)
	legacyOld := snapshot("", OutcomeStatusAccepted, oldTime)
	legacyOld.RunItemCount = 0
	legacyRejected := snapshot("", OutcomeStatusRejected, newTime)
	legacyOtherTarget := legacyOld
	legacyOtherTarget.ObjectNumber++
	firstRefreshed := snapshot("item-1", OutcomeStatusAccepted, newTime)
	secondRefreshed := snapshot("item-2", OutcomeStatusAccepted, newTime)

	for _, tc := range []struct {
		name         string
		observations []EpisodeOutcomeObservation
		want         []EpisodeOutcomeObservation
		err          string
	}{
		{
			name:         "modern only replacement and repeated acceptance",
			observations: []EpisodeOutcomeObservation{first, second, firstRejected, second, firstRejected},
			want:         []EpisodeOutcomeObservation{firstRejected, second},
		},
		{
			name:         "legacy only duplicates",
			observations: []EpisodeOutcomeObservation{legacyOld, legacyOld},
			want:         []EpisodeOutcomeObservation{legacyOld},
		},
		{
			name:         "legacy only replacement",
			observations: []EpisodeOutcomeObservation{legacyOld, legacyRejected, legacyOld},
			want:         []EpisodeOutcomeObservation{legacyRejected},
		},
		{
			name:         "mixed stale legacy preserves precise actions",
			observations: []EpisodeOutcomeObservation{legacyOld, first, second, legacyOld, first, second},
			want:         []EpisodeOutcomeObservation{first, second},
		},
		{
			name:         "mixed legacy does not affect another target",
			observations: []EpisodeOutcomeObservation{legacyOld, first, second, legacyOtherTarget, legacyOtherTarget},
			want:         []EpisodeOutcomeObservation{legacyOtherTarget, first, second},
		},
		{
			name:         "precise statuses supersede same instant legacy",
			observations: []EpisodeOutcomeObservation{snapshot("", OutcomeStatusRejected, "2026-10-02T03:00:00+02:00"), first, secondPending},
			want:         []EpisodeOutcomeObservation{first, secondPending},
		},
		{
			name:         "later precise rejection replaces acceptance with stale legacy",
			observations: []EpisodeOutcomeObservation{legacyOld, first, firstRejected, second},
			want:         []EpisodeOutcomeObservation{firstRejected, second},
		},
		{
			name:         "fully refreshed precise snapshots supersede newer legacy",
			observations: []EpisodeOutcomeObservation{first, second, snapshot("", OutcomeStatusRejected, legacyTime), firstRefreshed, secondRefreshed},
			want:         []EpisodeOutcomeObservation{firstRefreshed, secondRefreshed},
		},
		{
			name:         "partially refreshed precise snapshots do not supersede newer legacy",
			observations: []EpisodeOutcomeObservation{first, second, snapshot("", OutcomeStatusRejected, legacyTime), firstRefreshed},
			err:          "legacy outcome for run 1 is newer than precise item observations; reevaluate the run",
		},
		{
			name:         "newer legacy acceptance is ambiguous",
			observations: []EpisodeOutcomeObservation{first, second, snapshot("", OutcomeStatusAccepted, legacyTime)},
			err:          "legacy outcome for run 1 is newer than precise item observations; reevaluate the run",
		},
		{
			name:         "newer legacy rejection is ambiguous",
			observations: []EpisodeOutcomeObservation{first, second, snapshot("", OutcomeStatusRejected, legacyTime)},
			err:          "legacy outcome for run 1 is newer than precise item observations; reevaluate the run",
		},
		{
			name:         "newer legacy pending status is ambiguous",
			observations: []EpisodeOutcomeObservation{first, second, snapshot("", OutcomeStatusPending, legacyTime)},
			err:          "legacy outcome for run 1 is newer than precise item observations; reevaluate the run",
		},
		{
			name:         "conflicting stale legacy is superseded by precise snapshots",
			observations: []EpisodeOutcomeObservation{legacyOld, snapshot("", OutcomeStatusRejected, oldTime), first, second},
			want:         []EpisodeOutcomeObservation{first, second},
		},
		{
			name:         "conflicting stale modern statuses are superseded",
			observations: []EpisodeOutcomeObservation{first, snapshot("item-1", OutcomeStatusRejected, modernTime), firstRefreshed},
			want:         []EpisodeOutcomeObservation{firstRefreshed},
		},
		{
			name:         "conflicting stale legacy statuses are superseded",
			observations: []EpisodeOutcomeObservation{legacyOld, snapshot("", OutcomeStatusRejected, oldTime), legacyRejected},
			want:         []EpisodeOutcomeObservation{legacyRejected},
		},
		{
			name:         "conflicting latest modern statuses remain errors",
			observations: []EpisodeOutcomeObservation{firstRefreshed, firstRejected, legacyOld},
			err:          "conflicting outcomes",
		},
		{
			name:         "conflicting latest legacy statuses remain errors",
			observations: []EpisodeOutcomeObservation{legacyRejected, snapshot("", OutcomeStatusAccepted, newTime)},
			err:          "conflicting outcomes",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, reverse := range []bool{false, true} {
				input := slices.Clone(tc.observations)
				if reverse {
					slices.Reverse(input)
				}
				got, err := latestEpisodeOutcomes(input)
				if tc.err != "" {
					require.ErrorContains(t, err, tc.err)
					continue
				}
				require.NoError(t, err)
				assert.Equal(t, tc.want, got)
				data, err := buildEpisodeOutcomes(episodeLogs(1), input, nil)
				require.NoError(t, err)
				accepted := 0
				for _, item := range tc.want {
					if item.OutcomeStatus == OutcomeStatusAccepted {
						accepted++
					}
				}
				assert.Equal(t, len(tc.want), data.Episodes[0].Summary.Total)
				assert.Equal(t, accepted, data.Episodes[0].Summary.Accepted)
			}
		})
	}
}

func TestBuildEpisodeOutcomesIncompleteItems(t *testing.T) {
	t.Parallel()
	for _, count := range []int{0, 2} {
		item := episodeObservation(1, "item-1", OutcomeStatusAccepted)
		item.RunItemCount = count
		data, err := buildEpisodeOutcomes(episodeLogs(1), []EpisodeOutcomeObservation{item}, map[int64]episodeOutcomeCost{1: episodeCost(6)})
		require.NoError(t, err)
		assert.Equal(t, "partial", data.Episodes[0].OutcomeCoverage)
		assert.Nil(t, data.Episodes[0].CostPerAcceptedOutcome)
	}
}

func TestBuildEpisodeOutcomesRecordedMembershipCoverage(t *testing.T) {
	t.Parallel()
	for _, coverage := range []*EpisodeOutcomeMembershipCoverage{
		{Scope: "selected_runs", Status: "partial", Reasons: []string{"parent_not_selected"}},
		{Scope: "selected_runs", Status: "unknown", CollectionTruncated: true},
	} {
		logs := episodeLogs(1)
		logs.Episodes[0].Coverage = coverage
		data, err := buildEpisodeOutcomes(logs, []EpisodeOutcomeObservation{episodeObservation(1, "item-1", OutcomeStatusAccepted)},
			map[int64]episodeOutcomeCost{1: episodeCost(2)})
		require.NoError(t, err)
		row := data.Episodes[0]
		assert.Equal(t, "complete", row.OutcomeCoverage, "lineage and outcome coverage are independent")
		assert.Equal(t, "available", row.CostCoverage)
		assert.Equal(t, coverage, row.MembershipCoverage)
		assert.Nil(t, row.CostPerAcceptedOutcome)
	}
}

func TestLatestEpisodeOutcomesValidation(t *testing.T) {
	t.Parallel()
	item := episodeObservation(1, "item-1", OutcomeStatusAccepted)
	newer := item
	newer.OutcomeStatus = OutcomeStatusRejected
	newer.CheckedAt = "2026-10-03T01:00:00Z"
	data, err := buildEpisodeOutcomes(episodeLogs(1), []EpisodeOutcomeObservation{newer, item}, nil)
	require.NoError(t, err)
	assert.Zero(t, data.Episodes[0].Summary.Accepted, "a previous acceptance does not survive a later rejection")
	assert.Equal(t, 1, data.Episodes[0].Summary.Rejected)

	newer.CheckedAt = item.CheckedAt
	_, err = latestEpisodeOutcomes([]EpisodeOutcomeObservation{newer, item})
	require.ErrorContains(t, err, "conflicting outcomes")
	item.CheckedAt = "invalid"
	_, err = latestEpisodeOutcomes([]EpisodeOutcomeObservation{item})
	require.ErrorContains(t, err, "invalid checked_at")
	item.RunID = 0
	_, err = latestEpisodeOutcomes([]EpisodeOutcomeObservation{item})
	require.ErrorContains(t, err, "positive run_id")
}

func TestBuildEpisodeOutcomesMultipleEpisodes(t *testing.T) {
	t.Parallel()
	logs := episodeLogs(1, 2)
	logs.Episodes = append(logs.Episodes, episodeOutcomeMembership{EpisodeData: EpisodeData{EpisodeID: "ep-2", RunIDs: []int64{3}, TotalRuns: 1}})
	logs.Runs = append(logs.Runs, RunData{RunID: 3})
	observations := []EpisodeOutcomeObservation{
		episodeObservation(1, "item-1", OutcomeStatusAccepted),
		episodeObservation(2, "item-1", OutcomeStatusAccepted),
		episodeObservation(3, "item-1", OutcomeStatusUnknown),
		episodeObservation(99, "item-1", OutcomeStatusAccepted),
	}
	data, err := buildEpisodeOutcomes(logs, observations, nil)
	require.NoError(t, err)
	require.Len(t, data.Episodes, 2)
	assert.Equal(t, 2, data.Episodes[0].Summary.Accepted)
	assert.Zero(t, data.Episodes[1].Summary.Accepted)
	assert.Equal(t, 1, data.Episodes[1].Summary.Unknown)
	assert.Equal(t, []int64{99}, data.UnmatchedOutcomeRunIDs)

	logs.Episodes[1].RunIDs = []int64{2}
	_, err = buildEpisodeOutcomes(logs, observations, nil)
	require.ErrorContains(t, err, "overlapping episode membership")
}
