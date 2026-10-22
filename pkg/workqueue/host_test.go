package workqueue

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestExplicitResourceHostReadAndReplay(t *testing.T) {
	const host = "contoso.example"
	for _, checkpointed := range []bool{false, true} {
		name := "history"
		if checkpointed {
			name = "checkpoint"
		}
		t.Run(name, func(t *testing.T) {
			t.Setenv("GITHUB_SERVER_URL", "https://"+host)
			branch, _ := gitQueueBranch(t)
			branch.Host = host
			issue := Resource{
				Kind: "issue", Host: host, Repository: testRepository,
				RepositoryID: "1", ResourceID: "2", Number: "3",
			}
			pullRequest := issue
			pullRequest.Kind, pullRequest.ResourceID, pullRequest.Number = "pull_request", "4", "5"
			commits := testGenesis(t, func(policy *Policy) {
				policy.Projectors = []ProjectorRule{{
					Principal: testPrincipal, Workflow: defaultWorkerWorkflow, Ref: strings.Repeat("a", 40),
					Pools: []string{"default"}, Repositories: []string{testRepository},
					BackingIssues: []Resource{issue},
				}}
			})
			work := testNode(t, commits, "enterprise-host")
			work.Subject, work.BackingIssue = &issue, &issue
			work.DependsOn = []Dependency{{Kind: "pull_request", Resource: &pullRequest, Condition: "merged"}}
			commits = testSubmit(t, commits, "enterprise-host-submit", work)
			parent := writeRawQueueGit(t, branch, commits)
			if checkpointed {
				var err error
				commits, err = CompactCheckpoint(commits, parent, testActor("administrator"), 3000)
				if err != nil {
					t.Fatal(err)
				}
				writeRawQueueGit(t, branch, commits, parent)
			}
			for _, serverURL := range []string{"", "https://another.example"} {
				t.Run("server="+serverURL, func(t *testing.T) {
					t.Setenv("GITHUB_SERVER_URL", serverURL)
					read, err := branch.Read(context.Background())
					if err != nil || !sameJSON(read, commits) {
						t.Fatalf("explicit enterprise host did not validate ledger read: %v", err)
					}
					state, err := ReplayAtForHost(read, 4000, host)
					if err != nil {
						t.Fatalf("explicit enterprise host did not replay resource references: %v", err)
					}
					got := state.Works[work.WorkID]
					if got == nil || !sameJSON(got.Subject, &issue) || !sameJSON(got.DependsOn, work.DependsOn) {
						t.Fatal("enterprise subject or PR dependency was lost")
					}
					if os.Getenv("GITHUB_SERVER_URL") != serverURL {
						t.Fatal("explicit host validation changed process environment")
					}
					if _, err := Replay(read); err == nil {
						t.Fatal("default replay unexpectedly accepted a cross-host ledger")
					}
					for _, foreignHost := range []string{"github.com", "another.example"} {
						foreign := branch
						foreign.Host = foreignHost
						if _, err := foreign.Read(context.Background()); err == nil {
							t.Fatalf("ledger read accepted resources from another host: %s", foreignHost)
						}
						if _, err := ReplayAtForHost(read, 4000, foreignHost); err == nil {
							t.Fatalf("replay accepted resources from another host: %s", foreignHost)
						}
					}
				})
			}
		})
	}
}

func TestExplicitReplayHostValidation(t *testing.T) {
	t.Setenv("GITHUB_SERVER_URL", "")
	commits := testGenesis(t, nil)
	for _, host := range []string{"https://contoso.example", "contoso.example/path", "contoso.example:443", "user@contoso.example"} {
		if _, err := ReplayForHost(commits, host); err == nil {
			t.Fatalf("invalid explicit host accepted: %s", host)
		}
	}
	for _, host := range []string{"", "github.com", "CONTOSO.EXAMPLE"} {
		if _, err := ReplayForHost(commits, host); err != nil {
			t.Fatalf("valid explicit host rejected: %s: %v", host, err)
		}
	}
}

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
