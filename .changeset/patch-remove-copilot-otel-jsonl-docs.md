---
"gh-aw": patch
---

Remove stale documentation references to the `copilot-otel.jsonl` artifact, which was removed from the compiler and runtime in PR #32280. Docs, specs, and skills now describe Copilot CLI spans being exported directly to the configured OTLP backend instead of a local file mirror.
