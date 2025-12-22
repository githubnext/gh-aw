---
"gh-aw": patch
---

Optimize safe output checkout to use shallow fetch and targeted branch fetching

Safe output jobs for `create-pull-request` and `push-to-pull-request-branch` used
full repository checkouts (`fetch-depth: 0`). This change documents the optimization
to use shallow clones (`fetch-depth: 1`) and explicit branch fetches to reduce
network transfer and clone time for large repositories.

