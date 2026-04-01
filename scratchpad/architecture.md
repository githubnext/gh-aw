# Architecture Diagram

> Last updated: 2026-03-31 · Source: [Run §23789178746](https://github.com/github/gh-aw/actions/runs/23789178746)

## Overview

This diagram shows the package structure and dependencies of the `gh-aw` codebase.

```
┌─────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                      ENTRY POINTS                                               │
│  ┌──────────────────────────────────────────┐   ┌─────────────────────────────────────────┐   │
│  │              cmd/gh-aw                   │   │           cmd/gh-aw-wasm                │   │
│  │        GitHub CLI extension binary       │   │          WebAssembly target             │   │
│  └──────┬──────────────────────┬────────────┘   └──────────────────┬──────────────────────┘   │
│         │                      │                                    │                           │
├─────────┼──────────────────────┼────────────────────────────────────┼────────────────────────────┤
│         │    CORE PACKAGES     │                                    │                           │
│         ▼                      ▼                                    ▼                           │
│  ┌──────────────┐    ┌──────────────────────┐    ┌──────────────────────┐   ┌────────────────┐ │
│  │   pkg/cli    │──▶ │    pkg/workflow       │──▶ │      pkg/parser      │   │  pkg/console   │ │
│  │  Command     │    │  Workflow compilation │    │  Markdown/YAML       │ ◀─│  Terminal UI   │ │
│  │  impls       │    │  engine               │    │  parsing             │   │  rendering     │ │
│  └──────────────┘    └──────────────────────┘    └──────────────────────┘   └────────────────┘ │
│                                                                                                  │
│                 ↕ all core packages also use constants, types, and utilities ↕                  │
│                                                                                                  │
│                ┌────────────────────────────────────────────────────────────────────────────┐   │
│                │           pkg/constants · pkg/types  (shared primitives, no dependencies)  │   │
│                └────────────────────────────────────────────────────────────────────────────┘   │
├─────────────────────────────────────────────────────────────────────────────────────────────────┤
│                                        UTILITIES                                                 │
│  ┌──────────┐ ┌─────────────┐ ┌────────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐          │
│  │pkg/logger│ │pkg/stringutil│ │pkg/fileutil│ │pkg/gitutil│ │pkg/styles│ │ pkg/tty  │          │
│  └──────────┘ └─────────────┘ └────────────┘ └──────────┘ └──────────┘ └──────────┘          │
│  ┌──────────┐ ┌─────────────┐ ┌────────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐          │
│  │pkg/envutil│ │pkg/repoutil │ │pkg/sliceutil│ │pkg/semver│ │pkg/timeutil│ │pkg/mathutil│       │
│  └──────────┘ └─────────────┘ └────────────┘ └──────────┘ └──────────┘ └──────────┘          │
│  ┌─────────────┐                                                                                │
│  │ pkg/testutil│  (test support only)                                                          │
│  └─────────────┘                                                                                │
└─────────────────────────────────────────────────────────────────────────────────────────────────┘
```

## Package Reference

| Package | Layer | Description |
|---------|-------|-------------|
| `cmd/gh-aw` | Entry | GitHub CLI extension binary |
| `cmd/gh-aw-wasm` | Entry | WebAssembly target |
| `pkg/cli` | Core | Command implementations (cobra commands) |
| `pkg/workflow` | Core | Workflow compilation engine (MD → GitHub Actions YAML) |
| `pkg/parser` | Core | Markdown frontmatter and YAML parsing |
| `pkg/console` | Core | Terminal UI rendering and output formatting |
| `pkg/constants` | Shared | Shared constants and semantic type aliases (no internal deps) |
| `pkg/types` | Shared | Shared type definitions across packages (no internal deps) |
| `pkg/logger` | Utility | Namespace-based debug logging with zero overhead when disabled |
| `pkg/stringutil` | Utility | String manipulation utilities |
| `pkg/fileutil` | Utility | File path and file operation utilities |
| `pkg/gitutil` | Utility | Git repository utilities |
| `pkg/styles` | Utility | Centralized terminal color and style definitions |
| `pkg/tty` | Utility | TTY (terminal) detection utilities |
| `pkg/envutil` | Utility | Environment variable reading and validation |
| `pkg/repoutil` | Utility | GitHub repository slug and URL utilities |
| `pkg/sliceutil` | Utility | Generic slice utility functions |
| `pkg/semverutil` | Utility | Shared semantic versioning primitives |
| `pkg/timeutil` | Utility | Time formatting utilities |
| `pkg/mathutil` | Utility | Basic mathematical utility functions |
| `pkg/testutil` | Utility | Test support utilities (test-only) |
