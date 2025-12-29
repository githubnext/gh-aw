---
"gh-aw": patch
---

Refactor safe outputs into a centralized handler manager using a
factory pattern. Introduces a safe output handler manager and begins
refactoring individual handlers to the factory-based interface. This is
an internal refactor (WIP) that reorganizes handler initialization and
message dispatching; tests and workflow recompilation are still pending.

