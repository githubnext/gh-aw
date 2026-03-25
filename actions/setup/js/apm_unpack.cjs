// @ts-check
/// <reference types="@actions/github-script" />

/**
 * APM Bundle Unpacker
 *
 * JavaScript implementation of the APM (Agent Package Manager) bundle unpack
 * algorithm, equivalent to microsoft/apm unpacker.py.
 *
 * This module extracts and deploys an APM bundle (tar.gz archive) to the
 * GitHub Actions workspace. It replaces the `microsoft/apm-action` restore
 * step in the agent job, removing the external dependency for unpacking.
 *
 * Algorithm (mirrors unpacker.py):
 *   1. Locate the tar.gz bundle in APM_BUNDLE_DIR
 *   2. Extract to a temporary directory (with path-traversal / symlink guards)
 *   3. Locate the single top-level directory inside the extracted archive
 *   4. Read apm.lock.yaml from the bundle
 *   5. Collect the deduplicated deployed_files list from all dependencies
 *   6. Verify that every listed file actually exists in the bundle
 *   7. Copy files (additive, never deletes) to OUTPUT_DIR
 *   8. Clean up the temporary directory
 *
 * Environment variables:
 *   APM_BUNDLE_DIR  – directory containing the *.tar.gz bundle
 *                     (default: /tmp/gh-aw/apm-bundle)
 *   OUTPUT_DIR      – destination directory for deployed files
 *                     (default: GITHUB_WORKSPACE, then process.cwd())
 *
 * @module apm_unpack
 */

const fs = require("fs");
const path = require("path");
const os = require("os");

/** Lockfile filename used by current APM versions. */
const LOCKFILE_NAME = "apm.lock.yaml";

// ---------------------------------------------------------------------------
// YAML parser
// ---------------------------------------------------------------------------

/**
 * Unquote a YAML scalar value produced by PyYAML's safe_dump.
 *
 * Handles:
 *  - single-quoted strings: 'value'
 *  - double-quoted strings: "value"
 *  - null / ~ literals
 *  - boolean literals: true / false
 *  - integers
 *  - bare strings (returned as-is)
 *
 * @param {string} raw
 * @returns {string | number | boolean | null}
 */
function unquoteYaml(raw) {
  if (raw === undefined || raw === null) return null;
  const s = raw.trim();
  if (s === "" || s === "~" || s === "null") return null;
  if (s === "true") return true;
  if (s === "false") return false;
  if (/^-?\d+$/.test(s)) return parseInt(s, 10);
  if (/^-?\d+\.\d+$/.test(s)) return parseFloat(s);
  // Strip surrounding quotes
  if ((s.startsWith("'") && s.endsWith("'")) || (s.startsWith('"') && s.endsWith('"'))) {
    return s.slice(1, -1);
  }
  return s;
}

/**
 * @typedef {Object} LockedDependency
 * @property {string} repo_url
 * @property {string | null} host
 * @property {string | null} resolved_commit
 * @property {string | null} resolved_ref
 * @property {string | null} version
 * @property {string | null} virtual_path
 * @property {boolean} is_virtual
 * @property {number} depth
 * @property {string | null} resolved_by
 * @property {string | null} package_type
 * @property {string[]} deployed_files
 * @property {string | null} source
 * @property {string | null} local_path
 * @property {string | null} content_hash
 * @property {boolean} is_dev
 */

/**
 * @typedef {Object} APMLockfile
 * @property {string | null} lockfile_version
 * @property {string | null} generated_at
 * @property {string | null} apm_version
 * @property {LockedDependency[]} dependencies
 * @property {Record<string, any>} pack
 */

/**
 * Parse an APM lockfile (apm.lock.yaml) from a YAML string.
 *
 * This is a targeted parser for the specific output produced by PyYAML's
 * safe_dump (default_flow_style=False, sort_keys=False).  The format is:
 *
 *   lockfile_version: '1'            <- top-level scalar
 *   dependencies:                    <- top-level sequence key
 *   - repo_url: https://...          <- first key of a mapping item
 *     deployed_files:                <- nested sequence key (2-space indent)
 *     - .github/skills/foo/          <- sequence items (2-space indent)
 *   pack:                            <- top-level mapping key
 *     target: claude                 <- nested scalars (2-space indent)
 *
 * @param {string} content - Raw YAML string content of the lockfile.
 * @returns {APMLockfile}
 */
