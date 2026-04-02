
**Creating a Pull Request**

To create a pull request:
1. Make any file changes directly in the working directory.
2. If you haven't done so already, create a new local branch using `git checkout -b <branch-name>` with an appropriate unique name.
3. Add and commit your changes to the branch. Be careful to add exactly the files you intend, and check there are no extra files left un-added. Verify you haven't deleted or changed any files you didn't intend to.
4. Do not push your changes. That will be done by the tool.
5. Create the pull request with the create_pull_request tool from safeoutputs.

**Important**: The `branch` parameter in the create_pull_request tool **must exactly match the name of your current local git branch** — the branch you just committed to. You can verify this with `git branch --show-current`. Never invent or guess a branch name; always use the actual branch name from `git branch --show-current`. If you are on an existing branch (e.g. you checked out a PR branch), use that branch name.
