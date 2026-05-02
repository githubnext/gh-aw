# Architecture Diagram

> Last updated: 2026-05-02 · Source: Issue created by workflow run [§25248432943](https://github.com/github/gh-aw/actions/runs/25248432943)

## Overview

This diagram shows the package structure and dependencies of the `gh-aw` codebase. The project is organized into three layers: entry points (CLI binaries), core packages (main business logic), and utility packages (shared helpers).

```
┌─────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                        ENTRY POINTS                                             │
│                                                                                                 │
│          ┌──────────────────────────┐             ┌─────────────────────────┐                  │
│          │       cmd/gh-aw          │             │     cmd/gh-aw-wasm       │                  │
│          │   (main CLI binary)      │             │   (WebAssembly target)   │                  │
│          └────────────┬─────────────┘             └────────────┬────────────┘                  │
│                       │                                        │                                │
├───────────────────────┼────────────────────────────────────────┼────────────────────────────────┤
│                       ▼          CORE PACKAGES                 ▼                               │
│    ┌──────────────────────────┐       ┌────────────────────────────────┐                       │
│    │         pkg/cli          │──────▶│        pkg/workflow             │                       │
│    │  Command implementations │       │   Workflow compilation engine   │                       │
│    └────────────┬─────────────┘       └──────────────┬─────────────────┘                       │
│                 │                                     │                                         │
│                 │         ┌──────────────────────┐    │                                         │
│                 └────────▶│      pkg/parser       │◀──┘                                         │
│                           │  Markdown/YAML parsing│                                             │
│                           └──────────┬────────────┘                                             │
│                                      │                                                          │
│    ┌─────────────────────────────┐   │                                                          │
│    │        pkg/console          │◀──┘                                                          │
│    │   Terminal UI & formatting  │                                                              │
│    └──────────────┬──────────────┘                                                              │
│                   │                                                                              │
│    ┌──────────────┴────────────┐                                                                │
│    │       pkg/agentdrain      │                                                                │
│    │  Agent output streaming   │                                                                │
│    └───────────────────────────┘                                                                │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│                                      UTILITY PACKAGES                                           │
│                                                                                                 │
│  ┌────────────┐ ┌────────────┐ ┌─────────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────────┐ │
│  │ pkg/logger │ │pkg/fileutil│ │ pkg/gitutil  │ │pkg/types │ │pkg/const-│ │  pkg/stringutil  │ │
│  │Debug logger│ │File ops    │ │Git utilities │ │ Shared   │ │  ants    │ │ String helpers   │ │
│  └────────────┘ └────────────┘ └─────────────┘ │  types   │ │Constants │ └──────────────────┘ │
│                                                 └──────────┘ └──────────┘                      │
│  ┌────────────┐ ┌────────────┐ ┌─────────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────────┐ │
│  │pkg/sliceutil│ │pkg/typeutil│ │pkg/repoutil │ │pkg/envutil│ │pkg/styles│ │  pkg/actionpins  │ │
│  │Slice helpers│ │Type conv.  │ │Repo URL util│ │Env vars  │ │UI styles │ │Action pin resolver│ │
│  └────────────┘ └────────────┘ └─────────────┘ └──────────┘ └──────────┘ └──────────────────┘ │
│                                                                                                 │
│  ┌────────────┐ ┌────────────┐ ┌─────────────┐ ┌──────────┐ ┌──────────┐                      │
│  │ pkg/tty    │ │  pkg/stats  │ │pkg/semverutil│ │pkg/time- │ │pkg/test- │                      │
│  │TTY detect  │ │Statistics  │ │Semver primit.│ │  util    │ │  util    │                      │
│  └────────────┘ └────────────┘ └─────────────┘ └──────────┘ └──────────┘                      │
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
| `pkg/console` | Core | Terminal UI components and formatted output |
| `pkg/agentdrain` | Core | Agent output streaming and log draining |
| `pkg/actionpins` | Utility | GitHub Actions pin resolution |
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
