// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const path = require("path");
const crypto = require("crypto");

const { normalizeBranchName } = require("./normalize_branch_name.cjs");
const { estimateTokens } = require("./estimate_tokens.cjs");
const { writeLargeContentToFile } = require("./write_large_content_to_file.cjs");
const { getCurrentBranch } = require("./get_current_branch.cjs");
const { getBaseBranch } = require("./get_base_branch.cjs");
const { generateGitPatch } = require("./generate_git_patch.cjs");
const { generateGitBundle } = require("./generate_git_bundle.cjs");
const { enforceCommentLimits } = require("./comment_limit_helpers.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { ERR_CONFIG, ERR_SYSTEM, ERR_VALIDATION } = require("./error_codes.cjs");
const { findRepoCheckout } = require("./find_repo_checkout.cjs");
const { resolveTargetRepoConfig, resolveAndValidateRepo } = require("./repo_helpers.cjs");
const { getOrGenerateTemporaryId } = require("./temporary_id.cjs");
const { parseAllowedExtensionsEnv } = require("./allowed_extensions_helpers.cjs");

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
    if (!branchName) throw new Error(`${ERR_CONFIG}: GH_AW_ASSETS_BRANCH not set`);

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
      throw new Error(`${ERR_CONFIG}: File path must be within workspace directory (${workspaceDir}) or /tmp directory. ` + `Provided path: ${filePath} (resolved to: ${absolutePath})`);
    }

    // Validate file exists
    if (!fs.existsSync(filePath)) {
      throw new Error(`${ERR_SYSTEM}: File not found: ${filePath}`);
    }

    // Get file stats
    const stats = fs.statSync(filePath);
    const sizeBytes = stats.size;
    const sizeKB = Math.ceil(sizeBytes / 1024);

    // Check file size - read from environment variable if available
    const maxSizeKB = process.env.GH_AW_ASSETS_MAX_SIZE_KB ? parseInt(process.env.GH_AW_ASSETS_MAX_SIZE_KB, 10) : 10240; // Default 10MB
    if (sizeKB > maxSizeKB) {
      throw new Error(`${ERR_VALIDATION}: File size ${sizeKB} KB exceeds maximum allowed size ${maxSizeKB} KB`);
    }

    // Check file extension - read from environment variable if available
    const ext = path.extname(filePath).toLowerCase();
    const parsedAllowedExts = parseAllowedExtensionsEnv(process.env.GH_AW_ASSETS_ALLOWED_EXTS);
    if (parsedAllowedExts?.hasUnresolvedExpression) {
      throw new Error(`${ERR_CONFIG}: GH_AW_ASSETS_ALLOWED_EXTS contains unresolved GitHub Actions expression. Ensure expressions resolve before safe outputs validation.`);
    }
    const allowedExts = parsedAllowedExts
      ? parsedAllowedExts.normalizedValues
      : [
          // Default set as specified in problem statement
          ".png",
          ".jpg",
          ".jpeg",
        ];
    if (!allowedExts.includes(ext)) {
      throw new Error(`${ERR_VALIDATION}: File extension '${ext}' is not allowed. Allowed extensions: ${allowedExts.join(", ")}`);
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
    let url;
    try {
      const serverHostname = new URL(githubServer).hostname;
      if (serverHostname === "github.com") {
        url = `https://github.com/${repo}/blob/${normalizedBranchName}/${targetFileName}?raw=true`;
      } else {
        // GitHub Enterprise Server - raw content is served from the same host with /raw/ path
        url = `${githubServer}/${repo}/raw/${normalizedBranchName}/${targetFileName}`;
      }
    } catch {
      url = `${githubServer}/${repo}/raw/${normalizedBranchName}/${targetFileName}`;
    }

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
   * Supports multi-repo scenarios via the optional 'repo' parameter
   */
  const createPullRequestHandler = async args => {
    const entry = { ...args, type: "create_pull_request" };

    // Resolve target repo configuration and validate the target repo early
    // This is needed before getBaseBranch to ensure we resolve the base branch
    // for the correct repository (especially in cross-repo scenarios)
    const prConfig = config.create_pull_request || {};
    const { defaultTargetRepo, allowedRepos } = resolveTargetRepoConfig(prConfig);

    // Resolve and validate the target repository from the entry
    const repoResult = resolveAndValidateRepo(entry, defaultTargetRepo, allowedRepos, "pull request");
    if (!repoResult.success) {
      let error = repoResult.error;
      const owningRepo = process.env.GITHUB_REPOSITORY;
      if (entry.repo === owningRepo && defaultTargetRepo && defaultTargetRepo !== owningRepo) {
        error += ` Hint: This workflow runs in '${owningRepo}' but is configured to target '${defaultTargetRepo}'. Omit the 'repo' parameter to use the configured target, or pass repo: '${defaultTargetRepo}'.`;
      }
      return {
        content: [
          {
            type: "text",
            text: JSON.stringify({
              result: "error",
              error,
            }),
          },
        ],
        isError: true,
      };
    }
    const { repoParts } = repoResult;

    // Get base branch for the resolved target repository.
    // Prefer explicit safe-output config value when provided, otherwise fall back
    // to dynamic resolution from trigger context/default branch.
    const baseBranch = prConfig.base_branch || (await getBaseBranch(repoParts));

    // Determine the working directory for git operations
    // If repo is specified, find where it's checked out
    let repoCwd = null;
    let repoSlug = null;

    if (entry.repo && entry.repo.trim()) {
      // Use the validated/qualified repo slug from repoResult to avoid divergence
      // between the raw user input and the normalized/qualified repo name
      repoSlug = repoResult.repo;
      server.debug(`Multi-repo mode: looking for checkout of ${repoSlug}`);

      const checkoutResult = findRepoCheckout(repoSlug);
      if (!checkoutResult.success) {
        server.debug(`Failed to find repo checkout: ${checkoutResult.error}`);
        return {
          content: [
            {
              type: "text",
              text: JSON.stringify({
                result: "error",
                error: checkoutResult.error,
                details:
                  `Repository '${repoSlug}' was not found as a git checkout in the workspace. ` +
                  `For multi-repo workflows, use actions/checkout with a 'path' parameter to checkout ` +
                  `each repo to a subdirectory (e.g., 'repos/repo-a/').`,
              }),
            },
          ],
          isError: true,
        };
      }

      repoCwd = checkoutResult.path;
      server.debug(`Found repo checkout at: ${repoCwd}`);
    }

    // If branch is not provided, is empty, or equals the base branch, use the current branch from git
    // This handles cases where the agent incorrectly passes the base branch instead of the working branch
    if (!entry.branch || entry.branch.trim() === "" || entry.branch === baseBranch) {
      const detectedBranch = getCurrentBranch(repoCwd);

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

    // Determine transport format: "bundle" uses git bundle (preserves merge topology),
    // "am" (default) uses git format-patch / git am (good for linear histories).
    const patchFormat = prConfig["patch_format"] || config["patch_format"] || "am";
    const useBundle = patchFormat === "bundle";

    // Build common options for both patch and bundle generation
    const transportOptions = {};
    if (repoCwd) {
      transportOptions.cwd = repoCwd;
    }
    if (repoSlug) {
      transportOptions.repoSlug = repoSlug;
    }
    // Pass per-handler token so cross-repo PATs are used for git fetch when configured.
    // Falls back to GITHUB_TOKEN if not set.
    if (prConfig["github-token"]) {
      transportOptions.token = prConfig["github-token"];
    }

    if (useBundle) {
      // Bundle transport: preserves merge commits and per-commit metadata
      server.debug(`Generating bundle for create_pull_request with branch: ${entry.branch}${repoCwd ? ` in ${repoCwd} baseBranch: ${baseBranch}` : ""}`);
      const bundleResult = await generateGitBundle(entry.branch, baseBranch, transportOptions);

      if (!bundleResult.success) {
        const errorMsg = bundleResult.error || "Failed to generate bundle";
        server.debug(`Bundle generation failed: ${errorMsg}`);
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

      server.debug(`Bundle generated successfully: ${bundleResult.bundlePath} (${bundleResult.bundleSize} bytes)`);

      // Store the bundle path in the entry so consumers know which file to use
      entry.bundle_path = bundleResult.bundlePath;

      if (bundleResult.baseCommit) {
        entry.base_commit = bundleResult.baseCommit;
      }

      appendSafeOutput(entry);
      return {
        content: [
          {
            type: "text",
            text: JSON.stringify({
              result: "success",
              bundle: {
                path: bundleResult.bundlePath,
                size: bundleResult.bundleSize,
              },
            }),
          },
        ],
      };
    }

    // Patch transport (default): uses git format-patch / git am
    server.debug(`Generating patch for create_pull_request with branch: ${entry.branch}${repoCwd ? ` in ${repoCwd} baseBranch: ${baseBranch}` : ""}`);
    /** @type {Record<string, any>} */
    const patchOptions = { ...transportOptions };
    // Pass excluded_files so git excludes them via :(exclude) pathspecs at generation time.
    if (Array.isArray(prConfig.excluded_files) && prConfig.excluded_files.length > 0) {
      patchOptions.excludedFiles = prConfig.excluded_files;
    }
    const patchResult = await generateGitPatch(entry.branch, baseBranch, patchOptions);

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

    // Store the patch path in the entry so consumers know which file to use
    entry.patch_path = patchResult.patchPath;

    // Store the base commit SHA so the create_pull_request handler can use it
    // directly in the fallback path (the From <sha> header in format-patch output
    // contains the agent's commit SHA which won't exist in the target checkout)
    if (patchResult.baseCommit) {
      entry.base_commit = patchResult.baseCommit;
    }

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
   *
   * Note: Fork PR detection is handled by push_to_pull_request_branch.cjs handler
   * which fetches the PR and calls detectForkPR() with full PR data.
   */
  const pushToPullRequestBranchHandler = async args => {
    const entry = { ...args, type: "push_to_pull_request_branch" };

    // Resolve target repo configuration and validate the target repo early
    // This is needed before getBaseBranch to ensure we resolve the base branch
    // for the correct repository (especially in cross-repo scenarios)
    const pushConfig = config.push_to_pull_request_branch || {};
    const { defaultTargetRepo, allowedRepos } = resolveTargetRepoConfig(pushConfig);

    // Resolve and validate the target repository from the entry
    const repoResult = resolveAndValidateRepo(entry, defaultTargetRepo, allowedRepos, "push to PR branch");
    if (!repoResult.success) {
      return {
        content: [
          {
            type: "text",
            text: JSON.stringify({
              result: "error",
              error: repoResult.error,
            }),
          },
        ],
        isError: true,
      };
    }
    const { repoParts } = repoResult;

    // Get base branch for the resolved target repository
    const baseBranch = await getBaseBranch(repoParts);

    // Determine the working directory for git operations.
    // Look up the checkout path when the target repo is explicitly provided by the agent
    // or explicitly configured via target-repo in the workflow config — this ensures patch
    // generation runs from the correct directory when the target repo is checked out in a subdirectory.
    let repoCwd = null;
    const itemRepo = repoResult.repo;
    if ((entry.repo && entry.repo.trim()) || pushConfig["target-repo"]) {
      server.debug(`Looking for checkout of target repo: ${itemRepo}`);
      const checkoutResult = findRepoCheckout(itemRepo);
      if (!checkoutResult.success) {
        return {
          content: [
            {
              type: "text",
              text: JSON.stringify({
                result: "error",
                error: `Repository '${itemRepo}' not found in workspace. Check out the target repo with actions/checkout and set its 'path' input so the checkout can be located. If checking out multiple repositories, ensure each actions/checkout step uses the appropriate 'path' input.`,
              }),
            },
          ],
          isError: true,
        };
      }
      repoCwd = checkoutResult.path;
      entry.repo_cwd = repoCwd;
      server.debug(`Selected checkout folder for ${itemRepo}: ${repoCwd}`);
    }

    // If branch is not provided, is empty, or equals the base branch, use the current branch from git
    // This handles cases where the agent incorrectly passes the base branch instead of the working branch
    if (!entry.branch || entry.branch.trim() === "" || entry.branch === baseBranch) {
      const detectedBranch = getCurrentBranch(repoCwd);

      if (entry.branch === baseBranch) {
        server.debug(`Branch equals base branch (${baseBranch}), detecting actual working branch: ${detectedBranch}`);
      } else {
        server.debug(`Using current branch for push_to_pull_request_branch: ${detectedBranch}`);
      }

      entry.branch = detectedBranch;
    }

    // Determine transport format: "bundle" uses git bundle (preserves merge topology),
    // "am" (default) uses git format-patch / git am (good for linear histories).
    const pushPatchFormat = pushConfig["patch_format"] || config["patch_format"] || "am";
    const useBundle = pushPatchFormat === "bundle";

    // Build common options for both patch and bundle generation
    const pushTransportOptions = { mode: "incremental" };
    if (repoCwd) {
      pushTransportOptions.cwd = repoCwd;
      pushTransportOptions.repoSlug = repoResult.repo;
    }
    // Pass per-handler token so cross-repo PATs are used for git fetch when configured.
    // Falls back to GITHUB_TOKEN if not set.
    if (pushConfig["github-token"]) {
      pushTransportOptions.token = pushConfig["github-token"];
    }

    if (useBundle) {
      // Bundle transport: preserves merge commits and per-commit metadata
      server.debug(`Generating incremental bundle for push_to_pull_request_branch with branch: ${entry.branch}, baseBranch: ${baseBranch}`);
      const bundleResult = await generateGitBundle(entry.branch, baseBranch, pushTransportOptions);

      if (!bundleResult.success) {
        const errorMsg = bundleResult.error || "Failed to generate bundle";
        server.debug(`Bundle generation failed: ${errorMsg}`);
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

      server.debug(`Bundle generated successfully: ${bundleResult.bundlePath} (${bundleResult.bundleSize} bytes)`);

      // Store the bundle path in the entry so consumers know which file to use
      entry.bundle_path = bundleResult.bundlePath;

      if (bundleResult.baseCommit) {
        entry.base_commit = bundleResult.baseCommit;
      }

      appendSafeOutput(entry);
      return {
        content: [
          {
            type: "text",
            text: JSON.stringify({
              result: "success",
              bundle: {
                path: bundleResult.bundlePath,
                size: bundleResult.bundleSize,
              },
            }),
          },
        ],
      };
    }

    // Patch transport (default): uses git format-patch / git am
    // Incremental mode only includes commits since origin/branchName,
    // preventing patches that include already-existing commits
    server.debug(`Generating incremental patch for push_to_pull_request_branch with branch: ${entry.branch}, baseBranch: ${baseBranch}`);
    /** @type {Record<string, any>} */
    const pushPatchOptions = { ...pushTransportOptions };
    // Pass excluded_files so git excludes them via :(exclude) pathspecs at generation time.
    if (Array.isArray(pushConfig.excluded_files) && pushConfig.excluded_files.length > 0) {
      pushPatchOptions.excludedFiles = pushConfig.excluded_files;
    }
    const patchResult = await generateGitPatch(entry.branch, baseBranch, pushPatchOptions);

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
    server.debug(`Patch generated successfully: ${patchResult.patchPath} (${patchResult.patchSize} bytes, ${patchResult.patchLines} lines, diffSize=${patchResult.diffSize ?? "(n/a)"} bytes)`);

    // Store the patch path in the entry so consumers know which file to use
    entry.patch_path = patchResult.patchPath;

    // Store the base commit SHA so the push handler can use it directly
    if (patchResult.baseCommit) {
      entry.base_commit = patchResult.baseCommit;
    }

    // Store the incremental net diff size so push_to_pull_request_branch can
    // validate `max_patch_size` against the actual incremental change relative
    // to the existing PR branch head, not the (potentially much larger) size of
    // the format-patch transport file. This is critical for the long-running
    // branch pattern where the format-patch can include many
    // commits but each iteration only changes a few KB.
    if (typeof patchResult.diffSize === "number" && patchResult.diffSize >= 0) {
      entry.diff_size = patchResult.diffSize;
    }

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
   * Handler for push_repo_memory tool
   * Validates that memory files in the configured memory directory are within size limits.
   * Returns an error if any file or the total size exceeds the configured limits,
   * with guidance to reduce memory size before the workflow completes.
   */
  const pushRepoMemoryHandler = args => {
    const memoryId = (args && args.memory_id) || "default";
    const repoMemoryConfig = config.push_repo_memory;

    if (!repoMemoryConfig || !repoMemoryConfig.memories || repoMemoryConfig.memories.length === 0) {
      return {
        content: [
          {
            type: "text",
            text: JSON.stringify({ result: "success", message: "No repo-memory configured." }),
          },
        ],
      };
    }

    // Find the memory config for the requested memory_id
    const memoryConf = repoMemoryConfig.memories.find(m => m.id === memoryId);
    if (!memoryConf) {
      const availableIds = repoMemoryConfig.memories.map(m => m.id).join(", ");
      return {
        content: [
          {
            type: "text",
            text: JSON.stringify({
              result: "error",
              error: `Memory ID '${memoryId}' not found. Available memory IDs: ${availableIds}`,
            }),
          },
        ],
        isError: true,
      };
    }

    const memoryDir = memoryConf.dir;
    const maxFileSize = memoryConf.max_file_size || 10240;
    const maxPatchSize = memoryConf.max_patch_size || 10240;
    const maxFileCount = memoryConf.max_file_count || 100;
    // Allow 20% overhead for git diff format (headers, context lines, etc.)
    const effectiveMaxPatchSize = Math.floor(maxPatchSize * 1.2);

    if (!fs.existsSync(memoryDir)) {
      return {
        content: [
          {
            type: "text",
            text: JSON.stringify({ result: "success", message: `Memory directory '${memoryDir}' does not exist yet. No files to validate.` }),
          },
        ],
      };
    }

    // Recursively scan all files in the memory directory
    /** @type {Array<{relativePath: string, size: number}>} */
    const files = [];

    /**
     * @param {string} dirPath
     * @param {string} relativePath
     */
    function scanDir(dirPath, relativePath) {
      const entries = fs.readdirSync(dirPath, { withFileTypes: true });
      for (const entry of entries) {
        // Skip .git directory to avoid counting git metadata as memory content.
        // The memory directory is a git clone, so .git may contain pack files that
        // grow with each commit and must not be counted toward the memory size limit.
        if (entry.isDirectory() && entry.name === ".git") {
          continue;
        }
        const fullPath = path.join(dirPath, entry.name);
        const relPath = relativePath ? path.join(relativePath, entry.name) : entry.name;
        if (entry.isDirectory()) {
          scanDir(fullPath, relPath);
        } else if (entry.isFile()) {
          const stats = fs.statSync(fullPath);
          files.push({ relativePath: relPath.replace(/\\/g, "/"), size: stats.size });
        }
      }
    }

    try {
      scanDir(memoryDir, "");
    } catch (/** @type {any} */ error) {
      return {
        content: [
          {
            type: "text",
            text: JSON.stringify({
              result: "error",
              error: `Failed to scan memory directory: ${getErrorMessage(error)}`,
            }),
          },
        ],
        isError: true,
      };
    }

    // Check individual file sizes
    const oversizedFiles = files.filter(f => f.size > maxFileSize);
    if (oversizedFiles.length > 0) {
      const details = oversizedFiles.map(f => `  - ${f.relativePath} (${f.size} bytes > ${maxFileSize} bytes limit)`).join("\n");
      return {
        content: [
          {
            type: "text",
            text: JSON.stringify({
              result: "error",
              error:
                `${oversizedFiles.length} file(s) exceed the maximum file size of ${maxFileSize} bytes (${Math.ceil(maxFileSize / 1024)} KB):\n${details}\n\n` +
                `Please reduce the size of these files before the workflow completes. Consider summarizing or truncating the content.`,
            }),
          },
        ],
        isError: true,
      };
    }

    // Check file count
    if (files.length > maxFileCount) {
      return {
        content: [
          {
            type: "text",
            text: JSON.stringify({
              result: "error",
              error: `Too many files in memory: ${files.length} files exceeds the limit of ${maxFileCount} files.\n\n` + `Please reduce the number of files in '${memoryDir}' before the workflow completes.`,
            }),
          },
        ],
        isError: true,
      };
    }

    // Check total size. The effective limit allows 20% overhead to account for
    // git diff format overhead (headers, context lines, metadata). This mirrors
    // the same calculation in push_repo_memory.cjs. The totalSize is the raw
    // sum of file sizes; it is compared against the overhead-adjusted limit.
    const totalSize = files.reduce((sum, f) => sum + f.size, 0);
    const totalSizeKb = Math.ceil(totalSize / 1024);
    const effectiveMaxKb = Math.floor(effectiveMaxPatchSize / 1024);

    core.debug(`push_repo_memory validation: ${files.length} files, total ${totalSize} bytes, effective limit ${effectiveMaxPatchSize} bytes`);

    if (totalSize > effectiveMaxPatchSize) {
      return {
        content: [
          {
            type: "text",
            text: JSON.stringify({
              result: "error",
              error:
                `Total memory size (${totalSizeKb} KB) exceeds the allowed limit of ${effectiveMaxKb} KB ` +
                `(configured limit: ${Math.floor(maxPatchSize / 1024)} KB with 20% overhead for git diff format).\n\n` +
                `Please reduce the total size of files in '${memoryDir}' before the workflow completes. ` +
                `Consider: summarizing notes instead of keeping full history, removing outdated entries, or compressing data. ` +
                `Then call push_repo_memory again to verify the size is within limits.`,
            }),
          },
        ],
        isError: true,
      };
    }

    return {
      content: [
        {
          type: "text",
          text: JSON.stringify({
            result: "success",
            message: `Memory validation passed: ${files.length} file(s), ${totalSizeKb} KB total (limit: ${effectiveMaxKb} KB with 20% overhead).`,
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

    // Use helper to validate or generate temporary_id
    const tempIdResult = getOrGenerateTemporaryId(entry, "create_project");
    if (tempIdResult.error) {
      throw {
        code: -32602,
        message: tempIdResult.error,
      };
    }
    entry.temporary_id = tempIdResult.temporaryId;
    server.debug(`temporary_id for create_project: ${entry.temporary_id}`);

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

  /**
   * Handler for add_comment tool
   * Per Safe Outputs Specification MCE1: Enforces constraints during tool invocation
   * to provide immediate feedback to the LLM before recording to NDJSON
   * Also auto-generates a temporary_id if not provided and returns it to the agent
   */
  const addCommentHandler = args => {
    // Validate comment constraints before appending to safe outputs
    // This provides early feedback per Requirement MCE1 (Early Validation)
    try {
      const body = (args && args.body) || "";
      enforceCommentLimits(body);
    } catch (error) {
      // Return validation error with specific constraint violation details
      // Per Requirement MCE3 (Actionable Error Responses)
      // Use JSON-RPC error code -32602 (Invalid params) per MCP specification
      throw {
        code: -32602,
        message: getErrorMessage(error),
      };
    }

    // Build the entry with a temporary_id
    const entry = { ...(args || {}), type: "add_comment" };

    // Use helper to validate or generate temporary_id
    const tempIdResult = getOrGenerateTemporaryId(entry, "add_comment");
    if (tempIdResult.error) {
      throw {
        code: -32602,
        message: tempIdResult.error,
      };
    }
    entry.temporary_id = tempIdResult.temporaryId;
    server.debug(`temporary_id for add_comment: ${entry.temporary_id}`);

    // Append to safe outputs
    appendSafeOutput(entry);

    // Return the temporary_id to the agent so it can reference this comment
    return {
      content: [
        {
          type: "text",
          text: JSON.stringify({
            result: "success",
            temporary_id: entry.temporary_id,
            comment: `#${entry.temporary_id}`,
          }),
        },
      ],
    };
  };

  /**
   * Recursively copy all regular files from srcDir into destDir, preserving the relative
   * path structure under srcDir. Non-regular entries (sockets, devices, pipes, symlinks)
   * are skipped silently.
   * @param {string} srcDir - Absolute source directory path
   * @param {string} destDir - Absolute destination directory path
   */
  function copyDirectoryRecursive(srcDir, destDir) {
    if (!fs.existsSync(destDir)) {
      fs.mkdirSync(destDir, { recursive: true });
    }
    for (const ent of fs.readdirSync(srcDir, { withFileTypes: true })) {
      const srcPath = path.join(srcDir, ent.name);
      const destPath = path.join(destDir, ent.name);
      if (ent.isDirectory()) {
        copyDirectoryRecursive(srcPath, destPath);
      } else if (ent.isFile() && !ent.isSymbolicLink() && !fs.existsSync(destPath)) {
        fs.copyFileSync(srcPath, destPath);
      }
      // Skip symlinks, sockets, pipes, block/char devices — non-regular file types.
    }
  }

  /**
   * Handler for upload_artifact tool.
   *
   * When the agent calls upload_artifact with an absolute path (e.g.,
   * /tmp/gh-aw/python/charts/loc_by_language.png), the file lives only inside the
   * sandboxed container.  After the container exits the file is gone, so the safe_outputs
   * job running on a different runner cannot find it.
   *
   * This handler copies the file (or directory) to the staging directory
   * ($RUNNER_TEMP/gh-aw/safeoutputs/upload-artifacts/), which is bind-mounted rw into
   * the container.  The agent job then uploads that staging directory as the
   * safe-outputs-upload-artifacts artifact, and the safe_outputs job downloads it before
   * processing.
   *
   * For path-based requests with an absolute path the handler also rewrites entry.path to
   * the staging-relative basename so that upload_artifact.cjs on the safe_outputs runner
   * resolves the file from staging rather than trying the (non-existent) absolute path.
   *
   * Relative paths and filter-based requests are passed through unchanged because the
   * agent is expected to have placed those files in staging directly.
   */
  const uploadArtifactHandler = args => {
    const entry = { ...(args || {}), type: "upload_artifact" };

    if (typeof entry.path === "string" && path.isAbsolute(entry.path)) {
      const filePath = entry.path;

      if (!fs.existsSync(filePath)) {
        throw {
          code: -32602,
          message: `${ERR_VALIDATION}: upload_artifact: file not found: ${filePath}`,
        };
      }

      const stat = fs.lstatSync(filePath);
      if (stat.isSymbolicLink()) {
        throw {
          code: -32602,
          message: `${ERR_VALIDATION}: upload_artifact: symlinks are not allowed: ${filePath}`,
        };
      }

      const stagingDir = path.join(process.env.RUNNER_TEMP || "/tmp", "gh-aw", "safeoutputs", "upload-artifacts");
      if (!fs.existsSync(stagingDir)) {
        fs.mkdirSync(stagingDir, { recursive: true });
      }

      const destName = path.basename(filePath);

      if (stat.isDirectory()) {
        copyDirectoryRecursive(filePath, path.join(stagingDir, destName));
      } else {
        const destPath = path.join(stagingDir, destName);
        if (!fs.existsSync(destPath)) {
          fs.copyFileSync(filePath, destPath);
        }
      }

      // Rewrite to staging-relative path so upload_artifact.cjs resolves it from staging.
      entry.path = destName;
      server.debug(`upload_artifact: staged ${filePath} as ${destName}`);
    }

    appendSafeOutput(entry);

    const temporaryId = entry.temporary_id || null;
    return {
      content: [
        {
          type: "text",
          text: JSON.stringify({
            result: "success",
            ...(temporaryId ? { temporary_id: temporaryId } : {}),
          }),
        },
      ],
    };
  };

  return {
    defaultHandler,
    uploadAssetHandler,
    uploadArtifactHandler,
    createPullRequestHandler,
    pushToPullRequestBranchHandler,
    pushRepoMemoryHandler,
    createProjectHandler,
    addCommentHandler,
  };
}

module.exports = { createHandlers };
