---
title: MCP Gateway Specification
description: Formal specification for the Model Context Protocol (MCP) Gateway implementation following W3C conventions
sidebar:
  order: 1350
---

# MCP Gateway Specification

**Version**: 1.0.0  
**Status**: Draft Specification  
**Latest Version**: [mcp-gateway](/gh-aw/reference/mcp-gateway/)  
**Editor**: GitHub Agentic Workflows Team

---

## Abstract

This specification defines the Model Context Protocol (MCP) Gateway, a transparent proxy service that enables unified HTTP access to multiple MCP servers using different transport mechanisms (stdio, HTTP). The gateway provides protocol translation, server isolation, authentication, and health monitoring capabilities.

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

The MCP Gateway serves as a protocol translation layer between MCP clients expecting HTTP-based communication and MCP servers using various transport mechanisms. It enables:

- **Protocol Translation**: Converting between stdio and HTTP transports
- **Unified Access**: Single HTTP endpoint for multiple MCP servers
- **Server Isolation**: Enforcing boundaries between server instances
- **Authentication**: Token-based access control
- **Health Monitoring**: Service availability endpoints

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

- **stdio**: Standard input/output based communication
- **HTTP**: Direct HTTP-based MCP servers

The gateway MUST translate all upstream transports to HTTP for client communication.

### 3.3 Operational Model

The gateway operates in a headless mode:

1. Configuration is provided via **stdin** (JSON format)
2. Secrets are provided via **environment variables**
3. Startup output is written to **stdout** (rewritten configuration)
4. Error messages are written to **stderr**
5. HTTP server accepts client requests on configured port

---

## 4. Configuration

### 4.1 Configuration Format

The gateway MUST accept configuration via stdin in JSON format conforming to the Claude MCP configuration file schema.

#### 4.1.1 Configuration Structure

