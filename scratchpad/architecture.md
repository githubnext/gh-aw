# Architecture Diagram

> Last updated: 2026-07-06 · Source: [Issue #aw_arch_2026_0706](#aw_arch_2026_0706)

## Overview

This diagram shows the package structure and dependency layers of the `gh-aw` codebase.
The project compiles markdown workflow files into GitHub Actions YAML via a layered pipeline:
CLI entry points → core packages (cli, workflow, parser, console) → domain helpers → utility leaf packages.

```
┌──────────────────────────────────────────────────────────────────────────────────────────┐
│                                    ENTRY POINTS                                          │
│                                                                                          │
│   ┌──────────────────┐      ┌──────────────────────┐      ┌─────────────────────┐        │
│   │   cmd/gh-aw      │      │   cmd/gh-aw-wasm      │      │    cmd/linters      │       │
│   │   (main binary)  │      │   (WebAssembly target)│      │    (linter runner)  │       │
│   └────────┬─────────┘      └──────────┬────────────┘      └──────────┬──────────┘       │
│            │                            │                               │                │
├──────────────────────────────────────────────────────────────────────────────────────────┤
│            ▼           CORE PACKAGES    ▼                               ▼                │
│                                                                    pkg/linters/*         │
│                                                                                          │
│   ┌──────────────────────────┐   ┌──────────────────────────────────────────────────┐    │
│   │          cli             │   │                  workflow                         │   │
│   │  Command implementations │──▶│  Compilation engine (MD → GH Actions YAML)       │    │
│   │  compile, run, audit,    │   │  Frontmatter eval, engine dispatch,              │    │
│   │  mcp, logs, campaigns    │   │  lock file generation                            │    │
│   └──────────┬───────────────┘   └──────────────────┬───────────────────────────────┘    │
│              │             ┌────────────────────────┘                                    │
│              ▼             ▼                                                             │
│   ┌──────────────────────┐   ┌────────────────────────────────────────────────────┐      │
│   │       console        │   │                      parser                         │     │
│   │  Terminal UI         │   │  Markdown frontmatter & YAML parsing, schema        │     │
│   │  rendering and msg   │   │  validation, expression extraction                  │     │
│   └──────────────────────┘   └────────────────────────────────────────────────────┘      │
│                                                                                          │
│   ┌──────────────────┐  ┌────────────────────────────────┐  ┌────────────────────────┐   │
│   │      types       │  │           constants             │  │  workflow/compilerenv  │  │
│   │  Shared domain   │  │  Semantic type aliases,         │  │  Compiler env          │  │
│   │  type definitions│  │  engine/job names, flags        │  │  management            │  │
│   └──────────────────┘  └────────────────────────────────┘  │  (used by cli+workflow)│   │
│                                                              └────────────────────────┘  │
├──────────────────────────────────── DOMAIN PACKAGES ─────────────────────────────────────┤
│                                                                                          │
│  ┌─────────────────┐  ┌─────────────────┐  ┌───────────────────────┐  ┌────────────┐     │
│  │   actionpins    │  │   agentdrain    │  │        linters         │  │   stats    │    │
│  │  Action pin     │  │  Agent lifecycle│  │  Custom Go vet-style   │  │ Numerical  │    │
│  │  resolution &   │  │  & drain helpers│  │  code analyzers        │  │ statistics │    │
│  │  version mgmt   │  └─────────────────┘  └───────────────────────┘  └────────────┘     │
│  └─────────────────┘                                                                     │
│  ┌──────────────────────────────────────────────────────────────────────────────────┐    │
│  │  github  —  GitHub label ↔ objective-value mapping (configurable audit scoring)  │    │
│  └──────────────────────────────────────────────────────────────────────────────────┘    │
│  ┌──────────────────────────────────────────────────────────────────────────────────┐    │
│  │  intent  —  PR intent attribution resolver (maps PRs to issues/objectives)       │    │
│  └──────────────────────────────────────────────────────────────────────────────────┘    │
│  ┌──────────────────────────────────────────────────────────────────────────────────┐    │
│  │  modelsdev  —  models.dev catalog fetcher; LLM model pricing lookup (used by cli)│    │
│  └──────────────────────────────────────────────────────────────────────────────────┘    │
├──────────────────────────────────── UTILITY PACKAGES ────────────────────────────────────┤
│                                                                                          │
│  ┌──────────┐ ┌─────────┐ ┌────────────┐ ┌────────┐ ┌─────────┐ ┌──────────┐ ┌───────────┐  │
│  │ fileutil │ │ gitutil │ │ stringutil │ │ logger │ │ envutil │ │ errorutil│ │colorwriter│  │
│  └──────────┘ └─────────┘ └────────────┘ └────────┘ └─────────┘ └──────────┘ └───────────┘  │
│                                                                                          │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌─────────┐ ┌────────┐ ┌──────────┐ ┌─────┐      │
│  │ jsonutil │ │ repoutil │ │semverutil│ │ sliceutil│ │syncutil│ │ timeutil │ │ tty │     │
│  └──────────┘ └──────────┘ └──────────┘ └─────────┘ └────────┘ └──────────┘ └─────┘      │
│                                                                                          │
│  ┌──────────┐ ┌──────────┐ ┌─────────┐ ┌────────────────────────────┐ ┌──────────────┐   │
│  │ typeutil │ │  styles  │ │ setutil │ │  importinpututil           │ │  testutil    │   │
│  └──────────┘ └──────────┘ └─────────┘ └────────────────────────────┘ └──────────────┘   │
└──────────────────────────────────────────────────────────────────────────────────────────┘
```

## Package Reference

| Package | Layer | Description |
|---------|-------|-------------|
| `cli` | Core | Command implementations (compile, run, audit, mcp, logs, campaigns) |
| `workflow` | Core | Workflow compilation engine — markdown → GitHub Actions YAML |
| `parser` | Core | Markdown frontmatter & YAML parsing, schema validation, expression extraction |
| `console` | Core | Terminal UI rendering and message formatting |
| `types` | Core | Shared domain type definitions |
| `constants` | Core | Semantic type aliases, engine/job names, feature flags |
| `workflow/compilerenv` | Core | Compiler environment management (used by cli + workflow) |
| `github` | Domain | GitHub label ↔ objective-value mapping for audit/outcomes scoring |
| `intent` | Domain | PR intent attribution resolver (maps PRs to closing issues/objectives) |
| `actionpins` | Domain | GitHub Actions pin resolution and version management |
| `agentdrain` | Domain | Agent drain/lifecycle utilities |
| `linters` | Domain | Custom Go analysis linters (vet-style checks) |
| `stats` | Domain | Numerical statistics for metric collection |
| `modelsdev` | Domain | models.dev catalog fetcher; LLM model pricing lookup |
| `fileutil` | Util | File path and file operation helpers |
| `gitutil` | Util | Git repository helpers |
| `stringutil` | Util | String utilities (ANSI stripping, transforms) |
| `logger` | Util | Namespace-based debug logging with zero overhead |
| `envutil` | Util | Environment variable reading and validation |
| `errorutil` | Util | Error classification and inspection helpers |
| `jsonutil` | Util | JSON serialization helpers |
| `repoutil` | Util | GitHub repository slug and URL utilities |
| `semverutil` | Util | Semantic versioning primitives |
| `sliceutil` | Util | Generic slice operation helpers |
| `setutil` | Util | Generic set operation helpers (map[K]struct{}) |
| `syncutil` | Util | Concurrency synchronization helpers |
| `timeutil` | Util | Time formatting helpers |
| `tty` | Util | TTY detection utilities |
| `typeutil` | Util | General-purpose type conversion utilities |
| `styles` | Util | Centralized terminal color/style definitions |
| `importinpututil` | Util | Import path / sub-key resolver |
| `testutil` | Util | Test helpers (test-only, not used in production) |
| `colorwriter` | Util | Color-profile-aware io.Writer (NO_COLOR/COLORTERM/TERM aware) |

## Maintenance

### Sync Policy

The diagram and package table above must stay in sync with the actual Go package layout under `pkg/`. Without an explicit policy, new packages added to the codebase will silently diverge from this document.

**Sync trigger:** Any PR that adds or removes a Go package under `pkg/` **must** update this document. Reviewers should verify that:
1. The ASCII diagram reflects the new package and its correct layer (Core / Domain / Util).
2. A corresponding row is added to (or removed from) the Package Reference table.
3. The `Last updated` date on line 3 is refreshed.

**Tracking:** A CI lint rule or OWNERS note enforcing this policy is tracked in a future issue. Until that check is automated, the PR author is responsible for updating the diagram as part of their change.

**Ownership:** The `architecture.md` document is owned by the gh-aw maintainer team. Any contributor may propose updates via PR; changes to the diagram layers or the package categorization require at least one maintainer approval.
