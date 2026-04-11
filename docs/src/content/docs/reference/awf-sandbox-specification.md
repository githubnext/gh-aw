---
title: AWF Sandbox Specification
description: Formal specification for the Agent Workflow Firewall (AWF) binary interface, network filtering, and container isolation
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

This specification defines the command-line interface, network filtering behavior, and container isolation model for the Agent Workflow Firewall (AWF) binary. AWF is a Docker-based sandbox that wraps an agent engine command, providing domain-based network egress filtering, an API proxy sidecar for credential isolation, and filesystem isolation via controlled mounts. This document specifies what AWF accepts as input (CLI arguments), how it processes that input, and the behavioral guarantees it provides.

> **Note**: This specification describes the AWF binary interface — what AWF itself consumes and enforces. Workflow frontmatter fields, ecosystem domain identifiers, engine definitions, and other compiler-level constructs that produce AWF invocations are out of scope. The compiler resolves those higher-level abstractions into the flat domain lists and CLI arguments described here.

## Status of This Document

This section describes the status of this document at the time of publication. This is a draft specification and may be updated, replaced, or made obsolete by other documents at any time.

This document is governed by the GitHub Agentic Workflows project specifications process.

## Table of Contents

1. [Introduction](#1-introduction)
2. [Conformance](#2-conformance)
3. [Architecture](#3-architecture)
4. [AWF Command Interface](#4-awf-command-interface)
5. [Network Filtering](#5-network-filtering)
6. [Container Isolation](#6-container-isolation)
7. [API Proxy Sidecar](#7-api-proxy-sidecar)
8. [SSL Bump (HTTPS Inspection)](#8-ssl-bump-https-inspection)
9. [Logging and Audit](#9-logging-and-audit)
10. [Security Considerations](#10-security-considerations)
11. [Compliance Testing](#11-compliance-testing)

---

## 1. Introduction

### 1.1 Purpose

The Agent Workflow Firewall (AWF) is a command-line binary that provides a containerized sandbox for AI coding agent execution. It wraps an arbitrary engine command inside a Docker container with network egress filtering, credential isolation, and filesystem isolation. This specification formally defines:

- The AWF CLI argument interface
- Domain-based network filtering behavior (allow/block lists of resolved domains)
- Container mount and memory configuration
- API proxy sidecar for transparent LLM credential injection
- SSL Bump HTTPS content inspection
- Audit logging format and requirements

### 1.2 Scope

This specification covers:

- AWF binary command-line arguments and their semantics
- Network filtering of flat domain lists (literal domains, wildcards, protocol-qualified)
- Container filesystem mounts, memory limits, and environment variable handling
- API proxy sidecar behavior for LLM credential isolation
- SSL Bump HTTPS inspection mode
- Audit log output format

This specification does NOT cover:

- Workflow frontmatter syntax or YAML configuration format — these are compiler concerns that produce AWF invocations
- Ecosystem domain identifiers (e.g., `"defaults"`, `"github"`, `"python"`) — these are compiler constructs resolved into flat domain lists before AWF is invoked
- Engine definitions, provider selection, or authentication strategies — these are compiler-level types that determine which secrets and domains to pass to AWF
- MCP Gateway protocol behavior (see [MCP Gateway Specification](/gh-aw/reference/mcp-gateway/))
- Threat detection or input sanitization layers (see [Security Architecture Specification](/gh-aw/specs/security-architecture-spec/))

### 1.3 Design Goals

The AWF binary MUST be designed for:

- **Defense in Depth**: Multiple isolation layers (network, filesystem, credential) working independently
- **Least Privilege**: Default-deny network policy; only explicitly listed domains are reachable
- **Transparency**: All firewall decisions logged and auditable via structured audit logs
- **Engine Agnosticism**: AWF wraps any engine command uniformly — it does not know or care which engine is running inside the container
- **Simplicity**: AWF accepts flat domain lists and simple flags; all higher-level abstraction is handled by the caller (compiler)

---

## 2. Conformance

### 2.1 Conformance Classes

A **conforming AWF implementation** is one that satisfies all MUST, REQUIRED, and SHALL requirements in this specification.

A **partially conforming AWF implementation** is one that satisfies all MUST requirements but MAY lack support for optional features marked with SHOULD or MAY.

### 2.2 Requirements Notation

The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT", "SHOULD", "SHOULD NOT", "RECOMMENDED", "NOT RECOMMENDED", "MAY", and "OPTIONAL" in this document are to be interpreted as described in [RFC 2119](https://www.ietf.org/rfc/rfc2119.txt).

### 2.3 Compliance Levels

Implementations MUST support:

- **Level 1 (Required)**: Container isolation, domain-based network filtering, AWF binary execution, API proxy sidecar
- **Level 2 (Standard)**: SSL Bump HTTPS inspection, custom mounts, memory limits, environment variable exclusion
- **Level 3 (Complete)**: All optional features including CLI proxy sidecar, custom API targets with base paths

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
│  │  │  │        Engine Command (wrapped)       │    │   │  │
│  │  │  │   Arbitrary command passed via --     │    │   │  │
│  │  │  └──────────────────────────────────────┘    │   │  │
│  │  └───────────────────────────────────────────────┘   │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### 3.2 Isolation Layers

The AWF sandbox provides three independent isolation layers:

1. **Network Egress Filter**: Domain-based allowlist/blocklist controlling all outbound network traffic from the agent container. Implemented via a forward proxy (Squid). AWF receives pre-resolved flat domain lists — it does not interpret ecosystem identifiers or other higher-level abstractions.
2. **API Proxy Sidecar**: Transparent credential injection for LLM provider APIs. The agent process never sees raw API keys; the sidecar intercepts outbound requests to known LLM endpoints and injects authentication headers.
3. **Filesystem Isolation**: The agent executes inside a Docker container with controlled mounts. The workspace is mounted read-write; auxiliary directories are mounted read-only.

### 3.3 Operational Model

The AWF binary operates as a wrapper command that:

1. Starts a Docker container with configured network policies
2. Launches an API proxy sidecar on the host
3. Configures DNS to route through the forward proxy
4. Executes the wrapped engine command inside the container
5. Logs all network activity for audit

---

## 4. AWF Command Interface

### 4.1 Command Structure

The AWF binary MUST be invoked with the following structure:

```
<command-prefix> <expandable-args> <safe-args> -- <shell-wrapped-engine-command>
```

Where:

- `<command-prefix>` is `sudo -E awf` by default, or a custom command replacing the AWF binary
- `<expandable-args>` are arguments containing shell variables (e.g., `${GITHUB_WORKSPACE}`) that MUST NOT be single-quoted so the runner's shell can expand them
- `<safe-args>` are arguments that are safely shell-escaped
- `--` separates AWF arguments from the engine command
- `<shell-wrapped-engine-command>` is the engine command wrapped in `/bin/bash -c '...'`

**Example invocation:**

```bash
sudo -E awf \
  --container-workdir "${GITHUB_WORKSPACE}" \
  --mount "${RUNNER_TEMP}/gh-aw:${RUNNER_TEMP}/gh-aw:ro" \
  --env-all \
  --allow-domains 'api.github.com,github.com,registry.npmjs.org,host.docker.internal' \
  --log-level info \
  --proxy-logs-dir /tmp/gh-aw/sandbox/firewall/logs \
  --audit-dir /tmp/gh-aw/sandbox/firewall/audit \
  --enable-host-access \
  --image-tag 0.25.18 \
  --skip-pull \
  --enable-api-proxy \
  -- /bin/bash -c 'copilot-agent --prompt /tmp/prompt.md'
```

### 4.2 Required Arguments

The following arguments MUST always be present in every AWF invocation:

| Argument | Value | Description |
|----------|-------|-------------|
| `--env-all` | (flag) | Pass all host environment variables to the container. |
| `--container-workdir` | Shell-expandable path (e.g., `"${GITHUB_WORKSPACE}"`) | Set the container working directory. |
| `--allow-domains` | Comma-separated domain list | Domains the container is allowed to reach (see [Section 5](#5-network-filtering)). |
| `--log-level` | One of `"debug"`, `"info"`, `"warn"`, `"error"` | AWF log verbosity. Defaults to `"info"`. |
| `--proxy-logs-dir` | Absolute path | Directory for proxy log files. |
| `--audit-dir` | Absolute path | Directory for audit files (policy-manifest, squid.conf). |
| `--enable-host-access` | (flag) | Enable access to host services (API proxy sidecar, other sidecars). |
| `--image-tag` | Version string (without `v` prefix, e.g., `"0.25.18"`) | Pin Docker image version to match installed AWF binary. |
| `--skip-pull` | (flag) | Skip pulling Docker images (assumes pre-downloaded during setup). |
| `--enable-api-proxy` | (flag) | Enable the API proxy sidecar for LLM credential isolation. |

### 4.3 Conditional Arguments

The following arguments are added based on the caller's configuration:

| Argument | Description |
|----------|-------------|
| `--tty` | Allocate a pseudo-TTY in the container. Required by some engines (e.g., Claude). |
| `--exclude-env <VAR>` | Exclude a specific environment variable from the container. One flag per variable. Requires AWF ≥ v0.25.3. |
| `--block-domains <domains>` | Comma-separated list of blocked domains. Blocked domains take precedence over allowed domains. |
| `--mount <spec>` | Additional container mount in `"source:destination:mode"` format. |
| `--memory-limit <limit>` | Container memory limit (e.g., `"4g"`, `"8g"`). |
| `--ssl-bump` | Enable SSL Bump for HTTPS content inspection (see [Section 8](#8-ssl-bump-https-inspection)). |
| `--allow-urls <urls>` | Comma-separated URL patterns for HTTPS path-level filtering. Requires `--ssl-bump`. |
| `--openai-api-target <host>` | Custom OpenAI-compatible API target hostname for the API proxy. |
| `--anthropic-api-target <host>` | Custom Anthropic-compatible API target hostname for the API proxy. |
| `--copilot-api-target <host>` | Custom Copilot API target hostname for the API proxy. |
| `--openai-api-base-path <path>` | URL path prefix for OpenAI-compatible endpoints (e.g., `/serving-endpoints`). |
| `--anthropic-api-base-path <path>` | URL path prefix for Anthropic-compatible endpoints. |
| `--difc-proxy-host <host:port>` | CLI proxy sidecar host address. Requires AWF ≥ v0.25.17. |
| `--difc-proxy-ca-cert <path>` | CA certificate path for CLI proxy TLS. Requires AWF ≥ v0.25.17. |
| `--allow-host-service-ports <ports>` | Comma-separated ports for host service access from the container. |

### 4.4 Version-Gated Features

Some AWF arguments require a minimum binary version:

| Feature | Minimum Version | Arguments |
|---------|----------------|-----------|
| Environment variable exclusion | v0.25.3 | `--exclude-env` |
| CLI proxy sidecar | v0.25.17 | `--difc-proxy-host`, `--difc-proxy-ca-cert` |

When the AWF binary version is older than the minimum required, the caller MUST omit the corresponding arguments. The caller SHOULD log a warning when features are skipped due to version constraints.

---

## 5. Network Filtering

> **Note**: AWF does not interpret ecosystem identifiers (e.g., `"defaults"`, `"github"`, `"python"`). These are compiler constructs that are resolved into flat domain lists before AWF is invoked. AWF only sees the resulting comma-separated domain strings.

### 5.1 Domain List Format

AWF receives domain lists as comma-separated strings via `--allow-domains` and `--block-domains`. Each entry in the list MUST be one of:

- **Literal domain**: A specific domain name (e.g., `api.github.com`). AWF MUST allow traffic to this domain and all its subdomains.
- **Wildcard pattern**: A domain prefixed with `*.` (e.g., `*.googleapis.com`) matching any subdomain under the specified domain.
- **Protocol-qualified domain**: A domain prefixed with `https://` or `http://` to restrict to a specific protocol (e.g., `https://secure.api.example.com`).
- **IP address**: A literal IP address (e.g., `172.30.0.1`) for direct IP-based access.
- **Wildcard `*`**: Permits all outbound network traffic.

### 5.2 Allow/Block Semantics

AWF MUST enforce the following domain filtering rules:

1. **Default-deny**: All outbound network traffic MUST be denied unless the destination domain matches an entry in the `--allow-domains` list.
2. **Block precedence**: Domains in the `--block-domains` list MUST take precedence over `--allow-domains`. If a domain appears in both lists, it MUST be blocked.
3. **Subdomain inclusion**: When a domain is allowed (e.g., `github.com`), all subdomains MUST also be allowed (e.g., `api.github.com`, `raw.githubusercontent.com` under `github.com`). The same rule applies to blocked domains.
4. **Wildcard matching**: The entry `*.example.com` MUST match any subdomain of `example.com` (e.g., `api.example.com`, `cdn.example.com`) but MUST NOT match `example.com` itself.
5. **Universal wildcard**: When `--allow-domains` contains `*`, all outbound traffic MUST be permitted (subject to `--block-domains` overrides).

### 5.3 Proxy Implementation

AWF implements network filtering via a forward proxy (Squid):

1. All outbound HTTP/HTTPS traffic from the container MUST be routed through the proxy.
2. The proxy MUST evaluate each request against the allow/block domain lists.
3. Denied requests MUST receive an HTTP 403 response from the proxy.
4. All proxy decisions (allow/deny) MUST be logged in the proxy access log.

---

## 6. Container Isolation

### 6.1 Docker Container

AWF MUST execute the wrapped engine command inside a Docker container. The container provides:

- Process isolation from the host
- Network namespace routing through the AWF proxy
- Controlled filesystem access via mounts

### 6.2 Mounts

#### 6.2.1 Mount Syntax

Each `--mount` argument MUST follow the format `source:destination:mode` where:

- `source` is the host path (MUST NOT be empty)
- `destination` is the container path (MUST NOT be empty)
- `mode` MUST be either `ro` (read-only) or `rw` (read-write)

AWF MUST reject mounts that do not have exactly three colon-separated parts or use an invalid mode.

#### 6.2.2 Standard Mounts

The following mounts are typically provided by the caller for every AWF invocation:

| Host Path | Container Path | Mode | Description |
|-----------|---------------|------|-------------|
| `${GITHUB_WORKSPACE}` | `${GITHUB_WORKSPACE}` | rw | Agent working directory (repository checkout). |
| `${RUNNER_TEMP}/gh-aw` | `${RUNNER_TEMP}/gh-aw` | ro | gh-aw configuration and prompt files. |
| `${RUNNER_TEMP}/gh-aw` | `/host${RUNNER_TEMP}/gh-aw` | ro | Host-path alias for cross-reference. |

Additional mounts MAY be provided for specific features (e.g., artifact upload staging directories).

### 6.3 Working Directory

The `--container-workdir` argument sets the working directory inside the container. This MUST be a shell-expandable path (typically `${GITHUB_WORKSPACE}`) so the runner's shell resolves it before AWF receives the value.

### 6.4 Memory Limits

When `--memory-limit` is provided, AWF MUST set the Docker container memory limit to the specified value (e.g., `4g` for 4 gigabytes). If not provided, the container runs with the Docker default memory limit.

### 6.5 Environment Variables

#### 6.5.1 Environment Passthrough

The `--env-all` flag instructs AWF to pass all host environment variables to the container. This is REQUIRED for the engine command to access GitHub Actions context variables, PATH, and other runtime configuration.

#### 6.5.2 Environment Exclusion

The `--exclude-env <VAR>` flag (one per variable, AWF ≥ v0.25.3) instructs AWF to exclude specific environment variables from the container. This prevents the agent from reading secret values via shell introspection commands (`env`, `printenv`).

The caller MUST provide `--exclude-env` for each environment variable whose value references a secret. Excluded variables SHOULD be sorted alphabetically for deterministic output in compiled artifacts.

The API proxy sidecar handles authentication transparently, so excluded credential variables are not needed inside the container.

---

## 7. API Proxy Sidecar

### 7.1 Purpose

The API proxy sidecar provides transparent credential injection for LLM provider APIs. When `--enable-api-proxy` is set, AWF MUST start a proxy process on the host that:

1. Intercepts outbound requests from the container to known LLM provider endpoints
2. Injects authentication headers (API keys, bearer tokens) into the intercepted requests
3. Forwards the authenticated requests to the actual LLM provider

The agent process inside the container MUST NOT have direct access to raw API keys or tokens.

### 7.2 API Targets

AWF supports configuring custom API endpoints for credential injection:

| Argument | Default Target | Description |
|----------|---------------|-------------|
| `--openai-api-target <host>` | `api.openai.com` | OpenAI-compatible API target. |
| `--anthropic-api-target <host>` | `api.anthropic.com` | Anthropic-compatible API target. |
| `--copilot-api-target <host>` | GitHub Copilot endpoints | Copilot API target (GHEC, GHES, custom). |

### 7.3 Base Path Support

When the LLM API endpoint uses a URL path prefix (e.g., Azure OpenAI at `/openai/deployments/{model}/...` or corporate routers with path-based routing), the caller MUST provide the corresponding `--*-api-base-path` argument:

| Argument | Description |
|----------|-------------|
| `--openai-api-base-path <path>` | URL path prefix for OpenAI-compatible endpoints. |
| `--anthropic-api-base-path <path>` | URL path prefix for Anthropic-compatible endpoints. |

### 7.4 Host Access

The `--enable-host-access` flag is REQUIRED for the API proxy sidecar to function. It allows the container to reach `host.docker.internal`, where the proxy listens.

---

## 8. SSL Bump (HTTPS Inspection)

### 8.1 Purpose

SSL Bump enables HTTPS content inspection, allowing URL path-level filtering instead of domain-only filtering. When `--ssl-bump` is passed to AWF:

1. AWF MUST generate a dynamic CA certificate and inject it into the container's trust store.
2. AWF MUST intercept and decrypt HTTPS traffic for content inspection.
3. If `--allow-urls` is provided, AWF MUST filter based on full URL paths (not just domains).

### 8.2 URL Filtering

When `--allow-urls <urls>` is provided (comma-separated URL patterns):

1. AWF MUST permit HTTPS requests matching the URL patterns.
2. URL patterns MAY contain wildcards (e.g., `https://github.com/githubnext/*`).
3. `--allow-urls` without `--ssl-bump` MUST be ignored (URL-level filtering requires HTTPS interception).

### 8.3 Certificate Pinning

Applications that perform certificate pinning MAY fail when SSL Bump is active because the intercepted certificate chain differs from the expected pinned certificate. AWF SHOULD provide diagnostic logging when TLS handshake failures occur. Certificate pinning is NOT a supported configuration when SSL Bump is active.

---

## 9. Logging and Audit

### 9.1 Log Levels

The `--log-level` argument controls AWF's log verbosity. It MUST be one of:

| Level | Description |
|-------|-------------|
| `debug` | Verbose diagnostic output including all proxy decisions. |
| `info` | Standard operational logging (default). |
| `warn` | Warnings and errors only. |
| `error` | Errors only. |

### 9.2 Proxy Logs

AWF MUST write proxy logs to the directory specified by `--proxy-logs-dir`. These logs contain the forward proxy (Squid) access log with all HTTP/HTTPS requests and their allow/deny status.

### 9.3 Audit Directory

AWF MUST write audit artifacts to the directory specified by `--audit-dir`. These artifacts MUST include:

- **Policy manifest**: The resolved domain allow/block lists and firewall configuration as applied.
- **Proxy configuration**: The generated Squid configuration (squid.conf).
- **Docker Compose configuration**: The redacted Docker Compose file used to start the container.

Callers SHOULD upload the audit directory as a GitHub Actions artifact for post-run inspection.

---

## 10. Security Considerations

### 10.1 Credential Isolation

The AWF API proxy sidecar MUST intercept outbound requests to known LLM provider endpoints and inject authentication headers transparently. The agent process MUST NOT have direct access to raw API keys or tokens.

Callers MUST use `--exclude-env` (AWF ≥ v0.25.3) to prevent the agent from reading secret values via shell introspection commands (`env`, `printenv`). For earlier AWF versions, callers SHOULD log a warning that environment variable exclusion is not available.

### 10.2 Network Egress

All outbound network traffic from the agent container MUST pass through the AWF forward proxy. The proxy MUST enforce the configured allow/block domain lists. Domains not in the allowed list MUST be denied by default (default-deny policy).

### 10.3 Filesystem Isolation

The agent MUST execute inside a Docker container with controlled mount points. The workspace MUST be mounted read-write. Auxiliary directories (configuration, prompts) SHOULD be mounted read-only.

### 10.4 Audit Logging

The AWF MUST produce structured audit logs in the configured audit directory. These logs MUST include:

- Policy manifest (resolved domain lists and firewall configuration)
- Proxy access log (all HTTP/HTTPS requests with allow/deny decisions)
- Redacted Docker Compose configuration

### 10.5 SSL Bump Considerations

When SSL Bump is enabled:

1. The AWF generates a dynamic CA certificate that MUST be injected into the container's trust store.
2. Applications that perform certificate pinning MAY fail. AWF SHOULD provide diagnostic logging when TLS handshake failures occur.
3. SSL Bump SHOULD only be enabled when URL path-level filtering is required. Domain-only filtering is sufficient for most use cases.
4. Workflows that require certificate pinning MUST NOT enable SSL Bump.

---

## 11. Compliance Testing

### 11.1 AWF Command Tests

- **T-CMD-001**: AWF accepts and applies `--allow-domains` with a comma-separated list of literal domains.
- **T-CMD-002**: AWF accepts and applies `--block-domains` with blocked domains taking precedence over allowed.
- **T-CMD-003**: `--env-all` passes host environment variables to the container.
- **T-CMD-004**: `--exclude-env` prevents specified variables from being visible inside the container (AWF ≥ v0.25.3).
- **T-CMD-005**: `--container-workdir` sets the working directory inside the container.
- **T-CMD-006**: `--mount` with valid `source:dest:mode` format creates the specified mount.
- **T-CMD-007**: `--mount` with invalid format (wrong parts count, invalid mode) is rejected.
- **T-CMD-008**: `--memory-limit` sets the container memory limit.
- **T-CMD-009**: `--image-tag` pins the Docker image version.
- **T-CMD-010**: `--skip-pull` prevents Docker image pull.

### 11.2 Network Filtering Tests

- **T-NET-001**: Literal domain in `--allow-domains` permits traffic to that domain.
- **T-NET-002**: Subdomain of an allowed domain is also permitted.
- **T-NET-003**: Domain not in `--allow-domains` is denied (default-deny).
- **T-NET-004**: Wildcard pattern `*.example.com` matches subdomains but not `example.com` itself.
- **T-NET-005**: Blocked domain in `--block-domains` overrides the same domain in `--allow-domains`.
- **T-NET-006**: Protocol-qualified domain restricts to the specified protocol.
- **T-NET-007**: Universal wildcard `*` in `--allow-domains` permits all traffic.
- **T-NET-008**: IP address in `--allow-domains` permits traffic to that IP.

### 11.3 API Proxy Tests

- **T-PRX-001**: `--enable-api-proxy` starts the API proxy sidecar.
- **T-PRX-002**: `--openai-api-target` routes OpenAI requests to the specified host.
- **T-PRX-003**: `--anthropic-api-target` routes Anthropic requests to the specified host.
- **T-PRX-004**: `--copilot-api-target` routes Copilot requests to the specified host.
- **T-PRX-005**: `--openai-api-base-path` applies the path prefix to OpenAI requests.
- **T-PRX-006**: `--anthropic-api-base-path` applies the path prefix to Anthropic requests.
- **T-PRX-007**: `--enable-host-access` allows container to reach `host.docker.internal`.

### 11.4 SSL Bump Tests

- **T-SSL-001**: `--ssl-bump` enables HTTPS interception with dynamic CA certificate.
- **T-SSL-002**: `--allow-urls` with `--ssl-bump` filters by full URL path.
- **T-SSL-003**: `--allow-urls` without `--ssl-bump` is ignored.
- **T-SSL-004**: SSL Bump provides diagnostic logging for TLS handshake failures.

### 11.5 Logging and Audit Tests

- **T-LOG-001**: `--log-level debug` produces verbose output.
- **T-LOG-002**: `--log-level info` produces standard output.
- **T-LOG-003**: `--log-level warn` produces warnings and errors only.
- **T-LOG-004**: `--log-level error` produces errors only.
- **T-LOG-005**: Invalid log level is rejected.
- **T-LOG-006**: `--proxy-logs-dir` writes proxy access logs to the specified directory.
- **T-LOG-007**: `--audit-dir` writes policy manifest, squid.conf, and Docker Compose config.

### 11.6 Compliance Checklist

| Requirement | Test ID | Level | Status |
|-------------|---------|-------|--------|
| Domain-based allow filtering | T-NET-001–003 | 1 | Required |
| Wildcard domain matching | T-NET-004 | 1 | Required |
| Block domain precedence | T-NET-005 | 1 | Required |
| Default-deny policy | T-NET-003 | 1 | Required |
| Environment passthrough | T-CMD-003 | 1 | Required |
| Container workdir | T-CMD-005 | 1 | Required |
| Mount format validation | T-CMD-006–007 | 1 | Required |
| API proxy sidecar | T-PRX-001–007 | 1 | Required |
| AWF command arguments | T-CMD-001–010 | 1 | Required |
| Log level handling | T-LOG-001–005 | 1 | Required |
| Audit output | T-LOG-006–007 | 1 | Required |
| Environment exclusion | T-CMD-004 | 2 | Required |
| Memory limits | T-CMD-008 | 2 | Required |
| SSL Bump | T-SSL-001–004 | 2 | Required |
| Protocol-qualified domains | T-NET-006 | 2 | Required |
| Universal wildcard | T-NET-007 | 3 | Optional |
| Custom API targets | T-PRX-002–006 | 3 | Optional |

---

## References

### Normative References

- **[RFC 2119]** Bradner, S., "Key words for use in RFCs to Indicate Requirement Levels", BCP 14, RFC 2119, March 1997.
- **[Security Architecture Specification]** GitHub Agentic Workflows Team, "Security Architecture Specification", Candidate Recommendation.

### Informative References

- **[AWF Repository]** GitHub gh-aw-firewall project, `ghcr.io/github/gh-aw-firewall`.
- **[Docker Security]** Docker, Inc., "Docker Security", https://docs.docker.com/engine/security/.
- **[Squid Proxy]** Squid Project, "Squid: Optimising Web Delivery", https://www.squid-cache.org/.

---

## Change Log

### Version 1.0.0 (Draft)

- Initial specification release
- AWF CLI argument interface (required and conditional arguments)
- Network filtering behavior (flat domain lists, allow/block semantics, wildcard matching)
- Container isolation (mounts, memory limits, environment variables)
- API proxy sidecar (credential isolation, custom API targets, base path support)
- SSL Bump HTTPS content inspection
- Logging and audit requirements
- Compliance testing framework (37 test IDs across 6 categories)

---

*Copyright © 2026 GitHub, Inc. All rights reserved.*
