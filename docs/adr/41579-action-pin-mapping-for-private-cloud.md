# ADR-41579: Action-Pin Mapping for Private-Cloud Enterprises via aw.json

**Date**: 2026-06-26
**Status**: Draft
**Deciders**: Unknown

---

### Context

Enterprises operating GitHub Actions runners in private clouds cannot access public GitHub-hosted actions directly. When the `aw.json` compiler resolves action pins (e.g., `actions/checkout@v4`), it queries public pin tables and hardcoded SHA maps that are unreachable in air-gapped or internally mirrored environments. Without a way to redirect action references at the compiler layer, such enterprises must either patch action references directly in every workflow file or forgo pin enforcement entirely. The project's `aw.json` configuration file already serves as the canonical per-repository compiler configuration, making it the natural place to add a redirect table.

### Decision

We will add an `action_pins` map to `aw.json` that redirects action `owner/repo@ref` references to replacement `owner/repo@ref` references before pin resolution occurs. The mapping is applied as the first step of `ResolveActionPin`, so the rest of the resolution pipeline (dynamic lookup, hardcoded pins, SHA validation) operates on the mapped target. Schema validation enforces the `owner/repo@ref` format on both keys and values. The feature is intentionally not supported in workflow frontmatter to keep the configuration scoped to repository-level infrastructure concerns in `aw.json`.

### Alternatives Considered

#### Alternative 1: Frontmatter-level action overrides in individual workflow files

Each workflow could declare action redirects in its own frontmatter. This is rejected because it requires updating every workflow file to add redirect entries, does not provide a single authoritative source of mappings, and conflates workflow authoring with infrastructure configuration. The PR body explicitly notes this alternative is not supported.

#### Alternative 2: Proxy/registry endpoint configuration

A configurable registry mirror URL could intercept all action resolution at the network layer. This would require additional network infrastructure (a proxy service or registry mirror), configuration of TLS certificates and routing, and significant complexity both in the compiler and in customer environments. The `action_pins` map approach is self-contained in `aw.json`, requires no infrastructure changes, and lets enterprises selectively redirect only the specific actions they have mirrored.

#### Alternative 3: Wildcard or prefix-based matching

Mappings could support wildcards (e.g., `actions/*` → `acme-corp/*`). This was implicitly not chosen: the implementation uses exact `owner/repo@ref` key lookup via `FormatCacheKey`. Exact matching makes the mapping table auditable, avoids ambiguity when multiple patterns could match, and prevents accidental redirects. Each version must be mapped explicitly.

### Consequences

#### Positive
- Enterprises in private clouds can pin and resolve mirrored actions without modifying workflow YAML files.
- JSON schema validation (`propertyNames` + `additionalProperties` pattern) rejects malformed keys/values at configuration load time, catching mistakes early.
- Deduplication of console output (`ctx.Warnings["map:…"]`) ensures users see each active mapping exactly once per compiler invocation, keeping output readable.
- The mapping table is nil-safe and loaded lazily via `loadRepoConfig`; zero overhead when `action_pins` is absent.

#### Negative
- Exact `owner/repo@ref` key semantics mean each major version of each action must be mapped individually; there is no wildcard support. Large mirror sets require verbose configuration.
- The mapped target must itself be resolvable by the existing pin machinery (dynamic lookup or hardcoded pins). If the mirror repo is not in the embedded pin table and dynamic resolution is unavailable, resolution fails.
- `getActionPinMappings` calls `loadRepoConfig` independently from other callers; if `aw.json` is read multiple times per compilation, there is duplicated I/O (though this is consistent with the existing pattern for other `aw.json` fields).

#### Neutral
- The feature is only configurable in `aw.json`, not workflow frontmatter, which is a conscious scope boundary.
- Mapping application is logged at `Printf` level (not a user-visible warning), so diagnosing missed mappings requires debug log inspection.
- The implementation threads `ActionPinMappings` through `WorkflowData` and `PinContext`, following the existing propagation pattern for `ActionPinWarnings` and `AllowActionRefs`.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
