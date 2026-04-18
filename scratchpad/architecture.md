# Architecture Diagram

> Last updated: 2026-04-17 · Source: [Issue #26033](https://github.com/github/gh-aw/issues/26033)

## Overview

This diagram shows the package structure and dependencies of the `gh-aw` codebase.

```
┌──────────────────────────────────────────────────────────────────────────────────────────────┐
│  ENTRY POINTS                                                                                 │
│  ┌───────────────────────────────────────────┐    ┌──────────────────────────────────────┐   │
│  │              cmd/gh-aw                    │    │          cmd/gh-aw-wasm              │   │
│  │          Main CLI binary                  │    │          WebAssembly target          │   │
│  └──────────────────────┬────────────────────┘    └──────────────┬───────┬──────────────┘   │
│                          │                                        │       │                   │
│  ┌───────────────────────┘  internal/tools ──▶ pkg/cli           │       │                   │
│  │  (actions-build, generate-action-metadata)                     │       │                   │
├──┼────────────────────────────────────────────────────────────────┼───────┼───────────────────┤
│  CORE                    ▼                                        │       │                   │
│  ┌────────────────────────────────────────────────────────────┐   │       │                   │
│  │                       pkg/cli                               │   │       │                   │
│  │   Commands: compile, run, audit, mcp, logs, fix, add...     │   │       │                   │
│  └────────────────────────────────────────┬────────────────────┘   │       │                   │
│                                            │                        │       │                   │
│                                            ▼                        ▼       │                   │
│  ┌──────────────────────┐  ┌──────────────────────────────────────────┐     │                   │
│  │    pkg/agentdrain    │  │               pkg/workflow               │     │                   │
│  │  Log drain, cluster  │  │  Workflow compilation engine &           │     │                   │
│  │  & anomaly detection │  │  GitHub Actions YAML generation          │     │                   │
│  └──────────────────────┘  └────────────────────┬──────────┬─────────┘     │                   │
│                                                  │          │               │                   │
│                                                  ▼          │               ▼                   │
│  ┌───────────────────────────────────────────┐   │  ┌───────────────────────────────────────┐  │
│  │               pkg/parser                  │◀──┘  │         pkg/actionpins               │  │
│  │  Markdown frontmatter parsing &            │      │  GitHub Actions pin resolution        │  │
│  │  YAML schema validation                    │      │  (SHA pinning from version refs)      │  │
│  └───────────────────────────────────────────┘      └───────────────────────────────────────┘  │
│  ┌────────────────────────────────────────────────────────────────────────────────────────┐  │
│  │                            pkg/console                                                   │  │
│  │   Terminal UI: rendering, progress bars, spinner, prompts & output formatting            │  │
│  │   ← consumed by: cli · workflow · parser · actionpins                                    │  │
│  └───────────────────────────────────────────────────────────────────────────┬────────────┘  │
│                                                                                │               │
├────────────────────────────────────────────────────────────────────────────────┼───────────────┤
│  UTILITIES                                                                     ▼               │
│  ┌────────┐  ┌──────┐  ┌─────┐  ┌──────────┐  ┌──────────┐  ┌────────────┐  ┌──────────┐   │
│  │ logger │  │styles│  │ tty │  │  types   │  │constants │  │ stringutil │  │ sliceutil│   │
│  └────────┘  └──────┘  └─────┘  └──────────┘  └──────────┘  └────────────┘  └──────────┘   │
│  ┌─────────┐  ┌──────────┐  ┌────────┐  ┌─────────┐  ┌──────────┐  ┌──────┐  ┌────────┐   │
│  │typeutil │  │ fileutil │  │gitutil │  │repoutil │  │semverutil│  │stats │  │envutil │   │
│  └─────────┘  └──────────┘  └────────┘  └─────────┘  └──────────┘  └──────┘  └────────┘   │
│  ┌──────────┐  ┌─────────┐                                                                   │
│  │ timeutil │  │testutil │                                                                   │
│  └──────────┘  └─────────┘                                                                   │
└──────────────────────────────────────────────────────────────────────────────────────────────┘
```

## Package Reference

| Package | Layer | Description |
|---------|-------|-------------|
| cmd/gh-aw | Entry | Main CLI binary (GitHub CLI extension entry point) |
| cmd/gh-aw-wasm | Entry | WebAssembly compilation target |
| internal/tools/actions-build | Internal | Build and validate custom GitHub Actions |
| internal/tools/generate-action-metadata | Internal | Generate action.yml and README.md for JS modules |
| pkg/cli | Core | Command implementations: compile, run, audit, mcp, logs, fix, add, update |
| pkg/workflow | Core | Workflow compilation engine (Markdown → GitHub Actions YAML) |
| pkg/parser | Core | Markdown frontmatter parsing & YAML schema validation |
| pkg/console | Core | Terminal UI: rendering, progress bars, spinner, prompts & output formatting |
| pkg/agentdrain | Core | Log drain, clustering & anomaly detection (DRAIN3 algorithm) |
| pkg/actionpins | Core | GitHub Actions pin resolution — maps version refs to pinned SHAs |
| pkg/logger | Utility | Namespace-based debug logging with zero overhead when disabled |
| pkg/styles | Utility | Centralized terminal color and style definitions |
| pkg/tty | Utility | TTY (terminal) detection utilities |
| pkg/types | Utility | Shared type definitions across packages |
| pkg/constants | Utility | Shared constants: engine names, job names, feature flags, URLs |
| pkg/stringutil | Utility | String manipulation utilities |
| pkg/sliceutil | Utility | Generic slice utility functions |
| pkg/typeutil | Utility | Safe type conversion utilities for any/JSON/YAML values |
| pkg/fileutil | Utility | File path and file operation utilities |
| pkg/gitutil | Utility | Git repository utilities |
| pkg/repoutil | Utility | GitHub repository slug and URL utilities |
| pkg/semverutil | Utility | Semantic versioning primitives |
| pkg/stats | Utility | Numerical statistics utilities for metric collection |
| pkg/envutil | Utility | Environment variable reading and validation |
| pkg/timeutil | Utility | Time formatting utilities |
| pkg/testutil | Utility | Test support utilities (test-only) |
