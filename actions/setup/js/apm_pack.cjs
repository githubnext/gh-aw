// @ts-check
/// <reference types="@actions/github-script" />

/**
 * APM Bundle Packer
 *
 * JavaScript implementation of the APM (Agent Package Manager) bundle pack
 * algorithm, equivalent to microsoft/apm packer.py.
 *
 * This module creates a self-contained .tar.gz bundle from an already-installed
 * APM workspace (produced by `apm install`). It replaces the `microsoft/apm-action`
 * pack step in the APM job, removing the external dependency for packing.
 *
 * Algorithm (mirrors packer.py):
 *   1. Read apm.yml for package name and version (used for bundle directory name)
 *   2. Read apm.lock.yaml from the workspace
 *   3. Resolve the effective target (explicit > auto-detect from folder structure)
 *   4. Collect deployed_files from all dependencies, filtered by target with
 *      cross-target mapping (e.g. .github/skills/ → .claude/skills/ for claude target)
 *   5. Verify all referenced files exist on disk
 *   6. Copy files (skip symlinks) to output/<name>-<version>/ preserving structure
 *   7. Write an enriched apm.lock.yaml with a pack: header to the bundle directory
 *   8. Create a .tar.gz archive and remove the bundle directory
 *   9. Emit bundle-path output via core.setOutput
 *
 * Environment variables:
 *   APM_WORKSPACE     – project root with apm.lock.yaml and installed files
 *                       (default: /tmp/gh-aw/apm-workspace)
 *   APM_BUNDLE_OUTPUT – directory where the bundle archive is created
 *                       (default: /tmp/gh-aw/apm-bundle-output)
 *   APM_TARGET        – pack target: claude, copilot/vscode, cursor, opencode, all
 *                       (default: auto-detect from workspace folder structure)
 *
 * @module apm_pack
 */

"use strict";

const fs = require("fs");
const path = require("path");
const os = require("os");

/** Lockfile filename used by current APM versions. */
const LOCKFILE_NAME = "apm.lock.yaml";

/** apm.yml filename for package metadata. */
const APM_YML_NAME = "apm.yml";

// Import shared parsing utilities from apm_unpack to avoid duplication.
// Globals (core, exec) must be set before this module is loaded.
const { parseAPMLockfile, unquoteYaml } = require("./apm_unpack.cjs");

// ---------------------------------------------------------------------------
// Target / cross-target mapping constants (mirrors lockfile_enrichment.py)
// ---------------------------------------------------------------------------

/**
 * Authoritative mapping of target names to deployed-file path prefixes.
 * @type {Record<string, string[]>}
 */
const TARGET_PREFIXES = {
  copilot: [".github/"],
  vscode: [".github/"],
  claude: [".claude/"],
  cursor: [".cursor/"],
  opencode: [".opencode/"],
  all: [".github/", ".claude/", ".cursor/", ".opencode/"],
};

/**
 * Cross-target path equivalences for skills/ and agents/ directories.
 * Maps srcPrefix (disk/deployed_files path) → dstPrefix (bundle path) for a given target,
 * as used by filterFilesByTarget(). Only skills/ and agents/ are semantically identical
 * across targets.
 * @type {Record<string, Record<string, string>>}
 */
const CROSS_TARGET_MAPS = {
  claude: {
    ".github/skills/": ".claude/skills/",
    ".github/agents/": ".claude/agents/",
  },
  vscode: {
    ".claude/skills/": ".github/skills/",
    ".claude/agents/": ".github/agents/",
  },
  copilot: {
    ".claude/skills/": ".github/skills/",
    ".claude/agents/": ".github/agents/",
  },
  cursor: {
    ".github/skills/": ".cursor/skills/",
    ".github/agents/": ".cursor/agents/",
  },
  opencode: {
    ".github/skills/": ".opencode/skills/",
    ".github/agents/": ".opencode/agents/",
  },
};

