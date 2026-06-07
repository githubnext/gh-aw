---
"gh-aw": patch
---

Accept `${{ ... }}` GitHub Actions expressions inside `safe-outputs.*.samples` entries: such values now bypass compile-time JSON-schema validation and are preserved verbatim in the lock file's `GH_AW_SAMPLES` env value, so GitHub Actions substitutes the real value on the runner before `apply_samples.cjs` reads it. This unblocks `--use-samples` mode for `workflow_dispatch`-triggered safe-output tests that target a runtime-supplied issue/PR number (e.g. `item_number: ${{ github.event.inputs.issue_number }}`).
