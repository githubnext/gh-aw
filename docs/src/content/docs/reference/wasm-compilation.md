---
title: WebAssembly Compilation
description: How to compile the gh-aw workflow compiler to WebAssembly and use it in the browser or other JavaScript environments.
---

The gh-aw compiler can be built as a WebAssembly (Wasm) module, letting you compile agentic workflows directly in the browser without a server-side Go installation.

## Overview

The Wasm build packages the core compilation engine — markdown parsing, frontmatter extraction, import resolution, and YAML generation — into a single `.wasm` file. You load it with Go's standard `wasm_exec.js` runtime, then call a global `compileWorkflow()` function from JavaScript.

This is useful for:

- **Interactive playgrounds** where users experiment with workflow syntax
- **Editor integrations** that preview compiled YAML in real time
- **Offline tools** that need compilation without a backend

## Prerequisites

- Go 1.25 or later
- `make` (GNU Make)

## Building

Run the following from the repository root:

```bash
make build-wasm
```

This produces two artifacts:

| File | Description |
|------|-------------|
| `gh-aw.wasm` | The compiled WebAssembly module (~17 MB uncompressed) |
| `$(go env GOROOT)/misc/wasm/wasm_exec.js` | Go's Wasm runtime (ships with your Go installation) |

Copy both files to your project:

```bash
cp gh-aw.wasm your-project/
cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" your-project/
```

## Compression

The raw `.wasm` binary is ~17 MB. The build pipeline pre-compresses it with [brotli](https://github.com/google/brotli) at maximum quality (`-q 11`), producing a `gh-aw.wasm.br` file of ~5 MB — a ~70% reduction. GitHub Pages serves the `.br` file automatically when the browser sends `Accept-Encoding: br`.

```bash
# Manual compression (if not using the bundle script)
brotli -k -q 11 gh-aw.wasm    # produces gh-aw.wasm.br (~5 MB)
```

The docs site bundle script (`scripts/bundle-wasm-docs.sh`) handles this automatically. If `brotli` is not installed, it falls back to gzip (`-9`), which achieves ~6 MB.

## JavaScript API

### Loading the module

```html
<script src="wasm_exec.js"></script>
<script>
const go = new Go();
WebAssembly.instantiateStreaming(
  fetch("gh-aw.wasm"),
  go.importObject
).then((result) => {
  go.run(result.instance);
  // compileWorkflow is now available globally
});
</script>
```

### `compileWorkflow(markdown)`

Compiles a markdown workflow string into GitHub Actions YAML.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `markdown` | `string` | Yes | The full markdown workflow content, including frontmatter |

**Returns:** `Promise<{ yaml: string, warnings: string[], error: null }>`

On failure, the promise rejects with an `Error`.

### Basic example

```javascript
const result = await compileWorkflow(`---
name: hello-world
description: A simple greeting workflow
on:
  workflow_dispatch:
engine: copilot
---

Say "Hello, world!" as an issue comment.
`);

console.log(result.yaml);
```

## How it works

The Wasm build uses Go [build tags](https://pkg.go.dev/go/build#hdr-Build_Constraints) to swap platform-dependent code with lightweight stubs at compile time. The native build and Wasm build share the same core compiler — only the I/O and TUI layers differ.

### Build tag convention

Each stubbed file uses a pair of build constraints:

- **`//go:build js || wasm`** — the stub, compiled into the Wasm module
- **`//go:build !js && !wasm`** — the native implementation, excluded from Wasm

### What gets stubbed

The Wasm build replaces four categories of functionality:

**Terminal UI (`pkg/tty`, `pkg/styles`, `pkg/console`)**

The native build uses [Lip Gloss](https://github.com/charmbracelet/lipgloss), [Bubble Tea](https://github.com/charmbracelet/bubbletea), and [Huh](https://github.com/charmbracelet/huh) for styled terminal output, spinners, forms, and prompts. The Wasm stubs replace these with plain-text equivalents — no ANSI escape codes, no interactive input.

Stubbed console files: `banner`, `confirm`, `console`, `form`, `input`, `layout`, `list`, `progress`, `select`, `spinner`.

**External tool validation (`pkg/workflow`)**

The native compiler shells out to validate that tools like `npm`, `pip`, `docker`, `git`, and `gh` are installed and configured correctly. Since `os/exec` is unavailable in Wasm, these validators return `nil` (skip validation).

Stubbed validators: `npm_validation`, `pip_validation`, `docker_validation`, `git_helpers`, `github_cli`, `dependabot`, `repository_features_validation`.

**Remote imports (`pkg/parser`)**

Fetching imports from remote GitHub repositories requires HTTP calls and `gh` CLI authentication. In the Wasm build, remote imports return an error. A JavaScript import resolver callback is planned for a future release.

**GitHub token access (`pkg/parser`)**

The native build retrieves GitHub tokens via `gh auth token`. The Wasm build falls back to reading `GITHUB_TOKEN` or `GH_TOKEN` environment variables only.

### String-based compilation API

The Wasm entry point uses `CompileToYAML()`, a string-in/string-out API on the `Compiler` struct that returns YAML content without writing to disk. This method runs in no-emit mode (`WithNoEmit(true)`) and skips external validation (`WithSkipValidation(true)`).

## Architecture

The following diagram shows which packages have Wasm-specific stubs:

```
cmd/gh-aw-wasm/main.go          ← Wasm entry point (syscall/js)
    │
    ├── pkg/workflow/
    │   ├── compiler*.go              (shared — core compiler)
    │   ├── compiler_string_api.go    (shared — CompileToYAML)
    │   ├── npm_validation_wasm.go    (stub — returns nil)
    │   ├── pip_validation_wasm.go    (stub — returns nil)
    │   ├── docker_validation_wasm.go (stub — returns nil)
    │   ├── git_helpers_wasm.go       (stub — returns nil)
    │   ├── github_cli_wasm.go        (stub — returns nil)
    │   ├── dependabot_wasm.go        (stub — returns nil)
    │   └── repository_features_validation_wasm.go
    │                                 (stub — returns nil)
    │
    ├── pkg/parser/
    │   ├── github_wasm.go            (stub — env-only token)
    │   └── remote_fetch_wasm.go      (stub — no remote imports)
    │
    ├── pkg/console/                  (stub — 10 files, plain text)
    ├── pkg/styles/theme_wasm.go      (stub — no-op styles)
    └── pkg/tty/tty_wasm.go           (stub — no TTY detection)
```

## Limitations

The Wasm build is focused on compilation only. The following features are not available:

| Feature | Reason |
|---------|--------|
| Interactive TUI (spinners, prompts, forms) | No terminal in the browser |
| External tool validation (npm, pip, docker, git, gh) | No `os/exec` in Wasm |
| Remote imports (`owner/repo/path@ref`) | No HTTP client or `gh` CLI |
| Filesystem writes | Compiler runs in no-emit mode |
| CLI commands (`gh aw init`, `gh aw watch`, etc.) | Only the `compileWorkflow` API is exposed |

> [!NOTE]
> Import resolution (`importResolver` callback) is not currently supported in the Wasm build. Workflows that use `imports:` will produce an error. This feature is planned for a future release.
