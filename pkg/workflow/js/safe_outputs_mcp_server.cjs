const fs = require("fs");
const path = require("path");
const encoder = new TextEncoder();
const decoder = new TextDecoder();

function writeMessage(obj) {
  const json = JSON.stringify(obj);
  const bytes = encoder.encode(json);
  const header = `Content-Length: ${bytes.byteLength}\r\n\r\n`;
  const headerBytes = encoder.encode(header);
  fs.writeSync(1, headerBytes);
  fs.writeSync(1, bytes);
}

let buffer = Buffer.alloc(0);
function onData(chunk) {
  buffer = Buffer.concat([buffer, chunk]);
  while (true) {
    const sep = buffer.indexOf("\r\n\r\n");
    if (sep === -1) break;

    const headerPart = buffer.slice(0, sep).toString("utf8");
    const match = headerPart.match(/Content-Length:\s*(\d+)/i);
    if (!match) {
      buffer = buffer.slice(sep + 4);
      continue;
    }
    const length = parseInt(match[1], 10);
    const total = sep + 4 + length;
    if (buffer.length < total) break; // wait for full body

    const body = buffer.slice(sep + 4, total);
    buffer = buffer.slice(total);

    try {
      const msg = JSON.parse(body.toString("utf8"));
      handleMessage(msg);
    } catch (e) {
      const err = {
        jsonrpc: "2.0",
        id: null,
        error: { code: -32700, message: "Parse error", data: String(e) },
      };
      writeMessage(err);
    }
  }
}

process.stdin.on("data", onData);
process.stdin.on("error", () => { });
process.stdin.resume();

function replyResult(id, result) {
  if (id === undefined || id === null) return; // notification
  const res = { jsonrpc: "2.0", id, result };
  writeMessage(res);
}
function replyError(id, code, message, data) {
  const res = {
    jsonrpc: "2.0",
    id: id ?? null,
    error: { code, message, data },
  };
  writeMessage(res);
}

let safeOutputsConfig = {};
let outputFile = null;
function initializeSafeOutputsConfig() {
  const configEnv = process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG;
  if (configEnv) {
    try {
      safeOutputsConfig = JSON.parse(configEnv);
    } catch (e) {
      process.stderr.write(
        `[safe-outputs] Error parsing config: ${e instanceof Error ? e.message : String(e)}\n`
      );
      safeOutputsConfig = {};
    }
  }
  outputFile = process.env.GITHUB_AW_SAFE_OUTPUTS;
  if (!outputFile) {
    process.stderr.write(
      `[safe-outputs-mcp] Warning: GITHUB_AW_SAFE_OUTPUTS not set\n`
    );
  }
}

// Check if a safe-output type is enabled
function isToolEnabled(toolType) {
  return safeOutputsConfig[toolType] && safeOutputsConfig[toolType].enabled;
}

// Get max limit for a tool type
function getToolMaxLimit(toolType) {
  const config = safeOutputsConfig[toolType];
  return config && config.max ? config.max : 0; // 0 means unlimited
}

// Append safe output entry to file
function appendSafeOutput(entry) {
  if (!outputFile) {
    throw new Error("No output file configured");
  }
  const jsonLine = JSON.stringify(entry) + "\n";
  try {
    fs.appendFileSync(outputFile, jsonLine);
  } catch (error) {
    throw new Error(
      `Failed to write to output file: ${error instanceof Error ? error.message : String(error)}`
    );
  }
}

const defaultHandler = (type) => async (args) => {
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
}
const TOOLS = Object.fromEntries([{
  name: "create_issue",
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
  }
}, {
  name: "create_discussion",
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
}, {
  name: "add_issue_comment",
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
}, {
  name: "create_pull_request",
  description: "Create a new GitHub pull request",
  inputSchema: {
    type: "object",
    required: ["title", "body"],
    properties: {
      title: { type: "string", description: "Pull request title" },
      body: { type: "string", description: "Pull request body/description" },
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
}, {
  name: "create_pull_request_review_comment",
  description: "Create a review comment on a GitHub pull request",
  inputSchema: {
    type: "object",
    required: ["path", "line", "body"],
    properties: {
      path: { type: "string", description: "File path for the review comment" },
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
}, {
  name: "create_code_scanning_alert",
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
}, {
  name: "add_issue_label",
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
}, {
  name: "update_issue",
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
}, {
  name: "push_to_pr_branch",
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
}, {
  name: "missing_tool",
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
}].filter(({ name }) => isToolEnabled(name)).map(tool => [tool.name, tool]));
const SERVER_INFO = { name: "gh-aw-safe-outputs", version: "1.0.0" };

function handleMessage(req) {
  const { id, method, params } = req;

  try {
    if (method === "initialize") {
      initializeSafeOutputsConfig();
      const clientInfo = params?.clientInfo ?? {};
      const protocolVersion = params?.protocolVersion ?? undefined;
      const result = {
        serverInfo: SERVER_INFO,
        ...(protocolVersion ? { protocolVersion } : {}),
        capabilities: {
          tools: {},
        },
      };
      replyResult(id, result);
    }
    else if (method === "tools/list") {
      const list = [];
      Object.values(TOOLS).forEach(tool => {
        const toolType = tool.name.replace(/_/g, "-"); // Convert to kebab-case
        if (isToolEnabled(toolType)) {
          list.push({
            name: tool.name,
            description: tool.description,
            inputSchema: tool.inputSchema,
          });
        }
      });
      replyResult(id, { tools: list });
    }
    else if (method === "tools/call") {
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
      (async () => {
        try {
          const result = await handler(args);
          replyResult(id, { content: result.content });
        } catch (e) {
          replyError(id, -32000, `Tool '${name}' failed`, {
            message: e instanceof Error ? e.message : String(e),
          });
        }
      })();
      return;
    }
    replyError(id, -32601, `Method not found: ${method}`);
  } catch (e) {
    replyError(id, -32603, "Internal error", {
      message: e instanceof Error ? e.message : String(e),
    });
  }
}
process.stderr.write(
  `[${SERVER_INFO.name}] v${SERVER_INFO.version} ready on stdio\n  tools: ${Object.keys(TOOLS).join(", ")}\n`
);
