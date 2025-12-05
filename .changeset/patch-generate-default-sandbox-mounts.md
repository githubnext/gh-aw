---
"gh-aw": patch
---

Automatically generate default `sandbox.agent.mounts` entries from `tools.bash` entries.

When `sandbox.agent.mounts` is not explicitly configured, the runtime will now
derive reasonable default container mounts from `tools.bash` patterns (wildcards
and individual tool patterns). Explicit `sandbox.agent.mounts` in frontmatter
still take precedence.

This is an internal/tooling enhancement and does not change public CLI APIs.

