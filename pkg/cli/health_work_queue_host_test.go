//go:build !integration

package cli

import (
	"context"
	"testing"

	"github.com/github/gh-aw/pkg/workqueue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthWorkQueueEnterpriseResourceReferences(t *testing.T) {
	const host = "contoso.example"
	t.Setenv("GITHUB_SERVER_URL", "https://"+host)
	state, commits, _ := workOperatorFixture(t, false)
	node, err := workqueue.NewWork([]byte(`{"task":"enterprise"}`), "factory", "enterprise", "default", *state.Policy, 3000)
	require.NoError(t, err)
	node.Subject = &workqueue.Resource{
		Kind: "issue", Host: host, Repository: "owner/repo",
		RepositoryID: "1", ResourceID: "2", Number: "3",
	}
	node.DependsOn = []workqueue.Dependency{{
		Kind: "pull_request", Condition: "merged",
		Resource: &workqueue.Resource{
			Kind: "pull_request", Host: host, Repository: "owner/repo",
			RepositoryID: "1", ResourceID: "4", Number: "5",
		},
	}}
	actor := workqueue.Actor{Role: "administrator", Principal: "1001", Repository: "owner/repo"}
	request, err := workqueue.NewRequest("enterprise-submit", "submit", actor, workqueue.SubmitParameters{
		Nodes: []workqueue.WorkDefinition{node},
	})
	require.NoError(t, err)
	commits, _, _, err = workqueue.BuildCandidate(commits, actor, request, 3000)
	require.NoError(t, err)

	t.Setenv("GITHUB_SERVER_URL", "")
	collection := healthWorkQueueFixture(t)
	collection.Host = host
	healthReadWorkQueue = func(_ context.Context, branch workqueue.Branch) ([]workqueue.QueueCommit, error) {
		assert.Equal(t, collection.Host, branch.Host)
		return commits, nil
	}
	stats, err := readHealthWorkQueueStats("owner/repo", collection)
	require.NoError(t, err)
	assert.Equal(t, 4, stats.Available)
	require.Len(t, stats.Pools, 1)
	assert.Equal(t, 3, stats.Pools[0].SchedulingEligible)

	collection.Host = "github.com"
	_, err = readHealthWorkQueueStats("owner/repo", collection)
	require.ErrorContains(t, err, "resource_unauthorized")
}
