const fs = require("fs");
const path = require("path");
const crypto = require("crypto");

const encoder = new TextEncoder();
const configEnv = process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG;
if (!configEnv) throw new Error("GITHUB_AW_SAFE_OUTPUTS_CONFIG not set");
const safeOutputsConfigRaw = JSON.parse(configEnv); // uses dashes for keys
const safeOutputsConfig = Object.fromEntries(Object.entries(safeOutputsConfigRaw).map(([k, v]) => [k.replace(/-/g, "_"), v]));
const outputFile = process.env.GITHUB_AW_SAFE_OUTPUTS;
if (!outputFile) throw new Error("GITHUB_AW_SAFE_OUTPUTS not set, no output file");
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
      throw new Error(`Parse error: ${error instanceof Error ? error.message : String(error)}`);
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
      debug(`Parse error: ${error instanceof Error ? error.message : String(error)}`);
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

function appendSafeOutput(entry) {
  if (!outputFile) throw new Error("No output file configured");
  entry.type = entry.type.replace(/_/g, "-");
  const jsonLine = JSON.stringify(entry) + "\n";
  try {
    fs.appendFileSync(outputFile, jsonLine);
  } catch (error) {
    throw new Error(`Failed to write to output file: ${error instanceof Error ? error.message : String(error)}`);
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

const uploadAssetHandler = args => {
  const branchName = process.env.GITHUB_AW_ASSETS_BRANCH;
  if (!branchName) throw new Error("GITHUB_AW_ASSETS_BRANCH not set");

  const { path: filePath } = args;

  // Validate file path is within allowed directories
  const absolutePath = path.resolve(filePath);
  const workspaceDir = process.env.GITHUB_WORKSPACE || process.cwd();
  const tmpDir = "/tmp";

  const isInWorkspace = absolutePath.startsWith(path.resolve(workspaceDir));
  const isInTmp = absolutePath.startsWith(tmpDir);

  if (!isInWorkspace && !isInTmp) {
    throw new Error(
      `File path must be within workspace directory (${workspaceDir}) or /tmp directory. ` +
        `Provided path: ${filePath} (resolved to: ${absolutePath})`
    );
  }

  // Validate file exists
  if (!fs.existsSync(filePath)) {
    throw new Error(`File not found: ${filePath}`);
  }

  // Get file stats
  const stats = fs.statSync(filePath);
  const sizeBytes = stats.size;
  const sizeKB = Math.ceil(sizeBytes / 1024);

  // Check file size - read from environment variable if available
  const maxSizeKB = process.env.GITHUB_AW_ASSETS_MAX_SIZE_KB ? parseInt(process.env.GITHUB_AW_ASSETS_MAX_SIZE_KB, 10) : 10240; // Default 10MB
  if (sizeKB > maxSizeKB) {
    throw new Error(`File size ${sizeKB} KB exceeds maximum allowed size ${maxSizeKB} KB`);
  }

  // Check file extension - read from environment variable if available
  const ext = path.extname(filePath).toLowerCase();
  const allowedExts = process.env.GITHUB_AW_ASSETS_ALLOWED_EXTS
    ? process.env.GITHUB_AW_ASSETS_ALLOWED_EXTS.split(",").map(ext => ext.trim())
    : [
        // Default set as specified in problem statement
        ".png",
        ".jpg",
        ".jpeg",
      ];

  if (!allowedExts.includes(ext)) {
    throw new Error(`File extension '${ext}' is not allowed. Allowed extensions: ${allowedExts.join(", ")}`);
  }

  // Create assets directory
  const assetsDir = "/tmp/safe-outputs/assets";
  if (!fs.existsSync(assetsDir)) {
    fs.mkdirSync(assetsDir, { recursive: true });
  }

  // Read file and compute hash
  const fileContent = fs.readFileSync(filePath);
  const sha = crypto.createHash("sha256").update(fileContent).digest("hex");

  // Extract filename and extension
  const fileName = path.basename(filePath);
  const fileExt = path.extname(fileName).toLowerCase();

  // Copy file to assets directory with original name
  const targetPath = path.join(assetsDir, fileName);
  fs.copyFileSync(filePath, targetPath);

  // Generate target filename as sha + extension (lowercased)
  const targetFileName = (sha + fileExt).toLowerCase();

  const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
  const repo = process.env.GITHUB_REPOSITORY || "owner/repo";
  const url = `${githubServer.replace("github.com", "raw.githubusercontent.com")}/${repo}/${branchName}/${targetFileName}`;

  // Create entry for safe outputs
  const entry = {
    type: "upload_asset",
    path: filePath,
    fileName: fileName,
    sha: sha,
    size: sizeBytes,
    url: url,
    targetFileName: targetFileName,
  };

  appendSafeOutput(entry);

  return {
    content: [
      {
        type: "text",
        text: url,
      },
    ],
  };
};

const normTool = toolName => (toolName ? toolName.replace(/-/g, "_").toLowerCase() : undefined);
const ALL_TOOLS = [
  {
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
  },
  {
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
  },
  {
    name: "add_comment",
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
    name: "create_pull_request",
    description: "Create a new GitHub pull request",
    inputSchema: {
      type: "object",
      required: ["title", "body", "branch"],
      properties: {
        title: { type: "string", description: "Pull request title" },
        body: {
          type: "string",
          description: "Pull request body/description",
        },
        branch: {
          type: "string",
          description: "Required branch name",
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
    name: "create_pull_request_review_comment",
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
    name: "create_code_scanning_alert",
    description: "Create a code scanning alert. severity MUST be one of 'error', 'warning', 'info', 'note'.",
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
          description:
            ' Security severity levels follow the industry-standard Common Vulnerability Scoring System (CVSS) that is also used for advisories in the GitHub Advisory Database and must be one of "error", "warning", "info", "note".',
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
    name: "add_labels",
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
  },
  {
    name: "update_release",
    description: "Update a GitHub release",
    inputSchema: {
      type: "object",
      properties: {
        body: { type: "string", description: "Content to append to release body" },
        release_id: {
          type: ["number", "string"],
          description: "Optional release ID for target '*'",
        },
      },
      additionalProperties: false,
    },
  },
  {
    name: "push_to_pull_request_branch",
    description: "Push changes to a pull request branch",
    inputSchema: {
      type: "object",
      required: ["branch", "message"],
      properties: {
        branch: {
          type: "string",
          description: "The name of the branch to push to, should be the branch name associated with the pull request",
        },
        message: { type: "string", description: "Commit message" },
        pull_request_number: {
          type: ["number", "string"],
          description: "Optional pull request number for target '*'",
        },
      },
      additionalProperties: false,
    },
  },
  {
    name: "upload_asset",
    description: "Publish a file as a URL-addressable asset to an orphaned git branch",
    inputSchema: {
      type: "object",
      required: ["path"],
      properties: {
        path: {
          type: "string",
          description:
            "Path to the file to publish as an asset. Must be a file under the current workspace or /tmp directory. By default, images (.png, .jpg, .jpeg) are allowed, but can be configured via workflow settings.",
        },
      },
      additionalProperties: false,
    },
    handler: uploadAssetHandler,
  },
  {
    name: "missing_tool",
    description: "Report a missing tool or functionality needed to complete tasks",
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
];

debug(`v${SERVER_INFO.version} ready on stdio`);
debug(`  output file: ${outputFile}`);
debug(`  config: ${JSON.stringify(safeOutputsConfig)}`);
const unknownTools = Object.keys(safeOutputsConfig).filter(name => !ALL_TOOLS.find(t => t.name === normTool(name)));
if (unknownTools.length) throw new Error(`Unknown tools in configuration: ${unknownTools.join(", ")}`);
const TOOLS = Object.fromEntries(
  ALL_TOOLS.filter(({ name }) => Object.keys(safeOutputsConfig).find(config => normTool(config) === name)).map(tool => [tool.name, tool])
);
debug(`  tools: ${Object.keys(TOOLS).join(", ")}`);
if (!Object.keys(TOOLS).length) throw new Error("No tools enabled in configuration");

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
      console.error(`client info:`, clientInfo);
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
      const tool = TOOLS[normTool(name)];
      if (!tool) {
        replyError(id, -32601, `Tool not found: ${name} (${normTool(name)})`);
        return;
      }
      const handler = tool.handler || defaultHandler(tool.name);
      const requiredFields = tool.inputSchema && Array.isArray(tool.inputSchema.required) ? tool.inputSchema.required : [];
      if (requiredFields.length) {
        const missing = requiredFields.filter(f => {
          const value = args[f];
          return value === undefined || value === null || (typeof value === "string" && value.trim() === "");
        });
        if (missing.length) {
          replyError(id, -32602, `Invalid arguments: missing or empty ${missing.map(m => `'${m}'`).join(", ")}`);
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
