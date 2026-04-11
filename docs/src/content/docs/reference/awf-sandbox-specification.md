---
title: AWF Sandbox Specification
description: Formal specification for the Agent Workflow Firewall (AWF) sandbox configuration following W3C conventions
sidebar:
  order: 1355
---

# AWF Sandbox Specification

**Version**: 1.0.0
**Status**: Draft Specification
**Latest Version**: [awf-sandbox-specification](/gh-aw/reference/awf-sandbox-specification/)
**Editor**: GitHub Agentic Workflows Team

---

## Abstract

This specification defines the configuration format and behavioral requirements for the Agent Workflow Firewall (AWF), the network egress control and container isolation sandbox used by GitHub Agentic Workflows. AWF provides domain-based network filtering, credential isolation via an API proxy sidecar, container-level filesystem isolation, and engine-specific default policies. This specification also defines the extensible engine definition model that allows declarative configuration of inference providers, agentic engines, and combinations thereof.

## Status of This Document

This section describes the status of this document at the time of publication. This is a draft specification and may be updated, replaced, or made obsolete by other documents at any time.

This document is governed by the GitHub Agentic Workflows project specifications process.

## Table of Contents

1. [Introduction](#1-introduction)
2. [Conformance](#2-conformance)
3. [Architecture](#3-architecture)
4. [Sandbox Configuration](#4-sandbox-configuration)
5. [Network Egress Control](#5-network-egress-control)
6. [Firewall Configuration](#6-firewall-configuration)
7. [AWF Command Interface](#7-awf-command-interface)
8. [Agentic Engine Configuration](#8-agentic-engine-configuration)
9. [Engine-Specific Firewall Behavior](#9-engine-specific-firewall-behavior)
10. [Security Considerations](#10-security-considerations)
11. [Compliance Testing](#11-compliance-testing)

---

## 1. Introduction

### 1.1 Purpose

The Agent Workflow Firewall (AWF) provides a containerized sandbox environment for AI coding agent execution within GitHub Actions. It enforces network egress policies, isolates agent processes from the host environment, and proxies LLM API credentials to prevent exfiltration. This specification formally defines:

- The sandbox configuration format and its fields
- Network egress control via domain-based allow/block lists
- Firewall runtime parameters passed to the AWF binary
- The extensible engine definition model for configuring inference providers and agentic engines
- Engine-specific default firewall policies and domain lists

### 1.2 Scope

This specification covers:

- Sandbox top-level configuration (`sandbox.agent`, `sandbox.mcp`)
- Network permissions configuration (`network.allowed`, `network.blocked`, `network.firewall`)
- AWF command-line arguments and their mapping from workflow frontmatter
- Engine definition types (`EngineDefinition`, `ProviderSelection`, `AuthDefinition`)
- Engine-specific default domain lists and auto-enablement rules
- Container mount, memory, and environment variable isolation

This specification does NOT cover:

- MCP Gateway protocol behavior (see [MCP Gateway Specification](/gh-aw/reference/mcp-gateway/))
- Individual engine CLI installation or execution steps
- Threat detection or input sanitization layers (see [Security Architecture Specification](/gh-aw/specs/security-architecture-spec/))
- GitHub Actions workflow syntax

### 1.3 Design Goals

The AWF sandbox MUST be designed for:

- **Defense in Depth**: Multiple isolation layers (network, filesystem, credential) working independently
- **Least Privilege**: Default-deny network policy with explicit allowlists per engine
- **Transparency**: All firewall decisions logged and auditable via structured audit logs
- **Engine Agnosticism**: Uniform sandbox interface across all agentic engines (Copilot, Claude, Codex, Gemini, custom)
- **Extensibility**: Declarative engine definitions allowing new providers and auth strategies without code changes

---

## 2. Conformance

### 2.1 Conformance Classes

A **conforming AWF sandbox implementation** is one that satisfies all MUST, REQUIRED, and SHALL requirements in this specification.

A **partially conforming AWF sandbox implementation** is one that satisfies all MUST requirements but MAY lack support for optional features marked with SHOULD or MAY.

### 2.2 Requirements Notation

The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT", "SHOULD", "SHOULD NOT", "RECOMMENDED", "NOT RECOMMENDED", "MAY", and "OPTIONAL" in this document are to be interpreted as described in [RFC 2119](https://www.ietf.org/rfc/rfc2119.txt).

### 2.3 Compliance Levels

Implementations MUST support:

- **Level 1 (Required)**: Container isolation, domain-based network filtering, AWF binary execution, API proxy sidecar
- **Level 2 (Standard)**: SSL Bump HTTPS inspection, custom mounts, memory limits, environment variable exclusion, engine-specific defaults
- **Level 3 (Complete)**: All optional features including custom engine definitions, OAuth client-credentials auth, request shaping, CLI proxy sidecar

---

## 3. Architecture

### 3.1 Sandbox Model

```
┌─────────────────────────────────────────────────────────────┐
│                   GitHub Actions Runner                      │
│                                                             │
│  ┌───────────────────────────────────────────────────────┐  │
│  │              AWF (Agent Workflow Firewall)             │  │
│  │                                                       │  │
│  │  ┌─────────────┐  ┌──────────────┐  ┌─────────────┐  │  │
│  │  │  Network     │  │  API Proxy   │  │ Filesystem  │  │  │
│  │  │  Egress      │  │  Sidecar     │  │ Isolation   │  │  │
│  │  │  Filter      │  │  (LLM Auth)  │  │ (Mounts)    │  │  │
│  │  └──────┬──────┘  └──────┬───────┘  └──────┬──────┘  │  │
│  │         │                │                  │         │  │
│  │         ▼                ▼                  ▼         │  │
│  │  ┌───────────────────────────────────────────────┐   │  │
│  │  │           Agent Container (Docker)            │   │  │
│  │  │                                               │   │  │
│  │  │  ┌──────────────────────────────────────┐    │   │  │
│  │  │  │        Agentic Engine Runtime         │    │   │  │
│  │  │  │   (Copilot/Claude/Codex/Gemini/...)  │    │   │  │
│  │  │  └──────────────────────────────────────┘    │   │  │
│  │  └───────────────────────────────────────────────┘   │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                             │
│  ┌───────────────────────────────────────────────────────┐  │
│  │              MCP Gateway (Sidecar)                     │  │
│  │         (See MCP Gateway Specification)                │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### 3.2 Isolation Layers

The AWF sandbox provides three independent isolation layers:

1. **Network Egress Filter**: Domain-based allowlist/blocklist controlling all outbound network traffic from the agent container. Implemented via a forward proxy (Squid).
2. **API Proxy Sidecar**: Transparent credential injection for LLM provider APIs. The agent process never sees raw API keys; the sidecar intercepts outbound requests to known LLM endpoints and injects authentication headers.
3. **Filesystem Isolation**: The agent executes inside a Docker container with controlled mounts. The workspace is mounted read-write; auxiliary directories are mounted read-only.

### 3.3 Operational Model

The AWF operates as a wrapper command that:

1. Starts a Docker container with configured network policies
2. Launches an API proxy sidecar on the host
3. Configures DNS to route through the forward proxy
4. Executes the agent engine command inside the container
5. Logs all network activity for audit

---

## 4. Sandbox Configuration

### 4.1 Top-Level Structure

The `sandbox` field in workflow frontmatter configures the sandbox environment. It MUST conform to the following structure:

```yaml
sandbox:
  agent: <AgentSandboxConfig | "awf" | false>
  mcp: <MCPGatewayRuntimeConfig>
```

### 4.2 Agent Sandbox Configuration

#### 4.2.1 String Format

When `sandbox.agent` is a string, it MUST be one of the following values:

| Value | Description |
|-------|-------------|
| `awf` | Agent Workflow Firewall (default) |
| `default` | Alias for `awf` (backward compatibility) |
| `false` | Disables the agent sandbox |

When `sandbox.agent` is set to `false`, the firewall container is not started and the engine command runs directly on the runner host. The MCP gateway remains active.

#### 4.2.2 Object Format

When `sandbox.agent` is an object, it MUST conform to the `AgentSandboxConfig` structure:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | No | Agent sandbox identifier. MUST be `"awf"` or `"default"`. |
| `type` | string | No | Legacy field. SHOULD use `id` instead. |
| `command` | string | No | Custom command to replace the default AWF binary (e.g., a custom wrapper). |
| `args` | string[] | No | Additional arguments appended to the AWF command. |
| `env` | map[string]string | No | Environment variables set on the sandbox step. |
| `mounts` | string[] | No | Additional container mounts in `"source:destination:mode"` format. |
| `memory` | string | No | Memory limit for the AWF container (e.g., `"4g"`, `"8g"`). |

**Example — Object format with custom mounts and memory:**

```yaml
sandbox:
  agent:
    id: awf
    mounts:
      - "/opt/models:/models:ro"
      - "/opt/cache:/cache:rw"
    memory: "8g"
    args:
      - "--custom-flag"
```

#### 4.2.3 Mount Syntax

Each mount string MUST follow the format `"source:destination:mode"` where:

- `source` is the host path (MUST NOT be empty)
- `destination` is the container path (MUST NOT be empty)
- `mode` MUST be either `"ro"` (read-only) or `"rw"` (read-write)

Implementations MUST reject mounts that do not have exactly three colon-separated parts or use an invalid mode.

#### 4.2.4 Default Behavior

When the `sandbox` field is omitted from the workflow frontmatter, implementations MUST apply the following defaults:

1. `sandbox.agent` SHALL default to `awf`
2. The AWF firewall SHALL be enabled
3. The MCP gateway SHALL remain active

When `sandbox.agent` is explicitly set to `false`:

1. The AWF container SHALL NOT be started
2. The engine command SHALL run directly on the runner host
3. The MCP gateway SHALL remain active (it cannot be disabled)

### 4.3 MCP Gateway Runtime Configuration

The `sandbox.mcp` field configures the MCP Gateway sidecar. This specification defers the full MCP Gateway protocol behavior to the [MCP Gateway Specification](/gh-aw/reference/mcp-gateway/). The following fields are relevant to AWF sandbox integration:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `port` | integer | No | Gateway HTTP port. MUST be between 1 and 65535. |
| `api-key` | string | No | API key for gateway authentication. MAY contain `${{ secrets.* }}` expressions. |
| `version` | string | No | Gateway container image version. |
| `container` | string | No | Custom container image for the gateway. Default: `ghcr.io/github/gh-aw-mcpg`. |
| `domain` | string | No | Domain name for gateway access from inside the container. |
| `trusted-bots` | string[] | No | GitHub bot identities passed to the gateway's trust list. |
| `keepalive-interval` | integer | No | Keepalive ping interval in seconds for HTTP MCP backends. |
| `payload-dir` | string | No | Directory for large payload file exchange. Default: `/tmp/gh-aw/mcp-payloads`. |
| `payload-size-threshold` | integer | No | Size threshold (bytes) above which payloads are stored to disk. Default: `524288` (512 KB). |

---

## 5. Network Egress Control

### 5.1 Network Permissions Structure

The `network` field controls which domains the agent container can access. It MUST conform to the `NetworkPermissions` structure:

```yaml
network:
  allowed: <string[]>
  blocked: <string[]>
  firewall: <FirewallConfig | boolean>
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `allowed` | string[] | No | Domains or ecosystem identifiers permitted for outbound access. |
| `blocked` | string[] | No | Domains or ecosystem identifiers blocked from outbound access. Blocked entries take precedence over allowed. |
| `firewall` | object or boolean | No | AWF firewall configuration. When `true`, enables with defaults. When an object, configures specific AWF parameters. |

#### 5.1.1 Access Levels

Network permissions follow the principle of least privilege:

1. **Default Allow List** (`network: defaults`): Basic infrastructure domains only (certificates, JSON schema, Ubuntu, common package mirrors).
2. **Selective Access** (`network: { allowed: [...] }`): Only listed domains and ecosystems are accessible.
3. **No Access** (`network: {}`): All network access denied. The `allowed` list is empty and `ExplicitlyDefined` is true.
4. **Unrestricted** (`network: { allowed: ["*"] }`): All domains permitted. The AWF firewall SHOULD NOT be auto-enabled.

### 5.2 Allowed Domains

#### 5.2.1 Domain Format

Entries in the `allowed` list MUST be one of:

- **Ecosystem identifier**: A predefined keyword (e.g., `"defaults"`, `"github"`, `"python"`, `"node"`) that expands to a curated set of domains.
- **Literal domain**: A specific domain name (e.g., `"api.example.com"`). Subdomains are automatically included (e.g., `"github.com"` allows `"api.github.com"`).
- **Wildcard pattern**: A domain with a `*` prefix (e.g., `"*.example.com"`) matching any subdomain.
- **Protocol-qualified domain**: A domain prefixed with `"https://"` or `"http://"` to restrict to a specific protocol (e.g., `"https://secure.api.example.com"`).
- **Wildcard `"*"`**: Permits all outbound network traffic. When present, the firewall SHOULD NOT be auto-enabled.

#### 5.2.2 Ecosystem Identifiers

The following ecosystem identifiers MUST be supported:

| Identifier | Description |
|------------|-------------|
| `defaults` | Basic infrastructure: certificate authorities, OCSP responders, JSON schema hosts, Ubuntu archives, common package mirrors, Microsoft sources |
| `github` | GitHub platform: `github.com`, `api.github.com`, `*.githubusercontent.com`, `ghcr.io`, etc. |
| `python` | Python ecosystem: `pypi.org`, `files.pythonhosted.org`, etc. |
| `node` | Node.js ecosystem: `registry.npmjs.org`, `nodejs.org`, etc. |
| `go` | Go ecosystem: `proxy.golang.org`, `sum.golang.org`, etc. |
| `java` | Java ecosystem: Maven Central, Gradle repositories |
| `ruby` | Ruby ecosystem: `rubygems.org`, etc. |
| `dotnet` | .NET ecosystem: `nuget.org`, `api.nuget.org`, etc. |
| `dev-tools` | Common developer tools and CI/CD services |
| `local` | Loopback/localhost addresses |

Compound ecosystem identifiers (e.g., `"default-safe-outputs"`) MAY be supported. They expand to the union of their component ecosystems.

#### 5.2.3 Runtime-Derived Domains

When `runtimes` are specified in the workflow frontmatter (e.g., `runtimes: { node: { version: "20" } }`), the implementation MUST automatically include the corresponding ecosystem domains in the allowed list.

### 5.3 Blocked Domains

Entries in the `blocked` list follow the same format as `allowed` entries. Blocked entries MUST take precedence over allowed entries. All subdomains of a blocked domain MUST also be blocked.

### 5.4 Engine Default Domains

Each built-in engine defines a set of default domains REQUIRED for its authentication and operation. Implementations MUST include these domains in the allowed list when the corresponding engine is active.

#### 5.4.1 Copilot Default Domains

The Copilot engine REQUIRES the following domains:

- `api.business.githubcopilot.com`
- `api.enterprise.githubcopilot.com`
- `api.github.com`
- `api.githubcopilot.com`
- `api.individual.githubcopilot.com`
- `github.com`
- `host.docker.internal`
- `raw.githubusercontent.com`
- `registry.npmjs.org`
- `telemetry.enterprise.githubcopilot.com`

#### 5.4.2 Claude Default Domains

The Claude engine REQUIRES a larger set of domains including `api.anthropic.com`, `anthropic.com`, `statsig.anthropic.com`, certificate revocation endpoints, GitHub domains, package registries, and other infrastructure domains. The full list contains approximately 45 entries.

#### 5.4.3 Codex Default Domains

The Codex engine REQUIRES the following domains:

- `172.30.0.1` (AWF gateway IP for Rust DNS compatibility)
- `api.openai.com`
- `host.docker.internal`
- `openai.com`

#### 5.4.4 Gemini Default Domains

The Gemini engine REQUIRES the following domains:

- `*.googleapis.com`
- `generativelanguage.googleapis.com`
- `github.com`
- `host.docker.internal`
- `raw.githubusercontent.com`
- `registry.npmjs.org`

---

## 6. Firewall Configuration

### 6.1 FirewallConfig Structure

The `network.firewall` field, when an object, MUST conform to the `FirewallConfig` structure:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | boolean | `true` (when firewall is configured) | Enable or disable AWF. |
| `version` | string | Latest stable version | AWF binary and Docker image version (e.g., `"v0.25.18"`). |
| `args` | string[] | `[]` | Additional arguments passed to the AWF binary. |
| `log_level` | string | `"info"` | AWF log level. MUST be one of: `"debug"`, `"info"`, `"warn"`, `"error"`. |
| `ssl_bump` | boolean | `false` | Enable SSL Bump for HTTPS content inspection (allows URL path filtering). |
| `allow_urls` | string[] | `[]` | URL patterns to allow for HTTPS (REQUIRES `ssl_bump: true`). |

**Example — Firewall with SSL Bump:**

```yaml
network:
  allowed:
    - defaults
    - github
  firewall:
    enabled: true
    version: "v0.25.18"
    log_level: debug
    ssl_bump: true
    allow_urls:
      - "https://github.com/githubnext/*"
```

### 6.2 Firewall Auto-Enablement

The AWF firewall MUST be automatically enabled under the following conditions:

1. The engine is `copilot`, `codex`, or `claude`
2. Network restrictions are present (non-empty allowed or blocked lists)
3. The allowed list does NOT contain `"*"` (wildcard unrestricted access)
4. `sandbox.agent` is NOT set to `false`
5. No explicit `network.firewall` configuration already exists

When auto-enabled, the implementation MUST create a `FirewallConfig` with `enabled: true` and all other fields at their defaults.

### 6.3 Firewall Disable Validation

When `network.firewall.enabled` is `false` but network restrictions are specified in `network.allowed`:

- In **non-strict mode**: The implementation SHOULD emit a warning indicating network may not be properly sandboxed.
- In **strict mode**: The implementation MUST return an error indicating the firewall cannot be disabled when network restrictions are set.

### 6.4 Log Level Validation

The `log_level` field MUST be validated against the allowed values: `"debug"`, `"info"`, `"warn"`, `"error"`. An empty string is permitted and defaults to `"info"` at runtime. Any other value MUST be rejected with a validation error.

### 6.5 SSL Bump Configuration

SSL Bump enables HTTPS content inspection, allowing URL path-level filtering instead of domain-only filtering. When `ssl_bump` is `true`:

1. The `--ssl-bump` flag MUST be passed to the AWF binary.
2. If `allow_urls` is non-empty, the `--allow-urls` flag MUST be passed with a comma-separated list of URL patterns.
3. `allow_urls` without `ssl_bump: true` SHOULD be ignored (the URLs are only meaningful when SSL Bump is active).

---

## 7. AWF Command Interface

### 7.1 Command Structure

The AWF command MUST be constructed as follows:

```
<command-prefix> <expandable-args> <safe-args> -- <shell-wrapped-engine-command>
```

Where:

- `<command-prefix>` is `sudo -E awf` by default, or a custom command from `sandbox.agent.command`
- `<expandable-args>` are arguments containing shell variables (e.g., `${GITHUB_WORKSPACE}`) that MUST NOT be single-quoted
- `<safe-args>` are arguments that are safely shell-escaped
- `--` separates AWF arguments from the engine command
- `<shell-wrapped-engine-command>` is the engine command wrapped in `/bin/bash -c '...'`

### 7.2 Required Arguments

The following arguments MUST always be present:

| Argument | Value | Description |
|----------|-------|-------------|
| `--env-all` | (flag) | Pass all environment variables to the container. |
| `--container-workdir` | `"${GITHUB_WORKSPACE}"` | Set the container working directory. |
| `--allow-domains` | Comma-separated domains | Allowed outbound domains (see Section 5). |
| `--log-level` | `"info"` (default) | AWF log verbosity. |
| `--proxy-logs-dir` | `/tmp/gh-aw/sandbox/firewall/logs` | Directory for proxy log files. |
| `--audit-dir` | `/tmp/gh-aw/sandbox/firewall/audit` | Directory for audit files (policy-manifest, squid.conf). |
| `--enable-host-access` | (flag) | Enable access to host services (API proxy, MCP gateway). |
| `--image-tag` | AWF version (without `v` prefix) | Pin Docker image version to match installed binary. |
| `--skip-pull` | (flag) | Skip pulling images (pre-downloaded during setup). |
| `--enable-api-proxy` | (flag) | Enable the API proxy sidecar for LLM credential isolation. |

### 7.3 Conditional Arguments

The following arguments are conditionally added based on configuration:

| Argument | Condition | Description |
|----------|-----------|-------------|
| `--tty` | Engine requires TTY (e.g., Claude) | Allocate a pseudo-TTY in the container. |
| `--exclude-env <VAR>` | AWF version ≥ v0.25.3 and secret env vars present | Exclude specific environment variables from the container (one flag per variable). |
| `--block-domains <domains>` | `network.blocked` is non-empty | Comma-separated blocked domains. |
| `--mount <mount>` | `sandbox.agent.mounts` is non-empty | Additional container mounts. |
| `--memory-limit <limit>` | `sandbox.agent.memory` is non-empty | Container memory limit (e.g., `"4g"`). |
| `--ssl-bump` | `network.firewall.ssl_bump` is `true` | Enable HTTPS content inspection. |
| `--allow-urls <urls>` | `network.firewall.allow_urls` is non-empty and SSL Bump enabled | Comma-separated URL patterns for HTTPS. |
| `--openai-api-target <host>` | `OPENAI_BASE_URL` in engine.env | Custom OpenAI-compatible API target hostname. |
| `--anthropic-api-target <host>` | `ANTHROPIC_BASE_URL` in engine.env | Custom Anthropic-compatible API target hostname. |
| `--copilot-api-target <host>` | `engine.api-target` or `GITHUB_COPILOT_BASE_URL` in engine.env | Custom Copilot API target hostname. |
| `--openai-api-base-path <path>` | `OPENAI_BASE_URL` contains a path component | URL path prefix for OpenAI-compatible endpoints. |
| `--anthropic-api-base-path <path>` | `ANTHROPIC_BASE_URL` contains a path component | URL path prefix for Anthropic-compatible endpoints. |
| `--difc-proxy-host <host:port>` | `cli-proxy` feature flag enabled and AWF ≥ v0.25.17 | CLI proxy sidecar host address. |
| `--difc-proxy-ca-cert <path>` | `cli-proxy` feature flag enabled and AWF ≥ v0.25.17 | CA certificate for CLI proxy TLS. |
| `--allow-host-service-ports <ports>` | Workflow defines service containers with port mappings | Ports for host service access from the container. |

### 7.4 Standard Mounts

The following mounts MUST be applied to all AWF container invocations:

| Mount | Mode | Description |
|-------|------|-------------|
| `${GITHUB_WORKSPACE}` → `${GITHUB_WORKSPACE}` | rw | Agent working directory (repository checkout). |
| `${RUNNER_TEMP}/gh-aw` → `${RUNNER_TEMP}/gh-aw` | ro | gh-aw configuration and prompt files. |
| `${RUNNER_TEMP}/gh-aw` → `/host${RUNNER_TEMP}/gh-aw` | ro | Host-path alias for cross-reference. |

When `safe_outputs.upload_artifact` is configured, an additional mount MUST be added:

| Mount | Mode | Description |
|-------|------|-------------|
| `${RUNNER_TEMP}/gh-aw/safeoutputs/upload-artifacts` | rw | Staging directory for artifact uploads from inside the container. |

### 7.5 Environment Variable Exclusion

To prevent credential exfiltration via shell introspection (`env`, `printenv`), the implementation MUST exclude environment variables whose values reference GitHub Actions secrets (`${{ secrets.* }}`).

For each secret-bearing environment variable name returned by `GetRequiredSecretNames()`:

1. If the AWF version supports `--exclude-env` (≥ v0.25.3), the implementation MUST pass `--exclude-env <VAR_NAME>` for each excluded variable.
2. Excluded variables MUST be sorted alphabetically for deterministic output.
3. The API proxy sidecar handles authentication transparently, so excluded variables are not needed inside the container.

---

## 8. Agentic Engine Configuration

### 8.1 Engine Definition Model

The engine definition model provides a declarative layer for describing AI engines independently of their runtime implementation. This allows the catalog to carry identity and provider information without coupling to specific `CodingAgentEngine` runtime adapters.

#### 8.1.1 EngineDefinition Structure

An engine definition MUST conform to the following structure:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Unique engine identifier (e.g., `"copilot"`, `"claude"`, `"codex"`, `"gemini"`, or a custom ID). |
| `display-name` | string | No | Human-readable engine name (e.g., `"GitHub Copilot"`). |
| `description` | string | No | Description of the engine's capabilities. |
| `runtime-id` | string | No | Maps to a registered `CodingAgentEngine` in the `EngineRegistry`. Defaults to `id` when omitted. |
| `provider` | ProviderSelection | No | Inference provider configuration (see Section 8.2). |
| `models` | ModelSelection | No | Default and supported model configuration (see Section 8.3). |
| `auth` | AuthBinding[] | No | Authentication role-to-secret mappings. |
| `options` | map | No | Engine-specific options (free-form key-value pairs). |

#### 8.1.2 Built-in Engines

The engine catalog MUST pre-register the following built-in engine definitions:

| Engine ID | Display Name | Runtime ID | Description |
|-----------|-------------|------------|-------------|
| `copilot` | GitHub Copilot | `copilot` | GitHub Copilot CLI with agent support |
| `claude` | Claude | `claude` | Anthropic Claude Code coding agent |
| `codex` | Codex | `codex` | OpenAI Codex/GPT engine |
| `gemini` | Gemini | `gemini` | Google Gemini CLI coding agent |

The default engine, when no `engine` field is specified, MUST be `copilot`.

#### 8.1.3 Engine Resolution

Engine resolution MUST follow this order:

1. **Exact catalog match**: Look up the engine ID in the catalog definitions.
2. **Runtime-ID prefix fallback**: If no exact match, attempt a prefix match against registered runtime adapters (for backward compatibility, e.g., `"codex-experimental"` matching the `"codex"` runtime).
3. **Validation error**: If no match is found, return an error listing valid engines with optional fuzzy-match suggestions.

### 8.2 Provider Selection

The `ProviderSelection` type identifies the AI inference provider for an engine. It supports standard and custom backends.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | No | Provider identifier (e.g., `"anthropic"`, `"openai"`, `"google"`, or a custom provider name). |
| `auth` | AuthDefinition | No | Authentication configuration for the provider (see Section 8.4). |
| `request` | RequestShape | No | Request transformation configuration for non-standard backends (see Section 8.5). |

**Example — Custom provider with OpenAI-compatible endpoint:**

```yaml
engine:
  runtime:
    id: codex
  provider:
    name: azure-openai
    auth:
      strategy: api-key
      secret: AZURE_OPENAI_KEY
      header-name: api-key
    request:
      path-template: "/openai/deployments/{model}/chat/completions"
      query:
        api-version: "2024-10-01-preview"
```

### 8.3 Model Selection

The `ModelSelection` type specifies default and supported models for an engine.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `default` | string | No | Default model to use when no model is explicitly specified. |
| `supported` | string[] | No | List of supported model identifiers. |

### 8.4 Authentication Definition

The `AuthDefinition` type describes how an engine authenticates with its provider backend. Three authentication strategies are supported:

#### 8.4.1 API Key Strategy (`api-key`)

The default strategy. Sends a raw API key via a specified HTTP header.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `strategy` | string | No | MUST be `"api-key"` (default when `secret` is set and `strategy` is empty). |
| `secret` | string | Yes | GitHub Actions secret name holding the API key. |
| `header-name` | string | No | HTTP header name for the key injection (e.g., `"api-key"`, `"Authorization"`). |

**Example:**

```yaml
auth:
  strategy: api-key
  secret: ANTHROPIC_API_KEY
  header-name: x-api-key
```

#### 8.4.2 OAuth Client Credentials Strategy (`oauth-client-credentials`)

Exchanges client credentials for a bearer token before each API call.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `strategy` | string | Yes | MUST be `"oauth-client-credentials"`. |
| `token-url` | string | Yes | OAuth token endpoint URL. |
| `client-id-ref` | string | Yes | Secret name holding the OAuth client ID. |
| `client-secret-ref` | string | Yes | Secret name holding the OAuth client secret. |
| `token-field` | string | No | JSON field in the token response containing the access token. Defaults to `"access_token"`. |
| `header-name` | string | No | HTTP header for injecting the obtained token. |

**Example:**

```yaml
auth:
  strategy: oauth-client-credentials
  token-url: "https://auth.example.com/oauth/token"
  client-id-ref: OAUTH_CLIENT_ID
  client-secret-ref: OAUTH_CLIENT_SECRET
  token-field: access_token
```

#### 8.4.3 Bearer Token Strategy (`bearer`)

Sends a pre-obtained token as a standard `Authorization: Bearer` header.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `strategy` | string | Yes | MUST be `"bearer"`. |
| `secret` | string | Yes | Secret name holding the bearer token. |

**Example:**

```yaml
auth:
  strategy: bearer
  secret: MY_BEARER_TOKEN
```

#### 8.4.4 Required Secrets

Implementations MUST compute the required secret names from the `AuthDefinition`:

- For `api-key` and `bearer` strategies: the `secret` field.
- For `oauth-client-credentials` strategy: the `client-id-ref` and `client-secret-ref` fields.

### 8.5 Request Shaping

The `RequestShape` type describes non-standard URL and body transformations applied to API calls before they reach the provider backend.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path-template` | string | No | URL path template with `{model}` and other variable placeholders. |
| `query` | map[string]string | No | Static or template query parameters appended to every request. |
| `body-inject` | map[string]string | No | Key-value pairs injected into the JSON request body. Values MAY contain `{SECRET_NAME}` template references. |

**Example — Azure OpenAI with path-template and API version:**

```yaml
request:
  path-template: "/openai/deployments/{model}/chat/completions"
  query:
    api-version: "2024-10-01-preview"
```

### 8.6 Inline Engine Definition

Engines MAY be defined inline in the workflow frontmatter using the `engine.runtime` and `engine.provider` sub-objects. This allows combining a registered runtime adapter with a custom provider configuration without modifying the engine catalog.

```yaml
engine:
  runtime:
    id: codex
    version: "latest"
  provider:
    id: openai
    model: gpt-4
    auth:
      strategy: api-key
      secret: OPENAI_API_KEY
    request:
      path-template: "/v1/chat/completions"
```

When `engine.runtime` is present, the implementation MUST:

1. Set `IsInlineDefinition` to `true`.
2. Resolve the runtime adapter from the `runtime.id` field via the engine registry.
3. Apply provider configuration from the `provider` sub-object to the resolved engine target.

### 8.7 Engine Authentication Secrets

Each built-in engine defines a primary secret and optional alternative secrets:

| Engine | Primary Secret | Alternative Secrets | Key URL |
|--------|---------------|-------------------|---------|
| `copilot` | `COPILOT_GITHUB_TOKEN` | — | `https://github.com/settings/personal-access-tokens/new` |
| `claude` | `ANTHROPIC_API_KEY` | — | `https://console.anthropic.com/settings/keys` |
| `codex` | `OPENAI_API_KEY` | `CODEX_API_KEY` | `https://platform.openai.com/api-keys` |
| `gemini` | `GEMINI_API_KEY` | — | `https://aistudio.google.com/app/apikey` |

System-level secrets that are not engine-specific:

| Secret | Required | Description |
|--------|----------|-------------|
| `GH_AW_GITHUB_TOKEN` | Optional | PAT for GitHub write operations using a user identity. |
| `GH_AW_AGENT_TOKEN` | Optional | PAT for agent/bot assignment to issues or pull requests. |
| `GH_AW_GITHUB_MCP_SERVER_TOKEN` | Optional | Read-mostly token for isolating MCP server permissions. |

### 8.8 Custom API Targets

Engines MAY configure custom LLM API endpoints for corporate routers, Azure OpenAI, self-hosted APIs, or GitHub Enterprise:

| Environment Variable | AWF Flag | Description |
|---------------------|----------|-------------|
| `OPENAI_BASE_URL` | `--openai-api-target`, `--openai-api-base-path` | Custom OpenAI-compatible endpoint. |
| `ANTHROPIC_BASE_URL` | `--anthropic-api-target`, `--anthropic-api-base-path` | Custom Anthropic-compatible endpoint. |
| `GITHUB_COPILOT_BASE_URL` | `--copilot-api-target` | Custom Copilot endpoint (fallback when `engine.api-target` is not set). |

When a custom base URL contains a path component (e.g., `/serving-endpoints`), the implementation MUST extract and pass the path via the corresponding `--*-api-base-path` flag.

---

## 9. Engine-Specific Firewall Behavior

### 9.1 Copilot and Codex

For `copilot` and `codex` engines:

1. The firewall MUST be auto-enabled when network restrictions are present (see Section 6.2).
2. The Copilot default domains (Section 5.4.1) or Codex default domains (Section 5.4.3) MUST be included in the allowed list.
3. The `--copilot-api-target` flag SHOULD be set when `engine.api-target` or `GITHUB_COPILOT_BASE_URL` is configured.

### 9.2 Claude

For the `claude` engine:

1. The firewall MUST be auto-enabled when network restrictions are present (see Section 6.2).
2. The Claude default domains (Section 5.4.2) MUST be included in the allowed list.
3. The `--tty` flag MUST be passed to AWF (Claude requires a pseudo-TTY).
4. The `--anthropic-api-target` flag SHOULD be set when `ANTHROPIC_BASE_URL` is configured.

### 9.3 Gemini

For the `gemini` engine:

1. The firewall MAY be auto-enabled when network restrictions are present.
2. The Gemini default domains (Section 5.4.4) MUST be included in the allowed list.

### 9.4 Custom Engines

For engines defined via inline definitions (Section 8.6) or custom engine catalog entries:

1. The firewall behavior SHOULD follow the same auto-enablement rules as the runtime adapter they reference.
2. Custom engines MUST explicitly list all required domains in the `network.allowed` field; no implicit defaults are assumed beyond the runtime adapter's base set.
3. Custom API targets MUST be configured via `engine.env` environment variables.

---

## 10. Security Considerations

### 10.1 Credential Isolation

The AWF API proxy sidecar MUST intercept outbound requests to known LLM provider endpoints and inject authentication headers transparently. The agent process MUST NOT have direct access to raw API keys or tokens.

Implementations MUST use `--exclude-env` (when supported) to prevent the agent from reading secret values via shell introspection commands (`env`, `printenv`).

### 10.2 Network Egress

All outbound network traffic from the agent container MUST pass through the AWF forward proxy. The proxy MUST enforce the configured allow/block domain lists. Domains not in the allowed list and not blocked MUST be denied by default (default-deny policy).

### 10.3 Filesystem Isolation

The agent MUST execute inside a Docker container with controlled mount points. The workspace (`${GITHUB_WORKSPACE}`) MUST be mounted read-write. Auxiliary directories (gh-aw configuration, prompts) SHOULD be mounted read-only.

### 10.4 Audit Logging

The AWF MUST produce structured audit logs in the configured audit directory (`/tmp/gh-aw/sandbox/firewall/audit`). These logs MUST include:

- Policy manifest (resolved domain lists and firewall configuration)
- Proxy access log (all HTTP/HTTPS requests with allow/deny decisions)
- Redacted Docker Compose configuration

Implementations SHOULD upload audit logs as GitHub Actions artifacts for post-run inspection.

### 10.5 SSL Bump Considerations

When SSL Bump is enabled, the AWF intercepts and decrypts HTTPS traffic for content inspection. This has the following implications:

1. The AWF generates a dynamic CA certificate that MUST be injected into the container's trust store.
2. Applications that pin certificates MAY fail if they do not trust the injected CA.
3. SSL Bump SHOULD only be enabled when URL path-level filtering is required.

---

## 11. Compliance Testing

### 11.1 Sandbox Configuration Tests

- **T-SBX-001**: Default sandbox configuration creates AWF agent when `sandbox` field is omitted.
- **T-SBX-002**: `sandbox.agent: false` disables the AWF container but keeps MCP gateway active.
- **T-SBX-003**: `sandbox.agent: awf` explicitly enables AWF.
- **T-SBX-004**: Legacy `sandbox.type: default` is treated as AWF.
- **T-SBX-005**: Object format `sandbox.agent.id: awf` is accepted.
- **T-SBX-006**: Invalid sandbox agent type is rejected.

### 11.2 Mount Validation Tests

- **T-MNT-001**: Valid mount format `"source:dest:ro"` is accepted.
- **T-MNT-002**: Valid mount format `"source:dest:rw"` is accepted.
- **T-MNT-003**: Mount with invalid mode (not `"ro"` or `"rw"`) is rejected.
- **T-MNT-004**: Mount with fewer than 3 colon-separated parts is rejected.
- **T-MNT-005**: Mount with empty source is rejected.
- **T-MNT-006**: Mount with empty destination is rejected.

### 11.3 Network Permissions Tests

- **T-NET-001**: `network: defaults` expands to default ecosystem domains.
- **T-NET-002**: Ecosystem identifiers are expanded to their domain lists.
- **T-NET-003**: `network: {}` results in empty allowed list (deny all).
- **T-NET-004**: `network: { allowed: ["*"] }` permits all domains.
- **T-NET-005**: Blocked domains take precedence over allowed domains.
- **T-NET-006**: Runtime-derived domains are included in allowed list.
- **T-NET-007**: Protocol-qualified domains preserve protocol prefix.
- **T-NET-008**: Wildcard domain patterns are handled correctly.

### 11.4 Firewall Configuration Tests

- **T-FWL-001**: Valid log levels (`"debug"`, `"info"`, `"warn"`, `"error"`) are accepted.
- **T-FWL-002**: Invalid log level is rejected.
- **T-FWL-003**: Empty log level defaults to `"info"`.
- **T-FWL-004**: Firewall auto-enables for Copilot engine with network restrictions.
- **T-FWL-005**: Firewall auto-enables for Claude engine with network restrictions.
- **T-FWL-006**: Firewall does NOT auto-enable when allowed contains `"*"`.
- **T-FWL-007**: Firewall does NOT auto-enable when `sandbox.agent: false`.
- **T-FWL-008**: Explicit `network.firewall` configuration is preserved (not overridden).
- **T-FWL-009**: SSL Bump arguments are correctly generated.
- **T-FWL-010**: `allow_urls` requires `ssl_bump: true`.

### 11.5 AWF Command Tests

- **T-CMD-001**: Default command prefix is `sudo -E awf`.
- **T-CMD-002**: Custom command from `sandbox.agent.command` replaces default.
- **T-CMD-003**: `--env-all` is always present.
- **T-CMD-004**: `--exclude-env` flags are sorted alphabetically.
- **T-CMD-005**: `--exclude-env` is skipped for AWF versions < v0.25.3.
- **T-CMD-006**: `--allow-domains` contains engine defaults merged with user-specified domains.
- **T-CMD-007**: `--block-domains` is present when blocked domains are specified.
- **T-CMD-008**: Custom mounts from agent config are included.
- **T-CMD-009**: Memory limit is passed via `--memory-limit`.
- **T-CMD-010**: Custom API targets are correctly extracted from environment variables.

### 11.6 Engine Definition Tests

- **T-ENG-001**: All four built-in engines are registered in the catalog.
- **T-ENG-002**: Exact engine ID lookup resolves correctly.
- **T-ENG-003**: Runtime-ID prefix fallback resolves correctly (e.g., `"codex-experimental"`).
- **T-ENG-004**: Unknown engine ID produces a validation error with suggestions.
- **T-ENG-005**: Inline engine definition (`engine.runtime.id`) resolves via registry.
- **T-ENG-006**: Provider auth strategies are correctly parsed (api-key, oauth-client-credentials, bearer).
- **T-ENG-007**: Required secret names are computed from AuthDefinition.

### 11.7 Compliance Checklist

| Requirement | Test ID | Level | Status |
|-------------|---------|-------|--------|
| Default sandbox is AWF | T-SBX-001 | 1 | Required |
| Agent disable keeps MCP | T-SBX-002 | 1 | Required |
| Mount format validation | T-MNT-001–006 | 1 | Required |
| Domain-based filtering | T-NET-001–005 | 1 | Required |
| Firewall auto-enablement | T-FWL-004–008 | 1 | Required |
| AWF command construction | T-CMD-001–010 | 1 | Required |
| Log level validation | T-FWL-001–003 | 1 | Required |
| Runtime-derived domains | T-NET-006 | 2 | Required |
| SSL Bump | T-FWL-009–010 | 2 | Required |
| Environment variable exclusion | T-CMD-004–005 | 2 | Required |
| Engine catalog resolution | T-ENG-001–004 | 2 | Required |
| Inline engine definitions | T-ENG-005 | 3 | Optional |
| OAuth client-credentials | T-ENG-006 | 3 | Optional |
| Custom API targets | T-CMD-010 | 3 | Optional |

---

## References

### Normative References

- **[RFC 2119]** Bradner, S., "Key words for use in RFCs to Indicate Requirement Levels", BCP 14, RFC 2119, March 1997.
- **[MCP Gateway Specification]** GitHub Agentic Workflows Team, "MCP Gateway Specification", Draft, [/gh-aw/reference/mcp-gateway/](/gh-aw/reference/mcp-gateway/).
- **[Security Architecture Specification]** GitHub Agentic Workflows Team, "Security Architecture Specification", Candidate Recommendation.

### Informative References

- **[AWF Repository]** GitHub gh-aw-firewall project, `ghcr.io/github/gh-aw-firewall`.
- **[Docker Security]** Docker, Inc., "Docker Security", https://docs.docker.com/engine/security/.
- **[Squid Proxy]** Squid Project, "Squid: Optimising Web Delivery", https://www.squid-cache.org/.

---

## Change Log

### Version 1.0.0 (Draft)

- Initial specification release
- Sandbox configuration format (agent, MCP gateway)
- Network egress control (allowed/blocked domains, ecosystems, protocol-qualified, wildcards)
- Firewall configuration (log level, SSL Bump, auto-enablement, version pinning)
- AWF command interface (required/conditional arguments, standard mounts, env exclusion)
- Agentic engine configuration (EngineDefinition, ProviderSelection, AuthDefinition, RequestShape)
- Engine-specific firewall behavior (Copilot, Claude, Codex, Gemini, custom)
- Security considerations (credential isolation, network egress, filesystem, audit logging, SSL Bump)
- Compliance testing framework (47 test IDs across 6 categories)

---

*Copyright © 2026 GitHub, Inc. All rights reserved.*
