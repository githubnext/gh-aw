# ADR-47471: Unify Permission Handling for Custom and Safe Jobs via Shared Parser

**Date**: 2026-07-23
**Status**: Draft
**Deciders**: Unknown (automated draft — review before accepting)

---

### Context

The gh-aw workflow compiler processes YAML frontmatter that specifies GitHub Actions job configurations, including a `permissions` field. The GitHub Actions schema accepts permissions in two forms: an object form (e.g., `{contents: write, pull-requests: read}`) and shorthand strings (e.g., `read-all`, `write-all`, `none`). A shared `PermissionsParser` abstraction (`NewPermissionsParserFromValue`) already handles both forms elsewhere in the compiler. However, the permissions handling for custom jobs (`extractCustomJobCoreProperties`) and safe jobs (`buildSafeJobs`) contained separate code paths that only handled the object form via a type assertion (`map[string]any`). Shorthand string values were silently dropped, causing compiled workflows to omit the specified permissions entirely.

### Decision

We decided to route all permission values for custom jobs and safe jobs through the existing shared `NewPermissionsParserFromValue` parser instead of maintaining inline type-assertion branches. For safe jobs, we added a `RawPermissions any` field to `SafeJobConfig` to preserve the original untyped YAML value at parse time, and then render it through the shared parser at build time. The primary driver is correctness: the shared parser already handles both permission forms, so reusing it eliminates the silent-drop bug with no new abstraction required.

### Alternatives Considered

#### Alternative 1: Extend the Inline Type-Assertion Branches

Add an explicit `string` case alongside the existing `map[string]any` case in both `extractCustomJobCoreProperties` and `parseSafeJobsConfig`, handling shorthand strings directly in the same location. This keeps each function self-contained but duplicates the permission-format awareness that already lives in `PermissionsParser`, and would require the same change in every future code path that reads permissions.

#### Alternative 2: Pre-normalize Permissions to Object Form at Parse Time

Convert shorthand strings to their equivalent expanded object form (e.g., `read-all` → `{contents: read, pull-requests: read, …}`) during YAML parsing before the values reach the compiler. This would make the compiler's internal representation uniform, but requires enumerating all valid GitHub Actions permission scopes and maintaining that list, and changes the semantics of the rendered YAML (shorthands would no longer round-trip as shorthands).

### Consequences

#### Positive
- Shorthand permission values (`read-all`, `write-all`, `none`) are now preserved for both custom and safe jobs in compiled workflow output.
- A single parser handles all permission forms, so future shorthand additions or edge cases only need to be addressed in one place.
- Integration tests cover both permission forms for both job types, providing regression protection.

#### Negative
- `SafeJobConfig` now carries two overlapping fields: the typed `Permissions map[string]string` (populated for object-form permissions during YAML decode) and `RawPermissions any` (populated for all permissions). The `Permissions` map is still consulted as a fallback, creating a dual-path render that may confuse future maintainers.
- The `Permissions` map field on `SafeJobConfig` is partially redundant — it is only used when `RawPermissions` is nil — but removing it would break callers that may read it directly.

#### Neutral
- The fix changes which code path renders permissions for custom jobs: previously `formatIndentedYAMLField`, now `PermissionsParser.RenderToYAML`. Rendered YAML format may differ slightly (e.g., key ordering) for object-form permissions, though semantics are equivalent.
- Integration tests use the `//go:build integration` tag and do not run in the default test suite.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
