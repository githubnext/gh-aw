# ADR-55461: Centralize Engine Default Domain Sets

**Date**: 2026-08-24
**Status**: Accepted (automatic engine domain injection superseded by explicit `network.allowed` opt-in)
**Deciders**: Unknown

---

### Context

Engine-specific unconditional domain allow-lists were scattered across multiple separate static Go variables (`CopilotDefaultDomains`, `CodexDefaultDomains`, `ClaudeDefaultDomains`, `GeminiDefaultDomains`, `PiDefaultDomains`, `PiBaseDefaultDomains`) and an `ecosystem_domains.json` file that also served as the user-configurable ecosystem registry. This made it impossible to enumerate or inspect the full set of engine defaults in one place, and it conflated engine-internal allow-lists (automatically injected by the compiler) with user-selectable ecosystem identifiers (explicitly opted into via `network.allowed`). Specifically, the threat-detection allow-list was only represented as an ecosystem entry, obscuring its role as an internal Copilot detection default.

### Decision

We will centralize all domain allow-list data in the embedded `ecosystem_domains.json` file and load it once with `sync.OnceValues`. Engine-specific unconditional domain allow-lists are exposed in Go through an unexported package-level map (`engineDefaultDomainSets`) and a copy-returning public accessor (`GetEngineDefaultDomainSets()`) for analysis and reporting. Existing exported compatibility variables (`CopilotDefaultDomains`, etc.) derive their values from this registry at initialization time via a `copyEngineDefaultDomainSet` helper. The threat-detection allow-list will live in the engine-default registry while the legacy `network.allowed: [threat-detection]` ecosystem alias is retained as a compatibility path.

Update: engine domain sets remain centralized here, but normal agent runs no longer receive the selected engine's domain set automatically. Workflows must opt in with the matching `network.allowed` identifier (for example, `copilot`, `claude`, `codex`, `gemini`, or `pi`) when direct agent egress to those hosts is required.

### Alternatives Considered

#### Alternative 1: Keep separate static variables, add an aggregation function

Maintain each engine's domain list as its own `var` but introduce a function that aggregates them into a map for reporting. This avoids any initialization-time dependency between the registry and the compatibility variables, but leaves the domain lists distributed across the file. It does not solve the fundamental issue of drift between copies and fails to provide the single source of truth needed for the compiler to automatically reference them.

#### Alternative 2: Keep engine domain lists in Go code

Keep the engine domain lists as Go literals and only aggregate them in code. This preserves inline comments next to the data and avoids JSON parsing at package initialization time, but leaves the maintainer workflow split between code and data files and makes it harder to inspect all domain sets together.

### Consequences

#### Positive
- Single authoritative embedded data source for all ecosystem and engine allow-lists prevents content drift between the registry and the exported compatibility variables.
- `GetEngineDefaultDomainSets()` enables programmatic enumeration of all engine domain sets for analysis, reporting, and the new documentation tables in both network reference files.
- Threat-detection domains are represented as an engine-default set while preserving the legacy `network.allowed: [threat-detection]` compatibility alias.
- Immutability is enforced: `GetEngineDefaultDomainSets()` returns deep copies, and exported variables are initialized from copies, so external callers cannot corrupt the registry.

#### Negative
- `engineDefaultDomainSets` is a mutable package-level variable (not a constant), so code in the same package could modify it at runtime; tests must guard against this.
- Exported compatibility variables (`CopilotDefaultDomains`, etc.) now hold a snapshot from package initialization. Code that modifies these variables directly (e.g., in tests) will not affect what `GetEngineDefaultDomainSets()` returns, creating a subtle two-source-of-truth scenario within the package.
- Any future engine whose domain list needs dynamic construction cannot be fully expressed as a plain JSON array and will require refactoring the registry structure.

#### Neutral
- The PR adds `GetEngineDefaultDomainSets()` as a new public API surface. Future callers may depend on it, so the set of keys and the copy semantics become a stability commitment.
- Documentation tables for engine domain sets are now automatically derivable from the registry, but the two network reference files (`docs/src/content/docs/reference/network.md` and `.github/aw/network.md`) are still updated manually — there is no automated sync between the registry and the docs.

---

*ADR finalized after implementation and compatibility review.*