function parseAPMLockfile(content) {
  /** @type {APMLockfile} */
  const result = {
    lockfile_version: null,
    generated_at: null,
    apm_version: null,
    dependencies: [],
    pack: {},
  };

  const lines = content.split("\n");

  // Parser states
  const STATE_TOP = "top";
  const STATE_DEPS = "dependencies";
  const STATE_DEP_ITEM = "dep_item";
  const STATE_DEPLOYED_FILES = "deployed_files";
  const STATE_PACK = "pack";

  let state = STATE_TOP;
  /** @type {LockedDependency | null} */
  let currentDep = null;

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];

    // Skip blank lines and YAML comments
    if (!line.trim() || line.trim().startsWith("#")) continue;

    switch (state) {
      case STATE_TOP: {
        if (line === "dependencies:") {
          state = STATE_DEPS;
          break;
        }
        if (line === "pack:" || line.startsWith("pack: ")) {
          // pack may be a mapping block ("pack:") or an inline scalar ("pack: value")
          if (line === "pack:") {
            state = STATE_PACK;
          } else {
            const v = line.slice("pack:".length).trim();
            if (v) result.pack["_value"] = unquoteYaml(v);
          }
          break;
        }
        // Top-level scalar: key: value
        const topMatch = line.match(/^([\w-]+):\s*(.*)$/);
        if (topMatch) {
          const k = topMatch[1];
          const v = unquoteYaml(topMatch[2]);
          // @ts-ignore – dynamic key assignment on typed result
          result[k] = v;
        }
        break;
      }

      case STATE_DEPS: {
        if (line.startsWith("- ")) {
          // New dependency mapping item – save previous if any
          if (currentDep) result.dependencies.push(currentDep);
          currentDep = makeEmptyDep();
          state = STATE_DEP_ITEM;
          // The first key is on the same line as "- "
          const m = line.match(/^- ([\w-]+):\s*(.*)$/);
          if (m) assignDepField(currentDep, m[1], unquoteYaml(m[2]));
          break;
        }
        // Exiting dependencies section (non-indented, non-list line)
        if (!line.startsWith(" ")) {
          if (currentDep) {
            result.dependencies.push(currentDep);
            currentDep = null;
          }
          state = STATE_TOP;
          i--; // re-process this line
        }
        break;
      }

      case STATE_DEP_ITEM: {
        if (line.startsWith("- ")) {
          // Start of the next dependency item
          if (currentDep) result.dependencies.push(currentDep);
          currentDep = makeEmptyDep();
          const m = line.match(/^- ([\w-]+):\s*(.*)$/);
          if (m) assignDepField(currentDep, m[1], unquoteYaml(m[2]));
          break;
        }
        // 2-space indented key inside the mapping
        const depKeyMatch = line.match(/^  ([\w-]+):\s*(.*)$/);
        if (depKeyMatch) {
          const k = depKeyMatch[1];
          if (k === "deployed_files") {
            state = STATE_DEPLOYED_FILES;
          } else {
            if (currentDep) assignDepField(currentDep, k, unquoteYaml(depKeyMatch[2]));
          }
          break;
        }
        // Exiting dependencies section
        if (!line.startsWith(" ")) {
          if (currentDep) {
            result.dependencies.push(currentDep);
            currentDep = null;
          }
          state = STATE_TOP;
          i--;
        }
        break;
      }

      case STATE_DEPLOYED_FILES: {
        // deployed_files list items are at 2-space indent: "  - path"
        const fileMatch = line.match(/^  - (.+)$/);
        if (fileMatch) {
          if (currentDep) currentDep.deployed_files.push(String(unquoteYaml(String(fileMatch[1].trim()))));
          break;
        }
        // Any other 2-space key: back to dep_item
        if (line.match(/^  [\w-]+:/)) {
          state = STATE_DEP_ITEM;
          i--; // re-process
          break;
        }
        // New dependency item
        if (line.startsWith("- ")) {
          if (currentDep) result.dependencies.push(currentDep);
          currentDep = makeEmptyDep();
          state = STATE_DEP_ITEM;
          const m = line.match(/^- ([\w-]+):\s*(.*)$/);
          if (m) assignDepField(currentDep, m[1], unquoteYaml(m[2]));
          break;
        }
        // Exiting dependencies
        if (!line.startsWith(" ")) {
          if (currentDep) {
            result.dependencies.push(currentDep);
            currentDep = null;
          }
          state = STATE_TOP;
          i--;
        }
        break;
      }

      case STATE_PACK: {
        const packKeyMatch = line.match(/^  ([\w-]+):\s*(.*)$/);
        if (packKeyMatch) {
          result.pack[packKeyMatch[1]] = unquoteYaml(packKeyMatch[2]);
          break;
        }
        // Exiting pack mapping
        if (!line.startsWith(" ")) {
          state = STATE_TOP;
          i--;
        }
        break;
      }
    }
  }

  // Flush the last dependency
  if (currentDep) result.dependencies.push(currentDep);

  return result;
}

