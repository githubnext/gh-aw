# ADR-70749: Prefer AWF-Exported Token Usage Log for Evals Accounting

**Date**: 2026-10-21
**Status**: Draft
**Deciders**: pelikhan, Copilot (PR #70749)

---

### Context

The daily AI-credits budget relies on per-component accounting files, including `evals/token_usage.jsonl` produced by the compiled "Collect evals token usage" step. That step searched only fixed firewall directories (`<auditDir>/api-proxy-logs/token-usage.jsonl` and `<proxyLogsDir>/api-proxy-logs/token-usage.jsonl`). AWF (pinned v0.28.70) writes token usage to a per-run temporary path (for example `/tmp/api-proxy-logs-<random>/token-usage.jsonl`) and exports that path to downstream steps through `GITHUB_ENV` as `AWF_TOKEN_USAGE_LOG`. Because the collector never consulted this path, successful evals jobs uploaded no accounting. The scanner then counted each affected run at the conservative 1,000-AIC maximum. Eight of 13 historical runs in Smoke Copilot run 39522345151 were affected.

### Decision

We will make the AWF-exported `AWF_TOKEN_USAGE_LOG` environment variable the primary source for evals token-usage collection. If that variable names an existing file, the collector copies it even when it is empty. Only when the exported file is unavailable does the collector fall back to the legacy fixed directories, and that fallback still copies only non-empty files. A missing log never produces a zero-usage placeholder. This is the main reason for the change: AWF already publishes the exact runner-visible path, so consuming it removes guesswork about directory layout across default and ARC/DinD topologies.

### Alternatives Considered

#### Alternative 1: Extend the fixed-directory search (glob `/tmp/api-proxy-logs-*`)

Add more candidate directories or a glob for AWF's randomized temporary directory. This would work without depending on the environment variable. We rejected it because it couples the compiler to AWF's internal naming scheme, could match stale or unrelated logs on shared/self-hosted runners, and duplicates information AWF already exports explicitly.

#### Alternative 2: Configure AWF to write to a fixed, compiler-chosen path

Pass a deterministic log location to AWF so the old collector paths stay valid. This keeps the collector simple. However, it requires changing how AWF is invoked across topologies and depends on AWF honoring that override. The exported-variable contract already exists in the pinned version, so this alternative adds more change for the same result.

#### Alternative 3: Rely solely on `AWF_TOKEN_USAGE_LOG` with no fallback

Dropping the legacy directory search would simplify the script. We rejected it because older AWF versions or topologies that do not export the variable would regress to no accounting at all.

### Consequences

#### Positive
- Evals jobs record actual token usage instead of falling back to the conservative 1,000-AIC estimate, so daily-limit accounting becomes more accurate.
- Empty exported logs are kept, which keeps "the producer ran and logged nothing" separate from "no accounting was collected".
- The transport contract is documented in the AI-credits specification (`docs/src/content/docs/specs/ai-credits-specification.md`).

#### Negative
- The compiler now depends on an AWF-defined environment variable contract. If AWF renames or removes `AWF_TOKEN_USAGE_LOG`, the collector silently falls back to the legacy paths.
- An empty exported file now overrides a non-empty legacy-directory log. This is intended, but it could hide usage if AWF ever exports a path it did not write to.
- The change regenerates 168 workflow lock files, which makes the diff large and harder to review.

#### Neutral
- The fix only affects future runs. Missing historical files cannot be recovered, and unresolved estimates still expire through the existing rolling 24-hour window.
- Component coverage rules, conservative estimates, and daily limits do not change. Only the producer's choice of source file changes.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