```json
{
  "mcpServers": {
    "server-name": {
      "command": "string",
      "args": ["string"],
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
| `command` | string | Conditional* | Executable command for stdio servers |
| `args` | array[string] | No | Command arguments |
| `env` | object | No | Environment variables for the server process |
| `type` | string | No | Transport type: "stdio" or "http" (default: "stdio") |
| `url` | string | Conditional** | HTTP endpoint URL for HTTP servers |

*Required for stdio servers  
**Required for HTTP servers

#### 4.1.3 Gateway Configuration Fields

The optional `gateway` section configures gateway-specific behavior:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `port` | integer | 8080 | HTTP server port |
| `apiKey` | string | (none) | Bearer token for authentication |
| `domain` | string | localhost | Gateway domain (localhost or host.docker.internal) |
| `startupTimeout` | integer | 30 | Server startup timeout in seconds |
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
4. Log the undefined variable name to stderr
5. Exit with non-zero status code

#### 4.2.3 Example

Configuration:

```json
{
  "mcpServers": {
    "github": {
      "command": "docker",
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

1. Write a detailed error message to stderr including:
   - The specific validation error
   - The location in the configuration (JSON path)
   - Suggested corrective action
2. Exit with status code 1
3. NOT start the HTTP server
4. NOT initialize any MCP servers

---

## 5. Protocol Behavior

### 5.1 HTTP Server Interface

#### 5.1.1 Endpoint Structure

The gateway MUST expose the following HTTP endpoints:

```
POST /mcp/{server-name}/rpc
GET  /health
GET  /health/ready
GET  /health/live
```

#### 5.1.2 RPC Endpoint Behavior

**Request Format**:

```http
POST /mcp/{server-name}/rpc HTTP/1.1
Content-Type: application/json
Authorization: Bearer {apiKey}

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

#### 5.1.3 Request Routing

The gateway MUST:

1. Extract server name from URL path
2. Validate server exists in configuration
3. Route request to appropriate backend server
4. Translate protocols if necessary (stdio ↔ HTTP)
5. Return response to client

### 5.2 Protocol Translation

#### 5.2.1 Stdio to HTTP

For stdio-based servers, the gateway MUST:

1. Start the server process on first request (lazy initialization)
2. Write JSON-RPC request to server's stdin
3. Read JSON-RPC response from server's stdout
4. Return HTTP response to client
5. Maintain server process for subsequent requests
6. Buffer partial responses until complete JSON is received

#### 5.2.2 HTTP to HTTP

For HTTP-based servers, the gateway MUST:

1. Forward the JSON-RPC request to the server's URL
2. Apply any configured headers or authentication
3. Return the server's response to the client
4. Handle HTTP-level errors appropriately

#### 5.2.3 Tool Signature Preservation

The gateway MUST NOT modify:

- Tool names
- Tool parameters
- Tool return values
- Method signatures

This ensures transparent proxying without name mangling or schema transformation.

### 5.3 Timeout Handling

#### 5.3.1 Startup Timeout

The gateway MUST enforce `startupTimeout` for server initialization:

1. Start timer when server process is launched
2. Wait for server ready signal (stdio) or successful health check (HTTP)
3. If timeout expires, kill server process and return error
4. Log timeout error with server name and elapsed time

#### 5.3.2 Tool Timeout

The gateway MUST enforce `toolTimeout` for individual tool invocations:

1. Start timer when RPC request is sent to server
2. Wait for complete response
3. If timeout expires, return timeout error to client
4. Log timeout with server name, method, and elapsed time

### 5.4 Stdout Configuration Output

After successful initialization, the gateway MUST:

1. Write a complete Claude-formatted MCP server configuration to stdout
2. Include gateway connection details:
   ```json
   {
     "mcpServers": {
       "server-name": {
         "type": "http",
         "url": "http://{domain}:{port}/mcp/server-name/rpc",
         "headers": {
           "Authorization": "Bearer {apiKey}"
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

### 6.1 Process Isolation

For stdio servers, the gateway MUST:

1. Launch each server in a separate process
2. Maintain isolated stdin/stdout/stderr streams
3. Prevent cross-server communication
4. Terminate child processes on gateway shutdown

### 6.2 Resource Isolation

The gateway MUST ensure:

- Each server has isolated environment variables
- File descriptors are not shared between servers
- Network sockets are not shared (for HTTP servers)
- Server failures do not affect other servers

### 6.3 Security Boundaries

The gateway MUST NOT:

- Allow servers to access each other's configuration
- Share authentication credentials between servers
- Expose server implementation details to clients
- Allow cross-server tool invocations

---

## 7. Authentication

### 7.1 Bearer Token Authentication

When `gateway.apiKey` is configured, the gateway MUST:

1. Require `Authorization: Bearer {apiKey}` header on all RPC requests
2. Reject requests with missing or invalid tokens (HTTP 401)
3. Reject requests with malformed Authorization headers (HTTP 400)
4. NOT log API keys in plaintext

### 7.2 Optimal Temporary API Key

The gateway SHOULD support short-lived, auto-rotating API keys:

1. Generate a random API key on startup if not provided
2. Include key in stdout configuration output
3. Support key rotation via signal (e.g., SIGHUP)
4. Invalidate old keys after rotation grace period

### 7.3 Authentication Exemptions

The following endpoints MUST NOT require authentication:

- `/health`
- `/health/live`
- `/health/ready`

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

#### 8.1.2 Liveness Probe (`/health/live`)

```http
GET /health/live HTTP/1.1
```

Returns HTTP 200 if gateway process is running, HTTP 503 otherwise.

#### 8.1.3 Readiness Probe (`/health/ready`)

```http
GET /health/ready HTTP/1.1
```

Returns HTTP 200 if gateway can accept requests, HTTP 503 otherwise.

### 8.2 Health Check Behavior

The gateway SHOULD:

1. Periodically check server health (every 30 seconds)
2. Restart failed stdio servers automatically
3. Mark HTTP servers unhealthy if unreachable
4. Include health status in `/health` response
5. Update readiness based on critical server status

---

## 9. Error Handling

### 9.1 Startup Failures

If any configured server fails to start, the gateway MUST:

1. Write detailed error to stderr including:
   - Server name
   - Command/URL attempted
   - Error message from server process
   - Environment variable status
   - Stdout/stderr from failed process
2. Exit with status code 1
3. NOT start the HTTP server

### 9.2 Runtime Errors

For runtime errors, the gateway MUST:

1. Log errors to stderr with:
   - Timestamp
   - Server name
   - Request ID
   - Error details
2. Return JSON-RPC error response to client
3. Continue serving other requests
4. Attempt to restart failed stdio servers

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

- **T-ISO-001**: Process isolation verification
- **T-ISO-002**: Environment isolation verification
- **T-ISO-003**: Credential isolation verification
- **T-ISO-004**: Cross-server communication prevention
- **T-ISO-005**: Server failure isolation

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

#### A.1 Basic Stdio Server

```json
{
  "mcpServers": {
    "example": {
      "command": "node",
      "args": ["server.js"],
      "env": {
        "API_KEY": "${MY_API_KEY}"
      }
    }
  },
  "gateway": {
    "port": 8080,
    "apiKey": "gateway-secret-token"
  }
}
```

#### A.2 Mixed Transport Configuration

```json
{
  "mcpServers": {
    "local-server": {
      "command": "python",
      "args": ["server.py"],
      "type": "stdio"
    },
    "remote-server": {
      "type": "http",
      "url": "https://api.example.com/mcp"
    }
  },
  "gateway": {
    "port": 8080,
    "startupTimeout": 60,
    "toolTimeout": 120
  }
}
```

#### A.3 Docker-Based Server

```json
{
  "mcpServers": {
    "github": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "-e", "GITHUB_PERSONAL_ACCESS_TOKEN",
        "ghcr.io/github/github-mcp-server:latest"
      ],
      "env": {
        "GITHUB_PERSONAL_ACCESS_TOKEN": "${GITHUB_TOKEN}"
      }
    }
  }
}
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
| -32001 | Server unavailable | Server not responding |
| -32002 | Server timeout | Server response timeout |
| -32003 | Authentication failed | Invalid or missing credentials |

### Appendix C: Security Considerations

#### C.1 Credential Handling

- API keys MUST NOT be logged
- Environment variables MUST be isolated per server
- Secrets SHOULD be cleared from memory after use
- Token rotation SHOULD be supported

#### C.2 Network Security

- Gateway SHOULD support TLS/HTTPS
- Server URLs SHOULD be validated
- Cross-origin requests SHOULD be restricted
- Rate limiting SHOULD be implemented

#### C.3 Process Security

- Server processes SHOULD run with minimal privileges
- Resource limits SHOULD be enforced (CPU, memory, file descriptors)
- Temporary files SHOULD be cleaned up
- Process monitoring SHOULD detect anomalies

---

## References

### Normative References

- **[RFC 2119]** Key words for use in RFCs to Indicate Requirement Levels
- **[JSON-RPC 2.0]** JSON-RPC 2.0 Specification
- **[MCP]** Model Context Protocol Specification

### Informative References

- **[Claude-Config]** Claude Desktop MCP Configuration Format
- **[HTTP/1.1]** Hypertext Transfer Protocol -- HTTP/1.1

---

## Change Log

### Version 1.0.0 (Draft)

- Initial specification release
- Configuration format definition
- Protocol behavior specification
- Compliance test framework

---

*Copyright © 2026 GitHub, Inc. All rights reserved.*
