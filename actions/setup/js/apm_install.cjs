// @ts-check
/// <reference types="@actions/github-script" />

/**
 * APM Package Installer
 *
 * JavaScript reimplementation of `apm install`. Downloads APM packages
 * from GitHub and creates the installed workspace used by `apm pack`.
 *
 * Algorithm:
 *   1. Parse APM_PACKAGES (JSON array of package slugs) from the environment
 *   2. For each package slug:
 *      a. Parse the slug: owner/repo[/subpath][#ref]
 *      b. Resolve the ref (branch/tag/SHA) to a full commit SHA
 *      c. Scan the repo tree recursively for deployable files
 *         - Full package (no subpath): files under .github/, .claude/, .cursor/, .opencode/
 *         - Individual primitive (with subpath): files under {target_dir}/{subpath}/
 *      d. Download each file and write to APM_WORKSPACE at its original path
 *      e. Record the resolved dependency in the lockfile
 *   3. Write apm.yml (workspace metadata for the packer) to APM_WORKSPACE
 *   4. Write apm.lock.yaml (resolved dependency manifest) to APM_WORKSPACE
 *
 * Environment variables:
 *   GITHUB_APM_PAT   – GitHub token for API access (required for private repos;
 *                       falls back to GITHUB_TOKEN if not set)
 *   APM_PACKAGES     – JSON array of package slugs,
 *                       e.g. '["microsoft/apm-sample-package","org/repo/skills/foo#v2"]'
 *   APM_WORKSPACE    – destination directory for downloaded files + lockfile
 *                       (default: /tmp/gh-aw/apm-workspace)
 *
 * @module apm_install
 */

"use strict";

const fs = require("fs");
const path = require("path");

/** Lockfile filename written to the workspace. */
const LOCKFILE_NAME = "apm.lock.yaml";

/** apm.yml filename for workspace metadata (consumed by apm_pack). */
const APM_YML_NAME = "apm.yml";

/** Directories that contain deployable APM primitives. */
const TARGET_DIRS = [".github/", ".claude/", ".cursor/", ".opencode/"];

// ---------------------------------------------------------------------------
// Package slug parser
// ---------------------------------------------------------------------------

/**
 * @typedef {Object} PackageRef
 * @property {string} owner     - GitHub org / user
 * @property {string} repo      - Repository name
 * @property {string | null} subpath - Path within the repo (e.g. "skills/foo"); null = full package
 * @property {string | null} ref     - Git ref (branch, tag, SHA); null = default branch
 */

/**
 * Parse an APM package slug into its components.
 *
 * Formats:
 *   owner/repo
 *   owner/repo#ref
 *   owner/repo/path/to/primitive
 *   owner/repo/path/to/primitive#ref
 *
 * @param {string} slug
 * @returns {PackageRef}
 */
function parsePackageSlug(slug) {
  if (!slug || typeof slug !== "string") throw new Error(`Invalid package slug: ${JSON.stringify(slug)}`);

  // Split off optional #ref suffix
  const hashIdx = slug.indexOf("#");
  const ref = hashIdx >= 0 ? slug.slice(hashIdx + 1) || null : null;
  const pathPart = hashIdx >= 0 ? slug.slice(0, hashIdx) : slug;

  const parts = pathPart.split("/");
  if (parts.length < 2 || !parts[0] || !parts[1]) {
    throw new Error(`Invalid package slug (expected owner/repo[/subpath][#ref]): ${JSON.stringify(slug)}`);
  }

  const owner = parts[0];
  const repo = parts[1];
  const subpath = parts.length > 2 ? parts.slice(2).join("/") : null;

  return { owner, repo, subpath, ref };
}

// ---------------------------------------------------------------------------
// YAML scalar serializer (inline copy — avoids cross-module circular dep)
// ---------------------------------------------------------------------------

