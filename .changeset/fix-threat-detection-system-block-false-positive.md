---
"gh-aw": patch
---

Fixed threat detection reporting a false-positive prompt injection for gh-aw's own `<system>` prompt scaffolding. The detection agent reads the analyzed workflow's prompt file, which starts with the framework-generated `<system>` block (immutable security policy, safe-output tool instructions), and attributes that block to the agent output. Threat detection setup now mechanically removes that leading `<system>` block from the analyzed prompt file before analysis; `<system>` markup appearing later in the file is left intact so attacker-supplied lookalike blocks are still analyzed.
