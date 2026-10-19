------------------------- MODULE ActivationAdmission -------------------------
EXTENDS Naturals, TLC

CONSTANTS Fault, OutagePolicy, OTelEnabled
VARIABLE s

MaxAttempts == 3
PolicyCap == 2
OpaqueIDs == {"opaque-a", "opaque-b"}
Signals == {"span", "log", "metric"}
Contexts ==
    [endpoint : {"absent", "valid", "invalid"},
     identity : {"valid", "invalid"},
     triggeringActor : {"unused", "verified", "unverified"},
     localCap : 0..3,
     enforcement : BOOLEAN,
     hiddenSpend : {"low", "high"}]
Decisions ==
    [invocation : {"current", "foreign"}, cap : 1..3,
     cid : OpaqueIDs \cup {"opaque-foreign"}]
NoDecision == [invocation |-> "none", cap |-> 0, cid |-> "none"]
EmptyReply ==
    [kind |-> "none", invocation |-> "none", cap |-> 0,
     cid |-> "none", detail |-> "empty"]
MinimumCap(local, server) ==
    IF local = 0 \/ server < local THEN server ELSE local
Authorized ==
    /\ (s.context.identity = "valid" \/ Fault = "identity-bypass")
    /\ (s.context.triggeringActor # "unverified" \/ Fault = "trigger-bypass")
FreshDecision(cid) ==
    [invocation |-> IF Fault = "cross-invocation" THEN "foreign" ELSE "current",
     cap |-> IF Fault = "spend-ceiling" /\ s.context.hiddenSpend = "high"
             THEN 1 ELSE PolicyCap,
     cid |-> IF Fault = "cross-invocation" THEN "opaque-foreign" ELSE cid]
Decision(cid) ==
    IF s.cached = NoDecision THEN FreshDecision(cid)
    ELSE IF Fault = "retry-result"
         THEN [s.cached EXCEPT !.cid = IF @ = "opaque-a" THEN "opaque-b" ELSE "opaque-a"]
         ELSE s.cached

Init ==
    \E context \in Contexts :
        s = [context |-> context, phase |-> "configure", activation |-> "pending",
             admission |-> "none", attempts |-> 0, credential |-> "none",
             cached |-> NoDecision, firstDecision |-> NoDecision, reply |-> EmptyReply,
             firstReplica |-> 0, replica |-> 0, recovered |-> FALSE,
             denialAt |-> 0, malformed |-> FALSE, fatal |-> FALSE,
             effectiveCap |-> context.localCap, enforced |-> FALSE,
             cid |-> "none", runtimeCid |-> "none", started |-> FALSE,
             used |-> 0, totalUsed |-> 0, continuations |-> 0, usedBeforeContinuation |-> 0,
             privilegesActor |-> "original", financialEffects |-> 0,
             traceActive |-> FALSE, traceSent |-> FALSE,
             telemetry |-> [kind \in Signals |-> "none"],
             outcome |-> "none", lastEvent |-> "Init"]

Configure ==
    /\ s.phase = "configure"
    /\ s' = [s EXCEPT
        !.phase = IF s.context.endpoint = "valid"
                     \/ (s.context.endpoint = "absent" /\ Fault = "endpoint-bypass")
                  THEN "request"
                  ELSE IF s.context.endpoint = "absent" THEN "ready" ELSE "finalize",
        !.activation = IF s.context.endpoint = "absent" /\ Fault # "endpoint-bypass"
                       THEN "success"
                       ELSE IF s.context.endpoint = "invalid" THEN "failed" ELSE @,
        !.admission = IF s.context.endpoint = "absent" /\ Fault # "endpoint-bypass"
                      THEN "skipped"
                      ELSE IF s.context.endpoint = "invalid" THEN "blocked" ELSE @,
        !.lastEvent = "Configure"]

Send ==
    /\ s.phase = "request"
    /\ s' = [s EXCEPT
        !.phase = "response", !.attempts = @ + 1,
        !.credential = IF Fault = "credential-delegation" THEN "github-api" ELSE "oidc",
        !.traceActive = OTelEnabled \/ Fault = "force-tracing",
        !.traceSent = OTelEnabled \/ Fault = "force-tracing",
        !.lastEvent = "SendPOST"]

