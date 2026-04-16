# Architecture Diagram

> Last updated: 2026-04-16 · Source: [Issue #25999](https://github.com/github/gh-aw/issues/25999)

## Overview

This diagram shows the package structure and dependencies of the `gh-aw` codebase.

```
┌────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                         ENTRY POINTS                                               │
│  ┌────────────────────────────┐   ┌────────────────────────┐   ┌────────────────────────────────┐ │
│  │        cmd/gh-aw           │   │    cmd/gh-aw-wasm      │   │     internal/tools/ (×2)       │ │
│  │  GitHub CLI extension bin  │   │   WebAssembly target   │   │  actions-build, gen-metadata   │ │
│  └────────────┬───────────────┘   └────────────┬───────────┘   └──────────────┬─────────────────┘ │
│               │                                 │                              │                   │
├───────────────┼─────────────────────────────────┼──────────────────────────────┼───────────────────┤
│               ▼          CORE PACKAGES           ▼                              ▼                  │
│  ┌──────────────────┐   ┌──────────────────────┐   ┌────────────────────┐   ┌───────────────────┐ │
│  │    pkg/cli       │──▶│    pkg/workflow       │──▶│    pkg/parser      │   │   pkg/console     │ │
│  │  Command impls   │   │  Workflow compilation │   │  MD/YAML parsing   │◀──│  Terminal UI      │ │
│  └──────┬───────────┘   └─────────┬────────────┘   └────────────────────┘   └───────────────────┘ │
│         │                         │                                                                 │
│         ▼                         ▼                                                                 │
│  ┌──────────────────────┐   ┌────────────────────────┐                                            │
│  │    pkg/agentdrain    │   │    pkg/actionpins       │                                            │
│  │  Log template mining │   │  Action pin resolution  │                                            │
│  └──────────────────────┘   └────────────────────────┘                                            │
│                                                                                                     │
│                 ┌──────────────────────────────────────────────────────────────────────────────┐   │
│                 │           pkg/constants · pkg/types   (shared primitives, no deps)           │   │
│                 └──────────────────────────────────────────────────────────────────────────────┘   │
├─────────────────────────────────────────────────────────────────────────────────────────────────────┤
│                                           UTILITIES                                                 │
│  ┌──────────┐ ┌──────────────┐ ┌────────────┐ ┌───────────┐ ┌──────────┐ ┌──────────┐            │
│  │pkg/logger│ │pkg/stringutil│ │pkg/fileutil│ │pkg/gitutil│ │pkg/styles│ │  pkg/tty │            │
│  └──────────┘ └──────────────┘ └────────────┘ └───────────┘ └──────────┘ └──────────┘            │
│  ┌───────────┐ ┌─────────────┐ ┌─────────────┐ ┌──────────────┐ ┌────────────┐ ┌─────────────┐   │
│  │pkg/envutil│ │pkg/repoutil │ │pkg/sliceutil│ │pkg/semverutil│ │pkg/timeutil│ │pkg/typeutil │   │
│  └───────────┘ └─────────────┘ └─────────────┘ └──────────────┘ └────────────┘ └─────────────┘   │
│  ┌──────────┐ ┌──────────────┐                                                                    │
│  │pkg/stats │ │ pkg/testutil │  (test support only)                                              │
│  └──────────┘ └──────────────┘                                                                    │
└─────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

## Package Reference

| Package | Layer | Description |
|---------|-------|-------------|
| cmd/gh-aw | Entry | GitHub CLI extension binary entry point |
| cmd/gh-aw-wasm | Entry | WebAssembly target entry point |
| internal/tools/actions-build | Internal | Build/validate custom GitHub Actions |
| internal/tools/generate-action-metadata | Internal | Generate action.yml and README.md for JS modules |
| pkg/cli | Core | Command implementations (cobra commands) |
| pkg/workflow | Core | Workflow compilation engine (MD → GitHub Actions YAML) |
| pkg/parser | Core | Markdown frontmatter and YAML parsing |
| pkg/console | Core | Terminal UI rendering and formatting |
| pkg/agentdrain | Core | Drain3 log template mining and anomaly detection |
| pkg/actionpins | Core | Action pin resolution — maps repo refs to pinned commit SHAs |
| pkg/constants | Shared | Shared constants and semantic type aliases |
| pkg/types | Shared | Shared type definitions across packages |
| pkg/logger | Utility | Namespace-based debug logging with zero overhead when disabled |
| pkg/stringutil | Utility | String manipulation utilities |
| pkg/fileutil | Utility | File path and file operation utilities |
| pkg/gitutil | Utility | Git repository utilities |
| pkg/styles | Utility | Centralized terminal color and style definitions |
| pkg/tty | Utility | TTY (terminal) detection utilities |
| pkg/envutil | Utility | Environment variable reading and validation |
| pkg/repoutil | Utility | GitHub repository slug and URL utilities |
| pkg/sliceutil | Utility | Generic slice utility functions |
| pkg/semverutil | Utility | Semantic versioning primitives |
| pkg/timeutil | Utility | Time formatting utilities |
| pkg/typeutil | Utility | Safe type conversion utilities for any/JSON/YAML values |
| pkg/stats | Utility | Numerical statistics utilities for metric collection |
| pkg/testutil | Utility | Test support utilities (test-only) |
