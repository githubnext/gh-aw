package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/workqueue"
	"github.com/spf13/cobra"
)

var workClearConfirmAction = console.ConfirmAction

func workConfirmClear(cmd *cobra.Command, branch workqueue.Branch, state workqueue.Projection, targets []string, hard bool) error {
	preview := workClearPreview(branch, state, targets, hard)
	if _, err := fmt.Fprintln(cmd.ErrOrStderr(), preview); err != nil {
		return fmt.Errorf("clear_preview_failed: queue unchanged: %w", err)
	}
	confirmed, err := workClearConfirmAction("Apply this queue clear?", "Yes, clear this snapshot", "No, keep queue")
	if err != nil {
		return fmt.Errorf("clear_confirmation_failed: queue unchanged: %w", err)
	}
	if !confirmed {
		return errors.New("clear_cancelled: confirmation declined; queue unchanged")
	}
	return nil
}

func workClearPreview(branch workqueue.Branch, state workqueue.Projection, targets []string, hard bool) string {
	lines := []string{
		fmt.Sprintf("Queue clear preview: repository=%s branch=%s\nSnapshot: %s",
			workQueueText(branch.Remote), workQueueText(branch.Name), workQueueText(state.Tip)),
		fmt.Sprintf("Work: %d; available: %d; claimed: %d; completed: %d; cancelled: %d\nAdmission paused: %t; grants paused: %t",
			state.Stats.Work, state.Stats.Available, state.Stats.Claimed, state.Stats.Completed, state.Stats.Cancelled,
			state.AdmissionPaused, state.GrantsPaused),
	}
	reservations := 0
	for _, dispatch := range state.Dispatches {
		if !dispatch.Released {
			reservations++
		}
	}
	lines = append(lines, fmt.Sprintf("Claims: %d; unreleased native reservations: %d", len(state.Claims), reservations))
	selected := make(map[string]struct{}, len(targets))
	for _, id := range targets {
		selected[id] = struct{}{}
	}
	if hard {
		lines = append(lines, "HARD RESET: remove all Work and reset identities and fairness debt.\nOld identities may be resubmitted and settled effects may repeat. Git ancestry is retained.")
	} else {
		lines = append(lines, fmt.Sprintf("Cancel %d selected Work; retain pending deliveries, history and identities.\nNative runs are not stopped or released. Admission and grants remain unchanged.", len(targets)))
	}
	for _, work := range workQueueOrderedWork(state, workQueueFilter{}) {
		action := "RETAIN"
		_, cancel := selected[work.WorkID]
		if hard {
			action = "REMOVE"
		} else if cancel {
			action = "CANCEL"
		}
		lines = append(lines, fmt.Sprintf("%s %s state=%s delivery=%s claim=%s", action,
			workQueueText(work.WorkID), workQueueText(work.State), workQueueText(work.Barrier), workQueueText(work.ClaimID)))
	}
	return strings.Join(lines, "\n")
}
