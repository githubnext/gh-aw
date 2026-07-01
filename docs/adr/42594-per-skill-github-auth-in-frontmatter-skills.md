# ADR-42594: Per-Skill GitHub Auth in Frontmatter `skills`

**Date**: 2026-07-01
**Status**: Draft
**Deciders**: Unknown

---

### Context

The gh-aw workflow compiler supports a `skills[]` frontmatter field that installs external skill repositories during activation. Previously each entry was a plain string reference (`owner/repo@<sha>`) and all installs shared a single activation token. Operators who need to install skills from multiple GitHub organizations — each with its own access controls — cannot grant a single token the required permissions across all orgs without over-privileging it. A mechanism to scope authentication per skill entry is therefore needed.

### Decision

We will extend the `skills[]` frontmatter array to accept object-form entries alongside existing string entries. An object entry carries a required `skill` string (same format as string entries) plus optional `github-token` or `github-app` fields. The compiler emits one install step per skill reference; for object entries it injects the per-skill token or mints a GitHub App token before the step rather than using the workflow-level activation token.

### Alternatives Considered

#### Alternative 1: Separate `skill-auth` mapping field

A top-level `skill-auth:` object keyed by skill ref string could hold per-skill credentials. This keeps the `skills[]` array uniform strings but requires the compiler to join two separate structures. It is more verbose in the frontmatter and makes it harder to add inline credential entries in a single place; rejected in favour of keeping auth co-located with the skill reference.

#### Alternative 2: Environment-level token override only

Allow a workflow-level `github-token` override that applies to all skill installs. This is simpler but still forces a single token across all org boundaries, failing the multi-org use case. Rejected because it does not solve the core problem.

### Consequences

#### Positive
- Skill installs from multiple organizations with separate access controls are now possible from a single workflow.
- Per-skill credential scoping limits the blast radius of a leaked or over-privileged token.
- Backward-compatible: existing string-only workflows are unaffected; the compiler promotes them to `SkillReference{Skill: s}` internally.

#### Negative
- Activation job now emits one GitHub Actions step per skill instead of a single batched step, making compiled lock files larger and activation logs more verbose.
- Frontmatter schema complexity increases: operators must understand `string | object` union items and the `github-app` sub-object shape.

#### Neutral
- The `FrontmatterConfig.Skills []string` field is broadened to `[]any` to accept mixed arrays; a new `SkillReferences []SkillReference` parallel field carries the typed representation used by the compiler.
- JSON schema for `skills[]` items gains a fourth `oneOf` variant (the object form), requiring schema tooling to regenerate.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
