package cli

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/workqueue"
)

type WorkQueueDiagnostics struct {
	workQueueStatsReport
	AssignedClaims []WorkQueueAssignedClaimDiagnostics `json:"assigned_claims,omitempty"`
}

type WorkQueueAssignedClaimDiagnostics struct {
	Work  workQueueWorkMetadata  `json:"work"`
	Claim workQueueClaimMetadata `json:"claim"`
}

func capturedWorkQueueDiagnostics(state workqueue.Projection, envelope currentWorkQueueEnvelope) (*WorkQueueDiagnostics, error) {
	summary, err := workQueueStatsAt(state, "", *envelope.CapturedAt)
	if err != nil {
		return nil, err
	}
	summary.Status = "captured_snapshot"
	if summary.Repository == "" {
		summary.Repository = envelope.Origin.Repository
	}
	result := &WorkQueueDiagnostics{workQueueStatsReport: summary}
	if envelope.Worker != nil {
		for _, member := range envelope.Worker.Claims {
			claim, err := workQueueClaimMetadataAt(state, state.Claims[member.ClaimID])
			if err != nil {
				return nil, err
			}
			result.AssignedClaims = append(result.AssignedClaims, WorkQueueAssignedClaimDiagnostics{
				Work: workQueueWorkMetadataAt(state, state.Works[member.WorkID]), Claim: claim,
			})
		}
	}
	return result, nil
}

func workQueueDiagnosticsText(diagnostics *WorkQueueDiagnostics) string {
	if diagnostics == nil {
		return ""
	}
	var output strings.Builder
	fmt.Fprintf(&output, "    diagnostics: evaluated_at=%d (activation snapshot; no live reads)\n", diagnostics.EvaluatedAt)                                                        //nolint:fprintferrorunchecked // strings.Builder writes cannot fail.
	fmt.Fprintf(&output, "    work: available=%d claimed=%d completed=%d cancelled=%d\n", diagnostics.Available, diagnostics.Claimed, diagnostics.Completed, diagnostics.Cancelled) //nolint:fprintferrorunchecked // strings.Builder writes cannot fail.
	fmt.Fprintf(&output, "    claims: open=%d completed=%d cancelled=%d | delivery: pending=%d verified=%d failed=%d\n",                                                            //nolint:fprintferrorunchecked // strings.Builder writes cannot fail.
		diagnostics.ClaimsByState["open"], diagnostics.ClaimsByState["completed"], diagnostics.ClaimsByState["cancelled"],
		diagnostics.DeliveryByState["pending"], diagnostics.DeliveryByState["verified"], diagnostics.DeliveryByState["failed"])
	fmt.Fprintf(&output, "    reservations: retained=%d historical_dispatches=%d (not live Actions status)\n", diagnostics.NativeReservations, diagnostics.DispatchesTotal)  //nolint:fprintferrorunchecked // strings.Builder writes cannot fail.
	fmt.Fprintf(&output, "    activity: transactions_24h=%d since=%d until=%d\n", diagnostics.Activity.Transactions, diagnostics.Activity.Since, diagnostics.Activity.Until) //nolint:fprintferrorunchecked // strings.Builder writes cannot fail.
	if reasons := workQueueReasonCounts(diagnostics.ClaimCancellationReasons); reasons != "" {
		fmt.Fprintf(&output, "    claim_cancellations: %s\n", reasons) //nolint:fprintferrorunchecked // strings.Builder writes cannot fail.
	}
	if reasons := workQueueReasonCounts(diagnostics.DeliveryFailureReasons); reasons != "" {
		fmt.Fprintf(&output, "    delivery_failures: %s (unknown effects must not be replayed)\n", reasons) //nolint:fprintferrorunchecked // strings.Builder writes cannot fail.
	}
	for _, pool := range diagnostics.Pools[:min(len(diagnostics.Pools), 16)] {
		output.WriteString(workQueuePoolDiagnosticsText(pool))
	}
	if len(diagnostics.Pools) > 16 {
		output.WriteString("    pools: additional pools available in JSON\n")
	}
	for _, assigned := range diagnostics.AssignedClaims[:min(len(diagnostics.AssignedClaims), 16)] {
		fmt.Fprintf(&output, "    claim_diagnostics: %s current=%t work_attempts=%d claim_retry_not_before=%d delivery=%s", //nolint:fprintferrorunchecked // strings.Builder writes cannot fail.
			escapeWorkQueueDisplay(assigned.Claim.ClaimID), assigned.Claim.Current, assigned.Work.Attempts, assigned.Claim.RetryNotBefore, escapeWorkQueueDisplay(assigned.Claim.Delivery))
		if assigned.Claim.GrantedAt != nil {
			fmt.Fprintf(&output, " granted_at=%d", *assigned.Claim.GrantedAt) //nolint:fprintferrorunchecked // strings.Builder writes cannot fail.
		}
		if assigned.Claim.DeliveryFailureReason != "" {
			fmt.Fprintf(&output, " failure=%s disposition=%s", escapeWorkQueueDisplay(assigned.Claim.DeliveryFailureReason), escapeWorkQueueDisplay(assigned.Claim.Disposition)) //nolint:fprintferrorunchecked // strings.Builder writes cannot fail.
		}
		output.WriteByte('\n')
	}
	if len(diagnostics.AssignedClaims) > 16 {
		output.WriteString("    claim_diagnostics: additional original Claims available in JSON\n")
	}
	output.WriteString("    Completion is not verified delivery; finish intents and later operation logs do not update this snapshot.\n")
	return output.String()
}

