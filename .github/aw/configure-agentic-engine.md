---
description: Guide for configuring new declarative agentic engines — engine definition fields, auth wiring, behavior blocks, and validation.
---

# Configure a New Agentic Engine

Use this guide when adding or updating a built-in engine definition under `pkg/workflow/data/engines/` or when reviewing a proposed declarative engine configuration.

## Prefer declarative engine definitions

For CLI-style engines, define behavior in `pkg/workflow/data/engines/<id>.md`.

- prefer frontmatter-defined `engine.behaviors` over a bespoke Go wrapper
- keep install, config, execution, MCP, manifest, and capability metadata in the engine markdown file
- add Go changes only when the runtime cannot be expressed with the current declarative behavior schema

Built-in engine files are embedded from `pkg/workflow/data/engines/*.md`. Adding a new built-in engine should normally start with a new markdown file there.

## Gather the engine contract first

Before editing files, identify:

1. the stable engine `id` and display name
2. whether it can reuse an existing `runtime-id` or needs a new runtime adapter
3. the install source, package manager, package name, binary name, and version
4. the required secrets and whether they follow universal provider routing or engine-specific auth
5. any config file path, default content, and merge behavior
6. the execution command, fixed args, model wiring, and MCP config wiring
7. engine-owned manifest files or directories that must be protected from untrusted PR edits

## Engine definition shape

```aw wrap
engine:
  id: auggie
  display-name: Auggie
  experimental: true
  auth:
    - role: session
      secret: AUGMENT_SESSION_AUTH
  behaviors:
    supported-env-var-keys:
      - AUGMENT_SESSION_AUTH
    installation:
      package-manager: npm
      package-name: "@augmentcode/auggie"
      version: "1.0.0"
      step-name: Install Auggie
      binary-name: auggie
      include-node-setup: true
    config-file:
      path: .auggie.json
      step-name: Write Auggie Config
      content: '{"sandbox":"workspace-write"}'
      merge-strategy: json-merge
    execution:
      command-name: auggie
      args: [run]
      step-name: Execute Auggie CLI
      model-env-var: AUGGIE_MODEL
      mcp-config-env-var: AUGGIE_MCP_CONFIG
      write-timestamp: true
```

## Field guide

- `engine.id` is the public identifier used by workflow authors in `engine: <id>`.
- `display-name` and `description` should be human-readable because they surface in validation and docs.
- `runtime-id` is only needed when the definition reuses a different registered runtime adapter.
- `experimental: true` should be set for engines that are not yet considered stable.
- `provider` and `models` describe provider defaults and supported model metadata.
- `auth` declares engine-specific secret bindings forwarded into the runtime environment.
- `behaviors.capabilities` advertises runtime support such as `max-turns`, `tools-allowlist`, or `native-agent-file`.
- `behaviors.manifest` lists engine-owned files and path prefixes that affect runtime behavior.
- `behaviors.installation` defines CLI installation and optional verification steps.
- `behaviors.config-file` writes engine config before execution; use `json-merge` when the file must merge with rendered MCP content.
- `behaviors.execution` defines the command, fixed args, model binding, MCP binding, and timestamp behavior.
- `behaviors.mcp.config-path` points to the file where rendered MCP configuration should be written.

## Auth and provider rules

- prefer `secret-strategy: universal-llm-consumer` when the engine can reuse shared provider/model routing
- pair that with `execution.provider-env-mode: universal-llm-consumer` when the CLI expects provider env vars
- use `engine.auth` only for engine-specific secrets that must be injected directly into the CLI runtime
- keep `supported-env-var-keys` aligned with the env var names the CLI actually accepts
- do not hard-code credential values in markdown, Go, or tests

## Validation loop

1. add or update `pkg/workflow/data/engines/<id>.md`
2. if the schema surface changes, update `pkg/parser/schemas/main_workflow_schema.json`
3. if the engine is user-facing, update `docs/src/content/docs/reference/engines.md` and any generated schema reference as needed
4. if a new built-in engine ID is added, update tests that assert the catalog contents
5. run:

```bash
make build
go test ./pkg/workflow/... ./pkg/parser/...
```

## Anti-patterns

- do not add a new bespoke `*_engine.go` wrapper for behavior that already fits `engine.behaviors`
- do not store install metadata, CLI args, or config-file templates partly in Go and partly in markdown without a clear need
- do not omit manifest files for engine-owned config that changes runtime behavior
- do not use a mismatched `runtime-id` unless an existing runtime adapter is intentionally being reused
