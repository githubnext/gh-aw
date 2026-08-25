> [!WARNING]
> **Protected Files**
>
> The push to pull request branch was blocked because the change modifies protected files.
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

The changes are available in the workflow run artifacts:

**Workflow Run:** [View run details and download the artifact]({run_url})

```sh
{apply_instructions}
```

</details>

To route changes like this to a review issue instead of blocking, configure `protected-files: fallback-to-issue` in your workflow configuration.
