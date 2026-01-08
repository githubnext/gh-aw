---
title: MCP Gateway Specification
description: Formal specification for the Model Context Protocol (MCP) Gateway (gh-aw-mcpg) implementation following W3C conventions
sidebar:
  order: 1350
---

# MCP Gateway Specification

**Version**: 2.0.0  
**Status**: Draft Specification  
**Latest Version**: [mcp-gateway](/gh-aw/reference/mcp-gateway/)  
**Editor**: GitHub Agentic Workflows Team

---

## Abstract

This specification defines the Model Context Protocol (MCP) Gateway (`gh-aw-mcpg`), a per-server proxy that provides Streamable HTTP transport on the frontend while connecting to individual MCP servers using their native transport (stdio or HTTP) on the backend. Each MCP server has its own dedicated gateway instance, ensuring complete isolation and independent lifecycle management.

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

The MCP Gateway (`gh-aw-mcpg`) serves as a per-server protocol translation layer that provides Streamable HTTP transport to MCP clients while managing individual MCP server connections. The architecture follows a "one gateway per MCP server" model, where each gateway instance:

- **Provides Streamable HTTP Frontend**: Accepts MCP client connections using Streamable HTTP transport
- **Manages Backend Transport**: Connects to a single MCP server using its native transport (stdio or HTTP)
- **Ensures Isolation**: Complete separation between different MCP server instances
- **Handles Authentication**: Token-based access control per gateway instance
- **Monitors Health**: Individual health endpoints per gateway

### 1.2 Scope

This specification covers:

- Per-server gateway architecture and deployment model
- Streamable HTTP frontend protocol behavior
- Backend transport support (stdio and HTTP)
- Gateway lifecycle management
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

- **Per-Server Isolation**: Each MCP server has its own gateway instance
- **Streamable HTTP Support**: Frontend uses Streamable HTTP for all client communication
- **Transport Flexibility**: Backend supports both stdio and HTTP transports
- **Headless Operation**: No user interaction required during runtime
- **Fail-Fast Behavior**: Immediate failure with diagnostic information
- **Forward Compatibility**: Graceful rejection of unknown configuration features
- **Security**: Secure credential handling and network isolation

---

## 2. Conformance

### 2.1 Conformance Classes

A **conforming MCP Gateway implementation** is one that satisfies all MUST, REQUIRED, and SHALL requirements in this specification.

A **partially conforming MCP Gateway implementation** is one that satisfies all MUST requirements but MAY lack support for optional features marked with SHOULD or MAY.

### 2.2 Requirements Notation

The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT", "SHOULD", "SHOULD NOT", "RECOMMENDED", "NOT RECOMMENDED", "MAY", and "OPTIONAL" in this document are to be interpreted as described in [RFC 2119](https://www.ietf.org/rfc/rfc2119.txt).

### 2.3 Compliance Levels

Implementations MUST support:

- **Level 1 (Required)**: Basic proxy functionality, stdio backend transport, Streamable HTTP frontend
- **Level 2 (Standard)**: HTTP backend transport, authentication, health endpoints
- **Level 3 (Complete)**: All optional features including variable expressions, timeout configuration

---

## 3. Architecture

### 3.1 Gateway Model

The MCP Gateway follows a **one gateway per MCP server** architecture. Each gateway instance is responsible for a single MCP server, providing complete isolation and independent lifecycle management.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              MCP Client                                     │
│                       (e.g., AI Agent, Copilot)                             │
└───────────┬─────────────────────┬─────────────────────┬─────────────────────┘
            │                     │                     │
            │ Streamable HTTP     │ Streamable HTTP     │ Streamable HTTP
            ▼                     ▼                     ▼
