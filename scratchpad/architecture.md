# Architecture Diagram

> Last updated: 2026-04-18 · Source: [Run §24601274706](https://github.com/github/gh-aw/actions/runs/24601274706)

## Overview

This diagram shows the package structure and dependencies of the `gh-aw` codebase.

```
┌────────────────────────────────────────────────────────────────────────────────────────────┐
│                                      ENTRY POINTS                                          │
│                                                                                            │
│    ┌──────────────────────────────────────────┐  ┌────────────────────────────────────┐   │
│    │               cmd/gh-aw                  │  │         cmd/gh-aw-wasm             │   │
│    │            Main CLI binary               │  │       WebAssembly target           │   │
│    │  imports: cli,console,constants,         │  │  imports: parser, workflow         │   │
│    │           parser, workflow               │  │                                    │   │
│    └────────────┬─────────────────────────────┘  └──────────────┬─────────────────────┘   │
│                 │                                                 │                         │
├─────────────────┼─────────────────────────────────────────────────┼─────────────────────────┤
│                 │               CORE PACKAGES                      │                         │
│                 ▼                                                   ▼                         │
│    ┌──────────────────────────────────┐     ┌────────────────────────────────────────────┐  │
│    │            pkg/cli               │     │              pkg/workflow                  │  │
│    │  Command dispatch, flag handling,│────▶│  Workflow compilation and GitHub Actions   │  │
│    │  and CLI command execution       │     │  YAML generation                           │  │
│    └───┬───────────────────────────┬──┘     └───────────────────┬────────────────┬───────┘  │
│        │                           │                             │                │          │
│        │                           │                             ▼                ▼          │
│        │                           │              ┌─────────────────┐  ┌──────────────────┐ │
│        │                           │              │   pkg/parser    │  │  pkg/actionpins  │ │
│        │                           │              │  Markdown/YAML  │  │  GitHub Actions  │ │
│        │                           │              │  frontmatter &  │  │  pin version     │ │
│        │                           │              │  schema parsing │  │  resolution      │ │
│        │                           │              └────────┬────────┘  └──────────┬───────┘ │
│        │    ┌──────────────────────┼──────────────────────┘                       │         │
│        │    │                      │                                               │         │
│        │    ▼                      ▼                                               ▼         │
│        │  ┌──────────────────────────────────────────────────────────────────────────┐      │
│        ├─▶│                            pkg/console                                   │      │
│        │  │     Terminal UI: spinners, message formatting, styled output rendering    │      │
│        │  └──────────────────────────────────────────────────────────────────────────┘      │
│        │                                                                                     │
│        │  ┌──────────────────────────────────────────┐                                      │
│        ├─▶│              pkg/agentdrain               │                                      │
│        │  │   Agent log streaming and drain for CI   │                                      │
│        │  └──────────────────────────────────────────┘                                      │
│        │  ┌──────────────────────────────────────────┐                                      │
│        └─▶│                pkg/stats                  │                                      │
│           │   Numerical statistics and metrics        │                                      │
│           └──────────────────────────────────────────┘                                      │
├────────────────────────────────────────────────────────────────────────────────────────────┤
│                                   UTILITY PACKAGES                                         │
│  ┌──────────┐  ┌────────┐  ┌────────┐  ┌──────────┐  ┌─────────┐  ┌──────────┐  ┌──────┐  │
│  │constants │  │ types  │  │ logger │  │ fileutil │  │ gitutil │  │ repoutil │  │envutil│  │
│  └──────────┘  └────────┘  └────────┘  └──────────┘  └─────────┘  └──────────┘  └──────┘  │
│  ┌────────────┐  ┌──────────┐  ┌──────────┐  ┌────────────┐  ┌───────┐  ┌────────┐         │
│  │ stringutil │  │ sliceutil│  │ typeutil │  │ semverutil │  │  tty  │  │ styles │         │
│  └────────────┘  └──────────┘  └──────────┘  └────────────┘  └───────┘  └────────┘         │
│  ┌──────────┐  ┌──────────┐                                                                  │
│  │ timeutil │  │ testutil │   (consumed by all core packages above)                          │
│  └──────────┘  └──────────┘                                                                  │
└────────────────────────────────────────────────────────────────────────────────────────────┘
```

## Package Reference

| Package | Layer | Description |
|---------|-------|-------------|
| cmd/gh-aw | Entry | Main CLI binary — imports cli, console, constants, parser, workflow |
| cmd/gh-aw-wasm | Entry | WebAssembly compilation target — imports parser, workflow |
| pkg/cli | Core | Command dispatch, flag handling, and CLI command execution |
| pkg/workflow | Core | Workflow compilation and GitHub Actions YAML generation |
| pkg/parser | Core | Markdown/YAML/frontmatter parsing and schema validation |
| pkg/console | Core | Terminal UI: spinners, message formatting, styled output rendering |
| pkg/agentdrain | Core | Agent log streaming and drain for CI workflows |
| pkg/actionpins | Core | GitHub Actions pin version resolution |
| pkg/stats | Core | Numerical statistics and metrics collection |
| pkg/constants | Utility | Shared constants and semantic type aliases |
| pkg/types | Utility | Shared type definitions used across packages |
| pkg/logger | Utility | Namespace-based debug logging with zero overhead when disabled |
| pkg/fileutil | Utility | File path and file operation utilities |
| pkg/gitutil | Utility | Git repository utility functions |
| pkg/repoutil | Utility | GitHub repository slug and URL utilities |
| pkg/envutil | Utility | Environment variable reading and validation utilities |
| pkg/stringutil | Utility | String manipulation utilities including ANSI stripping |
| pkg/sliceutil | Utility | Generic slice operation utilities |
| pkg/typeutil | Utility | General-purpose type conversion utilities |
| pkg/semverutil | Utility | Shared semantic versioning primitives |
| pkg/tty | Utility | TTY (terminal) detection utilities |
| pkg/styles | Utility | Terminal style definitions (no-op for WASM builds) |
| pkg/timeutil | Utility | Time formatting and duration utilities |
| pkg/testutil | Utility | Shared test helper utilities |
