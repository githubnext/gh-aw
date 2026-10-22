package cli

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEpisodeRetrievalRetriesFailedCandidateAsRecordedParent(t *testing.T) {
	for _, failsAgain := range []bool{false, true} {
		t.Run(map[bool]string{false: "retry succeeds", true: "retry reports recorded failure"}[failsAgain], func(t *testing.T) {
			f := newEpisodeRetrievalFixture(t)
			f.add("org/one", 2, "canonical", "org/one", 2)
			f.add("org/one", 1, "canonical", "org/one", 1)
			f.add("org/one", 3, "canonical", "org/one", 1)
			attempts := 0
			request := func(ctx context.Context, host, endpoint string) ([]byte, error) {
				if strings.HasPrefix(endpoint, "repos/org/one/actions/runs/1/artifacts?") {
					attempts++
					if attempts == 1 || failsAgain {
						f.calls = append(f.calls, endpoint)
						return nil, errors.New("HTTP 503: transient artifact inventory failure")
					}
				}
				return f.request(ctx, host, endpoint)
			}
			opts := episodeRetrievalTestOptions(t)
			opts.MaxRuns = 3
			data, err := retrieveEpisodeLogs(t.Context(), opts, request)
			require.NoError(t, err)
			require.Equal(t, 2, attempts)
			require.Equal(t, 3, data.EpisodeRetrieval.RunsExamined)
			require.NotContains(t, data.EpisodeRetrieval.StopReasons, "run_limit")
			require.Equal(t, len(f.calls), data.EpisodeRetrieval.APIRequests)
			require.Equal(t, "candidate_unavailable", data.EpisodeRetrieval.Issues[0].Reason)
			require.Equal(t, int64(1), data.EpisodeRetrieval.Issues[0].RunID)
			if failsAgain {
				require.ElementsMatch(t, []int64{2, 3}, data.Episodes[0].RunIDs)
				require.Len(t, data.EpisodeRetrieval.Issues, 2)
				require.Equal(t, "recorded_run_unavailable", data.EpisodeRetrieval.Issues[1].Reason)
				require.Equal(t, int64(1), data.EpisodeRetrieval.Issues[1].RunID)
			} else {
				require.ElementsMatch(t, []int64{1, 2, 3}, data.Episodes[0].RunIDs)
				require.Len(t, data.EpisodeRetrieval.Issues, 1)
			}
		})
	}
}

func TestEpisodeRetrievalRetryHonorsHardAPIBudget(t *testing.T) {
	f := newEpisodeRetrievalFixture(t)
	f.add("org/one", 2, "canonical", "org/one", 2)
	f.add("org/one", 1, "canonical", "org/one", 1)
	f.add("org/one", 3, "canonical", "org/one", 1)
	request := func(ctx context.Context, host, endpoint string) ([]byte, error) {
		if strings.HasPrefix(endpoint, "repos/org/one/actions/runs/1/artifacts?") {
			f.calls = append(f.calls, endpoint)
			return nil, errors.New("HTTP 503: transient artifact inventory failure")
		}
		return f.request(ctx, host, endpoint)
	}
	opts := episodeRetrievalTestOptions(t)
	opts.MaxRuns, opts.MaxAPIRequests = 3, 8
	data, err := retrieveEpisodeLogs(t.Context(), opts, request)
	require.NoError(t, err)
	require.Equal(t, 3, data.EpisodeRetrieval.RunsExamined)
	require.Len(t, f.calls, opts.MaxAPIRequests)
	require.Equal(t, len(f.calls), data.EpisodeRetrieval.APIRequests)
	require.Equal(t, []string{"api_request_limit"}, data.EpisodeRetrieval.StopReasons)
	require.Equal(t, "repos/org/one/actions/runs/1", f.calls[len(f.calls)-1])
	require.Equal(t, "candidate_unavailable", data.EpisodeRetrieval.Issues[0].Reason)
	require.ElementsMatch(t, []int64{2, 3}, data.Episodes[0].RunIDs)
}

func TestEpisodeRetrievalDateOnlyEndIncludesEndDaySeed(t *testing.T) {
	f := newEpisodeRetrievalFixture(t)
	f.add("org/one", 2, "canonical", "org/one", 2)
	opts := episodeRetrievalTestOptions(t)
	opts.StartDate, opts.EndDate = "2026-10-19", "2026-10-19"
	data, err := retrieveEpisodeLogs(t.Context(), opts, f.request)
	require.NoError(t, err)
	require.Equal(t, []int64{2}, data.Episodes[0].RunIDs)
	require.Equal(t, time.Date(2026, 10, 19, 23, 59, 59, 0, time.UTC), data.EpisodeRetrieval.EndDate)
	for _, call := range f.calls {
		if strings.Contains(call, "/actions/runs?") {
			parsed, err := url.Parse(call)
			require.NoError(t, err)
			require.Equal(t, "2026-10-19T00:00:00Z..2026-10-19T23:59:59Z", parsed.Query().Get("created"))
		}
	}
}

func TestEpisodeRetrievalExplicitEndTimestampUnchanged(t *testing.T) {
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	for _, endDate := range []string{"2026-10-19T00:00:00Z", "2026-10-19T12:30:00+02:00"} {
		_, end, err := episodeRetrievalDates("2026-10-19", endDate, now)
		require.NoError(t, err)
		expected, err := time.Parse(time.RFC3339, endDate)
		require.NoError(t, err)
		require.True(t, expected.Equal(end))
	}
}