// ---------------------------------------------------------------------------
// apm.yml parser
// ---------------------------------------------------------------------------

/**
 * @typedef {Object} ApmYmlInfo
 * @property {string} name - Package name (defaults to directory name if missing)
 * @property {string} version - Package version (defaults to "0.0.0" if missing)
 */

/**
 * Parse an apm.yml file to extract package name and version.
 * These are used to name the bundle directory: <name>-<version>.
 *
 * @param {string} content - Raw YAML content of apm.yml
 * @param {string} [fallbackName] - Fallback name if not found in content
 * @returns {ApmYmlInfo}
 */
function parseApmYml(content, fallbackName = "bundle") {
  let name = fallbackName;
  let version = "0.0.0";

  for (const line of content.split("\n")) {
    const m = line.match(/^(name|version):\s*(.*)$/);
    if (m) {
      const v = unquoteYaml(m[2]);
      if (m[1] === "name" && v !== null && String(v).trim() !== "") {
        name = String(v).trim();
      } else if (m[1] === "version" && v !== null && String(v).trim() !== "") {
        version = String(v).trim();
      }
    }
  }

  return { name, version };
}

// ---------------------------------------------------------------------------
// Target detection
// ---------------------------------------------------------------------------

/**
 * Detect the effective pack target.
 *
 * Priority:
 *   1. Explicit target (from APM_TARGET environment variable)
 *   2. Auto-detect from workspace folder structure
 *
 * @param {string} workspaceDir - Project root to inspect for target folders
 * @param {string | null | undefined} explicitTarget - Explicit target string
 * @returns {string} Normalised target string
 */
function detectTarget(workspaceDir, explicitTarget) {
  if (explicitTarget) {
    const t = explicitTarget.trim().toLowerCase();
    if (t === "copilot" || t === "vscode" || t === "agents") return "vscode";
    if (t === "claude") return "claude";
    if (t === "cursor") return "cursor";
    if (t === "opencode") return "opencode";
    if (t === "all") return "all";
    core.warning(`[APM Pack] Unknown target '${t}' — falling back to 'all'`);
    return "all";
  }

  // Auto-detect from folder structure
  const githubDir = path.join(workspaceDir, ".github");
  const claudeDir = path.join(workspaceDir, ".claude");
  const cursorDir = path.join(workspaceDir, ".cursor");
  const opencodeDir = path.join(workspaceDir, ".opencode");
  const hasGitHub = fs.existsSync(githubDir) && fs.lstatSync(githubDir).isDirectory();
  const hasClaude = fs.existsSync(claudeDir) && fs.lstatSync(claudeDir).isDirectory();
  const hasCursor = fs.existsSync(cursorDir) && fs.lstatSync(cursorDir).isDirectory();
  const hasOpencode = fs.existsSync(opencodeDir) && fs.lstatSync(opencodeDir).isDirectory();

  const detected = [hasGitHub && ".github/", hasClaude && ".claude/", hasCursor && ".cursor/", hasOpencode && ".opencode/"].filter(Boolean);

  if (detected.length >= 2) {
    core.info(`[APM Pack] Auto-detected target: all (found ${detected.join(" and ")})`);
    return "all";
  }
  if (hasGitHub) {
    core.info("[APM Pack] Auto-detected target: vscode (found .github/ folder)");
    return "vscode";
  }
  if (hasClaude) {
    core.info("[APM Pack] Auto-detected target: claude (found .claude/ folder)");
    return "claude";
  }
  if (hasCursor) {
    core.info("[APM Pack] Auto-detected target: cursor (found .cursor/ folder)");
    return "cursor";
  }
  if (hasOpencode) {
    core.info("[APM Pack] Auto-detected target: opencode (found .opencode/ folder)");
    return "opencode";
  }
  core.info("[APM Pack] No target folders found — using 'all'");
  return "all";
}

