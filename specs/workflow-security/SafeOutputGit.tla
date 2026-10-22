---------------------------- MODULE SafeOutputGit ----------------------------
EXTENDS Naturals, Sequences

CONSTANTS Fault, AllowFullHistoryFetch, Shallow, ComparisonBudget, HistoryLimit
ASSUME AllowFullHistoryFetch \in BOOLEAN /\ Shallow \in BOOLEAN
ASSUME ComparisonBudget \in Nat \ {0} /\ HistoryLimit \in Nat \ {0}
ASSUME Fault \in {"none", "implicit-recovery", "optional-fetch", "lazy-fetch",
                  "wrong-recovery-token", "wrong-metadata-token",
                  "unbounded-local", "unbounded-history", "unbounded-comparisons",
                  "retry-global-error", "comparison-retry", "unbounded-request",
                  "ancestor-exemption", "unsafe-diagnostics"}

\* Finite outcome classes abstract process deadlines and output limits.
\* This is the push-to-PR bundle path, not an implementation of Git's DAG.
VARIABLE s
vars == <<s>>
Stages == {"prerequisites", "recover", "metadata", "inspect", "scan", "compare",
           "tree", "protect", "publish", "done"}
LocalOutcomes == {"ok", "missing", "unsupported", "timeout", "overflow"}
CompareOutcomes == {"behind", "identical", "ahead", "missing",
                    "authorization", "rate-limit", "service", "transport",
                    "timeout", "unknown"}
GlobalFailures == {"authorization", "rate-limit", "service", "transport",
                   "timeout", "unknown"}

Init ==
    \E missing, tipMerged, ancestorProtected, agentProtected \in BOOLEAN:
        s = [stage |-> "prerequisites", missing |-> missing, tipMerged |-> tipMerged,
             ancestorProtected |-> ancestorProtected, agentProtected |-> agentProtected,
             shallow |-> Shallow, recovered |-> FALSE,
             recoveryToken |-> "none", metadataToken |-> "none",
             optionalDownloaded |-> FALSE, sparseExpanded |-> FALSE,
             localBounded |-> TRUE, historyBounded |-> TRUE,
             requestBounded |-> TRUE, retries |-> 0, comparisons |-> 0,
             retriedGlobalFailure |-> FALSE, diagnosticsSafe |-> TRUE,
             baseline |-> "pr-head", protectionExempt |-> FALSE,
             fallback |-> FALSE, tipSeen |-> FALSE,
             continuedMissing |-> FALSE, published |-> FALSE, rejected |-> FALSE,
             lastEvent |-> "Init"]

CheckPrerequisites ==
    /\ s.stage = "prerequisites"
    /\ s' = [s EXCEPT
        !.stage = IF ~s.missing THEN "metadata"
                  ELSE IF AllowFullHistoryFetch \/ Fault = "implicit-recovery"
                       THEN "recover" ELSE "done",
        !.rejected = s.missing /\ ~AllowFullHistoryFetch /\ Fault # "implicit-recovery",
        !.lastEvent = "CheckPrerequisites"]

Recover(success) ==
    /\ s.stage = "recover"
    /\ s' = [s EXCEPT
        !.stage = IF success THEN "metadata" ELSE "done",
        !.recoveryToken = IF Fault = "wrong-recovery-token" THEN "target" ELSE "head",
        !.recovered = TRUE,
        !.shallow = IF success THEN FALSE ELSE @,
        !.rejected = ~success,
        !.diagnosticsSafe = success \/ Fault # "unsafe-diagnostics",
        !.lastEvent = IF success THEN "Recover" ELSE "RecoveryFailure"]

Metadata(outcome) ==
    /\ s.stage = "metadata"
    /\ LET success == outcome = "ok"
       IN s' = [s EXCEPT
           !.stage = IF success THEN "inspect" ELSE "protect",
           !.metadataToken = IF Fault = "wrong-metadata-token" THEN "head" ELSE "target",
           !.fallback = ~success,
           !.localBounded = ~(outcome \in {"timeout", "overflow"} /\ Fault = "unbounded-local"),
           !.diagnosticsSafe = success \/ Fault # "unsafe-diagnostics",
           !.lastEvent = IF success THEN "PinTargetMetadata" ELSE "MetadataFallback"]

