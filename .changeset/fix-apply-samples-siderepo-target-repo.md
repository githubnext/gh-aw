---
"gh-aw": patch
---

Fixed `apply_samples.cjs` `derivePrHeadRef` to consult `target-repo` from the safe-outputs config file (`GH_AW_SAFE_OUTPUTS_CONFIG_PATH`) when resolving the repository for PR head-ref lookups. Previously the function fell back directly to `GITHUB_REPOSITORY` (the host repo), causing a 404 for siderepo `push_to_pull_request_branch` workflow_dispatch samples that carry a `pull_request_number` but no explicit `repo` argument (issue #41292).
