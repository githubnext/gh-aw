# Architecture Diagram

> Last updated: 2026-04-21 · [Run §24714109908](https://github.com/github/gh-aw/actions/runs/24714109908)

## Overview

This diagram shows the package structure and dependencies of the `gh-aw` codebase.

```
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                       ENTRY POINTS                                               │
│  ┌──────────────────────────┐                         ┌─────────────────────────────┐            │
│  │       cmd/gh-aw          │                         │       cmd/gh-aw-wasm        │            │
│  │    (main CLI binary)     │                         │      (WebAssembly target)   │            │
│  └───┬──────────────────────┘                         └──────────┬──────────────────┘            │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│      ▼                         CORE PACKAGES                     ▼                               │
│  ┌──────────────────┐   ┌───────────────────────┐   ┌──────────────────────────────┐             │
│  │    pkg/cli       │──▶│    pkg/workflow        │──▶│         pkg/parser           │             │
│  │  CLI commands &  │   │  workflow compilation  │   │  markdown/YAML frontmatter   │             │
│  │  dispatch layer  │   │  & Actions generation  │   │  parsing & validation        │             │
│  └──────┬───────────┘   └──────┬─────────────────┘   └──────────────────────────────┘             │
│         └─────────────┐  ┌─────┘                                                                 │
│                       ▼  ▼                                                                       │
│  ┌─────────────────┐  ┌──────────────────────┐  ┌──────────────────────┐  ┌─────────────────┐   │
│  │  pkg/console    │  │   pkg/agentdrain     │  │   pkg/actionpins    │  │  pkg/constants  │   │
│  │ terminal UI &   │  │ Drain log anomaly    │  │ action pin          │  │ constants &     │   │
│  │ formatted output│  │ detection            │  │ resolution          │  │ type aliases    │   │
│  └─────────────────┘  └──────────────────────┘  └──────────────────────┘  └─────────────────┘   │
│                      also shared: pkg/stats · pkg/types · pkg/semverutil                         │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│                                  UTILITY PACKAGES                                                │
│  ┌────────┐ ┌────────┐ ┌─────┐ ┌──────────┐ ┌─────────┐ ┌──────────┐ ┌──────────┐            │
│  │ logger │ │ styles │ │ tty │ │ fileutil │ │ gitutil │ │ repoutil │ │ sliceutil│            │
│  └────────┘ └────────┘ └─────┘ └──────────┘ └─────────┘ └──────────┘ └──────────┘            │
│  ┌──────────┐ ┌─────────┐ ┌───────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐              │
│  │ typeutil │ │ envutil │ │ semverutil│ │stringutil│ │ timeutil │ │ testutil*│              │
│  └──────────┘ └─────────┘ └───────────┘ └──────────┘ └──────────┘ └──────────┘              │
│  (* test-only: pkg/testutil)                                                                     │
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
| pkg/console | Core | Terminal UI rendering and formatted output |
| pkg/agentdrain | Core | Drain log-based anomaly detection for agentic pipeline runs |
| pkg/actionpins | Core | GitHub Actions pin resolution |
| pkg/constants | Shared | Shared constants and semantic type aliases |
| pkg/types | Shared | Shared type definitions used across packages |
| pkg/stats | Shared | Numerical statistics utilities for metric collection |
| pkg/semverutil | Utility | Semantic versioning primitives |
| pkg/logger | Utility | Namespace-based debug logging with zero overhead |
| pkg/styles | Utility | Terminal style and color definitions |
| pkg/tty | Utility | TTY terminal detection utilities |
| pkg/fileutil | Utility | File path and file operation utilities |
| pkg/gitutil | Utility | Git operation utilities |
| pkg/repoutil | Utility | GitHub repository slug and URL utilities |
| pkg/sliceutil | Utility | Generic slice utilities |
| pkg/typeutil | Utility | General-purpose type conversion utilities |
| pkg/envutil | Utility | Environment variable reading and validation |
| pkg/stringutil | Utility | String manipulation utilities |
| pkg/timeutil | Utility | Time-related utilities |
| pkg/testutil | Utility | Test helper utilities (test-only) |
