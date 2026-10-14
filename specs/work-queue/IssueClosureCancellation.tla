---------------------- MODULE IssueClosureCancellation ----------------------
EXTENDS Naturals

\* One Issue and one Work. IssueRelation abstracts an exact immutable identity
\* match against subject/backing_issue; event-time labels are not consulted.
\* API reads are ordered; CAS publication and native termination evidence remain abstract.
CONSTANTS InitialWorkState, QueueInitialized, IssueRelation,
         FreshIssueClosed, FreshIdentityMatches, FreshQueueLabels

VARIABLES issueClosed, closureReports, closureStage, checkedLedgerReads,
         configurationReads, nativeAuthReads,
         issuePreflightReads, confirmationLedgerReads, freshIssueClosed,
         freshIdentityMatches, freshQueueLabels, cancellationIntent,
         workState, claimOpen, nativeRunning, verifiedResult,
         cancellationRecords, commentState, commentCount

vars == <<issueClosed, closureReports, closureStage, checkedLedgerReads,
         configurationReads, nativeAuthReads,
         issuePreflightReads, confirmationLedgerReads, freshIssueClosed,
         freshIdentityMatches, freshQueueLabels, cancellationIntent,
         workState, claimOpen, nativeRunning, verifiedResult,
         cancellationRecords, commentState, commentCount>>

ExactQueueBinding ==
  QueueInitialized /\ IssueRelation \in {"subject", "backing_issue"}
FreshIssueEligible ==
  freshIssueClosed /\ freshIdentityMatches /\ freshQueueLabels

Init ==
  /\ issueClosed = FALSE /\ closureReports = 0 /\ cancellationIntent = FALSE
  /\ closureStage = "idle"
  /\ checkedLedgerReads = 0
  /\ configurationReads = 0 /\ nativeAuthReads = 0
  /\ issuePreflightReads = 0 /\ confirmationLedgerReads = 0
  /\ freshIssueClosed = FreshIssueClosed
  /\ freshIdentityMatches = FreshIdentityMatches
  /\ freshQueueLabels = FreshQueueLabels
  /\ workState = InitialWorkState
  /\ claimOpen = (InitialWorkState = "active")
  /\ nativeRunning = (InitialWorkState = "active")
  /\ verifiedResult = FALSE /\ cancellationRecords = 0
  /\ commentState = "none" /\ commentCount = 0

BeginClosure ==
  /\ closureStage = "idle" /\ closureReports < 2
  /\ issueClosed' = TRUE /\ closureReports' = closureReports + 1
  /\ closureStage' = "ledger"
  /\ UNCHANGED <<checkedLedgerReads, configurationReads,
                nativeAuthReads, issuePreflightReads, confirmationLedgerReads,
                freshIssueClosed, freshIdentityMatches, freshQueueLabels,
                cancellationIntent, workState, claimOpen, nativeRunning,
                verifiedResult, cancellationRecords, commentState, commentCount>>

ReadCheckedLedger ==
  /\ closureStage = "ledger"
  /\ checkedLedgerReads' = checkedLedgerReads + 1
  /\ IF ~QueueInitialized
       THEN closureStage' = "queue_missing"
       ELSE IF ExactQueueBinding
              THEN closureStage' = "configuration"
              ELSE closureStage' = "unbound"
  /\ UNCHANGED <<issueClosed, closureReports, configurationReads,
                nativeAuthReads, issuePreflightReads, confirmationLedgerReads,
                freshIssueClosed, freshIdentityMatches, freshQueueLabels,
                cancellationIntent, workState, claimOpen, nativeRunning,
                verifiedResult, cancellationRecords, commentState, commentCount>>

ReadConfiguration ==
  /\ closureStage = "configuration"
  /\ configurationReads' = configurationReads + 1
  /\ closureStage' = "authentication"
  /\ UNCHANGED <<issueClosed, closureReports, checkedLedgerReads,
                nativeAuthReads, issuePreflightReads,
                confirmationLedgerReads, freshIssueClosed, freshIdentityMatches,
                freshQueueLabels, cancellationIntent, workState, claimOpen,
                nativeRunning, verifiedResult, cancellationRecords,
                commentState, commentCount>>