func workQueuePoolDiagnosticsText(pool workQueuePoolStats) string {
	if !pool.PolicyInstalled {
		return fmt.Sprintf("    pool: %s (historical only; not in captured Policy)\n", escapeWorkQueueDisplay(pool.Pool))
	}
	text := fmt.Sprintf("    pool: %s available=%d eligible=%d reservations=%d/%d prediction=%s (not a grant)",
		escapeWorkQueueDisplay(pool.Pool), pool.WorkByState["available"], pool.SchedulingEligible,
		pool.NativeReservations, pool.NativeLimit, escapeWorkQueueDisplay(pool.Prediction.Reason))
	if pool.OldestAvailable != nil {
		text += fmt.Sprintf(" oldest_work=%s age_ms=%d", escapeWorkQueueDisplay(pool.OldestAvailable.WorkID), pool.OldestAvailable.AgeMS)
	}
	if reasons := workQueueReasonCounts(pool.IneligibleReasons); reasons != "" {
		text += " blocked=" + reasons
	}
	return text + "\n"
}

func workQueueReasonCounts(counts map[string]int) string {
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys[:min(len(keys), 16)] {
		parts = append(parts, fmt.Sprintf("%s=%d", escapeWorkQueueDisplay(key), counts[key]))
	}
	if len(keys) > 16 {
		parts = append(parts, "... (additional reasons in JSON)")
	}
	return strings.Join(parts, ",")
}

func equalWorkQueueDiagnostics(left, right *WorkQueueDiagnostics) bool {
	if left == nil || right == nil {
		return left == right
	}
	return left.Stats == right.Stats && left.workQueueSnapshot == right.workQueueSnapshot &&
		left.PolicyMode == right.PolicyMode && left.NativeReservations == right.NativeReservations &&
		left.DispatchesTotal == right.DispatchesTotal &&
		equalWorkQueueDiagnosticCounts(left.ClaimsByState, right.ClaimsByState) &&
		equalWorkQueueDiagnosticCounts(left.ClaimCancellationReasons, right.ClaimCancellationReasons) &&
		equalWorkQueueDiagnosticCounts(left.DeliveryByState, right.DeliveryByState) &&
		equalWorkQueueDiagnosticCounts(left.DeliveryFailureReasons, right.DeliveryFailureReasons) &&
		equalWorkQueueDiagnosticCounts(left.DispatchesByState, right.DispatchesByState) &&
		equalWorkQueueDiagnosticCounts(left.ReservationsByState, right.ReservationsByState) &&
		left.Activity.Since == right.Activity.Since && left.Activity.Until == right.Activity.Until &&
		left.Activity.Transactions == right.Activity.Transactions && left.Activity.LastCommitID == right.Activity.LastCommitID &&
		equalWorkQueueReceiptPointers(left.Activity.LastAt, right.Activity.LastAt) &&
		equalWorkQueueDiagnosticCounts(left.Activity.Operations, right.Activity.Operations) &&
		(left.Pools == nil) == (right.Pools == nil) &&
		slices.EqualFunc(left.Pools, right.Pools, equalWorkQueuePoolStats) &&
		(left.AssignedClaims == nil) == (right.AssignedClaims == nil) &&
		slices.EqualFunc(left.AssignedClaims, right.AssignedClaims, equalWorkQueueAssignedClaimDiagnostics)
}

