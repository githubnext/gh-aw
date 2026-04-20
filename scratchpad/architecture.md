# Architecture Diagram

> Last updated: 2026-04-20 · Source: [Issue #27296](https://github.com/github/gh-aw/issues) · [Run §24658621102](https://github.com/github/gh-aw/actions/runs/24658621102)

## Overview

This diagram shows the package structure and dependencies of the `gh-aw` codebase.

```
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                       ENTRY POINTS                                               │
│                                                                                                  │
│  ┌──────────────────────────┐                         ┌─────────────────────────────┐            │
│  │       cmd/gh-aw          │                         │       cmd/gh-aw-wasm        │            │
│  │    (main CLI binary)     │                         │      (WebAssembly target)   │            │
│  └───┬────────┬─────────────┘                         └──────────┬─────────┬────────┘            │
│      │        │                                                   │         │                    │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│      ▼        ▼                    CORE PACKAGES                 ▼         ▼                     │
│                                                                                                  │
│  ┌──────────────────┐   ┌───────────────────────┐   ┌────────────────────────────┐               │
│  │    pkg/cli       │──▶│    pkg/workflow        │──▶│      pkg/parser            │              │
│  │  CLI commands &  │   │  workflow compilation  │   │  markdown/YAML frontmatter │              │
│  │  dispatch layer  │   │  & Actions generation  │   │  parsing & validation      │              │
│  └──────┬───────────┘   └──────┬─────────────────┘   └────────────────────────────┘              │
│         │                      │                                                                 │
│         └─────────────┐  ┌─────┘                                                                 │
│                       ▼  ▼                                                                       │
│             ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐                     │
│             │   pkg/console    │  │  pkg/agentdrain  │  │  pkg/actionpins  │                     │
│             │  terminal UI &   │  │  log streaming & │  │  action pin      │                     │
│             │  spinner/render  │  │  drain mgmt      │  │  resolution      │                     │
│             └──────────────────┘  └──────────────────┘  └──────────────────┘                     │
│                          also shared: pkg/stats · pkg/types · pkg/constants                      │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│                                  UTILITY PACKAGES                                                │
│                                                                                                  │
│  ┌────────┐ ┌────────┐ ┌─────┐ ┌──────────┐ ┌─────────┐ ┌──────────┐ ┌──────────┐            │
│  │ logger │ │ styles │ │ tty │ │ fileutil │ │ gitutil │ │ repoutil │ │ sliceutil│            │
│  └────────┘ └────────┘ └─────┘ └──────────┘ └─────────┘ └──────────┘ └──────────┘            │
│                                                                                                  │
│  ┌──────────┐ ┌─────────┐ ┌───────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐              │
│  │ typeutil │ │ envutil │ │ semverutil│ │stringutil│ │ timeutil │ │ testutil │              │
│  └──────────┘ └─────────┘ └───────────┘ └──────────┘ └──────────┘ └──────────┘              │
│                                                                                                  │
└──────────────────────────────────────────────────────────────────────────────────────────────────┘
```

## Package Reference

| Package | Layer | Description |
|---------|-------|-------------|
| cmd/gh-aw | Entry | Main CLI binary — imports cli, console, constants, parser, workflow |
| cmd/gh-aw-wasm | Entry | WebAssembly compilation target — imports parser, workflow |
| pkg/cli | Core | CLI command implementations and dispatch layer |
| pkg/workflow | Core | Workflow compilation and GitHub Actions YAML generation |
| pkg/parser | Core | Markdown/YAML frontmatter parsing and validation |
| pkg/console | Core | Terminal UI rendering, spinners, and formatted output |
| pkg/agentdrain | Core | Agent log streaming and drain management |
| pkg/actionpins | Core | GitHub Actions pin resolution |
| pkg/stats | Shared | Numerical statistics utilities for metric collection |
| pkg/types | Shared | Shared type definitions used across packages |
| pkg/constants | Shared | Shared constants and semantic type aliases |
| pkg/logger | Utility | Namespace-based debug logging with zero overhead |
| pkg/styles | Utility | Terminal style definitions (no-op for Wasm) |
| pkg/tty | Utility | TTY terminal detection utilities |
| pkg/fileutil | Utility | File path and file operation utilities |
| pkg/gitutil | Utility | Git operation utilities |
| pkg/repoutil | Utility | GitHub repository slug and URL utilities |
| pkg/sliceutil | Utility | Generic slice utilities |
| pkg/typeutil | Utility | General-purpose type conversion utilities |
| pkg/envutil | Utility | Environment variable reading and validation |
| pkg/semverutil | Utility | Shared semantic versioning primitives |
| pkg/stringutil | Utility | String manipulation utilities |
| pkg/timeutil | Utility | Time-related utilities |
| pkg/testutil | Utility | Test helper utilities |
