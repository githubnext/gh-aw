---
title: MCP Gateway Specification
description: Formal specification for the Model Context Protocol (MCP) Gateway implementation following W3C conventions
sidebar:
  order: 1350
---

# MCP Gateway Specification

**Version**: 1.1.0  
**Status**: Draft Specification  
**Latest Version**: [mcp-gateway](/gh-aw/reference/mcp-gateway/)  
**JSON Schema**: [mcp-gateway-config.schema.json](/gh-aw/schemas/mcp-gateway-config.schema.json)  
**Editor**: GitHub Agentic Workflows Team

---

## Abstract

This specification defines the Model Context Protocol (MCP) Gateway, a transparent proxy service that enables unified HTTP access to multiple MCP servers. The gateway supports containerized MCP servers and HTTP-based MCP servers. The gateway provides protocol translation, server isolation, authentication, and health monitoring capabilities.

## Status of This Document

This section describes the status of this document at the time of publication. This is a draft specification and may be updated, replaced, or made obsolete by other documents at any time.

This document is governed by the GitHub Agentic Workflows project specifications process.

## Table of Contents

1. [Introduction](#1-introduction)
2. [Conformance](#2-conformance)
3. [Architecture](#3-architecture)
4. [Configuration](#4-configuration)
5. [Protocol Behavior](#5-protocol-behavior)
6. [Server Isolation](#6-server-isolation)
7. [Authentication](#7-authentication)
8. [Health Monitoring](#8-health-monitoring)
9. [Error Handling](#9-error-handling)
10. [Compliance Testing](#10-compliance-testing)

---

## 1. Introduction

### 1.1 Purpose

The MCP Gateway serves as a protocol translation layer between MCP clients expecting HTTP-based communication and MCP servers running in containers or accessible via HTTP. It enables:

- **Protocol Translation**: Converting between containerized stdio servers and HTTP transports
- **Unified Access**: Single HTTP endpoint for multiple MCP servers
- **Server Isolation**: Enforcing boundaries between server instances through containerization
- **Authentication**: Token-based access control
- **Health Monitoring**: Service availability endpoints

The gateway requires that stdio-based MCP servers MUST be containerized. Direct command execution (stdio+command without containerization) is NOT supported because it cannot provide the necessary isolation and portability guarantees.

### 1.2 Scope

This specification covers:

- Gateway configuration format and semantics
- Protocol translation behavior
- Server lifecycle management
- Authentication mechanisms
- Health monitoring interfaces
- Error handling requirements

This specification does NOT cover:

- Model Context Protocol (MCP) core protocol semantics
- Individual MCP server implementations
- Client-side MCP implementations
- User interfaces or interactive features (e.g., elicitation)

### 1.3 Design Goals

The gateway MUST be designed for:

- **Headless Operation**: No user interaction required during runtime
- **Fail-Fast Behavior**: Immediate failure with diagnostic information
- **Forward Compatibility**: Graceful rejection of unknown configuration features
- **Security**: Isolation between servers and secure credential handling

---

## 2. Conformance

### 2.1 Conformance Classes

A **conforming MCP Gateway implementation** is one that satisfies all MUST, REQUIRED, and SHALL requirements in this specification.

A **partially conforming MCP Gateway implementation** is one that satisfies all MUST requirements but MAY lack support for optional features marked with SHOULD or MAY.

### 2.2 Requirements Notation

The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT", "SHOULD", "SHOULD NOT", "RECOMMENDED", "NOT RECOMMENDED", "MAY", and "OPTIONAL" in this document are to be interpreted as described in [RFC 2119](https://www.ietf.org/rfc/rfc2119.txt).

### 2.3 Compliance Levels

Implementations MUST support:

- **Level 1 (Required)**: Basic proxy functionality, stdio transport, configuration parsing
- **Level 2 (Standard)**: HTTP transport, authentication, health endpoints
- **Level 3 (Complete)**: All optional features including variable expressions, timeout configuration

---

## 3. Architecture

### 3.1 Gateway Model

```
┌─────────────────────────────────────────────────────────┐
│                      MCP Client                         │
│                    (HTTP Transport)                     │
└──────────────────────┬──────────────────────────────────┘
                       │ HTTP/JSON-RPC
                       ▼
┌─────────────────────────────────────────────────────────┐
│                    MCP Gateway                          │
│  ┌───────────────────────────────────────────────────┐  │
│  │  Authentication & Authorization Layer             │  │
│  └───────────────────────────────────────────────────┘  │
│  ┌───────────────────────────────────────────────────┐  │
│  │  Protocol Translation Layer                       │  │
│  └───────────────────────────────────────────────────┘  │
│  ┌───────────────────────────────────────────────────┐  │
│  │  Server Isolation & Lifecycle Management          │  │
│  └───────────────────────────────────────────────────┘  │
└──────┬──────────────┬──────────────┬───────────────────┘
       │              │              │
       │ stdio        │ HTTP         │ stdio
       ▼              ▼              ▼
  ┌─────────┐   ┌─────────┐   ┌─────────┐
  │ MCP     │   │ MCP     │   │ MCP     │
  │ Server  │   │ Server  │   │ Server  │
  │ 1       │   │ 2       │   │ N       │
  └─────────┘   └─────────┘   └─────────┘
```

### 3.2 Transport Support

The gateway MUST support the following transport mechanisms:

- **stdio (containerized)**: MCP servers running in containers with standard input/output based communication
- **HTTP**: Direct HTTP-based MCP servers

The gateway MUST translate all upstream transports to HTTP for client communication.

#### 3.2.1 Containerization Requirement

Stdio-based MCP servers MUST be containerized. The gateway SHALL NOT support direct command execution without containerization (stdio+command) because:

1. Containerization provides necessary process isolation and security boundaries
2. Containers enable reproducible environments across different deployment contexts
3. Container images provide versioning and dependency management
4. Containerization ensures portability and consistent behavior

Direct command execution of stdio servers (e.g., `command: "node server.js"` without a container) is explicitly NOT SUPPORTED by this specification.

### 3.3 Operational Model

The gateway operates in a headless mode:

1. Configuration is provided via **stdin** (JSON format)
2. Secrets are provided via **environment variables**
3. Startup output is written to **stdout** (rewritten configuration)
4. Error messages are written to **stdout** as error payloads
5. HTTP server accepts client requests on configured port

---

## 4. Configuration

### 4.1 Configuration Format

The gateway MUST accept configuration via stdin in JSON format conforming to the MCP configuration file schema.

**JSON Schema**: [mcp-gateway-config.schema.json](/gh-aw/schemas/mcp-gateway-config.schema.json)

#### 4.1.1 Configuration Structure

```json
{
  "mcpServers": {
    "server-name": {
      "container": "string",
      "entrypoint": "string",
      "entrypointArgs": ["string"],
      "mounts": ["source:dest:mode"],
      "env": {
        "VAR_NAME": "value"
      },
      "type": "stdio" | "http",
      "url": "string"
    }
  },
  "gateway": {
    "port": 8080,
    "apiKey": "string",
    "domain": "string",
    "startupTimeout": 30,
    "toolTimeout": 60
  }
}
```

#### 4.1.2 Server Configuration Fields

Each server configuration MUST support:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `container` | string | Conditional* | Container image for the MCP server (required for stdio servers) |
| `entrypoint` | string | No | Optional entrypoint override for container (equivalent to `docker run --entrypoint`) |
| `entrypointArgs` | array[string] | No | Arguments passed to container entrypoint (container only) |
| `mounts` | array[string] | No | Volume mounts for container (format: "source:dest:mode" where mode is "ro" or "rw") |
| `env` | object | No | Environment variables for the server process |
| `type` | string | No | Transport type: "stdio" or "http" (default: "stdio") |
| `url` | string | Conditional** | HTTP endpoint URL for HTTP servers |

*Required for stdio servers (containerized execution)  
**Required for HTTP servers

**Note**: The `command` field is NOT supported. Stdio servers MUST use the `container` field to specify a containerized MCP server. Direct command execution is not supported by this specification.

#### 4.1.3 Gateway Configuration Fields

The `gateway` section is required and configures gateway-specific behavior:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `port` | integer | Yes | HTTP server port |
| `domain` | string | Yes | Gateway domain (localhost or host.docker.internal) |
| `apiKey` | string | Yes | API key for authentication |
| `startupTimeout` | integer | No | Server startup timeout in seconds (default: 30) |
| `toolTimeout` | integer | No | Tool invocation timeout in seconds (default: 60) |

### 4.2 Variable Expression Rendering

#### 4.2.1 Syntax

Configuration values MAY contain variable expressions using the syntax:

```
"${VARIABLE_NAME}"
```

#### 4.2.2 Resolution Behavior

The gateway MUST:

1. Detect variable expressions in configuration values
2. Replace expressions with values from process environment variables
3. FAIL IMMEDIATELY if a referenced variable is not defined
4. Log the undefined variable name to stdout as an error payload
5. Exit with non-zero status code

#### 4.2.3 Example

Configuration:

```json
{
  "mcpServers": {
    "github": {
      "container": "ghcr.io/github/github-mcp-server:latest",
      "env": {
        "GITHUB_TOKEN": "${GITHUB_PERSONAL_ACCESS_TOKEN}"
      }
    }
  }
}
```

If `GITHUB_PERSONAL_ACCESS_TOKEN` is not set in the environment:

```
Error: undefined environment variable referenced: GITHUB_PERSONAL_ACCESS_TOKEN
Required by: mcpServers.github.env.GITHUB_TOKEN
```

### 4.3 Configuration Validation

#### 4.3.1 Unknown Features

The gateway MUST reject configurations containing unrecognized fields at the top level with an error message indicating:

- The unrecognized field name
- The location in the configuration
- A suggestion to check the specification version

#### 4.3.2 Schema Validation

The gateway MUST validate:

- Required fields are present
- Field types match expected types
- Value constraints are satisfied (e.g., port ranges)
- Mutually exclusive fields are not both present

#### 4.3.3 Fail-Fast Requirements

If configuration is invalid, the gateway MUST:

1. Write a detailed error message to stdout as an error payload including:
   - The specific validation error
   - The location in the configuration (JSON path)
   - Suggested corrective action
2. Exit with status code 1
3. NOT start the HTTP server
4. NOT initialize any MCP servers

---

## 5. Protocol Behavior

For complete details on the Model Context Protocol, see the [Model Context Protocol Specification](https://spec.modelcontextprotocol.io/).

### 5.1 HTTP Server Interface

#### 5.1.1 Endpoint Structure

The gateway MUST expose the following HTTP endpoints:

```
POST /mcp/{server-name}
GET  /health
POST /close
```

#### 5.1.2 RPC Endpoint Behavior

**Request Format**:

```http
POST /mcp/{server-name} HTTP/1.1
Content-Type: application/json
Authorization: {apiKey}

{
  "jsonrpc": "2.0",
  "method": "string",
  "params": {},
  "id": "string|number"
}
```

**Response Format**:

```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "jsonrpc": "2.0",
  "result": {},
  "id": "string|number"
}
```

**Error Response**:

```http
HTTP/1.1 500 Internal Server Error
Content-Type: application/json

{
  "jsonrpc": "2.0",
  "error": {
    "code": -32603,
    "message": "Internal error",
    "data": {}
  },
  "id": "string|number"
}
```

#### 5.1.3 Close Endpoint Behavior

The gateway MUST provide a `/close` endpoint for graceful shutdown and resource cleanup.

**Request Format**:

```http
POST /close HTTP/1.1
Authorization: {apiKey}
```

**Success Response**:

```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "status": "closed",
  "message": "Gateway shutdown initiated",
  "serversTerminated": 3
}
```

**Already Closed Response**:

```http
HTTP/1.1 410 Gone
Content-Type: application/json

{
  "error": "Gateway has already been closed"
}
```

**Behavior Requirements**:

The gateway MUST perform the following actions when the `/close` endpoint is called:

1. **Stop Accepting New Requests**: Immediately reject any new RPC requests to `/mcp/{server-name}` endpoints with HTTP 503 Service Unavailable
2. **Complete In-Flight Requests**: Allow currently processing requests to complete (with a reasonable timeout, e.g., 30 seconds)
3. **Terminate All Containers**: Stop all running MCP server containers:
   - Send SIGTERM to each container process
   - Wait up to 10 seconds for graceful shutdown
   - Send SIGKILL if container does not stop within timeout
   - Log termination status for each server
4. **Release Resources**:
   - Close all file descriptors and network sockets
   - Clean up temporary files and logs
   - Release volume mounts
   - Free allocated memory
5. **Return Response**: Send success response before process exits
6. **Exit Process**: Exit the gateway process with status code 0

**Idempotency**:

The `/close` endpoint MUST be idempotent:
- First call: Initiates shutdown and returns HTTP 200
- Subsequent calls: Returns HTTP 410 Gone indicating gateway is already closed

**Authentication**:

The `/close` endpoint MUST require authentication when `gateway.apiKey` is configured. Requests without valid authentication MUST be rejected with HTTP 401 Unauthorized.

#### 5.1.4 Request Routing

The gateway MUST:

1. Extract server name from URL path
2. Validate server exists in configuration
3. Route request to appropriate backend server
4. Translate protocols if necessary (stdio ↔ HTTP)
5. Return response to client

### 5.2 Protocol Translation

#### 5.2.1 Stdio (Containerized) to HTTP

For containerized stdio-based servers, the gateway MUST:

1. Start the container on first request (lazy initialization)
2. Write JSON-RPC request to container's stdin
3. Read JSON-RPC response from container's stdout
4. Return HTTP response to client
5. Maintain container for subsequent requests
6. Buffer partial responses until complete JSON is received

The gateway SHALL NOT support non-containerized command execution. All stdio servers MUST be containerized.

#### 5.2.2 HTTP to HTTP

For HTTP-based servers, the gateway MUST:

1. Forward the JSON-RPC request to the server's URL
2. Apply any configured headers or authentication
3. Return the server's response to the client
4. Handle HTTP-level errors appropriately

#### 5.2.3 Tool Signature Preservation

The gateway SHOULD NOT modify:

- Tool names
- Tool parameters
- Tool return values
- Method signatures

This ensures transparent proxying without name mangling or schema transformation.

### 5.3 Timeout Handling

#### 5.3.1 Startup Timeout

The gateway SHOULD enforce `startupTimeout` for server initialization:

1. Start timer when server container is launched
2. Wait for server ready signal (stdio) or successful health check (HTTP)
3. If timeout expires, kill server container and return error
4. Log timeout error with server name and elapsed time

#### 5.3.2 Tool Timeout

The gateway SHOULD enforce `toolTimeout` for individual tool invocations:

1. Start timer when RPC request is sent to server
2. Wait for complete response
3. If timeout expires, return timeout error to client
4. Log timeout with server name, method, and elapsed time

### 5.4 Stdout Configuration Output

After successful initialization, the gateway MUST:

1. Write a complete MCP server configuration to stdout
2. Include gateway connection details:
   ```json
   {
     "mcpServers": {
       "server-name": {
         "type": "http",
         "url": "http://{domain}:{port}/mcp/server-name",
         "headers": {
           "Authorization": "{apiKey}"
         }
       }
     }
   }
   ```
3. Write configuration as a single JSON document
4. Flush stdout buffer
5. Continue serving requests

This allows clients to dynamically discover gateway endpoints.

---

## 6. Server Isolation

### 6.1 Container Isolation

For stdio servers, the gateway MUST:

1. Launch each server in a separate container
2. Maintain isolated stdin/stdout/stderr streams
3. Prevent cross-container communication
4. Terminate containers on gateway shutdown (via `/close` endpoint or process termination)

All stdio-based MCP servers MUST be containerized to ensure:

- **Process Isolation**: Each container provides a separate process namespace
- **Resource Isolation**: Containers enforce CPU, memory, and filesystem boundaries
- **Network Isolation**: Containers provide isolated network namespaces
- **Security Boundaries**: Container runtimes enforce security policies and capabilities

The gateway SHALL NOT support non-containerized process execution for stdio servers.

### 6.2 Resource Isolation

The gateway MUST ensure:

- Each server has isolated environment variables within its container
- File descriptors are not shared between containers
- Network sockets are not shared (for HTTP servers)
- Container failures do not affect other containers

### 6.3 Security Boundaries

The gateway MUST NOT:

- Allow servers to access each other's configuration
- Share authentication credentials between servers
- Expose server implementation details to clients
- Allow cross-server tool invocations

---

## 7. Authentication

### 7.1 API Key Authentication

When `gateway.apiKey` is configured, the gateway MUST:

1. Require `Authorization: {apiKey}` header on all RPC requests to `/mcp/{server-name}` and `/close` endpoints
2. Reject requests with missing or invalid tokens (HTTP 401)
3. Reject requests with malformed Authorization headers (HTTP 400)
4. NOT log API keys in plaintext

### 7.2 Optimal Temporary API Key

The gateway SHOULD support temporary API keys:

1. Generate a random API key on startup if not provided
2. Include key in stdout configuration output

### 7.3 Authentication Exemptions

The following endpoints MUST NOT require authentication:

- `/health`

---

## 8. Health Monitoring

### 8.1 Health Endpoints

#### 8.1.1 General Health (`/health`)

```http
GET /health HTTP/1.1
```

Response:

```json
{
  "status": "healthy" | "unhealthy",
  "servers": {
    "server-name": {
      "status": "running" | "stopped" | "error",
      "uptime": 12345
    }
  }
}
```

### 8.2 Health Check Behavior

The gateway SHOULD:

1. Periodically check server health (every 30 seconds)
2. Restart failed containerized stdio servers automatically
3. Mark HTTP servers unhealthy if unreachable
4. Include health status in `/health` response
5. Update readiness based on critical server status

---

## 9. Error Handling

### 9.1 Startup Failures

If any configured server fails to start, the gateway MUST:

1. Write detailed error to stdout as an error payload including:
   - Server name
   - Container image or URL attempted
   - Error message from server container
   - Environment variable status
   - Stdout/stderr from failed container
2. Exit with status code 1
3. NOT start the HTTP server

### 9.2 Runtime Errors

For runtime errors, the gateway MUST:

1. Log errors to stdout as error payloads with:
   - Timestamp
   - Server name
   - Request ID
   - Error details
2. Return JSON-RPC error response to client
3. Continue serving other requests
4. Attempt to restart failed containerized stdio servers

### 9.3 Error Response Format

JSON-RPC errors MUST follow this structure:

```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32000,
    "message": "Server error",
    "data": {
      "server": "server-name",
      "detail": "Specific error information"
    }
  },
  "id": "request-id"
}
```

Error codes:

- `-32700`: Parse error
- `-32600`: Invalid request
- `-32601`: Method not found
- `-32603`: Internal error
- `-32000` to `-32099`: Server errors

### 9.4 Graceful Degradation

The gateway SHOULD:

1. Continue serving healthy servers when others fail
2. Return specific errors for unavailable servers
3. Attempt automatic recovery for transient failures
4. Provide clear client feedback about server status

---

## 10. Compliance Testing

### 10.1 Test Suite Requirements

A conforming implementation MUST pass the following test categories:

#### 10.1.1 Configuration Tests

- **T-CFG-001**: Valid stdio server configuration
- **T-CFG-002**: Valid HTTP server configuration
- **T-CFG-003**: Variable expression resolution
- **T-CFG-004**: Undefined variable error detection
- **T-CFG-005**: Unknown field rejection
- **T-CFG-006**: Missing required field detection
- **T-CFG-007**: Invalid type detection
- **T-CFG-008**: Port range validation

#### 10.1.2 Protocol Translation Tests

- **T-PTL-001**: Stdio request/response cycle
- **T-PTL-002**: HTTP passthrough
- **T-PTL-003**: Tool signature preservation
- **T-PTL-004**: Concurrent request handling
- **T-PTL-005**: Large payload handling
- **T-PTL-006**: Partial response buffering

#### 10.1.3 Isolation Tests

- **T-ISO-001**: Container isolation verification
- **T-ISO-002**: Environment isolation verification
- **T-ISO-003**: Credential isolation verification
- **T-ISO-004**: Cross-container communication prevention
- **T-ISO-005**: Container failure isolation

#### 10.1.4 Authentication Tests

- **T-AUTH-001**: Valid token acceptance
- **T-AUTH-002**: Invalid token rejection
- **T-AUTH-003**: Missing token rejection
- **T-AUTH-004**: Health endpoint exemption
- **T-AUTH-005**: Token rotation support

#### 10.1.5 Timeout Tests

- **T-TMO-001**: Startup timeout enforcement
- **T-TMO-002**: Tool timeout enforcement
- **T-TMO-003**: Timeout error messaging
- **T-TMO-004**: Partial response timeout
- **T-TMO-005**: Concurrent timeout handling

#### 10.1.6 Health Monitoring Tests

- **T-HLT-001**: Health endpoint availability
- **T-HLT-002**: Liveness probe accuracy
- **T-HLT-003**: Readiness probe accuracy
- **T-HLT-004**: Server status reporting
- **T-HLT-005**: Automatic restart behavior

#### 10.1.7 Error Handling Tests

- **T-ERR-001**: Startup failure reporting
- **T-ERR-002**: Runtime error handling
- **T-ERR-003**: Invalid request handling
- **T-ERR-004**: Server crash recovery
- **T-ERR-005**: Error message quality

#### 10.1.8 Gateway Lifecycle Tests

- **T-LIFE-001**: Close endpoint authentication
- **T-LIFE-002**: Close endpoint success response
- **T-LIFE-003**: Close endpoint idempotency (returns 410 on subsequent calls)
- **T-LIFE-004**: Container termination on close
- **T-LIFE-005**: Resource cleanup on close
- **T-LIFE-006**: In-flight request handling during shutdown
- **T-LIFE-007**: New requests rejected after close initiated

### 10.2 Compliance Checklist

| Requirement | Test ID | Level | Status |
|-------------|---------|-------|--------|
| Configuration parsing | T-CFG-* | 1 | Required |
| Variable expressions | T-CFG-003, T-CFG-004 | 3 | Optional |
| Stdio transport | T-PTL-001 | 1 | Required |
| HTTP transport | T-PTL-002 | 2 | Standard |
| Authentication | T-AUTH-* | 2 | Standard |
| Timeout handling | T-TMO-* | 3 | Optional |
| Health monitoring | T-HLT-* | 2 | Standard |
| Server isolation | T-ISO-* | 1 | Required |
| Error handling | T-ERR-* | 1 | Required |
| Gateway lifecycle | T-LIFE-* | 2 | Standard |
| Error handling | T-ERR-* | 1 | Required |

### 10.3 Test Execution

Implementations SHOULD provide:

1. Automated test runner
2. Test result reporting in standard format (e.g., TAP, JUnit)
3. Test fixtures for common scenarios
4. Performance benchmarks
5. Conformance report generation

---

## Appendices

### Appendix A: Example Configurations

#### A.1 Basic Containerized Stdio Server

```json
{
  "mcpServers": {
    "example": {
      "container": "ghcr.io/example/mcp-server:latest",
      "entrypointArgs": ["--verbose"],
      "env": {
        "API_KEY": "${MY_API_KEY}"
      }
    }
  },
  "gateway": {
    "port": 8080,
    "domain": "localhost",
    "apiKey": "gateway-secret-token"
  }
}
```

#### A.2 Server with Volume Mounts and Custom Entrypoint

```json
{
  "mcpServers": {
    "data-server": {
      "container": "ghcr.io/example/data-mcp:latest",
      "entrypoint": "/custom/entrypoint.sh",
      "entrypointArgs": ["--config", "/app/config.json"],
      "mounts": [
        "/host/data:/container/data:ro",
        "/host/config:/container/config:rw"
      ],
      "type": "stdio"
    }
  },
  "gateway": {
    "port": 8080,
    "domain": "localhost",
    "apiKey": "gateway-secret-token"
  }
}
```

#### A.3 Mixed Transport Configuration

```json
{
  "mcpServers": {
    "local-server": {
      "container": "ghcr.io/example/python-mcp:latest",
      "entrypointArgs": ["--config", "/app/config.json"],
      "type": "stdio"
    },
    "remote-server": {
      "type": "http",
      "url": "https://api.example.com/mcp"
    }
  },
  "gateway": {
    "port": 8080,
    "domain": "localhost",
    "startupTimeout": 60,
    "toolTimeout": 120
  }
}
```

#### A.4 GitHub MCP Server (Containerized)

```json
{
  "mcpServers": {
    "github": {
      "container": "ghcr.io/github/github-mcp-server:latest",
      "env": {
        "GITHUB_PERSONAL_ACCESS_TOKEN": "${GITHUB_TOKEN}"
      }
    }
  },
  "gateway": {
    "port": 8080,
    "domain": "localhost"
  }
}
```

### Appendix B: Gateway Lifecycle Examples

#### B.1 Closing the Gateway

**Request**:

```http
POST /close HTTP/1.1
Host: localhost:8080
Authorization: gateway-secret-token
```

**Success Response**:

```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "status": "closed",
  "message": "Gateway shutdown initiated",
  "serversTerminated": 3
}
```

**Example Shutdown Sequence**:

1. Client calls `POST /close`
2. Gateway stops accepting new RPC requests
3. Gateway waits for in-flight requests to complete (max 30s)
4. Gateway terminates containers:
   - `github` container: SIGTERM sent, stopped after 2s
   - `slack` container: SIGTERM sent, stopped after 1s
   - `data-server` container: SIGTERM sent, SIGKILL after 10s timeout
5. Gateway cleans up resources
6. Gateway returns success response
7. Gateway process exits with code 0

**Idempotent Behavior**:

```http
POST /close HTTP/1.1
Host: localhost:8080
Authorization: gateway-secret-token
```

```http
HTTP/1.1 410 Gone
Content-Type: application/json

{
  "error": "Gateway has already been closed"
}
```

### Appendix C: Error Code Reference

| Code | Name | Description |
|------|------|-------------|
| -32700 | Parse error | Invalid JSON received |
| -32600 | Invalid request | Invalid JSON-RPC request |
| -32601 | Method not found | Method does not exist |
| -32602 | Invalid params | Invalid method parameters |
| -32603 | Internal error | Internal JSON-RPC error |
| -32000 | Server error | Generic server error |
| -32001 | Server unavailable | Server not responding |
| -32002 | Server timeout | Server response timeout |
| -32003 | Authentication failed | Invalid or missing credentials |

### Appendix D: Security Considerations

#### D.1 Credential Handling

- API keys MUST NOT be logged
- Environment variables MUST be isolated per server
- Secrets SHOULD be cleared from memory after use

#### D.2 Network Security

- Gateway SHOULD support TLS/HTTPS
- Server URLs SHOULD be validated
- Cross-origin requests SHOULD be restricted
- Rate limiting SHOULD be implemented

#### D.3 Container Security

- Server containers SHOULD run with minimal privileges
- Resource limits SHOULD be enforced (CPU, memory, file descriptors)
- Temporary files SHOULD be cleaned up
- Container monitoring SHOULD detect anomalies
- Container images SHOULD be signed and verified
- Containers SHOULD use read-only root filesystems where possible

#### D.4 Shutdown Security

- The `/close` endpoint MUST require authentication to prevent unauthorized shutdown
- Gateway SHOULD log all shutdown attempts (both successful and failed) for audit purposes
- In-flight requests SHOULD have a reasonable timeout to prevent denial-of-service during shutdown
- Container termination SHOULD use SIGTERM first to allow graceful cleanup before SIGKILL

---

## References

### Normative References

- **[RFC 2119]** Key words for use in RFCs to Indicate Requirement Levels
- **[JSON-RPC 2.0]** JSON-RPC 2.0 Specification
- **[MCP]** Model Context Protocol Specification

### Informative References

- **[MCP-Config]** MCP Configuration Format
- **[HTTP/1.1]** Hypertext Transfer Protocol -- HTTP/1.1

---

## Change Log

### Version 1.1.0 (Draft)

- Added `/close` endpoint for graceful gateway shutdown
- Added gateway lifecycle compliance tests (T-LIFE-*)
- Added resource cleanup requirements
- Added shutdown security considerations
- Added gateway lifecycle examples in Appendix B

### Version 1.0.0 (Draft)

- Initial specification release
- Configuration format definition
- Protocol behavior specification
- Compliance test framework

---

*Copyright © 2026 GitHub, Inc. All rights reserved.*
