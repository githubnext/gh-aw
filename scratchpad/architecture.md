# Architecture Diagram

> Last updated: 2026-04-26 · Source: [Issue #🏗️ Architecture Diagram: gh-aw Package Architecture Diagram (2026-04-26)](https://github.com/github/gh-aw/issues)

## Overview

This diagram shows the package structure and dependencies of the `gh-aw` codebase.

```
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                        ENTRY POINTS                                              │
│                                                                                                  │
│            ┌──────────────────────┐               ┌──────────────────────────┐                 │
│            │     cmd/gh-aw        │               │    cmd/gh-aw-wasm        │                 │
│            │  Main CLI binary     │               │  WebAssembly target      │                 │
│            └──────────┬───────────┘               └────────────┬─────────────┘                 │
│                       │ cli,console,constants,                  │ parser,workflow                │
│                       │ parser,workflow                         │                               │
└───────────────────────┼─────────────────────────────────────────┼───────────────────────────────┘
                        │                                         │
┌───────────────────────▼─────────────────────────────────────────▼───────────────────────────────┐
│                                       CORE PACKAGES                                              │
│                                                                                                  │
│  ┌───────────────────────────────────────────────────────────────────────────────────────────┐  │
│  │  cli  ·  Command implementations: compile, run, mcp, logs, audit, campaigns, validate…    │  │
│  └──────┬───────────────────────────────────────────────────────────────────────┬────────────┘  │
│         │                                                                        │               │
│         ▼                                                                        ▼               │
│  ┌──────────────────────────────────────────┐  ┌────────────────────────────────────────────┐  │
│  │  workflow  ·  Workflow compilation engine │  │  parser  ·  Markdown frontmatter parsing   │  │
│  │  (MD → GitHub Actions YAML, MCP config)  │  │  and content/expression extraction         │  │
│  └──────┬───────────┬──────────────────────┘  └────────────────────────────────┬───────────┘  │
│         │           │                                                            │               │
│         ▼           ▼                                                            ▼               │
│  ┌─────────────┐  ┌─────────────────────────┐  ┌──────────────────────────────────────────┐   │
│  │ actionpins  │  │      agentdrain          │  │  console  ·  Terminal UI, formatting,    │   │
│  │ Action pin  │  │  Agent output streaming  │  │  struct-tag rendering, message types     │   │
│  │ resolution  │  │  and draining            │  └──────────────────────────────────────────┘   │
│  └─────────────┘  └─────────────────────────┘                                                  │
│                                                                                                  │
│  ┌────────────────────────────────────────────────┐                                             │
│  │  stats  ·  Numerical statistics for metrics    │                                             │
│  └────────────────────────────────────────────────┘                                             │
└──────────────────────────────────────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                       UTILITY PACKAGES                                           │
│                                                                                                  │
│  ┌───────────┐  ┌───────────┐  ┌───────────┐  ┌───────────┐  ┌───────────┐  ┌───────────┐    │
│  │ constants │  │  fileutil │  │  gitutil  │  │  logger   │  │ stringutil│  │  styles   │    │
│  └───────────┘  └───────────┘  └───────────┘  └───────────┘  └───────────┘  └───────────┘    │
│                                                                                                  │
│  ┌───────────┐  ┌───────────┐  ┌───────────┐  ┌───────────┐  ┌───────────┐  ┌───────────┐    │
│  │  envutil  │  │ repoutil  │  │ semverutil│  │ sliceutil │  │ timeutil  │  │   types   │    │
│  └───────────┘  └───────────┘  └───────────┘  └───────────┘  └───────────┘  └───────────┘    │
│                                                                                                  │
│  ┌───────────┐  ┌───────────┐  ┌───────────┐                                                   │
│  │ typeutil  │  │    tty    │  │  testutil │                                                    │
│  └───────────┘  └───────────┘  └───────────┘                                                   │
└──────────────────────────────────────────────────────────────────────────────────────────────────┘
```

## Package Reference

| Package | Layer | Description |
|---------|-------|-------------|
| cli | Core | Command implementations: compile, run, mcp, logs, audit, campaigns, validate |
| workflow | Core | Workflow compilation engine (MD → GitHub Actions YAML, MCP config) |
| parser | Core | Markdown frontmatter parsing and content/expression extraction |
| console | Core | Terminal UI, formatting, struct-tag rendering, message types |
| actionpins | Core | GitHub Actions pin resolution |
| agentdrain | Core | Agent output streaming and draining |
| stats | Core | Numerical statistics for metric collection |
| constants | Utility | Shared constants and semantic type aliases |
| envutil | Utility | Environment variable reading and validation utilities |
| fileutil | Utility | File path and file operation utilities |
| gitutil | Utility | Git repository utilities |
| logger | Utility | Namespace-based debug logging with zero overhead |
| repoutil | Utility | GitHub repository slug and URL utilities |
| semverutil | Utility | Semantic versioning primitives |
| sliceutil | Utility | Slice utility functions |
| stringutil | Utility | String utility functions including ANSI stripping |
| styles | Utility | Centralized style and color definitions for terminal output |
| testutil | Utility | Test helper utilities |
| timeutil | Utility | Time utility functions |
| tty | Utility | TTY detection utilities |
| types | Utility | Shared type definitions used across gh-aw packages |
| typeutil | Utility | General-purpose type conversion utilities |
