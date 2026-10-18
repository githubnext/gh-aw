----------------------- MODULE ClaimDeliveryRecovery -----------------------
EXTENDS Naturals, FiniteSets, TLC

\* Post-Completion refinement: two immutable Claims already ran their effects.
\* One retains its Work. Private creation receipts may be lost, never rebuilt
\* from artifacts. Native readback and exact identity are abstract premises,
\* not a proof of GitHub, cryptography, encoding, or the implementation.
CONSTANTS Deadline, Broken
ASSUME Deadline \in Nat
Claims == {"ordinary", "retained"}
Transports == {"bare", "enriched", "older-base", "foreign-head", "foreign-base",
               "invalid-sha", "unrelated-base", "unknown-field"}
Observations == {"unavailable", "active", "foreign", "rerun", "exact_terminal"}
BoundPrincipal == "credential-principal"
ProfilePrincipal == "absent"
VARIABLE state
vars == <<state>>

Pending(c) == c \notin state.results \cup state.failures
Original(c) == state.owner[c] = "original"
MetadataValid == state.transport \in {"bare", "enriched", "older-base"}
AcceptsMetadata ==
  (MetadataValid /\ ~(Broken = "reject-enriched" /\ state.transport = "enriched"))
  \/ Broken = "accept-metadata"
HasPrivateProof(c) == c \in state.receipts
MayUseProof(c) == HasPrivateProof(c) \/ Broken = "artifact-proof"
Candidates == {c \in Claims : Pending(c) /\ Original(c)
                  /\ (~state.released \/ Broken # "skip-released")}
BudgetExhausted == state.now = Deadline \/ Broken = "early-expiry"
TerminalProof == state.observation = "exact_terminal" \/ Broken = "foreign-evidence"

Init ==
  \E transport \in Transports, receipts \in SUBSET Claims :
    state = [now |-> 0, cursor |-> "ordinary", transport |-> transport,
             receipts |-> receipts, checked |-> {}, accepted |-> {},
             results |-> {}, resultProofs |-> {}, failures |-> {},
             proposals |-> [c \in Claims |-> "none"],
             failureEvidence |-> [c \in Claims |-> "none"],
             failureAt |-> [c \in Claims |-> 0],
             disposition |-> [c \in Claims |-> "pending"],
             owner |-> [c \in Claims |-> "original"],
             observation |-> "unavailable", nativeTerminal |-> FALSE,
             quiescent |-> FALSE, released |-> FALSE,
             releasePrincipal |-> "none", stalePublication |-> FALSE,
             replayed |-> FALSE]

Tick ==
  /\ state.now < Deadline
  /\ state' = [state EXCEPT !.now = @ + 1]

CheckReceipt(c) ==
  /\ Pending(c) /\ Original(c) /\ c \notin state.checked /\ MayUseProof(c)
  /\ state' = [state EXCEPT !.checked = @ \cup {c},
                 !.accepted = IF AcceptsMetadata THEN @ \cup {c} ELSE @]

PublishResult(c) ==
  /\ c \in state.accepted /\ c \notin state.results /\ MayUseProof(c)
  /\ (Pending(c) /\ Original(c)) \/ Broken = "stale-cas"
  /\ state' = [state EXCEPT !.results = @ \cup {c},
       !.resultProofs = IF MetadataValid /\ HasPrivateProof(c)
                       THEN @ \cup {c} ELSE @,
       !.stalePublication = @ \/ ~Pending(c) \/ ~Original(c)]

LoseReceipts ==
  /\ state.receipts # {}
  /\ state' = [state EXCEPT !.receipts = {}]

Quiesce ==
  /\ ~state.quiescent
  /\ state' = [state EXCEPT !.quiescent = TRUE]

Terminate ==
  /\ ~state.nativeTerminal
  /\ state' = [state EXCEPT !.nativeTerminal = TRUE, !.quiescent = TRUE]

Observe(evidence) ==
  /\ evidence \in Observations /\ evidence # state.observation
  /\ evidence = "exact_terminal" => state.nativeTerminal
  /\ evidence = "active" => ~state.nativeTerminal
  /\ state' = [state EXCEPT !.observation = evidence]

InlineRelease ==
  /\ ~state.released /\ state.quiescent
  /\ state' = [state EXCEPT !.released = TRUE,
       !.releasePrincipal = IF Broken = "profile-principal"
                            THEN ProfilePrincipal ELSE BoundPrincipal]

TerminalRelease ==
  /\ ~state.released /\ state.observation = "exact_terminal"
  /\ state' = [state EXCEPT !.released = TRUE,
                             !.releasePrincipal = BoundPrincipal]

\* A one-member rotating batch separates evidence preparation from checked
\* publication, allowing a Result/regrant to win before an old proposal.
Maintenance ==
  /\ state' = [state EXCEPT
       !.cursor = IF @ = "ordinary" THEN "retained" ELSE "ordinary",
       !.proposals[state.cursor] =
         IF state.cursor \in Candidates /\ BudgetExhausted /\ TerminalProof
         THEN state.observation ELSE @]

