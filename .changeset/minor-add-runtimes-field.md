---
"githubnext/gh-aw": minor
---

Add top-level `runtimes` field for runtime version overrides

Implements a new `runtimes` frontmatter field that allows users to override default runtime versions or define new runtimes in agentic workflows. Supports runtime version overrides for Node.js, Python, Ruby, Go, and other detected runtimes. Runtimes from imported shared workflows are automatically merged.
