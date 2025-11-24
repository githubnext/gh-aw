// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const path = require("path");
const crypto = require("crypto");
const { execSync } = require("child_process");

const { normalizeBranchName } = require("./normalize_branch_name.cjs");
const { estimateTokens } = require("./estimate_tokens.cjs");
const { generateCompactSchema } = require("./generate_compact_schema.cjs");
const { writeLargeContentToFile } = require("./write_large_content_to_file.cjs");
const { getCurrentBranch } = require("./get_current_branch.cjs");
const { getBaseBranch } = require("./get_base_branch.cjs");
const { generateGitPatch } = require("./generate_git_patch.cjs");

const encoder = new TextEncoder();
const SERVER_INFO = { name: "safeoutputs", version: "1.0.0" };
const debug = msg => process.stderr.write(`[${SERVER_INFO.name}] ${msg}\n`);

// Read configuration from file
const configPath = process.env.GH_AW_SAFE_OUTPUTS_CONFIG_PATH || "/tmp/gh-aw/safeoutputs/config.json";
let safeOutputsConfigRaw;

debug(`Reading config from file: ${configPath}`);

try {
  if (fs.existsSync(configPath)) {
    debug(`Config file exists at: ${configPath}`);
    const configFileContent = fs.readFileSync(configPath, "utf8");
    debug(`Config file content length: ${configFileContent.length} characters`);
    // Don't log raw content to avoid exposing sensitive configuration data
    debug(`Config file read successfully, attempting to parse JSON`);
    safeOutputsConfigRaw = JSON.parse(configFileContent);
    debug(`Successfully parsed config from file with ${Object.keys(safeOutputsConfigRaw).length} configuration keys`);
  } else {
    debug(`Config file does not exist at: ${configPath}`);
    debug(`Using minimal default configuration`);
    safeOutputsConfigRaw = {};
  }
} catch (error) {
  debug(`Error reading config file: ${error instanceof Error ? error.message : String(error)}`);
  debug(`Falling back to empty configuration`);
  safeOutputsConfigRaw = {};
}

const safeOutputsConfig = Object.fromEntries(Object.entries(safeOutputsConfigRaw).map(([k, v]) => [k.replace(/-/g, "_"), v]));
debug(`Final processed config: ${JSON.stringify(safeOutputsConfig)}`);

// Handle GH_AW_SAFE_OUTPUTS with default fallback
const outputFile = process.env.GH_AW_SAFE_OUTPUTS || "/tmp/gh-aw/safeoutputs/outputs.jsonl";
if (!process.env.GH_AW_SAFE_OUTPUTS) {
  debug(`GH_AW_SAFE_OUTPUTS not set, using default: ${outputFile}`);
}
// Always ensure the directory exists, regardless of whether env var is set
const outputDir = path.dirname(outputFile);
if (!fs.existsSync(outputDir)) {
  debug(`Creating output directory: ${outputDir}`);
  fs.mkdirSync(outputDir, { recursive: true });
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
function replyError(id, code, message) {
  // Don't send error responses for notifications (id is null/undefined)
  if (id === undefined || id === null) {
    debug(`Error for notification: ${message}`);
    return;
  }

  const error = { code, message };
  const res = {
    jsonrpc: "2.0",
    id,
    error,
  };
  writeMessage(res);
}

function appendSafeOutput(entry) {
  if (!outputFile) throw new Error("No output file configured");
  // Normalize type to use underscores (convert any dashes to underscores)
  entry.type = entry.type.replace(/-/g, "_");
  const jsonLine = JSON.stringify(entry) + "\n";
  try {
    fs.appendFileSync(outputFile, jsonLine);
  } catch (error) {
    throw new Error(`Failed to write to output file: ${error instanceof Error ? error.message : String(error)}`);
  }
}

const defaultHandler = type => args => {
  const entry = { ...(args || {}), type };

  // Check if any field in the entry has content exceeding 16000 tokens
  let largeContent = null;
  let largeFieldName = null;
  const TOKEN_THRESHOLD = 16000;

  for (const [key, value] of Object.entries(entry)) {
    if (typeof value === "string") {
      const tokens = estimateTokens(value);
      if (tokens > TOKEN_THRESHOLD) {
        largeContent = value;
        largeFieldName = key;
        debug(`Field '${key}' has ${tokens} tokens (exceeds ${TOKEN_THRESHOLD})`);
        break;
      }
    }
  }

  if (largeContent && largeFieldName) {
    // Write large content to file
    const fileInfo = writeLargeContentToFile(largeContent);

    // Replace large field with file reference
    entry[largeFieldName] = `[Content too large, saved to file: ${fileInfo.filename}]`;

    // Append modified entry to safe outputs
    appendSafeOutput(entry);

    // Return file info to the agent
    return {
      content: [
        {
          type: "text",
          text: JSON.stringify(fileInfo),
        },
      ],
    };
  }

  // Normal case - no large content
  appendSafeOutput(entry);
  return {
    content: [
      {
        type: "text",
        text: JSON.stringify({ result: "success" }),
      },
    ],
  };
};

const uploadAssetHandler = args => {
  const branchName = process.env.GH_AW_ASSETS_BRANCH;
  if (!branchName) throw new Error("GH_AW_ASSETS_BRANCH not set");

  // Normalize the branch name to ensure it's a valid git branch name
  const normalizedBranchName = normalizeBranchName(branchName);

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
  const maxSizeKB = process.env.GH_AW_ASSETS_MAX_SIZE_KB ? parseInt(process.env.GH_AW_ASSETS_MAX_SIZE_KB, 10) : 10240; // Default 10MB
  if (sizeKB > maxSizeKB) {
    throw new Error(`File size ${sizeKB} KB exceeds maximum allowed size ${maxSizeKB} KB`);
  }

  // Check file extension - read from environment variable if available
  const ext = path.extname(filePath).toLowerCase();
  const allowedExts = process.env.GH_AW_ASSETS_ALLOWED_EXTS
    ? process.env.GH_AW_ASSETS_ALLOWED_EXTS.split(",").map(ext => ext.trim())
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
  const assetsDir = "/tmp/gh-aw/safeoutputs/assets";
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
  const url = `${githubServer.replace("github.com", "raw.githubusercontent.com")}/${repo}/${normalizedBranchName}/${targetFileName}`;

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
        text: JSON.stringify({ result: url }),
      },
    ],
  };
};