Inspect(outcome) ==
    /\ s.stage = "inspect"
    /\ LET success == outcome = "ok"
           limited == outcome \in {"timeout", "overflow"}
           download == outcome = "missing" /\ Fault \in {"optional-fetch", "lazy-fetch"}
       IN s' = [s EXCEPT
           !.stage = IF success THEN IF s.tipMerged THEN "tree" ELSE "scan"
                     ELSE "protect",
           !.baseline = IF success /\ s.tipMerged THEN "tip" ELSE @,
           !.tipSeen = success /\ s.tipMerged,
           !.fallback = ~success,
           !.localBounded = ~(limited /\ Fault = "unbounded-local"),
           !.optionalDownloaded = download,
           !.shallow = IF download THEN FALSE ELSE @,
           !.sparseExpanded = download /\ Fault = "optional-fetch",
           !.diagnosticsSafe = success \/ Fault # "unsafe-diagnostics",
           !.lastEvent = "Inspect:" \o outcome]

Scan(records, outcome) ==
    /\ s.stage = "scan"
    /\ LET within == records <= HistoryLimit
           accept == outcome = "ok" /\ (within \/ Fault = "unbounded-history")
       IN s' = [s EXCEPT
           !.stage = IF accept THEN "compare" ELSE "protect",
           !.historyBounded = ~accept \/ within,
           !.localBounded = ~(outcome \in {"timeout", "overflow"} /\ Fault = "unbounded-local"),
           !.fallback = ~accept,
           !.diagnosticsSafe = outcome = "ok" \/ Fault # "unsafe-diagnostics",
           !.lastEvent = IF accept THEN "Scan" ELSE "ScanFallback"]

ComparisonLimit ==
    /\ s.stage = "compare" /\ s.comparisons >= ComparisonBudget
    /\ (Fault # "unbounded-comparisons" \/ s.comparisons > ComparisonBudget)
    /\ s' = [s EXCEPT !.stage = "protect", !.fallback = TRUE,
                      !.lastEvent = "ComparisonBudgetFallback"]

Compare(outcome) ==
    /\ s.stage = "compare"
    /\ (s.comparisons < ComparisonBudget \/
        (Fault = "unbounded-comparisons" /\ s.comparisons = ComparisonBudget))
    /\ LET confirmed == outcome \in {"behind", "identical"}
           global == outcome \in GlobalFailures
           retryGlobal == global /\ Fault = "retry-global-error"
           stop == global /\ ~retryGlobal
       IN s' = [s EXCEPT
           !.comparisons = @ + 1,
           !.stage = IF confirmed THEN "tree" ELSE IF stop THEN "protect" ELSE "compare",
           !.baseline = IF confirmed THEN "ancestor" ELSE @,
           !.fallback = stop,
           !.continuedMissing = @ \/ s.lastEvent = "Compare:missing",
           !.retriedGlobalFailure = @ \/ retryGlobal,
           !.retries = IF Fault = "comparison-retry" THEN 1 ELSE 0,
           !.requestBounded = ~(outcome = "timeout" /\ Fault = "unbounded-request"),
           !.diagnosticsSafe = ~global \/ Fault # "unsafe-diagnostics",
           !.lastEvent = "Compare:" \o outcome]

BuildTree(outcome) ==
    /\ s.stage = "tree"
    /\ s' = [s EXCEPT
        !.stage = "protect",
        !.baseline = IF outcome = "ok" THEN @ ELSE "pr-head",
        !.fallback = outcome # "ok",
        !.localBounded = ~(outcome \in {"timeout", "overflow"} /\ Fault = "unbounded-local"),
        !.diagnosticsSafe = outcome = "ok" \/ Fault # "unsafe-diagnostics",
        !.lastEvent = IF outcome = "ok" THEN "ValidationTree" ELSE "TreeFallback"]

ChooseProtection ==
    /\ s.stage = "protect"
    /\ s' = [s EXCEPT
        !.stage = "publish",
        !.protectionExempt = s.baseline = "tip" \/
                            (s.baseline = "ancestor" /\ Fault = "ancestor-exemption"),
        !.lastEvent = "ChooseProtection"]

