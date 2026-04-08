// @ts-check
/// <reference types="@actions/github-script" />

/**
 * upload_artifact handler
 *
 * Validates artifact upload requests emitted by the model via the upload_artifact safe output
 * tool, then uploads the approved files directly via the @actions/artifact REST API client.
 * The model must have already copied the files it wants to upload to
 * ${RUNNER_TEMP}/gh-aw/safeoutputs/upload-artifacts/ before calling the tool.
 *
 * This handler follows the per-message handler pattern used by the safe_outputs handler loop.
 * main(config) returns a per-message handler function that:
 * 1. Validates the request against the workflow's policy configuration.
 * 2. Resolves the requested files (path or filter-based) from the staging directory.
 * 3. Uploads the approved files directly via DefaultArtifactClient.uploadArtifact().
 * 4. Sets step outputs (slot_N_tmp_id, upload_artifact_count) for downstream consumers.
 * 5. Generates a temporary artifact ID for each upload and writes a resolver file.
 *
 * Configuration keys (passed via config parameter from handler manager):
 *   max-uploads           - Max number of upload_artifact calls allowed (default: 1)
 *   default-retention-days - Default retention period (default: 7)
 *   max-retention-days    - Maximum retention cap (default: 30)
 *   max-size-bytes        - Maximum total bytes per upload (default: 100 MB)
 *   allowed-paths         - Array of allowed path glob patterns
 *   allow-skip-archive    - true if skip_archive is permitted
 *   default-skip-archive  - true if skip_archive defaults to true
 *   default-if-no-files   - "error" or "ignore" (default: "error")
 *   filters-include       - Array of default include glob patterns
 *   filters-exclude       - Array of default exclude glob patterns
 *   staged                - true for staged/dry-run mode (skips actual upload)
 */

const fs = require("fs");
const path = require("path");
const { getErrorMessage } = require("./error_helpers.cjs");
const { globPatternToRegex } = require("./glob_pattern_helpers.cjs");
const { ERR_VALIDATION } = require("./error_codes.cjs");

/** Staging directory where the model places files to be uploaded. */
const STAGING_DIR = `${process.env.RUNNER_TEMP}/gh-aw/safeoutputs/upload-artifacts/`;

/** Prefix for temporary artifact IDs returned to the caller. */
const TEMP_ID_PREFIX = "tmp_artifact_";

/** Path where the resolver mapping (tmpId → artifact name) is written. */
const RESOLVER_FILE = `${process.env.RUNNER_TEMP}/gh-aw/artifact-resolver.json`;

/**
 * Generate a temporary artifact ID.
 * Format: tmp_artifact_<26 uppercase alphanumeric characters>
 * @returns {string}
 */
function generateTemporaryArtifactId() {
  const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";
  let id = TEMP_ID_PREFIX;
  for (let i = 0; i < 26; i++) {
    id += chars[Math.floor(Math.random() * chars.length)];
  }
  return id;
}

/**
 * Check whether a relative path matches any of the provided glob patterns.
 * @param {string} relPath - Path relative to the staging root
 * @param {string[]} patterns
 * @returns {boolean}
 */
function matchesAnyPattern(relPath, patterns) {
  if (patterns.length === 0) return false;
  return patterns.some(pattern => {
    const regex = globPatternToRegex(pattern);
    return regex.test(relPath);
  });
}

/**
 * Validate that a path does not escape the staging root using traversal sequences.
 * @param {string} filePath - Absolute path
 * @param {string} root - Absolute root directory (must end with /)
 * @returns {boolean}
 */
function isWithinRoot(filePath, root) {
  const resolved = path.resolve(filePath);
  const normalRoot = path.resolve(root);
  return resolved.startsWith(normalRoot + path.sep) || resolved === normalRoot;
}

/**
 * Recursively list all regular files under a directory.
 * @param {string} dir - Absolute directory path
 * @param {string} baseDir - Root used to compute relative paths
 * @returns {string[]} Relative paths from baseDir
 */
