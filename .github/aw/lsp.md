---
description: Language Server Protocol (LSP) configuration reference for gh-aw Copilot workflows — frontmatter syntax, supported servers, and file extension mapping.
---

# LSP Configuration

> ⚠️ **Experimental feature.** The `lsp` frontmatter field is experimental. Using it will emit a compile-time warning. The interface may change in future releases.

The `lsp` frontmatter field lets Copilot-engine workflows declare language servers. At compile time, the compiler:

1. Validates the configuration and rejects the workflow if `lsp` is used with a non-Copilot engine.
2. Generates `~/.copilot/settings.json` with an `lspServers` block the Copilot CLI reads at startup.
3. Injects install steps for known server ecosystems into the agent setup job.

> ⚠️ **`lsp` is only supported with `engine: copilot`**. Using it with any other engine causes a compile-time error.

## Syntax

```yaml
engine:
  id: copilot

lsp:
  <language-key>:
    command: <server-executable>
    args: [<arg1>, <arg2>]          # optional
    fileExtensions:
      ".<ext>": <language-id>       # at least one required
```

Each key under `lsp` is a language identifier (lowercase, alphanumeric, hyphens, or underscores). It maps to a server definition with:

| Field | Required | Description |
|---|---|---|
| `command` | **yes** | Executable name or path for the language server |
| `args` | no | Command-line arguments passed to the server on startup |
| `fileExtensions` | **yes** | Map of file extension (with leading `.`) to LSP language ID |
| `version` | no | Package version to install (e.g. `"5.8.3"`). Overrides the built-in pinned default for known servers. |

## Built-in Servers

For the languages below, the compiler automatically injects an install step — no manual `steps:` entry is needed. Each server is pinned to a known-good release by default; use the `version` field to override.

| Language key | Default version | Install command | Example `command` |
|---|---|---|---|
| `bash` | `5.4.0` | `npm install -g --ignore-scripts bash-language-server@5.4.0` | `bash-language-server` |
| `go` | `0.18.1` | `go install golang.org/x/tools/gopls@v0.18.1` | `gopls` |
| `php` | `1.14.1` | `npm install -g --ignore-scripts intelephense@1.14.1` | `intelephense` |
| `python` | `1.1.399` | `npm install -g --ignore-scripts pyright@1.1.399` | `pyright-langserver` |
| `ruby` | `0.50.0` | `gem install solargraph -v 0.50.0` | `solargraph` |
| `rust` | n/a | `rustup component add rust-analyzer` | `rust-analyzer` |
| `typescript` | `5.8.3` / `4.3.3` | `npm install -g --ignore-scripts typescript@5.8.3 typescript-language-server@4.3.3` | `typescript-language-server` |
| `yaml` | `1.15.0` | `npm install -g --ignore-scripts yaml-language-server@1.15.0` | `yaml-language-server` |

> The `version` field overrides the pinned version for the primary language server package (the last package in the install list). For `typescript`, it controls `typescript-language-server`; `typescript` itself stays at its hardcoded companion version (`5.8.3`).

Language keys not in this table still work — the compiler simply skips the auto-install step. Add a manual `steps:` entry to install the server yourself.

## Examples

### TypeScript / JavaScript

```yaml
engine:
  id: copilot

lsp:
  typescript:
    command: typescript-language-server
    args: ["--stdio"]
    fileExtensions:
      ".ts": typescript
      ".tsx": typescriptreact
      ".js": javascript
      ".cjs": javascript
      ".mjs": javascript
```

### Python

```yaml
engine:
  id: copilot

lsp:
  python:
    command: pyright-langserver
    args: ["--stdio"]
    fileExtensions:
      ".py": python
```

### Go

```yaml
engine:
  id: copilot

lsp:
  go:
    command: gopls
    fileExtensions:
      ".go": go
```

### Multiple Languages

```yaml
engine:
  id: copilot

lsp:
  typescript:
    command: typescript-language-server
    args: ["--stdio"]
    fileExtensions:
      ".ts": typescript
      ".js": javascript
  python:
    command: pyright-langserver
    args: ["--stdio"]
    fileExtensions:
      ".py": python
```

### Custom Server (no built-in install)

For servers without a built-in install spec, add a manual install step:

```yaml
engine:
  id: copilot

steps:
  - name: Install custom language server
    run: npm install -g my-custom-language-server

lsp:
  mylang:
    command: my-custom-language-server
    args: ["--stdio"]
    fileExtensions:
      ".ml": mylang
```

## Network Requirements

Installing LSP servers requires network access to the appropriate package registry. Add the matching ecosystem to `network.allowed`:

| Language | Ecosystem to add |
|---|---|
| `bash`, `php`, `python`, `typescript`, `yaml` | `node` |
| `go` | `go` |
| `ruby` | `ruby` |
| `rust` | `rust` |

```yaml
network:
  allowed:
    - node   # for npm-installed servers (typescript, yaml, python/pyright, etc.)
    - go     # for gopls
```

## Compile-time Validation

The compiler enforces these rules at compile time:

- `lsp` requires `engine: copilot` — any other engine causes an error.
- Each language entry must have a non-empty `command`.
- Each language entry must define at least one `fileExtensions` mapping.
- Language keys are case-insensitive and trimmed; duplicate keys that collapse to the same lowercase value are normalized deterministically with the lexicographically first original key winning, but should still be avoided because the result may be surprising.
