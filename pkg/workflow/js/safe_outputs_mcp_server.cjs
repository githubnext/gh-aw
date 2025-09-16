const fs = require("fs");
const encoder = new TextEncoder();
const configEnv = process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG;
if (!configEnv) throw new Error("GITHUB_AW_SAFE_OUTPUTS_CONFIG not set");
const safeOutputsConfig = JSON.parse(configEnv);
const outputFile = process.env.GITHUB_AW_SAFE_OUTPUTS;
if (!outputFile)
  throw new Error("GITHUB_AW_SAFE_OUTPUTS not set, no output file");
const SERVER_INFO = { name: "safe-outputs-mcp-server", version: "1.0.0" };
const debug = msg => process.stderr.write(`[${SERVER_INFO.name}] ${msg}\n`);
function writeMessage(obj) {
  const json = JSON.stringify(obj);
  debug(`send: ${json}`);
  const message = json + "\n";
  const bytes = encoder.encode(message);
  fs.writeSync(1, bytes);
}

class ReadBuffer {
  append(chunk) {
    this._buffer = this._buffer ? Buffer.concat([this._buffer, chunk]) : chunk;
  }

  readMessage() {
    if (!this._buffer) {
      return null;
    }

    const index = this._buffer.indexOf("\n");
    if (index === -1) {
      return null;
    }

    const line = this._buffer.toString("utf8", 0, index).replace(/\r$/, "");
    this._buffer = this._buffer.subarray(index + 1);

    if (line.trim() === "") {
      return this.readMessage(); // Skip empty lines recursively
    }

    try {
      return JSON.parse(line);
    } catch (error) {
      throw new Error(
        `Parse error: ${error instanceof Error ? error.message : String(error)}`
      );
    }
  }
}

const readBuffer = new ReadBuffer();

function onData(chunk) {
  readBuffer.append(chunk);
  processReadBuffer();
}

function processReadBuffer() {
  while (true) {
    try {
      const message = readBuffer.readMessage();
      if (!message) {
        break;
      }
      debug(`recv: ${JSON.stringify(message)}`);
      handleMessage(message);
    } catch (error) {
      // For parse errors, we can't know the request id, so we shouldn't send a response
      // according to JSON-RPC spec. Just log the error.
      debug(
        `Parse error: ${error instanceof Error ? error.message : String(error)}`
      );
    }
  }
}

function replyResult(id, result) {
  if (id === undefined || id === null) return; // notification
  const res = { jsonrpc: "2.0", id, result };
  writeMessage(res);
}
function replyError(id, code, message, data) {
  // Don't send error responses for notifications (id is null/undefined)
  if (id === undefined || id === null) {
    debug(`Error for notification: ${message}`);
    return;
  }

  const error = { code, message };
  if (data !== undefined) {
    error.data = data;
  }
  const res = {
    jsonrpc: "2.0",
    id,
    error,
  };
  writeMessage(res);
}

function isToolEnabled(name) {
  return safeOutputsConfig[name];
}

function appendSafeOutput(entry) {
  if (!outputFile) throw new Error("No output file configured");
  const jsonLine = JSON.stringify(entry) + "\n";
  try {
    fs.appendFileSync(outputFile, jsonLine);
  } catch (error) {
    throw new Error(
      `Failed to write to output file: ${error instanceof Error ? error.message : String(error)}`
    );
  }
}

const defaultHandler = type => args => {
  const entry = { ...(args || {}), type };
  appendSafeOutput(entry);
  return {
    content: [
      {
        type: "text",
        text: `success`,
      },
    ],
  };
};

