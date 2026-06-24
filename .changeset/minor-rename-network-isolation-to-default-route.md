---
"gh-aw": minor
---

Rename the frontmatter field `sandbox.agent.network-isolation` to `sandbox.agent.default-route` with inverted semantics.

The new `default-route` field controls whether AWF uses the default network route:
- `default-route: false` — disables the default route, enabling AWF network-isolation topology egress mode (`--network-isolation`). Equivalent to the old `network-isolation: true`.
- `default-route: true` or omitted — keeps the default route, normal mode. Equivalent to the old `network-isolation: false`.

This change is frontmatter-only; the underlying AWF `--network-isolation` flag and runtime behavior are unchanged.

**Migration:** Replace `sandbox.agent.network-isolation: true` with `sandbox.agent.default-route: false` in your workflow frontmatter.
