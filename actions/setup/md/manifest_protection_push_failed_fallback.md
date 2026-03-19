{main_body}

---

> [!WARNING]
> 🛡️ **Protected Files — Push Permission Denied**
>
> This was originally intended as a pull request, but the patch modifies protected files: {files}.
>
> The push was rejected because GitHub Actions does not have `workflows` permission to push these changes, and is never allowed to make such changes, or other authorization being used does not have this permission. A human must create the pull request manually.

To create a pull request with the changes:

```sh
# Download the patch from the workflow run
gh run download {run_id} -n agent-artifacts -D /tmp/agent-artifacts-{run_id}

# Create a new branch
git checkout -b {branch_name} {base_branch}

# Apply the patch (--3way handles cross-repo patches)
git am --3way /tmp/agent-artifacts-{run_id}/{patch_file}

# Push the branch and create the pull request
git push origin {branch_name}
gh pr create --title '{title}' --base {base_branch} --head {branch_name} --repo {repo}
```

{footer}
