----------------------------- MODULE DailyAIC -----------------------------
EXTENDS Naturals, Integers, FiniteSets, Sequences, TLC

CONSTANTS Runs, AccountingKinds, ProducerKinds, CacheKinds, MaxAttempts, Fault, Witness, TransportMode
ASSUME /\ Runs \subseteq {1, 2} /\ Runs # {}
       /\ MaxAttempts = 3
       /\ Fault \in {"none", "partial_history", "estimate_quota",
                     "reuse_estimate", "trust_issue_branch", "double_count",
                     "resurrect_detection_summary", "bypass_reserve",
                     "charge_storage", "charge_outage"}
       /\ TransportMode \in {"abstract", "concrete"}

TransportEnabled == TransportMode = "concrete"
Reserve == 100
QuotaStarts == IF TransportEnabled THEN {101, 105, 110} ELSE {110}

Components == {"agent", "detection", "evals"}
MeasuredKinds == {"raw", "agent_jsonl", "agent_json", "detection_jsonl",
                  "response_credits", "explicit_credits", "priced_tokens",
                  "retained_prior_attempt"}
ComponentZeroKinds == {"component_skipped", "runner_not_assigned", "execution_not_started",
                       "empty_detection", "failed_agent_empty",
                       "legacy_startup_failure", "legacy_sample_replay",
                       "failed_agent_missing", "failed_evals", "explicit_zero"}
UnresolvedKinds == {"missing_usage", "missing_component", "unreadable_preferred",
                    "malformed", "unknown_pricing", "stale_artifact",
                    "unverified_rerun", "conflicting_request", "zero_alias_unknown_pricing"}
AllAccountingKinds == MeasuredKinds \cup ComponentZeroKinds \cup UnresolvedKinds
JobZeroKinds == {"all_jobs_skipped", "no_billable_jobs"}
AllProducerKinds == {"push", "schedule", "workflow_dispatch", "issues",
                     "issue_comment", "pull_request_target", "pull_request",
                     "bad_branch", "fork", "other_workflow", "other_path",
                     "other_repository", "unknown_default"}
AllCacheKinds == {"absent", "expired", "invalid", "legacy", "stale",
                 "mismatched_attempt", "estimated", "recorded",
                 "full_recorded", "conflict"}
ASSUME /\ AccountingKinds \subseteq AllAccountingKinds
       /\ ProducerKinds \subseteq AllProducerKinds
       /\ CacheKinds \subseteq AllCacheKinds

Config(backend, limit, maximum, token, bypass, trusted, insecure, continue) ==
    [backend |-> backend, limit |-> limit, maximum |-> maximum,
     token |-> token, bypass |-> bypass, trusted |-> trusted,
     insecure |-> insecure, continue |-> continue]
Inputs ==
    {Config("artifacts", l, m, TRUE, FALSE, FALSE, FALSE, FALSE):
        l \in {0, 3, 9}, m \in {0, 8}}
    \cup {Config("artifacts", 9, 8, FALSE, FALSE, FALSE, FALSE, FALSE),
          Config("artifacts", 9, 8, FALSE, FALSE, FALSE, FALSE, TRUE),
          Config("artifacts", 9, 8, TRUE, FALSE, FALSE, FALSE, TRUE),
          Config("artifacts", 9, 8, TRUE, TRUE, FALSE, FALSE, FALSE)}
    \cup {Config("repo-memory", l, 0, FALSE, FALSE, t, i, FALSE):
        l \in {3, 9}, t \in BOOLEAN, i \in BOOLEAN}

