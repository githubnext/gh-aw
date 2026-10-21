# ADR-70749: Prefer AWF-Exported Token Usage Log for Evals Accounting

**Date**: 2026-10-21
**Status**: Accepted
**Deciders**: pelikhan, Copilot (PR #70749)

---

## Context

The daily AI-credits budget relies on per-component accounting files, including `evals/token_usage.jsonl` produced by the compiled "Collect evals token usage" step. That step searched only fixed firewall directories (`<auditDir>/api-proxy-logs/token-usage.jsonl` and `<proxyLogsDir>/api-proxy-logs/token-usage.jsonl`). AWF (pinned v0.28.70) writes token usage to a per-run temporary path and exports that runner-visible path to downstream steps through `GITHUB_ENV` as `AWF_TOKEN_USAGE_LOG`. For example, [run 39510857197](https://github.com/github/gh-aw/actions/runs/39510857197) exported `/tmp/api-proxy-logs-asbSi6/token-usage.jsonl`, but the collector ignored the variable.

Of 13 historical runs examined in [Smoke Copilot run 39522345151](https://github.com/github/gh-aw/actions/runs/39522345151), five had measured usage totaling 112.612865 AIC. Eight had missing evals token logs despite successful evals and received conservative 1,000-AIC estimates each. Successful execution alone does not establish complete accounting.

## Decision

Use the AWF-exported `AWF_TOKEN_USAGE_LOG` environment variable as the primary source for evals token-usage collection. If that variable names an existing file, the collector copies it even when it is empty. Only when the exported file is unavailable does the collector fall back to the legacy fixed directories, and that fallback still copies only non-empty files. A missing log never produces a zero-usage placeholder. AWF already publishes the exact runner-visible path, so consuming it removes guesswork about directory layout across default and ARC/DinD topologies.

An empty exported file takes precedence over a non-empty legacy log, preventing stale fallback data from being substituted. Preserving an empty file is not proof of zero usage: downstream component coverage and zero-accounting evidence rules still apply.

## Alternatives Considered

### Extend the fixed-directory search or glob `/tmp/api-proxy-logs-*`

Additional candidate paths would retain dependence on AWF's internal directory layout. A glob could also select stale or unrelated logs on shared/self-hosted runners. Neither is preferable to the exact exported path.

### Require a fixed, compiler-chosen AWF log path

A deterministic producer path would preserve the old collector model, but would require a supported override and changes to AWF invocation across topologies. This change does not establish such override support; it uses the existing exported-variable contract instead.

### Rely solely on `AWF_TOKEN_USAGE_LOG` with no fallback

Removing the legacy search would simplify the collector, but would lose existing accounting when the export is unavailable and a non-empty legacy log is present. Retaining that fallback limits the compatibility impact.

## Consequences

Future evals jobs can upload the producer's actual usage log, reducing estimates caused by this collection defect without weakening conservative accounting. The transport and empty-file semantics are documented in the [AI-credits specification](../src/content/docs/specs/ai-credits-specification.md).

The compiler depends on AWF's exported-variable contract. If it changes, legacy lookup remains available, but missing or unusable accounting still stays unresolved. An empty exported log intentionally suppresses legacy data even if the export is incomplete; it cannot independently authorize zero usage. Generated workflow lock files must be refreshed, increasing review noise.

The fix is prospective only: historical missing usage remains conservative and prior results are not rewritten. Daily limits, component coverage, estimate retry behavior, and rolling 24-hour expiry remain unchanged.
