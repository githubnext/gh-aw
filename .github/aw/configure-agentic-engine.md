---
description: Guide for configuring new declarative agentic engines — engine definition fields, auth wiring, behavior blocks, and validation.
---

# Configure a New Agentic Engine

Use this guide when adding or updating a declarative engine definition or when reviewing a proposed engine configuration.

## Prefer shared agentic workflow definitions

For CLI-style engines, start with a repository-scoped definition in `.github/workflows/shared/<id>.md`. Import it from a workflow that sets `engine.id: <id>`. Use the shared [`opencode.md`](../workflows/shared/opencode.md), [`goose.md`](../workflows/shared/goose.md), and [`aider.md`](../workflows/shared/aider.md) definitions as patterns.

- prefer frontmatter-defined `engine.behaviors` over a bespoke Go wrapper
- keep install, config, execution, MCP, manifest, and capability metadata in the engine markdown file
- keep engine-specific adapters and harnesses with the shared definition
- add Go changes only when the runtime cannot be expressed with the current declarative behavior schema

Promote an engine to `pkg/workflow/data/engines/<id>.md` only when it should ship as a built-in engine. Built-in engine files are embedded from `pkg/workflow/data/engines/*.md` and use the same declarative behavior shape.

## Gather the engine contract first

Do not begin from a generic engine template. Inspect the CLI documentation, source, help output, and a pinned release to answer every item below before editing files.

### LLM endpoint and model contract

1. Identify how the CLI accepts its LLM endpoint: environment variable, command flag, or config key.
2. Determine whether the endpoint is OpenAI-compatible, Anthropic-native, or another protocol, including any required path such as `/v1`.
3. Record the workflow-facing model syntax, normally `provider/model`, and the exact syntax passed to the CLI. Document any provider-prefix removal, replacement, or other model transformation.
4. Determine whether the CLI runs on the host or inside the AWF agent container. Do not copy a host URL into a container configuration or hard-code an `api-proxy` port or container IP.
5. Identify which configured provider must be selected at runtime and how to report an actionable error when the provider or model is unavailable.

### MCP contract

1. Determine whether the CLI has native MCP support. If it does not, set `engine.mcp: false` and rely on the compiler's proxy-backed tools.
2. If MCP is supported, identify the accepted transports, config path, root object name, server entry schema, and support for authorization headers.
3. Compare the native schema with the gateway's `{ "mcpServers": ... }` output. If they differ, generate the native config with `behaviors.mcp.config-adapter`.
4. Determine whether CLI-mounted servers must be filtered and whether gateway URLs must use the host or container domain.
5. Treat generated MCP configuration as sensitive when it contains gateway authorization headers; create it with owner-only permissions.

### Installation, config, and execution contract

1. Choose a stable engine `id` and display name and determine whether an existing `runtime-id` can be reused.
2. Identify the install source, package manager, package name, binary name, pinned version, and verification command.
3. Identify the config path and format, such as JSON, JSONC, YAML, or TOML. Determine whether gh-aw creates, replaces, or merges the file; do not use a JSON merge strategy for syntax that is only valid as JSONC.
4. Identify the non-interactive execution command, fixed arguments, prompt delivery mechanism, exit-code behavior, and any required environment variables.
5. List every engine-owned config file and directory in `behaviors.manifest`.
6. Identify required secrets and whether they use universal provider routing or engine-specific auth.

Do not implement the engine until each contract is known. If documentation and observed CLI behavior disagree, pin and test the supported behavior and record the limitation in the shared definition.

## Choose the declarative mechanism

| Requirement | Mechanism |
|---|---|
| Shared provider credentials and environment | `secret-strategy: universal-llm-consumer` and `execution.provider-env-mode: universal-llm-consumer` |
| Model passed through an environment variable | `execution.model-env-var` |
| `provider/model` must be rewritten for the CLI | `execution.model-env-provider-prefix`, or a harness for more complex transformations |
| Static engine configuration | `behaviors.config-file` with the correct path, content, and merge strategy |
| Gateway MCP output already matches the CLI | `behaviors.mcp.config-path` and an execution MCP config binding |
| Gateway MCP output needs another schema | `behaviors.mcp.config-adapter` |
| Runtime endpoint discovery or custom invocation | `behaviors.harness-script` using `awf_reflect.cjs` |
| No native MCP client | `engine.mcp: false` |

Use a `config-adapter` only to transform generated MCP configuration. Use a `harness-script` when endpoint selection, model transformation, prompt delivery, or CLI invocation requires runtime logic.

## Resolve LLM endpoints at runtime

Do not hard-code proxy ports or addresses in new shared engine definitions. Add a harness and resolve the configured endpoint from `/reflect` at execution time. Prefer the versioned helper beside the generated harness:

```javascript
const {
  fetchAWFReflect,
  resolveProviderEndpointFromReflect,
} = require("./awf_reflect.cjs");
```

The harness must:

1. check `AWF_REFLECT_ENABLED` before using the AWF endpoint
2. call `fetchAWFReflect()` and require a successful response
3. select the requested provider from `GH_AW_LLM_PROVIDER`
4. use only an endpoint with `configured: true`
5. map the resolved URL into the CLI's documented environment variable, flag, or config key
6. transform and validate the selected model using the discovered provider's syntax
7. read the prompt from `GH_AW_PROMPT`, spawn the CLI without shell interpolation, and preserve its exit status
8. fail with an actionable message when endpoint or model resolution is impossible

Use `resolveProviderEndpointFromReflect()` when the CLI accepts a base URL. Use `resolveOpenAICompatibleEndpointFromReflect()` when it needs an OpenAI-compatible host and request path separately, as in the shared Goose harness. Use `resolveMultiProviderFromReflect()` only when the CLI consumes a generated multi-provider catalog. Parse `/reflect` directly only when the shared helpers cannot represent the engine's contract.

`AWF_REFLECT_ENABLED=1` only indicates that reflection is available; it does not configure the CLI. The harness must fetch and apply the result. When AWF is disabled, preserve the CLI's documented environment-based fallback or fail clearly if the engine cannot run without reflection. See [LLM API Endpoint Discovery](llms.md) for the response shape and model-discovery behavior.

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

1. add or update `.github/workflows/shared/<id>.md`
2. import it from a minimal workflow that exercises the selected provider, model syntax, MCP mode, and prompt delivery
3. compile that workflow in strict mode and inspect the generated execution, config, MCP, and harness steps
4. if the schema surface changes, update `pkg/parser/schemas/main_workflow_schema.json`
5. if the engine becomes built-in, move the definition to `pkg/workflow/data/engines/<id>.md`, update the engine reference, and update tests that assert the catalog contents
6. run the relevant repository validation:

```bash
gh aw compile <workflow-name> --strict
make recompile
go test ./pkg/workflow/... ./pkg/parser/...
```

## Anti-patterns

- do not add a new bespoke `*_engine.go` wrapper for behavior that already fits `engine.behaviors`
- do not store install metadata, CLI args, or config-file templates partly in Go and partly in markdown without a clear need
- do not begin with a built-in engine when a shared agentic workflow definition can prove the contract first
- do not hard-code LLM proxy ports, container IPs, or a single provider endpoint when `/reflect` can resolve the selected provider
- do not claim MCP support until the generated gateway configuration matches the CLI's accepted schema and transport
- do not omit manifest files for engine-owned config that changes runtime behavior
- do not use a mismatched `runtime-id` unless an existing runtime adapter is intentionally being reused