Producer(k) ==
    [event |-> IF k \in {"push", "schedule", "workflow_dispatch", "issues",
                         "issue_comment", "pull_request_target", "pull_request"}
               THEN k ELSE "issues",
     workflow |-> k # "other_workflow", path |-> k # "other_path",
     repository |-> k # "other_repository", defaultKnown |-> k # "unknown_default",
     defaultHead |-> k # "bad_branch", headRepository |-> k # "fork"]
CorrectTrust(p) ==
    /\ p.workflow /\ p.path /\ p.repository
    /\ (p.event = "pull_request_target" \/
        (p.event \in {"push", "schedule", "workflow_dispatch", "issues", "issue_comment"}
         /\ p.defaultKnown /\ p.defaultHead /\ p.headRepository))
RestoreTrust(p) ==
    CorrectTrust(p) \/ (Fault = "trust_issue_branch" /\ p.event = "issues")

RawValue(r, c) == IF c = "agent" THEN r ELSE 1
ComponentValue(r, kind, c) ==
    IF kind = "explicit_zero"
       \/ (kind = "empty_detection" /\ c = "detection")
       \/ (kind = "failed_evals" /\ c = "evals")
       \/ (kind \in ComponentZeroKinds \ {"explicit_zero", "empty_detection", "failed_evals"}
           /\ c = "agent")
    THEN 0 ELSE RawValue(r, c)
RECURSIVE Sum(_, _)
Sum(keys, values) ==
    IF keys = {} THEN 0
    ELSE LET k == CHOOSE item \in keys: TRUE
         IN values[k] + Sum(keys \ {k}, values)
AccountingValue(r, kind) ==
    IF kind \in JobZeroKinds THEN 0
    ELSE Sum(Components, [c \in Components |-> ComponentValue(r, kind, c)])
CacheValue(r) == Sum(Components, [c \in Components |-> RawValue(r, c)])

\* Valid abstracts schema, numeric, repository and workflow checks; stamp models
\* an inclusive window [0, 2]. Sequence index breaks equal-timestamp ties.
Entry(r, stamp, value, valid) ==
    [run |-> r, stamp |-> stamp, value |-> value, valid |-> valid]
LedgerSamples ==
    {<<>>, <<Entry(1, 1, 2, TRUE)>>,
     <<Entry(1, 0, 1, TRUE), Entry(1, 2, 4, TRUE), Entry(1, 2, 3, TRUE)>>,
     <<Entry(1, -1, 8, TRUE), Entry(1, 3, 8, TRUE), Entry(1, 1, 8, FALSE)>>,
     <<Entry(1, 1, 2, TRUE), Entry(2, 2, 4, TRUE)>>}
LedgerIndices(entries, r) ==
    {i \in 1..Len(entries): entries[i].valid /\ entries[i].run = r
                           /\ entries[i].stamp \in 0..2}
LedgerValue(entries, r) ==
    LET indices == LedgerIndices(entries, r)
    IN IF indices = {} THEN 0
       ELSE LET last == CHOOSE i \in indices:
                    \A j \in indices:
                        entries[j].stamp < entries[i].stamp
                        \/ (entries[j].stamp = entries[i].stamp /\ j <= i)
            IN entries[last].value

ReadPhases == {"restore", "current", "workflow_history", "repo_history",
               "jobs", "artifacts", "redirect", "download"}
GitHubReadPhases == ReadPhases \ {"download"}
HistoricalReads == {"jobs", "artifacts", "redirect", "download"}
Failures == {"transient", "quota", "restricted", "unauthorized", "forbidden", "not_found"}
KnownResults == {"under_budget", "exceeded"}
ErrorResults == {"structural_error", "transient_error"}
Phases == ReadPhases \cup {"start", "resolve", "account", "ledger", "decide", "done"}
Status(cause) ==
    IF cause \in {"missing_token", "untrusted_ledger", "unauthorized", "forbidden", "not_found"}
    THEN "structural_error" ELSE "transient_error"
Maximum(a, b) == IF a > b THEN a ELSE b

VARIABLE s
vars == <<s>>
Init ==
    \E input \in Inputs, quota \in QuotaStarts:
        s = [input |-> input, phase |-> "start", status |-> "not_run",
             candidates |-> {}, complete |-> FALSE, counted |-> {}, active |-> 0,
             values |-> [r \in Runs |-> 0], sources |-> [r \in Runs |-> "none"],
             evidence |-> [r \in Runs |-> "none"],
             estimateCause |-> [r \in Runs |-> "none"],
             total |-> 0, estimated |-> 0, persisted |-> {},
             attempt |-> 0, maxAttempt |-> 0, recovered |-> FALSE,
             producer |-> Producer("pull_request"), cacheKind |-> "absent",
             cacheHits |-> {}, lookup |-> "workflow", failure |-> "none",
             ledger |-> <<>>, allowAgent |-> FALSE,
             remaining |-> quota, quotaStart |-> quota, epochResponses |-> 0,
             apiResponses |-> 0, resets |-> 0,
             networkUp |-> TRUE, networkChanges |-> 0,
             outageObserved |-> FALSE, outageEstimated |-> FALSE,
             storageReads |-> 0, storageCharges |-> 0, networkCharges |-> 0]

Stopped(cause) ==
    [s EXCEPT !.phase = "done", !.status = Status(cause),
                  !.failure = cause, !.allowAgent = s.input.continue,
                  !.maxAttempt = Maximum(@, s.attempt + 1)]
Commit(next, observed) ==
    LET storage == TransportEnabled /\ observed /\ s.phase = "download"
        storageCharge == storage /\ Fault = "charge_storage"
        networkCharge == TransportEnabled /\ ~observed
                         /\ s.phase \in GitHubReadPhases /\ Fault = "charge_outage"
        charged == TransportEnabled /\
                   ((observed /\ s.phase \in GitHubReadPhases)
                    \/ storageCharge \/ networkCharge)
        remaining == s.remaining - IF charged THEN 1 ELSE 0
        blocked == charged /\ remaining <= Reserve /\ Fault # "bypass_reserve"
        result == IF blocked THEN Stopped("quota") ELSE next
    IN s' = [result EXCEPT
                !.remaining = remaining,
                !.epochResponses = s.epochResponses + IF charged THEN 1 ELSE 0,
                !.apiResponses = s.apiResponses + IF charged THEN 1 ELSE 0,
                !.storageReads = s.storageReads + IF storage THEN 1 ELSE 0,
                !.storageCharges = s.storageCharges + IF storageCharge THEN 1 ELSE 0,
                !.networkCharges = s.networkCharges + IF networkCharge THEN 1 ELSE 0,
                !.outageObserved = @ \/ (TransportEnabled /\ ~observed /\ s.phase \in ReadPhases)]
