---
"gh-aw": patch
---

Report GitHub Actions step-level timeouts as timeouts instead of "engine terminated unexpectedly". A step killed by `timeout-minutes` terminates the engine externally and can leave no timeout signature in the agent log, so the failure was previously misdiagnosed as an engine crash. The compiled "Detect agent errors" step now receives the engine step outcome and its `timeout-minutes` budget, and `detect_agent_errors.cjs` classifies a failed engine step that ran for its full budget as `agentic_engine_timeout`.