FatalClientFailure ==
    /\ s.phase = "request"
    /\ LET failOpen == Fault = "fatal-as-outage" /\ OutagePolicy = "ignore"
       IN s' = [s EXCEPT
            !.phase = IF failOpen THEN "ready" ELSE "finalize",
            !.activation = IF failOpen THEN "success" ELSE "failed",
            !.admission = IF failOpen THEN "outage" ELSE "blocked",
            !.fatal = TRUE, !.lastEvent = "FatalClientFailure"]

ReturnAllow ==
    /\ s.phase = "response"
    /\ Authorized
    /\ \E cid \in OpaqueIDs, replica \in {1, 2}, delivered \in BOOLEAN :
        LET decision == Decision(cid)
        IN s' = [s EXCEPT
            !.phase = "evaluate",
            !.cached = decision,
            !.firstDecision = IF @ = NoDecision THEN decision ELSE @,
            !.firstReplica = IF @ = 0 THEN replica ELSE @,
            !.replica = replica,
            !.recovered = s.cached # NoDecision /\ delivered /\ replica # s.firstReplica,
            !.reply = IF delivered
                      THEN [kind |-> "allow", invocation |-> decision.invocation,
                            cap |-> decision.cap, cid |-> decision.cid, detail |-> "empty"]
                      ELSE [EmptyReply EXCEPT !.kind = "transient"],
            !.financialEffects = IF Fault = "admission-charge" THEN @ + 1 ELSE @,
            !.lastEvent = IF delivered THEN "Return200" ELSE "Lose200Response"]

ReturnDeny ==
    /\ s.phase = "response"
    /\ s' = [s EXCEPT
        !.phase = "evaluate", !.denialAt = s.attempts,
        !.reply = [EmptyReply EXCEPT !.kind = "deny",
                   !.detail = IF Fault = "denial-leak" THEN s.context.hiddenSpend ELSE @],
        !.lastEvent = "Return403"]

ReturnMalformed ==
    /\ s.phase = "response"
    /\ s' = [s EXCEPT
        !.phase = "evaluate", !.malformed = TRUE,
        !.reply = [EmptyReply EXCEPT !.kind = "malformed"],
        !.lastEvent = "ReturnInvalidProtocol"]

ReturnTransient ==
    /\ s.phase = "response"
    /\ s' = [s EXCEPT
        !.phase = "evaluate",
        !.reply = [EmptyReply EXCEPT !.kind = "transient"],
        !.lastEvent = "ReturnTransient"]

EvaluateAllow ==
    /\ s.phase = "evaluate" /\ s.reply.kind = "allow"
    /\ s' = [s EXCEPT
        !.phase = IF s.context.enforcement THEN "ready" ELSE "finalize",
        !.activation = IF s.context.enforcement THEN "success" ELSE "failed",
        !.admission = IF s.context.enforcement THEN "admitted" ELSE "blocked",
        !.effectiveCap = IF Fault = "cap-not-applied" THEN s.context.localCap
                         ELSE MinimumCap(s.context.localCap, s.reply.cap),
        !.enforced = s.context.enforcement,
        !.cid = s.reply.cid, !.lastEvent = "Validate200AndBudget"]

EvaluateRejection ==
    /\ s.phase = "evaluate" /\ s.reply.kind \in {"deny", "malformed"}
    /\ LET failOpen ==
               OutagePolicy = "ignore"
               /\ ((s.reply.kind = "deny" /\ Fault = "deny-as-outage")
                   \/ (s.reply.kind = "malformed" /\ Fault = "malformed-as-outage"))
       IN s' = [s EXCEPT
            !.phase = IF failOpen THEN "ready" ELSE "finalize",
            !.activation = IF failOpen THEN "success" ELSE "failed",
            !.admission = IF failOpen THEN "outage" ELSE "blocked",
            !.lastEvent = "BlockRejection"]

