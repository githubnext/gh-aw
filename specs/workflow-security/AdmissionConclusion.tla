------------------------- MODULE AdmissionConclusion -------------------------
EXTENDS Naturals, TLC

CONSTANTS Fault, OutagePolicy
VARIABLE s

MaxAttempts == 3
NoPayload == [invocation |-> "none", content |-> "none", bytes |-> 0, artifacts |-> {}]
CompactSession(artifacts) ==
    [invocation |-> "current", content |-> "compact-redacted", bytes |-> 1, artifacts |-> artifacts]
Payloads == [invocation : {"none", "current"},
             content : {"none", "compact-redacted", "agent-content", "changed"},
             bytes : 0..65, artifacts : SUBSET {"initial", "retry", "foreign"}]
Contexts ==
    [endpoint : {"absent", "valid", "invalid"},
     session : {"valid", "invalid"},
     identity : {"valid", "invalid"},
     admission : {"admitted", "ignored"},
     outcome : {"success", "failure", "cancelled"}]
Authorized == s.context.identity = "valid" \/ Fault = "identity-bypass"
FailedOutcome ==
    IF Fault = "delivery-fails-workflow" /\ OutagePolicy = "block"
    THEN "failure" ELSE s.context.outcome

Init ==
    \E context \in Contexts :
        s = [context |-> context, phase |-> "generate", eligible |-> FALSE,
             generated |-> FALSE, uploadAttempts |-> 0, uploadResult |-> "pending",
             uploadedArtifacts |-> {}, artifactContent |-> "full-redacted",
             snapshot |-> NoPayload, sentPayload |-> NoPayload, stored |-> NoPayload,
             attempts |-> 0, credential |-> "none", traceId |-> "none",
             processing |-> 0, financialEffects |-> 0,
             firstReplica |-> 0, recovered |-> FALSE, conflictSeen |-> FALSE,
             timedOut |-> FALSE, warning |-> FALSE, result |-> "none",
             outcome |-> context.outcome, lastEvent |-> "Init"]

SkipConclusion ==
    /\ s.phase = "generate"
    /\ s' = [s EXCEPT !.phase = "done", !.result = "skipped", !.lastEvent = "SkipConclusionJob"]

GenerateSession ==
    /\ s.phase = "generate"
    /\ s' = [s EXCEPT !.phase = "upload", !.eligible = TRUE,
             !.generated = s.context.session = "valid", !.lastEvent = "GenerateUnifiedSession"]

UploadArtifact ==
    /\ s.phase \in {"upload", "upload-retry"}
    /\ \E success \in BOOLEAN, hasID \in BOOLEAN :
        /\ success => hasID
        /\ s' = [s EXCEPT
            !.uploadAttempts = @ + 1,
            !.uploadResult = IF success THEN "success"
                             ELSE IF s.phase = "upload-retry" THEN "failure" ELSE "pending",
            !.uploadedArtifacts = IF hasID
                                  THEN @ \cup {IF s.phase = "upload" THEN "initial" ELSE "retry"} ELSE @,
            !.artifactContent = IF Fault = "compact-replaces-artifact" THEN "compact-redacted" ELSE @,
            !.phase = IF success \/ s.phase = "upload-retry" THEN "prepare" ELSE "upload-retry",
            !.lastEvent = IF s.phase = "upload" THEN "UploadUsageArtifact" ELSE "RetryUsageArtifact"]

PrepareConclusion ==
    /\ (s.phase = "prepare" \/ (Fault = "callback-before-upload" /\ s.phase = "upload"))
    /\ LET enabled == s.context.endpoint = "valid"
                      \/ (Fault = "endpoint-bypass" /\ s.context.endpoint = "absent")
           valid == s.generated \/ Fault = "invalid-session"
       IN s' = [s EXCEPT
            !.phase = IF enabled /\ valid THEN "request" ELSE "done",
            !.snapshot = IF enabled /\ valid
                         THEN [CompactSession(s.uploadedArtifacts) EXCEPT
                               !.content = IF Fault = "unfiltered-session" THEN "agent-content" ELSE @,
                               !.bytes = IF Fault = "oversized-payload" THEN 65 ELSE @,
                               !.artifacts = IF Fault = "artifact-id-spoof" THEN {"foreign"}
                                             ELSE IF Fault = "artifact-ids-omitted" THEN {} ELSE @]
                         ELSE @,
            !.result = IF enabled /\ valid THEN @
                       ELSE IF s.context.endpoint = "absent" THEN "skipped" ELSE "warning",
            !.warning = s.context.endpoint # "absent" /\ ~(enabled /\ valid),
            !.lastEvent = "PrepareConclusion"]

