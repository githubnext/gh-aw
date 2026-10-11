package workqueue

// AssignmentReceipt describes assignment identity and captured lifecycle facts
// without carrying work payloads, result descriptors, or actor credentials.
type AssignmentReceipt struct {
	DispatchID string         `json:"dispatch_id"`
	RequestID  string         `json:"request_id"`
	CommitID   string         `json:"commit_id"`
	State      string         `json:"state"`
	Released   bool           `json:"released"`
	Claims     []ClaimReceipt `json:"claims"`
}

type ClaimReceipt struct {
	Handle  string `json:"claim_handle"`
	ClaimID string `json:"claim_id"`
	WorkID  string `json:"work_id"`
	State   string `json:"state"`
	Barrier string `json:"delivery_barrier"`
}

// ToReceipt projects immutable identity only, preserving source claim order.
// Callers must validate the assignment and populate lifecycle facts from their
// captured projection.
func (assignment Assignment) ToReceipt() AssignmentReceipt {
	receipt := AssignmentReceipt{
		DispatchID: assignment.DispatchID,
		RequestID:  assignment.RequestID,
		CommitID:   assignment.CommitID,
	}
	if assignment.Claims != nil {
		receipt.Claims = make([]ClaimReceipt, 0, len(assignment.Claims))
		for _, claim := range assignment.Claims {
			receipt.Claims = append(receipt.Claims, claim.ToReceipt())
		}
	}
	return receipt
}

// ToReceipt omits work payloads and result references; lifecycle fields are
// populated separately from the captured projection.
func (claim AssignmentClaim) ToReceipt() ClaimReceipt {
	return ClaimReceipt{
		Handle:  claim.Handle,
		ClaimID: claim.ClaimID,
		WorkID:  claim.WorkID,
	}
}