EvaluateTransient ==
    /\ s.phase = "evaluate" /\ s.reply.kind = "transient"
    /\ LET retry == s.attempts < MaxAttempts + (IF Fault = "retry-overflow" THEN 1 ELSE 0)
           failOpen == OutagePolicy = "ignore" \/ Fault = "ignore-block"
       IN s' = [s EXCEPT
            !.phase = IF retry THEN "request" ELSE IF failOpen THEN "ready" ELSE "finalize",
            !.activation = IF retry THEN @ ELSE IF failOpen THEN "success" ELSE "failed",
            !.admission = IF retry THEN @ ELSE "outage",
            !.lastEvent = IF retry THEN "Retry" ELSE "ExhaustOutage"]

StartAgent ==
    /\ ((s.phase = "ready" /\ s.activation = "success")
        \/ (Fault = "activation-bypass" /\ s.phase = "finalize" /\ s.activation = "failed"))
    /\ s' = [s EXCEPT
        !.phase = "running", !.started = TRUE, !.runtimeCid = s.cid,
        !.privilegesActor = IF Fault = "actor-substitution" THEN "triggering" ELSE @,
        !.lastEvent = "StartAgent"]

ConsumeCredit ==
    /\ s.phase = "running"
    /\ s.totalUsed < 6
    /\ (s.effectiveCap = 0 \/ s.used < s.effectiveCap)
    /\ s' = [s EXCEPT !.used = @ + 1, !.totalUsed = @ + 1, !.lastEvent = "ConsumeCredit"]

ContinueAgent ==
    /\ s.phase = "running" /\ s.continuations = 0
    /\ s' = [s EXCEPT
        !.continuations = 1, !.used = IF Fault = "reset-budget" THEN 0 ELSE @,
        !.usedBeforeContinuation = s.totalUsed,
        !.lastEvent = "ContinueAgent"]

FinishAgent ==
    /\ s.phase = "running"
    /\ \E outcome \in {"success", "failure", "cancelled"} :
        s' = [s EXCEPT !.phase = "finalize", !.outcome = outcome, !.lastEvent = "FinishAgent"]

FinishTelemetry ==
    /\ s.phase = "finalize"
    /\ s' = [s EXCEPT
        !.phase = "done",
        !.telemetry = [kind \in Signals |->
            IF ~OTelEnabled THEN "disabled"
            ELSE IF Fault = "correlation-drop" /\ kind = "metric" THEN "none"
            ELSE IF s.started THEN s.runtimeCid ELSE s.cid],
        !.lastEvent = "FinishTelemetry"]

Next ==
    \/ Configure \/ Send \/ FatalClientFailure
    \/ ReturnAllow \/ ReturnDeny \/ ReturnMalformed \/ ReturnTransient
    \/ EvaluateAllow \/ EvaluateRejection \/ EvaluateTransient
    \/ StartAgent \/ ConsumeCredit \/ ContinueAgent \/ FinishAgent \/ FinishTelemetry

Spec == Init /\ [][Next]_s /\ WF_s(Next)

TypeOK ==
    /\ OutagePolicy \in {"ignore", "block"} /\ OTelEnabled \in BOOLEAN
    /\ s.context \in Contexts
    /\ s.phase \in {"configure", "request", "response", "evaluate", "ready",
                   "running", "finalize", "done"}
    /\ s.activation \in {"pending", "success", "failed"}
    /\ s.admission \in {"none", "skipped", "admitted", "blocked", "outage"}
    /\ s.attempts \in 0..4 /\ s.denialAt \in 0..4
    /\ s.credential \in {"none", "oidc", "github-api"}
    /\ s.cached \in Decisions \cup {NoDecision} /\ s.firstDecision \in Decisions \cup {NoDecision}
    /\ s.reply \in [kind : {"none", "allow", "deny", "malformed", "transient"},
                    invocation : {"none", "current", "foreign"}, cap : 0..3,
                    cid : OpaqueIDs \cup {"none", "opaque-foreign"},
                    detail : {"empty", "low", "high"}]
    /\ s.firstReplica \in 0..2 /\ s.replica \in 0..2
    /\ s.effectiveCap \in 0..3 /\ s.used \in 0..6 /\ s.totalUsed \in 0..6
    /\ s.continuations \in 0..1 /\ s.financialEffects \in 0..4
    /\ s.usedBeforeContinuation \in 0..6
    /\ s.cid \in OpaqueIDs \cup {"none", "opaque-foreign"}
    /\ s.runtimeCid \in OpaqueIDs \cup {"none", "opaque-foreign"}
    /\ s.privilegesActor \in {"original", "triggering"}
    /\ s.telemetry \in [Signals -> OpaqueIDs \cup {"none", "opaque-foreign", "disabled"}]
    /\ s.outcome \in {"none", "success", "failure", "cancelled"}
    /\ \A value \in {s.recovered, s.malformed, s.fatal, s.enforced,
                    s.started, s.traceActive, s.traceSent} : value \in BOOLEAN

