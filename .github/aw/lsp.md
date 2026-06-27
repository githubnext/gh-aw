---
description: Language Server Protocol (LSP) configuration reference for gh-aw Copilot workflows — frontmatter syntax, supported servers, and file extension mapping.
---

# LSP Configuration

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

## Built-in Servers

For the languages below, the compiler automatically injects an install step — no manual `steps:` entry is needed.

| Language key | Install command | Example `command` |
|---|---|---|
| `bash` | `npm install -g --ignore-scripts bash-language-server` | `bash-language-server` |
| `go` | `go install golang.org/x/tools/gopls@latest` | `gopls` |
| `php` | `npm install -g --ignore-scripts intelephense` | `intelephense` |
| `python` | `npm install -g --ignore-scripts pyright` | `pyright-langserver` |
| `ruby` | `gem install solargraph` | `solargraph` |
| `rust` | `rustup component add rust-analyzer` | `rust-analyzer` |
| `typescript` | `npm install -g --ignore-scripts typescript typescript-language-server` | `typescript-language-server` |
| `yaml` | `npm install -g --ignore-scripts yaml-language-server` | `yaml-language-server` |

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
- Language keys are case-insensitive and trimmed; duplicate keys that collapse to the same lowercase value cause nondeterministic behavior and should be avoided.
