---
title: Drive Memory
description: Experimental persistent workflow memory backed by GitHub Drives.
sidebar:
  order: 1510
---

Drive memory provides persistent file storage across workflow runs using the experimental [GitHub Drives preview](https://github.com/actions/gh-drives-preview). The repository must be enrolled in the preview, and the workflow must run on a Linux runner with FUSE support.

> [!CAUTION]
> Drive memory and the underlying GitHub Drives service are experimental. The configuration and storage behavior may change without notice.

## Enable drive memory

```aw wrap
---
tools:
  drive-memory: true
---
```

The compiler mounts the `default` drive and exposes it to the agent at `/tmp/gh-aw/drive-memory/`. It automatically grants the generated job `contents: read`, `id-token: write`, and the required `drives` permission.

## Configure a drive

```aw wrap
---
tools:
  drive-memory:
    drive-name: agent-memory
    disk-size: 20G
    prefetch: true
    description: Long-lived notes and state
    allowed-extensions: [".json", ".jsonl", ".txt", ".md"]
---
```

- `drive-name` selects the persistent drive. It defaults to `default`.
- `disk-size` sets the size when the drive is first created and defaults to `10G`.
- `prefetch` eagerly downloads existing content after mounting.
- `restore-only` mounts with `drives: read` and never commits local changes.
- `description`, `allowed-extensions`, and `validation` behave like their [cache-memory](/gh-aw/reference/cache-memory/) equivalents.

## Multiple drives

```aw wrap
---
tools:
  drive-memory:
    - id: notes
      drive-name: agent-notes
    - id: reference
      drive-name: shared-reference
      restore-only: true
      prefetch: true
---
```

The `default` entry is exposed at `/tmp/gh-aw/drive-memory/`; named entries use `/tmp/gh-aw/drive-memory-{id}/`.

## Persistence and threat detection

Without threat detection, the compiler checks out each drive before the agent and commits validated changes afterward. With threat detection enabled, the agent receives a non-publishing checkout and the compiler stages drive contents as an artifact. A separate `update_drive_memory` job publishes the artifact only after detection succeeds. Before replacing the drive, that job verifies it has not changed since the agent checked it out; concurrent updates cause the publish to fail instead of being overwritten.

Drive names are repository-wide and branch-aware according to the preview service. GitHub Drives allows one active writer for a drive, so overlapping runs that write the same drive can contend for the writer lease.

## Limitations

- GitHub-hosted `ubuntu-latest` is the supported preview runner.
- Repositories must be enrolled in the GitHub Drives preview.
- Drive mounts are not supported inside job containers.
- The upstream actions currently have no versioned release, so gh-aw pins the preview `main` commit.
- Do not store secrets in drive memory.
