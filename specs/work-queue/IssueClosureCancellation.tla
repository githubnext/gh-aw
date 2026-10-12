---------------------- MODULE IssueClosureCancellation ----------------------
EXTENDS Naturals

\* One Issue and one Work. GitHub delivery, exact relationship/label checks,
\* CAS, authenticated publication, and native termination evidence are abstracted.
CONSTANTS InitialWorkState, IssueRelation, FreshQueueLabels

VARIABLES issueClosed, closureReports, cancellationIntent, workState,
          claimOpen, nativeRunning, verifiedResult, cancellationRecords,
          commentState, commentCount

vars == <<issueClosed, closureReports, cancellationIntent, workState,
          claimOpen, nativeRunning, verifiedResult, cancellationRecords,
          commentState, commentCount>>

Init ==
  /\ issueClosed = FALSE /\ closureReports = 0 /\ cancellationIntent = FALSE
  /\ workState = InitialWorkState
  /\ claimOpen = (InitialWorkState = "active")
  /\ nativeRunning = (InitialWorkState = "active")
  /\ verifiedResult = FALSE /\ cancellationRecords = 0
  /\ commentState = "none" /\ commentCount = 0

CloseIssue ==
  /\ closureReports < 2
  /\ issueClosed' = TRUE /\ closureReports' = closureReports + 1
  /\ cancellationIntent' =
       (IssueRelation \in {"subject", "backing_issue"} /\ FreshQueueLabels)
  /\ UNCHANGED <<workState, claimOpen, nativeRunning, verifiedResult,
                 cancellationRecords, commentState, commentCount>>

RecordWorkCancellation ==
  /\ cancellationIntent /\ workState \in {"available", "active"}
  /\ workState' = "cancelled" /\ claimOpen' = FALSE
  /\ cancellationRecords' = cancellationRecords + 1
  /\ UNCHANGED <<issueClosed, closureReports, cancellationIntent,
                 nativeRunning, verifiedResult, commentState, commentCount>>

CompleteWork ==
  /\ workState = "active"
  /\ workState' = "completed" /\ claimOpen' = FALSE
  /\ UNCHANGED <<issueClosed, closureReports, cancellationIntent,
                 nativeRunning, verifiedResult, cancellationRecords,
                 commentState, commentCount>>

VerifyResult ==
  /\ workState = "completed" /\ ~verifiedResult
  /\ verifiedResult' = TRUE
  /\ UNCHANGED <<issueClosed, closureReports, cancellationIntent,
                 workState, claimOpen, nativeRunning, cancellationRecords,
                 commentState, commentCount>>

ObserveNativeTermination ==
  /\ nativeRunning
  /\ nativeRunning' = FALSE
  /\ UNCHANGED <<issueClosed, closureReports, cancellationIntent,
                 workState, claimOpen, verifiedResult, cancellationRecords,
                 commentState, commentCount>>

ConfirmClosure ==
  /\ cancellationIntent /\ workState \in {"cancelled", "completed"}
  /\ commentState = "none"
  /\ commentState' = workState /\ commentCount' = 1
  /\ UNCHANGED <<issueClosed, closureReports, cancellationIntent,
                 workState, claimOpen, nativeRunning, verifiedResult,
                 cancellationRecords>>

Next == CloseIssue \/ RecordWorkCancellation \/ CompleteWork
        \/ VerifyResult \/ ObserveNativeTermination \/ ConfirmClosure

Spec == Init /\ [][Next]_vars

TypeOK ==
  /\ InitialWorkState \in {"available", "active", "completed", "cancelled"}
  /\ IssueRelation \in {"subject", "backing_issue", "dependency"}
  /\ FreshQueueLabels \in BOOLEAN
  /\ issueClosed \in BOOLEAN /\ closureReports \in 0..2
  /\ cancellationIntent \in BOOLEAN
  /\ workState \in {"available", "active", "completed", "cancelled"}
  /\ claimOpen \in BOOLEAN /\ nativeRunning \in BOOLEAN
  /\ verifiedResult \in BOOLEAN /\ cancellationRecords \in 0..1
  /\ commentState \in {"none", "cancelled", "completed"}
  /\ commentCount \in 0..1

CancellationAuthority ==
  /\ cancellationRecords = 1 => cancellationIntent /\ workState = "cancelled"
  /\ workState = "cancelled" => ~claimOpen /\ ~verifiedResult
  /\ (IssueRelation = "dependency" \/ ~FreshQueueLabels) =>
       ~cancellationIntent /\ cancellationRecords = 0

TerminalIdempotence ==
  /\ workState \in {"cancelled", "completed"} => ~claimOpen
  /\ cancellationRecords <= 1
  /\ commentCount <= 1

ClosureConfirmation ==
  /\ commentState = "none" => commentCount = 0
  /\ commentState # "none" =>
       commentCount = 1 /\ cancellationIntent /\ commentState = workState

=============================================================================