┌───────────────────┐   ┌───────────────────┐   ┌───────────────────┐
│   gh-aw-mcpg      │   │   gh-aw-mcpg      │   │   gh-aw-mcpg      │
│   Gateway 1       │   │   Gateway 2       │   │   Gateway N       │
│   (Port 8080)     │   │   (Port 8081)     │   │   (Port 808N)     │
│  ┌─────────────┐  │   │  ┌─────────────┐  │   │  ┌─────────────┐  │
│  │ Auth Layer  │  │   │  │ Auth Layer  │  │   │  │ Auth Layer  │  │
│  └─────────────┘  │   │  └─────────────┘  │   │  └─────────────┘  │
│  ┌─────────────┐  │   │  ┌─────────────┐  │   │  ┌─────────────┐  │
│  │ Protocol    │  │   │  │ Protocol    │  │   │  │ Protocol    │  │
│  │ Translation │  │   │  │ Translation │  │   │  │ Translation │  │
│  └─────────────┘  │   │  └─────────────┘  │   │  └─────────────┘  │
└─────────┬─────────┘   └─────────┬─────────┘   └─────────┬─────────┘
          │                       │                       │
          │ stdio                 │ HTTP                  │ stdio
          ▼                       ▼                       ▼
   ┌────────────┐          ┌────────────┐          ┌────────────┐
   │ GitHub     │          │ Remote     │          │ Custom     │
   │ MCP Server │          │ MCP Server │          │ MCP Server │
   │ (Docker)   │          │ (HTTP)     │          │ (Process)  │
   └────────────┘          └────────────┘          └────────────┘
