package workqueue

import (
	"os"
	"strings"
	"testing"
)

func TestEnterpriseResourceHostValidationAndReplay(t *testing.T) {
	t.Setenv("GITHUB_SERVER_URL", "https://contoso.example")
	host, err := expectedResourceHost()
	if err != nil || host != "contoso.example" {
		t.Fatalf("expected configured enterprise host, got %q: %v", host, err)
	}
	resource := Resource{
		Kind: "issue", Host: host, Repository: testRepository,
		RepositoryID: "1", ResourceID: "2", Number: "3",
	}
	if err := validateResource(resource, PoolPolicy{AllowedRepositories: []string{testRepository}}); err != nil {
		t.Fatal(err)
	}
	if err := validateResource(Resource{Kind: "issue", Host: "github.com", Repository: testRepository,
		RepositoryID: "1", ResourceID: "2", Number: "3"}, PoolPolicy{AllowedRepositories: []string{testRepository}}); err == nil {
		t.Fatal("cross-host resources must remain rejected")
	}
	if err := validateEffectResource(EffectResource{
		"host": host, "repository": testRepository, "repository_id": "1",
	}); err != nil {
		t.Fatal(err)
	}
	if err := validateEffectResource(EffectResource{
		"host": "github.com", "repository": testRepository, "repository_id": "1",
	}); err == nil {
		t.Fatal("cross-host effect targets must remain rejected")
	}

	commits := testGenesis(t, nil)
	work := testNode(t, commits, "enterprise")
	work.Subject = &resource
	work.DependsOn = []Dependency{{Kind: "issue", Resource: &resource, Condition: "closed"}}
	commits = testSubmit(t, commits, "enterprise-submit", work)
	state, err := Replay(commits)
	if err != nil {
		t.Fatalf("enterprise resource ledger did not replay: %v", err)
	}
	if state.Works[work.WorkID].Subject == nil || state.Works[work.WorkID].Subject.Host != host {
		t.Fatal("enterprise Work subject was not preserved through replay")
	}

	if err := os.Setenv("GITHUB_SERVER_URL", "https://another.example"); err != nil {
		t.Fatal(err)
	}
	if _, err := Replay(commits); err == nil || !strings.Contains(err.Error(), "resource_unauthorized") {
		t.Fatalf("ledger replay under a different configured host was not rejected: %v", err)
	}
}

func TestExpectedResourceHostRejectsInvalidServerURLs(t *testing.T) {
	for _, serverURL := range []string{
		"http://contoso.example",
		"https://contoso.example:8443",
		"https://contoso.example:443",
		"https://user@contoso.example",
		"https://contoso.example/path",
		"https://contoso.example?query=1",
		"https://contoso.example#fragment",
	} {
		t.Run(serverURL, func(t *testing.T) {
			t.Setenv("GITHUB_SERVER_URL", serverURL)
			if _, err := expectedResourceHost(); err == nil {
				t.Fatal("invalid GITHUB_SERVER_URL was accepted")
			}
		})
	}
}
