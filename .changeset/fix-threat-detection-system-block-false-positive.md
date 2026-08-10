---
"gh-aw": patch
---

Fixed threat detection reporting a false-positive prompt injection for gh-aw's own `<system>` prompt scaffolding. The detection agent reads the analyzed workflow's prompt file, which starts with the framework-generated `<system>` block (immutable security policy, safe-output tool instructions), and attributed that block to the agent output. The detection prompt now marks the framework scaffolding as trusted and requires prompt-injection findings to point at untrusted external content present in the agent output, comment-memory files, or patch.
