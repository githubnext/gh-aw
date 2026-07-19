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
const { lookupCheckout } = require("./checkout_manifest.cjs");
const { generateGitPatch } = require("./generate_git_patch.cjs");
const { generateGitBundle } = require("./generate_git_bundle.cjs");
const { hasMergeCommitsInRange, execGitSync } = require("./git_helpers.cjs");
const { enforceCommentLimits } = require("./comment_limit_helpers.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { ERR_CONFIG, ERR_SYSTEM, ERR_VALIDATION } = require("./error_codes.cjs");
const { findRepoCheckout } = require("./find_repo_checkout.cjs");
const { resolveTargetRepoConfig, resolveAndValidateRepo } = require("./repo_helpers.cjs");
const { getOrGenerateTemporaryId } = require("./temporary_id.cjs");
const { parseAllowedExtensionsEnv } = require("./allowed_extensions_helpers.cjs");
const { sanitizeTitle, applyTitlePrefix } = require("./sanitize_title.cjs");
const { parseDeduplicateByTitle, normalizeTitleForDedup, findDuplicateByTitle } = require("./issue_title_dedup.cjs");
const { validateCreatePullRequestIntent, validatePushToPullRequestBranchIntent, validateCreateIssueIntent, validateAddCommentIntent } = require("./intent_probe.cjs");
const { globPatternToRegex } = require("./glob_pattern_helpers.cjs");
const { resolveInvocationContext } = require("./invocation_context_helpers.cjs");
const { lstatGuard } = require("./symlink_guard.cjs");

/** PR event names used for target:triggering context validation across all safe-output handlers. */
const PR_EVENT_NAMES = new Set(["pull_request", "pull_request_target", "pull_request_review", "pull_request_review_comment"]);

/**
 * Resolve effective event name and payload from an invocation context,
 * falling back to the raw GitHub Actions context.
 * @param {ReturnType<typeof resolveInvocationContext> | null | undefined} invocationContext
 * @param {any} rawContext
 */
function resolveEffectiveContext(invocationContext, rawContext) {
  return {
    effectiveEventName: invocationContext?.eventName || rawContext.eventName,
    effectivePayload: invocationContext?.eventPayload || rawContext.payload,
  };
}

/**
 * Read and parse a JSON file.
 * @param {string} filePath
 * @returns {any}
 */
function readJSONFile(filePath) {
  let parsed;
  try {
    parsed = JSON.parse(fs.readFileSync(filePath, "utf8"));
  } catch (err) {
    throw new Error("Failed to parse JSON file " + filePath + ": " + getErrorMessage(err), { cause: err });
  }
  return parsed;
}

const safeOutputsTools = readJSONFile(path.join(__dirname, "safe_outputs_tools.json"));

const safeOutputsToolMap = new Map(safeOutputsTools.map(tool => [tool.name, tool]));

/**
 * @param {string} error
 * @returns {{content: Array<{type: "text", text: string}>, isError: true}}
 */
function buildIntentErrorResponse(error) {
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

/**
 * Build an actionable missing temporary_id error for configured tools.
 * @param {string} toolName
 * @param {string} configKey
 * @returns {string}
 */
function buildMissingTemporaryIdError(toolName, configKey) {
  const temporaryIdExamples = {
    create_pull_request: "aw_pr1",
    create_issue: "aw_issue1",
  };
  const example = temporaryIdExamples[toolName] || "aw_item1";
  return `${toolName} requires 'temporary_id' when safe-outputs.${configKey}.require-temporary-id is enabled. Set temporary_id (for example "${example}") and retry.`;
}

/**
 * @param {Record<string, any>} safeOutputsConfig
 * @param {string} toolName
 * @returns {Record<string, any>}
 */
function getSafeOutputsToolConfig(safeOutputsConfig, toolName) {
  return safeOutputsConfig?.[toolName] || safeOutputsConfig?.[toolName.replace(/_/g, "-")] || {};
}

/**
 * @param {Record<string, any>} entry
 * @param {string[]} fieldNames
 * @returns {boolean}
 */
function hasExplicitTargetParameter(entry, fieldNames) {
  return fieldNames.some(field => entry[field] !== undefined && entry[field] !== null && String(entry[field]).trim() !== "");
}

/**
 * @param {string} toolName
 * @returns {{primary?: string, anyOf?: string[]} | null}
 */
function getWildcardTargetRequirement(toolName) {
  return safeOutputsToolMap.get(toolName)?.["x-safe-outputs-target-requirements"]?.["*"] || null;
}

/**
 * Returns true if `args` contains at least one meaningful field for update_pull_request:
 * a string `title`, a string `body`, or `update_branch === true`.
 * Mirrors the downstream requiresOneOf:title,body,update_branch validation in
 * safe_output_type_validator.cjs (which also excludes field === false from the count).
 * @param {Record<string, any> | null | undefined} args
 * @returns {boolean}
 */
function hasUpdatePullRequestFields(args) {
  const safeArgs = args || {};
  return typeof safeArgs.title === "string" || typeof safeArgs.body === "string" || safeArgs.update_branch === true;
}

/**
 * Parse branch pattern configuration from array or comma-separated string.
 * @param {string[]|string|undefined} value
 * @returns {string[]}
 */
function parseAllowedBranchPatterns(value) {
  if (Array.isArray(value)) {
    return value.map(item => String(item).trim()).filter(Boolean);
  }
  if (typeof value === "string") {
    return value
      .split(",")
      .map(item => item.trim())
      .filter(Boolean);
  }
  return [];
}

/**
 * @param {string} branch
 * @param {string[]} allowedPatterns
 * @returns {boolean}
 */
function isAllowedBranch(branch, allowedPatterns) {
  for (const pattern of allowedPatterns) {
    if (branch === pattern) {
      return true;
    }
    if (pattern === "*") {
      // Add this fast-path
      return true;
    }
    if (pattern.includes("*") && globPatternToRegex(pattern, { pathMode: true, caseSensitive: true }).test(branch)) {
      return true;
    }
  }
  return false;
}

/**
 * Resolve and validate a workspace-relative patch path.
 * @param {string|undefined} workspacePath
 * @returns {{success: true, absolutePath: string} | {success: false, error: string}}
 */
function resolvePatchWorkspacePath(workspacePath) {
  const candidatePath = typeof workspacePath === "string" ? workspacePath.trim() : "";
  if (!candidatePath) {
    return { success: false, error: "patch_workspace_path is empty" };
  }
  const workspaceRoot = path.resolve(process.env.GITHUB_WORKSPACE || process.cwd());
  const resolved = path.resolve(workspaceRoot, candidatePath);
  const relative = path.relative(workspaceRoot, resolved);
  if (relative.startsWith("..") || path.isAbsolute(relative)) {
    return { success: false, error: `Invalid patch_workspace_path '${candidatePath}': path must stay under GITHUB_WORKSPACE` };
  }
  if (!fs.existsSync(resolved)) {
    return { success: false, error: `Invalid patch_workspace_path '${candidatePath}': directory does not exist` };
  }
  if (!fs.statSync(resolved).isDirectory()) {
    return { success: false, error: `Invalid patch_workspace_path '${candidatePath}': path is not a directory` };
  }
  return { success: true, absolutePath: resolved };
}

/**
 * Ensure the current git checkout path is trusted in this process context.
 * Injects safe.directory via GIT_CONFIG_COUNT/KEY/VALUE env vars so that all
 * subsequent git commands in this process inherit the trust without writing to
 * ~/.gitconfig (no persistent global git config side effects).
 *
 * GIT_CONFIG_COUNT/KEY/VALUE is supported since git 2.31.
 *
 * @param {string} gitCwd
 * @param {{ debug: (message: string) => void }} server
 * @returns {void}
 */
function ensureSafeDirectoryTrust(gitCwd, server) {
  if (!gitCwd) return;

  // Check if gitCwd is already present in the injected env-var config to avoid
  // duplicate entries when the handler is called more than once in the same process.
  // The `|| 0` guard converts NaN (from a malformed pre-existing GIT_CONFIG_COUNT) to 0,
  // preventing GIT_CONFIG_KEY_NaN/VALUE_NaN entries that would corrupt the env-var config chain.
  const existingCount = parseInt(process.env.GIT_CONFIG_COUNT || "0", 10) || 0;
  for (let i = 0; i < existingCount; i++) {
    if (process.env[`GIT_CONFIG_KEY_${i}`] === "safe.directory" && process.env[`GIT_CONFIG_VALUE_${i}`] === gitCwd) {
      return;
    }
  }

  const idx = existingCount;
  process.env.GIT_CONFIG_COUNT = String(existingCount + 1);
  process.env[`GIT_CONFIG_KEY_${idx}`] = "safe.directory";
  process.env[`GIT_CONFIG_VALUE_${idx}`] = gitCwd;
  server.debug(`Configured git safe.directory for bridge context: ${gitCwd}`);
}

/**
 * Create handlers for safe output tools
 * @param {Object} server - The MCP server instance for logging
 * @param {Function} appendSafeOutput - Function to append entries to the output file
 * @param {Object} [config] - Optional configuration object with safe output settings
 * @returns {Object} An object containing all handler functions
 */
function createHandlers(server, appendSafeOutput, config = {}) {
  // Ensure the workspace is trusted for the lifetime of this server process.
  // This covers all current and future handlers automatically; per-handler calls
  // additionally cover per-request checkout paths that differ from GITHUB_WORKSPACE.
  ensureSafeDirectoryTrust(process.env.GITHUB_WORKSPACE || process.cwd(), server);

  const TOKEN_THRESHOLD = 16000;

  /**
   * Session-scoped per-type operation counters.
   * Incremented on every successful appendSafeOutput call (MCE4 dual enforcement).
   * @type {Map<string, number>}
   */
  const operationCounts = new Map();

  /**
   * Return the explicitly user-configured max for a safe-output type, or null if not set / unlimited.
   * Uses getSafeOutputsToolConfig for consistent key-normalisation (hyphens → underscores).
   * Does NOT fall back to validation-config defaults: MCP-time enforcement is only
   * applied when the user has explicitly set a limit; downstream enforcement covers defaults.
   * Per Safe Outputs Specification MCE5: the same config source as the processor.
   * @param {string} type - normalised safe-output type name (e.g. "add_comment")
   * @returns {number | null}
   */
  function getExplicitMax(type) {
    const toolConfig = getSafeOutputsToolConfig(config, type);
    if (!toolConfig || typeof toolConfig !== "object") return null;
    if (!("max" in toolConfig)) return null;
    const maxVal = toolConfig.max;
    if (maxVal === -1) return null; // -1 means unlimited
    if (typeof maxVal === "number" && Number.isInteger(maxVal) && maxVal > 0) {
      return maxVal;
    }
    return null;
  }

  /**
   * Enforce the per-type operation count limit at invocation time.
   * Throws a JSON-RPC -32602 error when the configured max has already been reached.
   * Per Safe Outputs Specification MCE4: Dual Enforcement — constraints MUST be
   * enforced at both invocation time (MCP server) and processing time (safe output
   * processor) to provide defence-in-depth.
   * @param {string} type - normalised safe-output type name
   */
  function enforcePerTypeMax(type) {
    const maxAllowed = getExplicitMax(type);
    if (maxAllowed === null) return; // no explicit limit configured
    const current = operationCounts.get(type) || 0;
    if (current >= maxAllowed) {
      throw {
        code: -32602,
        message: `E002: ${type} limit reached — ${current} of ${maxAllowed} already used this run`,
        data: {
          constraint: "max",
          type,
          limit: maxAllowed,
          guidance:
            `You have used all ${maxAllowed} ${type} operations for this run. ` +
            `Further ${type} calls will be ignored. Prioritize the most important items ` +
            `(e.g. consolidate multiple updates into one), or call noop. ` +
            `Note: other safe-output types have independent budgets, so applying one type ` +
            `without its companion type can leave inconsistent state.`,
        },
      };
    }
  }

  /**
   * Append a safe-output entry after enforcing the per-type max count.
   * Increments the session counter only after a successful write, mirroring the
   * approach used by inlineReviewCommentCount so that write errors do not advance
   * the counter.
   * Per Safe Outputs Specification MCE4: invocation-time half of dual enforcement.
   * @param {Record<string, any>} entry
   */
  const appendSafeOutputCounted = entry => {
    const type = entry?.type;
    if (type) enforcePerTypeMax(type);
    appendSafeOutput(entry);
    if (type) operationCounts.set(type, (operationCounts.get(type) || 0) + 1);
  };

  /**
   * Validate schema-declared explicit target parameters for wildcard-target tools.
   * @param {Record<string, any>} entry
   * @returns {{content: Array<{type: "text", text: string}>, isError: true} | null}
   */
  const validateWildcardTargetRequirement = entry => {
    const toolName = entry?.type;
    const requirement = getWildcardTargetRequirement(toolName);
    if (!requirement) {
      return null;
    }

    const toolConfig = getSafeOutputsToolConfig(config, toolName);
    if (toolConfig.target !== "*") {
      return null;
    }

    const anyOf = Array.isArray(requirement.anyOf) ? requirement.anyOf : [];
    if (anyOf.length === 0 || hasExplicitTargetParameter(entry, anyOf)) {
      return null;
    }

    const configKey = toolName.replace(/_/g, "-");
    const primary = requirement.primary || anyOf[0];
    const guidance = anyOf.length === 1 ? primary : `one of: ${anyOf.join(", ")}`;
    return buildIntentErrorResponse(`${toolName} requires ${primary} when safe-outputs.${configKey}.target is '*'. Provide ${guidance} and retry.`);
  };

  /**
   * Detect and offload large string fields to files.
   * @param {Record<string, any>} entry
   * @returns {Object | null} MCP response if large content was handled, else null
   */
  const maybeHandleLargeContent = entry => {
    /** @type {any} */
    let largeContent = null;
    /** @type {any} */
    let largeFieldName = null;

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

    if (!largeContent || !largeFieldName) {
      return null;
    }

    const fileInfo = writeLargeContentToFile(largeContent);
    entry[largeFieldName] = `[Content too large, saved to file: ${fileInfo.filename}]`;
    appendSafeOutputCounted(entry);

    return {
      content: [
        {
          type: "text",
          text: JSON.stringify(fileInfo),
        },
      ],
    };
  };

  /**
   * Default handler for safe output tools
   * Spec cross-reference: Safe Output Outcome Evaluation §2/§4/§5/§6/§7/§8/§9/§10/§11/§12/§13/§14/§15/§16/§18/§19/§20/§21/§22/§23/§24/§25/§26/§27/§28/§29.
   * @param {string} type - The tool type
   * @returns {Function} Handler function
   */
  const defaultHandler = type => args => {
    const entry = { ...(args || {}), type };
    const wildcardTargetValidationError = validateWildcardTargetRequirement(entry);
    if (wildcardTargetValidationError) {
      return wildcardTargetValidationError;
    }
    const largeContentResponse = maybeHandleLargeContent(entry);
    if (largeContentResponse) return largeContentResponse;

    // Normal case - no large content
    appendSafeOutputCounted(entry);
    return {
      content: [
        {
          type: "text",
          text: JSON.stringify({ result: "success" }),
        },
      ],
    };
  };

  const createIssueConfig = config.create_issue || {};
  let deduplicateByTitle = { enabled: false, maxDistance: 0 };
  try {
    deduplicateByTitle = parseDeduplicateByTitle(createIssueConfig.deduplicate_by_title);
  } catch (error) {
    throw new Error(`${ERR_VALIDATION}: ${getErrorMessage(error)}`, { cause: error });
  }
  const createIssueTitlePrefix = createIssueConfig.title_prefix ?? "";
  /** @type {Map<string, Array<{title: string, normalizedTitle: string}>>} */
  const seenIssueTitlesByRepo = new Map();

  /**
   * Handler for upload_asset tool
   * Spec cross-reference: not part of the numbered outcome types in Safe Output Outcome Evaluation v1.0.0.
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
    // Use RUNNER_TEMP so the staged files land on the host filesystem (shared with
    // the artifact-upload step), matching the same pattern used by upload_artifact.
    const assetsDir = path.join(process.env.RUNNER_TEMP || "/tmp", "gh-aw", "safeoutputs", "assets");
    if (!fs.existsSync(assetsDir)) {
      try {
        fs.mkdirSync(assetsDir, { recursive: true });
      } catch (err) {
        throw new Error(`Failed to create directory ${assetsDir}: ${String(err)}`, { cause: err });
      }
    }

    // Read file and compute hash
    let fileContent;
    try {
      fileContent = fs.readFileSync(filePath);
    } catch (err) {
      throw new Error(`Failed to read file ${filePath}: ${String(err)}`, { cause: err });
    }
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

    appendSafeOutputCounted(entry);

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
   * Spec cross-reference: Safe Output Outcome Evaluation §1 (`create_pull_request`).
   * Resolves the current branch if branch is not provided or is the base branch
   * Validates exploratory probe payloads against the resolved effective branch
   * Generates git patch for the changes (unless allow-empty is true)
   * Supports multi-repo scenarios via the optional 'repo' parameter
   */
  const createPullRequestHandler = async args => {
    const entry = { ...args, type: "create_pull_request" };
    if (config.create_pull_request?.require_temporary_id === true && !entry.temporary_id) {
      return buildIntentErrorResponse(buildMissingTemporaryIdError("create_pull_request", "create-pull-request"));
    }

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
    const configuredHeadRepo = typeof prConfig["head-repo"] === "string" ? prConfig["head-repo"].trim() : "";
    entry.head_repo = configuredHeadRepo || repoResult.repo;

    // Determine the working directory for git operations
    // If repo is specified or configured, find where it's checked out
    /** @type {any} */
    let repoCwd = null;
    /** @type {any} */
    let repoSlug = null;
    const patchWorkspacePath = typeof prConfig.patch_workspace_path === "string" ? prConfig.patch_workspace_path.trim() : "";
    const currentCheckoutRepo = typeof prConfig.current_checkout_repo === "string" ? prConfig.current_checkout_repo.trim() : "";
    const patchWorkspaceMatchesTargetRepo = patchWorkspacePath && (!currentCheckoutRepo || currentCheckoutRepo === repoResult.repo);

    if (patchWorkspaceMatchesTargetRepo) {
      const patchWorkspaceResult = resolvePatchWorkspacePath(patchWorkspacePath);
      if (!patchWorkspaceResult.success) {
        return {
          content: [
            {
              type: "text",
              text: JSON.stringify({
                result: "error",
                error: patchWorkspaceResult.error,
              }),
            },
          ],
          isError: true,
        };
      }
      repoCwd = patchWorkspaceResult.absolutePath;
      repoSlug = repoResult.repo;
      server.debug(`Using configured patch_workspace_path for create_pull_request: ${patchWorkspacePath} -> ${repoCwd}`);
    }

    if (((entry.repo && entry.repo.trim()) || prConfig["target-repo"]) && !repoCwd) {
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

    // Get base branch for the resolved target repository.
    // Priority:
    //   1. Explicit `base-branch` from the workflow config (no I/O, no fetch).
    //   2. Checkout manifest written by the workflow's setup phase (no network).
    //   3. Local origin/HEAD metadata + payload/API fallbacks via getBaseBranch.
    let baseBranch;
    const configuredBaseBranch = typeof prConfig.base_branch === "string" ? prConfig.base_branch.trim() : "";
    if (configuredBaseBranch) {
      baseBranch = configuredBaseBranch;
    } else {
      const manifestEntry = lookupCheckout(repoResult.repo);
      if (manifestEntry && manifestEntry.default_branch) {
        baseBranch = manifestEntry.default_branch;
        server.debug(`Using checkout-manifest default_branch for ${repoResult.repo}: ${baseBranch}`);
      } else {
        baseBranch = await getBaseBranch(repoParts, {
          preferLocalDefaultBranchMetadata: Boolean(repoCwd),
          cwd: repoCwd || undefined,
        });
      }
    }

    // Store the resolved base branch in the entry so the apply-time checkout step
    // can use it directly instead of inferring from event context.
    // This makes the safe output "self-describing" and fixes checkout for events
    // like issue_comment on PRs targeting non-default branches.
    entry.base_branch = baseBranch;

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

    // Reject if branch still equals base_branch after detection.
    // This means the base branch was incorrectly resolved (e.g., resolved to the
    // feature branch itself due to a confused event context). Writing a safe output
    // in this state would cause a cryptic git exit-1 in the safe_outputs job when
    // it tries to fetch a non-existent remote ref.
    if (entry.branch === entry.base_branch) {
      return {
        content: [
          {
            type: "text",
            text: JSON.stringify({
              result: "error",
              error: `Branch '${entry.branch}' equals base_branch '${entry.base_branch}'. Cannot create a pull request from a branch into itself. Ensure 'branch' is your feature branch and that the base branch resolves to the target (e.g., 'main' or 'master').`,
            }),
          },
        ],
        isError: true,
      };
    }

    const allowedBranches = parseAllowedBranchPatterns(prConfig.allowed_branches);
    if (allowedBranches.length > 0 && !isAllowedBranch(entry.branch, allowedBranches)) {
      return {
        content: [
          {
            type: "text",
            text: JSON.stringify({
              result: "error",
              error: `Branch '${entry.branch}' does not match allowed-branches. Allowed patterns: ${allowedBranches.join(", ")}`,
            }),
          },
        ],
        isError: true,
      };
    }

    const intentValidationError = validateCreatePullRequestIntent(entry);
    if (intentValidationError) {
      return buildIntentErrorResponse(intentValidationError);
    }

    // Check if allow-empty is enabled in configuration
    const allowEmpty = config.create_pull_request?.allow_empty === true;

    if (allowEmpty) {
      server.debug(`allow-empty is enabled for create_pull_request - skipping patch generation`);
      // Append the safe output entry without generating a patch
      appendSafeOutputCounted(entry);
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

    // Determine transport format: "bundle" (default) uses git bundle (preserves merge topology),
    // "am" uses git format-patch / git am (good for linear histories).
    // Use ?? (nullish coalescing) so an empty-string resolved value is preserved and
    // rejected below rather than silently falling back to "bundle".
    const patchFormat = prConfig["patch_format"] ?? config["patch_format"] ?? "bundle";
    const validPatchFormats = ["am", "bundle"];
    if (!validPatchFormats.includes(patchFormat)) {
      const errorMsg = `Invalid patch_format in configuration. Must be one of: ${validPatchFormats.join(", ")}`;
      server.debug(`create_pull_request: ${errorMsg}`);
      return {
        content: [
          {
            type: "text",
            text: JSON.stringify({
              result: "error",
              error: errorMsg,
            }),
          },
        ],
        isError: true,
      };
    }
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

    // SECURITY: Pin the branch ref to a SHA before generating any transport artifacts.
    // This prevents TOCTOU races where the agent flips the ref between patch and bundle
    // generation, causing the two to represent different commit sets.
    const gitCwd = repoCwd || process.env.GITHUB_WORKSPACE || process.cwd();
    ensureSafeDirectoryTrust(gitCwd, server);
    let pinnedSha;
    try {
      pinnedSha = execGitSync(["rev-parse", "--verify", `refs/heads/${entry.branch}^{commit}`], { cwd: gitCwd })
        .toString()
        .trim();
      server.debug(`Pinned branch '${entry.branch}' to SHA ${pinnedSha}`);
    } catch (pinError) {
      server.debug(`Failed to pin branch '${entry.branch}': ${getErrorMessage(pinError)}`);
      if (useBundle) {
        server.debug(`create_pull_request: proceeding without branch pinning for '${entry.branch}'; bundle generation will fall back to HEAD-based strategies when available`);
      }
      pinnedSha = null;
    }

    // Always generate a patch for policy enforcement (allowed-files/protected-files/excluded-files),
    // even when bundle transport is selected for apply-time commit transport.
    server.debug(`Generating patch for create_pull_request with branch: ${entry.branch}${repoCwd ? ` in ${repoCwd} baseBranch: ${baseBranch}` : ""}`);
    /** @type {Record<string, any>} */
    const patchOptions = { ...transportOptions };
    if (patchWorkspaceMatchesTargetRepo) {
      patchOptions.workspacePath = patchWorkspacePath;
    }
    // Pass excluded_files so git excludes them via :(exclude) pathspecs at generation time.
    if (Array.isArray(prConfig.excluded_files) && prConfig.excluded_files.length > 0) {
      patchOptions.excludedFiles = prConfig.excluded_files;
    }
    // Pass pinnedSha so patch generation uses the pinned commit, not a potentially-flipped ref
    if (pinnedSha) {
      patchOptions.pinnedSha = pinnedSha;
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

    // Patch/bundle paths are not transmitted via the safe-output entry: the
    // privileged safe_outputs job re-derives them from the (validated) branch name
    // using resolve_transport_paths.

    // Store the base commit SHA so the create_pull_request handler can use it
    // directly in the fallback path (the From <sha> header in format-patch output
    // contains the agent's commit SHA which won't exist in the target checkout)
    if (patchResult.baseCommit) {
      entry.base_commit = patchResult.baseCommit;
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

      // SECURITY: Verify the branch ref hasn't been flipped between patch and bundle
      // generation (TOCTOU check). If the SHA changed, the bundle may contain different
      // commits than the patch used for file-protection policy enforcement.
      if (pinnedSha) {
        try {
          const currentSha = execGitSync(["rev-parse", "--verify", `refs/heads/${entry.branch}^{commit}`], { cwd: gitCwd })
            .toString()
            .trim();
          if (currentSha !== pinnedSha) {
            server.debug(`SECURITY: Branch '${entry.branch}' SHA changed during transport generation (was ${pinnedSha}, now ${currentSha}). Aborting.`);
            return {
              content: [
                {
                  type: "text",
                  text: JSON.stringify({
                    result: "error",
                    error: "Branch ref changed during transport artifact generation. This may indicate a concurrent modification. Please retry.",
                    details: `Branch '${entry.branch}' pointed to ${pinnedSha} at start but ${currentSha} after bundle generation.`,
                  }),
                },
              ],
              isError: true,
            };
          }
        } catch (verifyError) {
          server.debug(`SECURITY: Failed to verify branch SHA after bundle generation: ${getErrorMessage(verifyError)}`);
          return {
            content: [
              {
                type: "text",
                text: JSON.stringify({
                  result: "error",
                  error: `Failed to verify branch integrity after bundle generation: ${getErrorMessage(verifyError)}`,
                }),
              },
            ],
            isError: true,
          };
        }
      }

      // Bundle path is not transmitted via the safe-output entry: the privileged
      // safe_outputs job re-derives it from the (validated) branch name using
      // resolve_transport_paths.

      // Prefer the base_commit captured from format-patch generation (used by
      // patch-based fallback/apply paths). Only fall back to bundle base commit
      // when patch generation did not record one.
      if (!entry.base_commit && bundleResult.baseCommit) {
        entry.base_commit = bundleResult.baseCommit;
      }

      appendSafeOutputCounted(entry);
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
              bundle: {
                path: bundleResult.bundlePath,
                size: bundleResult.bundleSize,
              },
            }),
          },
        ],
      };
    }

    appendSafeOutputCounted(entry);
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
   * Spec cross-reference: Safe Output Outcome Evaluation §17 (`push_to_pull_request_branch`).
   * The agent SHOULD supply a `branch` argument identifying the local branch it
   * committed onto. This is required for batch workflows that loop over multiple PRs
   * and checkout different branches; without it, the source branch would be inferred
   * from the current git HEAD which may not match the PR being processed. When
   * `branch` is omitted, it is derived from the current checkout as a fallback.
   * The destination branch is independently derived by the apply-time push handler
   * from pulls.get(pull_number).head.ref.
   *
   * Note: Fork PR detection is handled by push_to_pull_request_branch.cjs handler
   * which fetches the PR and calls detectForkPR() with full PR data.
   */
  const pushToPullRequestBranchHandler = async args => {
    const entry = { ...(args || {}), type: "push_to_pull_request_branch" };
    const wildcardTargetValidationError = validateWildcardTargetRequirement(entry);
    if (wildcardTargetValidationError) {
      return wildcardTargetValidationError;
    }

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
    const configuredHeadRepo = typeof pushConfig["head-repo"] === "string" ? pushConfig["head-repo"].trim() : "";
    entry.head_repo = configuredHeadRepo || repoResult.repo;

    // Determine the working directory for git operations.
    // Look up the checkout path when the target repo is explicitly provided by the agent
    // or explicitly configured via target-repo in the workflow config — this ensures patch
    // generation runs from the correct directory when the target repo is checked out in a subdirectory.
    /** @type {any} */
    let repoCwd = null;
    const itemRepo = repoResult.repo;
    const pushPatchWorkspacePath = typeof pushConfig.patch_workspace_path === "string" ? pushConfig.patch_workspace_path.trim() : "";
    const pushCurrentCheckoutRepo = typeof pushConfig.current_checkout_repo === "string" ? pushConfig.current_checkout_repo.trim() : "";
    const pushPatchWorkspaceMatchesTargetRepo = pushPatchWorkspacePath && (!pushCurrentCheckoutRepo || pushCurrentCheckoutRepo === itemRepo);

    if (pushPatchWorkspaceMatchesTargetRepo) {
      const patchWorkspaceResult = resolvePatchWorkspacePath(pushPatchWorkspacePath);
      if (!patchWorkspaceResult.success) {
        return {
          content: [
            {
              type: "text",
              text: JSON.stringify({
                result: "error",
                error: patchWorkspaceResult.error,
              }),
            },
          ],
          isError: true,
        };
      }
      repoCwd = patchWorkspaceResult.absolutePath;
      entry.repo_cwd = repoCwd;
      server.debug(`Using configured patch_workspace_path for push_to_pull_request_branch: ${pushPatchWorkspacePath} -> ${repoCwd}`);
    }

    if (((entry.repo && entry.repo.trim()) || pushConfig["target-repo"]) && !repoCwd) {
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

    // Get base branch for the resolved target repository.
    // Priority:
    //   1. Explicit `base-branch` from the workflow config (no I/O, no fetch).
    //   2. Checkout manifest written by the workflow's setup phase (no network).
    //   3. Local origin/HEAD metadata in the side-repo checkout (when available).
    //   4. Payload / GitHub API fallbacks via getBaseBranch.
    let baseBranch;
    const configuredBaseBranch = typeof pushConfig.base_branch === "string" ? pushConfig.base_branch.trim() : "";
    if (configuredBaseBranch) {
      baseBranch = configuredBaseBranch;
      server.debug(`Using configured base_branch for push_to_pull_request_branch: ${baseBranch}`);
    } else {
      const manifestEntry = lookupCheckout(itemRepo);
      if (manifestEntry && manifestEntry.default_branch) {
        baseBranch = manifestEntry.default_branch;
        server.debug(`Using checkout-manifest default_branch for ${itemRepo}: ${baseBranch}`);
      } else {
        baseBranch = await getBaseBranch(repoParts, {
          preferLocalDefaultBranchMetadata: Boolean(repoCwd),
          cwd: repoCwd || undefined,
        });
      }
    }

    // Store the resolved base branch in the entry so the apply-time checkout step
    // can use it directly instead of inferring from event context.
    // This makes the safe output "self-describing" and fixes checkout for events
    // like issue_comment on PRs targeting non-default branches.
    entry.base_branch = baseBranch;

    // Use the agent-supplied branch if provided; fall back to the current checkout.
    // The agent-supplied value is authoritative: in multi-PR batch workflows the
    // working tree may be checked out to a different PR's branch by the time the
    // MCP handler runs, so relying solely on HEAD can produce a wrong source ref
    // and cause the apply step to fail with "couldn't find remote ref".
    // The apply-time push job re-derives the destination from pulls.get(pull_number)
    // independently, so this branch name is used only as the source ref for the
    // incremental diff against origin/<branch>.
    if (entry.branch && typeof entry.branch === "string" && entry.branch.trim()) {
      entry.branch = entry.branch.trim();
      server.debug(`Using agent-supplied branch for push_to_pull_request_branch: ${entry.branch}`);
    } else {
      // Fallback: derive from the current checkout (backward compat for single-PR workflows
      // where the working tree is reliably on the PR head ref at call time).
      try {
        const detectedBranch = getCurrentBranch(repoCwd);
        server.debug(`Using current branch for push_to_pull_request_branch: ${detectedBranch}`);
        entry.branch = detectedBranch;
      } catch (branchErr) {
        return {
          content: [
            {
              type: "text",
              text: JSON.stringify({
                result: "error",
                error: `Failed to determine source branch for push_to_pull_request_branch: ${getErrorMessage(branchErr)}. Either supply a 'branch' argument explicitly or ensure the working tree is checked out to the pull request's head ref before calling this tool.`,
              }),
            },
          ],
          isError: true,
        };
      }
    }

    // Reject if the detected branch equals base_branch. This means the workspace
    // is checked out on the PR's base (e.g. main) rather than the PR's head ref,
    // so there is nothing to push. Writing a safe output in this state would
    // cause a cryptic git exit-1 in the safe_outputs job when it tries to fetch
    // a non-existent remote ref.
    if (entry.branch === entry.base_branch) {
      return {
        content: [
          {
            type: "text",
            text: JSON.stringify({
              result: "error",
              error: `Detected branch '${entry.branch}' equals base_branch '${entry.base_branch}'. The workspace is checked out on the base branch, not the pull request's head branch — there is nothing to push. Check out the PR's head ref and commit your changes there before calling push_to_pull_request_branch.`,
            }),
          },
        ],
        isError: true,
      };
    }

    const intentValidationError = validatePushToPullRequestBranchIntent(entry);
    if (intentValidationError) {
      return buildIntentErrorResponse(intentValidationError);
    }

    // Determine transport format: "bundle" (default) uses git bundle (preserves merge topology),
    // "am" uses git format-patch / git am (good for linear histories).
    // Use ?? (nullish coalescing) so an empty-string resolved value is preserved and
    // rejected below rather than silently falling back to "bundle".
    // Track whether the user explicitly set patch_format so we can auto-fall-back
    // to bundle transport when merge commits are detected (since `git am` cannot
    // apply merge commits). When the user explicitly chose a format, respect it.
    const patchFormatExplicit = pushConfig["patch_format"] !== undefined || config["patch_format"] !== undefined;
    const pushPatchFormat = pushConfig["patch_format"] ?? config["patch_format"] ?? "bundle";
    const validPushPatchFormats = ["am", "bundle"];
    if (!validPushPatchFormats.includes(pushPatchFormat)) {
      const errorMsg = `Invalid patch_format in configuration. Must be one of: ${validPushPatchFormats.join(", ")}`;
      server.debug(`push_to_pull_request_branch: ${errorMsg}`);
      return {
        content: [
          {
            type: "text",
            text: JSON.stringify({
              result: "error",
              error: errorMsg,
            }),
          },
        ],
        isError: true,
      };
    }
    let useBundle = pushPatchFormat === "bundle";

    // Auto-fallback: when patch_format is not explicitly configured and the
    // incremental range (origin/<branch>..<branch>) contains merge commits,
    // automatically switch to bundle transport. `git am` cannot
    // apply merge commits, so without this fallback long-running branches that
    // periodically merge their base branch locally would fail with add/add
    // conflicts on every push attempt. The detection is best-effort and uses
    // only local refs (no extra fetch); a detection miss simply preserves the
    // existing behavior.
    if (!useBundle && !patchFormatExplicit && entry.branch) {
      const hasMerges = hasMergeCommitsInRange(`refs/remotes/origin/${entry.branch}`, entry.branch, { cwd: repoCwd || undefined });
      if (hasMerges) {
        server.debug(`push_to_pull_request_branch: detected merge commit(s) in incremental range origin/${entry.branch}..${entry.branch}; auto-switching to bundle transport (set patch-format: am to override).`);
        useBundle = true;
      }
    }

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

    // SECURITY: Pin the branch ref to a SHA before generating any transport artifacts.
    // This prevents TOCTOU races where the agent flips the ref between patch and bundle
    // generation, causing the two to represent different commit sets.
    const pushGitCwd = repoCwd || process.env.GITHUB_WORKSPACE || process.cwd();
    ensureSafeDirectoryTrust(pushGitCwd, server);
    let pushPinnedSha;
    try {
      pushPinnedSha = execGitSync(["rev-parse", "--verify", `refs/heads/${entry.branch}^{commit}`], { cwd: pushGitCwd })
        .toString()
        .trim();
      server.debug(`Pinned branch '${entry.branch}' to SHA ${pushPinnedSha}`);
    } catch (pinError) {
      server.debug(`Failed to pin branch '${entry.branch}': ${getErrorMessage(pinError)}`);
      if (useBundle) {
        return {
          content: [
            {
              type: "text",
              text: JSON.stringify({
                result: "error",
                error: `Failed to pin branch '${entry.branch}' before bundle generation: ${getErrorMessage(pinError)}`,
                details: `Bundle transport requires branch pinning to prevent patch/bundle desynchronization. Retry after ensuring the branch exists locally (for example: git branch --list '${entry.branch}').`,
              }),
            },
          ],
          isError: true,
        };
      }
      pushPinnedSha = null;
    }

    // Full-branch allowed_files check: validate that ALL commits on the PR branch
    // (relative to origin/baseBranch) only touch files permitted by allowed_files.
    // The incremental patch check at apply-time only inspects the net diff between
    // origin/<branch> and the local branch tip; this catches disallowed files that
    // appear in earlier commits on the branch (e.g. a Copilot branch that also
    // modified .github/workflows/**) and returns an actionable error to the agent
    // before any transport artifacts are generated.
    if (Array.isArray(pushConfig.allowed_files) && pushConfig.allowed_files.length > 0) {
      try {
        // Use the pinned SHA as the range head to avoid any TOCTOU window between
        // the time the SHA was recorded and the time of the git log query.  If no
        // pinned SHA is available (e.g. non-bundle path), skip the check so we do
        // not race against a mutable ref; the apply-time check still enforces policy.
        if (!pushPinnedSha) {
          server.debug("Full-branch allowed-files check skipped: branch SHA not pinned (non-bundle path)");
        } else {
          const branchHistoryFiles = execGitSync(["log", "--name-only", "--pretty=format:", `origin/${baseBranch}..${pushPinnedSha}`, "--"], { cwd: pushGitCwd })
            .toString()
            .split("\n")
            .map(s => s.trim())
            .filter(Boolean);

          if (branchHistoryFiles.length > 0) {
            const allowedPatterns = pushConfig.allowed_files.map(p => globPatternToRegex(p));
            // Files matching excluded_files are intentionally exempt: they will be stripped
            // from the patch at generation time via :(exclude) pathspecs, so they won't be
            // present in the final changeset applied to the branch.
            const excludedPatterns = Array.isArray(pushConfig.excluded_files) ? pushConfig.excluded_files.map(p => globPatternToRegex(p)) : [];
            const uniqueFiles = [...new Set(branchHistoryFiles)];
            const disallowedFiles = uniqueFiles.filter(f => !allowedPatterns.some(re => re.test(f)) && !excludedPatterns.some(re => re.test(f)));

            if (disallowedFiles.length > 0) {
              const sample = disallowedFiles.slice(0, 5);
              const remaining = disallowedFiles.length - sample.length;
              const filesStr = remaining > 0 ? `${sample.join(", ")} (+${remaining} more)` : sample.join(", ");
              server.debug(`Full-branch allowed-files check failed: ${filesStr}`);
              return {
                content: [
                  {
                    type: "text",
                    text: JSON.stringify({
                      result: "error",
                      error: `Cannot push to pull request branch: the branch '${entry.branch}' history contains commits that modify files outside the allowed-files configuration: ${filesStr}. Remove the disallowed file changes from your commits and retry, or update the allowed-files configuration to include these files.`,
                      disallowed_files: disallowedFiles,
                    }),
                  },
                ],
                isError: true,
              };
            }
          }
        }
      } catch (fullBranchCheckError) {
        // Non-fatal: if origin/baseBranch is not available locally or git fails,
        // skip the full-branch check and continue.  The apply-time policy check in
        // push_to_pull_request_branch.cjs will still enforce allowed_files against
        // the incremental patch content.
        server.debug(`Full-branch allowed-files check skipped (non-fatal): ${getErrorMessage(fullBranchCheckError)}`);
      }
    }

    // Always generate an incremental patch for policy enforcement (allowed-files/protected-files/excluded-files),
    // even when bundle transport is selected for apply-time commit transport.
    server.debug(`Generating incremental patch for push_to_pull_request_branch with branch: ${entry.branch}, baseBranch: ${baseBranch}`);
    /** @type {Record<string, any>} */
    const pushPatchOptions = { ...pushTransportOptions };
    if (pushPatchWorkspaceMatchesTargetRepo) {
      pushPatchOptions.workspacePath = pushPatchWorkspacePath;
    }
    // Pass excluded_files so git excludes them via :(exclude) pathspecs at generation time.
    if (Array.isArray(pushConfig.excluded_files) && pushConfig.excluded_files.length > 0) {
      pushPatchOptions.excludedFiles = pushConfig.excluded_files;
    }
    // Pass pinnedSha so patch generation uses the pinned commit, not a potentially-flipped ref
    if (pushPinnedSha) {
      pushPatchOptions.pinnedSha = pushPinnedSha;
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

    // Patch/bundle paths are not transmitted via the safe-output entry: the
    // privileged safe_outputs job re-derives them from the (validated) branch name
    // using resolve_transport_paths.

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

      // SECURITY: Verify the branch ref hasn't been flipped between patch and bundle
      // generation (TOCTOU check).
      if (pushPinnedSha) {
        try {
          const currentSha = execGitSync(["rev-parse", "--verify", `refs/heads/${entry.branch}^{commit}`], { cwd: pushGitCwd })
            .toString()
            .trim();
          if (currentSha !== pushPinnedSha) {
            server.debug(`SECURITY: Branch '${entry.branch}' SHA changed during transport generation (was ${pushPinnedSha}, now ${currentSha}). Aborting.`);
            return {
              content: [
                {
                  type: "text",
                  text: JSON.stringify({
                    result: "error",
                    error: "Branch ref changed during transport artifact generation. This may indicate a concurrent modification. Please retry.",
                    details: `Branch '${entry.branch}' pointed to ${pushPinnedSha} at start but ${currentSha} after bundle generation.`,
                  }),
                },
              ],
              isError: true,
            };
          }
        } catch (verifyError) {
          server.debug(`SECURITY: Failed to verify branch SHA after bundle generation: ${getErrorMessage(verifyError)}`);
          return {
            content: [
              {
                type: "text",
                text: JSON.stringify({
                  result: "error",
                  error: `Failed to verify branch integrity after bundle generation: ${getErrorMessage(verifyError)}`,
                }),
              },
            ],
            isError: true,
          };
        }
      }

      // Bundle path is not transmitted via the safe-output entry: the privileged
      // safe_outputs job re-derives it from the (validated) branch name using
      // resolve_transport_paths.

      // Prefer the base_commit captured from format-patch generation (used by
      // patch-based fallback/apply paths). Only fall back to bundle base commit
      // when patch generation did not record one.
      if (!entry.base_commit && bundleResult.baseCommit) {
        entry.base_commit = bundleResult.baseCommit;
      }

      appendSafeOutputCounted(entry);
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
              bundle: {
                path: bundleResult.bundlePath,
                size: bundleResult.bundleSize,
              },
            }),
          },
        ],
      };
    }

    appendSafeOutputCounted(entry);
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
   * Spec cross-reference: not part of the numbered outcome types in Safe Output Outcome Evaluation v1.0.0.
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
    // The effective limit is max_patch_size × 1.2, matching the push gate in push_repo_memory.cjs.
    // This catches cases where total memory content is close to or exceeds the push diff limit.
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

    // Check total content size. totalSize is the raw sum of all file sizes in the memory
    // directory (excluding .git). It is compared against max_patch_size × 1.2 so that
    // a full rewrite of all memory content would remain within the push gate limit in
    // push_repo_memory.cjs, which applies the same 1.2× factor to diff additions.
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
                `(configured max-patch-size: ${Math.floor(maxPatchSize / 1024)} KB).\n\n` +
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
            message: `Memory validation passed: ${files.length} file(s), ${totalSizeKb} KB total (limit: ${effectiveMaxKb} KB).`,
          }),
        },
      ],
    };
  };

  /**
   * Handler for create_issue tool
   * Applies title-based within-run deduplication for immediate feedback.
   */
  const createIssueHandler = args => {
    const entry = { ...(args || {}), type: "create_issue" };
    if (createIssueConfig.require_temporary_id === true && !entry.temporary_id) {
      return buildIntentErrorResponse(buildMissingTemporaryIdError("create_issue", "create-issue"));
    }
    const intentValidationError = validateCreateIssueIntent(entry);
    if (intentValidationError) {
      return buildIntentErrorResponse(intentValidationError);
    }

    const { defaultTargetRepo, allowedRepos } = resolveTargetRepoConfig(createIssueConfig);
    const repoResult = resolveAndValidateRepo(entry, defaultTargetRepo, allowedRepos, "issue");
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
    const resolvedRepo = repoResult.repo;

    let resolvedTitle = entry.title?.trim() || "";
    if (!resolvedTitle) {
      resolvedTitle = entry.body?.trim() || "Agent Output";
    }
    resolvedTitle = applyTitlePrefix(sanitizeTitle(resolvedTitle, createIssueTitlePrefix), createIssueTitlePrefix);

    if (deduplicateByTitle.enabled) {
      const normalizedTitle = normalizeTitleForDedup(resolvedTitle);
      const seenTitles = seenIssueTitlesByRepo.get(resolvedRepo) || [];
      const duplicate = findDuplicateByTitle(normalizedTitle, seenTitles, deduplicateByTitle.maxDistance);
      if (duplicate) {
        const droppedEntry = {
          ...entry,
          _dropped_duplicate_by_title: true,
          _dedup_source: "mcp-within-run",
          _duplicate_title: duplicate.title,
          _duplicate_distance: duplicate.distance,
        };
        const largeContentResponse = maybeHandleLargeContent(droppedEntry);
        if (!largeContentResponse) {
          appendSafeOutputCounted(droppedEntry);
        }
        return {
          content: [
            {
              type: "text",
              text: JSON.stringify({
                result: "duplicate_dropped",
                reason: `Duplicate create_issue title matched "${duplicate.title}" (distance=${duplicate.distance})`,
              }),
            },
          ],
        };
      }
      seenTitles.push({ title: resolvedTitle, normalizedTitle });
      seenIssueTitlesByRepo.set(resolvedRepo, seenTitles);
    }

    const largeContentResponse = maybeHandleLargeContent(entry);
    if (largeContentResponse) return largeContentResponse;

    appendSafeOutputCounted(entry);
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
   * Handler for create_project tool
   * Spec cross-reference: not part of the numbered outcome types in Safe Output Outcome Evaluation v1.0.0.
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
    appendSafeOutputCounted(entry);

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
   * Spec cross-reference: Safe Output Outcome Evaluation §3 (`add_comment`).
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

    // Refuse discussion-specific requests when discussions are not enabled in config.
    // reply_to_id is a discussion-only field; its presence unambiguously means the
    // agent is targeting a GitHub Discussion.  Guard here (MCP phase) so the agent
    // gets immediate, actionable feedback rather than a late failure at execution time.
    const addCommentConfig = getSafeOutputsToolConfig(config, "add_comment");
    const discussionsEnabled = addCommentConfig.discussions === true;
    const hasReplyToId = args?.reply_to_id != null && String(args.reply_to_id).trim() !== "";
    if (hasReplyToId && !discussionsEnabled) {
      return buildIntentErrorResponse(
        "add_comment with reply_to_id targets a GitHub Discussion, but discussion comments are not enabled for this workflow. " +
          "Set 'discussions: true' in the workflow's safe-outputs.add-comment configuration to enable discussion comments and request discussions:write permission."
      );
    }

    // Reject target:triggering early when no explicit item number and no issue/PR/discussion context.
    // Per Safe Outputs Specification MCE1: provides actionable feedback before writing to NDJSON.
    // Mirrors update_issue validation; explicit item_number bypasses this check because the
    // downstream handler resolves explicit numbers before falling back to triggering context.
    const effectiveAddCommentTarget = addCommentConfig.target || "triggering";
    const hasExplicitItemNumber = args?.item_number != null || args?.issue_number != null || args?.["pr-number"] != null;
    if (effectiveAddCommentTarget === "triggering" && !hasExplicitItemNumber) {
      /** @type {any} */
      let invocationContext = null;
      try {
        invocationContext = resolveInvocationContext(context);
      } catch (err) {
        // A validation error (e.g. disallowed target_repo / SEC-005) is a real failure — surface it.
        const errMsg = getErrorMessage(err);
        if (errMsg.startsWith(ERR_VALIDATION)) {
          return buildIntentErrorResponse(errMsg);
        }
        // Unexpected structural error: skip validation and let downstream handle gracefully.
      }
      if (invocationContext != null) {
        const { effectiveEventName, effectivePayload } = resolveEffectiveContext(invocationContext, context);
        const isIssueCommentOnPR = effectiveEventName === "issue_comment" && Boolean(effectivePayload?.issue?.pull_request);
        const isIssueContext = effectiveEventName === "issues" || (effectiveEventName === "issue_comment" && !isIssueCommentOnPR);
        const isPRContext = PR_EVENT_NAMES.has(effectiveEventName) || isIssueCommentOnPR;
        const isDiscussionContext = effectiveEventName === "discussion" || effectiveEventName === "discussion_comment";
        if (!isIssueContext && !isPRContext && !isDiscussionContext) {
          return buildIntentErrorResponse(
            `add_comment requires an issue, pull request, or discussion context but the workflow is running on a "${effectiveEventName}" event. ` +
              `The add-comment handler uses target: triggering which only applies when an issue, pull request, or discussion triggered the workflow. ` +
              `To report results from this workflow, use create_discussion or create_issue instead. ` +
              `If you need to comment on a specific item, provide an explicit item_number.`
          );
        }
      }
    }

    // Build the entry with a temporary_id
    const entry = { ...(args || {}), type: "add_comment" };
    const wildcardTargetValidationError = validateWildcardTargetRequirement(entry);
    if (wildcardTargetValidationError) {
      return wildcardTargetValidationError;
    }
    const intentValidationError = validateAddCommentIntent(entry);
    if (intentValidationError) {
      return buildIntentErrorResponse(intentValidationError);
    }

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
    appendSafeOutputCounted(entry);

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
   * Session-scoped counter for buffered inline review comments.
   * Incremented by createPullRequestReviewCommentHandler, read by submitPullRequestReviewHandler
   * to guard against empty review submissions at the MCP server phase.
   */
  let inlineReviewCommentCount = 0;

  /**
   * Handler for create_pull_request_review_comment tool (MCP server phase).
   * Increments the session-scoped inline comment counter so that the subsequent
   * submitPullRequestReviewHandler can detect an otherwise-empty review.
   * Per Safe Outputs Specification MCE1: enforces constraints during tool invocation
   * to provide immediate feedback to the LLM before recording to NDJSON.
   */
  const createPullRequestReviewCommentHandler = args => {
    const result = defaultHandler("create_pull_request_review_comment")(args);
    // Increment only after the default handler returns successfully; if it throws
    // (e.g. due to large-content rejection or an append write error) the counter
    // must not advance so the empty-review guard remains accurate.
    if (!result?.isError) {
      inlineReviewCommentCount++;
    }
    return result;
  };

  /**
   * Handler for submit_pull_request_review tool (MCP server phase).
   * Validates the review before writing it to the NDJSON output so that the agent
   * receives an immediate MCP error rather than a silent 422 at finalization time.
   *
   * Checks performed:
   *  1. REQUEST_CHANGES requires a non-empty body (GitHub API requirement).
   *  2. If the review body is empty AND no inline comments were buffered during this
   *     session, the review would be contentless and GitHub would return 422 — reject
   *     early (mirrors Sub-pattern A guard in pr_review_buffer.cjs).
   *
   * Per Safe Outputs Specification MCE1: enforces constraints during tool invocation
   * to provide immediate feedback to the LLM before recording to NDJSON.
   */
  const submitPullRequestReviewHandler = args => {
    const body = (args && typeof args.body === "string" ? args.body : "").trim();
    const event = args && args.event ? String(args.event).toUpperCase() : "COMMENT";

    const VALID_REVIEW_EVENTS = ["APPROVE", "REQUEST_CHANGES", "COMMENT"];
    if (!VALID_REVIEW_EVENTS.includes(event)) {
      throw {
        code: -32602,
        message: `${ERR_VALIDATION}: submit_pull_request_review: invalid event '${args.event}'. Must be one of: ${VALID_REVIEW_EVENTS.join(", ")}`,
      };
    }

    if (event === "REQUEST_CHANGES" && !body) {
      throw {
        code: -32602,
        message: `${ERR_VALIDATION}: submit_pull_request_review: 'body' is required when event is REQUEST_CHANGES`,
      };
    }

    if (!body && inlineReviewCommentCount === 0) {
      throw {
        code: -32602,
        message:
          `${ERR_VALIDATION}: submit_pull_request_review: review body is empty and no ` +
          `create_pull_request_review_comment calls were made — GitHub would return 422 for a contentless review. ` +
          `Provide a non-empty 'body' or call create_pull_request_review_comment before submitting.`,
      };
    }

    // Reset the counter after a successful review submission so that subsequent
    // reviews in the same MCP session start with a clean slate.
    inlineReviewCommentCount = 0;

    return defaultHandler("submit_pull_request_review")(args);
  };

  /**
   * Handler for dismiss_pull_request_review tool (MCP server phase).
   * Enforces justification minimum length and actor-author consistency before recording.
   */
  const dismissPullRequestReviewHandler = args => {
    const justification = (args && typeof args.justification === "string" ? args.justification : "").trim();
    if (justification.length < 20) {
      throw {
        code: -32602,
        message: `${ERR_VALIDATION}: dismiss_pull_request_review: 'justification' must be at least 20 characters`,
      };
    }

    const actor = (process.env.GITHUB_ACTOR || context?.actor || "github-actions[bot]").trim() || "github-actions[bot]";
    const author = args && typeof args.author === "string" ? args.author.trim() : "";
    if (author && author !== actor) {
      throw {
        code: -32602,
        message: `${ERR_VALIDATION}: dismiss_pull_request_review: 'author' must match current workflow actor (${actor})`,
      };
    }

    return defaultHandler("dismiss_pull_request_review")(args);
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
      try {
        fs.mkdirSync(destDir, { recursive: true });
      } catch (err) {
        throw new Error(`Failed to create directory ${destDir}: ${String(err)}`, { cause: err });
      }
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
   * Spec cross-reference: not part of the numbered outcome types in Safe Output Outcome Evaluation v1.0.0.
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

      const stat = lstatGuard(filePath);
      if (stat === null) {
        throw {
          code: -32602,
          message: `${ERR_VALIDATION}: upload_artifact: symlinks are not allowed: ${filePath}`,
        };
      }

      const stagingDir = path.join(process.env.RUNNER_TEMP || "/tmp", "gh-aw", "safeoutputs", "upload-artifacts");
      if (!fs.existsSync(stagingDir)) {
        try {
          fs.mkdirSync(stagingDir, { recursive: true });
        } catch (err) {
          throw new Error(`Failed to create directory ${stagingDir}: ${String(err)}`, { cause: err });
        }
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

    appendSafeOutputCounted(entry);

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

  /**
   * Handler for update_issue tool
   * Spec cross-reference: Safe Output Outcome Evaluation §update_issue.
   * Per Safe Outputs Specification MCE1: Enforces context constraints during tool invocation
   * to provide immediate feedback to the LLM before recording to NDJSON.
   * Rejects `target: triggering` (the default) when the workflow has no issue context
   * (e.g. on schedule or push events), so the agent receives an actionable error
   * instead of a downstream Process Safe Outputs failure.
   */
  const updateIssueHandler = args => {
    const updateIssueConfig = getSafeOutputsToolConfig(config, "update_issue");
    const effectiveTarget = updateIssueConfig.target || "triggering";

    if (effectiveTarget === "triggering") {
      /** @type {any} */
      let invocationContext = null;
      try {
        invocationContext = resolveInvocationContext(context);
      } catch (err) {
        // A validation error (e.g. disallowed target_repo / SEC-005) is a real failure — surface it.
        const errMsg = getErrorMessage(err);
        if (errMsg.startsWith(ERR_VALIDATION)) {
          return buildIntentErrorResponse(errMsg);
        }
        // Unexpected structural error: skip validation and let downstream handle gracefully.
      }
      if (invocationContext != null) {
        const { effectiveEventName, effectivePayload } = resolveEffectiveContext(invocationContext, context);
        const isIssueCommentOnPR = effectiveEventName === "issue_comment" && Boolean(effectivePayload?.issue?.pull_request);
        const isIssueContext = effectiveEventName === "issues" || (effectiveEventName === "issue_comment" && !isIssueCommentOnPR);

        if (!isIssueContext) {
          return buildIntentErrorResponse(
            `update_issue requires an issue context but the workflow is running on a "${effectiveEventName}" event. ` +
              `The update-issue handler uses target: triggering which only applies when an issue triggered the workflow. ` +
              `To report results from this workflow, use create_discussion or create_issue instead. ` +
              `If you need to update a specific issue, the workflow must configure update-issue: target: '*' and you must supply issue_number.`
          );
        }
      }
    }

    return defaultHandler("update_issue")(args || {});
  };

  /**
   * Handler for update_pull_request tool
   * Spec cross-reference: Safe Output Outcome Evaluation §update_pull_request.
   * Per Safe Outputs Specification MCE1: Enforces constraints during tool invocation
   * to provide immediate feedback to the LLM before recording to NDJSON.
   * Uses hasUpdatePullRequestFields to validate that at least one of 'title', 'body',
   * or 'update_branch' is provided before recording to NDJSON.
   * Rejects `target: triggering` (the default) when the workflow has no pull request context
   * (e.g. on schedule or push events), so the agent receives an actionable error
   * instead of a downstream Process Safe Outputs failure.
   */
  const updatePullRequestHandler = args => {
    if (!hasUpdatePullRequestFields(args)) {
      throw {
        code: -32602,
        message: `${ERR_VALIDATION}: update_pull_request requires at least one of: 'title', 'body', 'update_branch' fields`,
      };
    }

    const updatePRConfig = getSafeOutputsToolConfig(config, "update_pull_request");
    const effectivePRTarget = updatePRConfig.target || "triggering";
    if (effectivePRTarget === "triggering") {
      /** @type {any} */
      let invocationContext = null;
      try {
        invocationContext = resolveInvocationContext(context);
      } catch (err) {
        // A validation error (e.g. disallowed target_repo / SEC-005) is a real failure — surface it.
        const errMsg = getErrorMessage(err);
        if (errMsg.startsWith(ERR_VALIDATION)) {
          return buildIntentErrorResponse(errMsg);
        }
        // Unexpected structural error: skip validation and let downstream handle gracefully.
      }
      if (invocationContext != null) {
        const { effectiveEventName, effectivePayload } = resolveEffectiveContext(invocationContext, context);
        const isIssueCommentOnPR = effectiveEventName === "issue_comment" && Boolean(effectivePayload?.issue?.pull_request);
        const isPRContext = PR_EVENT_NAMES.has(effectiveEventName) || isIssueCommentOnPR;

        if (!isPRContext) {
          return buildIntentErrorResponse(
            `update_pull_request requires a pull request context but the workflow is running on a "${effectiveEventName}" event. ` +
              `The update-pull-request handler uses target: triggering which only applies when a pull request triggered the workflow. ` +
              `To report results from this workflow, use create_discussion or create_issue instead. ` +
              `If you need to update a specific pull request, the workflow must configure update-pull-request: target: '*' and you must supply pull_request_number.`
          );
        }
      }
    }

    return defaultHandler("update_pull_request")(args || {});
  };

  return {
    defaultHandler,
    uploadAssetHandler,
    uploadArtifactHandler,
    createPullRequestHandler,
    pushToPullRequestBranchHandler,
    pushRepoMemoryHandler,
    createIssueHandler,
    createProjectHandler,
    addCommentHandler,
    createPullRequestReviewCommentHandler,
    submitPullRequestReviewHandler,
    dismissPullRequestReviewHandler,
    updateIssueHandler,
    updatePullRequestHandler,
  };
}

module.exports = {
  buildIntentErrorResponse,
  createHandlers,
  hasUpdatePullRequestFields,
};
