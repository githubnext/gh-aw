---
on:
  workflow_dispatch:
  issues:
    types: [opened, labeled]
  pull_request:
    types: [opened, labeled]

safe-outputs:
  add-issue-label:
    allowed: [test-safe-output, automation, bug, enhancement, documentation, question]
    max: 3
  staged: true

engine:
  id: custom
  steps:
    - name: Generate Add Issue Label Safe Output
      uses: actions/github-script@v7
      env:
          GITHUB_AW_SAFE_OUTPUTS_TOOL_CALLS: "{\"type\": \"add-issue-label\", \"labels\": [\"test-safe-output\", \"automation\"]}"
      with:
        script: |
            const { spawn } = require("child_process");
            const path = require("path");
            const serverPath = path.join("/tmp/safe-outputs/mcp-server.cjs");
            const { GITHUB_AW_SAFE_OUTPUTS_TOOL_CALLS } = process.env;
            function parseJsonl(input) {
                if (!input) return [];
                return input
                    .split(/\r?\n/)
                    .map((l) => l.trim())
                    .filter(Boolean)
                    .map((line) => JSON.parse(line));
            }
            const toolCalls = parseJsonl(GITHUB_AW_SAFE_OUTPUTS_TOOL_CALLS)
            const child = spawn(process.execPath, [serverPath], {
                stdio: ["pipe", "pipe", "pipe"],
                env: process.env,
            });
            let stdoutBuffer = Buffer.alloc(0);
            const pending = new Map();
            let nextId = 1;
            function writeMessage(obj) {
                const json = JSON.stringify(obj);
                const header = `Content-Length: ${Buffer.byteLength(json)}\r\n\r\n`;
                child.stdin.write(header + json);
            }
            function sendRequest(method, params) {
                const id = nextId++;
                const req = { jsonrpc: "2.0", id, method, params };
                return new Promise((resolve, reject) => {
                    pending.set(id, { resolve, reject });
                    writeMessage(req);
                    // simple timeout
                    const to = setTimeout(() => {
                        if (pending.has(id)) {
                            pending.delete(id);
                            reject(new Error(`Request timed out: ${method}`));
                        }
                    }, 5000);
                    // wrap resolve to clear timeout
                    const origResolve = resolve;
                    resolve = (value) => {
                        clearTimeout(to);
                        origResolve(value);
                    };
                });
            }

            function handleMessage(msg) {
                if (msg.method && !msg.id) {
                    console.error("<- notification", msg.method, msg.params || "");
                    return;
                }
                if (msg.id !== undefined && (msg.result !== undefined || msg.error !== undefined)) {
                    const waiter = pending.get(msg.id);
                    if (waiter) {
                        pending.delete(msg.id);
                        if (msg.error) waiter.reject(new Error(msg.error.message || JSON.stringify(msg.error)));
                        else waiter.resolve(msg.result);
                    } else {
                        console.error("<- response with unknown id", msg.id);
                    }
                    return;
                }
                console.error("<- unexpected message", msg);
            }

            child.stdout.on("data", (chunk) => {
                stdoutBuffer = Buffer.concat([stdoutBuffer, chunk]);
                while (true) {
                    const sep = stdoutBuffer.indexOf("\r\n\r\n");
                    if (sep === -1) break;
                    const header = stdoutBuffer.slice(0, sep).toString("utf8");
                    const match = header.match(/Content-Length:\s*(\d+)/i);
                    if (!match) {
                        // Remove header and continue
                        stdoutBuffer = stdoutBuffer.slice(sep + 4);
                        continue;
                    }
                    const length = parseInt(match[1], 10);
                    const total = sep + 4 + length;
                    if (stdoutBuffer.length < total) break; // wait for full message
                    const body = stdoutBuffer.slice(sep + 4, total).toString("utf8");
                    stdoutBuffer = stdoutBuffer.slice(total);

                    let parsed = null;
                    try {
                        parsed = JSON.parse(body);
                    } catch (e) {
                        console.error("Failed to parse server message", e);
                        continue;
                    }
                    handleMessage(parsed);
                }
            });
            child.stderr.on("data", (d) => {
                process.stderr.write("[server] " + d.toString());
            });
            child.on("exit", (code, sig) => {
                console.error("server exited", code, sig);
            });

            (async () => {
                try {
                    console.error("Starting MCP client -> spawning server at", serverPath);
                    const init = await sendRequest("initialize", {
                        clientInfo: { name: "mcp-stdio-client", version: "0.1.0" },
                        protocolVersion: "2024-11-05",
                    });
                    console.error("initialize ->", init);
                    const toolsList = await sendRequest("tools/list", {});
                    console.error("tools/list ->", toolsList);
                    for (const toolCall of toolCalls) {
                        const { type, ...args } = toolCall;
                        console.error("Calling tool:", type, args);
                        try {
                            const res = await sendRequest("tools/call", { name: type, arguments: args });
                            console.error("tools/call ->", res);
                        } catch (err) {
                            console.error("tools/call error for", type, err);
                        }
                    }

                    // Clean up: give server a moment to flush, then exit
                    setTimeout(() => {
                        try {
                            child.kill();
                        } catch (e) { }
                        process.exit(0);
                    }, 200);
                } catch (e) {
                    console.error("Error in MCP client:", e);
                    try {
                        child.kill();
                    } catch (e) { }
                    process.exit(1);
                }
            })();
        
    - name: Verify Safe Output File
      run: |
        echo "Generated safe output entries:"
        if [ -f "$GITHUB_AW_SAFE_OUTPUTS" ]; then
          cat "$GITHUB_AW_SAFE_OUTPUTS"
        else
          echo "No safe outputs file found"
        fi

permissions: read-all
---

# Test Safe Output - Add Issue Label

This workflow tests the `add-issue-label` safe output functionality using a custom engine that directly writes to the safe output file.

## Purpose

This workflow validates the add-issue-label safe output type by:
- Generating a JSON entry with the `add-issue-label` type
- Including the required labels array
- Using staged mode to prevent actual GitHub interactions
- Demonstrating custom engine safe output writing for label addition

## Trigger Events

- **workflow_dispatch**: Manual execution for testing
- **issues.opened**: Responds to new issues being created
- **issues.labeled**: Responds to issues being labeled
- **pull_request.opened**: Responds to new pull requests being created
- **pull_request.labeled**: Responds to pull requests being labeled

## Safe Output Configuration

- **staged: true**: Prevents real GitHub interactions
- **allowed**: Restricts labels to a predefined allowlist for security
- **max: 3**: Limits to three labels per workflow run

## Custom Engine Implementation

The workflow uses a custom engine with GitHub Actions steps to:
1. Generate the appropriate add-issue-label JSON output
2. Include labels that are within the allowed list
3. Append it to the $GITHUB_AW_SAFE_OUTPUTS file
4. Verify the output was generated correctly

This demonstrates how custom engines can leverage the safe output system for adding labels to issues and pull requests while respecting security constraints.