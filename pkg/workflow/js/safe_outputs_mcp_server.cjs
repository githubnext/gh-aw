/* Safe-Outputs MCP Tools-only server over stdio
   - No external deps (zero dependencies)
   - JSON-RPC 2.0 + Content-Length framing (LSP-style)
   - Implements: initialize, tools/list, tools/call
   - Each safe-output type is exposed as a tool
   - Tool calls append to GITHUB_AW_SAFE_OUTPUTS file
   - Controlled by GITHUB_AW_SAFE_OUTPUTS_CONFIG environment variable
   - Node 18+ recommended
*/

const fs = require("fs");
const path = require("path");

// --------- Basic types ---------
/* 
type JSONValue = null | boolean | number | string | JSONValue[] | { [k: string]: JSONValue };

type JSONRPCRequest = {
  jsonrpc: "2.0";
  id?: number | string;
  method: string;
  params?: any;
};

type JSONRPCResponse = {
  jsonrpc: "2.0";
  id: number | string | null;
  result?: any;
  error?: { code: number; message: string; data?: any };
};
*/

// --------- Basic message framing (Content-Length) ----------
const encoder = new TextEncoder();
const decoder = new TextDecoder();

function writeMessage(obj) {
  const json = JSON.stringify(obj);
  const bytes = encoder.encode(json);
  const header = `Content-Length: ${bytes.byteLength}\r\n\r\n`;
  const headerBytes = encoder.encode(header);
  // Write headers then body to stdout (synchronously to preserve order)
  fs.writeSync(1, headerBytes);
  fs.writeSync(1, bytes);
}

let buffer = Buffer.alloc(0);

