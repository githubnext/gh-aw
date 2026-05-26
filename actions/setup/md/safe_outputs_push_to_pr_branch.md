
**Pushing Changes to a Pull Request Branch**

To push changes to the branch of a pull request:
1. Make any file changes directly in the working directory.
2. Add and commit your changes to the local copy of the pull request branch. Be careful to add exactly the files you intend, and verify you haven't deleted or changed any files you didn't intend to.
3. Push the branch to the repo by using the push_to_pull_request_branch tool from safeoutputs.

**Important constraints:**
- This tool is **append-only**: it adds new commits on top of the existing PR branch. Force-push is NOT supported.
- Do NOT use `git merge` to bring another branch (e.g., `main`) into the PR branch — merge commits cannot be signed; the action will attempt to squash them into a single linear commit before pushing, but this rewrites history. Use `git rebase` instead (e.g., `git rebase origin/main`) to avoid the rewrite.
