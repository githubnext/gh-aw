---
"gh-aw": patch
---

Implement the safe output handler manager which centralizes dispatch
of agent safe-output messages to dedicated JavaScript handlers. This
refactors multiple conditional workflow steps into a single
`Process Safe Outputs` step and adds configuration fallback logic for
handler loading. Includes tests and documentation updates.

