package workqueue

import "encoding/json"

func workExpired(work *WorkState, at int64) bool {
	return work.ExpiresAt != nil && at >= *work.ExpiresAt &&
		work.State == "available" && work.Barrier != "pending" && work.Barrier != "failed"
}

// Expiration is a deterministic consequence of immutable deadlines and trusted
// committed time. It never closes a Claim or releases a native reservation.
func (state *Projection) advanceExpiration(at int64) {
	state.timeWatermark = max(state.timeWatermark, at)
	if state.expiringWorks == nil {
		state.expiringWorks = map[string]bool{}
		for id, work := range state.Works {
			if work.ExpiresAt != nil {
				state.expiringWorks[id] = true
			}
		}
	}
	for id := range state.expiringWorks {
		work := state.Works[id]
		if work.State == "completed" || work.State == "cancelled" {
			delete(state.expiringWorks, id)
			continue
		}
		if !workExpired(work, state.timeWatermark) {
			continue
		}
		work.State = "cancelled"
		work.CancellationReason = "work_expired"
		delete(state.expiringWorks, id)
	}
}

func expiringGrant(state Projection, operations []Operation, at int64) bool {
	for _, operation := range operations {
		var claim ClaimOperation
		if json.Unmarshal(operation, &claim) != nil || claim.Kind != "Claim" {
			continue
		}
		if work := state.Works[claim.WorkID]; work != nil && workExpired(work, max(at, state.timeWatermark)) {
			return true
		}
	}
	return false
}

// ReplayAt adds a disposable current-time view after validating historical
// decisions at committed time. It must not be used to construct checkpoints.
func ReplayAt(commits []QueueCommit, at int64) (Projection, error) {
	return ReplayAtForHost(commits, at, "")
}

// ReplayAtForHost adds a disposable current-time view using the selected resource
// host. An empty host preserves ReplayAt's GITHUB_SERVER_URL-based default.
func ReplayAtForHost(commits []QueueCommit, at int64, host string) (Projection, error) {
	if at < 0 || at > MaxTimestamp {
		return Projection{}, queueError("timestamp_invalid", "invalid replay timestamp")
	}
	state, err := ReplayForHost(commits, host)
	if err != nil {
		return state, err
	}
	state.advanceExpiration(at)
	state.Stats = Stats{}
	state.updateReplayStats(len(state.Requests))
	return state, nil
}
