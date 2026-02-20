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
    file-glob: ["memory/custom-agent-for-aw/*.md", "memory/custom-agent-for-aw/*.json"]
    max-file-size: 1048576  # 1MB (default 10KB)
    max-file-count: 50      # default 100
    target-repo: "owner/repository"
    create-orphan: true     # default
    allowed-extensions: [".json", ".txt", ".md"]  # Restrict file types (default: empty/all files allowed)
---
```

**Branch Prefix**: Use `branch-prefix` to customize the branch name prefix (default is `memory`). The prefix must be 4-32 characters, alphanumeric with hyphens/underscores, and cannot be `copilot`. When set, branches are created as `{branch-prefix}/{id}` instead of `memory/{id}`.

**File Type Restrictions**: Use `allowed-extensions` to restrict which file types can be stored (default: empty/all files allowed). When specified, only files with listed extensions (e.g., `[".json", ".txt", ".md"]`) can be saved. Files with disallowed extensions will trigger validation failures.

**Note**: File glob patterns must include the full branch path structure. For branch `memory/custom-agent-for-aw`, use patterns like `memory/custom-agent-for-aw/*.json` to match files stored at that path within the branch.

## Multiple Configurations

```aw wrap
---
tools:
  repo-memory:
    - id: insights
      branch-prefix: daily  # Creates daily/insights branch
      file-glob: ["daily/insights/*.md"]
    - id: state
      file-glob: ["memory/state/*.json"]
      max-file-size: 524288  # 512KB
---
```

Mounts at `/tmp/gh-aw/repo-memory-{id}/` during workflow execution. Required `id` determines folder name; `branch-name` defaults to `{branch-prefix}/{id}` (where `branch-prefix` defaults to `memory`). Files are stored within the git branch at the branch name path (e.g., for branch `memory/code-metrics`, files are stored at `memory/code-metrics/` within the branch). **File glob patterns must include the full branch path.**

## Behavior

Branches auto-create as orphans (default) or clone with `--depth 1`. Changes auto-commit after validation (`file-glob`, `max-file-size`, `max-file-count`), pull with `-X ours` (your changes win), and push when changes detected and threat detection passes. Auto-adds `contents: write` permission.

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
- **Permission denied**: Compiler auto-adds `contents: write`.
- **Validation failures**: Match `file-glob`, stay under `max-file-size` (10KB default) and `max-file-count` (100 default).
- **Changes not persisting**: Check directory path, workflow completion, push errors in logs.
- **Merge conflicts**: Uses `-X ours` (your changes win). Read before writing to preserve data.

## Security

Don't store sensitive data in repo memory. Repo memory follows repository permissions.

Use private repos for sensitive data, avoid storing secrets, set constraints (`file-glob`, `max-file-size`, `max-file-count`), consider branch protection, use `target-repo` to isolate.

## Examples

See [Deep Report](https://github.com/github/gh-aw/blob/main/.github/workflows/deep-report.md) and [Daily Firewall Report](https://github.com/github/gh-aw/blob/main/.github/workflows/daily-firewall-report.md) for long-term insights and historical data tracking.

## Related Documentation

- [Cache Memory](/gh-aw/reference/cache-memory/) - GitHub Actions cache-based storage with 7-day retention
- [Frontmatter](/gh-aw/reference/frontmatter/) - Complete frontmatter configuration guide
- [Safe Outputs](/gh-aw/reference/safe-outputs/) - Output processing and automation