Stop(cause) == Commit(Stopped(cause), s.networkUp)
NetworkChange ==
    /\ TransportEnabled /\ s.phase \in ReadPhases /\ s.networkChanges < 2
    /\ s' = [s EXCEPT !.networkUp = ~@, !.networkChanges = @ + 1]
QuotaReset ==
    /\ TransportEnabled /\ s.phase \in ReadPhases /\ s.resets = 0
    /\ s' = [s EXCEPT !.remaining = 110, !.quotaStart = 110,
                     !.epochResponses = 0, !.resets = 1]
Start ==
    /\ s.phase = "start"
    /\ IF s.input.limit = 0
       THEN s' = [s EXCEPT !.phase = "done", !.status = "disabled", !.allowAgent = TRUE]
       ELSE IF s.input.bypass
       THEN s' = [s EXCEPT !.phase = "done", !.status = "skipped", !.allowAgent = TRUE]
       ELSE IF s.input.backend = "repo-memory"
       THEN IF s.input.trusted \/ s.input.insecure
            THEN s' = [s EXCEPT !.phase = "ledger"]
            ELSE Stop("untrusted_ledger")
       ELSE IF ~s.input.token THEN Stop("missing_token")
       ELSE s' = [s EXCEPT !.phase = "restore"]

Advance(phase) ==
    Commit([s EXCEPT !.phase = phase, !.attempt = 0,
                  !.maxAttempt = Maximum(@, s.attempt + 1),
                  !.recovered = @ \/ s.attempt > 0], TRUE)
Restore ==
    /\ s.phase = "restore" /\ s.networkUp
    /\ \E k \in CacheKinds, p \in ProducerKinds:
        IF k = "conflict" /\ RestoreTrust(Producer(p))
        THEN Stop("conflicting_snapshot")
        ELSE Commit([s EXCEPT !.phase = "current", !.cacheKind = k,
                           !.producer = Producer(p), !.attempt = 0,
                           !.maxAttempt = Maximum(@, s.attempt + 1),
                           !.recovered = @ \/ s.attempt > 0], TRUE)
HasRecordedCache(r) ==
    RestoreTrust(s.producer)
    /\ (s.cacheKind = "full_recorded" \/ (s.cacheKind = "recorded" /\ r = 1))
HasEstimatedCache(r) ==
    RestoreTrust(s.producer) /\ s.cacheKind = "estimated" /\ r = 1

