# Architecture Diagram

> Last updated: 2026-05-11 · Source: [Issue created by workflow run §25663288420](https://github.com/github/gh-aw/actions/runs/25663288420)

## Overview

This diagram shows the package structure and dependencies of the `gh-aw` codebase. The project is organized into three layers: entry points (CLI binaries), core packages (main business logic), and utility packages (shared helpers).

```
┌──────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                          ENTRY POINTS                                                │
│                                                                                                      │
│          ┌─────────────────────────┐                   ┌───────────────────────────┐                │
│          │       cmd/gh-aw         │                   │      cmd/gh-aw-wasm        │                │
│          │   Main CLI binary        │                   │   WebAssembly build target │                │
│          └────────────┬────────────┘                   └─────────────┬─────────────┘                │
│                       │ imports: cli, workflow, parser, console,      │                              │
│                       │ constants                                     │                              │
├───────────────────────┼───────────────────────────────────────────────┼──────────────────────────────┤
│                       ▼              CORE PACKAGES                    ▼                              │
│                                                                                                      │
│  ┌──────────────────────────────┐   ┌───────────────────────────┐   ┌──────────────────────────┐   │
│  │           pkg/cli            │──▶│       pkg/workflow         │──▶│       pkg/parser          │   │
│  │  Command implementations     │   │  Workflow compilation      │   │  Markdown frontmatter     │   │
│  │  (compile, run, audit, logs, │   │  engine (Markdown →        │   │  parsing & content        │   │
│  │   mcp, stats)                │   │  GitHub Actions YAML)      │   │  extraction               │   │
│  └──────────────────────────────┘   └───────────────────────────┘   └──────────────────────────┘   │
│           │  also uses:                      │ also uses:                      │                     │
│           │  parser, agentdrain,             │  actionpins, console            │                     │
│           │  stats, repoutil                 │                                 │                     │
│           │                                  │                                 │                     │
│  ┌────────▼──────────┐  ┌────────────────────▼────────────────────────────────▼──────────────────┐ │
│  │  pkg/agentdrain   │  │                     pkg/console                                         │ │
│  │  Agent log drain  │  │  Terminal UI formatting, rendering, and style management                │ │
│  └───────────────────┘  └────────────────────────────────────────────────────────────────────────┘ │
│                                                                                                      │
│  ┌───────────────────────────────────────────────────────────────────────────────────────────────┐  │
│  │  pkg/actionpins — GitHub Actions pin resolution         pkg/stats — Metrics & statistics      │  │
│  └───────────────────────────────────────────────────────────────────────────────────────────────┘  │
│                                                                                                      │
├──────────────────────────────────────────────────────────────────────────────────────────────────────┤
│                                         UTILITY PACKAGES                                             │
│                                                                                                      │
│  ┌───────────────┐  ┌──────────────┐  ┌──────────────┐  ┌────────────┐  ┌──────────────────────┐  │
│  │ pkg/constants │  │  pkg/types   │  │  pkg/logger  │  │pkg/styles  │  │   pkg/stringutil     │  │
│  │ Shared consts │  │ Shared type  │  │ Debug logging│  │Terminal    │  │  String utilities    │  │
│  │ & type aliases│  │ definitions  │  │ (zero-cost)  │  │colors/styles│  │                      │  │
│  └───────────────┘  └──────────────┘  └──────────────┘  └────────────┘  └──────────────────────┘  │
│                                                                                                      │
│  ┌───────────────┐  ┌──────────────┐  ┌──────────────┐  ┌────────────┐  ┌──────────────────────┐  │
│  │ pkg/fileutil  │  │ pkg/gitutil  │  │ pkg/jsonutil │  │pkg/envutil │  │   pkg/sliceutil      │  │
│  │ File path &   │  │ Git repo     │  │ JSON utility │  │ Env var    │  │  Generic slice       │  │
│  │ I/O utilities │  │ utilities    │  │ functions    │  │ utilities  │  │  utilities           │  │
│  └───────────────┘  └──────────────┘  └──────────────┘  └────────────┘  └──────────────────────┘  │
│                                                                                                      │
│  ┌───────────────┐  ┌──────────────┐  ┌──────────────┐  ┌────────────┐  ┌──────────────────────┐  │
│  │ pkg/repoutil  │  │pkg/semverutil│  │ pkg/typeutil │  │ pkg/timeutil│  │   pkg/tty            │  │
│  │ GitHub repo   │  │ Semantic     │  │ Type convert │  │ Time helpers│  │  TTY detection       │  │
│  │ slug/URL util │  │ versioning   │  │ utilities    │  │            │  │                      │  │
│  └───────────────┘  └──────────────┘  └──────────────┘  └────────────┘  └──────────────────────┘  │
│                                                                                                      │
│  ┌──────────────────────────────────┐                                                               │
│  │  pkg/testutil  (test builds only)│                                                               │
│  │  Test helper utilities           │                                                               │
│  └──────────────────────────────────┘                                                               │
└──────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

## Package Reference

| Package | Layer | Description |
|---------|-------|-------------|
| `cmd/gh-aw` | Entry Point | Main CLI binary |
| `cmd/gh-aw-wasm` | Entry Point | WebAssembly build target |
| `pkg/cli` | Core | Command implementations (compile, run, audit, logs, mcp, stats) |
| `pkg/workflow` | Core | Workflow compilation engine (Markdown → GitHub Actions YAML) |
| `pkg/parser` | Core | Markdown frontmatter parsing and content extraction |
| `pkg/console` | Core | Terminal UI formatting, rendering, and style management |
| `pkg/agentdrain` | Core | Agent log draining and streaming |
| `pkg/actionpins` | Core | GitHub Actions pin resolution |
| `pkg/stats` | Core | Numerical statistics for metric collection |
| `pkg/constants` | Utility | Shared constants and semantic type aliases |
| `pkg/types` | Utility | Shared type definitions |
| `pkg/logger` | Utility | Namespace-based debug logging (zero overhead) |
| `pkg/styles` | Utility | Centralized terminal style/color definitions |
| `pkg/stringutil` | Utility | String utility functions |
| `pkg/fileutil` | Utility | File path and I/O operation utilities |
| `pkg/gitutil` | Utility | Git repository utilities |
| `pkg/jsonutil` | Utility | JSON utility functions |
| `pkg/repoutil` | Utility | GitHub repository slug/URL utilities |
| `pkg/envutil` | Utility | Environment variable reading/validation |
| `pkg/sliceutil` | Utility | Generic slice utilities |
| `pkg/typeutil` | Utility | General-purpose type conversion utilities |
| `pkg/semverutil` | Utility | Semantic versioning primitives |
| `pkg/timeutil` | Utility | Time helper utilities |
| `pkg/tty` | Utility | TTY detection utilities |
| `pkg/testutil` | Utility | Test helper utilities (test builds only) |