function listFilesRecursive(dir, baseDir) {
  /** @type {string[]} */
  const files = [];
  if (!fs.existsSync(dir)) return files;

  const entries = fs.readdirSync(dir, { withFileTypes: true });
  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      files.push(...listFilesRecursive(fullPath, baseDir));
    } else if (entry.isFile()) {
      // Reject symlinks – entry.isFile() returns false for symlinks unless dereferenced.
      // We check explicitly to avoid following symlinks.
      const stat = fs.lstatSync(fullPath);
      if (!stat.isSymbolicLink()) {
        files.push(path.relative(baseDir, fullPath));
      } else {
        core.warning(`Skipping symlink: ${fullPath}`);
      }
    }
  }
  return files;
}

/**
 * Resolve the list of files to upload for a single request.
 * Applies: staging root → allowed-paths → request include/exclude → dedup + sort.
 *
 * @param {Record<string, any>} request - Parsed upload_artifact record
 * @param {string[]} allowedPaths - Policy allowed-paths patterns
 * @param {string[]} defaultInclude - Policy default include patterns
 * @param {string[]} defaultExclude - Policy default exclude patterns
 * @returns {{ files: string[], error: string|null }}
 */
function resolveFiles(request, allowedPaths, defaultInclude, defaultExclude) {
  const hasMutuallyExclusive = ("path" in request ? 1 : 0) + ("filters" in request ? 1 : 0);
  if (hasMutuallyExclusive !== 1) {
    return { files: [], error: "exactly one of 'path' or 'filters' must be present" };
  }

  /** @type {string[]} candidateRelPaths */
  let candidateRelPaths;

  if ("path" in request) {
    const reqPath = String(request.path);
    // Reject absolute paths
    if (path.isAbsolute(reqPath)) {
      return { files: [], error: `path must be relative (staging-dir-relative), got absolute path: ${reqPath}` };
    }
    // Reject traversal
    const resolved = path.resolve(STAGING_DIR, reqPath);
    if (!isWithinRoot(resolved, STAGING_DIR)) {
      return { files: [], error: `path must not traverse outside staging directory: ${reqPath}` };
    }
    if (!fs.existsSync(resolved)) {
      return { files: [], error: `path does not exist in staging directory: ${reqPath}` };
    }
    const stat = fs.lstatSync(resolved);
    if (stat.isSymbolicLink()) {
      return { files: [], error: `symlinks are not allowed: ${reqPath}` };
    }
    if (stat.isDirectory()) {
      candidateRelPaths = listFilesRecursive(resolved, STAGING_DIR);
    } else {
      candidateRelPaths = [reqPath];
    }
  } else {
    // Filter-based selection: start from all files in the staging directory.
    const allFiles = listFilesRecursive(STAGING_DIR, STAGING_DIR);
    const requestFilters = request.filters || {};
    const include = /** @type {string[]} */ requestFilters.include || defaultInclude;
    const exclude = /** @type {string[]} */ requestFilters.exclude || defaultExclude;

    candidateRelPaths = allFiles.filter(f => {
      if (include.length > 0 && !matchesAnyPattern(f, include)) return false;
      if (exclude.length > 0 && matchesAnyPattern(f, exclude)) return false;
      return true;
    });
  }

  // Apply allowed-paths policy filter.
  if (allowedPaths.length > 0) {
    candidateRelPaths = candidateRelPaths.filter(f => matchesAnyPattern(f, allowedPaths));
  }

  // Deduplicate and sort deterministically.
  const unique = Array.from(new Set(candidateRelPaths)).sort();
  return { files: unique, error: null };
}

/**
 * Validate skip_archive constraints:
 * - skip_archive may only be used for a single file.
 * - directories are rejected (already expanded to file list).
 *
 * @param {boolean} skipArchive
 * @param {string[]} files
 * @returns {string|null} Error message or null
 */
function validateSkipArchive(skipArchive, files) {
  if (!skipArchive) return null;
  if (files.length !== 1) {
    return `skip_archive=true requires exactly one selected file, but ${files.length} files matched`;
  }
  return null;
}

/**
 * Compute total size of the given file list (relative paths from STAGING_DIR).
 * @param {string[]} files
 * @returns {number} Total size in bytes
 */
function computeTotalSize(files) {
  let total = 0;
  for (const f of files) {
    const abs = path.join(STAGING_DIR, f);
    try {
      total += fs.statSync(abs).size;
    } catch {
      // Ignore missing files (already validated upstream).
    }
  }
  return total;
}

/**
 * Derive a sanitised artifact name from a path or a slot index.
 * @param {Record<string, any>} request
 * @param {number} slotIndex
 * @returns {string}
 */
