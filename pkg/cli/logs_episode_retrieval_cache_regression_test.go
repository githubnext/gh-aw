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

func episodeCacheRegressionArchive(t *testing.T, info AwInfo, tokens int) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	file, err := writer.Create("aw_info.json")
	require.NoError(t, err)
	require.NoError(t, json.NewEncoder(file).Encode(info))
	if tokens > 0 {
		file, err = writer.Create("token_usage.jsonl")
		require.NoError(t, err)
		_, err = fmt.Fprintf(file, "{\"model\":\"gpt-4.1\",\"input_tokens\":%d,\"output_tokens\":2,\"ai_credits_this_response\":0.25,\"ai_credits_total\":0.25}\n", tokens)
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())
	return buffer.Bytes()
}

func TestEpisodeCacheRegressionExpiredCompletedEvidence(t *testing.T) {
	fixture := newEpisodeRetrievalFixture(t)
	fixture.add("org/one", 2, "canonical", "org/one", 2)
	opts := episodeRetrievalTestOptions(t)
	first, err := retrieveEpisodeLogs(context.Background(), opts, fixture.request)
	require.NoError(t, err)
	firstDirs, present, err := readEpisodeArtifactDirs(first.Runs[0].LogsPath)
	require.NoError(t, err)
	require.True(t, present)
	require.Len(t, firstDirs, 1)
	fixture.calls = nil
	request := func(ctx context.Context, host, endpoint string) ([]byte, error) {
		if strings.Contains(endpoint, "/artifacts?") {
			fixture.calls = append(fixture.calls, endpoint)
			return json.Marshal(map[string]any{"artifacts": []episodeArtifact{{ID: 20, Name: "usage", Expired: true}}})
		}
		return fixture.request(ctx, host, endpoint)
	}
	second, err := retrieveEpisodeLogs(context.Background(), opts, request)
	require.NoError(t, err)
	require.Equal(t, first.Summary.TotalTokens, second.Summary.TotalTokens)
	require.Empty(t, second.EpisodeRetrieval.Issues)
	require.NotContains(t, strings.Join(fixture.calls, "\n"), "/zip")
	_, err = os.Stat(filepath.Join(firstDirs[0], "token_usage.jsonl"))
	require.NoError(t, err)
	usage, err := analyzeTokenUsageAICOnly(first.Runs[0].LogsPath, false)
	require.NoError(t, err)
	require.InDelta(t, 0.25, usage.TotalAIC, 0.000001)
}

func TestEpisodeCacheRegressionExpiredUncachedUsage(t *testing.T) {
	fixture := newEpisodeRetrievalFixture(t)
	fixture.add("org/one", 2, "canonical", "org/one", 2)
	request := func(ctx context.Context, host, endpoint string) ([]byte, error) {
		if strings.Contains(endpoint, "/artifacts?") {
			return json.Marshal(map[string]any{"artifacts": []episodeArtifact{
				{ID: 20, Name: "info"}, {ID: 21, Name: "usage", Expired: true},
			}})
		}
		return fixture.request(ctx, host, endpoint)
	}
	data, err := retrieveEpisodeLogs(context.Background(), episodeRetrievalTestOptions(t), request)
	require.NoError(t, err)
	require.Equal(t, []int64{2}, data.Episodes[0].RunIDs)
	require.Contains(t, data.Episodes[0].Coverage.Reasons, "retrieval_expired_artifact")
	require.NotContains(t, strings.Join(fixture.calls, "\n"), "/21/zip")
}

func TestEpisodeCacheRegressionUnavailableUsageRetainsMembership(t *testing.T) {
	for _, status := range []int{404, 410} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			fixture := newEpisodeRetrievalFixture(t)
			fixture.add("org/one", 2, "canonical", "org/one", 1)
			fixture.add("org/one", 1, "canonical", "org/one", 1)
			request := func(ctx context.Context, host, endpoint string) ([]byte, error) {
				switch {
				case strings.Contains(endpoint, "/artifacts?"):
					id := int64(20)
					if strings.Contains(endpoint, "/runs/1/") {
						id = 10
					}
					return json.Marshal(map[string]any{"artifacts": []episodeArtifact{
						{ID: id, Name: "info"}, {ID: id + 1, Name: "usage"},
					}})
				case strings.HasSuffix(endpoint, "/11/zip"), strings.HasSuffix(endpoint, "/21/zip"):
					return nil, fmt.Errorf("gh: artifact unavailable (HTTP %d)", status)
				}
				return fixture.request(ctx, host, endpoint)
			}
			data, err := retrieveEpisodeLogs(context.Background(), episodeRetrievalTestOptions(t), request)
			require.NoError(t, err)
			require.ElementsMatch(t, []int64{1, 2}, data.Episodes[0].RunIDs)
			require.Len(t, data.EpisodeRetrieval.Issues, 2)
			require.Contains(t, data.Episodes[0].Coverage.Reasons, "retrieval_missing_artifact")
		})
	}
}

