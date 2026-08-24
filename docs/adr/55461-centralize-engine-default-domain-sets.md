# ADR-55461: Centralize Engine Default Domain Sets

**Date**: 2026-08-24
**Status**: Draft
**Deciders**: Unknown

---

### Context

Engine-specific unconditional domain allow-lists were scattered across multiple separate static Go variables (`CopilotDefaultDomains`, `CodexDefaultDomains`, `ClaudeDefaultDomains`, `GeminiDefaultDomains`, `PiDefaultDomains`, `PiBaseDefaultDomains`) and an `ecosystem_domains.json` file that also served as the user-configurable ecosystem registry. This made it impossible to enumerate or inspect the full set of engine defaults in one place, and it conflated engine-internal allow-lists (automatically injected by the compiler) with user-selectable ecosystem identifiers (explicitly opted into via `network.allowed`). Specifically, the threat-detection allow-list being stored in the ecosystem JSON meant users could theoretically select it as a `network.allowed` entry, which was never the intended behavior.

### Decision

We will centralize all engine-specific unconditional domain allow-lists into a single unexported package-level map (`engineDefaultDomainSets`) and expose a copy-returning public accessor (`GetEngineDefaultDomainSets()`) for analysis and reporting. Existing exported compatibility variables (`CopilotDefaultDomains`, etc.) will derive their values from this registry at initialization time via a `copyEngineDefaultDomainSet` helper. The threat-detection allow-list will be removed from `ecosystem_domains.json` and moved exclusively into the new registry, preventing it from being user-selectable.

### Alternatives Considered

#### Alternative 1: Keep separate static variables, add an aggregation function

Maintain each engine's domain list as its own `var` but introduce a function that aggregates them into a map for reporting. This avoids any initialization-time dependency between the registry and the compatibility variables, but leaves the domain lists distributed across the file. It does not solve the fundamental issue of drift between copies and fails to provide the single source of truth needed for the compiler to automatically reference them.

#### Alternative 2: Store all engine domain lists in a JSON/YAML configuration file

Move all engine domain lists to a structured data file (similar to `ecosystem_domains.json`) and load them at runtime. This would make the lists editable without recompilation and easily inspectable, but introduces parse-at-startup overhead, a potential initialization failure path, and blurs the distinction between code-managed engine defaults and data-managed ecosystem lists. The type safety and inline documentation benefits of Go code are also lost.

### Consequences

#### Positive
- Single authoritative source (`engineDefaultDomainSets`) for all engine allow-lists prevents content drift between the registry and the exported compatibility variables.
- `GetEngineDefaultDomainSets()` enables programmatic enumeration of all engine domain sets for analysis, reporting, and the new documentation tables in both network reference files.
- Threat-detection domains are correctly separated from user-selectable ecosystem identifiers, eliminating the path for users to select `threat-detection` as a `network.allowed` value.
- Immutability is enforced: `GetEngineDefaultDomainSets()` returns deep copies, and exported variables are initialized from copies, so external callers cannot corrupt the registry.

#### Negative
- `engineDefaultDomainSets` is a mutable package-level variable (not a constant), so code in the same package could modify it at runtime; tests must guard against this.
- Exported compatibility variables (`CopilotDefaultDomains`, etc.) now hold a snapshot from package initialization. Code that modifies these variables directly (e.g., in tests) will not affect what `GetEngineDefaultDomainSets()` returns, creating a subtle two-source-of-truth scenario within the package.
- Any future engine whose domain list needs dynamic construction cannot be expressed as a plain `[]string` literal in the map and will require refactoring the registry structure.

#### Neutral
- The PR adds `GetEngineDefaultDomainSets()` as a new public API surface. Future callers may depend on it, so the set of keys and the copy semantics become a stability commitment.
- Documentation tables for engine domain sets are now automatically derivable from the registry, but the two network reference files (`docs/src/content/docs/reference/network.md` and `.github/aw/network.md`) are still updated manually — there is no automated sync between the registry and the docs.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