function deriveArtifactName(request, slotIndex) {
  if (typeof request.name === "string" && request.name.trim()) {
    return request.name.trim().replace(/[^a-zA-Z0-9._\-]/g, "-");
  }
  if ("path" in request && typeof request.path === "string") {
    const base = path.basename(String(request.path)).replace(/[^a-zA-Z0-9._\-]/g, "-");
    if (base) return base;
  }
  return `artifact-slot-${slotIndex}`;
}

/**
 * Clamp a retention value between 1 and the policy maximum.
 * @param {number|undefined} requested
 * @param {number} defaultDays
 * @param {number} maxDays
 * @returns {number}
 */
function clampRetention(requested, defaultDays, maxDays) {
  if (typeof requested !== "number" || requested < 1) return defaultDays;
  return Math.min(requested, maxDays);
}

/**
 * Create or return the @actions/artifact DefaultArtifactClient.
 * global.__createArtifactClient can be set in tests to inject a mock client factory.
 * Uses dynamic import() because @actions/artifact v2+ is an ES module.
 * @returns {Promise<{ uploadArtifact: (name: string, files: string[], rootDir: string, opts: object) => Promise<{id?: number, size?: number}> }>}
 */
async function getArtifactClient() {
  if (typeof global.__createArtifactClient === "function") {
    return global.__createArtifactClient();
  }
  const { DefaultArtifactClient } = await import("@actions/artifact");
  return new DefaultArtifactClient();
}

/**
 * Returns a per-message handler function that processes a single upload_artifact request.
 *
 * @param {Object} config - Handler configuration from the safe outputs config
 * @returns {Promise<Function>} Per-message handler function
 */
