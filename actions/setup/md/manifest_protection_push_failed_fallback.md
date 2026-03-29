{main_body}

---

> [!WARNING]
> **Protected Files — Push Permission Denied**
>
> This was originally intended as a pull request, but the patch modifies protected files. A human must create the pull request manually.

<details>
<summary>Protected files</summary>

{files}

The push was rejected because GitHub Actions does not have `workflows` permission to push these changes, and is never allowed to make such changes, or other authorization being used does not have this permission.

</details>

<details>
<summary><b>Create the pull request manually</b></summary>

```sh
# Download the patch from the workflow run
gh run download {run_id} -n agent -D /tmp/agent-{run_id}

# Create a new branch
git checkout -b {branch_name} {base_branch}

# Apply the patch (--3way handles cross-repo patches)
git am --3way /tmp/agent-{run_id}/{patch_file}

# Push the branch and create the pull request
git push origin {branch_name}
gh pr create --title '{title}' --base {base_branch} --head {branch_name} --repo {repo}
```

</details>

{footer}