/**
 * @returns {LockedDependency}
 */
function makeEmptyDep() {
  return {
    repo_url: "",
    host: null,
    resolved_commit: null,
    resolved_ref: null,
    version: null,
    virtual_path: null,
    is_virtual: false,
    depth: 1,
    resolved_by: null,
    package_type: null,
    deployed_files: [],
    source: null,
    local_path: null,
    content_hash: null,
    is_dev: false,
  };
}

/**
 * Assign a parsed YAML field to a LockedDependency object.
 * @param {LockedDependency} dep
 * @param {string} key
 * @param {string | number | boolean | null} value
 */
function assignDepField(dep, key, value) {
  switch (key) {
    case "repo_url":
      dep.repo_url = String(value ?? "");
      break;
    case "host":
      dep.host = value !== null ? String(value) : null;
      break;
    case "resolved_commit":
      dep.resolved_commit = value !== null ? String(value) : null;
      break;
    case "resolved_ref":
      dep.resolved_ref = value !== null ? String(value) : null;
      break;
    case "version":
      dep.version = value !== null ? String(value) : null;
      break;
    case "virtual_path":
      dep.virtual_path = value !== null ? String(value) : null;
      break;
    case "is_virtual":
      dep.is_virtual = value === true || value === "true";
      break;
    case "depth":
      dep.depth = typeof value === "number" ? value : parseInt(String(value ?? "1"), 10);
      break;
    case "resolved_by":
      dep.resolved_by = value !== null ? String(value) : null;
      break;
    case "package_type":
      dep.package_type = value !== null ? String(value) : null;
      break;
    case "source":
      dep.source = value !== null ? String(value) : null;
      break;
    case "local_path":
      dep.local_path = value !== null ? String(value) : null;
      break;
    case "content_hash":
      dep.content_hash = value !== null ? String(value) : null;
      break;
    case "is_dev":
      dep.is_dev = value === true || value === "true";
      break;
    default:
      // Unknown field – ignore silently
      break;
  }
}

// ---------------------------------------------------------------------------
// Bundle location helpers
// ---------------------------------------------------------------------------

/**
 * Find the first *.tar.gz file in the given directory.
 *
 * @param {string} bundleDir - Directory that contains the bundle archive.
 * @returns {string} Absolute path to the tar.gz file.
 * @throws {Error} If no bundle file is found.
 */
function findBundleFile(bundleDir) {
  core.info(`[APM Unpack] Scanning bundle directory: ${bundleDir}`);

  if (!fs.existsSync(bundleDir)) {
    throw new Error(`APM bundle directory not found: ${bundleDir}`);
  }

  const entries = fs.readdirSync(bundleDir);
  core.info(`[APM Unpack] Found ${entries.length} entries in bundle directory: ${entries.join(", ")}`);

  const tarGzFiles = entries.filter(e => e.endsWith(".tar.gz"));
  if (tarGzFiles.length === 0) {
    throw new Error(`No *.tar.gz bundle found in ${bundleDir}. ` + `Contents: ${entries.length === 0 ? "(empty)" : entries.join(", ")}`);
  }
  if (tarGzFiles.length > 1) {
    core.warning(`[APM Unpack] Multiple bundles found in ${bundleDir}: ${tarGzFiles.join(", ")}. ` + `Using the first one: ${tarGzFiles[0]}`);
  }

  const bundlePath = path.join(bundleDir, tarGzFiles[0]);
  core.info(`[APM Unpack] Selected bundle: ${bundlePath}`);
  return bundlePath;
}