func equalWorkQueuePoolStats(left, right workQueuePoolStats) bool {
	return left.Pool == right.Pool && left.PolicyInstalled == right.PolicyInstalled &&
		left.NativeReservations == right.NativeReservations && left.LogicalLimit == right.LogicalLimit &&
		left.NativeLimit == right.NativeLimit && left.MaxAttempts == right.MaxAttempts &&
		left.SchedulingEligible == right.SchedulingEligible && left.UnknownAge == right.UnknownAge &&
		equalWorkQueueReceiptPointers(left.OldestAvailable, right.OldestAvailable) &&
		equalWorkQueueDiagnosticCounts(left.WorkByState, right.WorkByState) && equalWorkQueueDiagnosticCounts(left.ClaimsByState, right.ClaimsByState) &&
		equalWorkQueueDiagnosticCounts(left.DeliveryByState, right.DeliveryByState) && equalWorkQueueDiagnosticCounts(left.IneligibleReasons, right.IneligibleReasons) &&
		left.Prediction.Reason == right.Prediction.Reason &&
		(left.Prediction.Assignments == nil) == (right.Prediction.Assignments == nil) &&
		slices.EqualFunc(left.Prediction.Assignments, right.Prediction.Assignments, func(a, b workQueueReplayAssignment) bool {
			return a.WorkerProfile == b.WorkerProfile && (a.WorkIDs == nil) == (b.WorkIDs == nil) && slices.Equal(a.WorkIDs, b.WorkIDs)
		})
}

func equalWorkQueueDiagnosticCounts(left, right map[string]int) bool {
	return (left == nil) == (right == nil) && maps.Equal(left, right)
}

func equalWorkQueueAssignedClaimDiagnostics(left, right WorkQueueAssignedClaimDiagnostics) bool {
	a, b := left.Claim, right.Claim
	a.GrantedAt, a.TerminalAt, b.GrantedAt, b.TerminalAt = nil, nil, nil, nil
	x, y := left.Work, right.Work
	return a == b && equalWorkQueueReceiptPointers(left.Claim.GrantedAt, right.Claim.GrantedAt) &&
		equalWorkQueueReceiptPointers(left.Claim.TerminalAt, right.Claim.TerminalAt) &&
		x.WorkID == y.WorkID && x.GraphID == y.GraphID && x.NodeKey == y.NodeKey && x.Pool == y.Pool &&
		x.WorkerProfile == y.WorkerProfile && x.State == y.State && x.Priority == y.Priority &&
		x.AdmittedPriority == y.AdmittedPriority && x.FairnessKey == y.FairnessKey && x.Position == y.Position &&
		x.EnqueuedAt == y.EnqueuedAt && x.Attempts == y.Attempts && x.RetryNotBefore == y.RetryNotBefore &&
		x.ClaimID == y.ClaimID && x.Delivery == y.Delivery && x.CompletionID == y.CompletionID &&
		x.ResultCommitID == y.ResultCommitID && x.Disposition == y.Disposition && x.CancellationReason == y.CancellationReason &&
		equalWorkQueueReceiptPointers(x.AdmittedAt, y.AdmittedAt) && equalWorkQueueReceiptPointers(x.MaxAttempts, y.MaxAttempts) &&
		equalWorkQueueReceiptPointers(x.Issue, y.Issue) && (x.DependsOn == nil) == (y.DependsOn == nil) &&
		slices.EqualFunc(x.DependsOn, y.DependsOn, func(a, b workqueue.Dependency) bool {
			return a.Kind == b.Kind && a.WorkID == b.WorkID && a.Condition == b.Condition && equalWorkQueueReceiptPointers(a.Resource, b.Resource)
		})
}
