------------------------- MODULE SafeOutputTransport -------------------------
EXTENDS Naturals, Sequences, FiniteSets

CONSTANTS Fault, Transport
ASSUME Transport \in {"patch", "bundle"}
ASSUME Fault \in {"none", "blocking-open", "reopen-original", "rebaseline",
                  "skip-publication-check", "retry-integrity",
                  "cleanup-blocked", "cleanup-error-hidden", "unsafe-diagnostics",
                  "redundant-hashing", "unqualified-producer", "ambiguous-selection"}

\* Byte classes abstract collision-free digests, not Git's or the OS's parser.
VARIABLE s
vars == <<s>>
Stages == {"resolve", "capture", "validate", "apply", "conflict", "publish", "complete",
           "failed", "done"}
Resources == {"am-state", "bundle-ref"}

Init ==
    s = [stage |-> "resolve", original |-> 0, identity |-> 0,
         producerRepo |-> IF Fault = "unqualified-producer" THEN "bare" ELSE "owner/repo",
         consumerRepo |-> "owner/repo", namespaces |-> {}, resolved |-> FALSE,
         regular |-> TRUE, captured |-> 0, capturedIdentity |-> 0,
         trusted |-> 0, snapshot |-> 0, snapshotExists |-> FALSE,
         transformed |-> FALSE, mutated |-> FALSE, localChecked |-> FALSE,
         failed |-> FALSE, diagnosticsSafe |-> TRUE, blocked |-> FALSE,
         permanentRetries |-> 0, localHashes |-> 0, attempts |-> 0,
         resources |-> {}, cleanupError |-> FALSE, cleanupReported |-> FALSE,
         applied |-> {}, writes |-> {}, lastEvent |-> "Init"]

Integrity ==
    /\ s.regular /\ s.original = s.captured
    /\ s.identity = s.capturedIdentity
    /\ (~s.snapshotExists \/ s.snapshot = s.trusted)

Reject ==
    s' = [s EXCEPT !.stage = "failed", !.failed = TRUE,
                  !.diagnosticsSafe = Fault # "unsafe-diagnostics",
                  !.blocked = ~s.regular /\ Fault = "blocking-open",
                  !.permanentRetries = IF Fault = "retry-integrity" THEN 1 ELSE 0,
                  !.lastEvent = "IntegrityFailure"]

Capture ==
    /\ s.stage = "capture"
    /\ IF ~s.regular THEN Reject
       ELSE s' = [s EXCEPT !.stage = "validate", !.captured = s.original,
                          !.capturedIdentity = s.identity, !.trusted = s.original,
                          !.lastEvent = "Capture"]

Resolve(namespaces) ==
    /\ s.stage = "resolve"
    /\ namespaces \in {{"qualified"}, {"legacy"}, {"qualified", "legacy"}}
    /\ IF Cardinality(namespaces) > 1 /\ Fault # "ambiguous-selection" THEN Reject
       ELSE s' = [s EXCEPT !.stage = "capture", !.namespaces = namespaces,
                          !.resolved = TRUE, !.lastEvent = "Resolve"]

Mutate(kind) ==
    /\ s.stage \in {"capture", "validate", "apply", "conflict", "publish", "complete"}
    /\ ~s.mutated
    /\ kind \in {"bytes", "identity", "fifo", "snapshot"}
    /\ (s.stage # "capture" \/ kind = "fifo")
    /\ (kind # "snapshot" \/ s.snapshotExists)
    /\ s' = [s EXCEPT
        !.original = IF kind = "bytes" THEN 1 ELSE @,
        !.identity = IF kind = "identity" THEN 1 ELSE @,
        !.regular = IF kind = "fifo" THEN FALSE ELSE @,
        !.snapshot = IF kind = "snapshot" THEN 1 ELSE @,
        !.captured = IF kind = "bytes" /\ Fault = "rebaseline" THEN 1 ELSE @,
        !.mutated = TRUE, !.lastEvent = "Mutate:" \o kind]

Transform ==
    /\ s.stage = "validate" /\ ~s.transformed
    /\ IF ~Integrity THEN Reject
       ELSE s' = [s EXCEPT !.trusted = 2, !.transformed = TRUE,
                          !.lastEvent = "TrustedTransform"]

Validate ==
    /\ s.stage = "validate"
    /\ IF ~Integrity THEN Reject
       ELSE s' = [s EXCEPT !.stage = "apply", !.snapshot = s.trusted,
                          !.snapshotExists = TRUE, !.lastEvent = "ValidateSnapshot"]

LocalCommand ==
    /\ s.stage \in {"apply", "publish"} /\ ~s.localChecked
    /\ s' = [s EXCEPT !.localChecked = TRUE,
                      !.localHashes = IF Fault = "redundant-hashing" THEN 1 ELSE 0,
                      !.lastEvent = "UnrelatedLocalCommand"]

Apply(success) ==
    /\ s.stage = "apply"
    /\ IF ~Integrity THEN Reject
       ELSE s' = [s EXCEPT
           !.stage = IF success THEN "publish" ELSE "conflict",
           !.applied = @ \cup {
               [bytes |-> IF Fault = "reopen-original" THEN s.original ELSE s.snapshot,
                validated |-> s.trusted]},
           !.resources = IF Transport = "bundle" THEN {"bundle-ref"}
                         ELSE IF success THEN {} ELSE {"am-state"},
           !.lastEvent = IF success THEN "Apply" ELSE "ApplyConflict"]

