# ADR-56353: Always Configure OTLP from Enterprise Default Variables and Secrets

**Date**: 2026-08-27
**Status**: Draft
**Deciders**: Unknown (automated draft from PR #56353 — review and finalize)

---

### Context

Agentic workflows in gh-aw previously required per-workflow opt-in for OTLP telemetry via an `observability.otlp` frontmatter block (typically pulled in through a shared import). Enterprises wanting to enable telemetry for all workflows had to modify every workflow individually, creating adoption friction and making centralized observability governance impractical. The compiler already has a `compilerenv` default-env infrastructure for injecting org/enterprise-level values; OTLP configuration was not wired into it. GitHub Actions makes org- and enterprise-level variables and secrets available as `vars.*` / `secrets.*` expressions, which are the appropriate mechanism for centralized defaults.

### Decision

We will make the compiler always emit OTLP environment variables (`OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_EXPORTER_OTLP_HEADERS`, `GH_AW_OTLP_ENDPOINTS`, and `GH_AW_OTLP_IF_MISSING`) in compiled workflow YAML, drawing from two new org/enterprise-level defaults — `GH_AW_DEFAULT_OTLP_ENDPOINT` (variable) and `GH_AW_DEFAULT_OTLP_HEADERS` (secret). Existing `observability.otlp` frontmatter wins on precedence so already-configured workflows are unaffected. When neither is set, `GH_AW_OTLP_IF_MISSING: ignore` causes the runtime to silently drop endpoints whose URL is empty, making the default path a no-op. A runtime bash guard (`check_otlp_default_credentials.sh`) fails the job when an endpoint is configured without accompanying headers.

### Alternatives Considered

#### Alternative 1: Continue requiring per-workflow opt-in via frontmatter

Each workflow that needs telemetry explicitly includes `observability.otlp` frontmatter or imports a shared snippet that does so. This was rejected because it does not solve the centralized governance problem — an enterprise administrator cannot enable telemetry for all workflows from one place, and new workflows start without telemetry by default unless contributors remember to add the frontmatter.

#### Alternative 2: Org-level variable read by the existing opt-in path (no compiler change)

Introduce an org-level variable that the existing `observability.otlp` resolution checks at runtime, without changing the compiler to emit OTLP env vars by default. This was rejected because it still requires an active code path in the frontmatter resolution — workflows with no `observability.otlp` frontmatter at all would not check the variable. A compiler-level default is the only way to guarantee every compiled workflow has the env vars present.

### Consequences

#### Positive
- Enterprises can configure OTLP once at the org/enterprise level and all agentic workflows inherit it automatically with no per-workflow changes.
- Workflows without explicit OTLP frontmatter do not break when defaults are absent — `GH_AW_OTLP_IF_MISSING: ignore` makes the empty-URL endpoint a silent no-op at runtime.
- Existing workflows with explicit `observability.otlp` frontmatter are unaffected; their configuration takes precedence and the new env vars are simply overwritten.

#### Negative
- Every compiled workflow now declares `GH_AW_DEFAULT_OTLP_HEADERS` in its manifest as a referenced secret, even when OTLP is not actually used, increasing manifest noise and slightly expanding the secret-reference surface area.
- The runtime credential guard fails hard (job error) when an endpoint is set without headers, which may be too strict for enterprises running unauthenticated internal collectors; this strictness is scoped only to the enterprise-default path, not to explicit frontmatter-configured endpoints.

#### Neutral
- The collector domain is not known at compile time (it is an expression-valued variable), so no firewall allowlist entry is added automatically; enterprises must add their collector to `network.allowed` themselves, which is documented but requires a manual step.
- `GH_AW_DEFAULT_OTLP_HEADERS` is registered as compiler-internal in `safe_update_enforcement.go` so that safe-update does not flag every recompile as introducing a new secret reference.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
