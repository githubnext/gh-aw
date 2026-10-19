package workqueue

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestBranchPublicationExpirationRetryRejectsNonextendingHistory(t *testing.T) {
	for _, change := range []string{"deleted", "truncated", "rewritten"} {
		t.Run(change, func(t *testing.T) {
			branch, mock := newQueueAPI(t)
			commits, _, _ := expiringWorkFixture(t)
			installMockLog(t, mock, commits)
			clockReads := 0
			branch.publicationNow = func() int64 {
				clockReads++
				if clockReads == 1 {
					return 3499
				}
				if clockReads == 2 {
					switch change {
					case "deleted":
						mock.head = ""
					case "truncated":
						installMockLog(t, mock, commits[:1])
					case "rewritten":
						rewritten := testGenesis(t, nil)
						rewritten = testSubmit(t, rewritten, "rewritten-submit", testNode(t, rewritten, "rewritten"))
						installMockLog(t, mock, rewritten)
					}
				}
				return 3500
			}
			actor := testActor("administrator")
			request, err := NewRequest("expiration-retry", "dispatch_next", actor, DispatchParameters{
				Pool: "default", MaxClaims: 1, MaxDispatches: 1, MaxBytes: 48 << 10,
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := branch.Publish(context.Background(), actor, request); err == nil ||
				!strings.HasPrefix(err.Error(), "ledger_nonextending:") {
				t.Fatalf("expiration retry accepted %s history: %v", change, err)
			}
			if clockReads != 2 || mock.refWrites != 0 {
				t.Fatalf("nonextending history reached regeneration/publication: clock reads=%d, writes=%d", clockReads, mock.refWrites)
			}
		})
	}
}

func TestBranchPublicationExpirationRetryRegeneratesUnchangedHead(t *testing.T) {
	branch, mock := newQueueAPI(t)
	commits, old, fresh := expiringWorkFixture(t)
	installMockLog(t, mock, commits)
	head := mock.head
	clockReads := 0
	branch.publicationNow = func() int64 {
		clockReads++
		if mock.refWrites != 0 {
			t.Fatal("expiration retry wrote before regenerating its candidate")
		}
		if mock.head != head {
			t.Fatal("expiration retry unexpectedly changed HEAD before publication")
		}
		if clockReads == 1 {
			return 3499
		}
		return 3500
	}
	actor := testActor("administrator")
	request, err := NewRequest("expiration-retry", "dispatch_next", actor, DispatchParameters{
		Pool: "default", MaxClaims: 1, MaxDispatches: 1, MaxBytes: 48 << 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	published, err := branch.Publish(context.Background(), actor, request)
	if err != nil {
		t.Fatal(err)
	}
	if !published.Changed || published.Commit == nil || published.Commit.At != 3500 || clockReads != 4 || mock.refWrites != 1 {
		t.Fatalf("expiration retry did not regenerate once before writing: %+v, clock reads=%d, writes=%d", published, clockReads, mock.refWrites)
	}
	var claim ClaimOperation
	if len(published.Commit.Operations) != 1 || json.Unmarshal(published.Commit.Operations[0], &claim) != nil || claim.WorkID != fresh.WorkID {
		t.Fatal("regenerated candidate retained the expired first selection")
	}
	accepted, err := branch.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	state, err := Replay(accepted)
	if err != nil || state.Works[old.WorkID].CancellationReason != "work_expired" ||
		len(state.Claims) != 1 || len(state.Requests) != len(commits)+1 {
		t.Fatalf("expiration retry lost history or double-charged a grant: %v", err)
	}
	branch.publicationNow = func() int64 { return 4000 }
	again, err := branch.Publish(context.Background(), actor, request)
	if err != nil || again.Changed || again.Commit == nil || again.Commit.ID != published.Commit.ID || mock.refWrites != 1 {
		t.Fatalf("stable request recovery republished the regenerated grant: %+v %v", again, err)
	}
}
