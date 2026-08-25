{main_body}

---

> [!WARNING]
> **Protected Files — Push Permission Denied**
>
> This was originally intended as a pull request, but the change modifies protected files. A human must create the pull request manually.
>
> <details>
> <summary>Protected files</summary>
>
> {files}
>
> The push was rejected because GitHub Actions does not have `workflows` permission to push these changes, and is never allowed to make such changes, or other authorization being used does not have this permission.
>
> </details>

<details>
<summary>Create the pull request manually</summary>

```sh
{apply_instructions}

# Push the branch and create the pull request
git push origin {branch_name}
gh pr create --title '{title}' --base {base_branch} --head {branch_name} --repo {repo}
```

</details>

{footer}
