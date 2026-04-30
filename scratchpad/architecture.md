# Architecture Diagram

> Last updated: 2026-04-30 · Source: [🏗️ Architecture Diagram: Full rebuild — gh-aw package architecture (2026-04-30)](https://github.com/github/gh-aw/issues)

## Overview

This diagram shows the package structure and dependencies of the `gh-aw` codebase.

```
┌────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                        ENTRY POINTS                                            │
│   ┌─────────────────────────────┐              ┌───────────────────────────────┐              │
│   │        cmd/gh-aw            │              │       cmd/gh-aw-wasm          │              │
│   │  Main CLI binary / cobra    │              │   WebAssembly compilation     │              │
│   └──────────────┬──────────────┘              └───────────────────────────────┘              │
│                  │ imports: cli, workflow, parser, console, constants                           │
├──────────────────┼─────────────────────────────────────────────────────────────────────────────┤
│                  ▼                       CORE PACKAGES                                         │
│   ┌─────────────────────┐   ┌──────────────────────┐   ┌──────────────────────┐              │
│   │       pkg/cli        │──▶│    pkg/workflow       │──▶│     pkg/parser       │              │
│   │  Command impls for   │   │  Workflow compilation  │   │  Markdown frontmatter│              │
│   │  compile/run/logs    │   │  engine (MD→YAML)     │   │  parsing & extraction│              │
│   └──────┬───────────────┘   └────────┬─────────────┘   └──────────────────────┘              │
│          │                            │                                                         │
│          │                   ┌────────▼─────────────┐   ┌──────────────────────┐              │
│          │                   │    pkg/actionpins     │   │    pkg/agentdrain    │              │
│          │                   │  Action pin resolution│   │  Agent output drain  │              │
│          └──────────────────▶│  (SHA pinning for CI) │   │  (stream multiplexer)│              │
│                              └───────────────────────┘   └──────────────────────┘              │
│                                                                                                 │
│   ┌───────────────────────┐   ┌──────────────────────┐   ┌──────────────────────┐             │
│   │     pkg/console       │   │    pkg/constants      │   │     pkg/stats        │             │
│   │  Terminal UI formatting│   │  Shared typed consts  │   │  Numerical statistics│             │
│   │  (success/warn/error) │   │  (engines, versions)  │   │  for metric tracking │             │
│   └───────────────────────┘   └──────────────────────┘   └──────────────────────┘             │
├─────────────────────────────────────────────────────────────────────────────────────────────────┤
│                                     UTILITY PACKAGES                                            │
│  ┌────────┐ ┌────────┐ ┌────────┐ ┌──────────┐ ┌─────────┐ ┌──────────┐ ┌────────────┐    │
│  │ logger │ │  tty   │ │ styles │ │ fileutil │ │ gitutil │ │stringutil│ │ sliceutil  │    │
│  └────────┘ └────────┘ └────────┘ └──────────┘ └─────────┘ └──────────┘ └────────────┘    │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐             │
│  │ repoutil │ │ envutil  │ │ timeutil │ │semverutil│ │ typeutil │ │  types   │             │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘ └──────────┘ └──────────┘             │
└─────────────────────────────────────────────────────────────────────────────────────────────────┘
```

## Package Reference

| Package | Layer | Description |
|---------|-------|-------------|
| cli | Core | Command implementations for compile/run/logs/audit |
| workflow | Core | Workflow compilation engine (Markdown → GitHub Actions YAML) |
| parser | Core | Markdown frontmatter parsing and content extraction |
| actionpins | Core | GitHub Actions pin resolution (SHA pinning for CI) |
| agentdrain | Core | Agent output drain / stream multiplexer |
| console | Core | Terminal UI formatting (success/warn/error messages) |
| constants | Core | Shared typed constants (engines, versions, job names) |
| stats | Core | Numerical statistics for metric tracking |
| logger | Utility | Namespace-based debug logging with zero overhead |
| tty | Utility | TTY (terminal) detection utilities |
| styles | Utility | Centralized style and color definitions for terminal output |
| fileutil | Utility | File path and file operation utilities |
| gitutil | Utility | Git repository utilities |
| stringutil | Utility | String manipulation utilities |
| sliceutil | Utility | Slice operation utilities |
| repoutil | Utility | GitHub repository slug and URL utilities |
| envutil | Utility | Environment variable reading and validation |
| timeutil | Utility | Time utilities |
| semverutil | Utility | Shared semantic versioning primitives |
| typeutil | Utility | General-purpose type conversion utilities |
| types | Utility | Shared type definitions used across packages |