function onData(chunk) {
  buffer = Buffer.concat([buffer, chunk]);

  // Parse multiple framed messages if present
  while (true) {
    const sep = buffer.indexOf("\r\n\r\n");
    if (sep === -1) break;

    const headerPart = buffer.slice(0, sep).toString("utf8");
    const match = headerPart.match(/Content-Length:\s*(\d+)/i);
    if (!match) {
      // Malformed header; drop this chunk
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
      // If we can't parse, there's no id to reply to reliably
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
process.stdin.on("error", () => {
  // Non-fatal
});
process.stdin.resume();

// ---------- Utilities ----------
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

// ---------- Safe-outputs configuration ----------
let safeOutputsConfig = {};
let outputFile = null;

// Parse configuration from environment
function initializeSafeOutputsConfig() {
  // Get safe-outputs configuration
  const configEnv = process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG;
  if (configEnv) {
    try {
      safeOutputsConfig = JSON.parse(configEnv);
    } catch (e) {
      // Log error to stderr (not part of protocol)
      process.stderr.write(
        `[safe-outputs-mcp] Error parsing config: ${e.message}\n`
      );
      safeOutputsConfig = {};
    }
  }

  // Get output file path
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

  // Ensure the entry is a complete JSON object
  const jsonLine = JSON.stringify(entry) + "\n";

  try {
    fs.appendFileSync(outputFile, jsonLine);
  } catch (error) {
    throw new Error(`Failed to write to output file: ${error.message}`);
  }
}

// ---------- Tool registry ----------
const TOOLS = Object.create(null);

// Create-issue tool
TOOLS["create_issue"] = {
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
  },
  async handler(args) {
    if (!isToolEnabled("create-issue")) {
      throw new Error("create-issue safe-output is not enabled");
    }

    const entry = {
      type: "create-issue",
      title: args.title,
      body: args.body,
    };

    if (args.labels && Array.isArray(args.labels)) {
      entry.labels = args.labels;
    }

    appendSafeOutput(entry);

    return {
      content: [
        {
          type: "text",
          text: `Issue creation queued: "${args.title}"`,
        },
      ],
    };
  },
};

// Create-discussion tool
TOOLS["create_discussion"] = {
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
  async handler(args) {
    if (!isToolEnabled("create-discussion")) {
      throw new Error("create-discussion safe-output is not enabled");
    }

    const entry = {
      type: "create-discussion",
      title: args.title,
      body: args.body,
    };

    if (args.category) {
      entry.category = args.category;
    }

    appendSafeOutput(entry);

    return {
      content: [
        {
          type: "text",
          text: `Discussion creation queued: "${args.title}"`,
        },
      ],
    };
  },
};

// Add-issue-comment tool
TOOLS["add_issue_comment"] = {
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
  async handler(args) {
    if (!isToolEnabled("add-issue-comment")) {
      throw new Error("add-issue-comment safe-output is not enabled");
    }

    const entry = {
      type: "add-issue-comment",
      body: args.body,
    };

    if (args.issue_number) {
      entry.issue_number = args.issue_number;
    }

    appendSafeOutput(entry);

    return {
      content: [
        {
          type: "text",
          text: "Comment creation queued",
        },
      ],
    };
  },
};

// Create-pull-request tool
TOOLS["create_pull_request"] = {
  name: "create_pull_request",
  description: "Create a new GitHub pull request",
  inputSchema: {
    type: "object",
    required: ["title", "body", "head", "base"],
    properties: {
      title: { type: "string", description: "Pull request title" },
      body: { type: "string", description: "Pull request body/description" },
      head: { type: "string", description: "Head branch name" },
      base: { type: "string", description: "Base branch name" },
      draft: { type: "boolean", description: "Create as draft PR" },
    },
    additionalProperties: false,
  },
  async handler(args) {
    if (!isToolEnabled("create-pull-request")) {
      throw new Error("create-pull-request safe-output is not enabled");
    }

    const entry = {
      type: "create-pull-request",
      title: args.title,
      body: args.body,
      head: args.head,
      base: args.base,
    };

    if (args.draft !== undefined) {
      entry.draft = args.draft;
    }

    appendSafeOutput(entry);

    return {
      content: [
        {
          type: "text",
          text: `Pull request creation queued: "${args.title}"`,
        },
      ],
    };
  },
};

// Create-pull-request-review-comment tool
TOOLS["create_pull_request_review_comment"] = {
  name: "create_pull_request_review_comment",
  description: "Create a review comment on a GitHub pull request",
  inputSchema: {
    type: "object",
    required: ["body"],
    properties: {
      body: { type: "string", description: "Review comment body" },
      path: { type: "string", description: "File path for line comment" },
      line: { type: "number", description: "Line number for comment" },
      pull_number: {
        type: "number",
        description: "PR number (optional for current context)",
      },
    },
    additionalProperties: false,
  },
  async handler(args) {
    if (!isToolEnabled("create-pull-request-review-comment")) {
      throw new Error(
        "create-pull-request-review-comment safe-output is not enabled"
      );
    }

    const entry = {
      type: "create-pull-request-review-comment",
      body: args.body,
    };

    if (args.path) entry.path = args.path;
    if (args.line) entry.line = args.line;
    if (args.pull_number) entry.pull_number = args.pull_number;

    appendSafeOutput(entry);

    return {
      content: [
        {
          type: "text",
          text: "PR review comment creation queued",
        },
      ],
    };
  },
};

// Create-repository-security-advisory tool
TOOLS["create_repository_security_advisory"] = {
  name: "create_repository_security_advisory",
  description: "Create a repository security advisory",
  inputSchema: {
    type: "object",
    required: ["summary", "description"],
    properties: {
      summary: { type: "string", description: "Advisory summary" },
      description: { type: "string", description: "Advisory description" },
      severity: {
        type: "string",
        enum: ["low", "moderate", "high", "critical"],
        description: "Advisory severity",
      },
      cve_id: { type: "string", description: "CVE ID if known" },
    },
    additionalProperties: false,
  },
  async handler(args) {
    if (!isToolEnabled("create-repository-security-advisory")) {
      throw new Error(
        "create-repository-security-advisory safe-output is not enabled"
      );
    }

    const entry = {
      type: "create-repository-security-advisory",
      summary: args.summary,
      description: args.description,
    };

    if (args.severity) entry.severity = args.severity;
    if (args.cve_id) entry.cve_id = args.cve_id;

    appendSafeOutput(entry);

    return {
      content: [
        {
          type: "text",
          text: `Security advisory creation queued: "${args.summary}"`,
        },
      ],
    };
  },
};

// Add-issue-label tool
TOOLS["add_issue_label"] = {
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
  async handler(args) {
    if (!isToolEnabled("add-issue-label")) {
      throw new Error("add-issue-label safe-output is not enabled");
    }

    const entry = {
      type: "add-issue-label",
      labels: args.labels,
    };

    if (args.issue_number) {
      entry.issue_number = args.issue_number;
    }

    appendSafeOutput(entry);

    return {
      content: [
        {
          type: "text",
          text: `Labels queued for addition: ${args.labels.join(", ")}`,
        },
      ],
    };
  },
};

// Update-issue tool
TOOLS["update_issue"] = {
  name: "update_issue",
  description: "Update a GitHub issue",
  inputSchema: {
    type: "object",
    properties: {
      title: { type: "string", description: "New issue title" },
      body: { type: "string", description: "New issue body" },
      state: {
        type: "string",
        enum: ["open", "closed"],
        description: "Issue state",
      },
      issue_number: {
        type: "number",
        description: "Issue number (optional for current context)",
      },
    },
    additionalProperties: false,
  },
  async handler(args) {
    if (!isToolEnabled("update-issue")) {
      throw new Error("update-issue safe-output is not enabled");
    }

    const entry = {
      type: "update-issue",
    };

    if (args.title) entry.title = args.title;
    if (args.body) entry.body = args.body;
    if (args.state) entry.state = args.state;
    if (args.issue_number) entry.issue_number = args.issue_number;

    // Must have at least one field to update
    if (!args.title && !args.body && !args.state) {
      throw new Error(
        "Must specify at least one field to update (title, body, or state)"
      );
    }

    appendSafeOutput(entry);

    return {
      content: [
        {
          type: "text",
          text: "Issue update queued",
        },
      ],
    };
  },
};

// Push-to-pr-branch tool
TOOLS["push_to_pr_branch"] = {
  name: "push_to_pr_branch",
  description: "Push changes to a pull request branch",
  inputSchema: {
    type: "object",
    required: ["files"],
    properties: {
      files: {
        type: "array",
        items: {
          type: "object",
          required: ["path", "content"],
          properties: {
            path: { type: "string", description: "File path" },
            content: { type: "string", description: "File content" },
          },
        },
        description: "Files to create or update",
      },
      commit_message: { type: "string", description: "Commit message" },
      branch: {
        type: "string",
        description: "Branch name (optional for current context)",
      },
    },
    additionalProperties: false,
  },
  async handler(args) {
    if (!isToolEnabled("push-to-pr-branch")) {
      throw new Error("push-to-pr-branch safe-output is not enabled");
    }

    const entry = {
      type: "push-to-pr-branch",
      files: args.files,
    };

    if (args.commit_message) entry.commit_message = args.commit_message;
    if (args.branch) entry.branch = args.branch;

    appendSafeOutput(entry);

    return {
      content: [
        {
          type: "text",
          text: `Branch push queued with ${args.files.length} file(s)`,
        },
      ],
    };
  },
};

// Missing-tool tool
TOOLS["missing_tool"] = {
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
  async handler(args) {
    if (!isToolEnabled("missing-tool")) {
      throw new Error("missing-tool safe-output is not enabled");
    }

    const entry = {
      type: "missing-tool",
      tool: args.tool,
      reason: args.reason,
    };

    if (args.alternatives) {
      entry.alternatives = args.alternatives;
    }

    appendSafeOutput(entry);

    return {
      content: [
        {
          type: "text",
          text: `Missing tool reported: ${args.tool}`,
        },
      ],
    };
  },
};

// ---------- MCP handlers ----------
const SERVER_INFO = { name: "safe-outputs-mcp-server", version: "1.0.0" };

function handleMessage(req) {
  const { id, method, params } = req;

  try {
    if (method === "initialize") {
      // Initialize configuration on first connection
      initializeSafeOutputsConfig();

      const clientInfo = params?.clientInfo ?? {};
      const protocolVersion = params?.protocolVersion ?? undefined;

      // Advertise that we only support tools (list + call)
      const result = {
        serverInfo: SERVER_INFO,
        // If the client sent a protocolVersion, echo it back for transparency.
        ...(protocolVersion ? { protocolVersion } : {}),
        capabilities: {
          tools: {}, // minimal placeholder object; clients usually just gate on presence
        },
      };
      replyResult(id, result);
      return;
    }

    if (method === "tools/list") {
      const list = [];

      // Only expose tools that are enabled in the configuration
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
      return;
    }

    if (method === "tools/call") {
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

      (async () => {
        try {
          const result = await tool.handler(args);
          // Result shape expected by typical MCP clients for tool calls
          replyResult(id, { content: result.content });
        } catch (e) {
          replyError(id, -32000, `Tool '${name}' failed`, {
            message: String(e?.message ?? e),
          });
        }
      })();
      return;
    }

    // Unknown method
    replyError(id, -32601, `Method not found: ${method}`);
  } catch (e) {
    replyError(id, -32603, "Internal error", {
      message: String(e?.message ?? e),
    });
  }
}

// Optional: log a startup banner to stderr for debugging (not part of the protocol)
process.stderr.write(
  `[${SERVER_INFO.name}] v${SERVER_INFO.version} ready on stdio\n`
);
