# Architecture Diagram

> Last updated: 2026-03-01 · Source: [Issue #19025](https://github.com/github/gh-aw/issues/19025)

## Overview

This diagram shows the package structure and dependencies of the `gh-aw` codebase.

```
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                                       ENTRY POINTS                                           │
│  ┌──────────────────────────────────────┐    ┌──────────────────────────────────────────┐  │
│  │            cmd/gh-aw                 │    │           cmd/gh-aw-wasm                 │  │
│  │   Main CLI binary & all commands     │    │         WebAssembly target               │  │
│  └──────────────────┬───────────────────┘    └──────────────────────────┬───────────────┘  │
├─────────────────────┼──────────────────────────────────────────────────┼──────────────────┤
│                     ▼           CORE PACKAGES                          ▼                   │
│  ┌────────────────────────────┐            ┌───────────────────────────────────────────┐   │
│  │        pkg/cli             │            │            pkg/workflow                   │   │
│  │  Command implementations   ├───────────▶│  Workflow compilation engine &            │   │
│  │  and all CLI subcommands   │            │  orchestration                            │   │
│  └──────────────┬─────────────┘            └──────────────────┬────────────────────────┘   │
│                 │                                              │                             │
│                 └────────────────────┬─────────────────────────┘                            │
│                                      ▼                                                       │
│                          ┌──────────────────────────────┐                                   │
│                          │        pkg/parser            │                                   │
│                          │  Markdown frontmatter &      │                                   │
│                          │  YAML parsing                │                                   │
│                          └──────────────┬───────────────┘                                   │
│                                         │                                                    │
│                          ┌──────────────▼───────────────┐                                   │
│                          │        pkg/console           │                                   │
│                          │  Terminal UI & styled output │                                   │
│                          └──────────────────────────────┘                                   │
│                                                                                              │
│           ↕ all core packages also depend on constants, types, and utilities ↕              │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│                                   SHARED DEFINITIONS                                         │
│  ┌─────────────────────────────────────────┐  ┌──────────────────────────────────────────┐  │
│  │             pkg/constants               │  │              pkg/types                   │  │
│  │  Versions, flags, URLs, engine names    │  │  Shared type definitions across packages │  │
│  └─────────────────────────────────────────┘  └──────────────────────────────────────────┘  │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│                                       UTILITIES                                              │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌──────────┐ ┌──────────┐ ┌─────────┐ ┌─────────┐   │
│  │fileutil │ │ gitutil │ │ logger  │ │stringutil│ │ sliceutil│ │repoutil │ │   tty   │   │
│  └─────────┘ └─────────┘ └─────────┘ └──────────┘ └──────────┘ └─────────┘ └─────────┘   │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐  ┌─────────┐                            │
│  │ envutil │ │timeutil │ │mathutil │ │testutil │  │ styles  │                             │
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘  └─────────┘                            │
└──────────────────────────────────────────────────────────────────────────────────────────────┘
```

## Package Reference

| Package | Layer | Description |
|---------|-------|-------------|
| `pkg/cli` | Core | CLI command implementations and subcommands |
| `pkg/workflow` | Core | Workflow compilation engine and orchestration |
| `pkg/parser` | Core | Markdown frontmatter and YAML parsing |
| `pkg/console` | Core | Terminal UI and styled output rendering |
| `pkg/constants` | Shared | Application-wide constants (versions, flags, URLs, engine names) |
| `pkg/types` | Shared | Shared type definitions across packages |
| `pkg/fileutil` | Utility | File path and operation utilities |
| `pkg/gitutil` | Utility | Git repository utilities |
| `pkg/logger` | Utility | Namespace-based debug logging with zero overhead |
| `pkg/stringutil` | Utility | String manipulation utilities |
| `pkg/sliceutil` | Utility | Slice manipulation utilities |
| `pkg/repoutil` | Utility | GitHub repository slug and URL utilities |
| `pkg/tty` | Utility | TTY detection utilities |
| `pkg/envutil` | Utility | Environment variable reading and validation |
| `pkg/timeutil` | Utility | Time utilities |
| `pkg/mathutil` | Utility | Basic mathematical utility functions |
| `pkg/testutil` | Utility | Testing helper utilities |
| `pkg/styles` | Utility | Centralized terminal style and color definitions |