async function main(config = {}) {
  const maxUploads = typeof config["max-uploads"] === "number" ? config["max-uploads"] : 1;
  const defaultRetentionDays = typeof config["default-retention-days"] === "number" ? config["default-retention-days"] : 7;
  const maxRetentionDays = typeof config["max-retention-days"] === "number" ? config["max-retention-days"] : 30;
  const maxSizeBytes = typeof config["max-size-bytes"] === "number" ? config["max-size-bytes"] : 104857600;
  const allowSkipArchive = config["allow-skip-archive"] === true;
  const defaultSkipArchive = config["default-skip-archive"] === true;
  const defaultIfNoFiles = typeof config["default-if-no-files"] === "string" ? config["default-if-no-files"] : "error";
  const allowedPaths = Array.isArray(config["allowed-paths"]) ? config["allowed-paths"] : [];
  const filtersInclude = Array.isArray(config["filters-include"]) ? config["filters-include"] : [];
  const filtersExclude = Array.isArray(config["filters-exclude"]) ? config["filters-exclude"] : [];
  const isStaged = config["staged"] === true || process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  core.info(`upload_artifact handler: max_uploads=${maxUploads}, default_retention=${defaultRetentionDays}, max_retention=${maxRetentionDays}`);
  core.info(`Allowed paths: ${allowedPaths.length > 0 ? allowedPaths.join(", ") : "(none – all staging files allowed)"}`);

  // Slot index tracks which slot each successful request maps to.
  let slotIndex = 0;

  /** @type {Record<string, string>} resolver: tmpId → artifact name */
  const resolver = {};

  /**
   * Per-message handler: processes one upload_artifact request.
   *
   * Called by the safe_outputs handler manager for each `upload_artifact` message emitted
   * by the model. State (slotIndex, resolver) is shared across calls via closure so that
   * successive requests are assigned to sequential slot directories.
   *
   * @param {Object} message - The upload_artifact message from the model
   * @param {Object} resolvedTemporaryIds - Map of already-resolved temporary IDs (unused here)
   * @param {Map<string, any>} temporaryIdMap - Shared temp-ID map; the handler does not modify it
   * @returns {Promise<{success: boolean, error?: string, skipped?: boolean, tmpId?: string, artifactName?: string, slotIndex?: number}>}
   */
  return async function handleUploadArtifact(message, resolvedTemporaryIds, temporaryIdMap) {
    if (slotIndex >= maxUploads) {
      return {
        success: false,
        error: `${ERR_VALIDATION}: upload_artifact: exceeded max-uploads policy (${maxUploads}). Reduce the number of upload_artifact calls or raise max-uploads in workflow configuration.`,
      };
    }

    const i = slotIndex;

    // Resolve skip_archive.
    const skipArchive = typeof message.skip_archive === "boolean" ? message.skip_archive : defaultSkipArchive;
    if (skipArchive && !allowSkipArchive) {
      return {
        success: false,
        error: `${ERR_VALIDATION}: upload_artifact: skip_archive=true is not permitted. Enable it with allow.skip-archive: true in workflow configuration.`,
      };
    }

    // Resolve files.
    const { files, error: resolveError } = resolveFiles(message, allowedPaths, filtersInclude, filtersExclude);
    if (resolveError) {
      return { success: false, error: `${ERR_VALIDATION}: upload_artifact: ${resolveError}` };
    }

    if (files.length === 0) {
      if (defaultIfNoFiles === "ignore") {
        core.warning(`upload_artifact: no files matched, skipping (if-no-files=ignore)`);
        return { success: false, skipped: true, error: "No files matched the selection criteria" };
      }
      return {
        success: false,
        error: `${ERR_VALIDATION}: upload_artifact: no files matched the selection criteria. Check allowed-paths, filters, or use defaults.if-no-files: ignore to skip empty uploads.`,
      };
    }

    // Validate skip_archive file-count constraint.
    const skipArchiveError = validateSkipArchive(skipArchive, files);
    if (skipArchiveError) {
      return { success: false, error: `${ERR_VALIDATION}: upload_artifact: ${skipArchiveError}` };
    }

    // Validate total size.
    const totalSize = computeTotalSize(files);
    if (totalSize > maxSizeBytes) {
      return {
        success: false,
        error: `${ERR_VALIDATION}: upload_artifact: total file size ${totalSize} bytes exceeds max-size-bytes limit of ${maxSizeBytes} bytes.`,
      };
    }

    // Compute retention days.
    const retentionDays = clampRetention(typeof message.retention_days === "number" ? message.retention_days : undefined, defaultRetentionDays, maxRetentionDays);

    // Derive artifact name and generate temporary ID.
    const artifactName = deriveArtifactName(message, i);
    const tmpId = generateTemporaryArtifactId();
    resolver[tmpId] = artifactName;

    core.info(`Slot ${i}: artifact="${artifactName}", files=${files.length}, size=${totalSize}B, retention=${retentionDays}d, skip_archive=${skipArchive}, tmp_id=${tmpId}`);

    if (!isStaged) {
      // Upload files directly via @actions/artifact REST API.
      const absoluteFiles = files.map(f => path.join(STAGING_DIR, f));
      const client = await getArtifactClient();
      try {
        const uploadResult = await client.uploadArtifact(artifactName, absoluteFiles, STAGING_DIR, { retentionDays });
        core.info(`Uploaded artifact "${artifactName}" (id=${uploadResult.id ?? "n/a"}, size=${uploadResult.size ?? totalSize}B)`);
      } catch (err) {
        return {
          success: false,
          error: `${ERR_VALIDATION}: upload_artifact: failed to upload artifact "${artifactName}": ${getErrorMessage(err)}`,
        };
      }
    } else {
      core.info(`Staged mode: skipping artifact upload for slot ${i}`);
    }

    // Set step outputs so downstream jobs can reference the tmp ID.
    core.setOutput(`slot_${i}_tmp_id`, tmpId);
    core.setOutput(`slot_${i}_file_count`, String(files.length));
    core.setOutput(`slot_${i}_size_bytes`, String(totalSize));

    slotIndex++;

    // Update the count output.
    core.setOutput("upload_artifact_count", String(slotIndex));

    // Write/update resolver mapping so downstream steps can resolve tmp IDs to artifact names.
    try {
      fs.mkdirSync(path.dirname(RESOLVER_FILE), { recursive: true });
      fs.writeFileSync(RESOLVER_FILE, JSON.stringify(resolver, null, 2));
      core.info(`Wrote artifact resolver mapping to ${RESOLVER_FILE}`);
    } catch (err) {
      core.warning(`Failed to write artifact resolver file: ${getErrorMessage(err)}`);
    }

    return {
      success: true,
      tmpId,
      artifactName,
      slotIndex: i,
    };
  };
}

module.exports = { main };
