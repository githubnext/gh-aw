---
"gh-aw": minor
---

Rename the frontmatter field `sandbox.agent.network-isolation` to `sandbox.agent.sudo` with inverted semantics.

The new `sudo` field controls whether AWF runs with sudo:
- `sudo: false` — disables sudo, enabling AWF network-isolation topology egress mode (`--network-isolation`). Equivalent to the old `network-isolation: true`.
- `sudo: true` or omitted — keeps sudo enabled, normal mode. Equivalent to the old `network-isolation: false`.

This change is frontmatter-only; the underlying AWF `--network-isolation` flag and runtime behavior are unchanged.

**Migration:** Replace `sandbox.agent.network-isolation: true` with `sandbox.agent.sudo: false` in your workflow frontmatter.
