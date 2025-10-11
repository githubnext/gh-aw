---
"gh-aw": patch
---

Add GitHub Copilot agent setup workflow

Adds a `.github/workflows/copilot-setup-steps.yml` workflow file to configure the GitHub Copilot coding agent environment with preinstalled tools and dependencies. The workflow mirrors the setup steps from the CI workflow's build job, including Node.js, Go, JavaScript dependencies, development tools, and build step. This provides Copilot agents with a fully configured development environment and speeds up agent workflows.
