---
"gh-aw": patch
---

Remove the deprecated Anthropic Sandbox Runtime (SRT) backend, keep AWF as the supported sandbox, and migrate any `sandbox.agent: srt` usages to `sandbox.agent: awf` automatically.
