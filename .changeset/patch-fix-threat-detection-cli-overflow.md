---
"gh-aw": patch
---

Fix threat detection CLI overflow by using file access instead of inlining agent output

The threat detection job was passing the entire agent output to the detection agent via environment variables, which could cause CLI argument overflow errors when the agent output was large. Modified the threat detection system to use a file-based approach where the agent reads the output file directly using bash tools (cat, head, tail, wc, grep, ls, jq) instead of inlining the full content into the prompt.