ActivationGate == s.started => s.activation = "success"
NoEndpointNoRequest == s.context.endpoint = "absent" => s.attempts = 0 /\ s.credential = "none"
InvalidConfigurationBlocks == s.context.endpoint = "invalid" => s.activation # "success" /\ s.attempts = 0
OIDCOnly == s.credential \in {"none", "oidc"}
IdentityVerified == s.firstDecision # NoDecision => s.context.identity = "valid"
TriggeringActorVerified == s.firstDecision # NoDecision => s.context.triggeringActor # "unverified"
OriginalActorPrivileges == s.privilegesActor = "original"
InvocationBound ==
    IF s.cached = NoDecision THEN TRUE
    ELSE s.cached.invocation = "current" /\ s.cached.cid \in OpaqueIDs
StableRetryDecision == s.cached = s.firstDecision
RetryBound == s.attempts <= MaxAttempts
DenialBlocks == s.denialAt > 0 => s.activation # "success" /\ s.attempts = s.denialAt
MalformedBlocks == s.malformed => s.activation # "success"
FatalErrorsBlock == s.fatal => s.activation = "failed"
OutagePolicyHonored == s.admission = "outage" /\ OutagePolicy = "block" => s.activation = "failed"
CeilingApplied ==
    s.admission = "admitted" =>
        s.enforced /\ s.effectiveCap = MinimumCap(s.context.localCap, s.reply.cap)
        /\ s.effectiveCap > 0
SharedBudget ==
    s.admission = "admitted" => s.used = s.totalUsed /\ s.totalUsed <= s.effectiveCap
NoFabricatedAdmission ==
    s.admission \in {"skipped", "outage"} =>
        s.cid = "none" /\ ~s.enforced /\ s.effectiveCap = s.context.localCap
PrivateDenial == s.reply.kind = "deny" => s.reply.detail = "empty"
SpendPrivateCeiling == IF s.firstDecision = NoDecision THEN TRUE ELSE s.firstDecision.cap = PolicyCap
NoFinancialEffects == s.financialEffects = 0
CorrelationPreserved ==
    /\ (s.admission = "admitted" => s.cid \in OpaqueIDs)
    /\ (s.started => s.runtimeCid = s.cid)
    /\ (s.phase = "done" /\ OTelEnabled => \A kind \in Signals : s.telemetry[kind] = s.cid)
OptionalTracing ==
    ~OTelEnabled =>
        ~s.traceActive /\ ~s.traceSent
        /\ (s.phase = "done" => \A kind \in Signals : s.telemetry[kind] = "disabled")
Termination == <>(s.phase = "done")

NoAdmittedStart == ~(s.started /\ s.admission = "admitted")
NoSkippedStart == ~(s.started /\ s.admission = "skipped")
NoOutageStart == ~(s.started /\ s.admission = "outage")
NoRetryRecovery == ~(s.started /\ s.admission = "admitted" /\ s.recovered /\ s.attempts > 1)
NoVerifiedRerun == ~(s.started /\ s.admission = "admitted" /\ s.context.triggeringActor = "verified")
NoUntracedAdmission == ~(s.started /\ s.admission = "admitted" /\ ~OTelEnabled /\ ~s.traceSent)
NoSharedContinuation ==
    ~(s.started /\ s.admission = "admitted" /\ s.continuations = 1
      /\ s.usedBeforeContinuation > 0 /\ s.totalUsed > s.usedBeforeContinuation)
NoCompletedDenial == ~(s.phase = "done" /\ s.denialAt > 0 /\ ~s.started)
=============================================================================