AuthenticateNativeRun ==
  /\ closureStage = "authentication"
  /\ nativeAuthReads' = nativeAuthReads + 1
  /\ closureStage' = "preflight"
  /\ UNCHANGED <<issueClosed, closureReports, checkedLedgerReads,
                configurationReads, issuePreflightReads,
                confirmationLedgerReads, freshIssueClosed, freshIdentityMatches,
                freshQueueLabels, cancellationIntent, workState, claimOpen,
                nativeRunning, verifiedResult, cancellationRecords,
                commentState, commentCount>>

FreshIssuePreflight ==
  /\ closureStage = "preflight"
  /\ issuePreflightReads' = issuePreflightReads + 1
  /\ cancellationIntent' = cancellationIntent \/ FreshIssueEligible
  /\ IF FreshIssueEligible
       THEN IF workState \in {"cancelled", "completed"}
               THEN closureStage' = "ledger_confirmation"
               ELSE closureStage' = "publish"
       ELSE closureStage' = "stale"
  /\ UNCHANGED <<issueClosed, closureReports, checkedLedgerReads,
                configurationReads, nativeAuthReads,
                confirmationLedgerReads, freshIssueClosed, freshIdentityMatches,
                freshQueueLabels, workState, claimOpen, nativeRunning,
                verifiedResult, cancellationRecords, commentState, commentCount>>

RecordWorkCancellation ==
  /\ closureStage = "publish" /\ cancellationIntent
  /\ FreshIssueEligible /\ workState \in {"available", "active"}
  /\ workState' = "cancelled" /\ claimOpen' = FALSE
  /\ cancellationRecords' = cancellationRecords + 1
  /\ closureStage' = "ledger_confirmation"
  /\ UNCHANGED <<issueClosed, closureReports, checkedLedgerReads,
                configurationReads, nativeAuthReads,
                issuePreflightReads, confirmationLedgerReads, freshIssueClosed,
                freshIdentityMatches, freshQueueLabels, cancellationIntent,
                nativeRunning, verifiedResult, commentState, commentCount>>

IssueChangesBeforeCAS ==
  /\ closureStage = "publish" /\ FreshIssueEligible
  /\ freshIssueClosed' = FALSE /\ freshIdentityMatches' = FALSE
  /\ freshQueueLabels' = FALSE
  /\ UNCHANGED <<issueClosed, closureReports, closureStage, checkedLedgerReads,
                configurationReads, nativeAuthReads,
                issuePreflightReads, confirmationLedgerReads, cancellationIntent,
                workState, claimOpen, nativeRunning, verifiedResult,
                cancellationRecords, commentState, commentCount>>

RejectStaleAtCAS ==
  /\ closureStage = "publish" /\ ~FreshIssueEligible
  /\ closureStage' = "stale"
  /\ UNCHANGED <<issueClosed, closureReports, checkedLedgerReads,
                configurationReads, nativeAuthReads,
                issuePreflightReads, confirmationLedgerReads, freshIssueClosed,
                freshIdentityMatches, freshQueueLabels, cancellationIntent,
                workState, claimOpen, nativeRunning, verifiedResult,
                cancellationRecords, commentState, commentCount>>

PreserveTerminalOutcome ==
  /\ closureStage = "publish" /\ workState \in {"cancelled", "completed"}
  /\ closureStage' = "ledger_confirmation"
  /\ UNCHANGED <<issueClosed, closureReports, checkedLedgerReads,
                configurationReads, nativeAuthReads,
                issuePreflightReads, confirmationLedgerReads, freshIssueClosed,
                freshIdentityMatches, freshQueueLabels, cancellationIntent,
                workState, claimOpen, nativeRunning, verifiedResult,
                cancellationRecords, commentState, commentCount>>