/**
 * After extracting the tar.gz, locate the inner content directory.
 *
 * The APM packer creates archives with a single top-level directory
 * (e.g. "my-package-1.2.3/") that wraps all bundle contents.
 * If no such single directory exists, the extraction root is returned.
 *
 * @param {string} extractedDir - Root of the extracted archive.
 * @returns {string} Path to the source directory containing apm.lock.yaml.
 */
function findSourceDir(extractedDir) {
  const entries = fs.readdirSync(extractedDir, { withFileTypes: true });
  const dirs = entries.filter(e => e.isDirectory() && !e.isSymbolicLink());

  if (dirs.length === 1 && entries.length === 1) {
    // Single top-level directory: this is the bundle root
    const sourceDir = path.join(extractedDir, dirs[0].name);
    core.info(`[APM Unpack] Bundle root directory: ${sourceDir}`);
    return sourceDir;
  }

  // Multiple entries or no subdirectory: use extractedDir itself
  core.info(`[APM Unpack] No single top-level directory found (${entries.length} entries). ` + `Using extracted root: ${extractedDir}`);
  return extractedDir;
}

/**
 * Locate the lockfile inside the source directory.
 *
 * @param {string} sourceDir
 * @returns {string} Absolute path to the lockfile.
 * @throws {Error} If the lockfile is not found.
 */
function findLockfile(sourceDir) {
  const primary = path.join(sourceDir, LOCKFILE_NAME);
  if (fs.existsSync(primary)) {
    core.info(`[APM Unpack] Found lockfile: ${primary}`);
    return primary;
  }
  // List source dir for debugging
  const entries = fs.readdirSync(sourceDir).join(", ");
  throw new Error(`${LOCKFILE_NAME} not found in bundle. ` + `Source directory (${sourceDir}) contains: ${entries || "(empty)"}`);
}

// ---------------------------------------------------------------------------
// File collection and verification
// ---------------------------------------------------------------------------

/**
 * Walk all dependencies in the lockfile and return a deduplicated, ordered list
 * of deployed_files paths together with a per-dependency map.
 *
 * Mirrors the Python unpacker's collection loop:
 *   for dep in lockfile.get_all_dependencies():
 *       for f in dep.deployed_files:
 *           ...unique_files.append(f)
 *
 * @param {APMLockfile} lockfile
 * @returns {{ uniqueFiles: string[], depFileMap: Record<string, string[]> }}
 */
function collectDeployedFiles(lockfile) {
  /** @type {Set<string>} */
  const seen = new Set();
  /** @type {string[]} */
  const uniqueFiles = [];
  /** @type {Record<string, string[]>} */
  const depFileMap = {};

  for (const dep of lockfile.dependencies) {
    const depKey = dep.is_virtual && dep.virtual_path ? `${dep.repo_url}/${dep.virtual_path}` : dep.source === "local" && dep.local_path ? dep.local_path : dep.repo_url;

    /** @type {string[]} */
    const depFiles = [];
    for (const f of dep.deployed_files) {
      depFiles.push(f);
      if (!seen.has(f)) {
        seen.add(f);
        uniqueFiles.push(f);
      }
    }
    if (depFiles.length > 0) {
      depFileMap[depKey] = depFiles;
    }
  }

  return { uniqueFiles, depFileMap };
}

/**
 * Verify that every file listed in deployed_files actually exists in the bundle.
 *
 * @param {string} sourceDir - Extracted bundle directory.
 * @param {string[]} uniqueFiles - Deduplicated list of relative file paths.
 * @throws {Error} If any listed file is missing from the bundle.
 */
function verifyBundleContents(sourceDir, uniqueFiles) {
  const missing = uniqueFiles.filter(f => {
    const candidate = path.join(sourceDir, f);
    return !fs.existsSync(candidate);
  });

  if (missing.length > 0) {
    throw new Error(`Bundle verification failed – the following deployed files are missing from the bundle:\n` + missing.map(m => `  - ${m}`).join("\n"));
  }
  core.info(`[APM Unpack] Bundle verification passed (${uniqueFiles.length} file(s) verified)`);
}

// ---------------------------------------------------------------------------
// Security helpers
// ---------------------------------------------------------------------------

