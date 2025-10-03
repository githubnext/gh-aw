const fs = require("fs");
const path = require("path");
const crypto = require("crypto");

const encoder = new TextEncoder();
const SERVER_INFO = { name: "safe-outputs-mcp-server", version: "1.0.0" };
const debug = msg => process.stderr.write(`[${SERVER_INFO.name}] ${msg}\n`);

// Handle GITHUB_AW_SAFE_OUTPUTS_CONFIG with default fallback
const configEnv = process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG;
let safeOutputsConfigRaw;

if (!configEnv) {
  // Default config file path
  const defaultConfigPath = "/tmp/safe-outputs/config.json";
  debug(`GITHUB_AW_SAFE_OUTPUTS_CONFIG not set, attempting to read from default path: ${defaultConfigPath}`);

  try {
    if (fs.existsSync(defaultConfigPath)) {
      debug(`Reading config from file: ${defaultConfigPath}`);
      const configFileContent = fs.readFileSync(defaultConfigPath, "utf8");
      debug(`Config file content length: ${configFileContent.length} characters`);
      // Don't log raw content to avoid exposing sensitive configuration data
      debug(`Config file read successfully, attempting to parse JSON`);
      safeOutputsConfigRaw = JSON.parse(configFileContent);
      debug(`Successfully parsed config from file with ${Object.keys(safeOutputsConfigRaw).length} configuration keys`);
    } else {
      debug(`Config file does not exist at: ${defaultConfigPath}`);
      debug(`Using minimal default configuration`);
      safeOutputsConfigRaw = {};
    }
  } catch (error) {
    debug(`Error reading config file: ${error instanceof Error ? error.message : String(error)}`);
    debug(`Falling back to empty configuration`);
    safeOutputsConfigRaw = {};
  }
} else {
  debug(`Using GITHUB_AW_SAFE_OUTPUTS_CONFIG from environment variable`);
  debug(`Config environment variable length: ${configEnv.length} characters`);
  try {
    safeOutputsConfigRaw = JSON.parse(configEnv); // uses dashes for keys
    debug(`Successfully parsed config from environment: ${JSON.stringify(safeOutputsConfigRaw)}`);
  } catch (error) {
    debug(`Error parsing config from environment: ${error instanceof Error ? error.message : String(error)}`);
    throw new Error(`Failed to parse GITHUB_AW_SAFE_OUTPUTS_CONFIG: ${error instanceof Error ? error.message : String(error)}`);
  }
}

const safeOutputsConfig = Object.fromEntries(Object.entries(safeOutputsConfigRaw).map(([k, v]) => [k.replace(/-/g, "_"), v]));
debug(`Final processed config: ${JSON.stringify(safeOutputsConfig)}`);

// Handle GITHUB_AW_SAFE_OUTPUTS with default fallback
const outputFile = process.env.GITHUB_AW_SAFE_OUTPUTS || "/tmp/safe-outputs/outputs.jsonl";
if (!process.env.GITHUB_AW_SAFE_OUTPUTS) {
  debug(`GITHUB_AW_SAFE_OUTPUTS not set, using default: ${outputFile}`);
  // Ensure the directory exists
  const outputDir = path.dirname(outputFile);
  if (!fs.existsSync(outputDir)) {
    debug(`Creating output directory: ${outputDir}`);
    fs.mkdirSync(outputDir, { recursive: true });
  }
}
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

// Create a comprehensive tools map including both predefined tools and dynamic safe-jobs
const TOOLS = {};

// Add predefined tools that are enabled in configuration
ALL_TOOLS.forEach(tool => {
  if (Object.keys(safeOutputsConfig).find(config => normTool(config) === tool.name)) {
    TOOLS[tool.name] = tool;
  }
});

// Add safe-jobs as dynamic tools
Object.keys(safeOutputsConfig).forEach(configKey => {
  const normalizedKey = normTool(configKey);

  // Skip if it's already a predefined tool
  if (TOOLS[normalizedKey]) {
    return;
  }

  // Check if this is a safe-job (not in ALL_TOOLS)
  if (!ALL_TOOLS.find(t => t.name === normalizedKey)) {
    const jobConfig = safeOutputsConfig[configKey];

    // Create a dynamic tool for this safe-job
    const dynamicTool = {
      name: normalizedKey,
      description: `Custom safe-job: ${configKey}`,
      inputSchema: {
        type: "object",
        properties: {},
        additionalProperties: true, // Allow any properties for flexibility
      },
      handler: args => {
        // Create a generic safe-job output entry
        const entry = {
          type: normalizedKey,
          ...args,
        };

        // Write the entry to the output file
        const entryJSON = JSON.stringify(entry);
        fs.appendFileSync(outputFile, entryJSON + "\n");

        // Use output from safe-job config if available
        const outputText =
          jobConfig && jobConfig.output
            ? jobConfig.output
            : `Safe-job '${configKey}' executed successfully with arguments: ${JSON.stringify(args)}`;

        return {
          content: [
            {
              type: "text",
              text: outputText,
            },
          ],
        };
      },
    };

    // Add input schema based on job configuration if available
    if (jobConfig && jobConfig.inputs) {
      dynamicTool.inputSchema.properties = {};
      dynamicTool.inputSchema.required = [];

      Object.keys(jobConfig.inputs).forEach(inputName => {
        const inputDef = jobConfig.inputs[inputName];
        const propSchema = {
          type: inputDef.type || "string",
          description: inputDef.description || `Input parameter: ${inputName}`,
        };

        if (inputDef.options && Array.isArray(inputDef.options)) {
          propSchema.enum = inputDef.options;
        }

        dynamicTool.inputSchema.properties[inputName] = propSchema;

        if (inputDef.required) {
          dynamicTool.inputSchema.required.push(inputName);
        }
      });
    }

    TOOLS[normalizedKey] = dynamicTool;
  }
});

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
        const toolDef = {
          name: tool.name,
          description: tool.description,
          inputSchema: tool.inputSchema,
        };

        // Patch add_labels tool description with allowed labels if configured
        if (tool.name === "add_labels" && safeOutputsConfig.add_labels?.allowed) {
          const allowedLabels = safeOutputsConfig.add_labels.allowed;
          if (Array.isArray(allowedLabels) && allowedLabels.length > 0) {
            toolDef.description = `Add labels to a GitHub issue or pull request. Allowed labels: ${allowedLabels.join(", ")}`;
          }
        }

        list.push(toolDef);
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
