---
"gh-aw": patch
---

Add actions directory structure and Go-based build tooling; initial actions and Makefile targets.

This documents the changes introduced by PR #5953: create an `actions/` directory, add `actions-build`, `actions-validate`, and `actions-clean` targets, initial actions `setup-safe-inputs` and `setup-safe-outputs`, and supporting Go CLI commands for building and validating actions.

Fixes githubnext/gh-aw#5948

