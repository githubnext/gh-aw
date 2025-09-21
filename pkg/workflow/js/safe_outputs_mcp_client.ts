import type { SafeOutputItems } from "./types/safe-outputs";

interface ToolCall {
  type: string;
  [key: string]: any;
}

interface JsonRpcRequest {
  jsonrpc: string;
  id: number;
  method: string;
  params?: any;
}

interface JsonRpcResponse {
  id?: number;
  result?: any;
  error?: {
    message?: string;
  };
  method?: string;
  params?: any;
}

interface PendingRequest {
  resolve: (value: any) => void;
  reject: (reason: any) => void;
}

async function safeOutputsMcpClientMain(): Promise<void> {
  const { spawn } = require("child_process");
  const path = require("path");
  const { Buffer } = require("buffer");
  const { setTimeout, clearTimeout } = require("timers");

  const serverPath = path.join("/tmp/safe-outputs/mcp-server.cjs");
  const { GITHUB_AW_SAFE_OUTPUTS_TOOL_CALLS } = process.env;

  function parseJsonl(input: string | undefined): ToolCall[] {
    if (!input) return [];
    return input
      .split(/\r?\n/)
      .map(l => l.trim())
      .filter(Boolean)
      .map(line => JSON.parse(line));
  }

  const toolCalls = parseJsonl(GITHUB_AW_SAFE_OUTPUTS_TOOL_CALLS);

  const child = spawn(process.execPath, [serverPath], {
    stdio: ["pipe", "pipe", "pipe"],
    env: process.env,
  });

  let stdoutBuffer: any = Buffer.alloc(0);
  const pending = new Map<number, PendingRequest>();
  let nextId = 1;

  function writeMessage(obj: JsonRpcRequest): void {
    const json = JSON.stringify(obj);
    const message = json + "\n";
    child.stdin.write(message);
  }

  function sendRequest(method: string, params?: any): Promise<any> {
    const id = nextId++;
    const req: JsonRpcRequest = { jsonrpc: "2.0", id, method, params };
    return new Promise((resolve, reject) => {
      pending.set(id, { resolve, reject });
      writeMessage(req);
      // simple timeout
      const to: any = setTimeout(() => {
        if (pending.has(id)) {
          pending.delete(id);
          reject(new Error(`Request timed out: ${method}`));
        }
      }, 5000);
      // wrap resolve to clear timeout
      const origResolve = resolve;
      resolve = (value: any) => {
        clearTimeout(to);
        origResolve(value);
      };
    });
  }

  function handleMessage(msg: JsonRpcResponse): void {
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

  child.stdout.on("data", (chunk: any) => {
    stdoutBuffer = Buffer.concat([stdoutBuffer, chunk]);
    while (true) {
      const newlineIndex = stdoutBuffer.indexOf("\n");
      if (newlineIndex === -1) break;

      const line = stdoutBuffer.slice(0, newlineIndex).toString("utf8").replace(/\r$/, "");
      stdoutBuffer = stdoutBuffer.slice(newlineIndex + 1);

      if (line.trim() === "") continue; // Skip empty lines

      let parsed: any = null;
      try {
        parsed = JSON.parse(line);
      } catch (e) {
        console.error("Failed to parse server message", e);
        continue;
      }
      if (parsed) {
        handleMessage(parsed);
      }
    }
  });

  child.stderr.on("data", (d: any) => {
    process.stderr.write("[server] " + d.toString());
  });

  child.on("exit", (code: number | null, sig: any) => {
    console.error("server exited", code, sig);
  });

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
        const res = await sendRequest("tools/call", {
          name: type,
          arguments: args,
        });
        console.error("tools/call ->", res);
      } catch (err) {
        console.error("tools/call error for", type, err);
      }
    }

    // Clean up: give server a moment to flush, then exit
    setTimeout(() => {
      try {
        child.kill();
      } catch (e) {}
      process.exit(0);
    }, 200);
  } catch (e) {
    console.error("Error in MCP client:", e);
    try {
      child.kill();
    } catch (e) {}
    process.exit(1);
  }
}

(async () => {
  await safeOutputsMcpClientMain();
})();
