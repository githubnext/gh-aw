# Architecture Diagram

> Last updated: 2026-04-27 · Source: [🏗️ Architecture Diagram: Full rebuild — gh-aw package architecture (2026-04-27)](https://github.com/github/gh-aw/issues)

## Overview

This diagram shows the package structure and dependencies of the `gh-aw` codebase.

```
┌──────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                         ENTRY POINTS                                                 │
│                                                                                                      │
│               ┌────────────────────────────┐          ┌──────────────────┐                          │
│               │        cmd/gh-aw           │          │  cmd/gh-aw-wasm  │                          │
│               │  Main CLI binary / cobra   │          │  WebAssembly target                         │
│               └──────┬─────────────────────┘          └──────────────────┘                          │
│                      │ imports: cli, workflow, parser, console, constants                            │
└──────────────────────┼───────────────────────────────────────────────────────────────────────────────┘
                       │
┌──────────────────────▼───────────────────────────────────────────────────────────────────────────────┐
│                                        CORE PACKAGES                                                 │
│                                                                                                      │
│  ┌──────────────────────────────┐    ┌────────────────────────────────┐                             │
│  │           pkg/cli            │    │         pkg/workflow            │                             │
│  │  Command implementations     │───▶│  Markdown → GH Actions YAML   │                             │
│  │  compile, run, audit, mcp,   │    │  compiler, engine config,      │                             │
│  │  logs, stats                 │    │  MCP, tools, expressions       │                             │
│  └──────────────┬───────────────┘    └────────────────┬───────────────┘                             │
│                 │                                      │                                             │
│                 │         ┌───────────────────────────▼───────────────────────┐                     │
│                 │         │                  pkg/parser                        │                     │
│                 └────────▶│  Markdown frontmatter parsing, YAML schema valid. │                     │
│                           └───────────────────────────┬───────────────────────┘                     │
│                                                       │                                              │
│                 ┌─────────────────────────────────────▼──────────────┐                              │
│                 │                 pkg/console                          │                              │
│                 │  Terminal UI formatting: success/error/info/warning │                              │
│                 └────────────────────────────────────────────────────┘                              │
│                                                                                                      │
└──────────────────────────────────────────────────────────────────────────────────────────────────────┘
                       │ (all core packages import utilities below)
┌──────────────────────▼───────────────────────────────────────────────────────────────────────────────┐
│                                      UTILITY PACKAGES                                                │
│                                                                                                      │
│  ┌─────────┐ ┌──────────┐ ┌────────────┐ ┌──────────┐ ┌──────────┐ ┌─────────┐ ┌──────────┐      │
│  │ logger  │ │constants │ │ stringutil │ │ sliceutil│ │  gitutil │ │fileutil │ │  types   │      │
│  │ (debug) │ │(typed K) │ │  (string)  │ │ (slices) │ │  (git)   │ │ (files) │ │ (shared) │      │
│  └─────────┘ └──────────┘ └────────────┘ └──────────┘ └──────────┘ └─────────┘ └──────────┘      │
│                                                                                                      │
│  ┌─────────┐ ┌──────────┐ ┌────────────┐ ┌──────────┐ ┌──────────┐ ┌─────────┐ ┌──────────┐      │
│  │ styles  │ │ typeutil │ │   timeutil │ │   tty    │ │semverutil│ │  stats  │ │ repoutil │      │
│  │(colors) │ │ (cast)   │ │  (timing)  │ │ (detect) │ │ (semver) │ │(metrics)│ │ (repos)  │      │
│  └─────────┘ └──────────┘ └────────────┘ └──────────┘ └──────────┘ └─────────┘ └──────────┘      │
│                                                                                                      │
│  ┌───────────────┐  ┌─────────────┐  ┌──────────┐  ┌──────────┐                                   │
│  │  actionpins   │  │  agentdrain │  │  envutil │  │ testutil │                                   │
│  │ (pin resolve) │  │ (agent I/O) │  │  (env)   │  │ (testing)│                                   │
│  └───────────────┘  └─────────────┘  └──────────┘  └──────────┘                                   │
└──────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

## Package Reference

| Package | Layer | Description |
|---------|-------|-------------|
| `cmd/gh-aw` | Entry | Main CLI binary (cobra root command) |
| `cmd/gh-aw-wasm` | Entry | WebAssembly compilation target |
| `pkg/cli` | Core | Command implementations: compile, run, audit, mcp, logs, stats |
| `pkg/workflow` | Core | Markdown → GitHub Actions YAML compilation engine |
| `pkg/parser` | Core | Markdown frontmatter parsing and YAML schema validation |
| `pkg/console` | Core | Terminal UI formatting (success/error/info/warning/progress) |
| `pkg/logger` | Utility | Namespace-based debug logging with zero overhead (531 imports) |
| `pkg/constants` | Utility | Shared typed constants: engines, versions, feature flags (221) |
| `pkg/stringutil` | Utility | String manipulation utilities (193 imports) |
| `pkg/sliceutil` | Utility | Slice manipulation utilities (37 imports) |
| `pkg/gitutil` | Utility | Git operation utilities (33 imports) |
| `pkg/fileutil` | Utility | File path and file operation utilities (21 imports) |
| `pkg/types` | Utility | Shared type definitions (20 imports) |
| `pkg/styles` | Utility | Centralized style and color definitions (18 imports) |
| `pkg/timeutil` | Utility | Time utilities (17 imports) |
| `pkg/typeutil` | Utility | General-purpose type conversion utilities (14 imports) |
| `pkg/tty` | Utility | TTY terminal detection utilities (13 imports) |
| `pkg/semverutil` | Utility | Semantic versioning primitives (11 imports) |
| `pkg/stats` | Utility | Numerical statistics utilities (3 imports) |
| `pkg/repoutil` | Utility | GitHub repository slug and URL utilities (3 imports) |
| `pkg/agentdrain` | Utility | Agent drain I/O utilities (3 imports) |
| `pkg/actionpins` | Utility | GitHub Actions pin resolution (3 imports) |
| `pkg/envutil` | Utility | Environment variable reading and validation (1 import) |
| `pkg/testutil` | Utility | Test helper utilities (test-only) |