PublishFailure(c) ==
  /\ state.proposals[c] # "none" /\ c \notin state.failures
  /\ BudgetExhausted
  /\ (Pending(c) /\ Original(c)) \/ Broken = "stale-cas"
  /\ state' = [state EXCEPT !.failures = @ \cup {c},
       !.failureEvidence[c] = state.proposals[c], !.failureAt[c] = state.now,
       !.disposition[c] = IF Broken = "unknown-as-none" THEN "none" ELSE "unknown",
       !.stalePublication = @ \/ ~Pending(c) \/ ~Original(c)]

RegrantRetained ==
  /\ state.owner["retained"] = "original"
  /\ "retained" \in state.results
     \/ (Broken = "failed-regrant" /\ "retained" \in state.failures)
  /\ state' = [state EXCEPT !.owner["retained"] = "new-claim"]

ReplayEffects ==
  /\ Broken = "replay-effects" /\ state.released /\ ~state.replayed
  /\ \E c \in Claims : Pending(c)
  /\ state' = [state EXCEPT !.replayed = TRUE]

Next ==
  Tick \/ LoseReceipts \/ Quiesce \/ Terminate \/ InlineRelease \/ TerminalRelease
  \/ Maintenance \/ RegrantRetained \/ ReplayEffects
  \/ (\E evidence \in Observations : Observe(evidence))
  \/ (\E c \in Claims : CheckReceipt(c) \/ PublishResult(c) \/ PublishFailure(c))
Spec == Init /\ [][Next]_vars
FairSpec == Spec /\ WF_vars(Tick) /\ WF_vars(Maintenance)
                 /\ \A c \in Claims : WF_vars(PublishFailure(c))

TypeOK ==
  /\ state.now \in 0..Deadline /\ state.cursor \in Claims
  /\ state.transport \in Transports /\ state.receipts \subseteq Claims
  /\ state.checked \subseteq Claims /\ state.accepted \subseteq Claims
  /\ state.results \subseteq Claims /\ state.resultProofs \subseteq Claims
  /\ state.failures \subseteq Claims
  /\ state.proposals \in [Claims -> Observations \cup {"none"}]
  /\ state.failureEvidence \in [Claims -> Observations \cup {"none"}]
  /\ state.failureAt \in [Claims -> 0..Deadline]
  /\ state.disposition \in [Claims -> {"pending", "unknown", "none"}]
  /\ state.owner \in [Claims -> {"original", "new-claim"}]
  /\ state.observation \in Observations
  /\ state.nativeTerminal \in BOOLEAN /\ state.quiescent \in BOOLEAN
  /\ state.released \in BOOLEAN
  /\ state.releasePrincipal \in {"none", ProfilePrincipal, BoundPrincipal}
  /\ state.stalePublication \in BOOLEAN /\ state.replayed \in BOOLEAN
NativeReadbackAgreement ==
  \A c \in state.checked : (c \in state.accepted) <=> MetadataValid
ResultAuthority == state.results \subseteq state.resultProofs
ExclusiveBarrier == state.results \cap state.failures = {}
PublicationFence == ~state.stalePublication
FailureBudget == \A c \in state.failures : state.failureAt[c] = Deadline
FailureAuthority ==
  \A c \in state.failures :
    state.failureEvidence[c] = "exact_terminal" /\ state.nativeTerminal
UnknownEffects ==
  \A c \in state.failures : state.disposition[c] = "unknown"
CleanupAuthority ==
  state.released => state.quiescent /\ state.releasePrincipal = BoundPrincipal
RetainedFence ==
  state.owner["retained"] = "new-claim" =>
    "retained" \in state.results /\ "retained" \notin state.failures
NoReplay == ~state.replayed
Safety == TypeOK /\ NativeReadbackAgreement /\ ResultAuthority
          /\ ExclusiveBarrier /\ PublicationFence /\ FailureBudget
          /\ FailureAuthority /\ UnknownEffects /\ CleanupAuthority
          /\ RetainedFence /\ NoReplay

\* Stable exact terminal reads are an explicit environment assumption.
\* Fair clock, rotating maintenance and publication do not imply API availability.
EventualSettlement ==
  (<>[](state.observation = "exact_terminal")) =>
    (\A c \in Claims : Pending(c) ~> ~Pending(c))
NoReleasedRecovery ==
  ~(state.released /\ "ordinary" \in state.failures
     /\ "ordinary" \notin state.results /\ state.receipts = {})
NoEnrichedResult ==
  ~(state.transport = "enriched" /\ "ordinary" \in state.results)
NoRetainedRecovery ==
  ~(state.released /\ "ordinary" \in state.failures
     /\ "retained" \in state.results /\ state.owner["retained"] = "new-claim")
NoUnavailablePending ==
  ~(state.released /\ state.now = Deadline /\ state.observation = "unavailable"
     /\ Pending("ordinary") /\ Pending("retained"))
=============================================================================