/**
 * Validate that a relative path from the lockfile is safe to deploy.
 * Rejects absolute paths and path-traversal attempts (mirrors unpacker.py).
 *
 * @param {string} relPath - Relative path string from deployed_files.
 * @throws {Error} If the path is unsafe.
 */
function assertSafePath(relPath) {
  if (path.isAbsolute(relPath) || relPath.startsWith("/")) {
    throw new Error(`Refusing to unpack unsafe absolute path from bundle lockfile: ${JSON.stringify(relPath)}`);
  }
  const parts = relPath.split(/[\\/]/);
  if (parts.includes("..")) {
    throw new Error(`Refusing to unpack path-traversal entry from bundle lockfile: ${JSON.stringify(relPath)}`);
  }
}

/**
 * Verify that the resolved destination path stays within outputDirResolved.
 *
 * @param {string} destPath - Absolute destination path.
 * @param {string} outputDirResolved - Resolved absolute output directory.
 * @throws {Error} If the dest escapes the output directory.
 */
function assertDestInsideOutput(destPath, outputDirResolved) {
  const resolved = path.resolve(destPath);
  if (!resolved.startsWith(outputDirResolved + path.sep) && resolved !== outputDirResolved) {
    throw new Error(`Refusing to unpack path that escapes the output directory: ${JSON.stringify(destPath)}`);
  }
}

// ---------------------------------------------------------------------------
// Copy helpers
// ---------------------------------------------------------------------------

/**
 * Recursively copy a directory tree from src to dest, skipping symbolic links.
 * Parent directories are created automatically.
 *
 * @param {string} src - Source directory.
 * @param {string} dest - Destination directory.
 * @returns {number} Number of files copied.
 */
function copyDirRecursive(src, dest) {
  let count = 0;
  const entries = fs.readdirSync(src, { withFileTypes: true });
  for (const entry of entries) {
    const srcPath = path.join(src, entry.name);
    const destPath = path.join(dest, entry.name);
    if (entry.isSymbolicLink()) {
      // Security: skip symlinks (mirrors unpacker.py's ignore_symlinks)
      core.warning(`[APM Unpack] Skipping symlink: ${srcPath}`);
      continue;
    }
    if (entry.isDirectory()) {
      fs.mkdirSync(destPath, { recursive: true });
      count += copyDirRecursive(srcPath, destPath);
    } else if (entry.isFile()) {
      fs.mkdirSync(path.dirname(destPath), { recursive: true });
      fs.copyFileSync(srcPath, destPath);
      count++;
    }
  }
  return count;
}

// ---------------------------------------------------------------------------
// Main unpack function
// ---------------------------------------------------------------------------

/**
 * @typedef {Object} UnpackResult
 * @property {string} bundlePath - Path to the original bundle archive.
 * @property {string[]} files - Unique list of deployed file paths.
 * @property {boolean} verified - Whether bundle completeness was verified.
 * @property {Record<string, string[]>} dependencyFiles - Files per dependency key.
 * @property {number} skippedCount - Files skipped (symlinks, missing).
 * @property {Record<string, any>} packMeta - Pack metadata from lockfile.
 */

/**
 * Extract and apply an APM bundle to an output directory.
 *
 * This is the core implementation that mirrors the Python unpack_bundle()
 * function in unpacker.py.  All extraction and copying is done with the same
 * additive-only, symlink-skipping, path-traversal-checking semantics.
 *
 * @param {object} params
 * @param {string} params.bundleDir - Directory containing the *.tar.gz bundle.
 * @param {string} params.outputDir - Target directory to copy files into.
 * @param {boolean} [params.skipVerify] - Skip completeness verification.
 * @param {boolean} [params.dryRun] - Resolve file list but write nothing.
 * @returns {Promise<UnpackResult>}
 */
