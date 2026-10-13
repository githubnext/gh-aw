# ADR-68081: Document Known Legacy Routing Discrepancies as Per-Case Allowances in Model-Routing Goldens

**Date**: 2026-10-12
**Status**: Draft
**Proposed deciders**: PR author (SivaKesava1), gh-aw maintainers

---

### Context

`pkg/cli/model_routing_golden_test.go` replays recorded run artifacts through the audit CLI in two variants — "downloaded" (full artifact set, including `usage/aw_session.jsonl` and `agent/aw_info.json`) and "legacy" (the reduced artifact set older runs produce) — and asserts that the *raw routing facts* (status, selected model/effort, endpoints, routed/outcome counts, deviations) agree between the two. The four new fixtures added for issue #68076 cover routed **failure** paths (budget exhaustion with steering, provider unavailable) and routed paths where some artifacts carry no `model_routing` block at all. For two of those fixtures the two variants genuinely disagree: `parse_token_usage.cjs` relabels a budget failure as `harness_selection_mismatch` only when the session file is present, and `applyAwInfoModelRouting` skips `agent/aw_info.json`, so the legacy variant loses `effective_endpoint`/`selected_endpoint`. These are real product discrepancies, not fixture defects, so the existing strict cross-variant equality assertion cannot hold as written.

### Decision

We will record each known downloaded-vs-legacy discrepancy as an explicit, per-case `legacyRoutingAllowance map[string]string` on `modelRoutingGoldenCase`, keyed by routing-identity field name with the reason as the value. During cross-variant comparison the test asserts that each allowed field **still differs**, fails with a "remove the allowance and its README entry" message if it no longer does, then zeroes that field before the remaining identity comparison stays strict. The projection is also widened with `failure`, `degraded_classification` and `degraded_reason`, and the failure code joins the cross-variant identity so routed-failure fixtures assert on the failure itself rather than only on status. The primary driver is that tolerating a discrepancy silently (or loosening the assertion globally) would let an unrelated regression slip through; a self-expiring, documented allowance keeps the guard tight and forces the fixture and README to be updated when the underlying bug is fixed.

### Alternatives Considered

These alternatives are inferred from the diff and PR description; they do not record maintainer approval or rejection.

#### Alternative 1: Fix the underlying discrepancies in `parse_token_usage.cjs` / `applyAwInfoModelRouting` in this PR

Making the legacy path read `agent/aw_info.json` and keeping the proxy failure status consistent would make the fixtures pass under the existing strict assertion with no new test machinery. It was not chosen because this PR's scope is capturing evidence (golden fixtures regenerated from real runs) ahead of behavioural change; fixing the audit behaviour in the same change would mean the goldens never record the current, shipped behaviour, and the fix would land without a regression baseline.

#### Alternative 2: Drop the divergent fields from the cross-variant identity for all cases

Removing `EffectiveEndpoint`, `SelectedEndpoint` and `Status` from `routingIdentity` is a one-line change that makes all cases pass. It was rejected because those fields are exactly the routing facts the golden test exists to protect; weakening them repo-wide to accommodate two fixtures would disable the check for the other cases and leave no record of why.

#### Alternative 3: Record the divergence only in `expected.legacy.json` and the README

The per-variant expected files already capture the differing values, so the discrepancy is visible in review. This was insufficient on its own: the cross-variant assertion would still fail, and a prose-only note never expires — once the product bug is fixed nothing would prompt removal of the stale documentation.

### Consequences

#### Positive
- Routed failure paths and `model_routing`-less artifact layouts now have golden coverage, including the failure code, degraded-classification flag and degraded reason.
- Each tolerated discrepancy is machine-checked and self-expiring: fixing the product bug turns the test red until the allowance and its README row are deleted, so documentation cannot drift from behaviour.
- The remaining routing-identity comparison stays strict for every other field and every other case; the escape hatch is scoped to named fields on named cases.

#### Negative
- The allowance mechanism uses `reflect.FieldByName` and zeroes fields on a local copy, which is more machinery (and more failure modes, e.g. a typo'd field name) than a plain `reflect.DeepEqual`.
- Known product bugs are now encoded as "expected" in test data; a reader who skips the allowance reason may mistake the legacy output (missing endpoints, divergent status) for intended behaviour.
- Fixture volume grows substantially (≈3k added lines of recorded artifacts and expected outputs), increasing repository size and the cost of regenerating goldens.

#### Neutral
- `make fmt` (prettier) reformats the new fixtures' `aw_info.json`; the change is whitespace-only but required by `fmt-check-json`, so recorded artifacts are not byte-identical to the downloaded originals.
- `codex-routed-provider-unavailable` projects identically in both variants, so it carries no `expected.legacy.json` and no `legacyDifference`.
- The README gains rows for the four fixtures plus a "known discrepancies" section, making it the human-facing counterpart to the in-code allowances.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
