# ADR-68951: Enforce explicit native editing policies through engine capabilities

**Date**: 2026-10-15
**Status**: Draft
**Deciders**: @pelikhan (PR #68951)

---

### Context

gh-aw lets a workflow declare `tools.edit` to control an engine's native file-editing tools, but enforcement was inconsistent across engines. Gemini's tool mapping granted `write_file`/`replace` even when `edit` was explicitly `false`, Copilot's wildcard permission handling could override an explicit denial, and the Copilot SDK tool catalog grouped process-input tools with editing tools so disabling editing could affect unrelated capabilities. Codex and Agy have no mechanism to disable their native editors at all, so an `edit: false` setting was silently ignored there. The result was that a security-relevant frontmatter setting produced different — and sometimes opposite — runtime behavior depending on the engine, with no compile-time signal to the author.

### Decision

We will model native editing as explicit engine capabilities (`NativeEdit` and `NativeEditDisable` on `AgenticEngineCapabilities`, surfaced declaratively as `engine.behaviors.capabilities.native-edit` / `native-edit-disable`) and validate explicit boolean `tools.edit` settings against them at compile time in `Compiler.validateNativeEditSupport`. An explicit `edit: true` requires `NativeEdit`; an explicit `edit: false` requires `NativeEditDisable`; otherwise compilation fails with an actionable error instead of silently dropping the policy. Validation runs before engine defaulting/fallback so a boolean policy cannot be discarded. Copilot, Claude, Gemini, and Pi declare both capabilities; Codex and Agy declare only `NativeEdit` and therefore reject `edit: false`. Gemini's editor mapping is fixed, Copilot emits an explicit write denial rather than relying on wildcards, and the Copilot SDK separates editor and shell/process catalogs. Custom (behavior-defined) engines opt in by declaring the capabilities and consuming the compiler-owned `GH_AW_TOOLS_EDIT` runtime environment variable. Omitted `edit` and the legacy `edit:` / `edit: {}` forms keep existing engine defaults, preserving compatibility for current workflows.

### Alternatives Considered

#### Alternative 1: Keep per-engine ad-hoc handling and just fix the bugs

We could have patched Gemini's mapping and Copilot's wildcard handling without introducing shared capabilities. This is the smallest diff and avoids a new cross-engine contract. It was rejected because it leaves no structural guarantee: the next engine (or the next refactor of an existing one) can reintroduce the same silent-ignore behavior, and Codex/Agy would still accept `edit: false` while doing nothing about it.

#### Alternative 2: Warn instead of failing compilation for unsupported booleans

Emitting a compiler warning when an engine cannot honor an explicit boolean would avoid breaking any workflow that currently sets `edit: false` on Codex or Agy. It was rejected because `edit: false` reads as a restriction, and a warning that is easy to miss in CI logs leaves the author believing a restriction is in force when it is not. Failing closed makes the gap visible at authoring time. (This is a genuinely close call, and the cost is recorded under Negative consequences.)

#### Alternative 3: Enforce editing restrictions at the filesystem/sandbox layer

Instead of per-engine tool policy, gh-aw could block writes in the sandbox so the policy holds regardless of engine support. It was rejected for this change because it is a much larger behavioral change that would also affect legitimately enabled shell and MCP tools, scoped memory, and sandbox scratch space. Native-editor policy and filesystem isolation are deliberately kept as separate concerns.

### Consequences

#### Positive
- Explicit `tools.edit` booleans now behave consistently across Copilot, Claude, Gemini, and Pi, and a policy that cannot be enforced is reported at compile time instead of being silently dropped.
- Disabling native editing no longer side-affects shell/process tools: the Copilot SDK catalogs are separated and native editing never implies shell execution.
- Third-party/custom engines have a documented, testable integration contract (`native-edit`, `native-edit-disable`, `GH_AW_TOOLS_EDIT`) instead of undefined behavior.

#### Negative
- Workflows that currently set `edit: false` with the Codex or Agy engines will fail to compile and must be updated, even though their runtime behavior was already unrestricted.
- Capability declarations for custom engines are trusted, not verified: gh-aw cannot prove a third-party harness actually applies `GH_AW_TOOLS_EDIT`, so the contract can be declared and not honored.
- Every engine implementation now carries additional capability state that must be kept accurate as upstream CLIs change, adding ongoing maintenance surface.

#### Neutral
- Omitted `edit` and the legacy `edit:` / `edit: {}` forms are untouched, so existing workflows and custom-engine integrations keep their current defaults.
- Disabling native editors is not a filesystem write barrier; separately enabled shell or MCP tools can still write, and Claude's scoped memory and sandbox scratch grants are unchanged.
- The change touches the workflow JSON schema, frontmatter/tools/third-party-agent reference docs, and one generated `.lock.yml`, so schema-freshness and recompile checks are part of the validation path. `tools.read` behavior is unchanged.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
