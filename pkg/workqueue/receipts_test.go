package workqueue

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssignmentToReceipt(t *testing.T) {
	t.Parallel()
	assignment := Assignment{
		Version: 3, DispatchID: "dispatch", RequestID: "request", CommitID: "commit",
		PolicyEpoch: "epoch", Pool: "pool", WorkerProfile: "profile",
		Claims: []AssignmentClaim{
			{Handle: "h2", ClaimID: "c2", WorkID: "w2", Work: json.RawMessage(`{"private":"payload"}`),
				ResultRefs: []ResultReference{{WorkID: "parent", ResultCommitID: "result",
					Descriptor: json.RawMessage(`{"private":"descriptor"}`)}}},
			{Handle: "h1", ClaimID: "c1", WorkID: "w1"},
		},
	}
	receipt := assignment.ToReceipt()
	assert.Equal(t, AssignmentReceipt{
		DispatchID: "dispatch", RequestID: "request", CommitID: "commit",
		Claims: []ClaimReceipt{
			{Handle: "h2", ClaimID: "c2", WorkID: "w2"},
			{Handle: "h1", ClaimID: "c1", WorkID: "w1"},
		},
	}, receipt)
	data, err := json.Marshal(receipt)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"dispatch_id":"dispatch","request_id":"request","commit_id":"commit","state":"","released":false,
		"claims":[
			{"claim_handle":"h2","claim_id":"c2","work_id":"w2","state":"","delivery_barrier":""},
			{"claim_handle":"h1","claim_id":"c1","work_id":"w1","state":"","delivery_barrier":""}
		]
	}`, string(data))
	receipt.Claims[0].Handle = "changed"
	assert.Equal(t, "h2", assignment.Claims[0].Handle, "receipts must not alias assignment claims")
}

func TestAssignmentToReceiptPreservesClaimsShape(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name   string
		claims []AssignmentClaim
		want   string
	}{
		{name: "nil", want: "null"},
		{name: "empty", claims: []AssignmentClaim{}, want: "[]"},
	} {
		t.Run(test.name, func(t *testing.T) {
			receipt := (Assignment{Claims: test.claims}).ToReceipt()
			data, err := json.Marshal(receipt)
			require.NoError(t, err)
			assert.JSONEq(t, `{"dispatch_id":"","request_id":"","commit_id":"","state":"","released":false,"claims":`+test.want+`}`, string(data))
		})
	}
}

func TestAssignmentClaimToReceipt(t *testing.T) {
	t.Parallel()
	claim := AssignmentClaim{
		Handle: "handle", ClaimID: "claim", WorkID: "work",
		Work:       json.RawMessage(`{"private":"payload"}`),
		ResultRefs: []ResultReference{{Descriptor: json.RawMessage(`{"private":"descriptor"}`)}},
	}
	receipt := claim.ToReceipt()
	assert.Equal(t, ClaimReceipt{Handle: "handle", ClaimID: "claim", WorkID: "work"}, receipt)
	data, err := json.Marshal(receipt)
	require.NoError(t, err)
	assert.JSONEq(t, `{"claim_handle":"handle","claim_id":"claim","work_id":"work","state":"","delivery_barrier":""}`, string(data))
	assert.Equal(t, ClaimReceipt{}, (AssignmentClaim{}).ToReceipt())
}