// ---------------------------------------------------------------------------
// File filtering with cross-target mapping
// ---------------------------------------------------------------------------

/**
 * @typedef {Object} FilterResult
 * @property {string[]} files - Filtered (and cross-target mapped) file paths for the bundle.
 * @property {Record<string, string>} pathMappings - Maps bundle_path → disk_path for cross-target remaps.
 */

/**
 * Filter deployed file paths by target prefix, with cross-target mapping.
 *
 * When files are deployed under one target prefix (e.g. .github/skills/)
 * but the pack target is different (e.g. claude), skills and agents are
 * remapped to the equivalent target path.  Commands, instructions, and
 * hooks are NOT remapped — they are target-specific.
 *
 * Mirrors _filter_files_by_target in lockfile_enrichment.py exactly.
 *
 * @param {string[]} deployedFiles - List of relative file paths from deployed_files.
 * @param {string} target - Normalised target string.
 * @returns {FilterResult}
 */
function filterFilesByTarget(deployedFiles, target) {
  const prefixes = TARGET_PREFIXES[target] || TARGET_PREFIXES["all"];
  // Direct matches: files that start with a target prefix
  const direct = deployedFiles.filter(f => prefixes.some(p => f.startsWith(p)));

  /** @type {Record<string, string>} */
  const pathMappings = {};
  const crossMap = CROSS_TARGET_MAPS[target] || {};

  if (Object.keys(crossMap).length > 0) {
    const directSet = new Set(direct);
    for (const f of deployedFiles) {
      if (directSet.has(f)) continue;
      for (const [srcPrefix, dstPrefix] of Object.entries(crossMap)) {
        if (f.startsWith(srcPrefix)) {
          const mapped = dstPrefix + f.slice(srcPrefix.length);
          if (!directSet.has(mapped)) {
            direct.push(mapped);
            directSet.add(mapped);
            pathMappings[mapped] = f; // bundle_path → disk_path
          }
          break;
        }
      }
    }
  }

  return { files: direct, pathMappings };
}

// ---------------------------------------------------------------------------
// YAML serialization for the enriched lockfile
// ---------------------------------------------------------------------------

/**
 * Serialize a scalar value to a YAML string, matching PyYAML safe_dump style.
 *
 * Strings that look like YAML keywords or numbers are single-quoted.
 * Other strings are returned as-is. Booleans and numbers are serialized
 * without quotes. Null becomes the literal string "null".
 *
 * @param {string | number | boolean | null | undefined} value
 * @returns {string}
 */
function scalarToYaml(value) {
  if (value === null || value === undefined) return "null";
  if (typeof value === "boolean") return value ? "true" : "false";
  if (typeof value === "number") return String(value);
  const s = String(value);
  // Quote strings that YAML would parse as non-strings (mirrors PyYAML safe_dump quoting)
  if (
    s === "" ||
    s === "null" ||
    s === "~" ||
    s === "true" ||
    s === "false" ||
    s === "yes" ||
    s === "no" ||
    s === "on" ||
    s === "off" ||
    /^-?\d+$/.test(s) ||
    /^-?\d+\.\d+$/.test(s) ||
    // YAML 1.1 parses ISO 8601 timestamps as datetime objects; quote to preserve string type
    /^\d{4}-\d{2}-\d{2}T/.test(s)
  ) {
    return `'${s.replace(/'/g, "''")}'`;
  }
  return s;
}

/**
 * Serialize an enriched APM lockfile to YAML.
 *
 * The output format matches PyYAML safe_dump: the pack: section is
 * prepended, followed by top-level metadata, then the dependencies
 * sequence with filtered deployed_files.
 *
 * This output is parseable by both:
 *   - Python yaml.safe_load (used by apm unpack)
 *   - Our parseAPMLockfile (used by apm_unpack.cjs)
 *
 * @param {import("./apm_unpack.cjs").APMLockfile} lockfile - Parsed lockfile
 * @param {import("./apm_unpack.cjs").LockedDependency[]} filteredDeps - Deps with filtered deployed_files
 * @param {{ format: string, target: string, packed_at: string, mapped_from?: string[] }} packMeta
 * @returns {string} YAML string
 */
