------------------------ MODULE ClaimDeliveryHealth ------------------------
EXTENDS Naturals, FiniteSets, TLC

\* Focused safe-output-manager boundary: all handler passes succeed, but only
\* trusted finalization can establish Result/cancellation. Pending and failed
\* delivery must still fail the aggregate pass after every sibling is processed.
\* Receipt truth and exact native terminal evidence are independent premises.
CONSTANTS Staged, Broken
ASSUME Staged \in BOOLEAN
Claims == {1, 2}
DeliveryStates == {"unprocessed", "pending", "failed", "result", "cancelled", "preview"}
VARIABLES delivery, results, verified, cancelled, report, outcome, terminal, released
vars == <<delivery, results, verified, cancelled, report, outcome, terminal, released>>
Processed == {c \in Claims : delivery[c] # "unprocessed"}
Unsettled == {c \in Claims : delivery[c] \in {"pending", "failed"}}
Settled == {c \in Claims : delivery[c] \in {"result", "cancelled"}}
ExpectedOutcome ==
  IF Staged THEN "staged_preview"
  ELSE IF Settled # Claims THEN "delivery_pending"
  ELSE IF cancelled = Claims THEN "cancelled"
  ELSE IF cancelled # {} THEN "completed_with_cancellations"
  ELSE "completed"

Init ==
  /\ delivery = [c \in Claims |-> "unprocessed"]
  /\ results = {} /\ verified = {} /\ cancelled = {}
  /\ report = "none" /\ outcome = "none"
  /\ terminal = FALSE /\ released = FALSE

Process(c, state) ==
  /\ report = "none" /\ c \in Claims \ Processed
  /\ state \in IF Staged THEN {"preview"}
              ELSE {"pending", "failed", "result", "cancelled"}
  /\ delivery' = [delivery EXCEPT ![c] = state]
  /\ results' = IF state = "result" THEN results \cup {c} ELSE results
  /\ verified' = IF state = "result" THEN verified \cup {c} ELSE verified
  /\ cancelled' = IF state = "cancelled" THEN cancelled \cup {c} ELSE cancelled
  /\ UNCHANGED <<report, outcome, terminal, released>>

Report ==
  /\ report = "none"
  /\ Processed = Claims
     \/ (Broken = "early-stop" /\ Unsettled # {})
  /\ report' = IF Staged THEN "preview"
               ELSE IF Unsettled # {} /\ Broken # "false-green"
                    THEN "failure" ELSE "success"
  /\ outcome' = ExpectedOutcome
  /\ UNCHANGED <<delivery, results, verified, cancelled, terminal, released>>

TerminalEvidence ==
  /\ report # "none" /\ ~terminal /\ terminal' = TRUE
  /\ UNCHANGED <<delivery, results, verified, cancelled, report, outcome, released>>

Release ==
  /\ ~Staged /\ ~released /\ (terminal \/ Broken = "premature-release")
  /\ released' = TRUE
  /\ results' = IF Broken = "release-result"
                THEN results \cup Unsettled ELSE results
  /\ UNCHANGED <<delivery, verified, cancelled, report, outcome, terminal>>

Next ==
  \/ \E c \in Claims, state \in DeliveryStates : Process(c, state)
  \/ Report \/ TerminalEvidence \/ Release
Spec == Init /\ [][Next]_vars

TypeOK ==
  /\ delivery \in [Claims -> DeliveryStates]
  /\ results \subseteq Claims /\ verified \subseteq Claims /\ cancelled \subseteq Claims
  /\ report \in {"none", "success", "failure", "preview"}
  /\ outcome \in {"none", "staged_preview", "delivery_pending", "cancelled",
                  "completed_with_cancellations", "completed"}
  /\ terminal \in BOOLEAN /\ released \in BOOLEAN
IndependentSiblings == report # "none" => Processed = Claims
ReportTruth ==
  report # "none" =>
    /\ outcome = ExpectedOutcome
    /\ IF Staged THEN report = "preview"
       ELSE (report = "failure" <=> Unsettled # {})
            /\ (report = "success" <=> Unsettled = {})
ResultAuthority ==
  /\ results = verified
  /\ results = {c \in Claims : delivery[c] = "result"}
  /\ cancelled = {c \in Claims : delivery[c] = "cancelled"}
  /\ results \cap cancelled = {}
StagedIsolation == Staged => results = {} /\ cancelled = {} /\ ~released
ReleaseAuthority == released => terminal /\ ~Staged
Safety == TypeOK /\ IndependentSiblings /\ ReportTruth /\ ResultAuthority
          /\ StagedIsolation /\ ReleaseAuthority

NoPendingSibling ==
  ~(report = "failure" /\ delivery[1] = "pending" /\ 2 \in results)
NoFailedSibling ==
  ~(report = "failure" /\ delivery[1] = "failed" /\ 2 \in results)
NoMixedSuccess ==
  ~(report = "success" /\ outcome = "completed_with_cancellations"
    /\ cancelled = {1} /\ results = {2})
NoReleasedPending ==
  ~(released /\ delivery[1] = "pending" /\ results = {2})
=============================================================================
