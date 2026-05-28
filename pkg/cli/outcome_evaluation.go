package cli

import "strings"

// OutcomeStatus is the normalized classification for a safe output outcome.
type OutcomeStatus string

const (
	OutcomeStatusAccepted OutcomeStatus = "accepted"
	OutcomeStatusRejected OutcomeStatus = "rejected"
	OutcomeStatusPending  OutcomeStatus = "pending"
	OutcomeStatusIgnored  OutcomeStatus = "ignored"
	OutcomeStatusSkipped  OutcomeStatus = "skipped"
	OutcomeStatusUnknown  OutcomeStatus = "unknown"
)

// EvidenceStrength describes how confidently the outcome can be inferred.
type EvidenceStrength string

const (
	EvidenceStrong EvidenceStrength = "strong"
	EvidenceMedium EvidenceStrength = "medium"
	EvidenceWeak   EvidenceStrength = "weak"
	EvidenceNone   EvidenceStrength = "none"
)

// OutcomeEvaluation is the shared normalized outcome model.
type OutcomeEvaluation struct {
	OutcomeStatus    OutcomeStatus    `json:"outcome_status"`
	EvidenceStrength EvidenceStrength `json:"evidence_strength"`
	Signal           string           `json:"signal,omitempty"`
}

func normalizeOutcomeEvaluation(report OutcomeReport) OutcomeEvaluation {
	if report.OutcomeStatus != "" && report.EvidenceStrength != "" {
		return report.OutcomeEvaluation
	}

	if report.EvalError != "" || report.Result == OutcomeError {
		return OutcomeEvaluation{
			OutcomeStatus:    OutcomeStatusUnknown,
			EvidenceStrength: EvidenceWeak,
			Signal:           "evaluation_error",
		}
	}

	detail := strings.ToLower(strings.TrimSpace(report.Detail))

	switch {
	case strings.Contains(detail, "object still exists"):
		return OutcomeEvaluation{OutcomeStatus: OutcomeStatusUnknown, EvidenceStrength: EvidenceWeak, Signal: "target_exists_only"}
	case strings.Contains(detail, "closed without merge"):
		return OutcomeEvaluation{OutcomeStatus: OutcomeStatusRejected, EvidenceStrength: EvidenceStrong, Signal: "closed_without_merge"}
	case strings.Contains(detail, "closed as not planned"):
		return OutcomeEvaluation{OutcomeStatus: OutcomeStatusRejected, EvidenceStrength: EvidenceStrong, Signal: "closed_not_planned"}
	case strings.Contains(detail, "closed by bot"):
		return OutcomeEvaluation{OutcomeStatus: OutcomeStatusUnknown, EvidenceStrength: EvidenceMedium, Signal: "lifecycle"}
	case strings.Contains(detail, "merged"):
		return OutcomeEvaluation{OutcomeStatus: OutcomeStatusAccepted, EvidenceStrength: EvidenceStrong, Signal: "merged"}
	case strings.Contains(detail, "reopened"):
		return OutcomeEvaluation{OutcomeStatus: OutcomeStatusRejected, EvidenceStrength: EvidenceStrong, Signal: "reopened"}
	case strings.Contains(detail, "deleted"):
		return OutcomeEvaluation{OutcomeStatus: OutcomeStatusRejected, EvidenceStrength: EvidenceStrong, Signal: "deleted"}
	case strings.Contains(detail, "completed"):
		return OutcomeEvaluation{OutcomeStatus: OutcomeStatusAccepted, EvidenceStrength: EvidenceStrong, Signal: "completed"}
	case strings.Contains(detail, "milestone still assigned"):
		return OutcomeEvaluation{OutcomeStatus: OutcomeStatusAccepted, EvidenceStrength: EvidenceMedium, Signal: "milestone_assigned"}
	case strings.Contains(detail, "milestone removed"):
		return OutcomeEvaluation{OutcomeStatus: OutcomeStatusRejected, EvidenceStrength: EvidenceMedium, Signal: "milestone_removed"}
	case strings.Contains(detail, "reviews submitted"):
		return OutcomeEvaluation{OutcomeStatus: OutcomeStatusAccepted, EvidenceStrength: EvidenceMedium, Signal: "reviewed"}
	case strings.Contains(detail, "awaiting review"), strings.Contains(detail, "no reviews yet"):
		return OutcomeEvaluation{OutcomeStatus: OutcomeStatusPending, EvidenceStrength: EvidenceMedium, Signal: "awaiting_review"}
	case strings.Contains(detail, "no engagement"):
		return OutcomeEvaluation{OutcomeStatus: OutcomeStatusIgnored, EvidenceStrength: EvidenceMedium, Signal: "no_engagement"}
	case strings.Contains(detail, "human comments"), strings.Contains(detail, "with comments"):
		return OutcomeEvaluation{OutcomeStatus: OutcomeStatusPending, EvidenceStrength: EvidenceMedium, Signal: "acted_on"}
	case strings.Contains(detail, "open"):
		return OutcomeEvaluation{OutcomeStatus: OutcomeStatusPending, EvidenceStrength: EvidenceMedium, Signal: "open"}
	case strings.Contains(detail, "closed"):
		if report.Result == OutcomeRejected {
			return OutcomeEvaluation{OutcomeStatus: OutcomeStatusRejected, EvidenceStrength: EvidenceStrong, Signal: "closed"}
		}
		return OutcomeEvaluation{OutcomeStatus: OutcomeStatusAccepted, EvidenceStrength: EvidenceStrong, Signal: "closed"}
	}

	switch report.Result {
	case OutcomeAccepted:
		return OutcomeEvaluation{OutcomeStatus: OutcomeStatusAccepted, EvidenceStrength: EvidenceMedium, Signal: "acted_on"}
	case OutcomeRejected:
		return OutcomeEvaluation{OutcomeStatus: OutcomeStatusRejected, EvidenceStrength: EvidenceMedium, Signal: "rejected"}
	case OutcomePending:
		return OutcomeEvaluation{OutcomeStatus: OutcomeStatusPending, EvidenceStrength: EvidenceMedium, Signal: "pending"}
	case OutcomeIgnored:
		return OutcomeEvaluation{OutcomeStatus: OutcomeStatusIgnored, EvidenceStrength: EvidenceMedium, Signal: "ignored"}
	case OutcomeUnknown:
		return OutcomeEvaluation{OutcomeStatus: OutcomeStatusUnknown, EvidenceStrength: EvidenceWeak, Signal: "unknown"}
	default:
		return OutcomeEvaluation{OutcomeStatus: OutcomeStatusUnknown, EvidenceStrength: EvidenceWeak, Signal: "unknown"}
	}
}
