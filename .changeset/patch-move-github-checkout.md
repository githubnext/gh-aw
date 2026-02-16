---
"gh-aw": patch
---

Move the `.github` and `.agents` sparse checkout into the activation job so prompt rendering validation and other activation-time steps can rely on the repository files before the agent job runs, and regenerate the workflows to capture the new setup.