func TestEpisodeCacheRegressionPriorAttemptExcluded(t *testing.T) {
	fixture := newEpisodeRetrievalFixture(t)
	fixture.add("org/one", 2, "canonical", "org/one", 2)
	run := fixture.runs["org/one/2"]
	run.RunAttempt = 2
	fixture.runs["org/one/2"] = run
	oldArchive := episodeCacheRegressionArchive(t, AwInfo{
		RunID: 2, RunAttempt: "1", Context: &AwContext{EpisodeID: "old-episode", Repo: "other/repo", RunID: "99"},
	}, 1000)
	currentArchive := episodeCacheRegressionArchive(t, AwInfo{
		RunID: 2, RunAttempt: "2", Context: fixture.contexts[2],
	}, 10)
	request := func(ctx context.Context, host, endpoint string) ([]byte, error) {
		switch {
		case strings.Contains(endpoint, "/artifacts?"):
			return json.Marshal(map[string]any{"artifacts": []episodeArtifact{
				{ID: 20, Name: "old-info"}, {ID: 21, Name: "current-info"},
				{ID: 22, Name: "old-usage"}, {ID: 23, Name: "current-usage"},
			}})
		case strings.HasSuffix(endpoint, "/20/zip"), strings.HasSuffix(endpoint, "/22/zip"):
			return oldArchive, nil
		case strings.HasSuffix(endpoint, "/21/zip"):
			return episodeCacheRegressionArchive(t, AwInfo{RunID: 2, RunAttempt: "2", Context: fixture.contexts[2]}, 0), nil
		case strings.HasSuffix(endpoint, "/23/zip"):
			return currentArchive, nil
		}
		return fixture.request(ctx, host, endpoint)
	}
	data, err := retrieveEpisodeLogs(context.Background(), episodeRetrievalTestOptions(t), request)
	require.NoError(t, err)
	require.Equal(t, "canonical", data.Episodes[0].EpisodeID)
	require.Equal(t, 12, data.Summary.TotalTokens)
	require.Empty(t, data.EpisodeRetrieval.Issues)
	require.Len(t, data.Runs[0].EpisodeHops, 1)
	require.Equal(t, "org/one", data.Runs[0].EpisodeHops[0].CallerRepository)
	require.Equal(t, "2", data.Runs[0].EpisodeHops[0].CallerRunID)
	dirs, present, err := readEpisodeArtifactDirs(data.Runs[0].LogsPath)
	require.NoError(t, err)
	require.True(t, present)
	require.Len(t, dirs, 2)
	for _, dir := range dirs {
		require.NotContains(t, []string{"20-info", "22-usage"}, filepath.Base(dir))
	}
}

func TestEpisodeCacheRegressionConcurrentPublicationPreservesReaders(t *testing.T) {
	runDir := t.TempDir()
	run := WorkflowRun{Repository: "org/one", DatabaseID: 2, Attempt: 1}
	artifact := episodeArtifact{ID: 20, Name: "usage"}
	archive := episodeCacheRegressionArchive(t, AwInfo{RunID: 2, RunAttempt: "1", Context: &AwContext{EpisodeID: "canonical"}}, 10)
	secondDownloading := make(chan struct{})
	firstPublished := make(chan struct{})
	first := &episodeCollector{opts: EpisodeRetrievalOptions{MaxAPIRequests: 10}, request: func(context.Context, string, string) ([]byte, error) {
		<-secondDownloading
		return archive, nil
	}}
	second := &episodeCollector{opts: EpisodeRetrievalOptions{MaxAPIRequests: 10}, request: func(context.Context, string, string) ([]byte, error) {
		close(secondDownloading)
		<-firstPublished
		return archive, nil
	}}
	result := make(chan error, 1)
	go func() {
		_, err := second.loadArtifact(context.Background(), run, artifact, runDir)
		result <- err
	}()
	dir, err := first.loadArtifact(context.Background(), run, artifact, runDir)
	require.NoError(t, err)
	reader, err := os.Open(filepath.Join(dir, "token_usage.jsonl"))
	require.NoError(t, err)
	defer reader.Close()
	before, err := reader.Stat()
	require.NoError(t, err)
	close(firstPublished)
	require.NoError(t, <-result)
	after, err := os.Stat(filepath.Join(dir, "token_usage.jsonl"))
	require.NoError(t, err)
	require.True(t, os.SameFile(before, after), "a second publisher must not remove a completed reader's evidence")
}

