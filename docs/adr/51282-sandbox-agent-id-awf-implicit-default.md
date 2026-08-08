# ADR-51282: Make `sandbox.agent.id: awf` the Implicit Default

**Date**: 2026-08-08
**Status**: Draft
**Deciders**: Unknown

---

### Context

The `gh-aw` platform supports agentic workflows via the Agent Workflow Firewall (AWF). The `sandbox.agent` YAML block accepts an `id` field that names the engine to use. In practice, `awf` has always been the only supported engine, and the runtime has always defaulted to it when `id` is omitted. Despite this, documentation and examples instructed authors to write `id: awf` explicitly — including a claim that strict mode *required* it. Over time, approximately 120 workflow files, Markdown samples, and documentation snippets accumulated this redundant declaration. The explicit field carries no information, creates maintenance overhead, and misleads authors into thinking the field is meaningful.

### Decision

We will make `awf` the officially documented implicit default for `sandbox.agent`, remove the incorrect strict-mode requirement for explicit `id: awf`, introduce a `sandbox-agent-id-awf-removal` codemod so existing workflows can be cleaned up via `gh aw fix`, and strip `id: awf` from all in-tree workflow samples, documentation, and the network-firewall migration emitter.

### Alternatives Considered

#### Alternative 1: Keep `id: awf` as Required in All Workflows

Continue requiring authors to write `id: awf` explicitly in every `sandbox.agent` block. This preserves backwards compatibility and keeps every workflow self-documenting about its chosen engine.

Rejected because: the declaration is redundant — `awf` is the only engine, the runtime always defaulted to it, and maintaining ~120+ instances of boilerplate increases noise without adding clarity. It also perpetuates a false implication that other engines are selectable.

#### Alternative 2: Introduce a Second Engine to Justify Explicit `id` Configuration

Add a second supported engine so that the `id` field becomes meaningful and the explicit declaration carries real information.

Rejected because: introducing a second engine is a separate and much larger architectural decision. Deferring cleanup of the current redundancy until another engine exists would leave the codebase in a misleading state indefinitely. If a second engine is added in the future, it can reintroduce explicit `id` requirements at that time.

### Consequences

#### Positive
- Eliminates ~120+ lines of redundant `id: awf` boilerplate from workflow files and documentation
- Documentation now accurately reflects that `id` can be omitted, reducing author confusion
- The new `sandbox-agent-id-awf-removal` codemod enables teams to migrate existing external workflows with a single `gh aw fix` invocation
- Corrects the incorrect strict-mode documentation that claimed `id: awf` was required

#### Negative
- External teams and repositories that copied the `id: awf` pattern now carry stale configuration; they must run the codemod or update manually
- Removes an explicit signal from workflow files about which engine is in use, which may reduce immediate readability for newcomers who don't know the default

#### Neutral
- The `sandbox.agent: false` escape hatch and the `dangerously-disable-sandbox-agent` flag are unaffected by this change
- The `bounded-queries` feature that previously documented `sandbox.agent.id: awf` in its example block now correctly references `sandbox.agent` without the `id` field

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