Settle(r, value, source, evidence, cause, hit) ==
    Commit([s EXCEPT !.phase = "resolve", !.active = 0, !.attempt = 0,
                  !.maxAttempt = Maximum(@, s.attempt + 1),
                  !.counted = @ \cup {r}, !.persisted = @ \cup {r},
                  !.values[r] = value, !.sources[r] = source, !.evidence[r] = evidence,
                  !.estimateCause[r] = IF source = "estimated" THEN cause ELSE "none",
                  !.recovered = @ \/ (source = "recorded" /\ s.attempt > 0),
                  !.outageEstimated = @ \/ (source = "estimated" /\ ~s.networkUp),
                  !.total = @ + IF Fault = "double_count" THEN 2 * value ELSE value,
                  !.estimated = @ + IF source = "estimated" THEN value ELSE 0,
                  !.cacheHits = IF hit THEN @ \cup {r} ELSE @], s.networkUp)
Estimate(cause, evidence) ==
    IF s.input.maximum > 0
       /\ (cause \in {"transient", "unresolved"}
           \/ (Fault = "estimate_quota" /\ cause = "quota"))
    THEN Settle(s.active, s.input.maximum, "estimated", evidence, cause, FALSE)
    ELSE Stop(cause)

History ==
    /\ s.phase \in {"workflow_history", "repo_history"} /\ s.networkUp
    /\ \E candidates \in SUBSET Runs:
        Commit([s EXCEPT !.phase = "resolve", !.candidates = candidates,
                       !.complete = TRUE, !.attempt = 0,
                       !.maxAttempt = Maximum(@, s.attempt + 1),
                       !.recovered = @ \/ s.attempt > 0], TRUE)
History404 ==
    /\ s.phase = "workflow_history" /\ s.networkUp
    /\ Commit([s EXCEPT !.phase = "repo_history", !.lookup = "repository",
                     !.attempt = 0, !.maxAttempt = Maximum(@, s.attempt + 1)], TRUE)
IncompleteHistory ==
    /\ s.phase \in {"workflow_history", "repo_history"} /\ s.networkUp
    /\ \E cause \in {"incomplete_history", "invalid_history"}:
        IF Fault = "partial_history"
        THEN s' = [s EXCEPT !.phase = "resolve", !.candidates = Runs, !.complete = FALSE]
        ELSE Stop(cause)

ReadSuccess ==
    /\ s.networkUp
    /\ (\/ (s.phase = "current" /\ Advance("workflow_history"))
        \/ (s.phase = "jobs" /\
            (Advance("artifacts") \/
             \E kind \in JobZeroKinds: Settle(s.active, 0, "recorded", kind, "none", FALSE)))
        \/ (s.phase = "artifacts" /\ Advance("redirect"))
        \/ (s.phase = "redirect" /\ Advance("download"))
        \/ (s.phase = "download" /\ Advance("account")))
ReadFailure ==
    /\ s.phase \in ReadPhases
    /\ \E cause \in Failures:
        /\ s.networkUp \/ cause = "transient"
        /\ ~(s.phase = "workflow_history" /\ cause = "not_found")
        /\ IF cause = "transient" /\ s.attempt + 1 < MaxAttempts
           THEN Commit([s EXCEPT !.attempt = @ + 1,
                              !.maxAttempt = Maximum(@, s.attempt + 1)], s.networkUp)
           ELSE IF s.phase \in HistoricalReads THEN Estimate(cause, cause)
           ELSE Stop(cause)
Resolve ==
    /\ s.phase = "resolve" /\ s.counted # s.candidates
    /\ LET r == CHOOSE run \in s.candidates \ s.counted: TRUE
       IN IF HasRecordedCache(r)
             \/ (Fault = "reuse_estimate" /\ HasEstimatedCache(r))
          THEN Settle(r, CacheValue(r), "recorded", "scan_cache", "none", TRUE)
          ELSE s' = [s EXCEPT !.phase = "jobs", !.active = r, !.attempt = 0]
Account ==
    /\ s.phase = "account"
    /\ \E kind \in AccountingKinds:
        IF kind \in UnresolvedKinds THEN Estimate("unresolved", kind)
        ELSE LET value == AccountingValue(s.active, kind)
             IN Settle(s.active,
                       IF Fault = "resurrect_detection_summary" /\ kind = "empty_detection"
                       THEN value + 8 ELSE value,
                       "recorded", kind, "none", FALSE)