const TOOLS = Object.fromEntries(
  [
    {
      name: "create-issue",
      description: "Create a new GitHub issue",
      inputSchema: {
        type: "object",
        required: ["title", "body"],
        properties: {
          title: { type: "string", description: "Issue title" },
          body: { type: "string", description: "Issue body/description" },
          labels: {
            type: "array",
            items: { type: "string" },
            description: "Issue labels",
          },
        },
        additionalProperties: false,
      },
    },
    {
      name: "create-discussion",
      description: "Create a new GitHub discussion",
      inputSchema: {
        type: "object",
        required: ["title", "body"],
        properties: {
          title: { type: "string", description: "Discussion title" },
          body: { type: "string", description: "Discussion body/content" },
          category: { type: "string", description: "Discussion category" },
        },
        additionalProperties: false,
      },
    },
    {
      name: "add-issue-comment",
      description: "Add a comment to a GitHub issue or pull request",
      inputSchema: {
        type: "object",
        required: ["body"],
        properties: {
          body: { type: "string", description: "Comment body/content" },
          issue_number: {
            type: "number",
            description: "Issue or PR number (optional for current context)",
          },
        },
        additionalProperties: false,
      },
    },
    {
      name: "create-pull-request",
      description: "Create a new GitHub pull request",
      inputSchema: {
        type: "object",
        required: ["title", "body"],
        properties: {
          title: { type: "string", description: "Pull request title" },
          body: {
            type: "string",
            description: "Pull request body/description",
          },
          branch: {
            type: "string",
            description:
              "Optional branch name (will be auto-generated if not provided)",
          },
          labels: {
            type: "array",
            items: { type: "string" },
            description: "Optional labels to add to the PR",
          },
        },
        additionalProperties: false,
      },
    },
    {
      name: "create-pull-request-review-comment",
      description: "Create a review comment on a GitHub pull request",
      inputSchema: {
        type: "object",
        required: ["path", "line", "body"],
        properties: {
          path: {
            type: "string",
            description: "File path for the review comment",
          },
          line: {
            type: ["number", "string"],
            description: "Line number for the comment",
          },
          body: { type: "string", description: "Comment body content" },
          start_line: {
            type: ["number", "string"],
            description: "Optional start line for multi-line comments",
          },
          side: {
            type: "string",
            enum: ["LEFT", "RIGHT"],
            description: "Optional side of the diff: LEFT or RIGHT",
          },
        },
        additionalProperties: false,
      },
    },
    {
      name: "create-code-scanning-alert",
      description: "Create a code scanning alert",
      inputSchema: {
        type: "object",
        required: ["file", "line", "severity", "message"],
        properties: {
          file: {
            type: "string",
            description: "File path where the issue was found",
          },
          line: {
            type: ["number", "string"],
            description: "Line number where the issue was found",
          },
          severity: {
            type: "string",
            enum: ["error", "warning", "info", "note"],
            description: "Severity level",
          },
          message: {
            type: "string",
            description: "Alert message describing the issue",
          },
          column: {
            type: ["number", "string"],
            description: "Optional column number",
          },
          ruleIdSuffix: {
            type: "string",
            description: "Optional rule ID suffix for uniqueness",
          },
        },
        additionalProperties: false,
      },
    },
    {
      name: "add-issue-label",
      description: "Add labels to a GitHub issue or pull request",
      inputSchema: {
        type: "object",
        required: ["labels"],
        properties: {
          labels: {
            type: "array",
            items: { type: "string" },
            description: "Labels to add",
          },
          issue_number: {
            type: "number",
            description: "Issue or PR number (optional for current context)",
          },
        },
        additionalProperties: false,
      },
    },
    {
      name: "update-issue",
      description: "Update a GitHub issue",
      inputSchema: {
        type: "object",
        properties: {
          status: {
            type: "string",
            enum: ["open", "closed"],
            description: "Optional new issue status",
          },
          title: { type: "string", description: "Optional new issue title" },
          body: { type: "string", description: "Optional new issue body" },
          issue_number: {
            type: ["number", "string"],
            description: "Optional issue number for target '*'",
          },
        },
        additionalProperties: false,
      },
    },
    {
      name: "push-to-pr-branch",
      description: "Push changes to a pull request branch",
      inputSchema: {
        type: "object",
        properties: {
          message: { type: "string", description: "Optional commit message" },
          pull_request_number: {
            type: ["number", "string"],
            description: "Optional pull request number for target '*'",
          },
        },
        additionalProperties: false,
      },
    },
    {
      name: "missing-tool",
      description:
        "Report a missing tool or functionality needed to complete tasks",
      inputSchema: {
        type: "object",
        required: ["tool", "reason"],
        properties: {
          tool: { type: "string", description: "Name of the missing tool" },
          reason: { type: "string", description: "Why this tool is needed" },
          alternatives: {
            type: "string",
            description: "Possible alternatives or workarounds",
          },
        },
        additionalProperties: false,
      },
    },
  ]
    .filter(({ name }) => isToolEnabled(name))
    .map(tool => [tool.name, tool])
);

debug(`v${SERVER_INFO.version} ready on stdio`);
debug(`  output file: ${outputFile}`);
debug(`  config: ${JSON.stringify(safeOutputsConfig)}`);
debug(`  tools: ${Object.keys(TOOLS).join(", ")}`);
if (!Object.keys(TOOLS).length)
  throw new Error("No tools enabled in configuration");

function handleMessage(req) {
  // Validate basic JSON-RPC structure
  if (!req || typeof req !== "object") {
    debug(`Invalid message: not an object`);
    return;
  }

  if (req.jsonrpc !== "2.0") {
    debug(`Invalid message: missing or invalid jsonrpc field`);
    return;
  }

  const { id, method, params } = req;

  // Validate method field
  if (!method || typeof method !== "string") {
    replyError(id, -32600, "Invalid Request: method must be a string");
    return;
  }

  try {
    if (method === "initialize") {
      const clientInfo = params?.clientInfo ?? {};
      console.error(`client initialized:`, clientInfo);
      const protocolVersion = params?.protocolVersion ?? undefined;
      const result = {
        serverInfo: SERVER_INFO,
        ...(protocolVersion ? { protocolVersion } : {}),
        capabilities: {
          tools: {},
        },
      };
      replyResult(id, result);
    } else if (method === "tools/list") {
      const list = [];
      Object.values(TOOLS).forEach(tool => {
        list.push({
          name: tool.name,
          description: tool.description,
          inputSchema: tool.inputSchema,
        });
      });
      replyResult(id, { tools: list });
    } else if (method === "tools/call") {
      const name = params?.name;
      const args = params?.arguments ?? {};
      if (!name || typeof name !== "string") {
        replyError(id, -32602, "Invalid params: 'name' must be a string");
        return;
      }
      const tool = TOOLS[name];
      if (!tool) {
        replyError(id, -32601, `Tool not found: ${name}`);
        return;
      }
      const handler = tool.handler || defaultHandler(tool.name);
      const requiredFields =
        tool.inputSchema && Array.isArray(tool.inputSchema.required)
          ? tool.inputSchema.required
          : [];
      if (requiredFields.length) {
        const missing = requiredFields.filter(f => args[f] === undefined);
        if (missing.length) {
          replyError(
            id,
            -32602,
            `Invalid arguments: missing ${missing.map(m => `'${m}'`).join(", ")}`
          );
          return;
        }
      }
      const result = handler(args);
      const content = result && result.content ? result.content : [];
      replyResult(id, { content });
    } else if (/^notifications\//.test(method)) {
      debug(`ignore ${method}`);
    } else {
      replyError(id, -32601, `Method not found: ${method}`);
    }
  } catch (e) {
    replyError(id, -32603, "Internal error", {
      message: e instanceof Error ? e.message : String(e),
    });
  }
}

process.stdin.on("data", onData);
process.stdin.on("error", err => debug(`stdin error: ${err}`));
process.stdin.resume();
debug(`listening...`);
