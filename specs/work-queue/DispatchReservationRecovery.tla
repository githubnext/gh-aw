-------------------- MODULE DispatchReservationRecovery --------------------
EXTENDS Naturals, FiniteSets, TLC

\* Three already-granted singleton profiles, as in the ESLint daily cohort.
\* Exact terminal evidence abstracts authenticated repository, native run,
\* workflow, principal and originating attempt checks. A later dispatcher
\* attempt and worker attempt one are different authority boundaries.
\* Prepared recovery can race a durable start marker. Recover is the checked
\* publication/CAS boundary, not permission derived from a stale snapshot.
CONSTANTS SweepBound, Broken
ASSUME SweepBound \in 1..3

Groups == {"miner", "refiner", "monster"}
Evidence == {"none", "active", "unavailable", "foreign", "later-attempt", "exact-terminal"}
VARIABLES originTerminal, observed, selected, preparedEvidence, phase, sends,
          released, cancelled, completed, releaseEvidence, releaseMarked,
          retried, debt

vars == <<originTerminal, observed, selected, preparedEvidence, phase, sends,
          released, cancelled, completed, releaseEvidence, releaseMarked,
          retried, debt>>

Init ==
  /\ originTerminal = FALSE /\ observed = "none"
  /\ selected = {} /\ preparedEvidence = "none"
  /\ phase = [g \in Groups |-> "reserved"]
  /\ sends = [g \in Groups |-> 0]
  /\ released = {} /\ cancelled = {} /\ completed = {}
  /\ releaseEvidence = [g \in Groups |-> "none"]
  /\ releaseMarked = {} /\ retried = {} /\ debt = 3

OriginEnds ==
  /\ ~originTerminal /\ originTerminal' = TRUE
  /\ UNCHANGED <<observed, selected, preparedEvidence, phase, sends,
                 released, cancelled, completed, releaseEvidence,
                 releaseMarked, retried, debt>>

Observe(e) ==
  /\ e \in Evidence \ {"none"}
  /\ (e # "exact-terminal" \/ originTerminal)
  /\ observed' = e
  /\ UNCHANGED <<originTerminal, selected, preparedEvidence, phase, sends,
                 released, cancelled, completed, releaseEvidence,
                 releaseMarked, retried, debt>>

Prepare(batch) ==
  /\ batch \subseteq {g \in Groups : phase[g] = "reserved" /\ g \notin released}
  /\ Cardinality(batch) \in 1..SweepBound
  /\ observed = "exact-terminal" \/ (Broken = "evidence" /\ observed # "none")
  /\ selected' = batch /\ preparedEvidence' = observed
  /\ UNCHANGED <<originTerminal, observed, phase, sends, released,
                 cancelled, completed, releaseEvidence, releaseMarked,
                 retried, debt>>

Mark(g) ==
  /\ phase[g] = "reserved" /\ g \notin released
  /\ phase' = [phase EXCEPT ![g] = "marked"]
  /\ UNCHANGED <<originTerminal, observed, selected, preparedEvidence, sends,
                 released, cancelled, completed, releaseEvidence,
                 releaseMarked, retried, debt>>

Send(g) ==
  /\ g \notin released
  /\ (phase[g] = "marked"
      \/ (Broken = "resend" /\ phase[g] = "uncertain" /\ sends[g] = 1))
  /\ sends' = [sends EXCEPT ![g] = @ + 1]
  /\ phase' = [phase EXCEPT ![g] = "uncertain"]
  /\ UNCHANGED <<originTerminal, observed, selected, preparedEvidence,
                 released, cancelled, completed, releaseEvidence,
                 releaseMarked, retried, debt>>

Bind(g) ==
  /\ phase[g] = "uncertain" /\ g \notin released
  /\ phase' = [phase EXCEPT ![g] = "bound"]
  /\ UNCHANGED <<originTerminal, observed, selected, preparedEvidence, sends,
                 released, cancelled, completed, releaseEvidence,
                 releaseMarked, retried, debt>>

Complete(g) ==
  /\ phase[g] = "bound" /\ g \notin released
  /\ completed' = completed \cup {g}
  /\ UNCHANGED <<originTerminal, observed, selected, preparedEvidence,
                 phase, sends, released, cancelled, releaseEvidence,
                 releaseMarked, retried, debt>>

Recover(g) ==
  /\ g \in selected /\ g \notin released
  /\ preparedEvidence = "exact-terminal" \/ Broken = "evidence"
  /\ phase[g] = "reserved" \/ Broken = "stale"
  /\ released' = released \cup {g}
  /\ cancelled' = IF g \notin completed THEN cancelled \cup {g} ELSE cancelled
  /\ releaseEvidence' = [releaseEvidence EXCEPT ![g] = preparedEvidence]
  /\ releaseMarked' = IF phase[g] # "reserved"
                      THEN releaseMarked \cup {g} ELSE releaseMarked
  /\ debt' = IF Broken = "refund" THEN debt - 1 ELSE debt
  /\ UNCHANGED <<originTerminal, observed, selected, preparedEvidence,
                 phase, sends, completed, retried>>

\* Retry is a new ordinary fair grant after installed backoff/attempt checks.
\* It does not reopen the original Claim or resend its original dispatch.
\* Scheduling order, retry timing, and retry launch are outside this model.
Retry(g) ==
  /\ g \in released /\ g \notin retried
  /\ retried' = retried \cup {g} /\ debt' = debt + 1
  /\ UNCHANGED <<originTerminal, observed, selected, preparedEvidence,
                 phase, sends, released, cancelled, completed,
                 releaseEvidence, releaseMarked>>

Next ==
  \/ OriginEnds
  \/ \E e \in Evidence \ {"none"} : Observe(e)
  \/ \E batch \in SUBSET Groups : Prepare(batch)
  \/ \E g \in Groups : Mark(g) \/ Send(g) \/ Bind(g) \/ Complete(g)
                      \/ Recover(g) \/ Retry(g)
Spec == Init /\ [][Next]_vars

TypeOK ==
  /\ originTerminal \in BOOLEAN /\ observed \in Evidence
  /\ selected \subseteq Groups /\ preparedEvidence \in Evidence
  /\ phase \in [Groups -> {"reserved", "marked", "uncertain", "bound"}]
  /\ sends \in [Groups -> 0..2]
  /\ released \subseteq Groups /\ cancelled \subseteq Groups
  /\ completed \subseteq Groups /\ releaseMarked \subseteq Groups
  /\ releaseEvidence \in [Groups -> Evidence]
  /\ retried \subseteq released /\ debt \in 0..6
PrelaunchEvidence == \A g \in released :
                       originTerminal /\ releaseEvidence[g] = "exact-terminal"
StartFence == releaseMarked = {}
              /\ (\A g \in released : phase[g] = "reserved" /\ sends[g] = 0)
ClaimPreservation == cancelled \cap completed = {}
                     /\ cancelled = released \ completed
OneSender == \A g \in Groups : sends[g] <= 1
NoRefund == debt = 3 + Cardinality(retried)
BoundedSweep == Cardinality(selected) <= SweepBound
Safety == TypeOK /\ PrelaunchEvidence /\ StartFence /\ ClaimPreservation
          /\ OneSender /\ NoRefund /\ BoundedSweep

NoSiblingRecovery ==
  ~(phase["miner"] = "bound" /\ sends["miner"] = 1
    /\ released = {"refiner", "monster"} /\ "miner" \notin cancelled)
NoRetryRecovery == retried = {}
NoPrelaunchRace ==
  ~(\E g \in selected : preparedEvidence = "exact-terminal"
                       /\ phase[g] = "marked" /\ g \notin released)
=============================================================================
