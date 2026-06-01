
**Pushing Changes to a Pull Request Branch**

To push changes to the branch of a pull request:
1. Make any file changes directly in the working directory.
2. Add and commit your changes to the local copy of the pull request branch. Be careful to add exactly the files you intend, and verify you haven't deleted or changed any files you didn't intend to.
3. Push the branch to the repo by using the push_to_pull_request_branch tool from safeoutputs.

**Multi-checkout workflows (`checkout:` with multiple repositories):**
- `push_to_pull_request_branch` operates on the checkout for the target repository (the directory matching the `path:` value in your workflow's checkout step).
- Run all `git` commands from that checkout directory before calling the tool. Use a subshell (`(cd <target-checkout-path> && git ...)`) or `pushd`/`popd` to avoid changing the working directory for subsequent commands in the same step.
- If needed, check out the PR branch locally from `origin/<pr-branch>` first.
**Important constraints:**
- This tool is **append-only**: it adds new commits on top of the existing PR branch. Force-push is NOT supported.
- Do NOT use `git merge` to bring another branch (e.g., `main`) into the PR branch — merge commits cannot be signed; the action will attempt to squash them into a single linear commit before pushing, but this rewrites history. Use `git rebase` instead (e.g., `git rebase origin/main`) to avoid the rewrite.
- **No git credentials are available**: Git credentials are intentionally removed after checkout for security. Do NOT attempt `git fetch`, `git pull`, or any other network git operation that requires authentication (e.g., private repos) — it will fail. Remote-tracking refs already fetched during checkout (e.g., `origin/main`) are available for local operations like `git rebase origin/main`. If you need a branch or ref that is not already present locally, stop and report this rather than attempting to fetch it.
