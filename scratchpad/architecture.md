# Architecture Diagram

> Last updated: 2026-04-23 · Source: Issue #28016 (Architecture Diagram)

## Overview

This diagram shows the package structure and dependencies of the `gh-aw` codebase.

```
┌──────────────────────────────────────────────────────────────────────────────────────────────┐
│                                      ENTRY POINTS                                             │
│                                                                                               │
│         ┌─────────────────────┐                    ┌─────────────────────┐                   │
│         │     cmd/gh-aw       │                    │  cmd/gh-aw-wasm     │                   │
│         │  (CLI binary)       │                    │  (WASM target)      │                   │
│         └──────────┬──────────┘                    └──────────┬──────────┘                   │
│                    │  cli, workflow, parser, console           │                               │
└────────────────────┼──────────────────────────────────────────┼───────────────────────────────┘
                     │                                           │
┌────────────────────▼──────────────────────────────────────────▼───────────────────────────────┐
│                                     CORE PACKAGES                                              │
│                                                                                               │
│  ┌──────────────────────────────────────────────────────────────────────────────────────┐    │
│  │  pkg/cli  ·  Command implementations (compile, audit, add, mcp, logs, upgrade, ...)  │    │
│  └──────┬──────────┬──────────────────────────────┬──────────────────────────┬──────────┘    │
│         │          │                              │                          │                │
│         ▼          ▼                              ▼                          ▼                │
│  ┌────────────┐  ┌──────────────────────┐  ┌─────────────────┐  ┌──────────────────────┐   │
│  │pkg/workflow│  │    pkg/agentdrain     │  │ pkg/actionpins  │  │    pkg/console       │   │
│  │ Compilation│  │ Log analysis/anomaly  │  │ Pin resolution  │  │  Terminal UI/format  │   │
│  │  engine    │  │ detection/clustering  │  │  & versioning   │  │                      │   │
│  └──────┬─────┘  └──────────────────────┘  └────────┬────────┘  └──────────┬───────────┘   │
│         │                                            │                       │               │
│         ▼                                            │                       │               │
│  ┌─────────────────────────────────────┐             │                       │               │
│  │           pkg/parser                │◀────────────┘                       │               │
│  │  Markdown frontmatter parsing &     │                                     │               │
│  │  YAML content extraction            │                                     │               │
│  └─────────────────────────────────────┘                                     │               │
│                                                                               │               │
└───────────────────────────────────────────────────────────────────────────────┼───────────────┘
                                                                                │
┌───────────────────────────────────────────────────────────────────────────────▼───────────────┐
│                                    UTILITY PACKAGES                                            │
│                                                                                               │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────┐             │
│  │pkg/constants│  │ pkg/types  │  │pkg/typeutil│  │pkg/fileutil│  │ pkg/gitutil│             │
│  │ Typed const │  │ Shared type│  │ Type conv. │  │ File/path  │  │ Git repo   │             │
│  │ & flags     │  │ definitions│  │ utilities  │  │ operations │  │ utilities  │             │
│  └────────────┘  └────────────┘  └────────────┘  └────────────┘  └────────────┘             │
│                                                                                               │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────┐             │
│  │pkg/stringutil│  │pkg/sliceutil│  │ pkg/logger │  │ pkg/styles │  │  pkg/tty   │             │
│  │ String ops  │  │ Generic    │  │ Namespace  │  │ Terminal   │  │ Terminal   │             │
│  │ & ANSI strip│  │ slice utils│  │ debug log  │  │ colors     │  │ detection  │             │
│  └────────────┘  └────────────┘  └────────────┘  └────────────┘  └────────────┘             │
│                                                                                               │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────┐             │
│  │pkg/semverutil│  │ pkg/envutil│  │pkg/repoutil│  │  pkg/stats │  │pkg/timeutil│             │
│  │ Semver ops  │  │ Env var    │  │ Repo slug  │  │ Numerical  │  │ Time/dur.  │             │
│  │ & parsing   │  │ validation │  │ & URL utils│  │ statistics │  │ formatting │             │
│  └────────────┘  └────────────┘  └────────────┘  └────────────┘  └────────────┘             │
└───────────────────────────────────────────────────────────────────────────────────────────────┘
```

## Package Reference

| Package | Layer | Description |
|---------|-------|-------------|
| cmd/gh-aw | Entry | Main CLI binary |
| cmd/gh-aw-wasm | Entry | WebAssembly compilation target |
| pkg/cli | Core | Command implementations: compile, audit, add, mcp, logs, upgrade, codemod, checks, etc. |
| pkg/workflow | Core | Workflow compilation engine — transforms markdown frontmatter into GitHub Actions YAML |
| pkg/parser | Core | Markdown frontmatter parsing and content extraction |
| pkg/console | Core | Terminal UI components and user-facing formatting utilities |
| pkg/actionpins | Core | GitHub Actions pin resolution and version pinning |
| pkg/agentdrain | Core | Log analysis, anomaly detection, and clustering for workflow audit |
| pkg/constants | Utility | Shared typed constants: versions, URLs, feature flags, engine/job names |
| pkg/types | Utility | Shared type definitions used across packages |
| pkg/typeutil | Utility | General-purpose type conversion utilities |
| pkg/fileutil | Utility | File path and file operation helpers |
| pkg/gitutil | Utility | Git repository interaction utilities |
| pkg/stringutil | Utility | String manipulation utilities (ANSI stripping, normalization) |
| pkg/sliceutil | Utility | Generic slice operation utilities |
| pkg/logger | Utility | Namespace-based debug logging with zero overhead when disabled |
| pkg/styles | Utility | Centralized terminal style and color definitions (used by console) |
| pkg/tty | Utility | TTY (terminal) detection and width utilities |
| pkg/semverutil | Utility | Semantic versioning primitives |
| pkg/envutil | Utility | Environment variable reading and validation |
| pkg/repoutil | Utility | GitHub repository slug and URL parsing utilities |
| pkg/stats | Utility | Numerical statistics for metric collection (used by agentdrain) |
| pkg/timeutil | Utility | Time formatting and duration utilities |
| pkg/testutil | Utility | Testing helpers (test builds only) |