```

### 3.2 Transport Support

Each gateway instance MUST support the following transport configurations:

**Frontend Transport (Client-Facing)**:
- **Streamable HTTP**: All client communication uses Streamable HTTP transport as defined in the MCP specification

**Backend Transport (Server-Facing)**:
- **stdio**: Standard input/output based communication with local processes or containers
- **HTTP**: Direct HTTP-based communication with remote MCP servers

The gateway translates between Streamable HTTP (frontend) and the server's native transport (backend).

### 3.3 Per-Server Deployment

Each MCP server requires its own gateway instance. The deployment model:

1. **One Gateway Per Server**: Each configured MCP server gets a dedicated gateway process
2. **Independent Ports**: Each gateway listens on a unique port
3. **Independent Authentication**: Each gateway has its own API key
4. **Independent Lifecycle**: Gateway instances start, stop, and restart independently

Example deployment for a workflow with three MCP servers:

| Server Name | Gateway Port | Backend Transport | Backend Target |
|-------------|--------------|-------------------|----------------|
| github      | 8080         | stdio             | Docker container |
| playwright  | 8081         | stdio             | Docker container |
| remote-api  | 8082         | HTTP              | https://api.example.com |

### 3.4 Operational Model

Each gateway instance operates in headless mode:

1. Configuration is provided via **stdin** (JSON format) at startup
2. Secrets are provided via **environment variables**
3. Startup output is written to **stdout** (gateway endpoint information)
4. Error messages are written to **stdout** as error payloads
5. Streamable HTTP server accepts client requests on the configured port

---

## 4. Configuration

### 4.1 Configuration Format

Each gateway instance MUST accept configuration via stdin in JSON format. Since each gateway manages exactly one MCP server, the configuration is per-server.

:::note
This configuration format is specific to the `gh-aw-mcpg` gateway implementation. The workflow frontmatter schema (`sandbox.mcp`) provides a higher-level configuration that the orchestrator uses to spawn gateway instances with this format.
:::

#### 4.1.1 Configuration Structure

```json
{
  "server": {
    "name": "server-name",
    "command": "string",
    "args": ["string"],
    "container": "string",
    "entrypointArgs": ["string"],
    "env": {
      "VAR_NAME": "value"
    },
    "type": "stdio",
    "url": "string"
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

Each gateway manages exactly one MCP server. The `server` section MUST support:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Unique identifier for the MCP server |
| `command` | string | Conditional* | Executable command for stdio backend |
| `args` | array[string] | No | Command arguments |
| `container` | string | Conditional* | Container image for the MCP server (mutually exclusive with command) |
| `entrypointArgs` | array[string] | No | Arguments passed to container entrypoint (container only) |
| `env` | object | No | Environment variables for the server process |
| `type` | string | No | Backend transport type: "stdio" or "http" (default: "stdio") |
| `url` | string | Conditional** | Backend HTTP endpoint URL for HTTP servers |

*Either `command` or `container` is required for stdio backend  
**Required for HTTP backend

#### 4.1.3 Gateway Configuration Fields

The `gateway` section configures the gateway's Streamable HTTP frontend and operational behavior:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `port` | integer | 8080 | Streamable HTTP server port |
| `apiKey` | string | (none) | API key for frontend authentication |
| `domain` | string | localhost | Gateway domain (localhost or host.docker.internal) |
| `startupTimeout` | integer | 30 | Backend server startup timeout in seconds |
| `toolTimeout` | integer | 60 | Tool invocation timeout in seconds |

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

Configuration for a single gateway instance managing the GitHub MCP server:

```json
{
  "server": {
    "name": "github",
    "command": "docker",
    "args": ["run", "-i", "--rm", "ghcr.io/github/github-mcp-server:latest"],
    "env": {
      "GITHUB_TOKEN": "${GITHUB_PERSONAL_ACCESS_TOKEN}"
    }
  },
  "gateway": {
    "port": 8080,
    "apiKey": "generated-api-key"
  }
}
```

If `GITHUB_PERSONAL_ACCESS_TOKEN` is not set in the environment:

```
Error: undefined environment variable referenced: GITHUB_PERSONAL_ACCESS_TOKEN
Required by: server.env.GITHUB_TOKEN
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
3. NOT start the Streamable HTTP server
4. NOT initialize the MCP server backend

---

## 5. Protocol Behavior

For complete details on the Model Context Protocol, see the [Model Context Protocol Specification](https://spec.modelcontextprotocol.io/).

### 5.1 Streamable HTTP Frontend

Each gateway instance provides a Streamable HTTP transport interface for MCP clients. This enables efficient bidirectional communication with support for streaming responses.

#### 5.1.1 Endpoint Structure

The gateway MUST expose the following HTTP endpoints at `http://{domain}:{port}`:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/mcp` | POST | MCP protocol endpoint for tool invocations |
| `/health` | GET | Health check endpoint |

Note: Since each gateway manages exactly one MCP server, the endpoint does not require a server name parameter. The full URL is `http://{domain}:{port}/mcp`.

#### 5.1.2 Streamable HTTP Transport

The gateway MUST implement Streamable HTTP transport as defined in the MCP specification:

**Request Format**:

```http
POST /mcp HTTP/1.1
Content-Type: application/json
Authorization: Bearer {apiKey}
Accept: text/event-stream

{
  "jsonrpc": "2.0",
  "method": "string",
  "params": {},
  "id": "string|number"
}
```

**Response Format** (Streaming):

```http
HTTP/1.1 200 OK
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive

event: message
data: {"jsonrpc":"2.0","result":{},"id":"string|number"}

```

**Response Format** (Non-Streaming):

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

#### 5.1.3 Request Processing

The gateway MUST:

1. Accept incoming Streamable HTTP requests on the `/mcp` endpoint
2. Authenticate the request using the configured API key
3. Translate the request to the backend transport format
4. Forward the request to the MCP server
5. Translate the response back to Streamable HTTP
6. Return the response to the client

### 5.2 Backend Protocol Translation

#### 5.2.1 Streamable HTTP to stdio

For stdio-based MCP servers, the gateway MUST:

1. Start the server process on first request (lazy initialization)
2. Translate Streamable HTTP request to JSON-RPC and write to server's stdin
3. Read JSON-RPC response from server's stdout
4. Translate response to Streamable HTTP format
5. Maintain server process for subsequent requests
6. Buffer partial responses until complete JSON is received

#### 5.2.2 Streamable HTTP to HTTP

For HTTP-based MCP servers, the gateway MUST:

1. Translate Streamable HTTP request to the backend HTTP format
2. Forward the request to the server's URL
3. Apply any configured headers or authentication
4. Translate the backend response to Streamable HTTP
5. Handle HTTP-level errors appropriately

#### 5.2.3 Tool Signature Preservation

The gateway MUST NOT modify:

- Tool names
- Tool parameters
- Tool return values
- Method signatures

This ensures transparent proxying without name mangling or schema transformation.

### 5.3 Timeout Handling

#### 5.3.1 Startup Timeout

The gateway SHOULD enforce `startupTimeout` for server initialization:

1. Start timer when server process is launched
2. Wait for server ready signal (stdio) or successful health check (HTTP)
3. If timeout expires, kill server process and return error
4. Log timeout error with server name and elapsed time

#### 5.3.2 Tool Timeout

The gateway SHOULD enforce `toolTimeout` for individual tool invocations:

1. Start timer when RPC request is sent to server
2. Wait for complete response
3. If timeout expires, return timeout error to client
4. Log timeout with server name, method, and elapsed time

### 5.4 Stdout Configuration Output

After successful initialization, each gateway instance MUST:

1. Write connection details to stdout for client configuration
2. Include the Streamable HTTP endpoint information:
   ```json
   {
     "server": {
       "name": "server-name",
       "url": "http://{domain}:{port}/mcp",
       "transport": "streamable-http",
       "headers": {
         "Authorization": "Bearer {apiKey}"
       }
     }
   }
   ```
3. Write configuration as a single JSON document
4. Flush stdout buffer
5. Continue serving requests

This allows the orchestrator to dynamically configure MCP clients with gateway endpoints.

---

## 6. Server Isolation

### 6.1 Per-Gateway Isolation

Since each gateway manages exactly one MCP server, isolation is inherent in the architecture:

1. **Process Isolation**: Each gateway runs as a separate process
2. **Port Isolation**: Each gateway listens on its own unique port
3. **Authentication Isolation**: Each gateway has its own API key
4. **Lifecycle Isolation**: Gateway instances start, stop, and restart independently

### 6.2 Backend Process Management

For stdio backend servers, the gateway MUST:

1. Launch the server as a child process or container
2. Maintain exclusive access to the server's stdin/stdout/stderr streams
3. Terminate the child process/container on gateway shutdown

For HTTP backend servers, the gateway MUST:

1. Maintain a connection pool to the backend server
2. Handle connection lifecycle (reconnection, timeout, etc.)
3. Clean up connections on gateway shutdown

### 6.3 Security Boundaries

The gateway MUST:

- Isolate environment variables per gateway instance
- NOT log API keys or secrets in plaintext
- NOT expose backend server implementation details to clients
- NOT forward internal errors that could leak sensitive information

---

## 7. Authentication

### 7.1 API Key Authentication

When `gateway.apiKey` is configured, the gateway MUST:

1. Require `Authorization: {apiKey}` header on all RPC requests
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

### 8.1 Health Endpoint

Each gateway instance MUST expose a health endpoint for monitoring:

#### 8.1.1 Health Check (`/health`)

```http
GET /health HTTP/1.1
```

Response:

```json
{
  "status": "healthy" | "unhealthy",
  "server": {
    "name": "server-name",
    "status": "running" | "stopped" | "error",
    "transport": "stdio" | "http",
    "uptime": 12345
  },
  "gateway": {
    "port": 8080,
    "uptime": 12345
  }
}
```

### 8.2 Health Check Behavior

The gateway SHOULD:

1. Periodically check backend server health (every 30 seconds)
2. Restart failed stdio backend servers automatically
3. Mark HTTP backend servers unhealthy if unreachable
4. Include health status in `/health` response
5. Update overall status based on backend server status

---

## 9. Error Handling

### 9.1 Startup Failures

If the backend server fails to start, the gateway MUST:

1. Write detailed error to stdout as an error payload including:
   - Server name
   - Command/URL attempted
   - Error message from server process
   - Environment variable status
   - Stdout/stderr from failed process
2. Exit with status code 1
3. NOT start the Streamable HTTP frontend server

### 9.2 Runtime Errors

For runtime errors, the gateway MUST:

1. Log errors to stdout as error payloads with:
   - Timestamp
   - Server name
   - Request ID
   - Error details
2. Return JSON-RPC error response to client
3. Continue serving requests if possible
4. Attempt to restart failed stdio backend servers

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

Since each gateway manages exactly one server, graceful degradation is limited to:

1. Returning clear error messages when backend is unavailable
2. Attempting automatic recovery for transient failures
3. Providing health status updates for monitoring systems

---

## 10. Compliance Testing

### 10.1 Test Suite Requirements

A conforming implementation MUST pass the following test categories:

#### 10.1.1 Configuration Tests

- **T-CFG-001**: Valid stdio backend configuration
- **T-CFG-002**: Valid HTTP backend configuration
- **T-CFG-003**: Variable expression resolution
- **T-CFG-004**: Undefined variable error detection
- **T-CFG-005**: Unknown field rejection
- **T-CFG-006**: Missing required field detection
- **T-CFG-007**: Invalid type detection
- **T-CFG-008**: Port range validation

#### 10.1.2 Streamable HTTP Frontend Tests

- **T-SH-001**: Streamable HTTP endpoint availability
- **T-SH-002**: Server-sent events streaming
- **T-SH-003**: Non-streaming response fallback
- **T-SH-004**: Content-Type negotiation
- **T-SH-005**: Connection keep-alive

#### 10.1.3 Backend Protocol Translation Tests

- **T-PTL-001**: Streamable HTTP to stdio translation
- **T-PTL-002**: Streamable HTTP to HTTP translation
- **T-PTL-003**: Tool signature preservation
- **T-PTL-004**: Concurrent request handling
- **T-PTL-005**: Large payload handling
- **T-PTL-006**: Partial response buffering

#### 10.1.4 Per-Gateway Isolation Tests

- **T-ISO-001**: Process isolation per gateway
- **T-ISO-002**: Port isolation verification
- **T-ISO-003**: API key isolation verification
- **T-ISO-004**: Lifecycle independence

#### 10.1.5 Authentication Tests

- **T-AUTH-001**: Valid token acceptance
- **T-AUTH-002**: Invalid token rejection
- **T-AUTH-003**: Missing token rejection
- **T-AUTH-004**: Health endpoint exemption
- **T-AUTH-005**: Bearer token format support

#### 10.1.6 Timeout Tests

- **T-TMO-001**: Backend startup timeout enforcement
- **T-TMO-002**: Tool timeout enforcement
- **T-TMO-003**: Timeout error messaging
- **T-TMO-004**: Streaming timeout handling

#### 10.1.7 Health Monitoring Tests

- **T-HLT-001**: Health endpoint availability
- **T-HLT-002**: Backend server status reporting
- **T-HLT-003**: Uptime tracking
- **T-HLT-004**: Automatic restart behavior

#### 10.1.8 Error Handling Tests

- **T-ERR-001**: Startup failure reporting
- **T-ERR-002**: Runtime error handling
- **T-ERR-003**: Invalid request handling
- **T-ERR-004**: Backend crash recovery
- **T-ERR-005**: Error message quality

### 10.2 Compliance Checklist

| Requirement | Test ID | Level | Status |
|-------------|---------|-------|--------|
| Configuration parsing | T-CFG-* | 1 | Required |
| Streamable HTTP frontend | T-SH-* | 1 | Required |
| Stdio backend | T-PTL-001 | 1 | Required |
| HTTP backend | T-PTL-002 | 2 | Standard |
| Authentication | T-AUTH-* | 2 | Standard |
| Timeout handling | T-TMO-* | 3 | Optional |
| Health monitoring | T-HLT-* | 2 | Standard |
| Per-gateway isolation | T-ISO-* | 1 | Required |
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

#### A.1 Basic Stdio Backend

Configuration for a gateway with stdio backend:

```json
{
  "server": {
    "name": "example",
    "command": "node",
    "args": ["server.js"],
    "env": {
      "API_KEY": "${MY_API_KEY}"
    }
  },
  "gateway": {
    "port": 8080,
    "apiKey": "gateway-secret-token"
  }
}
```

#### A.2 HTTP Backend

Configuration for a gateway with HTTP backend:

```json
{
  "server": {
    "name": "remote-api",
    "type": "http",
    "url": "https://api.example.com/mcp"
  },
  "gateway": {
    "port": 8081,
    "apiKey": "gateway-secret-token",
    "startupTimeout": 60,
    "toolTimeout": 120
  }
}
```

#### A.3 Docker Container Backend

Configuration for a gateway with Docker container backend:

```json
{
  "server": {
    "name": "github",
    "container": "ghcr.io/github/github-mcp-server:latest",
    "env": {
      "GITHUB_PERSONAL_ACCESS_TOKEN": "${GITHUB_TOKEN}"
    }
  },
  "gateway": {
    "port": 8082,
    "domain": "host.docker.internal"
  }
}
```

#### A.4 Multi-Gateway Deployment

Example orchestration of multiple gateways for a workflow:

```bash
# Start gateway for GitHub MCP server
echo '{"server":{"name":"github","container":"ghcr.io/github/github-mcp-server:latest"},"gateway":{"port":8080}}' | gh-aw-mcpg &

# Start gateway for Playwright MCP server
echo '{"server":{"name":"playwright","container":"mcr.microsoft.com/playwright/mcp"},"gateway":{"port":8081}}' | gh-aw-mcpg &

# Start gateway for remote API
echo '{"server":{"name":"remote","type":"http","url":"https://api.example.com/mcp"},"gateway":{"port":8082}}' | gh-aw-mcpg &
```

### Appendix B: Error Code Reference

| Code | Name | Description |
|------|------|-------------|
| -32700 | Parse error | Invalid JSON received |
| -32600 | Invalid request | Invalid JSON-RPC request |
| -32601 | Method not found | Method does not exist |
| -32602 | Invalid params | Invalid method parameters |
| -32603 | Internal error | Internal JSON-RPC error |
| -32000 | Server error | Generic server error |
| -32001 | Backend unavailable | Backend server not responding |
| -32002 | Backend timeout | Backend response timeout |
| -32003 | Authentication failed | Invalid or missing API key |

### Appendix C: Security Considerations

#### C.1 Credential Handling

- API keys MUST NOT be logged in plaintext
- Environment variables MUST be isolated per gateway instance
- Secrets SHOULD be cleared from memory after use
- Backend credentials MUST NOT be exposed to clients

#### C.2 Network Security

- Gateway SHOULD support TLS/HTTPS for frontend connections
- Backend URLs SHOULD be validated before connection
- Rate limiting SHOULD be implemented per gateway instance
- Cross-origin requests SHOULD be restricted

#### C.3 Process Security

- Backend processes SHOULD run with minimal privileges
- Resource limits SHOULD be enforced (CPU, memory, file descriptors)
- Temporary files SHOULD be cleaned up on shutdown
- Process monitoring SHOULD detect anomalies

### Appendix D: Streamable HTTP Transport

The MCP Gateway uses Streamable HTTP transport for client-facing communication. This transport:

1. **Supports Streaming**: Responses can be streamed using Server-Sent Events (SSE)
2. **Backward Compatible**: Falls back to single-response mode when streaming is not requested
3. **Connection Efficient**: Supports HTTP/1.1 keep-alive and HTTP/2 multiplexing
4. **Standard Protocol**: Follows the MCP Streamable HTTP transport specification

#### D.1 Content-Type Negotiation

| Client Accept Header | Response Content-Type | Behavior |
|---------------------|----------------------|----------|
| `text/event-stream` | `text/event-stream` | Streaming SSE response |
| `application/json` | `application/json` | Single JSON response |
| `*/*` or missing | `application/json` | Single JSON response (default) |

---

## References

### Normative References

- **[RFC 2119]** Key words for use in RFCs to Indicate Requirement Levels
- **[JSON-RPC 2.0]** JSON-RPC 2.0 Specification
- **[MCP]** Model Context Protocol Specification
- **[MCP-Streamable-HTTP]** MCP Streamable HTTP Transport Specification

### Informative References

- **[MCP-Config]** MCP Configuration Format
- **[HTTP/1.1]** Hypertext Transfer Protocol -- HTTP/1.1
- **[SSE]** Server-Sent Events (HTML Living Standard)

---

## Change Log

### Version 2.0.0 (Draft)

- **Architecture Change**: Updated to one gateway per MCP server model
- **Frontend Protocol**: Changed from HTTP/JSON-RPC to Streamable HTTP transport
- **Backend Protocols**: Clarified HTTP and stdio backend transport support
- **Configuration Format**: Updated to per-server configuration structure
- **Isolation Model**: Simplified to per-gateway isolation
- **Test Suite**: Updated compliance tests for new architecture

### Version 1.0.0 (Draft)

- Initial specification release
- Unified gateway for multiple servers
- Configuration format definition
- Protocol behavior specification
- Compliance test framework

---

*Copyright © 2026 GitHub, Inc. All rights reserved.*
