---
"gh-aw": patch
---

Optimize safe output jobs to use shallow repository checkouts and targeted
branch fetching for `create-pull-request` and
`push-to-pull-request-branch` safe output jobs. This reduces network transfer and
clone time for large repositories by using `fetch-depth: 1` and fetching only the
required branch.