SendConclusion ==
    /\ s.phase = "request"
    /\ s' = [s EXCEPT
        !.phase = "response", !.attempts = @ + 1,
        !.credential = IF Fault = "token-handoff" THEN "admission-oidc" ELSE "conclusion-oidc",
        !.sentPayload = IF Fault = "retry-payload" /\ s.attempts > 0
                        THEN [s.snapshot EXCEPT !.content = "changed"] ELSE s.snapshot,
        !.traceId = IF Fault = "retry-trace" /\ s.attempts > 0 THEN "foreign-trace" ELSE "caller-trace",
        !.lastEvent = "POSTConclusion"]

AcceptConclusion ==
    /\ s.phase = "response" /\ Authorized
    /\ \E delivered \in BOOLEAN, replica \in {1, 2} :
        LET commit == Fault # "ack-before-store"
            first == s.stored = NoPayload
        IN s' = [s EXCEPT
            !.stored = IF commit /\ first THEN s.sentPayload ELSE @,
            !.processing = IF commit /\ (first \/ Fault = "duplicate-processing") THEN @ + 1 ELSE @,
            !.financialEffects = IF Fault = "conclusion-charge" THEN @ + 1 ELSE @,
            !.firstReplica = IF first THEN replica ELSE @,
            !.recovered = ~first /\ delivered /\ replica # s.firstReplica,
            !.phase = IF delivered THEN "done" ELSE "evaluate",
            !.result = IF delivered THEN "delivered" ELSE @,
            !.lastEvent = IF delivered THEN "Accept202Or204" ELSE "LoseConclusionAcknowledgement"]

ConflictingReplay ==
    /\ s.phase = "response" /\ s.stored # NoPayload
    /\ s' = [s EXCEPT
        !.stored = IF Fault = "conflict-overwrite" THEN [@ EXCEPT !.content = "changed"] ELSE @,
        !.phase = "done", !.conflictSeen = TRUE, !.warning = TRUE,
        !.result = "warning", !.outcome = FailedOutcome, !.lastEvent = "RejectConflictingReplay"]

RejectConclusion ==
    /\ s.phase = "response"
    /\ s' = [s EXCEPT !.phase = "done", !.warning = TRUE,
             !.result = "warning", !.outcome = FailedOutcome, !.lastEvent = "RejectConclusion"]

TransientFailure ==
    /\ s.phase = "response"
    /\ s' = [s EXCEPT !.phase = "evaluate", !.lastEvent = "TransientConclusionFailure"]

EvaluateTransient ==
    /\ s.phase = "evaluate"
    /\ LET retry == s.attempts < MaxAttempts + (IF Fault = "retry-overflow" THEN 1 ELSE 0)
       IN s' = [s EXCEPT
            !.phase = IF retry THEN "request" ELSE "done",
            !.warning = ~retry, !.result = IF retry THEN @ ELSE "warning",
            !.outcome = IF retry THEN @ ELSE FailedOutcome,
            !.lastEvent = IF retry THEN "RetryConclusion" ELSE "ExhaustConclusionRetries"]

DeadlineExpired ==
    /\ s.phase \in {"request", "response", "evaluate"}
    /\ s' = [s EXCEPT
        !.phase = IF Fault = "deadline-retry" THEN "request" ELSE "done",
        !.timedOut = TRUE, !.warning = TRUE, !.result = "warning",
        !.outcome = FailedOutcome, !.lastEvent = "ConclusionDeadlineExpired"]

FatalClientFailure ==
    /\ s.phase = "request"
    /\ s' = [s EXCEPT !.phase = "done", !.warning = TRUE,
             !.result = "warning", !.outcome = FailedOutcome, !.lastEvent = "ConclusionCredentialFailure"]