function serializeLockfileYaml(lockfile, filteredDeps, packMeta) {
  const lines = [];

  // Pack metadata section (prepended first, as in Python's enrich_lockfile_for_pack)
  lines.push("pack:");
  lines.push(`  format: ${scalarToYaml(packMeta.format)}`);
  lines.push(`  target: ${scalarToYaml(packMeta.target)}`);
  lines.push(`  packed_at: ${scalarToYaml(packMeta.packed_at)}`);
  if (packMeta.mapped_from && packMeta.mapped_from.length > 0) {
    lines.push("  mapped_from:");
    for (const prefix of packMeta.mapped_from) {
      lines.push(`  - ${prefix}`);
    }
  }

  // Top-level metadata fields
  if (lockfile.lockfile_version !== null) {
    lines.push(`lockfile_version: ${scalarToYaml(lockfile.lockfile_version)}`);
  }
  if (lockfile.generated_at !== null) {
    lines.push(`generated_at: ${scalarToYaml(lockfile.generated_at)}`);
  }
  if (lockfile.apm_version !== null) {
    lines.push(`apm_version: ${scalarToYaml(lockfile.apm_version)}`);
  }

  // Dependencies sequence
  lines.push("dependencies:");
  for (const dep of filteredDeps) {
    lines.push(`- repo_url: ${scalarToYaml(dep.repo_url)}`);
    if (dep.host !== null) lines.push(`  host: ${scalarToYaml(dep.host)}`);
    else lines.push(`  host: null`);
    if (dep.resolved_commit !== null) lines.push(`  resolved_commit: ${scalarToYaml(dep.resolved_commit)}`);
    else lines.push(`  resolved_commit: null`);
    if (dep.resolved_ref !== null) lines.push(`  resolved_ref: ${scalarToYaml(dep.resolved_ref)}`);
    else lines.push(`  resolved_ref: null`);
    if (dep.version !== null) lines.push(`  version: ${scalarToYaml(dep.version)}`);
    else lines.push(`  version: null`);
    if (dep.virtual_path !== null) lines.push(`  virtual_path: ${scalarToYaml(dep.virtual_path)}`);
    else lines.push(`  virtual_path: null`);
    lines.push(`  is_virtual: ${dep.is_virtual ? "true" : "false"}`);
    lines.push(`  depth: ${dep.depth}`);
    if (dep.resolved_by !== null) lines.push(`  resolved_by: ${scalarToYaml(dep.resolved_by)}`);
    else lines.push(`  resolved_by: null`);
    if (dep.package_type !== null) lines.push(`  package_type: ${scalarToYaml(dep.package_type)}`);
    else lines.push(`  package_type: null`);
    if (dep.source !== null) lines.push(`  source: ${scalarToYaml(dep.source)}`);
    else lines.push(`  source: null`);
    if (dep.local_path !== null) lines.push(`  local_path: ${scalarToYaml(dep.local_path)}`);
    else lines.push(`  local_path: null`);
    if (dep.content_hash !== null) lines.push(`  content_hash: ${scalarToYaml(dep.content_hash)}`);
    else lines.push(`  content_hash: null`);
    lines.push(`  is_dev: ${dep.is_dev ? "true" : "false"}`);
    // Preserve unknown fields so the enriched lockfile is non-destructive
    for (const [k, v] of Object.entries(dep.extra || {})) {
      lines.push(`  ${k}: ${scalarToYaml(v)}`);
    }
    lines.push("  deployed_files:");
    for (const f of dep.deployed_files) {
      lines.push(`  - ${scalarToYaml(f)}`);
    }
  }

  return lines.join("\n") + "\n";
}