ReadLedger ==
    /\ s.phase = "ledger"
    /\ \E entries \in LedgerSamples:
        LET values == [r \in Runs |-> LedgerValue(entries, r)]
            counted == {r \in Runs: LedgerIndices(entries, r) # {}}
        IN s' = [s EXCEPT !.phase = "decide", !.ledger = entries,
                         !.values = values, !.counted = counted,
                         !.sources = [r \in Runs |-> IF r \in counted THEN "recorded" ELSE "none"],
                         !.evidence = [r \in Runs |-> IF r \in counted THEN "ledger" ELSE "none"],
                         !.total = Sum(counted, values)]
FinishScan ==
    /\ s.phase = "resolve" /\ s.counted = s.candidates
    /\ s' = [s EXCEPT !.phase = "decide"]
Decide ==
    /\ s.phase = "decide"
    /\ s' = [s EXCEPT !.phase = "done",
                     !.status = IF s.total < s.input.limit THEN "under_budget" ELSE "exceeded",
                     !.allowAgent = s.total < s.input.limit]
Done == s.phase = "done" /\ UNCHANGED s
Next == Start \/ Restore \/ History \/ History404 \/ IncompleteHistory
        \/ ReadSuccess \/ ReadFailure \/ Resolve \/ Account \/ ReadLedger
        \/ FinishScan \/ Decide \/ Done \/ NetworkChange \/ QuotaReset
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)

TypeOK ==
    /\ s.input \in Inputs /\ s.phase \in Phases
    /\ s.status \in KnownResults \cup ErrorResults \cup {"not_run", "disabled", "skipped"}
    /\ s.counted \subseteq Runs /\ s.candidates \subseteq Runs
    /\ s.active \in Runs \cup {0}
    /\ s.values \in [Runs -> Nat]
    /\ s.sources \in [Runs -> {"none", "recorded", "estimated"}]
    /\ s.total \in Nat /\ s.estimated \in Nat
    /\ s.attempt \in 0..(MaxAttempts - 1)
    /\ s.allowAgent \in BOOLEAN
    /\ s.remaining \in Nat /\ s.apiResponses \in Nat
    /\ s.epochResponses \in Nat /\ s.resets \in 0..1
    /\ s.networkUp \in BOOLEAN /\ s.networkChanges \in 0..2
    /\ s.outageObserved \in BOOLEAN /\ s.outageEstimated \in BOOLEAN
QuotaBalance == s.remaining = s.quotaStart - s.epochResponses
CircuitBreaker ==
    (TransportEnabled /\ s.remaining <= Reserve)
    => s.phase = "done" /\ s.failure = "quota" /\ s.status \in ErrorResults
StorageDoesNotConsumeQuota == s.storageCharges = 0
NetworkAccounting == s.networkCharges = 0
CompleteWindowRequired ==
    (s.input.backend = "artifacts" /\ s.status \in KnownResults)
    => (s.complete /\ s.counted = s.candidates)
TotalsReconcile == s.total = Sum(s.counted, s.values)
EstimatesReconcile ==
    s.estimated = Sum({r \in s.counted: s.sources[r] = "estimated"}, s.values)
WholeRunMaximum ==
    \A r \in s.counted:
        s.sources[r] = "estimated" => s.input.maximum > 0 /\ s.values[r] = s.input.maximum
NoForbiddenEstimates ==
    \A r \in s.counted:
        s.sources[r] = "estimated" => s.estimateCause[r] \in {"transient", "unresolved"}