/**
 * Serialize a scalar value to YAML, quoting strings that YAML would misinterpret
 * (e.g. keywords, numbers, ISO timestamps). Mirrors PyYAML safe_dump quoting.
 *
 * @param {string | number | boolean | null | undefined} value
 * @returns {string}
 */
function scalarToYaml(value) {
  if (value === null || value === undefined) return "null";
  if (typeof value === "boolean") return value ? "true" : "false";
  if (typeof value === "number") return String(value);
  const s = String(value);
  // YAML keywords and special patterns that must be quoted to preserve string type
  const YAML_KEYWORDS = new Set(["", "null", "~", "true", "false", "yes", "no", "on", "off"]);
  const NEEDS_QUOTING =
    YAML_KEYWORDS.has(s) ||
    /^-?\d+$/.test(s) || // integer
    /^-?\d+\.\d+$/.test(s) || // float
    /^\d{4}-\d{2}-\d{2}T/.test(s); // ISO 8601 datetime
  if (NEEDS_QUOTING) {
    return `'${s.replace(/'/g, "''")}'`;
  }
  return s;
}

// ---------------------------------------------------------------------------
// Lockfile writer
// ---------------------------------------------------------------------------

/**
 * @typedef {Object} InstalledDependency
 * @property {string} repo_url
 * @property {string} resolved_commit
 * @property {string} resolved_ref
 * @property {string[]} deployed_files
 */

/**
 * Write the workspace apm.lock.yaml based on the resolved dependencies.
 *
 * @param {string} workspaceDir
 * @param {InstalledDependency[]} dependencies
 */
function writeWorkspaceLockfile(workspaceDir, dependencies) {
  const lines = [];
  lines.push(`lockfile_version: ${scalarToYaml("1")}`);
  lines.push(`generated_at: ${scalarToYaml(new Date().toISOString())}`);
  lines.push("apm_version: null");
  lines.push("dependencies:");

  for (const dep of dependencies) {
    lines.push(`- repo_url: ${scalarToYaml(dep.repo_url)}`);
    lines.push(`  host: github.com`);
    lines.push(`  resolved_commit: ${scalarToYaml(dep.resolved_commit)}`);
    lines.push(`  resolved_ref: ${scalarToYaml(dep.resolved_ref)}`);
    lines.push(`  version: null`);
    lines.push(`  virtual_path: null`);
    lines.push(`  is_virtual: false`);
    lines.push(`  depth: 1`);
    lines.push(`  resolved_by: apm_install.cjs`);
    lines.push(`  package_type: apm`);
    lines.push(`  source: null`);
    lines.push(`  local_path: null`);
    lines.push(`  content_hash: null`);
    lines.push(`  is_dev: false`);
    lines.push(`  deployed_files:`);
    for (const f of dep.deployed_files) {
      lines.push(`  - ${scalarToYaml(f)}`);
    }
  }

  const lockfileContent = lines.join("\n") + "\n";
  fs.writeFileSync(path.join(workspaceDir, LOCKFILE_NAME), lockfileContent, "utf-8");
}

/**
 * Write a minimal apm.yml to the workspace (consumed by apm_pack for bundle naming).
 *
 * @param {string} workspaceDir
 */
function writeWorkspaceApmYml(workspaceDir) {
  const content = "name: gh-aw-workspace\nversion: 0.0.0\n";
  fs.writeFileSync(path.join(workspaceDir, APM_YML_NAME), content, "utf-8");
}

// ---------------------------------------------------------------------------
// GitHub API client factory
// ---------------------------------------------------------------------------

/**
 * Create an authenticated Octokit client.
 *
 * Priority:
 *   1. Custom token in GITHUB_APM_PAT (may differ from GITHUB_TOKEN for private repos)
 *   2. GITHUB_TOKEN (workflow token, public repos + repos accessible to workflow)
 *
 * When running via actions/github-script, global.github is available but is
 * authenticated with GITHUB_TOKEN. We create a dedicated instance with
 * GITHUB_APM_PAT so private package repos are accessible.
 *
 * @param {string} token - GitHub PAT or workflow token
 * @returns Octokit instance
 */