// ---------------------------------------------------------------------------
// Security helpers (mirrors assertSafePath / assertDestInsideOutput in unpacker)
// ---------------------------------------------------------------------------

/**
 * Validate that a relative path from the lockfile is safe to pack.
 * Rejects absolute paths and path-traversal attempts.
 *
 * @param {string} relPath - Relative path string from deployed_files.
 * @throws {Error} If the path is unsafe.
 */
function assertSafePackPath(relPath) {
  if (path.isAbsolute(relPath) || relPath.startsWith("/")) {
    throw new Error(`Refusing to pack unsafe absolute path from lockfile: ${JSON.stringify(relPath)}`);
  }
  const parts = relPath.split(/[\\/]/);
  if (parts.includes("..")) {
    throw new Error(`Refusing to pack path-traversal entry from lockfile: ${JSON.stringify(relPath)}`);
  }
}

/**
 * Verify the resolved destination path stays within the bundle directory.
 *
 * @param {string} destPath - Absolute destination path.
 * @param {string} bundleDirResolved - Resolved absolute bundle directory.
 * @throws {Error} If the dest escapes the bundle directory.
 */
function assertPackDestInside(destPath, bundleDirResolved) {
  const resolved = path.resolve(destPath);
  if (!resolved.startsWith(bundleDirResolved + path.sep) && resolved !== bundleDirResolved) {
    throw new Error(`Refusing to pack path that escapes the bundle directory: ${JSON.stringify(destPath)}`);
  }
}

// ---------------------------------------------------------------------------
// Copy helpers (mirrors copyDirRecursive / listDirRecursive in apm_unpack)
// ---------------------------------------------------------------------------

/**
 * Recursively copy a directory tree from src to dest, skipping symbolic links.
 *
 * @param {string} src - Source directory.
 * @param {string} dest - Destination directory.
 * @returns {number} Number of files copied.
 */
function copyDirForPack(src, dest) {
  let count = 0;
  const entries = fs.readdirSync(src, { withFileTypes: true });
  for (const entry of entries) {
    const srcPath = path.join(src, entry.name);
    const destPath = path.join(dest, entry.name);
    if (entry.isSymbolicLink()) {
      core.warning(`[APM Pack] Skipping symlink: ${srcPath}`);
      continue;
    }
    if (entry.isDirectory()) {
      fs.mkdirSync(destPath, { recursive: true });
      count += copyDirForPack(srcPath, destPath);
    } else if (entry.isFile()) {
      fs.mkdirSync(path.dirname(destPath), { recursive: true });
      fs.copyFileSync(srcPath, destPath);
      count++;
    }
  }
  return count;
}

/**
 * List all file paths recursively under dir, relative to dir. Symbolic links skipped.
 *
 * @param {string} dir
 * @returns {string[]}
 */
function listDirForPack(dir) {
  /** @type {string[]} */
  const result = [];
  try {
    const entries = fs.readdirSync(dir, { withFileTypes: true });
    for (const entry of entries) {
      if (entry.isSymbolicLink()) continue;
      if (entry.isDirectory()) {
        const sub = listDirForPack(path.join(dir, entry.name));
        result.push(...sub.map(s => entry.name + "/" + s));
      } else {
        result.push(entry.name);
      }
    }
  } catch {
    // Best-effort listing
  }
  return result;
}

// ---------------------------------------------------------------------------
// Main pack function
// ---------------------------------------------------------------------------

/**
 * @typedef {Object} PackResult
 * @property {string} bundlePath - Absolute path to the created .tar.gz archive.
 * @property {string[]} files - Unique list of bundle file paths (filtered by target).
 * @property {string} target - Effective target used for packing.
 * @property {Record<string, string>} pathMappings - Cross-target path mappings used.
 */

