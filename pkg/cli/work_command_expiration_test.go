package cli

import (
	"encoding/json"
	"testing"

	"github.com/github/gh-aw/pkg/workqueue"
)

func TestWorkConfigureSubmissionExpiration(t *testing.T) {
	policy := workqueue.DefaultPolicy("1001", "owner/repo")
	work, err := workqueue.NewWork(json.RawMessage(`{"task":"expiry"}`), "graph", "root", "default", policy, 1000)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"0", "3500", "9007199254740991", "-1", "9007199254740992"} {
		t.Run(value, func(t *testing.T) {
			cmd := workSubmitCommand()
			if err := cmd.Flags().Set("expires-at", value); err != nil {
				t.Fatal(err)
			}
			configured, err := workConfigureSubmission(cmd, workqueue.Projection{}, policy, work)
			if value == "-1" || value == "9007199254740992" {
				if err == nil {
					t.Fatal("accepted an invalid deadline")
				}
				return
			}
			if err != nil || configured.ExpiresAt == nil {
				t.Fatalf("deadline was not retained: %v", err)
			}
			state := workqueue.Projection{Works: map[string]*workqueue.WorkState{work.WorkID: {WorkDefinition: configured}}}
			reused, err := workConfigureSubmission(workSubmitCommand(), state, policy, work)
			if err != nil || reused.ExpiresAt == nil || *reused.ExpiresAt != *configured.ExpiresAt {
				t.Fatalf("omitted retry deadline was reset: %v", err)
			}
			if err := cmd.Flags().Set("expires-at", "3501"); err != nil {
				t.Fatal(err)
			}
			if _, err := workConfigureSubmission(cmd, state, policy, work); err == nil {
				t.Fatal("explicit retry deadline mutated an admitted Work")
			}
		})
	}
}