function createOctokit(token) {
  // @actions/github is bundled with actions/github-script and available in
  // older CJS-compatible versions. When running standalone with @actions/github
  // v9+ (ESM-only), fall back to @octokit/core + plugins.
  try {
    // @ts-ignore – dynamic require at runtime
    const { getOctokit } = require("@actions/github");
    return getOctokit(token);
  } catch {
    // @actions/github v9+ is ESM-only; use @octokit/core + rest-endpoint-methods
    // @ts-ignore – dynamic require at runtime
    const { Octokit } = require("@octokit/core");
    // @ts-ignore – dynamic require at runtime
    const { restEndpointMethods } = require("@octokit/plugin-rest-endpoint-methods");
    // @ts-ignore – dynamic require at runtime
    const { paginateRest } = require("@octokit/plugin-paginate-rest");
    const MyOctokit = Octokit.plugin(restEndpointMethods, paginateRest);
    return new MyOctokit({ auth: token });
  }
}

// ---------------------------------------------------------------------------
// GitHub REST helpers
// ---------------------------------------------------------------------------

/**
 * Resolve a git ref (branch name, tag, SHA, etc.) to a full commit SHA.
 * Returns the commit SHA and the effective ref string used.
 *
 * @param {*} octokit
 * @param {string} owner
 * @param {string} repo
 * @param {string | null} ref
 * @returns {Promise<{commitSha: string, resolvedRef: string, treeSha: string}>}
 */
async function resolveCommit(octokit, owner, repo, ref) {
  let effectiveRef = ref;
  if (!effectiveRef) {
    const { data: repoData } = await octokit.rest.repos.get({ owner, repo });
    effectiveRef = repoData.default_branch;
  }

  const { data: commitData } = await octokit.rest.repos.getCommit({
    owner,
    repo,
    ref: effectiveRef,
  });

  // effectiveRef is always non-null here (set to default_branch if null was passed)
  const resolvedRef = effectiveRef ?? "";
  return {
    commitSha: commitData.sha,
    resolvedRef,
    treeSha: commitData.commit.tree.sha,
  };
}

/**
 * Get the recursive file tree for a commit.
 * Returns only blob (file) entries with their paths and blob SHAs.
 *
 * @param {*} octokit
 * @param {string} owner
 * @param {string} repo
 * @param {string} treeSha
 * @returns {Promise<Array<{path: string, sha: string}>>}
 */
async function getFileTree(octokit, owner, repo, treeSha) {
  const { data } = await octokit.rest.git.getTree({
    owner,
    repo,
    tree_sha: treeSha,
    recursive: "1",
  });
  const blobs = (data.tree || []).filter(entry => entry.type === "blob" && entry.path);
  return blobs.map(entry => ({
    path: /** @type {string} */ entry.path,
    sha: entry.sha || "",
  }));
}

/**
 * Download raw file content from a GitHub repo at a specific commit SHA.
 *
 * @param {*} octokit
 * @param {string} owner
 * @param {string} repo
 * @param {string} filePath
 * @param {string} ref
 * @returns {Promise<Buffer>}
 */
async function downloadFileContent(octokit, owner, repo, filePath, ref) {
  const { data } = await octokit.rest.repos.getContent({
    owner,
    repo,
    path: filePath,
    ref,
  });

  // getContent returns different shapes; we only handle file blobs here
  if (Array.isArray(data)) {
    throw new Error(`Expected a file at '${filePath}' in ${owner}/${repo} but got a directory listing`);
  }
  if (!("content" in data) || !("encoding" in data)) {
    throw new Error(`Unexpected content shape for '${filePath}' in ${owner}/${repo}`);
  }

  // Content is base64-encoded (the default encoding from the API)
  const content = /** @type {{ content: string; encoding: string }} */ data;
  if (content.encoding !== "base64") {
    throw new Error(`Unexpected encoding '${content.encoding}' for '${filePath}'`);
  }
  return Buffer.from(content.content.replace(/\n/g, ""), "base64");
}