/**
 * Create a self-contained APM bundle from an installed workspace.
 *
 * Mirrors pack_bundle() in packer.py.
 *
 * @param {object} params
 * @param {string} params.workspaceDir - Project root with apm.lock.yaml + installed files.
 * @param {string} params.outputDir - Directory where the bundle archive will be written.
 * @param {string | null} [params.target] - Explicit target, or null to auto-detect.
 * @param {string} [params.format] - Bundle format (default: "apm").
 * @returns {Promise<PackResult>}
 */
async function packBundle({ workspaceDir, outputDir, target = null, format = "apm" }) {
  core.info("=== APM Bundle Pack ===");
  core.info(`[APM Pack] Workspace directory : ${workspaceDir}`);
  core.info(`[APM Pack] Output directory    : ${outputDir}`);

  if (!fs.existsSync(workspaceDir)) {
    throw new Error(`APM workspace directory not found: ${workspaceDir}`);
  }

  // 1. Read apm.yml for package name / version
  const apmYmlPath = path.join(workspaceDir, APM_YML_NAME);
  let pkgName = path.basename(workspaceDir);
  let pkgVersion = "0.0.0";
  if (fs.existsSync(apmYmlPath)) {
    const apmYmlContent = fs.readFileSync(apmYmlPath, "utf-8");
    const info = parseApmYml(apmYmlContent, pkgName);
    pkgName = info.name;
    pkgVersion = info.version;
    core.info(`[APM Pack] Package             : ${pkgName}@${pkgVersion}`);
  } else {
    core.warning(`[APM Pack] ${APM_YML_NAME} not found — using directory name and version 0.0.0`);
  }

  // 2. Read apm.lock.yaml
  const lockfilePath = path.join(workspaceDir, LOCKFILE_NAME);
  if (!fs.existsSync(lockfilePath)) {
    throw new Error(`${LOCKFILE_NAME} not found in workspace: ${workspaceDir}\n` + "Run 'apm install' first to resolve dependencies.");
  }
  const lockfileContent = fs.readFileSync(lockfilePath, "utf-8");
  core.info(`[APM Pack] Lockfile size: ${lockfileContent.length} bytes`);

  // 3. Parse lockfile
  const lockfile = parseAPMLockfile(lockfileContent);
  core.info(`[APM Pack] Lockfile version  : ${lockfile.lockfile_version}`);
  core.info(`[APM Pack] APM version        : ${lockfile.apm_version}`);
  core.info(`[APM Pack] Dependencies       : ${lockfile.dependencies.length}`);

  // 4. Resolve effective target
  const effectiveTarget = detectTarget(workspaceDir, target);
  core.info(`[APM Pack] Target             : ${effectiveTarget}`);

  // 5. Collect deployed_files from all dependencies, filtered by target
  /** @type {string[]} */
  const allDeployed = [];
  for (const dep of lockfile.dependencies) {
    allDeployed.push(...dep.deployed_files);
  }
  const { files: filteredFiles, pathMappings } = filterFilesByTarget(allDeployed, effectiveTarget);

  // Deduplicate while preserving order (mirrors Python's seen set)
  /** @type {Set<string>} */
  const seen = new Set();
  /** @type {string[]} */
  const uniqueFiles = [];
  for (const f of filteredFiles) {
    if (!seen.has(f)) {
      seen.add(f);
      uniqueFiles.push(f);
    }
  }
  core.info(`[APM Pack] Files to bundle (after filter + dedup): ${uniqueFiles.length}`);

  // 6. Verify each path is safe and exists on disk
  const workspaceDirResolved = path.resolve(workspaceDir);
  const missing = [];
  for (const bundlePath of uniqueFiles) {
    assertSafePackPath(bundlePath);
    // For cross-target mapped files, verify the ORIGINAL (on-disk) path
    const diskRelPath = pathMappings[bundlePath] || bundlePath;
    // Strip trailing slash for existence check
    const diskRelPathClean = diskRelPath.endsWith("/") ? diskRelPath.slice(0, -1) : diskRelPath;
    const absPath = path.join(workspaceDirResolved, diskRelPathClean);
    // Guard: destination must stay inside workspace
    const resolvedAbs = path.resolve(absPath);
    if (!resolvedAbs.startsWith(workspaceDirResolved + path.sep) && resolvedAbs !== workspaceDirResolved) {
      throw new Error(`Refusing to pack path that escapes workspace directory: ${JSON.stringify(diskRelPath)}`);
    }
    if (!fs.existsSync(absPath)) {
      missing.push(diskRelPath);
    }
  }
  if (missing.length > 0) {
    throw new Error(`The following deployed files are missing on disk — run 'apm install' to restore them:\n` + missing.map(m => `  - ${m}`).join("\n"));
  }
  core.info(`[APM Pack] All ${uniqueFiles.length} file(s) verified on disk`);

  // 7. Build bundle directory: output/<name>-<version>/
  const bundleDirName = `${pkgName}-${pkgVersion}`;
  const bundleDir = path.join(path.resolve(outputDir), bundleDirName);
  const bundleDirResolved = path.resolve(bundleDir);
  fs.mkdirSync(bundleDir, { recursive: true });
  core.info(`[APM Pack] Bundle directory    : ${bundleDir}`);

  // 8. Copy files preserving directory structure (skip symlinks)
  let copied = 0;
  for (const bundleRelPath of uniqueFiles) {
    const diskRelPath = pathMappings[bundleRelPath] || bundleRelPath;
    const diskRelPathClean = diskRelPath.endsWith("/") ? diskRelPath.slice(0, -1) : diskRelPath;
    const bundleRelPathClean = bundleRelPath.endsWith("/") ? bundleRelPath.slice(0, -1) : bundleRelPath;
    const src = path.join(workspaceDirResolved, diskRelPathClean);
    const dest = path.join(bundleDir, bundleRelPathClean);

    // Defense-in-depth: verify mapped destination stays inside bundle
    assertPackDestInside(dest, bundleDirResolved);

    if (!fs.existsSync(src)) continue;
    const srcLstat = fs.lstatSync(src);
    if (srcLstat.isSymbolicLink()) {
      core.warning(`[APM Pack] Skipping symlink: ${diskRelPath}`);
      continue;
    }

    if (srcLstat.isDirectory() || bundleRelPath.endsWith("/")) {
      core.info(`[APM Pack] Copying directory: ${diskRelPath}${pathMappings[bundleRelPath] ? ` → ${bundleRelPathClean}` : ""}`);
      fs.mkdirSync(dest, { recursive: true });
      const n = copyDirForPack(src, dest);
      core.info(`[APM Pack]   → Copied ${n} file(s) from ${diskRelPath}`);
      copied += n;
    } else {
      core.info(`[APM Pack] Copying file: ${diskRelPath}${pathMappings[bundleRelPath] ? ` → ${bundleRelPathClean}` : ""}`);
      fs.mkdirSync(path.dirname(dest), { recursive: true });
      fs.copyFileSync(src, dest, 0 /* no COPYFILE_EXCL */);
      copied++;
    }
  }
  core.info(`[APM Pack] Done copying: ${copied} file(s)`);

  // 9. Compute mapped_from for pack: header (source prefixes used in cross-target mapping)
  const crossMap = CROSS_TARGET_MAPS[effectiveTarget] || {};
  const usedSrcPrefixes = new Set();
  for (const original of Object.values(pathMappings)) {
    for (const srcPrefix of Object.keys(crossMap)) {
      if (original.startsWith(srcPrefix)) {
        usedSrcPrefixes.add(srcPrefix);
        break;
      }
    }
  }

  // Build per-dep filtered dep list (each dep gets deployed_files filtered by target)
  const filteredDeps = lockfile.dependencies.map(dep => {
    const { files: depFiles } = filterFilesByTarget(dep.deployed_files, effectiveTarget);
    return { ...dep, deployed_files: depFiles };
  });

  // 10. Write enriched apm.lock.yaml to bundle directory
  const packMeta = {
    format,
    target: effectiveTarget,
    packed_at: new Date().toISOString(),
    mapped_from: Array.from(usedSrcPrefixes).sort(),
  };
  const enrichedLockfile = serializeLockfileYaml(lockfile, filteredDeps, packMeta);
  const bundleLockfilePath = path.join(bundleDir, LOCKFILE_NAME);
  fs.writeFileSync(bundleLockfilePath, enrichedLockfile, "utf-8");
  core.info(`[APM Pack] Wrote enriched lockfile: ${bundleLockfilePath}`);

  // Log bundle contents
  const bundleFiles = listDirForPack(bundleDir);
  core.info(`[APM Pack] Bundle contains ${bundleFiles.length} file(s):`);
  bundleFiles.slice(0, 30).forEach(f => core.info(`  ${f}`));
  if (bundleFiles.length > 30) core.info(`  ... and ${bundleFiles.length - 30} more`);

  // 11. Create .tar.gz archive and remove bundle directory
  const archiveName = `${bundleDirName}.tar.gz`;
  const archivePath = path.join(path.resolve(outputDir), archiveName);
  core.info(`[APM Pack] Creating archive: ${archivePath}`);
  await exec.exec("tar", ["-czf", archivePath, "-C", path.resolve(outputDir), bundleDirName]);
  fs.rmSync(bundleDir, { recursive: true, force: true });
  core.info(`[APM Pack] Archive created: ${archivePath}`);

  return {
    bundlePath: archivePath,
    files: uniqueFiles,
    target: effectiveTarget,
    pathMappings,
  };
}