func TestEpisodeCacheRegressionCollectionInventoriesRemainIndependent(t *testing.T) {
	fixture := newEpisodeRetrievalFixture(t)
	fixture.add("org/one", 2, "canonical", "org/one", 2)
	opts := episodeRetrievalTestOptions(t)
	first, err := retrieveEpisodeLogs(context.Background(), opts, fixture.request)
	require.NoError(t, err)
	request := func(ctx context.Context, host, endpoint string) ([]byte, error) {
		if strings.Contains(endpoint, "/artifacts?") {
			return json.Marshal(map[string]any{"artifacts": []episodeArtifact{{ID: 21, Name: "usage"}}})
		}
		if strings.HasSuffix(endpoint, "/21/zip") {
			return episodeCacheRegressionArchive(t, AwInfo{RunID: 2, RunAttempt: "1", Context: fixture.contexts[2]}, 25), nil
		}
		return fixture.request(ctx, host, endpoint)
	}
	second, err := retrieveEpisodeLogs(context.Background(), opts, request)
	require.NoError(t, err)
	require.NotEqual(t, first.Runs[0].LogsPath, second.Runs[0].LogsPath)
	firstDirs, _, err := readEpisodeArtifactDirs(first.Runs[0].LogsPath)
	require.NoError(t, err)
	secondDirs, _, err := readEpisodeArtifactDirs(second.Runs[0].LogsPath)
	require.NoError(t, err)
	require.Equal(t, "20-usage", filepath.Base(firstDirs[0]))
	require.Equal(t, "21-usage", filepath.Base(secondDirs[0]))
	firstUsage, err := analyzeTokenUsage(first.Runs[0].LogsPath, false)
	require.NoError(t, err)
	require.Equal(t, 12, firstUsage.TotalTokens())
	secondUsage, err := analyzeTokenUsage(second.Runs[0].LogsPath, false)
	require.NoError(t, err)
	require.Equal(t, 27, secondUsage.TotalTokens())
}

func TestEpisodeCacheRegressionConflictingUsageStillRejected(t *testing.T) {
	fixture := newEpisodeRetrievalFixture(t)
	fixture.add("org/one", 2, "canonical", "org/one", 2)
	conflicting := episodeCacheRegressionArchive(t, AwInfo{RunID: 999, RunAttempt: "1", Context: fixture.contexts[2]}, 10)
	request := func(ctx context.Context, host, endpoint string) ([]byte, error) {
		if strings.Contains(endpoint, "/artifacts?") {
			return json.Marshal(map[string]any{"artifacts": []episodeArtifact{{ID: 20, Name: "info"}, {ID: 21, Name: "usage"}}})
		}
		if strings.HasSuffix(endpoint, "/21/zip") {
			return conflicting, nil
		}
		return fixture.request(ctx, host, endpoint)
	}
	_, err := retrieveEpisodeLogs(context.Background(), episodeRetrievalTestOptions(t), request)
	require.ErrorContains(t, err, "conflicts with the requested repository/run identity")
	require.NotErrorIs(t, err, errEpisodeArtifactExpired)
}

func TestEpisodeCacheRegressionInventoryIgnoresArchiveManifests(t *testing.T) {
	root := t.TempDir()
	artifactDir := filepath.Join(root, "20-usage")
	require.NoError(t, os.Mkdir(artifactDir, 0o700))
	require.NoError(t, writeSensitiveFile(filepath.Join(artifactDir, ".episode-complete"), []byte("v1\n")))
	require.NoError(t, writeSensitiveFile(filepath.Join(artifactDir, ".episode-artifacts.json"), []byte(`["../99-usage"]`)))
	dirs, present, err := readEpisodeArtifactDirs(artifactDir)
	require.NoError(t, err)
	require.False(t, present)
	require.Empty(t, dirs)

	collectionDir := filepath.Join(root, "renamed-report")
	require.NoError(t, os.Mkdir(collectionDir, 0o700))
	manifest := filepath.Join(collectionDir, ".episode-artifacts.json")
	require.NoError(t, writeSensitiveFile(manifest, []byte(`["../20-usage"]`)))
	dirs, present, err = readEpisodeArtifactDirs(collectionDir)
	require.NoError(t, err)
	require.True(t, present)
	require.Equal(t, []string{artifactDir}, dirs)

	require.NoError(t, writeSensitiveFile(manifest, []byte(`["../../20-usage"]`)))
	_, present, err = readEpisodeArtifactDirs(collectionDir)
	require.True(t, present)
	require.ErrorContains(t, err, "invalid episode artifact inventory entry")
}
