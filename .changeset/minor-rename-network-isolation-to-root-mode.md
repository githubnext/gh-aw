---
"gh-aw": minor
---

Rename the frontmatter field `sandbox.agent.network-isolation` to `sandbox.agent.root-mode` with inverted semantics.

The new `root-mode` field controls whether AWF runs in root mode:
- `root-mode: false` — disables root mode, enabling AWF network-isolation topology egress mode (`--network-isolation`). Equivalent to the old `network-isolation: true`.
- `root-mode: true` or omitted — keeps root mode enabled, normal mode. Equivalent to the old `network-isolation: false`.

This change is frontmatter-only; the underlying AWF `--network-isolation` flag and runtime behavior are unchanged.

**Migration:** Replace `sandbox.agent.network-isolation: true` with `sandbox.agent.root-mode: false` in your workflow frontmatter.
