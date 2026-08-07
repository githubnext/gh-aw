# Formal Notes: replace-label-spec.md

**Last formalized**: 2026-08-07-15-46-35
**Notation**: TLA+ / Z3-style guard conjunction
**Issue**: (created via safe-output; number assigned post-run)

## Predicates

| ID | Predicate | Description |
|---|---|---|
| P1 | `SanitizedLabelValue` | label_to_remove/label_to_add sanitized: trimmed, control chars stripped (RL-007) |
| P2 | `RateLimitRetryEligible` | HTTP 429 always retries; HTTP 403 retries only with Retry-After header (RL-037/RL-048) |
| P3 | `RateLimitRetryBoundedRetries` | Retries bounded by max attempts; exhaustion is a hard error (RL-037/RL-045) |
| P4 | `NewLabelSetContainsAddExactlyOnce` | label_to_add appears exactly once in computed set; label_to_remove excluded (RL-041/RL-042) |
| P5 | `RESTFailureIsHardError` | Non-2xx setLabels response ⇒ {success:false, error} + core.error() (RL-046) |
| P6 | `LabelMustPreExist` | label_to_add must pre-exist in repo; no label creation on agent's behalf (RL-052) |
| P7 | `ServerSideEnforcementOnly` | Allow/blocklist evaluated server-side; agent-claimed bypass flag has no effect (RL-049) |
| P8 | `TokenScopeMinimum` | Token must include issues:write scope (RL-054/RL-051) |

## Key Invariants

- Blocklist evaluation precedes allowlist evaluation (security boundary) — RL-003
- label_to_remove absence on item is not an error; operation proceeds add-only (RL-034, idempotency goal)
- Staged mode MUST NOT invoke any write API call, even though gate-check reads may proceed (RL-027/RL-056)
- Partial-success (HTTP 200 but returned labels don't match expected) is treated as `rejected`, same SETLABELS_FAILED error code, no new error code introduced (RL-057/RL-058/RL-059)
- Single REST call (`PUT .../labels`) replaces the whole label set atomically — no separate add/remove round trip (RL-036)

## Edge Cases Identified

- label_to_add == label_to_remove: final set must contain exactly one occurrence, not zero
- label_to_add already present before replace: dedup must not produce two entries
- Rate limit exhausted after max retries: must hard-fail, not silently retry indefinitely
- Non-rate-limit HTTP errors (422, 500) must NOT enter the retry loop at all
- Agent-supplied "bypass" flags on the message must be structurally ignorable — validated only server-side

## Notes for Future Runs

- Core predicates (schema validation RL-004..009, count gate RL-010..012, target resolution RL-013..020,
  label validation ordering RL-021..023, gate checks RL-024..026, staged mode RL-027..028, label-set
  computation RL-029/033/034, cross-repo RL-014/015/050, transitions) are already covered in
  `pkg/workflow/replace_label_formal_test.go` and `replace_label_transitions_formal_test.go` — do not
  duplicate these in future runs on this spec.
- This run's new file `pkg/workflow/replace_label_security_formal_test.go` fills the §7 (error handling)
  and §8 (security considerations) gap: sanitization, rate-limit retry, REST hard-error semantics,
  label pre-existence enforcement, server-side-only allow/blocklist enforcement, and token scope minimum.
- Sandbox could not run `go build`/`go test` (repo requires go 1.26.5, sandbox has 1.25.12, toolchain
  download blocked by network policy) — validated via `gofmt` only. Recommend CI verification.
- Remaining lightly-formalized areas for a future pass: RL-055 (dedicated per-type token SHOULD),
  RL-053 (required-labels gate must use server-fetched state, not agent-supplied state) — could be
  formalized as a temporal/consistency predicate in a follow-up run.