/**
 * Handler for create_pull_request tool
 * Resolves the current branch if branch is not provided or is the base branch
 * Generates git patch for the changes
 */
const createPullRequestHandler = args => {
  const entry = { ...args, type: "create_pull_request" };
  const baseBranch = getBaseBranch();

  // If branch is not provided, is empty, or equals the base branch, use the current branch from git
  // This handles cases where the agent incorrectly passes the base branch instead of the working branch
  if (!entry.branch || entry.branch.trim() === "" || entry.branch === baseBranch) {
    const detectedBranch = getCurrentBranch();

    if (entry.branch === baseBranch) {
      debug(`Branch equals base branch (${baseBranch}), detecting actual working branch: ${detectedBranch}`);
    } else {
      debug(`Using current branch for create_pull_request: ${detectedBranch}`);
    }

    entry.branch = detectedBranch;
  }

  // Generate git patch
  debug(`Generating patch for create_pull_request with branch: ${entry.branch}`);
  const patchResult = generateGitPatch(entry.branch);

  if (!patchResult.success) {
    // Patch generation failed or patch is empty
    const errorMsg = patchResult.error || "Failed to generate patch";
    debug(`Patch generation failed: ${errorMsg}`);
    throw new Error(errorMsg);
  }

  debug(`Patch generated successfully: ${patchResult.patchPath} (${patchResult.patchSize} bytes, ${patchResult.patchLines} lines)`);

  appendSafeOutput(entry);
  return {
    content: [
      {
        type: "text",
        text: JSON.stringify({
          result: "success",
          patch: {
            path: patchResult.patchPath,
            size: patchResult.patchSize,
            lines: patchResult.patchLines,
          },
        }),
      },
    ],
  };
};

/**
 * Handler for push_to_pull_request_branch tool
 * Resolves the current branch if branch is not provided or is the base branch
 * Generates git patch for the changes
 */
