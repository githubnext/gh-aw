---
"gh-aw": patch
---

Add if condition to custom safe output jobs to check agent output

Custom safe output jobs now automatically include an `if` condition that checks whether the safe output type (job ID) is present in the agent output, matching the behavior of built-in safe output jobs. When users provide a custom `if` condition, it's combined with the safe output type check using AND logic.