async function unpackBundle({ bundleDir, outputDir, skipVerify = false, dryRun = false }) {
  core.info("=== APM Bundle Unpack ===");
  core.info(`[APM Unpack] Bundle directory : ${bundleDir}`);
  core.info(`[APM Unpack] Output directory : ${outputDir}`);
  core.info(`[APM Unpack] Skip verify      : ${skipVerify}`);
  core.info(`[APM Unpack] Dry run          : ${dryRun}`);

  // 1. Find the archive
  const bundlePath = findBundleFile(bundleDir);

  // 2. Extract to temporary directory
  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "apm-unpack-"));
  core.info(`[APM Unpack] Temp directory   : ${tempDir}`);

  let sourceDir;
  try {
    core.info(`[APM Unpack] Extracting archive: ${bundlePath}`);
    await exec.exec("tar", ["-xzf", bundlePath, "-C", tempDir]);
    core.info(`[APM Unpack] Extraction complete`);

    // 3. Find the inner bundle directory
    sourceDir = findSourceDir(tempDir);

    // List bundle contents for debugging
    const allBundleFiles = listDirRecursive(sourceDir);
    core.info(`[APM Unpack] Bundle contains ${allBundleFiles.length} file(s):`);
    allBundleFiles.slice(0, 50).forEach(f => core.info(`  ${f}`));
    if (allBundleFiles.length > 50) {
      core.info(`  ... and ${allBundleFiles.length - 50} more`);
    }

    // 4. Read lockfile
    const lockfilePath = findLockfile(sourceDir);
    const lockfileContent = fs.readFileSync(lockfilePath, "utf-8");
    core.info(`[APM Unpack] Lockfile size: ${lockfileContent.length} bytes`);

    // 5. Parse lockfile
    const lockfile = parseAPMLockfile(lockfileContent);
    core.info(`[APM Unpack] Lockfile version  : ${lockfile.lockfile_version}`);
    core.info(`[APM Unpack] APM version        : ${lockfile.apm_version}`);
    core.info(`[APM Unpack] Dependencies       : ${lockfile.dependencies.length}`);

    if (lockfile.pack && Object.keys(lockfile.pack).length > 0) {
      core.info(`[APM Unpack] Pack metadata      : ${JSON.stringify(lockfile.pack)}`);
    }

    for (const dep of lockfile.dependencies) {
      core.info(`[APM Unpack]   dep: ${dep.repo_url}` + (dep.resolved_ref ? `@${dep.resolved_ref}` : "") + (dep.resolved_commit ? ` (${dep.resolved_commit.slice(0, 8)})` : "") + ` – ${dep.deployed_files.length} file(s)`);
      dep.deployed_files.forEach(f => core.info(`    → ${f}`));
    }

    // 6. Collect deployed files (deduplicated)
    const { uniqueFiles, depFileMap } = collectDeployedFiles(lockfile);
    core.info(`[APM Unpack] Total deployed files (deduplicated): ${uniqueFiles.length}`);

    // 7. Verify bundle completeness
    if (!skipVerify) {
      verifyBundleContents(sourceDir, uniqueFiles);
    } else {
      core.info("[APM Unpack] Skipping bundle verification (skipVerify=true)");
    }

    const verified = !skipVerify;

    // 8. Dry-run early exit
    if (dryRun) {
      core.info("[APM Unpack] Dry-run mode: resolved file list without writing");
      return {
        bundlePath,
        files: uniqueFiles,
        verified,
        dependencyFiles: depFileMap,
        skippedCount: 0,
        packMeta: lockfile.pack,
      };
    }

    // 9. Copy files to output directory (additive only, never deletes)
    const outputDirResolved = path.resolve(outputDir);
    fs.mkdirSync(outputDirResolved, { recursive: true });

    let skipped = 0;
    let copied = 0;

    for (const relPath of uniqueFiles) {
      // Guard: reject unsafe paths from the lockfile
      assertSafePath(relPath);

      const dest = path.join(outputDirResolved, relPath);
      assertDestInsideOutput(dest, outputDirResolved);

      // Strip trailing slash for path operations (directories end with /)
      const relPathClean = relPath.endsWith("/") ? relPath.slice(0, -1) : relPath;
      const src = path.join(sourceDir, relPathClean);

      if (!fs.existsSync(src)) {
        core.warning(`[APM Unpack] Skipping missing entry: ${relPath}`);
        skipped++;
        continue;
      }

      // Security: skip symlinks
      const srcLstat = fs.lstatSync(src);
      if (srcLstat.isSymbolicLink()) {
        core.warning(`[APM Unpack] Skipping symlink: ${relPath}`);
        skipped++;
        continue;
      }

      if (srcLstat.isDirectory() || relPath.endsWith("/")) {
        core.info(`[APM Unpack] Copying directory: ${relPath}`);
        const destDir = path.join(outputDirResolved, relPathClean);
        fs.mkdirSync(destDir, { recursive: true });
        const n = copyDirRecursive(src, destDir);
        core.info(`[APM Unpack]   → Copied ${n} file(s) from ${relPath}`);
        copied += n;
      } else {
        core.info(`[APM Unpack] Copying file: ${relPath}`);
        fs.mkdirSync(path.dirname(dest), { recursive: true });
        fs.copyFileSync(src, dest);
        copied++;
      }
    }

    core.info(`[APM Unpack] Done: ${copied} file(s) copied, ${skipped} skipped`);
    core.info(`[APM Unpack] Deployed to: ${outputDirResolved}`);

    // Log what was deployed for easy verification
    const deployedFiles = listDirRecursive(outputDirResolved);
    core.info(`[APM Unpack] Output directory now contains ${deployedFiles.length} file(s) (top-level snapshot):`);
    deployedFiles.slice(0, 30).forEach(f => core.info(`  ${f}`));

    return {
      bundlePath,
      files: uniqueFiles,
      verified,
      dependencyFiles: depFileMap,
      skippedCount: skipped,
      packMeta: lockfile.pack,
    };
  } finally {
    // Always clean up temp directory
    try {
      fs.rmSync(tempDir, { recursive: true, force: true });
      core.info(`[APM Unpack] Cleaned up temp directory: ${tempDir}`);
    } catch (cleanupErr) {
      core.warning(`[APM Unpack] Failed to clean up temp directory ${tempDir}: ${cleanupErr}`);
    }
  }
}

