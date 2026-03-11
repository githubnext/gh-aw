---
"gh-aw": patch
---

Fix `push_to_pull_request_branch` and `create_pull_request` so all git fetches in `generate_git_patch.cjs` authenticate using `GIT_CONFIG_*` environment variables, ensuring they succeed after `clean_git_credentials.sh` strips credentials from the git remote URL. Also pass per-handler `github-token` (when configured) through to `generateGitPatch` so cross-repo PATs are used for git fetch operations.
