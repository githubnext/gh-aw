# Architecture Diagram

> Last updated: 2026-04-19 · Source: [Run §24625358414](https://github.com/github/gh-aw/actions/runs/24625358414)

## Overview

This diagram shows the package structure and dependencies of the `gh-aw` codebase.

```
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│  ENTRY POINTS                                                                                    │
│                                                                                                  │
│       ┌─────────────────────────┐                    ┌───────────────────────────┐              │
│       │       cmd/gh-aw         │                    │     cmd/gh-aw-wasm         │              │
│       │     (main CLI binary)   │                    │    (WebAssembly target)    │              │
│       └─────────────┬───────────┘                    └─────────────┬─────────────┘              │
│                     │                                               │                            │
├─────────────────────┼───────────────────────────────────────────────┼────────────────────────────┤
│  CORE PACKAGES      ▼                                               ▼                            │
│                                                                                                  │
│  ┌─────────────────────────┐    ┌─────────────────────────┐    ┌──────────────────────────┐   │
│  │        pkg/cli           │───▶│      pkg/workflow        │───▶│       pkg/parser          │   │
│  │  Command implementations │    │  Workflow compile engine  │    │  Markdown/YAML parsing   │   │
│  └──────────┬───────────────┘    └───────────┬─────────────┘    └──────────────────────────┘   │
│             │                                │                                                   │
│  ┌──────────▼──────────────┐    ┌────────────▼────────────┐                                    │
│  │     pkg/agentdrain      │    │     pkg/actionpins       │                                    │
│  │  Agent log drain/cluster│    │  Action pin resolution   │                                    │
│  └─────────────────────────┘    └─────────────────────────┘                                    │
│             │                                │                                                   │
│             └──────────────────┬─────────────┘                                                   │
│                                ▼                                                                  │
│             ┌──────────────────────────────────────────────┐                                    │
│             │               pkg/console                     │                                    │
│             │       Terminal UI & message formatting         │                                    │
│             └──────────────────────────────────────────────┘                                    │
│                                │                                                                  │
├────────────────────────────────┼──────────────────────────────────────────────────────────────────┤
│  UTILITY PACKAGES              ▼                                                                 │
│                                                                                                  │
│  Rendering:  ┌──────────┐  ┌──────────┐  ┌──────────┐                                         │
│              │  logger  │  │  styles  │  │   tty    │                                         │
│              └──────────┘  └──────────┘  └──────────┘                                         │
│                                                                                                  │
│  Data:       ┌──────────┐  ┌──────────┐  ┌───────────┐  ┌───────────┐  ┌─────────┐           │
│              │  types   │  │ typeutil │  │ constants  │  │ sliceutil │  │  stats  │           │
│              └──────────┘  └──────────┘  └───────────┘  └───────────┘  └─────────┘           │
│                                                                                                  │
│  Files/Git:  ┌──────────┐  ┌──────────┐  ┌───────────┐  ┌───────────┐  ┌───────────┐        │
│              │ fileutil │  │ gitutil  │  │ repoutil  │  │ semverutil│  │ stringutil│        │
│              └──────────┘  └──────────┘  └───────────┘  └───────────┘  └───────────┘        │
│                                                                                                  │
│  Others:     ┌──────────┐  ┌──────────┐  ┌───────────┐                                        │
│              │ envutil  │  │ timeutil │  │ testutil  │                                        │
│              └──────────┘  └──────────┘  └───────────┘                                        │
└──────────────────────────────────────────────────────────────────────────────────────────────────┘
```

## Package Reference

| Package | Layer | Description |
|---------|-------|-------------|
| cmd/gh-aw | Entry | Main CLI binary — imports cli, console, constants, parser, workflow |
| cmd/gh-aw-wasm | Entry | WebAssembly compilation target — imports parser, workflow |
| pkg/cli | Core | Command implementations for all gh-aw CLI commands |
| pkg/workflow | Core | Workflow compilation engine (markdown → GitHub Actions YAML) |
| pkg/parser | Core | Markdown frontmatter parsing and content extraction |
| pkg/console | Core | Terminal UI rendering and message formatting |
| pkg/agentdrain | Core | Agent log drain and cluster template detection |
| pkg/actionpins | Core | GitHub Actions version pin resolution |
| pkg/logger | Utility | Namespace-based debug logging with zero overhead |
| pkg/styles | Utility | Centralized style and color definitions for terminal output |
| pkg/tty | Utility | TTY (terminal) detection utilities |
| pkg/types | Utility | Shared type definitions used across gh-aw packages |
| pkg/typeutil | Utility | General-purpose type conversion utilities |
| pkg/constants | Utility | Shared constants and semantic type aliases |
| pkg/sliceutil | Utility | Utility functions for working with slices |
| pkg/stats | Utility | Numerical statistics utilities for metric collection |
| pkg/fileutil | Utility | File path and file operation utilities |
| pkg/gitutil | Utility | Git and GitHub API utility functions |
| pkg/stringutil | Utility | String manipulation utility functions |
| pkg/repoutil | Utility | GitHub repository slug and URL utilities |
| pkg/semverutil | Utility | Semantic versioning primitives |
| pkg/envutil | Utility | Environment variable reading and validation utilities |
| pkg/timeutil | Utility | Duration and time formatting utilities |
| pkg/testutil | Utility | Shared test helper utilities |
