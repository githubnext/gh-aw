---
"gh-aw": patch
---

Add a `mentions` configuration to `safe-outputs` to control how `@mentions`
are filtered in AI-generated content. The option supports both boolean and
object forms for fine-grained control (e.g., `allow-team-members`,
`allow-context`, an explicit `allowed` list, and a `max` per-message limit).

This change is non-breaking and preserves existing behavior when the setting
is unspecified.

