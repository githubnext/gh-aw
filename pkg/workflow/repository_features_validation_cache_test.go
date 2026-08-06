//go:build !integration

package workflow

import (
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
)

type countingRoundTripper struct {
	callCount int32
	body      string
}

func (c *countingRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	atomic.AddInt32(&c.callCount, 1)
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(c.body)),
	}, nil
}

func TestCheckRepositoryHasDiscussionsUncachedWithClient_UsesDiskCacheAcrossClients(t *testing.T) {
	cacheDir := t.TempDir()

	newClient := func(rt http.RoundTripper) *api.GraphQLClient {
		t.Helper()
		client, err := api.NewGraphQLClient(api.ClientOptions{
			Host:        "github.com",
			AuthToken:   "test-token",
			EnableCache: true,
			CacheTTL:    repositoryFeaturesCacheTTL,
			CacheDir:    cacheDir,
			Transport:   rt,
		})
		if err != nil {
			t.Fatalf("failed to create GraphQL client: %v", err)
		}
		return client
	}

	transport1 := &countingRoundTripper{
		body: `{"data":{"repository":{"hasDiscussionsEnabled":true}}}`,
	}

	hasDiscussions, err := checkRepositoryHasDiscussionsUncachedWithClient("github/gh-aw", newClient(transport1))
	if err != nil {
		t.Fatalf("first query failed: %v", err)
	}
	if !hasDiscussions {
		t.Fatal("expected discussions to be enabled")
	}
	if got := atomic.LoadInt32(&transport1.callCount); got != 1 {
		t.Fatalf("expected first client to hit transport once, got %d", got)
	}

	transport2 := &countingRoundTripper{
		body: `{"data":{"repository":{"hasDiscussionsEnabled":true}}}`,
	}

	hasDiscussions, err = checkRepositoryHasDiscussionsUncachedWithClient("github/gh-aw", newClient(transport2))
	if err != nil {
		t.Fatalf("second query failed: %v", err)
	}
	if !hasDiscussions {
		t.Fatal("expected discussions to be enabled")
	}
	if got := atomic.LoadInt32(&transport2.callCount); got != 0 {
		t.Fatalf("expected second client to use disk cache (no transport calls), got %d", got)
	}
}

func TestCheckRepositoryHasIssuesUncachedWithClient_UsesDiskCacheAcrossClients(t *testing.T) {
	cacheDir := t.TempDir()

	newClient := func(rt http.RoundTripper) *api.RESTClient {
		t.Helper()
		client, err := api.NewRESTClient(api.ClientOptions{
			Host:        "github.com",
			AuthToken:   "test-token",
			EnableCache: true,
			CacheTTL:    repositoryFeaturesCacheTTL,
			CacheDir:    cacheDir,
			Transport:   rt,
		})
		if err != nil {
			t.Fatalf("failed to create REST client: %v", err)
		}
		return client
	}

	transport1 := &countingRoundTripper{
		body: `{"has_issues":true}`,
	}

	hasIssues, err := checkRepositoryHasIssuesUncachedWithClient("github/gh-aw", newClient(transport1))
	if err != nil {
		t.Fatalf("first query failed: %v", err)
	}
	if !hasIssues {
		t.Fatal("expected issues to be enabled")
	}
	if got := atomic.LoadInt32(&transport1.callCount); got != 1 {
		t.Fatalf("expected first client to hit transport once, got %d", got)
	}

	transport2 := &countingRoundTripper{
		body: `{"has_issues":true}`,
	}

	hasIssues, err = checkRepositoryHasIssuesUncachedWithClient("github/gh-aw", newClient(transport2))
	if err != nil {
		t.Fatalf("second query failed: %v", err)
	}
	if !hasIssues {
		t.Fatal("expected issues to be enabled")
	}
	if got := atomic.LoadInt32(&transport2.callCount); got != 0 {
		t.Fatalf("expected second client to use disk cache (no transport calls), got %d", got)
	}
}
