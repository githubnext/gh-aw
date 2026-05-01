# Architecture Diagram

> Last updated: 2026-05-01 · Source: [Architecture Diagram Issue](https://github.com/github/gh-aw/issues)

## Overview

This diagram shows the package structure and dependencies of the `gh-aw` codebase, organized in three layers: entry points, core packages, and utility packages.

```
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                        ENTRY POINTS                                              │
│                                                                                                  │
│          ┌─────────────────────────────────┐       ┌──────────────────────────────────┐         │
│          │           cmd/gh-aw             │       │         cmd/gh-aw-wasm           │         │
│          │  Main CLI binary (gh extension) │       │  WebAssembly compilation target  │         │
│          └────────────┬────────────────────┘       └────────────────────┬─────────────┘         │
│                       │  imports cli, workflow,                         │                        │
│                       │  parser, console, constants                     │                        │
├───────────────────────┼─────────────────────────────────────────────────┼────────────────────────┤
│                       ▼          CORE PACKAGES                          ▼                        │
│                                                                                                  │
│  ┌──────────────────────────┐    ┌──────────────────────────┐    ┌──────────────────────────┐   │
│  │         pkg/cli          │──▶│       pkg/workflow        │──▶│       pkg/parser         │   │
│  │  Command implementations │    │  Workflow compiler engine │    │  Markdown/YAML frontmatter│  │
│  └───────────┬──────────────┘    └──────────┬───────────────┘    └──────────┬───────────────┘  │
│              │                              │                               │                    │
│              │       ┌──────────────────────┘                               │                    │
│              │       │         ┌────────────────────────────────────────────┘                    │
│              ▼       ▼         ▼                                                                  │
│  ┌──────────────────────────┐    ┌──────────────────────────┐    ┌──────────────────────────┐   │
│  │       pkg/console        │    │      pkg/agentdrain      │    │      pkg/actionpins      │   │
│  │  Terminal UI formatting  │    │  Agent output streaming  │    │  Action pin resolution   │   │
│  └───────────┬──────────────┘    └──────────────────────────┘    └──────────────────────────┘  │
│              │                                                                                    │
│  ┌───────────┴──────────────┐    ┌──────────────────────────┐                                   │
│  │        pkg/stats         │    │       pkg/styles         │                                    │
│  │   Numerical statistics   │    │   Terminal color/styles  │                                    │
│  └──────────────────────────┘    └──────────────────────────┘                                   │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│                                     UTILITY PACKAGES                                             │
│                                                                                                  │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌──────────┐  │
│  │pkg/fileutil│  │pkg/gitutil │  │ pkg/logger │  │pkg/stringut│  │pkg/sliceutil  │ pkg/tty  │  │
│  │File path & │  │Git repo    │  │Namespace   │  │String      │  │Slice helper│  │TTY detect│  │
│  │operations  │  │operations  │  │debug logs  │  │utilities   │  │functions   │  │          │  │
│  └────────────┘  └────────────┘  └────────────┘  └────────────┘  └────────────┘  └──────────┘  │
│                                                                                                  │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌──────────┐  │
│  │pkg/constants  │pkg/repoutil│  │pkg/semverut│  │ pkg/types  │  │pkg/typeutil│  │pkg/envutil  │
│  │Shared consts  │Repo slug & │  │Semver      │  │Shared type │  │Type convert│  │Env var   │  │
│  │& type aliases │URL helpers │  │primitives  │  │definitions │  │utilities   │  │utilities │  │
│  └────────────┘  └────────────┘  └────────────┘  └────────────┘  └────────────┘  └──────────┘  │
│                                                                                                  │
│  ┌────────────┐  ┌────────────┐                                                                  │
│  │pkg/timeutil│  │pkg/testutil│                                                                  │
│  │Time helpers│  │Test helpers│                                                                  │
│  └────────────┘  └────────────┘                                                                  │
└──────────────────────────────────────────────────────────────────────────────────────────────────┘
```

## Package Reference

| Package | Layer | Description |
|---------|-------|-------------|
| cmd/gh-aw | Entry Point | Main CLI binary (`gh aw` extension) |
| cmd/gh-aw-wasm | Entry Point | WebAssembly compilation target |
| pkg/cli | Core | Command implementations (compile, run, audit, logs, mcp, etc.) |
| pkg/workflow | Core | Workflow compiler engine — markdown→GitHub Actions YAML |
| pkg/parser | Core | Markdown frontmatter parsing and content extraction |
| pkg/console | Core | Terminal UI formatting (success/info/warning/error messages) |
| pkg/agentdrain | Core | Agent output streaming and draining |
| pkg/actionpins | Core | GitHub Actions pin resolution (SHA pinning) |
| pkg/stats | Core | Numerical statistics utilities for metric collection |
| pkg/styles | Core | Centralized terminal color/style definitions |
| pkg/constants | Utility | Shared constants and semantic type aliases |
| pkg/envutil | Utility | Environment variable reading and validation |
| pkg/fileutil | Utility | File path and file operation utilities |
| pkg/gitutil | Utility | Git repository operations |
| pkg/logger | Utility | Namespace-based debug logging (zero overhead) |
| pkg/repoutil | Utility | GitHub repository slug and URL helpers |
| pkg/semverutil | Utility | Semantic versioning primitives |
| pkg/sliceutil | Utility | Generic slice helper functions |
| pkg/stringutil | Utility | String manipulation utilities |
| pkg/timeutil | Utility | Time helper utilities |
| pkg/tty | Utility | TTY (terminal) detection |
| pkg/types | Utility | Shared type definitions across packages |
| pkg/typeutil | Utility | General-purpose type conversion utilities |
| pkg/testutil | Utility | Test helper utilities (test-only use) |
