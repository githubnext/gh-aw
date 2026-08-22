---
title: Drive Memory
description: Private-preview GitHub Drives integration reference.
sidebar:
  order: 1510
---

Drive memory is an experimental, feature-gated integration backed by the [GitHub Drives preview](https://github.com/actions/gh-drives-preview). Do not configure it unless GitHub has explicitly enrolled the repository in the private preview.

> [!CAUTION]
> Drive memory and the underlying GitHub Drives service are experimental. This
> reference records behavior for enrolled preview repositories and is not a
> recommendation for general use.

## Preview behavior

For enrolled preview repositories, the compiler mounts configured drives into the agent at `/tmp/gh-aw/drive-memory/` (or `/tmp/gh-aw/drive-memory-{id}/` for named entries). It grants the generated job `contents: read`, `id-token: write`, and the required `drives` permission.

The compiler checks out each drive before the agent and commits validated changes afterward. With threat detection enabled, it stages drive contents as an artifact; a separate `update_drive_memory` job publishes it only after detection succeeds and verifies the drive has not changed since checkout.

Drive names are repository-wide and branch-aware according to the preview service. GitHub Drives allows one active writer for a drive, so overlapping runs that write the same drive can contend for the writer lease.

## Limitations

- GitHub-hosted `ubuntu-latest` is the supported preview runner.
- Repositories must be enrolled in the GitHub Drives preview.
- Drive mounts are not supported inside job containers.
- The upstream actions currently have no versioned release, so gh-aw pins the preview `main` commit.
- Do not store secrets in drive memory.
