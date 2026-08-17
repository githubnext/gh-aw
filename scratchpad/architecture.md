# Architecture Diagram

> Last updated: 2026-08-17 · Source: Architecture Diagram issue

## Overview

This diagram shows the package structure and dependencies of the `gh-aw` codebase.

```
================================================================================================
                                    ENTRY POINTS
================================================================================================

   cmd/gh-aw            cmd/gh-aw-wasm         cmd/linters
   (CLI binary)         (WASM target)          (analysis binary)
        |                     |                       |
        v                     v                       v
------------------------------------------------------------------------------------------------
                                   CORE PACKAGES
------------------------------------------------------------------------------------------------

   pkg/cli  ────────▶  pkg/workflow  ────────▶  pkg/parser
   (400 files)         (478 files)               (markdown/YAML
   commands: run,      compiles workflow          frontmatter parsing)
   compile, audit,     markdown -> GH Actions          |
   mcp, trial            YAML                          |
     |    |    |             |       \                 |
     |    |    |             |        \                v
     |    |    v             v         v          pkg/github / pkg/githubapi
     |    | pkg/console  pkg/agentdrain             (GitHub API client,
     |    | (terminal UI) (agent log mining/         model registry)
     |    |               drain pipeline)
     |    v
     |  pkg/linters (custom Go static-analysis rules, used by cmd/linters)
     v
   pkg/intent, pkg/modelsdev, pkg/actionpins, pkg/importinpututil
   (intent detection, model metadata, action-pin resolution, import inputs)

------------------------------------------------------------------------------------------------
                                      UTILITIES
------------------------------------------------------------------------------------------------

  pkg/logger      pkg/gitutil     pkg/fileutil    pkg/stringutil   pkg/repoutil
  pkg/typeutil    pkg/sliceutil   pkg/setutil     pkg/syncutil     pkg/jsonutil
  pkg/semverutil  pkg/ctxutil     pkg/envutil     pkg/errorutil    pkg/tty
  pkg/colorwriter pkg/styles      pkg/timeutil    pkg/stats        pkg/testutil
  pkg/types       pkg/constants

Legend: ──▶ / | = "depends on / imports" (arrow points from dependent to dependency)
```

## Package Reference

| Package | Layer | Description |
|---------|-------|--------------|
| cli | Core | Command-line interface implementations (run, compile, audit, mcp, trial, etc.) |
| workflow | Core | Workflow compilation engine — compiles markdown+frontmatter into GitHub Actions YAML |
| parser | Core | Markdown/YAML frontmatter parsing and content extraction |
| console | Core | Terminal UI components and formatted output |
| agentdrain | Core | Agent log mining / drain pipeline |
| github | Core | GitHub API mapping/client helpers |
| githubapi | Core | GitHub API client wrapper |
| linters | Core | Custom Go static-analysis linters (used by cmd/linters) |
| intent | Core | Intent detection helpers |
| modelsdev | Core | AI model metadata/registry |
| actionpins | Core | Action pin resolution (SHA pinning for GitHub Actions) |
| importinpututil | Core | Import input utilities used by parser/workflow |
| logger | Utility | Namespace-based debug logging |
| gitutil | Utility | Git repository helper utilities |
| fileutil | Utility | File path and file operation helpers |
| stringutil | Utility | String manipulation utilities |
| repoutil | Utility | GitHub repository slug/URL utilities |
| typeutil | Utility | Untyped value conversion/extraction helpers |
| sliceutil | Utility | Slice manipulation utilities |
| setutil | Utility | Set data structure utilities |
| syncutil | Utility | Concurrency helpers (e.g., OnceLoader) |
| jsonutil | Utility | JSON helper utilities |
| semverutil | Utility | Semantic versioning primitives |
| ctxutil | Utility | context.Context handling helpers |
| envutil | Utility | Environment variable reading/validation |
| errorutil | Utility | Error classification/inspection helpers |
| tty | Utility | TTY (terminal) detection |
| colorwriter | Utility | Low-level color-aware writer (used by logger) |
| styles | Utility | Centralized terminal style/color definitions |
| timeutil | Utility | Time-related utilities |
| stats | Utility | Numerical statistics utilities |
| testutil | Utility | Shared test helpers |
| types | Utility | Shared type definitions |
| constants | Utility | Shared constant definitions |