ConfirmTerminalLedger ==
  /\ closureStage = "ledger_confirmation"
  /\ workState \in {"cancelled", "completed"}
  /\ confirmationLedgerReads' = confirmationLedgerReads + 1
  /\ closureStage' = "confirmation"
  /\ UNCHANGED <<issueClosed, closureReports, checkedLedgerReads,
                configurationReads, nativeAuthReads,
                issuePreflightReads, freshIssueClosed, freshIdentityMatches,
                freshQueueLabels, cancellationIntent, workState, claimOpen,
                nativeRunning, verifiedResult, cancellationRecords,
                commentState, commentCount>>

FinishUnbound ==
  /\ closureStage \in {"unbound", "queue_missing"}
  /\ closureStage' = "idle"
  /\ UNCHANGED <<issueClosed, closureReports, checkedLedgerReads,
                configurationReads, nativeAuthReads,
                issuePreflightReads, confirmationLedgerReads, freshIssueClosed,
                freshIdentityMatches, freshQueueLabels, cancellationIntent,
                workState, claimOpen, nativeRunning, verifiedResult,
                cancellationRecords, commentState, commentCount>>

FinishStale ==
  /\ closureStage = "stale"
  /\ closureStage' = "idle"
  /\ UNCHANGED <<issueClosed, closureReports, checkedLedgerReads,
                configurationReads, nativeAuthReads,
                issuePreflightReads, confirmationLedgerReads, freshIssueClosed,
                freshIdentityMatches, freshQueueLabels, cancellationIntent,
                workState, claimOpen, nativeRunning, verifiedResult,
                cancellationRecords, commentState, commentCount>>

CompleteWork ==
  /\ workState = "active"
  /\ workState' = "completed" /\ claimOpen' = FALSE
  /\ UNCHANGED <<issueClosed, closureReports, closureStage, checkedLedgerReads,
                configurationReads, nativeAuthReads,
                issuePreflightReads, confirmationLedgerReads, freshIssueClosed,
                freshIdentityMatches, freshQueueLabels, cancellationIntent,
                nativeRunning, verifiedResult, cancellationRecords,
                commentState, commentCount>>

VerifyResult ==
  /\ workState = "completed" /\ ~verifiedResult
  /\ verifiedResult' = TRUE
  /\ UNCHANGED <<issueClosed, closureReports, closureStage, checkedLedgerReads,
                configurationReads, nativeAuthReads,
                issuePreflightReads, confirmationLedgerReads, freshIssueClosed,
                freshIdentityMatches, freshQueueLabels, cancellationIntent,
                workState, claimOpen, nativeRunning, cancellationRecords,
                commentState, commentCount>>

ObserveNativeTermination ==
  /\ nativeRunning
  /\ nativeRunning' = FALSE
  /\ UNCHANGED <<issueClosed, closureReports, closureStage, checkedLedgerReads,
                configurationReads, nativeAuthReads,
                issuePreflightReads, confirmationLedgerReads, freshIssueClosed,
                freshIdentityMatches, freshQueueLabels, cancellationIntent,
                workState, claimOpen, verifiedResult, cancellationRecords,
                commentState, commentCount>>

PostClosureConfirmation ==
  /\ closureStage = "confirmation"
  /\ commentState = "none"
  /\ commentState' = workState /\ commentCount' = 1
  /\ closureStage' = "idle"
  /\ UNCHANGED <<issueClosed, closureReports, checkedLedgerReads,
                configurationReads, nativeAuthReads,
                issuePreflightReads, confirmationLedgerReads, freshIssueClosed,
                freshIdentityMatches, freshQueueLabels, cancellationIntent,
                workState, claimOpen, nativeRunning, verifiedResult,
                cancellationRecords>>

