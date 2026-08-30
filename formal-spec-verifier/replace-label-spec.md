# Formal Notes: replace-label-spec.md

**Last formalized**: 2026-08-30-15-28-16
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
| P9 | `PreWriteAllowBlocklistRecheck` | Allow/blocklist re-checked against current policy immediately before write, independent of earlier gate pass (RL-049a) |
| P10 | `RequiredLabelsGateServerState` | required-labels gate MUST use server-fetched label state, not agent-supplied state (RL-053) |
| P11 | `TokenOverPrivilegeRecommendsDedicated` | Token with scopes beyond issues:write minimum SHOULD use a dedicated per-type token (RL-055) |
| P12 | `PartialSuccessSetLabelsFailed` | Partial-success (add missing / remove still present) maps to `rejected` + single SETLABELS_FAILED code, no new error code (RL-057/058/059) |
| P13 | `CrossRepoReachabilityAndScope` | Cross-repo write requires both target reachability and adequate token scope; either failing blocks the write (RL-050/RL-051) |

## Key Invariants

- Blocklist evaluation precedes allowlist evaluation (security boundary) — RL-003
- label_to_remove absence on item is not an error; operation proceeds add-only (RL-034, idempotency goal)
- Staged mode MUST NOT invoke any write API call, even though gate-check reads may proceed (RL-027/RL-056)
- Partial-success (HTTP 200 but returned labels don't match expected) is treated as `rejected`, same SETLABELS_FAILED error code, no new error code introduced (RL-057/RL-058/RL-059)
- Single REST call (`PUT .../labels`) replaces the whole label set atomically — no separate add/remove round trip (RL-036)
- Allow/blocklist policy MUST be re-checked at the moment of write, not merely at an earlier gate — a policy change between check and write time must block the write (RL-049a)
- required-labels gate is only spec-compliant when its label set comes from a fresh server-side fetch — agent-supplied label claims are untrusted input for gating purposes (RL-053)
- Cross-repo writes require conjunctive satisfaction of reachability (RL-050) AND token scope (RL-051); neither condition alone is sufficient

## Edge Cases Identified

- label_to_add == label_to_remove: final set must contain exactly one occurrence, not zero
- label_to_add already present before replace: dedup must not produce two entries
- Rate limit exhausted after max retries: must hard-fail, not silently retry indefinitely
- Non-rate-limit HTTP errors (422, 500) must NOT enter the retry loop at all
- Agent-supplied "bypass" flags on the message must be structurally ignorable — validated only server-side
- Earlier gate check passes but policy changes before the actual write call (race/TOCTOU-style) — must still block (RL-049a)
- required-labels gate evaluated against agent-claimed labels happens to pass but is non-compliant regardless of outcome (RL-053)
- Token carrying default/broad workflow scopes (e.g. contents:write, actions:write) alongside issues:write — flagged as over-privileged even though it satisfies the MUST-have minimum (RL-055)

## Notes for Future Runs

- Core predicates (schema validation RL-004..009, count gate RL-010..012, target resolution RL-013..020,
  label validation ordering RL-021..023, gate checks RL-024..026, staged mode RL-027..028, label-set
  computation RL-029/033/034, cross-repo RL-014/015/050, transitions) are already covered in
  `pkg/workflow/replace_label_formal_test.go` and `replace_label_transitions_formal_test.go` — do not
  duplicate these in future runs on this spec.
- The 2026-08-07 run's file `pkg/workflow/replace_label_security_formal_test.go` fills the §7 (error
  handling) and §8 (security considerations) gap: sanitization, rate-limit retry, REST hard-error
  semantics, label pre-existence enforcement, server-side-only allow/blocklist enforcement, and token
  scope minimum.
- This run's new file `pkg/workflow/replace_label_governance_formal_test.go` closes the remaining
  lightly-formalized areas flagged previously: RL-049/RL-049a (pre-write allow/blocklist recheck),
  RL-053 (required-labels gate server-state requirement), RL-055 (dedicated per-type token
  recommendation), a re-verification of RL-057/058/059 (partial-success → SETLABELS_FAILED mapping),
  and the RL-050/RL-051 cross-repo conjunctive-gate interaction — do not duplicate these predicates in
  future runs on this spec.
- Sandbox could not run `go build`/`go test` in either run (repo requires a newer Go toolchain than
  available in sandbox / network policy blocks toolchain download) — validated via visual review only.
  Recommend CI verification of the new file.
- Remaining areas for a potential future pass: none of the currently normative RL-xxx requirements
  appear unformalized at this point; a future run could instead focus on cross-cutting consistency
  between `replace-label` and the sibling `add-labels`/`remove-labels` safe-output types (e.g. shared
  sanitization/rate-limit code paths), or on stronger temporal-logic (TLA+) modeling of the Stage
  5→7→8 pipeline ordering guarantees.
