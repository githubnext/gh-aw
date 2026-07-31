---
"gh-aw": patch
---

Fixed `gh aw logs` reporting "No workflow runs with artifacts found matching the specified criteria" (and an empty `.runs` array with `--json`) even when artifacts were downloaded successfully. The batch collection loop assigned its results with `:=`, which declared a loop-scoped `processedRuns` slice that shadowed the accumulator, so every batch of processed runs was discarded and the command kept paginating until the run list was exhausted.
