# Agentic Workflows Dashboard — Agent Guide

## Overview

This is a GitHub Copilot canvas extension that renders a live dashboard for agentic workflow definitions, runs, usage, and experiments. It is a loopback Node.js extension: the extension process spawns a small HTTP server for each canvas panel, and the iframe served by that server drives the UI.

---

## Directory Structure

```
.github/extensions/agentic-workflows-dashboard/
│
│  ── TypeScript sources (canonical) ──────────────────────────
├── src/
│   ├── dashboard-cli.ts      CLI runner (spawn, binary detection, getStatus)
│   ├── dashboard-config.ts   Shared constants (cache TTL, window presets, etc.)
│   ├── dashboard-data.ts     Data-access layer (getDefinitions, getRuns, getUsage, …)
│   ├── dashboard-logs.ts     gh aw logs argument/option helpers
│   ├── usage-forecast.ts     Usage summary and forecast aggregation
│   ├── models.ts             Shared TypeScript types
│   ├── pagination.ts         Pagination utility
│   ├── app.ts                Web frontend entry point (bundled → web/app.js)
│   └── alpinejs.d.ts         Alpine.js type declarations
│
│  ── Compiled JS outputs (DO NOT edit manually) ───────────────
├── dashboard-cli.js          compiled from src/dashboard-cli.ts
├── dashboard-config.js       compiled from src/dashboard-config.ts
├── dashboard-data.js         compiled from src/dashboard-data.ts
├── dashboard-logs.js         compiled from src/dashboard-logs.ts
├── usage-forecast.js         compiled from src/usage-forecast.ts
│
│  ── Hand-authored files ──────────────────────────────────────
├── extension.mjs             Extension entry point (not compiled — edit directly)
│
│  ── Web frontend (auto-generated) ────────────────────────────
├── web/
│   ├── app.js                bundled by esbuild from src/app.ts
│   ├── models.js             (empty re-export, unused at runtime)
│   ├── index.html            UI template
│   └── styles.css            Stylesheet
│
│  ── Tests ────────────────────────────────────────────────────
├── test/
│   ├── dashboard-cli.test.ts
│   ├── dashboard-data.test.ts
│   ├── dashboard-logs.test.ts
│   ├── pagination.test.ts
│   └── usage-forecast.test.ts
│
├── package.json
├── tsconfig.json             Used for type-checking (--noEmit); outDir=./web
└── vitest.config.ts
```

---

## TypeScript / JavaScript Relationship

| File | Source | How to regenerate |
|---|---|---|
| `dashboard-cli.js` | `src/dashboard-cli.ts` | `npm run build:ts` |
| `dashboard-config.js` | `src/dashboard-config.ts` | `npm run build:ts` |
| `dashboard-data.js` | `src/dashboard-data.ts` | `npm run build:ts` |
| `dashboard-logs.js` | `src/dashboard-logs.ts` | `npm run build:ts` |
| `usage-forecast.js` | `src/usage-forecast.ts` | `npm run build:ts` |
| `web/app.js` | `src/app.ts` | `npm run build:web` |
| `extension.mjs` | — hand-authored — | edit directly |

**Rule:** Always edit `src/*.ts` files, then run `npm run build:ts` to regenerate the root-level `.js` files. Never edit the root-level `.js` files by hand.

> Note: `tsconfig.json` uses `outDir: "./web"` and `--noEmit` for type-checking only.
> `build:ts` uses a separate `tsconfig.emit.json` with `outDir: "."` and `rootDir: "src"`.

---

## Build

```sh
# Install dependencies (first time or after package.json changes)
npm ci

# Type-check all TypeScript sources
npm run typecheck

# Compile src/*.ts → root *.js (after any change to src/ backend files)
npm run build:ts

# Bundle src/app.ts → web/app.js (after any change to the web frontend)
npm run build:web

# Full build (typecheck + build:web)
npm run build
```

---

## Format

```sh
npm run fmt          # format all files (TS, HTML, CSS, MJS, JSON)
npm run fmt:ts       # TypeScript only (src/**/*.ts, test/**/*.ts)
npm run fmt:js       # extension.mjs only
npm run fmt:html     # web/index.html
npm run fmt:css      # web/styles.css
npm run fmt:json     # copilot-extension.json, package.json
```

---

## Lint

```sh
npm run lint         # typecheck + test (full lint gate)
npm run typecheck    # type-check only (no emit)
```

---

## Test

```sh
npm test             # run all unit tests via vitest
```

Tests live in `test/` and are written in TypeScript. Vitest runs them directly (no separate compile step needed).

---

## Makefile targets (from repo root)

```sh
make test-canvas-extension   # npm ci + build + test
make fmt-canvas-extension    # npm ci + fmt
make lint-canvas-extension   # npm ci + lint
```

---

## Debugging

All runtime log output goes to the extension's log file. The log path is printed by the Copilot CLI when the extension starts, and can be retrieved with:

```sh
# From the Copilot CLI agent:
extensions_manage inspect --name agentic-workflows-dashboard
```

### Logging strategy

All `console.error` calls use a **`[filename]` category prefix** so log lines are greppable by source file:

| Prefix | File |
|---|---|
| `[dashboard-cli]` | `src/dashboard-cli.ts` |
| `[dashboard-data]` | `src/dashboard-data.ts` |
| `[extension]` | `extension.mjs` |

**What is logged:**
- `[dashboard-cli]`: every `spawn` call (file, args, cwd), close events when stderr is non-empty or exit code ≠ 0, spawn errors, binary detection result, `getStatus` outcome
- `[dashboard-data]`: cache hits/misses for every data function, CLI fetch start/finish with counts, JSON parse failures with output snippet, per-batch log fetch progress, errors from every async function
- `[extension]`: startup (`__dirname`, `workspacePath`), HTTP server port, per-API-request timing, request errors, canvas open/close lifecycle

To tail the log live:
```powershell
Get-Content -Wait -Tail 50 "<log-path-from-inspect>"
```

To filter by category:
```powershell
Get-Content "<log-path>" | Select-String "[dashboard-cli]" -SimpleMatch
```

---

## Known Pitfalls

- **`session.workspacePath` and `process.cwd()` are NOT the git repo root.** `session.workspacePath` points to the session-state folder (`~/.copilot/session-state/<id>`). `process.cwd()` resolves to the Copilot runtime directory (`~/.copilot`). The git repo root must be derived from `__dirname`: for a project-scoped extension at `.github/extensions/<name>/`, use `resolve(__dirname, "../../..")`.
- **`detached: true` on Windows** causes spawned processes to allocate a new console, which can redirect their stdout to that console instead of the pipe. Use `windowsHide: true` without `detached: true` for subprocess spawning.
- **Root `.js` files in git** — they are committed alongside the TS sources. After editing `src/`, run `npm run build:ts` and commit both.