const pushToPullRequestBranchHandler = args => {
  const entry = { ...args, type: "push_to_pull_request_branch" };
  const baseBranch = getBaseBranch();

  // If branch is not provided, is empty, or equals the base branch, use the current branch from git
  // This handles cases where the agent incorrectly passes the base branch instead of the working branch
  if (!entry.branch || entry.branch.trim() === "" || entry.branch === baseBranch) {
    const detectedBranch = getCurrentBranch();

    if (entry.branch === baseBranch) {
      debug(`Branch equals base branch (${baseBranch}), detecting actual working branch: ${detectedBranch}`);
    } else {
      debug(`Using current branch for push_to_pull_request_branch: ${detectedBranch}`);
    }

    entry.branch = detectedBranch;
  }

  // Generate git patch
  debug(`Generating patch for push_to_pull_request_branch with branch: ${entry.branch}`);
  const patchResult = generateGitPatch(entry.branch);

  if (!patchResult.success) {
    // Patch generation failed or patch is empty
    const errorMsg = patchResult.error || "Failed to generate patch";
    debug(`Patch generation failed: ${errorMsg}`);
    throw new Error(errorMsg);
  }

  debug(`Patch generated successfully: ${patchResult.patchPath} (${patchResult.patchSize} bytes, ${patchResult.patchLines} lines)`);

  appendSafeOutput(entry);
  return {
    content: [
      {
        type: "text",
        text: JSON.stringify({
          result: "success",
          patch: {
            path: patchResult.patchPath,
            size: patchResult.patchSize,
            lines: patchResult.patchLines,
          },
        }),
      },
    ],
  };
};

const normTool = toolName => (toolName ? toolName.replace(/-/g, "_").toLowerCase() : undefined);

// Load tools from the tools.json file
const toolsPath = process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH || "/tmp/gh-aw/safeoutputs/tools.json";
let ALL_TOOLS = [];

debug(`Reading tools from file: ${toolsPath}`);

try {
  if (fs.existsSync(toolsPath)) {
    debug(`Tools file exists at: ${toolsPath}`);
    const toolsFileContent = fs.readFileSync(toolsPath, "utf8");
    debug(`Tools file content length: ${toolsFileContent.length} characters`);
    debug(`Tools file read successfully, attempting to parse JSON`);
    ALL_TOOLS = JSON.parse(toolsFileContent);
    debug(`Successfully parsed ${ALL_TOOLS.length} tools from file`);
  } else {
    debug(`Tools file does not exist at: ${toolsPath}`);
    debug(`Using empty tools array`);
    ALL_TOOLS = [];
  }
} catch (error) {
  debug(`Error reading tools file: ${error instanceof Error ? error.message : String(error)}`);
  debug(`Falling back to empty tools array`);
  ALL_TOOLS = [];
}

// Attach handlers to tools that need them
// Handlers must be attached after loading from JSON since functions can't be serialized
ALL_TOOLS.forEach(tool => {
  if (tool.name === "create_pull_request") {
    tool.handler = createPullRequestHandler;
  } else if (tool.name === "push_to_pull_request_branch") {
    tool.handler = pushToPullRequestBranchHandler;
  } else if (tool.name === "upload_asset") {
    tool.handler = uploadAssetHandler;
  }
});

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
      description: jobConfig && jobConfig.description ? jobConfig.description : `Custom safe-job: ${configKey}`,
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
              text: JSON.stringify({ result: outputText }),
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

        // Patch update_issue tool description with allowed operations if configured
        if (tool.name === "update_issue" && safeOutputsConfig.update_issue) {
          const config = safeOutputsConfig.update_issue;
          const allowedOps = [];
          if (config.status !== false) allowedOps.push("status");
          if (config.title !== false) allowedOps.push("title");
          if (config.body !== false) allowedOps.push("body");

          if (allowedOps.length > 0 && allowedOps.length < 3) {
            // Only patch if some operations are restricted (not all 3 allowed)
            toolDef.description = `Update a GitHub issue. Allowed updates: ${allowedOps.join(", ")}`;
          }
        }

        // Patch upload_asset tool description with constraints from environment
        if (tool.name === "upload_asset") {
          const maxSizeKB = process.env.GH_AW_ASSETS_MAX_SIZE_KB ? parseInt(process.env.GH_AW_ASSETS_MAX_SIZE_KB, 10) : 10240;
          const allowedExts = process.env.GH_AW_ASSETS_ALLOWED_EXTS
            ? process.env.GH_AW_ASSETS_ALLOWED_EXTS.split(",").map(ext => ext.trim())
            : [".png", ".jpg", ".jpeg"];

          toolDef.description = `Publish a file as a URL-addressable asset to an orphaned git branch. Maximum file size: ${maxSizeKB} KB. Allowed extensions: ${allowedExts.join(", ")}`;
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
      replyResult(id, { content, isError: false });
    } else if (/^notifications\//.test(method)) {
      debug(`ignore ${method}`);
    } else {
      replyError(id, -32601, `Method not found: ${method}`);
    }
  } catch (e) {
    replyError(id, -32603, e instanceof Error ? e.message : String(e));
  }
}

process.stdin.on("data", onData);
process.stdin.on("error", err => debug(`stdin error: ${err}`));
process.stdin.resume();
debug(`listening...`);
