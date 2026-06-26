//go:build !integration

package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveVersionLabelDoesNotCacheFailedTagLoad(t *testing.T) {
	origRun := runGHVersionLabel
	t.Cleanup(func() {
		runGHVersionLabel = origRun
		clearVersionLabelCache()
	})
	clearVersionLabelCache()

	calls := 0
	runGHVersionLabel = func(ctx context.Context, msg string, args ...string) ([]byte, error) {
		calls++
		return nil, errors.New("transient error")
	}

	sha := "0123456789abcdef0123456789abcdef01234567"
	assert.Equal(t, shortRef(sha), resolveVersionLabel(context.Background(), "octo/repo", sha))
	assert.Equal(t, shortRef(sha), resolveVersionLabel(context.Background(), "octo/repo", sha))
	assert.Equal(t, 2, calls, "failed tag loads should not be cached")
}

func TestLoadRepoTagMapPaginatesAndKeepsFirstTagPerSHA(t *testing.T) {
	origRun := runGHVersionLabel
	t.Cleanup(func() { runGHVersionLabel = origRun })

	pageCalls := make([]string, 0, 3)
	runGHVersionLabel = func(ctx context.Context, msg string, args ...string) ([]byte, error) {
		require.Len(t, args, 2)
		endpoint := args[1]
		pageCalls = append(pageCalls, endpoint)

		switch {
		case strings.HasSuffix(endpoint, "page=1"):
			tags := make([]repoTagEntry, 0, 100)
			for i := range 100 {
				tags = append(tags, repoTagEntry{
					Name: fmt.Sprintf("v1.0.%d", i),
					Commit: struct {
						SHA string `json:"sha"`
					}{SHA: fmt.Sprintf("sha-page1-%d", i)},
				})
			}
			tags[0] = repoTagEntry{Name: "v2.0.0", Commit: struct {
				SHA string `json:"sha"`
			}{SHA: "same-sha"}}
			tags[1] = repoTagEntry{Name: "v1.9.9", Commit: struct {
				SHA string `json:"sha"`
			}{SHA: "same-sha"}}
			return mustMarshalTags(t, tags), nil
		case strings.HasSuffix(endpoint, "page=2"):
			return mustMarshalTags(t, []repoTagEntry{
				{Name: "v3.0.0", Commit: struct {
					SHA string `json:"sha"`
				}{SHA: "sha-page2"}},
			}), nil
		default:
			return []byte("[]"), nil
		}
	}

	tagMap, ok := loadRepoTagMap(context.Background(), "octo/repo")
	require.True(t, ok)
	assert.Equal(t, "v2.0.0", tagMap["same-sha"], "first tag should win for duplicate SHA")
	assert.Equal(t, "v3.0.0", tagMap["sha-page2"], "pagination should include tags from later pages")
	require.GreaterOrEqual(t, len(pageCalls), 2)
	assert.Contains(t, pageCalls[0], "page=1")
	assert.Contains(t, pageCalls[1], "page=2")
}

func mustMarshalTags(t *testing.T, tags []repoTagEntry) []byte {
	t.Helper()
	b, err := json.Marshal(tags)
	require.NoError(t, err)
	return b
}