// ---------------------------------------------------------------------------
// Entry point
// ---------------------------------------------------------------------------

/**
 * Main entry point called by the github-script step.
 *
 * Reads configuration from environment variables:
 *   APM_WORKSPACE     – project root with apm.lock.yaml and installed files
 *                       (default: /tmp/gh-aw/apm-workspace)
 *   APM_BUNDLE_OUTPUT – directory where the bundle archive is created
 *                       (default: /tmp/gh-aw/apm-bundle-output)
 *   APM_TARGET        – pack target (default: auto-detect)
 */
async function main() {
  const workspaceDir = process.env.APM_WORKSPACE || "/tmp/gh-aw/apm-workspace";
  const outputDir = process.env.APM_BUNDLE_OUTPUT || "/tmp/gh-aw/apm-bundle-output";
  const target = process.env.APM_TARGET || null;

  core.info("[APM Pack] Starting APM bundle packing");
  core.info(`[APM Pack] APM_WORKSPACE     : ${workspaceDir}`);
  core.info(`[APM Pack] APM_BUNDLE_OUTPUT : ${outputDir}`);
  core.info(`[APM Pack] APM_TARGET        : ${target || "(auto-detect)"}`);

  try {
    fs.mkdirSync(outputDir, { recursive: true });
    const result = await packBundle({ workspaceDir, outputDir, target });

    core.info("[APM Pack] ✅ APM bundle packed successfully");
    core.info(`[APM Pack]    Bundle path     : ${result.bundlePath}`);
    core.info(`[APM Pack]    Files bundled   : ${result.files.length}`);
    core.info(`[APM Pack]    Target          : ${result.target}`);

    core.setOutput("bundle-path", result.bundlePath);
  } catch (err) {
    const msg = err instanceof Error ? err.message : String(err);
    core.error(`[APM Pack] ❌ Failed to pack APM bundle: ${msg}`);
    throw err;
  }
}

module.exports = {
  main,
  packBundle,
  parseApmYml,
  detectTarget,
  filterFilesByTarget,
  serializeLockfileYaml,
  scalarToYaml,
  assertSafePackPath,
  assertPackDestInside,
  copyDirForPack,
  listDirForPack,
};