RecordedCacheOnly == \A r \in s.cacheHits: HasRecordedCache(r)
CacheProducerTrusted == s.cacheHits # {} => CorrectTrust(s.producer)
RecordedEvidence ==
    \A r \in s.counted:
        (s.sources[r] = "recorded" /\ s.evidence[r] # "ledger")
        => IF s.evidence[r] = "scan_cache" THEN s.values[r] = CacheValue(r)
           ELSE /\ s.evidence[r] \in MeasuredKinds \cup ComponentZeroKinds \cup JobZeroKinds
                /\ s.values[r] = AccountingValue(r, s.evidence[r])
ZeroEvidence ==
    \A r \in s.counted:
        (s.input.backend = "artifacts" /\ s.sources[r] = "recorded" /\ s.values[r] = 0)
        => s.evidence[r] \in JobZeroKinds \cup {"explicit_zero", "scan_cache"}
ThresholdDecision ==
    /\ (s.status = "under_budget" => s.total < s.input.limit)
    /\ (s.status = "exceeded" => s.total >= s.input.limit)
UnknownNeverUnderBudget == s.failure # "none" => s.status \notin KnownResults
BypassDoesNotRead ==
    s.status \in {"disabled", "skipped"}
    => s.counted = {} /\ s.total = 0 /\ s.maxAttempt = 0
ReadAttemptBound == s.maxAttempt <= MaxAttempts
LedgerTrust ==
    (s.input.backend = "repo-memory" /\ s.status \in KnownResults)
    => s.input.trusted \/ s.input.insecure
LedgerReconcile ==
    (s.input.backend = "repo-memory" /\ s.status \in KnownResults)
    => s.total = Sum(Runs, [r \in Runs |-> LedgerValue(s.ledger, r)]) /\ s.estimated = 0
ContinuationPolicy ==
    s.phase = "done" => s.allowAgent =
        (s.status \in {"under_budget", "disabled", "skipped"}
         \/ (s.status \in ErrorResults /\ s.input.continue))
EventuallyDone == <> (s.phase = "done")

Reached ==
    CASE Witness \in AllAccountingKinds \cup JobZeroKinds ->
            \E r \in s.counted: s.evidence[r] = Witness
      [] Witness = "estimated" -> s.estimated > 0
      [] Witness = "cache_hit" -> s.cacheHits # {}
      [] Witness = "history_fallback" -> s.lookup = "repository" /\ s.status \in KnownResults
      [] Witness = "retry_recovery" -> s.recovered /\ s.status \in KnownResults
      [] Witness = "ledger" -> s.input.backend = "repo-memory" /\ s.status \in KnownResults
      [] Witness = "ledger_empty" -> s.input.backend = "repo-memory" /\ s.status = "under_budget" /\ s.total = 0
      [] Witness = "issues_cache" -> s.producer.event = "issues" /\ s.cacheHits # {}
      [] Witness = "issue_comment_cache" -> s.producer.event = "issue_comment" /\ s.cacheHits # {}
      [] Witness = "estimated_reinspection" ->
            s.cacheKind = "estimated" /\ \E r \in s.counted:
                s.sources[r] = "recorded" /\ s.evidence[r] # "scan_cache"
      [] Witness = "rerun_reinspection" ->
            s.cacheKind = "mismatched_attempt" /\ s.counted # {} /\ s.cacheHits = {}
      [] Witness = "missing_maximum" ->
            s.input.maximum = 0 /\ s.failure = "unresolved" /\ s.status \in ErrorResults
      [] Witness = "quota_stop" -> s.failure = "quota" /\ s.status \in ErrorResults
      [] Witness = "permission_stop" ->
            s.failure \in {"unauthorized", "forbidden", "not_found"} /\ s.status \in ErrorResults
      [] Witness = "invalid_history" -> s.failure = "invalid_history" /\ s.status \in ErrorResults
      [] Witness = "restricted_stop" -> s.failure = "restricted" /\ s.status \in ErrorResults
      [] Witness = "reserve_stop" -> TransportEnabled /\ s.remaining <= Reserve /\ s.failure = "quota"
      [] Witness = "quota_reset" -> s.resets = 1 /\ s.status \in KnownResults
      [] Witness = "network_recovery" ->
            s.networkChanges = 2 /\ s.networkUp /\ s.outageObserved /\ s.status \in KnownResults
      [] Witness = "outage_exhausted" ->
            ~s.networkUp /\ s.outageObserved /\ s.failure = "transient" /\ s.maxAttempt = MaxAttempts
      [] Witness = "outage_estimate" -> s.outageEstimated
      [] Witness = "signed_storage_isolated" -> s.storageReads > 0 /\ s.status \in KnownResults
      [] Witness = "partial_snapshot" ->
            s.status \in ErrorResults /\ s.persisted # {} /\ s.counted # s.candidates
      [] Witness = "mixed_sources" ->
            s.status \in KnownResults
            /\ (\E r \in s.counted: s.sources[r] = "recorded")
            /\ (\E r \in s.counted: s.sources[r] = "estimated")
      [] Witness = "under_budget" -> s.status = "under_budget"
      [] Witness = "exceeded" -> s.status = "exceeded"
      [] Witness = "disabled" -> s.status = "disabled"
      [] Witness = "skipped" -> s.status = "skipped"
      [] OTHER -> FALSE
WitnessNotReached == ~Reached
=============================================================================
