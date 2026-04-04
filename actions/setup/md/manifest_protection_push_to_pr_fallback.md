> [!WARNING]
> **Protected Files**
>
> The push to pull request branch was blocked because the patch modifies protected files.
>
> **Target Pull Request:** [#{pull_number}]({pr_url})
>
> **Please review the changes carefully** before pushing them to the pull request branch. These files may affect project dependencies, CI/CD pipelines, or agent behaviour.
>
> <details>
> <summary>Protected files</summary>
>
> {files}
>
> </details>

---

<details>
<summary>Apply the patch after review</summary>

The patch is available in the workflow run artifacts:

**Workflow Run:** [View run details and download patch artifact]({run_url})

```sh
# Download the artifact from the workflow run
gh run download {run_id} -n agent -D /tmp/agent-{run_id}

# Apply the patch to the pull request branch
git fetch origin {branch_name}
git checkout {branch_name}
git am --3way /tmp/agent-{run_id}/{patch_file_name}
git push origin {branch_name}
```

</details>

To route changes like this to a review issue instead of blocking, configure `protected-files: fallback-to-issue` in your workflow configuration.
