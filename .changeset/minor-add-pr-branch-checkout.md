---
"gh-aw": minor
---

Add PR branch checkout when pull request context is available

When a workflow is triggered by pull_request events or comments on pull requests, the repository is now automatically checked out to the actual PR branch instead of remaining in a detached HEAD state. This enables agentic jobs to make commits and push changes to the PR branch.

The implementation uses JavaScript with actions/github-script@v8 and exec.exec from @actions/exec for better maintainability. The checkout step is added after git configuration for all workflows with contents: read permission and uses runtime conditions to only execute when PR context is available.