ReuseClosureConfirmation ==
  /\ closureStage = "confirmation" /\ commentState # "none"
  /\ closureStage' = "idle"
  /\ UNCHANGED <<issueClosed, closureReports, checkedLedgerReads,
                configurationReads, nativeAuthReads,
                issuePreflightReads, confirmationLedgerReads, freshIssueClosed,
                freshIdentityMatches, freshQueueLabels, cancellationIntent,
                workState, claimOpen, nativeRunning, verifiedResult,
                cancellationRecords, commentState, commentCount>>

Next == BeginClosure \/ ReadCheckedLedger \/ ReadConfiguration
       \/ AuthenticateNativeRun \/ FreshIssuePreflight
       \/ RecordWorkCancellation \/ IssueChangesBeforeCAS
       \/ RejectStaleAtCAS \/ PreserveTerminalOutcome
       \/ ConfirmTerminalLedger \/ FinishUnbound \/ FinishStale
       \/ CompleteWork \/ VerifyResult \/ ObserveNativeTermination
       \/ PostClosureConfirmation \/ ReuseClosureConfirmation

Spec == Init /\ [][Next]_vars

TypeOK ==
  /\ InitialWorkState \in {"available", "active", "completed", "cancelled"}
  /\ QueueInitialized \in BOOLEAN
  /\ IssueRelation \in {"subject", "backing_issue", "dependency"}
  /\ FreshIssueClosed \in BOOLEAN /\ FreshIdentityMatches \in BOOLEAN
  /\ FreshQueueLabels \in BOOLEAN
  /\ issueClosed \in BOOLEAN /\ closureReports \in 0..2
  /\ closureStage \in {"idle", "ledger", "queue_missing", "unbound",
                      "configuration", "authentication", "preflight",
                      "publish", "stale", "ledger_confirmation", "confirmation"}
  /\ checkedLedgerReads \in 0..2
  /\ configurationReads \in 0..2 /\ nativeAuthReads \in 0..2
  /\ issuePreflightReads \in 0..2 /\ confirmationLedgerReads \in 0..2
  /\ freshIssueClosed \in BOOLEAN /\ freshIdentityMatches \in BOOLEAN
  /\ freshQueueLabels \in BOOLEAN
  /\ cancellationIntent \in BOOLEAN
  /\ workState \in {"available", "active", "completed", "cancelled"}
  /\ claimOpen \in BOOLEAN /\ nativeRunning \in BOOLEAN
  /\ verifiedResult \in BOOLEAN /\ cancellationRecords \in 0..1
  /\ commentState \in {"none", "cancelled", "completed"}
  /\ commentCount \in 0..1

ReadOrdering ==
  /\ configurationReads <= checkedLedgerReads
  /\ nativeAuthReads <= configurationReads
  /\ issuePreflightReads <= nativeAuthReads
  /\ confirmationLedgerReads <= issuePreflightReads

UnboundAvoidsAmplification ==
  (IssueRelation = "dependency" \/ ~QueueInitialized) =>
    /\ configurationReads = 0
    /\ nativeAuthReads = 0
    /\ issuePreflightReads = 0
    /\ confirmationLedgerReads = 0

CancellationAuthority ==
  /\ cancellationRecords = 1 => cancellationIntent /\ workState = "cancelled"
  /\ workState = "cancelled" => ~claimOpen /\ ~verifiedResult
  /\ cancellationRecords = 1 =>
       ExactQueueBinding /\ FreshIssueEligible /\ checkedLedgerReads > 0
       /\ configurationReads > 0 /\ nativeAuthReads > 0 /\ issuePreflightReads > 0
  /\ IssueRelation = "dependency" =>
       ~cancellationIntent /\ cancellationRecords = 0

TerminalIdempotence ==
  /\ workState \in {"cancelled", "completed"} => ~claimOpen
  /\ cancellationRecords <= 1
  /\ commentCount <= 1

ClosureConfirmation ==
  /\ commentState = "none" => commentCount = 0
  /\ commentState # "none" =>
       commentCount = 1 /\ cancellationIntent /\ commentState = workState
       /\ confirmationLedgerReads > 0

=============================================================================
