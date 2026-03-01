// @ts-check

const fs = require("fs");
const path = require("path");
const { execGitSync } = require("./git_helpers.cjs");

/**
 * Debug logging helper - logs to stderr when DEBUG env var matches
 * @param {string} message - Debug message to log
 */
function debugLog(message) {
  const debug = process.env.DEBUG || "";
  if (debug === "*" || debug.includes("find_repo_checkout") || debug.includes("multi-repo")) {
    console.error(`[find_repo_checkout] ${message}`);
  }
}

/**
 * Normalize a repo slug to lowercase owner/repo format
 * @param {string} repoSlug - The repo slug (owner/repo)
 * @returns {string} Normalized lowercase slug
 */
function normalizeRepoSlug(repoSlug) {
  if (!repoSlug) return "";
  return repoSlug.toLowerCase().trim();
}

/**
 * Extract repo slug from a git remote URL
 * Handles various URL formats:
 * - https://github.com/owner/repo.git
 * - https://github.com/owner/repo
 * - git@github.com:owner/repo.git
 * - git@github.com:owner/repo
 * @param {string} remoteUrl - The git remote URL
 * @returns {string|null} The repo slug (owner/repo) or null if not parseable
 */
function extractRepoSlugFromUrl(remoteUrl) {
  if (!remoteUrl) return null;

  // Remove .git suffix if present
  let url = remoteUrl.trim();
  if (url.endsWith(".git")) {
    url = url.slice(0, -4);
  }

  // Handle HTTPS URLs: https://github.com/owner/repo
  const httpsMatch = url.match(/https?:\/\/[^/]+\/([^/]+\/[^/]+)$/);
  if (httpsMatch) {
    return normalizeRepoSlug(httpsMatch[1]);
  }

  // Handle SSH URLs: git@github.com:owner/repo
  const sshMatch = url.match(/git@[^:]+:([^/]+\/[^/]+)$/);
  if (sshMatch) {
    return normalizeRepoSlug(sshMatch[1]);
  }

  return null;
}

/**
 * Find all .git directories within a base path (non-recursive into .git dirs)
 * @param {string} basePath - The base path to search from
 * @param {number} [maxDepth=5] - Maximum directory depth to search
 * @returns {string[]} Array of paths to .git directories
 */
function findGitDirectories(basePath, maxDepth = 5) {
  const gitDirs = [];

  /**
   * Recursively scan directories
   * @param {string} dir - Current directory
   * @param {number} depth - Current depth
   */
  function scan(dir, depth) {
    if (depth > maxDepth) return;

    try {
      const entries = fs.readdirSync(dir, { withFileTypes: true });

      for (const entry of entries) {
        if (!entry.isDirectory()) continue;

        const fullPath = path.join(dir, entry.name);

        // Skip common non-repo directories for performance
        if (entry.name === "node_modules" || entry.name === ".npm" || entry.name === ".cache") {
          continue;
        }

        if (entry.name === ".git") {
          // Found a git directory - add the parent (the repo root)
          gitDirs.push(dir);
          // Don't recurse into .git directories or the repo they're in
          // (to avoid nested repos unless they're actually separate checkouts)
          continue;
        }

        // Recurse into subdirectories
        scan(fullPath, depth + 1);
      }
    } catch {
      // Ignore permission errors etc
    }
  }

  scan(basePath, 0);
  return gitDirs;
}

/**
 * Get the remote origin URL for a git repository
 * @param {string} repoPath - Path to the repository root
 * @returns {string|null} The remote URL or null if not found
 */
function getRemoteOriginUrl(repoPath) {
  try {
    const url = execGitSync(["config", "--get", "remote.origin.url"], { cwd: repoPath });
    return url.trim();
  } catch {
    return null;
  }
}

/**
 * Find the checkout directory for a given repo slug
 * Searches the workspace for git repos and matches by remote URL
 *
 * @param {string} repoSlug - The repository slug to find (owner/repo format)
 * @param {string} [workspaceRoot] - The workspace root to search from
 * @returns {Object} Result with success status and path or error
 */
function findRepoCheckout(repoSlug, workspaceRoot) {
  const ws = workspaceRoot || process.env.GITHUB_WORKSPACE || process.cwd();
  const targetSlug = normalizeRepoSlug(repoSlug);

  debugLog(`Searching for repo: ${targetSlug} in workspace: ${ws}`);

  if (!targetSlug) {
    return {
      success: false,
      error: "Invalid repo slug provided",
    };
  }

  // Find all git directories in the workspace
  const gitDirs = findGitDirectories(ws);
  debugLog(`Found ${gitDirs.length} git directories: ${gitDirs.join(", ")}`);

  // Check each git directory for a matching remote
  for (const repoPath of gitDirs) {
    const remoteUrl = getRemoteOriginUrl(repoPath);
    if (!remoteUrl) {
      debugLog(`No remote URL found for: ${repoPath}`);
      continue;
    }

    const foundSlug = extractRepoSlugFromUrl(remoteUrl);
    debugLog(`Repo at ${repoPath} has slug: ${foundSlug}`);

    if (foundSlug === targetSlug) {
      debugLog(`Found match: ${repoPath}`);
      return {
        success: true,
        path: repoPath,
        repoSlug: targetSlug,
      };
    }
  }

  // Special case: check workspace root as a potential match
  // This handles the scenario where only the root is a repo
  const rootRemoteUrl = getRemoteOriginUrl(ws);
  if (rootRemoteUrl) {
    const rootSlug = extractRepoSlugFromUrl(rootRemoteUrl);
    debugLog(`Workspace root has slug: ${rootSlug}`);
    if (rootSlug === targetSlug) {
      return {
        success: true,
        path: ws,
        repoSlug: targetSlug,
      };
    }
  }

  return {
    success: false,
    error: `Repository '${repoSlug}' not found in workspace. Make sure it's checked out using actions/checkout with a path.`,
    searchedPaths: gitDirs,
  };
}

/**
 * Build a map of all checked-out repos in the workspace
 * @param {string} [workspaceRoot] - The workspace root to search from
 * @returns {Map<string, string>} Map of repo slug -> checkout path
 */
function buildRepoCheckoutMap(workspaceRoot) {
  const ws = workspaceRoot || process.env.GITHUB_WORKSPACE || process.cwd();
  const map = new Map();

  const gitDirs = findGitDirectories(ws);

  for (const repoPath of gitDirs) {
    const remoteUrl = getRemoteOriginUrl(repoPath);
    if (!remoteUrl) continue;

    const slug = extractRepoSlugFromUrl(remoteUrl);
    if (slug) {
      map.set(slug, repoPath);
    }
  }

  // Also check workspace root
  const rootRemoteUrl = getRemoteOriginUrl(ws);
  if (rootRemoteUrl) {
    const rootSlug = extractRepoSlugFromUrl(rootRemoteUrl);
    if (rootSlug && !map.has(rootSlug)) {
      map.set(rootSlug, ws);
    }
  }

  debugLog(`Built repo checkout map with ${map.size} entries`);
  return map;
}

module.exports = {
  findRepoCheckout,
  buildRepoCheckoutMap,
  extractRepoSlugFromUrl,
  normalizeRepoSlug,
  findGitDirectories,
};