/**
 * List all file paths recursively under dir, relative to dir.
 * Symbolic links are skipped.
 *
 * @param {string} dir
 * @returns {string[]}
 */
function listDirRecursive(dir) {
  /** @type {string[]} */
  const result = [];
  try {
    const entries = fs.readdirSync(dir, { withFileTypes: true });
    for (const entry of entries) {
      if (entry.isSymbolicLink()) continue;
      const rel = entry.name;
      if (entry.isDirectory()) {
        const sub = listDirRecursive(path.join(dir, entry.name));
        result.push(...sub.map(s => rel + "/" + s));
      } else {
        result.push(rel);
      }
    }
  } catch {
    // Best-effort listing
  }
  return result;
}

// ---------------------------------------------------------------------------
// Entry point
// ---------------------------------------------------------------------------

/**
 * Main entry point called by the github-script step.
 *
 * Reads configuration from environment variables:
 *   APM_BUNDLE_DIR  – directory with the bundle tar.gz (default: /tmp/gh-aw/apm-bundle)
 *   OUTPUT_DIR      – destination for deployed files (default: GITHUB_WORKSPACE)
 */
async function main() {
  const bundleDir = process.env.APM_BUNDLE_DIR || "/tmp/gh-aw/apm-bundle";
  const outputDir = process.env.OUTPUT_DIR || process.env.GITHUB_WORKSPACE || process.cwd();

  core.info("[APM Unpack] Starting APM bundle unpacking");
  core.info(`[APM Unpack] APM_BUNDLE_DIR  : ${bundleDir}`);
  core.info(`[APM Unpack] OUTPUT_DIR      : ${outputDir}`);

  try {
    const result = await unpackBundle({ bundleDir, outputDir });

    core.info("[APM Unpack] ✅ APM bundle unpacked successfully");
    core.info(`[APM Unpack]    Files deployed  : ${result.files.length}`);
    core.info(`[APM Unpack]    Files skipped   : ${result.skippedCount}`);
    core.info(`[APM Unpack]    Verified        : ${result.verified}`);
  } catch (err) {
    const msg = err instanceof Error ? err.message : String(err);
    core.error(`[APM Unpack] ❌ Failed to unpack APM bundle: ${msg}`);
    throw err;
  }
}

module.exports = {
  main,
  unpackBundle,
  parseAPMLockfile,
  unquoteYaml,
  collectDeployedFiles,
  findBundleFile,
  findSourceDir,
  findLockfile,
  verifyBundleContents,
  assertSafePath,
  assertDestInsideOutput,
  copyDirRecursive,
  listDirRecursive,
};