// ---------------------------------------------------------------------------
// Package installation
// ---------------------------------------------------------------------------

/**
 * Determine which files from a repo tree should be installed for a given package ref.
 *
 * - Full package (no subpath): all files under .github/, .claude/, .cursor/, .opencode/
 * - Primitive subpath: files under {target_dir}/{subpath}/ for every target_dir
 *
 * @param {Array<{path: string, sha: string}>} tree
 * @param {string | null} subpath
 * @returns {Array<{path: string, sha: string}>}
 */
function selectDeployableFiles(tree, subpath) {
  if (!subpath) {
    // Full package — include all files under any known target directory
    return tree.filter(entry => TARGET_DIRS.some(tdir => entry.path.startsWith(tdir)));
  }

  // Individual primitive — look for the subpath under every target directory,
  // plus the subpath itself if files live directly at it (no target prefix)
  const normalizedSubpath = subpath.endsWith("/") ? subpath : subpath + "/";
  return tree.filter(entry => {
    // Case 1: file already has a target dir prefix (common for APM package repos)
    if (TARGET_DIRS.some(tdir => entry.path.startsWith(tdir + normalizedSubpath))) return true;
    // Case 2: file is directly under the subpath (without target dir prefix)
    if (entry.path.startsWith(normalizedSubpath)) return true;
    // Case 3: exact match (e.g. subpath points to a single file)
    if (entry.path === subpath) return true;
    return false;
  });
}

/**
 * Install a single APM package into the workspace.
 *
 * @param {*} octokit - Authenticated Octokit instance
 * @param {PackageRef} pkgRef - Parsed package reference
 * @param {string} workspaceDir - Absolute path to workspace
 * @returns {Promise<InstalledDependency>}
 */