Publish ==
    /\ s.stage = "publish"
    /\ LET denied == s.agentProtected \/ (s.ancestorProtected /\ ~s.protectionExempt)
       IN s' = [s EXCEPT !.stage = "done", !.published = ~denied,
                          !.rejected = denied,
                          !.lastEvent = IF denied THEN "ProtectedRejection" ELSE "Publish"]

Next ==
    \/ CheckPrerequisites \/ ComparisonLimit \/ ChooseProtection \/ Publish
    \/ \E success \in BOOLEAN: Recover(success)
    \/ \E outcome \in LocalOutcomes:
           Metadata(outcome) \/ Inspect(outcome) \/ BuildTree(outcome)
    \/ \E records \in {HistoryLimit, HistoryLimit + 1}, outcome \in LocalOutcomes:
           Scan(records, outcome)
    \/ \E outcome \in CompareOutcomes: Compare(outcome)

Spec == Init /\ [][Next]_vars

TypeOK ==
    /\ s.stage \in Stages
    /\ s.missing \in BOOLEAN /\ s.tipMerged \in BOOLEAN
    /\ s.ancestorProtected \in BOOLEAN /\ s.agentProtected \in BOOLEAN
    /\ s.shallow \in BOOLEAN /\ s.recovered \in BOOLEAN
    /\ s.recoveryToken \in {"none", "head", "target"}
    /\ s.metadataToken \in {"none", "head", "target"}
    /\ s.optionalDownloaded \in BOOLEAN /\ s.sparseExpanded \in BOOLEAN
    /\ s.localBounded \in BOOLEAN /\ s.historyBounded \in BOOLEAN
    /\ s.requestBounded \in BOOLEAN /\ s.retries \in 0..1
    /\ s.comparisons \in 0..(ComparisonBudget + 1)
    /\ s.retriedGlobalFailure \in BOOLEAN /\ s.diagnosticsSafe \in BOOLEAN
    /\ s.baseline \in {"pr-head", "tip", "ancestor"}
    /\ s.protectionExempt \in BOOLEAN /\ s.fallback \in BOOLEAN
    /\ s.tipSeen \in BOOLEAN
    /\ s.continuedMissing \in BOOLEAN
    /\ s.published \in BOOLEAN /\ s.rejected \in BOOLEAN

RecoveryOptIn == s.recovered => AllowFullHistoryFetch /\ s.missing
ShallowPreserved == ~s.recovered => s.shallow = Shallow
NoOptionalMaterialization == ~s.optionalDownloaded /\ ~s.sparseExpanded
RecoveryAuthentication == s.recovered => s.recoveryToken = "head"
MetadataAuthentication == s.metadataToken \in {"none", "target"}
BoundedLocalInspection == s.localBounded
BoundedHistory == s.historyBounded
BoundedComparisons == s.comparisons <= ComparisonBudget
ExactTipFastPath == s.tipSeen => s.comparisons = 0
ConservativeFallback == s.fallback => s.baseline = "pr-head"
NoGlobalFailureRetry == ~s.retriedGlobalFailure
NoComparisonRetries == s.retries = 0
BoundedRequests == s.requestBounded
ProtectionProvenance == s.protectionExempt => s.baseline = "tip"
ProtectedPublication ==
    s.published => ~s.agentProtected /\ (~s.ancestorProtected \/ s.baseline = "tip")
SafeDiagnostics == s.diagnosticsSafe

\* Deliberately false reachability predicates; checked only with Fault = "none".
NoAncestorBaseline == ~(s.stage = "publish" /\ s.baseline = "ancestor")
NoExactTipBaseline == ~(s.stage = "publish" /\ s.baseline = "tip")
NoRecovery == ~(s.recovered /\ s.stage = "metadata")
NoConservativeFallback == ~s.fallback
NoSafePublication == ~s.published
NoProtectedRejection == ~(s.rejected /\ s.stage = "done" /\ ~s.missing)
NoMissingParentContinuation == ~(s.continuedMissing /\ s.comparisons = 2)
=============================================================================
