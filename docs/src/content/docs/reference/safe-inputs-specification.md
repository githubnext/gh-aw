---
title: Safe Inputs Specification
description: Formal specification for Safe Inputs custom MCP tools following W3C conventions
sidebar:
  order: 1360
---

# Safe Inputs Specification

**Version**: 1.1.0  
**Status**: Draft Specification  
**Latest Version**: [safe-inputs-specification](/gh-aw/reference/safe-inputs-specification/)  
**JSON Schema**: [safe-inputs-config.schema.json](/gh-aw/schemas/safe-inputs-config.schema.json)  
**Editor**: GitHub Agentic Workflows Team

---

## Abstract

This specification defines Safe Inputs, an extension to the MCP Gateway that enables inline definition of custom MCP tools directly in workflow frontmatter using JavaScript, shell scripts, Python, or Go. Safe Inputs provides ephemeral, containerized tool execution with controlled secret access through a standardized MCP tools interface. Tool execution is stateless and session-independent, providing process isolation and security boundaries for custom functionality.

## Status of This Document

This section describes the status of this document at the time of publication. This is a draft specification and may be updated, replaced, or made obsolete by other documents at any time.

This document is governed by the GitHub Agentic Workflows project specifications process.

## Table of Contents

1. [Introduction](#1-introduction)
2. [Conformance](#2-conformance)
3. [Architecture](#3-architecture)
4. [Configuration Format](#4-configuration-format)
5. [Tool Execution](#5-tool-execution)
6. [Language Support](#6-language-support)
7. [Security Model](#7-security-model)
8. [Large Output Handling](#8-large-output-handling)
9. [Integration with MCP Gateway](#9-integration-with-mcp-gateway)
10. [Compliance Testing](#10-compliance-testing)

---

## 1. Introduction

### 1.1 Purpose

Safe Inputs enables developers to define custom MCP tools inline in workflow frontmatter without requiring external MCP server implementations. It solves the following problems:

- **Rapid Tool Development**: Define tools directly in workflow without creating separate services
- **Secret Isolation**: Provide controlled access to secrets through explicit environment variable mapping
- **Language Flexibility**: Support multiple implementation languages (JavaScript, Shell, Python, Go)
- **Process Isolation**: Execute tools in containerized environments with security boundaries
- **Ephemeral Execution**: Stateless tool invocations without session management overhead

### 1.2 Scope

This specification covers:

- Safe Inputs configuration format in workflow frontmatter
- Tool definition structure and validation rules
- Supported implementation languages and their execution models
- Secret access and environment variable handling
- Tool input/output schemas and validation
- Large output handling mechanisms
- Integration with MCP Gateway infrastructure

This specification does NOT cover:

- MCP Gateway core protocol (see [MCP Gateway Specification](/gh-aw/reference/mcp-gateway/))
- MCP protocol semantics (see [Model Context Protocol Specification](https://spec.modelcontextprotocol.io/))
- External MCP server implementations
- Agent client implementations
- UI or interactive features

### 1.3 Design Goals

Safe Inputs is designed for:

- **Developer Convenience**: Minimal configuration overhead for common tool patterns
- **Security by Default**: Explicit secret access, process isolation, output sanitization
- **Stateless Execution**: No session management, each invocation is independent
- **Language Agnostic**: Support multiple implementation languages with consistent behavior
- **Gateway Integration**: Seamless integration with MCP Gateway as configuration extension

### 1.4 Relationship to MCP Gateway

Safe Inputs is an **extension** to the MCP Gateway Specification. The MCP Gateway allows additional fields in its configuration format, and Safe Inputs leverages this extensibility to provide inline tool definitions. Safe Inputs configurations are processed during workflow compilation and translated into MCP server configurations that are gatewayed by the MCP Gateway infrastructure.

---

## 2. Conformance

### 2.1 Conformance Classes

A **conforming Safe Inputs implementation** is one that satisfies all MUST, REQUIRED, and SHALL requirements in this specification.

A **partially conforming Safe Inputs implementation** is one that satisfies all MUST requirements for JavaScript tools but MAY lack support for Shell, Python, or Go implementations.

### 2.2 Requirements Notation

The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT", "SHOULD", "SHOULD NOT", "RECOMMENDED", "NOT RECOMMENDED", "MAY", and "OPTIONAL" in this document are to be interpreted as described in [RFC 2119](https://www.ietf.org/rfc/rfc2119.txt).

### 2.3 Compliance Levels

Implementations MUST support:

- **Level 1 (Required)**: JavaScript tools, basic input validation, HTTP transport
- **Level 2 (Standard)**: Shell and Python tools, timeout handling, secret isolation
- **Level 3 (Complete)**: Go tools, large output handling, all optional features

---

## 3. Architecture

### 3.1 System Architecture

```
┌─────────────────────────────────────────────────────────┐
│                 Workflow Frontmatter                    │
│                  (safe-inputs:)                         │
└──────────────────────┬──────────────────────────────────┘
                       │ Compilation
                       ▼
┌─────────────────────────────────────────────────────────┐
│              Safe Inputs MCP Server                     │
│  ┌───────────────────────────────────────────────────┐  │
│  │  Tool Registry & Configuration Loader             │  │
│  └───────────────────────────────────────────────────┘  │
│  ┌───────────────────────────────────────────────────┐  │
│  │  HTTP MCP Server (JSON-RPC over HTTP)            │  │
│  └───────────────────────────────────────────────────┘  │
└──────┬──────────────┬──────────────┬───────────────────┘
       │              │              │
       │ JavaScript   │ Shell        │ Python/Go
       ▼              ▼              ▼
  ┌─────────┐   ┌─────────┐   ┌─────────┐
  │ In-     │   │ Docker  │   │ Docker  │
  │ Process │   │ Container│   │ Container│
  │         │   │         │   │         │
  └─────────┘   └─────────┘   └─────────┘
```

### 3.2 Execution Model

Safe Inputs operates with the following execution model:

1. **Compilation Phase**: Workflow frontmatter is parsed and validated
2. **Server Startup**: Safe Inputs MCP server starts with tool configurations
3. **Tool Registration**: Each tool is registered with the MCP server
4. **Invocation**: Agent invokes tool via MCP protocol (HTTP transport)
5. **Execution**: Tool handler executes in appropriate runtime environment
6. **Response**: Result is returned via JSON-RPC response
7. **Cleanup**: Ephemeral resources are cleaned up after invocation

### 3.3 Transport Model

Safe Inputs MUST use HTTP transport for MCP communication. The transport architecture is:

- **Client → Gateway**: HTTP with JSON-RPC payloads
- **Gateway → Safe Inputs Server**: HTTP with JSON-RPC payloads
- **Safe Inputs Server**: HTTP server on configurable port (default: 3000)
- **Authentication**: API key-based authentication via Authorization header

Stdio transport is NOT supported for Safe Inputs.

---

## 4. Configuration Format

### 4.1 Frontmatter Structure

Safe Inputs configuration MUST be defined in the `safe-inputs:` section of workflow frontmatter:

```yaml
safe-inputs:
  tool-name:
    description: "Tool description"
    inputs:
      param-name:
        type: string
        required: true
        description: "Parameter description"
        default: "default-value"
    script: |
      // JavaScript implementation
    env:
      SECRET_NAME: "${{ secrets.SECRET_NAME }}"
    timeout: 60
```

**JSON Schema**: [safe-inputs-config.schema.json](/gh-aw/schemas/safe-inputs-config.schema.json)

### 4.2 Tool Configuration Fields

Each tool configuration MUST contain:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `description` | string | Yes | Human-readable tool description shown to agents |

Each tool configuration MAY contain:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `inputs` | object | No | Input parameter definitions (JSON Schema format) |
| `script` | string | Conditional* | JavaScript (CommonJS) implementation |
| `run` | string | Conditional* | Shell script implementation |
| `py` | string | Conditional* | Python script implementation |
| `go` | string | Conditional* | Go code implementation |
| `env` | object | No | Environment variables (typically secrets) |
| `timeout` | integer | No | Execution timeout in seconds (default: 60, applies to run/py/go only) |
| `dependencies` | array[string] | No | Package dependencies to install in execution environment (runtime-specific) |

*Exactly ONE of `script`, `run`, `py`, or `go` MUST be provided per tool.

### 4.3 Dependencies

The `dependencies` field allows specification of runtime dependencies that MUST be installed before tool execution. The package manager is inferred from the implementation language:

- **JavaScript (`script:`)**: Dependencies installed via `npm install`
- **Shell (`run:`)**: Dependencies installed via appropriate package manager (apt, yum, etc.)
- **Python (`py:`)**: Dependencies installed via `pip install`
- **Go (`go:`)**: Dependencies installed via `go get`

**Example**:

```yaml
safe-inputs:
  analyze-json:
    description: "Analyze JSON with jq"
    inputs:
      json:
        type: string
        required: true
    run: |
      echo "$INPUT_JSON" | jq '.data | length'
    dependencies:
      - jq
    timeout: 30
```

**Python Dependencies Example**:

```yaml
safe-inputs:
  fetch-url:
    description: "Fetch URL with requests library"
    inputs:
      url:
        type: string
        required: true
    py: |
      import requests
      import json
      response = requests.get(inputs.get('url'))
      print(json.dumps({"status": response.status_code, "content_length": len(response.text)}))
    dependencies:
      - requests
    timeout: 60
```

**Requirements**:
- Implementations MUST install dependencies before first tool invocation
- Dependencies SHOULD be cached for subsequent invocations
- Dependency installation failures MUST result in tool execution errors
- Package names MUST be valid for the target package manager
- Implementations MAY enforce security policies on allowed packages

### 4.4 Input Parameter Schema

Input parameters follow JSON Schema conventions:

```yaml
inputs:
  param-name:
    type: string|number|boolean|array|object
    description: "Parameter description"
    required: true|false
    default: value
    enum: [value1, value2, ...]
```

**Supported Types**:
- `string` - Text values
- `number` - Numeric values (integer or float)
- `boolean` - True/false values
- `array` - List of values
- `object` - Structured data

**Validation Options**:
- `required: true` - Parameter must be provided by agent
- `default: value` - Default value if not provided
- `enum: [...]` - Restrict to specific values
- `description: "..."` - Help text for agent tool selection

### 4.5 Environment Variables

Environment variables provide secret access to tools:

```yaml
env:
  API_KEY: "${{ secrets.SERVICE_API_KEY }}"
  DATABASE_URL: "${{ secrets.DATABASE_URL }}"
```

**Requirements**:
- Environment variable values MAY contain GitHub Actions secret expressions (`${{ secrets.NAME }}`)
- Secret expressions MUST be resolved during compilation
- Secrets MUST be masked in logs
- Only explicitly declared environment variables are available to tools

### 4.6 Timeout Configuration

Timeout applies to Shell (`run:`), Python (`py:`), and Go (`go:`) tools:

```yaml
timeout: 120  # 2 minutes
```

**Behavior**:
- Default timeout: 60 seconds
- Minimum timeout: 1 second
- Maximum timeout: Implementation-dependent (SHOULD be at least 600 seconds)
- Timeout enforcement: Process MUST be terminated with SIGTERM, then SIGKILL after grace period
- JavaScript tools (`script:`) execute in-process and do NOT have timeout enforcement

### 4.7 Validation Requirements

Implementations MUST validate:

1. **Required Fields**: `description` field is present and non-empty
2. **Mutually Exclusive Implementations**: Exactly one of `script`, `run`, `py`, `go` is provided
3. **Input Schema**: Input definitions follow JSON Schema conventions
4. **Timeout Range**: Timeout value is positive integer (minimum 1 second)
5. **Environment Variables**: Environment variable names are valid identifiers (uppercase alphanumeric with underscores)
6. **Tool Names**: Tool names match pattern `^[a-zA-Z][a-zA-Z0-9_-]*$`
7. **Dependencies**: Dependency names are valid for target package manager

Implementations SHOULD validate:

1. **Script Syntax**: Syntax errors in implementation code (language-specific)
2. **Input Types**: Input parameter types are supported JSON Schema types
3. **Reserved Names**: Tool names do not conflict with built-in MCP methods
4. **Description Length**: Tool descriptions are clear and concise (recommended 10-200 characters)
5. **Timeout Reasonableness**: Timeout values are reasonable for tool purpose (warn if >600 seconds)

---

## 5. Tool Execution

### 5.1 Invocation Flow

1. Agent sends JSON-RPC request to Safe Inputs MCP server
2. Server validates request format and authentication
3. Server validates tool inputs against schema
4. Server dispatches to appropriate language handler
5. Handler executes tool implementation
6. Handler captures output and errors
7. Server returns JSON-RPC response to agent

### 5.2 Input Validation

Implementations MUST:

1. Validate all required parameters are provided
2. Reject requests with missing required parameters (JSON-RPC error -32602)
3. Apply default values for optional parameters
4. Validate enum constraints if specified
5. Coerce types where possible (e.g., string to number)

### 5.3 Error Handling

Implementations MUST return JSON-RPC errors for:

- **Missing Tool** (-32601): Tool name not found in registry
- **Invalid Parameters** (-32602): Required parameter missing or invalid type
- **Execution Error** (-32603): Tool execution failed (syntax error, runtime error, timeout)
- **Internal Error** (-32603): Server-side error during processing

Error responses MUST include:
- Standard JSON-RPC error structure
- Human-readable error message
- Error details in `data` field (stack trace, line numbers, etc.)

### 5.4 Execution Isolation

Each tool invocation MUST be isolated:

- **Process Isolation**: Shell/Python/Go tools execute in separate containers
- **Environment Isolation**: Only declared environment variables are available
- **Filesystem Isolation**: Tools have access only to their execution environment
- **Network Isolation**: Tools inherit network permissions from workflow configuration

JavaScript tools execute in-process but MUST have:
- Isolated module scope
- No access to server internals
- Limited execution time (via V8 isolates or similar)

### 5.5 Output Capture

Implementations MUST:

1. Capture stdout from tool execution
2. Parse JSON output if possible
3. Return output in JSON-RPC result field
4. Handle large outputs per Section 8 (Large Output Handling)

For Shell/Python/Go tools:
- Stdout contains the tool result (MUST be valid JSON)
- Stderr is logged but not returned to agent
- Exit code 0 indicates success
- Non-zero exit code indicates failure

For JavaScript tools:
- Return value is the tool result
- Thrown errors indicate failure
- Async functions are awaited

---

## 6. Language Support

### 6.1 JavaScript Tools (`script:`)

#### 6.1.1 Execution Environment

JavaScript tools MUST:
- Execute in Node.js environment
- Use CommonJS module format
- Be wrapped in async function with destructured inputs
- Have access to `process.env` for secrets
- Have access to GitHub Actions global objects (`github`, `context`, `core`, `io`, `exec`, `glob`, `artifact`)

#### 6.1.2 Available Global Objects

JavaScript tools have access to standard GitHub Actions JavaScript libraries without explicit import:

- **`github`**: GitHub API client from `@actions/github`
- **`context`**: Workflow context information from `@actions/github`
- **`core`**: Actions core utilities from `@actions/core`
- **`io`**: File I/O utilities from `@actions/io`
- **`exec`**: Command execution utilities from `@actions/exec`
- **`glob`**: File pattern matching from `@actions/glob`
- **`artifact`**: Artifact management from `@actions/artifact`

**Example using global objects**:

```yaml
safe-inputs:
  create-issue:
    description: "Create a GitHub issue"
    inputs:
      title:
        type: string
        required: true
      body:
        type: string
        required: true
    script: |
      const octokit = github.getOctokit(process.env.GITHUB_TOKEN);
      const { data } = await octokit.rest.issues.create({
        owner: context.repo.owner,
        repo: context.repo.repo,
        title,
        body
      });
      return { number: data.number, url: data.html_url };
    env:
      GITHUB_TOKEN: "${{ secrets.GITHUB_TOKEN }}"
```

**Requirements**:
- Global objects MUST be available without `require()` statements
- Tools MAY use these globals alongside user code
- Implementations MUST provide same version of libraries as GitHub Actions runtime
- No restrictions on where tools execute (in-process or containerized)

#### 6.1.3 Code Wrapping

Implementation code is wrapped as:

```javascript
async function execute(inputs) {
  const { param1, param2 } = inputs;
  // User code here
}
```

#### 6.1.4 Example

```yaml
safe-inputs:
  greet-user:
    description: "Greet a user by name"
    inputs:
      name:
        type: string
        required: true
    script: |
      return { message: `Hello, ${name}!` };
```

### 6.2 Shell Tools (`run:`)

#### 6.2.1 Execution Environment

Shell tools MUST:
- Execute in bash shell
- Run in containerized environment (Docker)
- Have inputs as environment variables with `INPUT_` prefix
- Output valid JSON to stdout

#### 6.2.2 Input Mapping

Input parameters are mapped to environment variables:
- Parameter `repo` becomes `$INPUT_REPO`
- Parameter `state` becomes `$INPUT_STATE`
- Naming convention: `INPUT_${UPPERCASE_PARAM_NAME}`

#### 6.2.3 Example

```yaml
safe-inputs:
  list-prs:
    description: "List pull requests"
    inputs:
      repo:
        type: string
        required: true
      state:
        type: string
        default: "open"
    run: |
      gh pr list --repo "$INPUT_REPO" --state "$INPUT_STATE" --json number,title
    env:
      GH_TOKEN: "${{ secrets.GITHUB_TOKEN }}"
    timeout: 30
```

### 6.3 Python Tools (`py:`)

#### 6.3.1 Execution Environment

Python tools MUST:
- Execute using Python 3.10+ interpreter
- Run in containerized environment (Docker)
- Have access to standard library modules
- Receive inputs as dictionary variable `inputs`
- Output valid JSON to stdout

#### 6.3.2 Input Access

Input parameters are available via `inputs` dictionary:
- `inputs.get('param_name')` - Access parameter value
- `inputs.get('param_name', default)` - Access with default
- Parameters use original names (not uppercased)

#### 6.3.3 Example

```yaml
safe-inputs:
  analyze-data:
    description: "Analyze numeric data"
    inputs:
      numbers:
        type: string
        description: "Comma-separated numbers"
        required: true
    py: |
      import json
      
      numbers_str = inputs.get('numbers', '')
      numbers = [float(x.strip()) for x in numbers_str.split(',') if x.strip()]
      
      result = {
          "count": len(numbers),
          "sum": sum(numbers),
          "average": sum(numbers) / len(numbers) if numbers else 0
      }
      
      print(json.dumps(result))
    timeout: 60
```

### 6.4 Go Tools (`go:`)

#### 6.4.1 Execution Environment

Go tools MUST:
- Execute using `go run` command
- Run in containerized environment (Docker)
- Have access to standard library imports
- Receive inputs as `map[string]any` from stdin
- Output valid JSON to stdout

#### 6.4.2 Code Wrapping

Implementation code is wrapped in:

```go
package main

import (
    "encoding/json"
    "fmt"
    "io"
    "os"
)

func main() {
    // Parse inputs from stdin
    var inputs map[string]any
    decoder := json.NewDecoder(os.Stdin)
    if err := decoder.Decode(&inputs); err != nil {
        fmt.Fprintf(os.Stderr, "Error parsing inputs: %v\n", err)
        os.Exit(1)
    }
    
    // User code here
}
```

#### 6.4.3 Available Imports

The following imports are automatically included:
- `encoding/json` - JSON encoding/decoding
- `fmt` - Formatted I/O
- `io` - I/O primitives
- `os` - Operating system functionality

Additional imports MAY be added by the user in their code.

#### 6.4.4 Example

```yaml
safe-inputs:
  calculate:
    description: "Perform calculations"
    inputs:
      a:
        type: number
        required: true
      b:
        type: number
        required: true
    go: |
      a := inputs["a"].(float64)
      b := inputs["b"].(float64)
      result := map[string]any{
          "sum": a + b,
          "product": a * b,
      }
      json.NewEncoder(os.Stdout).Encode(result)
    timeout: 30
```

---

## 7. Security Model

### 7.1 Secret Isolation

Implementations MUST:

1. **Explicit Access**: Only environment variables declared in `env:` are available to tools
2. **Secret Masking**: Secrets referenced via `${{ secrets.NAME }}` are masked in logs
3. **No Global Access**: Tools cannot access workflow secrets not explicitly declared
4. **Environment Isolation**: Each tool has isolated environment variable namespace

### 7.2 Process Isolation

Implementations MUST provide:

1. **Containerization**: Shell, Python, and Go tools execute in Docker containers
2. **Process Boundaries**: Each invocation is a separate process
3. **Resource Limits**: Containers enforce CPU, memory, and filesystem limits
4. **Network Restrictions**: Network access controlled by workflow configuration

JavaScript tools SHOULD provide:

1. **Module Isolation**: Tools execute in isolated module scope
2. **Limited Execution**: Use V8 isolates or similar for CPU/memory limits
3. **No Server Access**: Tools cannot access server internals or other tools

### 7.3 Input Sanitization

Implementations MUST:

1. Validate input types against schema before execution
2. Reject inputs that do not conform to schema
3. Prevent code injection via input validation
4. Apply length limits to string inputs (SHOULD be at least 10KB)

### 7.4 Output Sanitization

Implementations MUST:

1. Parse and validate JSON output from tools
2. Reject non-JSON output from Shell/Python/Go tools
3. Apply size limits to output (see Section 8)
4. Remove or mask any accidental secret exposure in output

### 7.5 Timeout Enforcement

Implementations MUST:

1. Enforce timeout for Shell/Python/Go tools
2. Terminate processes that exceed timeout
3. Send SIGTERM, wait for grace period (5 seconds), then SIGKILL
4. Return timeout error to agent
5. Clean up container resources after timeout

---

## 8. Large Output Handling

### 8.1 Size Threshold

When tool output exceeds 500 characters, implementations MUST:

1. Save complete output to a file
2. Generate unique filename in accessible location
3. Return metadata response instead of full content

### 8.2 Metadata Response Format

```json
{
  "content": {
    "type": "file",
    "path": "/tmp/tool-output-abc123.json",
    "size": 15234,
    "message": "Output too large (15234 bytes). Saved to file."
  },
  "preview": {
    "schema": {
      "type": "array",
      "items": { "type": "object" }
    },
    "first_item": { ... },
    "item_count": 42
  }
}
```

**Required Fields**:
- `content.type`: MUST be "file"
- `content.path`: File path accessible to agent
- `content.size`: File size in bytes
- `content.message`: Human-readable explanation

**Optional Fields**:
- `preview.schema`: JSON schema of content
- `preview.first_item`: First item in array/list
- `preview.item_count`: Number of items in collection

### 8.3 File Access

Implementations MUST:

1. Store output files in location accessible to agent
2. Use unique, non-predictable filenames
3. Clean up files after workflow completion
4. Enforce file size limits (SHOULD be at least 10MB)

---

## 9. Integration with MCP Gateway

### 9.1 Configuration Extension

Safe Inputs extends the MCP Gateway configuration format. During workflow compilation:

1. Safe Inputs tools are compiled into MCP server configuration
2. Configuration is passed to MCP Gateway as additional server
3. Gateway routes requests to Safe Inputs MCP server
4. Safe Inputs server handles tool execution

### 9.2 Gateway Communication

Safe Inputs MCP server MUST:

1. Expose HTTP endpoint for MCP communication
2. Accept JSON-RPC requests from gateway
3. Require authentication via Authorization header
4. Return JSON-RPC responses to gateway

### 9.3 Configuration Generation

At compilation time, Safe Inputs generates:

```json
{
  "mcpServers": {
    "safeinputs": {
      "type": "http",
      "url": "http://localhost:3000",
      "headers": {
        "Authorization": "generated-api-key"
      }
    }
  }
}
```

This configuration is merged with other MCP servers and passed to gateway.

### 9.4 Server Lifecycle

Safe Inputs MCP server:

1. **Startup**: Server starts during workflow initialization
2. **Tool Registration**: All tools are registered at startup
3. **Runtime**: Server accepts requests throughout workflow execution
4. **Shutdown**: Server terminates when workflow completes
5. **Cleanup**: All ephemeral resources are cleaned up

---

## 10. Compliance Testing

### 10.1 Test Suite Requirements

A conforming implementation MUST pass the following test categories:

#### 10.1.1 Configuration Tests

- **T-CFG-001**: Valid tool with JavaScript implementation
- **T-CFG-002**: Valid tool with Shell implementation
- **T-CFG-003**: Valid tool with Python implementation
- **T-CFG-004**: Valid tool with Go implementation
- **T-CFG-005**: Tool with all input parameter types
- **T-CFG-006**: Tool with environment variables
- **T-CFG-007**: Tool with custom timeout
- **T-CFG-008**: Reject tool without description
- **T-CFG-009**: Reject tool with multiple implementations
- **T-CFG-010**: Reject tool with invalid timeout

#### 10.1.2 Input Validation Tests

- **T-VAL-001**: Required parameter validation
- **T-VAL-002**: Optional parameter with default
- **T-VAL-003**: Enum constraint validation
- **T-VAL-004**: Type coercion (string to number)
- **T-VAL-005**: Invalid type rejection
- **T-VAL-006**: Missing required parameter error

#### 10.1.3 Execution Tests

- **T-EXE-001**: JavaScript tool successful execution
- **T-EXE-002**: Shell tool successful execution
- **T-EXE-003**: Python tool successful execution
- **T-EXE-004**: Go tool successful execution
- **T-EXE-005**: Tool with secret access
- **T-EXE-006**: Tool timeout enforcement
- **T-EXE-007**: Tool execution error handling
- **T-EXE-008**: Tool with JSON output parsing

#### 10.1.4 Security Tests

- **T-SEC-001**: Secret isolation verification
- **T-SEC-002**: Environment variable isolation
- **T-SEC-003**: Process isolation (Shell/Python/Go)
- **T-SEC-004**: Input sanitization
- **T-SEC-005**: Output sanitization
- **T-SEC-006**: Secret masking in logs
- **T-SEC-007**: Dependency installation security
- **T-SEC-008**: GitHub Actions global objects access control

#### 10.1.5 Large Output Tests

- **T-OUT-001**: Output under 500 characters (direct return)
- **T-OUT-002**: Output over 500 characters (file save)
- **T-OUT-003**: Metadata response format
- **T-OUT-004**: File accessibility to agent
- **T-OUT-005**: JSON schema preview generation

#### 10.1.6 Dependencies Tests

- **T-DEP-001**: npm dependency installation for JavaScript tools
- **T-DEP-002**: pip dependency installation for Python tools
- **T-DEP-003**: go get dependency installation for Go tools
- **T-DEP-004**: apt/yum dependency installation for shell tools
- **T-DEP-005**: Dependency caching behavior
- **T-DEP-006**: Dependency installation failure handling

#### 10.1.7 Integration Tests

- **T-INT-001**: MCP Gateway configuration generation
- **T-INT-002**: HTTP MCP server startup
- **T-INT-003**: Authentication with gateway
- **T-INT-004**: JSON-RPC request handling
- **T-INT-005**: Error response format

### 10.2 Compliance Checklist

| Requirement | Test ID | Level | Status |
|-------------|---------|-------|--------|
| JavaScript tools | T-CFG-001, T-EXE-001 | 1 | Required |
| Shell tools | T-CFG-002, T-EXE-002 | 2 | Standard |
| Python tools | T-CFG-003, T-EXE-003 | 2 | Standard |
| Go tools | T-CFG-004, T-EXE-004 | 3 | Complete |
| Input validation | T-VAL-* | 1 | Required |
| Secret isolation | T-SEC-001, T-SEC-002 | 1 | Required |
| Process isolation | T-SEC-003 | 2 | Standard |
| Timeout handling | T-EXE-006 | 2 | Standard |
| Large output handling | T-OUT-* | 3 | Complete |
| Dependencies support | T-DEP-* | 2 | Standard |
| GitHub Actions globals | T-SEC-008 | 1 | Required |
| MCP Gateway integration | T-INT-* | 1 | Required |

### 10.3 Test Execution

Implementations SHOULD provide:

1. Automated test runner for compliance suite
2. Test result reporting in standard format
3. Test fixtures for common scenarios
4. Integration test environment setup
5. Conformance report generation

---

## Appendices

### Appendix A: Complete Examples

#### A.1 JavaScript Tool with Secrets

```yaml
safe-inputs:
  fetch-api-data:
    description: "Fetch data from external API"
    inputs:
      endpoint:
        type: string
        required: true
        description: "API endpoint path"
      method:
        type: string
        default: "GET"
        enum: ["GET", "POST", "PUT", "DELETE"]
    script: |
      const apiKey = process.env.API_KEY;
      const baseUrl = process.env.API_BASE_URL;
      
      const response = await fetch(`${baseUrl}/${endpoint}`, {
        method,
        headers: {
          'Authorization': `Bearer ${apiKey}`,
          'Content-Type': 'application/json'
        }
      });
      
      if (!response.ok) {
        throw new Error(`API request failed: ${response.status}`);
      }
      
      return await response.json();
    env:
      API_KEY: "${{ secrets.SERVICE_API_KEY }}"
      API_BASE_URL: "https://api.example.com"
```

#### A.2 Shell Tool with GitHub CLI

```yaml
safe-inputs:
  list-issues:
    description: "List GitHub issues using gh CLI"
    inputs:
      repo:
        type: string
        required: true
        description: "Repository in owner/name format"
      state:
        type: string
        default: "open"
        enum: ["open", "closed", "all"]
      limit:
        type: number
        default: 30
        description: "Maximum number of issues to return"
    run: |
      #!/bin/bash
      set -euo pipefail
      
      gh issue list \
        --repo "$INPUT_REPO" \
        --state "$INPUT_STATE" \
        --limit "$INPUT_LIMIT" \
        --json number,title,state,createdAt,author
    env:
      GH_TOKEN: "${{ secrets.GITHUB_TOKEN }}"
    timeout: 60
```

#### A.3 Python Tool with Data Processing

```yaml
safe-inputs:
  process-metrics:
    description: "Process and aggregate metric data"
    inputs:
      data:
        type: string
        required: true
        description: "JSON array of metric objects"
      group_by:
        type: string
        default: "category"
        description: "Field to group by"
    py: |
      import json
      from collections import defaultdict
      
      # Parse input data
      data_str = inputs.get('data', '[]')
      data = json.loads(data_str)
      group_by = inputs.get('group_by', 'category')
      
      # Group and aggregate
      groups = defaultdict(list)
      for item in data:
        key = item.get(group_by, 'unknown')
        groups[key].append(item)
      
      # Calculate statistics
      result = {}
      for key, items in groups.items():
        values = [item.get('value', 0) for item in items]
        result[key] = {
          'count': len(items),
          'sum': sum(values),
          'avg': sum(values) / len(values) if values else 0,
          'min': min(values) if values else 0,
          'max': max(values) if values else 0
        }
      
      print(json.dumps(result))
    timeout: 120
```

#### A.4 Go Tool with HTTP Request

```yaml
safe-inputs:
  health-check:
    description: "Check health of multiple endpoints"
    inputs:
      urls:
        type: string
        required: true
        description: "Comma-separated list of URLs"
      timeout_seconds:
        type: number
        default: 10
        description: "Timeout for each request"
    go: |
      import (
        "context"
        "net/http"
        "strings"
        "time"
      )
      
      // Parse URLs
      urlsStr := inputs["urls"].(string)
      urls := strings.Split(urlsStr, ",")
      timeoutSecs := int(inputs["timeout_seconds"].(float64))
      
      // Check each URL
      results := make(map[string]any)
      client := &http.Client{
        Timeout: time.Duration(timeoutSecs) * time.Second,
      }
      
      for _, url := range urls {
        url = strings.TrimSpace(url)
        start := time.Now()
        
        resp, err := client.Get(url)
        duration := time.Since(start).Milliseconds()
        
        if err != nil {
          results[url] = map[string]any{
            "status": "error",
            "error": err.Error(),
            "duration_ms": duration,
          }
          continue
        }
        
        results[url] = map[string]any{
          "status": "success",
          "status_code": resp.StatusCode,
          "duration_ms": duration,
        }
        resp.Body.Close()
      }
      
      json.NewEncoder(os.Stdout).Encode(results)
    timeout: 60
```

#### A.5 Complete Tool with Dependencies and GitHub Integration

```yaml
safe-inputs:
  analyze-pr-complexity:
    description: "Analyze pull request complexity and provide metrics"
    inputs:
      pr_number:
        type: number
        required: true
        description: "Pull request number to analyze"
      include_files:
        type: boolean
        default: true
        description: "Include per-file analysis"
    script: |
      const octokit = github.getOctokit(process.env.GITHUB_TOKEN);
      const { owner, repo } = context.repo;
      
      // Fetch PR data
      const { data: pr } = await octokit.rest.pulls.get({
        owner,
        repo,
        pull_number: pr_number
      });
      
      // Fetch PR files
      const { data: files } = await octokit.rest.pulls.listFiles({
        owner,
        repo,
        pull_number: pr_number
      });
      
      // Calculate complexity metrics
      const metrics = {
        pr_number: pr_number,
        title: pr.title,
        author: pr.user.login,
        files_changed: files.length,
        total_additions: files.reduce((sum, f) => sum + f.additions, 0),
        total_deletions: files.reduce((sum, f) => sum + f.deletions, 0),
        total_changes: files.reduce((sum, f) => sum + f.changes, 0),
        complexity_score: 0
      };
      
      // Calculate complexity score (simple heuristic)
      metrics.complexity_score = 
        (metrics.files_changed * 2) + 
        (metrics.total_changes / 10);
      
      // Add per-file analysis if requested
      if (include_files) {
        metrics.file_analysis = files.map(f => ({
          filename: f.filename,
          status: f.status,
          additions: f.additions,
          deletions: f.deletions,
          changes: f.changes
        }));
      }
      
      return metrics;
    env:
      GITHUB_TOKEN: "${{ secrets.GITHUB_TOKEN }}"
    timeout: 120
```

### Appendix B: Error Response Examples

#### B.1 Missing Required Parameter

```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32602,
    "message": "Invalid params",
    "data": {
      "missing": ["name"],
      "provided": ["age"],
      "schema": {
        "name": { "type": "string", "required": true },
        "age": { "type": "number", "required": false }
      }
    }
  },
  "id": "req-123"
}
```

#### B.2 Tool Execution Timeout

```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32603,
    "message": "Internal error",
    "data": {
      "error": "Tool execution timeout",
      "timeout_seconds": 60,
      "tool": "process-data"
    }
  },
  "id": "req-456"
}
```

#### B.3 Invalid Tool Output

```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32603,
    "message": "Internal error",
    "data": {
      "error": "Tool output is not valid JSON",
      "stdout": "Error: Cannot parse data\n",
      "stderr": "SyntaxError at line 42"
    }
  },
  "id": "req-789"
}
```

#### B.4 Dependency Installation Failure

```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32603,
    "message": "Internal error",
    "data": {
      "error": "Dependency installation failed",
      "dependency": "requests",
      "package_manager": "pip",
      "stderr": "Could not find a version that satisfies the requirement requests"
    }
  },
  "id": "req-101"
}
```

### Appendix C: Security Considerations

#### C.1 Secret Management

- Secrets MUST be explicitly declared in `env:` section
- Secrets SHOULD use GitHub Actions secret syntax (`${{ secrets.NAME }}`)
- Secrets MUST be masked in all logs and output
- Tools SHOULD NOT log or return secrets in output
- Secret rotation SHOULD be handled at workflow level

#### C.2 Input Validation

- All input parameters MUST be validated against schema
- String inputs SHOULD have length limits
- Numeric inputs SHOULD have range limits
- Input sanitization MUST prevent code injection
- Malicious input MUST NOT compromise host system

#### C.3 Container Security

- Shell/Python/Go tools SHOULD run in minimal containers
- Containers SHOULD have read-only root filesystem where possible
- Resource limits SHOULD be enforced (CPU, memory, disk)
- Network access SHOULD be restricted by default
- Container images SHOULD be verified and signed

#### C.4 Output Security

- Large outputs SHOULD be saved to temporary files
- Temporary files SHOULD be cleaned up after workflow
- Output SHOULD NOT contain secrets or sensitive data
- Output size SHOULD be limited to prevent DoS
- File paths SHOULD be non-predictable

#### C.5 Dependency Security

- Implementations SHOULD validate package names against known malicious packages
- Dependency sources SHOULD be from trusted registries (npm, PyPI, Go modules)
- Implementations MAY enforce allowlists for permitted packages
- Dependency versions SHOULD be pinned when possible
- Security advisories SHOULD be checked for known vulnerabilities

#### C.6 GitHub Actions Integration Security

- Global objects MUST be provided in sandboxed environment
- Token access MUST be controlled through explicit env declarations
- API rate limits SHOULD be enforced to prevent abuse
- Actions permissions SHOULD follow least privilege principle
- Audit logging SHOULD track all GitHub API operations

---

## References

### Normative References

- **[RFC 2119]** Key words for use in RFCs to Indicate Requirement Levels
- **[JSON-RPC 2.0]** JSON-RPC 2.0 Specification
- **[MCP]** Model Context Protocol Specification
- **[JSON Schema]** JSON Schema Specification (Draft 7)

### Informative References

- **[MCP Gateway Specification]** GitHub Agentic Workflows MCP Gateway Specification
- **[GitHub Actions]** GitHub Actions Workflow Syntax
- **[Docker]** Docker Container Runtime

---

## Change Log

### Version 1.1.0 (Draft)

- **Added**: Dependencies support (Section 4.3)
  - `dependencies` field for specifying runtime package dependencies
  - Package manager inferred from implementation language (npm, pip, go get, apt/yum)
  - Dependencies installed before tool execution
  - Examples added for Python requests and shell jq dependencies
- **Added**: GitHub Actions global objects for JavaScript tools (Section 6.1.2)
  - Global `github`, `context`, `core`, `io`, `exec`, `glob`, `artifact` objects
  - Available without explicit `require()` statements
  - No restrictions on execution location (in-process or containerized)
  - Example demonstrating GitHub API usage via global objects
- **Updated**: Section numbering to accommodate new sections

### Version 1.0.0 (Draft)

- Initial specification release
- Configuration format definition
- Language support (JavaScript, Shell, Python, Go)
- Security model specification
- Large output handling
- MCP Gateway integration
- Compliance test framework

---

*Copyright © 2026 GitHub, Inc. All rights reserved.*