async function installPackage(octokit, pkgRef, workspaceDir) {
  const { owner, repo, subpath, ref } = pkgRef;
  const repoUrl = `https://github.com/${owner}/${repo}`;

  core.info(`[APM Install] Installing ${owner}/${repo}${subpath ? `/${subpath}` : ""}${ref ? `#${ref}` : ""}`);

  // Resolve ref → commit SHA + tree SHA
  const { commitSha, resolvedRef, treeSha } = await resolveCommit(octokit, owner, repo, ref);
  core.info(`[APM Install]   ref: ${resolvedRef} → ${commitSha.slice(0, 12)}`);

  // Get recursive file tree
  const tree = await getFileTree(octokit, owner, repo, treeSha);
  core.info(`[APM Install]   repo tree: ${tree.length} blob(s)`);

  // Filter to deployable files
  const deployable = selectDeployableFiles(tree, subpath);
  if (deployable.length === 0) {
    core.warning(`[APM Install] No deployable files found in ${owner}/${repo}${subpath ? `/${subpath}` : ""}. ` + `Checked for files under ${TARGET_DIRS.join(", ")}${subpath ? `${subpath}/` : ""}.`);
  }
  core.info(`[APM Install]   deployable: ${deployable.length} file(s)`);

  // Download and write each file
  const deployedFiles = [];
  const workspaceDirResolved = path.resolve(workspaceDir);

  for (let i = 0; i < deployable.length; i++) {
    const entry = deployable[i];
    const filePath = entry.path;

    // Security: reject absolute paths and path traversal
    if (path.isAbsolute(filePath) || filePath.includes("..")) {
      core.warning(`[APM Install] Skipping unsafe path from ${owner}/${repo}: ${JSON.stringify(filePath)}`);
      continue;
    }

    const destAbsPath = path.resolve(path.join(workspaceDir, filePath));
    // Guard: destination must stay inside workspace
    if (!destAbsPath.startsWith(workspaceDirResolved + path.sep) && destAbsPath !== workspaceDirResolved) {
      core.warning(`[APM Install] Skipping path that escapes workspace: ${JSON.stringify(filePath)}`);
      continue;
    }

    fs.mkdirSync(path.dirname(destAbsPath), { recursive: true });
    const content = await downloadFileContent(octokit, owner, repo, filePath, commitSha);
    fs.writeFileSync(destAbsPath, content);
    deployedFiles.push(filePath);

    if ((i + 1) % 10 === 0 || i + 1 === deployable.length) {
      core.info(`[APM Install]   progress: ${i + 1}/${deployable.length} downloaded`);
    }
  }

  core.info(`[APM Install] ✓ ${owner}/${repo}: ${deployedFiles.length} file(s) installed`);

  return {
    repo_url: repoUrl,
    resolved_commit: commitSha,
    resolved_ref: resolvedRef,
    deployed_files: deployedFiles,
  };
}

// ---------------------------------------------------------------------------
// Entry point
// ---------------------------------------------------------------------------

/**
 * Main entry point.
 *
 * Accepts an optional options object for dependency injection (used in tests).
 *
 * @param {object} [opts]
 * @param {*} [opts.octokitOverride]    - Override Octokit client (for testing)
 * @param {string} [opts.workspaceDir]  - Override workspace directory (for testing)
 * @param {string[]} [opts.packages]    - Override packages list (for testing)
 * @param {string} [opts.token]         - Override auth token (for testing)
 */
async function main(opts = {}) {
  const { octokitOverride = null, workspaceDir = process.env.APM_WORKSPACE || "/tmp/gh-aw/apm-workspace", packages = parsePackagesFromEnv(), token = process.env.GITHUB_APM_PAT || process.env.GITHUB_TOKEN || "" } = opts;

  core.info("=== APM Package Install ===");
  core.info(`[APM Install] Workspace directory: ${workspaceDir}`);
  core.info(`[APM Install] Packages           : ${packages.length}`);

  if (packages.length === 0) {
    core.warning("[APM Install] No packages to install (APM_PACKAGES is empty)");
    fs.mkdirSync(workspaceDir, { recursive: true });
    writeWorkspaceApmYml(workspaceDir);
    writeWorkspaceLockfile(workspaceDir, []);
    return;
  }

  const octokit = octokitOverride || createOctokit(token);

  fs.mkdirSync(workspaceDir, { recursive: true });

  /** @type {InstalledDependency[]} */
  const dependencies = [];

  for (const slug of packages) {
    core.info(`[APM Install] ── ${slug}`);
    const pkgRef = parsePackageSlug(slug);
    const dep = await installPackage(octokit, pkgRef, workspaceDir);
    dependencies.push(dep);
  }

  writeWorkspaceApmYml(workspaceDir);
  writeWorkspaceLockfile(workspaceDir, dependencies);

  core.info(`[APM Install] ✅ Installed ${dependencies.length} package(s)`);
  core.info(`[APM Install]    Workspace: ${workspaceDir}`);
}

/**
 * Parse APM_PACKAGES env var into an array of package slug strings.
 * Accepts a JSON array: '["owner/repo","owner/repo2"]'
 *
 * @returns {string[]}
 */
function parsePackagesFromEnv() {
  const raw = process.env.APM_PACKAGES;
  if (!raw || raw.trim() === "") return [];
  try {
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) throw new Error("APM_PACKAGES must be a JSON array");
    return parsed.filter(p => typeof p === "string" && p.trim() !== "");
  } catch (e) {
    throw new Error(`[APM Install] Failed to parse APM_PACKAGES env var: ${e instanceof Error ? e.message : String(e)}\n` + `  Expected a JSON array, e.g.: ["owner/repo", "owner/repo/skills/foo#v2"]`);
  }
}

module.exports = {
  main,
  parsePackageSlug,
  selectDeployableFiles,
  writeWorkspaceLockfile,
  writeWorkspaceApmYml,
  parsePackagesFromEnv,
  // Exported for tests only
  resolveCommit,
  installPackage,
  createOctokit,
  scalarToYaml,
};
