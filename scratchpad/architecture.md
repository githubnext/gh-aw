# Architecture Diagram

> Last updated: 2026-04-22 · Source: [Issue #27791](https://github.com/github/gh-aw/issues)

## Overview

This diagram shows the package structure and dependencies of the `gh-aw` codebase.

```
┌──────────────────────────────────────────────────────────────────────────────────────────────┐
│                                      ENTRY POINTS                                            │
│    ┌──────────────────────────────┐              ┌──────────────────────────────┐            │
│    │         cmd/gh-aw            │              │       cmd/gh-aw-wasm         │            │
│    │   Main CLI binary            │              │   WebAssembly / JS target    │            │
│    └──────────────────────┬───────┘              └────────┬─────────────────────┘            │
│       (cli,workflow,parser,│console,constants)  (parser,  │workflow)                         │
├───────────────────────────▼──────────────────────────────▼──────────────────────────────────┤
│                                      CORE PACKAGES                                           │
│                                                                                              │
│  ┌─────────────────────┐    ┌──────────────────────────┐    ┌──────────────────────┐        │
│  │      pkg/cli         │──▶│      pkg/workflow          │──▶│      pkg/parser       │        │
│  │  Command routing &   │   │  MD→YAML compilation       │   │  Markdown/YAML       │        │
│  │  implementations     │   │  engine                    │   │  frontmatter parsing │        │
│  └──────────┬───────────┘   └──────────┬──────┬──────────┘   └──────────┬───────────┘        │
│             │                          │      │                          │                   │
│             │                          │      └──▶ pkg/actionpins        │                   │
│             │                          │           (Action SHA pinning)  │                   │
│             ▼                          ▼                                 ▼                   │
│          ┌──────────────────────────────────────────────────────────────────┐               │
│          │                       pkg/console                                │               │
│          │          Terminal UI rendering & message formatting               │               │
│          └──────────────┬──────────────────┬───────────────────┬────────────┘               │
│                         ▼                  ▼                   ▼                            │
│                    pkg/styles          pkg/tty            pkg/logger                        │
│                  (color/style      (TTY detection)    (debug logging)                       │
│                   definitions)                                                               │
│                                                                                              │
│  ┌─────────────────────┐    ┌──────────────────────────┐    ┌──────────────────────┐        │
│  │   pkg/agentdrain    │    │      pkg/constants        │    │      pkg/types        │        │
│  │  Agent output       │    │  Semantic type aliases    │    │  Shared type defs    │        │
│  │  drain & streaming  │    │  & shared constants       │    │                      │        │
│  └─────────────────────┘    └──────────────────────────┘    └──────────────────────┘        │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│                                  UTILITY PACKAGES                                            │
│                                                                                              │
│  ┌─────────┐  ┌─────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐         │
│  │fileutil │  │ gitutil │  │stringutil│  │ repoutil │  │semverutil│  │sliceutil │         │
│  └─────────┘  └─────────┘  └──────────┘  └──────────┘  └──────────┘  └──────────┘         │
│  ┌─────────┐  ┌─────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐                       │
│  │ typeutil│  │ envutil │  │ timeutil │  │  stats   │  │ testutil │                       │
│  └─────────┘  └─────────┘  └──────────┘  └──────────┘  └──────────┘                       │
└──────────────────────────────────────────────────────────────────────────────────────────────┘
```

## Package Reference

| Package | Layer | Description |
|---------|-------|-------------|
| cmd/gh-aw | Entry | Main CLI binary — imports cli, console, constants, parser, workflow |
| cmd/gh-aw-wasm | Entry | WebAssembly / JS compilation target — imports parser, workflow |
| pkg/cli | Core | Command routing & implementations |
| pkg/workflow | Core | MD→YAML workflow compilation engine |
| pkg/parser | Core | Markdown/YAML frontmatter parsing & extraction |
| pkg/console | Core | Terminal UI rendering & message formatting |
| pkg/actionpins | Core | GitHub Actions pin resolution (SHA pinning) |
| pkg/agentdrain | Core | Agent output drain & streaming |
| pkg/constants | Core | Semantic type aliases & shared constants |
| pkg/types | Core | Shared type definitions |
| pkg/fileutil | Utility | File path & file operation utilities |
| pkg/gitutil | Utility | Git operation utilities |
| pkg/stringutil | Utility | String manipulation utilities |
| pkg/logger | Utility | Namespace-based debug logging with zero overhead |
| pkg/repoutil | Utility | GitHub repository slug & URL utilities |
| pkg/semverutil | Utility | Semantic versioning primitives |
| pkg/sliceutil | Utility | Slice operation utilities |
| pkg/styles | Utility | Centralized style & color definitions for terminal |
| pkg/tty | Utility | TTY (terminal) detection utilities |
| pkg/typeutil | Utility | General-purpose type conversion utilities |
| pkg/envutil | Utility | Environment variable reading & validation utilities |
| pkg/timeutil | Utility | Time-related utilities |
| pkg/stats | Utility | Numerical statistics utilities for metric collection |
| pkg/testutil | Utility | Test helper utilities (test-only) |
