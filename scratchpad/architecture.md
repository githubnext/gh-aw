# Architecture Diagram

> Last updated: 2026-05-03 · Source: [Issue created by workflow run §25275056556](https://github.com/github/gh-aw/actions/runs/25275056556)

## Overview

This diagram shows the package structure and dependencies of the `gh-aw` codebase. The project is organized into three layers: entry points (CLI binaries), core packages (main business logic), and utility packages (shared helpers).

```
┌─────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                        ENTRY POINTS                                             │
│          ┌──────────────────────────┐             ┌──────────────────────────┐                  │
│          │       cmd/gh-aw          │             │      cmd/gh-aw-wasm       │                  │
│          │   (main CLI binary)      │             │   (WebAssembly target)    │                  │
│          └──────┬───────────────────┘             └────────────┬─────────────┘                  │
│                 │                                              │                                 │
├─────────────────┼──────────────────────────────────────────────┼─────────────────────────────────┤
│                 ▼            CORE PACKAGES                     ▼                                │
│    ┌────────────────────────┐         ┌─────────────────────────────────┐                       │
│    │       pkg/cli          │────────▶│          pkg/workflow            │                       │
│    │  Command implementations│         │  Workflow compilation engine     │                       │
│    └────┬──────────┬────────┘         └──────────────┬──────────────────┘                       │
│         │          │                                  │                                          │
│         │          └───────────────────┐   ┌─────────┘                                          │
│         │                             ▼   ▼                                                     │
│         │                    ┌─────────────────────┐    ┌──────────────────────┐                │
│         │                    │      pkg/parser      │    │    pkg/actionpins    │                │
│         │                    │   MD/YAML parsing    │    │   Pin resolution &   │                │
│         │                    └──────────┬───────────┘    │   version pinning    │                │
│         │                              │                 └──────────────────────┘                │
│         ▼                              ▼                                                         │
│    ┌──────────────────┐     ┌─────────────────────────────────────────────────────────────┐     │
│    │  pkg/agentdrain  │     │                    pkg/console                              │     │
│    │  Log streaming   │     │  Terminal UI rendering and message formatting               │     │
│    └──────────────────┘     │  (used by cli, workflow, parser, actionpins)               │     │
│                             └─────────────────────────────────────────────────────────────┘     │
├─────────────────────────────────────────────────────────────────────────────────────────────────┤
│                                      UTILITY PACKAGES                                           │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐  │
│  │  logger  │ │constants │ │  types   │ │ typeutil │ │stringutil│ │sliceutil │ │ fileutil │  │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘ └──────────┘ └──────────┘ └──────────┘  │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐  │
│  │ gitutil  │ │  styles  │ │   tty    │ │semverutil│ │ repoutil │ │ envutil  │ │ timeutil │  │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘ └──────────┘ └──────────┘ └──────────┘  │
│  ┌──────────┐ ┌──────────┐                                                                     │
│  │  stats   │ │ testutil │                                                                     │
│  └──────────┘ └──────────┘                                                                     │
└─────────────────────────────────────────────────────────────────────────────────────────────────┘
```

## Package Reference

| Package | Layer | Description |
|---------|-------|-------------|
| `cmd/gh-aw` | Entry Point | Main CLI binary |
| `cmd/gh-aw-wasm` | Entry Point | WebAssembly compilation target |
| `pkg/cli` | Core | Command implementations for all `gh aw` subcommands |
| `pkg/workflow` | Core | Workflow compilation engine (markdown → GitHub Actions YAML) |
| `pkg/parser` | Core | Markdown frontmatter parsing and content extraction |
| `pkg/console` | Core | Terminal UI components and formatted output (widely used) |
| `pkg/agentdrain` | Core | Agent output streaming and log draining |
| `pkg/actionpins` | Core | GitHub Actions pin resolution and version management |
| `pkg/constants` | Utility | Shared constants and semantic type aliases |
| `pkg/envutil` | Utility | Environment variable reading and validation |
| `pkg/fileutil` | Utility | File path and file operation helpers |
| `pkg/gitutil` | Utility | Git repository utilities |
| `pkg/logger` | Utility | Namespace-based debug logging with zero overhead |
| `pkg/repoutil` | Utility | GitHub repository slug and URL utilities |
| `pkg/semverutil` | Utility | Semantic versioning primitives |
| `pkg/sliceutil` | Utility | Generic slice helper functions |
| `pkg/stats` | Utility | Numerical statistics for metric collection |
| `pkg/stringutil` | Utility | String manipulation helpers |
| `pkg/styles` | Utility | Centralized terminal color/style definitions |
| `pkg/testutil` | Utility | Shared test helpers |
| `pkg/timeutil` | Utility | Time formatting and duration utilities |
| `pkg/tty` | Utility | TTY (terminal) detection utilities |
| `pkg/types` | Utility | Shared type definitions used across packages |
| `pkg/typeutil` | Utility | General-purpose type conversion utilities |
