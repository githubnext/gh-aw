---
"gh-aw": patch
---

Fix: Ensure `create-pull-request` and `push-to-pull-request-branch` safe outputs
are applied correctly by downloading the patch artifact, checking out the
repository, configuring git, and using the appropriate token when available.

This is an internal tooling fix for action workflows; it does not change the
public CLI API.

--
PR: #7167