Next ==
    \/ SkipConclusion \/ GenerateSession \/ UploadArtifact \/ PrepareConclusion
    \/ SendConclusion \/ AcceptConclusion \/ ConflictingReplay \/ RejectConclusion
    \/ TransientFailure \/ EvaluateTransient \/ DeadlineExpired \/ FatalClientFailure

Spec == Init /\ [][Next]_s /\ WF_s(Next)

TypeOK ==
    /\ OutagePolicy \in {"ignore", "block"}
    /\ s.context \in Contexts
    /\ s.phase \in {"generate", "upload", "upload-retry", "prepare", "request", "response", "evaluate", "done"}
    /\ s.uploadAttempts \in 0..2 /\ s.uploadResult \in {"pending", "success", "failure"}
    /\ s.uploadedArtifacts \subseteq {"initial", "retry"}
    /\ s.artifactContent \in {"full-redacted", "compact-redacted"}
    /\ s.snapshot \in Payloads /\ s.sentPayload \in Payloads /\ s.stored \in Payloads
    /\ s.attempts \in 0..4 /\ s.processing \in 0..4 /\ s.financialEffects \in 0..4
    /\ s.credential \in {"none", "admission-oidc", "conclusion-oidc"}
    /\ s.traceId \in {"none", "caller-trace", "foreign-trace"}
    /\ s.firstReplica \in 0..2
    /\ s.result \in {"none", "skipped", "delivered", "warning"}
    /\ s.outcome \in {"success", "failure", "cancelled"}
    /\ \A value \in {s.eligible, s.generated, s.recovered, s.conflictSeen, s.timedOut, s.warning} :
           value \in BOOLEAN
ArtifactBeforeConclusion ==
    s.phase \in {"request", "response", "evaluate"} \/ s.attempts > 0 =>
        s.uploadResult # "pending" /\ s.uploadAttempts > 0
NoEndpointNoRequest ==
    s.context.endpoint = "absent" => s.attempts = 0 /\ s.credential = "none"
InvalidInputNoRequest ==
    s.context.endpoint = "invalid" \/ s.context.session = "invalid" => s.attempts = 0
ConclusionOIDCOnly == s.credential \in {"none", "conclusion-oidc"}
InvocationVerified == s.stored # NoPayload => s.context.identity = "valid"
CompactProjection == s.snapshot # NoPayload => s.snapshot.content = "compact-redacted"
BoundedPayload == s.snapshot.bytes <= 64
ArtifactReferences == s.snapshot # NoPayload => s.snapshot.artifacts = s.uploadedArtifacts
FullArtifactPreserved == s.artifactContent = "full-redacted"
StablePayload == s.attempts > 0 => s.sentPayload = s.snapshot
RetryBound == s.attempts <= MaxAttempts
DeadlineHonored == s.timedOut => s.phase \notin {"request", "response", "evaluate"}
BestEffort == s.outcome = s.context.outcome
DurableAcceptance == s.result = "delivered" => s.stored # NoPayload /\ s.processing = 1
SingleProcessing == s.processing <= 1
AcceptedSessionImmutable == s.stored # NoPayload => s.stored = s.snapshot
NoFinancialEffects == s.financialEffects = 0
TraceCorrelation == s.attempts > 0 => s.traceId = "caller-trace"
JobEligibility == s.attempts > 0 => s.eligible
Termination == <>(s.phase = "done")

NoDeliveredConclusion == ~(s.result = "delivered")
NoRetryRecovery == ~(s.result = "delivered" /\ s.recovered /\ s.attempts > 1)
NoUploadFailureDelivery == ~(s.result = "delivered" /\ s.uploadResult = "failure")
NoAgentFailureDelivery == ~(s.result = "delivered" /\ s.context.outcome = "failure")
NoIgnoredAdmissionDelivery == ~(s.result = "delivered" /\ s.context.admission = "ignored")
NoRejectedConflict == ~(s.conflictSeen /\ s.stored = s.snapshot /\ s.processing = 1)
NoDeadlineWarning == ~(s.phase = "done" /\ s.timedOut /\ s.warning)
NoArtifactReferenceDelivery == ~(s.result = "delivered" /\ s.snapshot.artifacts # {})
=============================================================================
