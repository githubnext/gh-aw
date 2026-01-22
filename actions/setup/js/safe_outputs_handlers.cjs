// @ts-check

const fs = require("fs");
const path = require("path");
const crypto = require("crypto");

const { normalizeBranchName } = require("./normalize_branch_name.cjs");
const { estimateTokens } = require("./estimate_tokens.cjs");
const { writeLargeContentToFile } = require("./write_large_content_to_file.cjs");
const { getCurrentBranch } = require("./get_current_branch.cjs");
const { getBaseBranch } = require("./get_base_branch.cjs");
const { generateGitPatch } = require("./generate_git_patch.cjs");

/**
 * Create handlers for safe output tools
 * @param {Object} server - The MCP server instance for logging
 * @param {Function} appendSafeOutput - Function to append entries to the output file
 * @param {Object} [config] - Optional configuration object with safe output settings
 * @returns {Object} An object containing all handler functions
 */
function createHandlers(server, appendSafeOutput, config = {}) {
  /**
   * Default handler for safe output tools
   * @param {string} type - The tool type
   * @returns {Function} Handler function
   */
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
          server.debug(`Field '${key}' has ${tokens} tokens (exceeds ${TOKEN_THRESHOLD})`);
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

  /**
   * Handler for upload_asset tool
   */
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
      throw new Error(`File path must be within workspace directory (${workspaceDir}) or /tmp directory. ` + `Provided path: ${filePath} (resolved to: ${absolutePath})`);
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
   * Generates git patch for the changes (unless allow-empty is true)
   */
  const createPullRequestHandler = args => {
    const entry = { ...args, type: "create_pull_request" };
    const baseBranch = getBaseBranch();

    // If branch is not provided, is empty, or equals the base branch, use the current branch from git
    // This handles cases where the agent incorrectly passes the base branch instead of the working branch
    if (!entry.branch || entry.branch.trim() === "" || entry.branch === baseBranch) {
      const detectedBranch = getCurrentBranch();

      if (entry.branch === baseBranch) {
        server.debug(`Branch equals base branch (${baseBranch}), detecting actual working branch: ${detectedBranch}`);
      } else {
        server.debug(`Using current branch for create_pull_request: ${detectedBranch}`);
      }

      entry.branch = detectedBranch;
    }

    // Check if allow-empty is enabled in configuration
    const allowEmpty = config.create_pull_request?.allow_empty === true;

    if (allowEmpty) {
      server.debug(`allow-empty is enabled for create_pull_request - skipping patch generation`);
      // Append the safe output entry without generating a patch
      appendSafeOutput(entry);
      return {
        content: [
          {
            type: "text",
            text: JSON.stringify({
              result: "success",
              message: "Pull request prepared (allow-empty mode - no patch generated)",
              branch: entry.branch,
            }),
          },
        ],
      };
    }

    // Generate git patch
    server.debug(`Generating patch for create_pull_request with branch: ${entry.branch}`);
    const patchResult = generateGitPatch(entry.branch);

    if (!patchResult.success) {
      // Patch generation failed or patch is empty
      const errorMsg = patchResult.error || "Failed to generate patch";
      server.debug(`Patch generation failed: ${errorMsg}`);

      // Return error as content so the agent can see it, rather than throwing
      // which causes the tool call to fail silently in some MCP clients
      return {
        content: [
          {
            type: "text",
            text: JSON.stringify({
              result: "error",
              error: errorMsg,
              details: "No commits were found to create a pull request. Make sure you have committed your changes using git add and git commit before calling create_pull_request.",
            }),
          },
        ],
        isError: true,
      };
    }

    // prettier-ignore
    server.debug(`Patch generated successfully: ${patchResult.patchPath} (${patchResult.patchSize} bytes, ${patchResult.patchLines} lines)`);

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
        server.debug(`Branch equals base branch (${baseBranch}), detecting actual working branch: ${detectedBranch}`);
      } else {
        server.debug(`Using current branch for push_to_pull_request_branch: ${detectedBranch}`);
      }

      entry.branch = detectedBranch;
    }

    // Generate git patch
    server.debug(`Generating patch for push_to_pull_request_branch with branch: ${entry.branch}`);
    const patchResult = generateGitPatch(entry.branch);

    if (!patchResult.success) {
      // Patch generation failed or patch is empty
      const errorMsg = patchResult.error || "Failed to generate patch";
      server.debug(`Patch generation failed: ${errorMsg}`);

      // Return error as content so the agent can see it, rather than throwing
      // which causes the tool call to fail silently in some MCP clients
      return {
        content: [
          {
            type: "text",
            text: JSON.stringify({
              result: "error",
              error: errorMsg,
              details: "No commits were found to push to the pull request branch. Make sure you have committed your changes using git add and git commit before calling push_to_pull_request_branch.",
            }),
          },
        ],
        isError: true,
      };
    }

    // prettier-ignore
    server.debug(`Patch generated successfully: ${patchResult.patchPath} (${patchResult.patchSize} bytes, ${patchResult.patchLines} lines)`);

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
   * Handler for create_project tool
   * Auto-generates a temporary ID if not provided and returns it to the agent
   */
  const createProjectHandler = args => {
    const entry = { ...(args || {}), type: "create_project" };

    // Generate temporary_id if not provided
    if (!entry.temporary_id) {
      entry.temporary_id = "aw_" + crypto.randomBytes(6).toString("hex");
      server.debug(`Auto-generated temporary_id for create_project: ${entry.temporary_id}`);
    }

    // Append to safe outputs
    appendSafeOutput(entry);

    // Return the temporary_id to the agent so it can reference this project
    return {
      content: [
        {
          type: "text",
          text: JSON.stringify({
            result: "success",
            temporary_id: entry.temporary_id,
            project: `#${entry.temporary_id}`,
          }),
        },
      ],
    };
  };

  return {
    defaultHandler,
    uploadAssetHandler,
    createPullRequestHandler,
    pushToPullRequestBranchHandler,
    createProjectHandler,
  };
}

module.exports = { createHandlers };