CheckConflict ==
    /\ s.stage = "conflict"
    /\ IF ~Integrity THEN Reject
       ELSE s' = [s EXCEPT !.stage = "failed", !.lastEvent = "CheckConflict"]

\* A rate-limited attempt has no effect; a subsequent attempt checks again.
Publish(success) ==
    /\ s.stage = "publish" /\ s.attempts < 2
    /\ IF ~Integrity /\ Fault # "skip-publication-check" THEN Reject
       ELSE s' = [s EXCEPT
           !.stage = IF success \/ s.attempts = 1 THEN "complete" ELSE @,
           !.attempts = @ + 1,
           !.writes = IF success
                      THEN @ \cup {[verified |-> Integrity, failed |-> s.failed]}
                      ELSE @,
           !.lastEvent = IF success THEN "Publish" ELSE "RateLimited"]

Complete ==
    /\ s.stage = "complete"
    /\ IF ~Integrity THEN Reject
       ELSE s' = [s EXCEPT !.stage = "failed", !.lastEvent = "HandlerComplete"]

\* "failed" is the common finally path, also used after successful completion.
Cleanup(success) ==
    /\ s.stage = "failed"
    /\ LET denied == s.failed /\ Fault = "cleanup-blocked"
           error == s.resources # {} /\ (~success \/ denied)
       IN s' = [s EXCEPT
           !.stage = "done", !.snapshotExists = FALSE,
           !.resources = IF error THEN @ ELSE {},
           !.cleanupError = error,
           !.cleanupReported = error /\ ~denied /\ Fault # "cleanup-error-hidden",
           !.lastEvent = IF error THEN "CleanupFailure" ELSE "Cleanup"]

Next ==
    \/ \E namespaces \in {{"qualified"}, {"legacy"}, {"qualified", "legacy"}}:
           Resolve(namespaces)
    \/ Capture \/ Transform \/ Validate \/ LocalCommand \/ CheckConflict \/ Complete
    \/ \E kind \in {"bytes", "identity", "fifo", "snapshot"}: Mutate(kind)
    \/ \E success \in BOOLEAN: Apply(success) \/ Publish(success) \/ Cleanup(success)

Spec == Init /\ [][Next]_vars

TypeOK ==
    /\ s.stage \in Stages
    /\ s.producerRepo \in {"bare", "owner/repo"} /\ s.consumerRepo = "owner/repo"
    /\ s.namespaces \subseteq {"qualified", "legacy"} /\ s.resolved \in BOOLEAN
    /\ s.original \in 0..1 /\ s.captured \in 0..1
    /\ s.identity \in 0..1 /\ s.capturedIdentity \in 0..1
    /\ s.trusted \in 0..2 /\ s.snapshot \in 0..2
    /\ s.regular \in BOOLEAN /\ s.snapshotExists \in BOOLEAN
    /\ s.transformed \in BOOLEAN /\ s.mutated \in BOOLEAN
    /\ s.localChecked \in BOOLEAN /\ s.failed \in BOOLEAN
    /\ s.diagnosticsSafe \in BOOLEAN /\ s.blocked \in BOOLEAN
    /\ s.permanentRetries \in 0..1 /\ s.localHashes \in 0..1
    /\ s.attempts \in 0..2 /\ s.resources \subseteq Resources
    /\ s.cleanupError \in BOOLEAN /\ s.cleanupReported \in BOOLEAN
    /\ s.applied \subseteq [bytes: 0..2, validated: 0..2]
    /\ s.writes \subseteq [verified: BOOLEAN, failed: BOOLEAN]

StableConsumption == \A a \in s.applied: a.bytes = a.validated
QualifiedRepository == s.resolved => s.producerRepo = s.consumerRepo
UnambiguousResolution == s.resolved => Cardinality(s.namespaces) = 1
OriginalBaseline == s.stage \notin {"resolve", "capture"} => s.captured = 0
PublicationIntegrity == \A w \in s.writes: w.verified /\ ~w.failed
PermanentFailure == s.permanentRetries = 0
NonblockingOpen == ~s.blocked
SafeDiagnostics == s.diagnosticsSafe
BoundedLocalCost == s.localHashes = 0
CleanupConfinement ==
    s.resources \subseteq IF Transport = "patch" THEN {"am-state"} ELSE {"bundle-ref"}
CleanupDisposition ==
    s.stage = "done" =>
        /\ ~s.snapshotExists
        /\ (s.resources = {} \/ s.cleanupReported)
        /\ (s.cleanupError => s.cleanupReported)

\* Deliberately false reachability predicates, not safety requirements.
NoSuccessfulWrite == s.writes = {}
NoRejectedMutation == ~s.failed
NoRateLimitRetry == s.attempts < 2
NoCleanupFailure == ~s.cleanupError
NoConflictCleanup == ~(s.stage = "done" /\ s.applied # {} /\ s.attempts = 0)
=============================================================================
