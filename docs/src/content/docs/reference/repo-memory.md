---
title: Repo Memory
description: Guide to using repo-memory for persistent file storage via Git branches with unlimited retention.
sidebar:
  order: 1510
---

Repo memory provides persistent file storage via Git branches with unlimited retention. The compiler auto-configures branch cloning/creation, file access at `/tmp/gh-aw/repo-memory-{id}/`, commits/pushes, and merge conflict resolution (your changes win).

## Enabling Repo Memory

```aw wrap
---
tools:
  repo-memory: true
---
```

Creates branch `memory/default` at `/tmp/gh-aw/repo-memory-default/`. Files are stored within the branch at the branch name path (`memory/default/`). Files auto-commit/push after workflow completion.

## Advanced Configuration

```aw wrap
---
tools:
  repo-memory:
    branch-name: memory/custom-agent-for-aw
    branch-prefix: tracking  # Custom prefix instead of "memory"
    description: "Long-term insights"
    file-glob: ["*.md", "*.json"]
    max-file-size: 1048576  # 1MB (default 100KB)
    max-file-count: 50      # default 100
    max-patch-size: 1048576  # 1MB max (default 10KB)
    target-repo: "owner/repository"
    create-orphan: true     # default
    allowed-extensions: [".json", ".txt", ".md"]  # Restrict file types (default: empty/all files allowed)
---
```

**Branch Prefix**: Use `branch-prefix` to customize the branch name prefix (default is `memory`). The prefix must be 4-32 characters, alphanumeric with hyphens/underscores, and cannot be `copilot`. When set, branches are created as `{branch-prefix}/{id}` instead of `memory/{id}`.

**File Type Restrictions**: Use `allowed-extensions` to restrict which file types can be stored (default: empty/all files allowed). When specified, only files with listed extensions (e.g., `[".json", ".txt", ".md"]`) can be saved. Files with disallowed extensions will trigger validation failures.

**Patch Size Limit**: Use `max-patch-size` to limit the total size of changes in a single push (default: 10KB, max: 1MB). The total size of the git diff (all staged changes combined) must not exceed this value. If it does, the push is rejected with an error. Use this to prevent large unintentional memory updates.

**Note**: File glob patterns are matched against the **relative file path** within the artifact directory, not the branch path. Use bare extension patterns like `*.json` or `*.md` — do **not** include the branch name (e.g. `memory/custom-agent-for-aw/*.json` is incorrect).

## Multiple Configurations

```aw wrap
---
tools:
  repo-memory:
    - id: insights
      branch-prefix: daily  # Creates daily/insights branch
      file-glob: ["*.md"]
    - id: state
      file-glob: ["*.json"]
      max-file-size: 524288  # 512KB
---
```

Mounts at `/tmp/gh-aw/repo-memory-{id}/` during workflow execution. Required `id` determines folder name; `branch-name` defaults to `{branch-prefix}/{id}` (where `branch-prefix` defaults to `memory`). Files are stored within the git branch at the branch name path (e.g., for branch `memory/code-metrics`, files are stored at `memory/code-metrics/` within the branch). **File glob patterns are matched against the relative file path within the artifact directory — never include the branch name in patterns.**

## Behavior

Branches auto-create as orphans (default) or clone with `--depth 1`. Changes auto-commit after validation (`file-glob`, `max-file-size`, `max-file-count`) and push when changes detected and threat detection passes.

Commits are pushed via the [GitHub GraphQL `createCommitOnBranch` mutation](https://docs.github.com/en/graphql/reference/mutations#createcommitonbranch), which signs each commit with GitHub's GPG key. This means repo-memory pushes are automatically **Verified** and satisfy repository rulesets that require signed commits (e.g. enterprise "Commits must have verified signatures" baselines).

:::note[Signed-commit fallback limitation]
The GraphQL mutation does not support symlinks, executable files (`chmod +x`), or submodule entries. If your memory artifact contains any of these, the helper falls back to a plain `git push`, which will be rejected by signed-commit rulesets. Keep memory artifacts as regular plain-text files (`.json`, `.jsonl`, `.txt`, `.md`, `.csv` — the default `allowed-extensions`).
:::

## Comparison with Cache Memory

| Feature | Cache Memory | Repo Memory |
|---------|--------------|-------------|
| Storage | GitHub Actions Cache | Git Branches |
| Retention | 7 days | Unlimited |
| Size Limit | 10GB/repo | Repository limits |
| Version Control | No | Yes |
| Performance | Fast | Slower |
| Best For | Temporary/sessions | Long-term/history |

For fast 7-day caching without version control, see [Cache Memory](/gh-aw/reference/cache-memory/).

## Troubleshooting

- **Branch not created**: Ensure `create-orphan: true` or create manually.
- **Validation failures**: Match `file-glob`, stay under `max-file-size` (100KB default), `max-file-count` (100 default), and `max-patch-size` (10KB default).
- **Patch too large**: If the total diff exceeds `max-patch-size` (default 10KB), the push is rejected. Reduce the number or size of changes, or increase `max-patch-size` in the configuration.
- **Changes not persisting**: Check directory path, workflow completion, push errors in logs.
- **Merge conflicts**: Concurrent pushes are handled: if another run has pushed since the branch was checked out, the GraphQL mutation replays your file diff on top of the latest remote state (your changes win).
- **GH013 — Commits must have verified signatures**: Repo-memory uses GraphQL signed commits, so this error should not appear for standard plain-text memory files. If it does, your artifact contains a symlink, executable file, or submodule entry that forced a fallback to `git push`. Remove the unsupported file type and re-run.

## Security

Don't store sensitive data in repo memory. Repo memory follows repository permissions.

Use private repos for sensitive data, avoid storing secrets, set constraints (`file-glob`, `max-file-size`, `max-file-count`, `max-patch-size`), consider branch protection, use `target-repo` to isolate.

## Examples

See [Deep Report](https://github.com/github/gh-aw/blob/main/.github/workflows/deep-report.md) and [Daily Firewall Report](https://github.com/github/gh-aw/blob/main/.github/workflows/daily-firewall-report.md) for long-term insights and historical data tracking.

## Related Documentation

- [Cache Memory](/gh-aw/reference/cache-memory/) - GitHub Actions cache-based storage with 7-day retention
- [Frontmatter](/gh-aw/reference/frontmatter/) - Complete frontmatter configuration guide
- [Safe Outputs](/gh-aw/reference/safe-outputs/) - Output processing and automation
