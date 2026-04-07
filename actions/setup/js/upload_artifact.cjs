// @ts-check
/// <reference types="@actions/github-script" />

/**
 * upload_artifact handler
 *
 * Validates and stages artifact upload requests emitted by the model via the upload_artifact
 * safe output tool. The model must have already copied the files it wants to upload to
 * ${RUNNER_TEMP}/gh-aw/safeoutputs/upload-artifacts/ before calling the tool.
 *
 * This handler:
 * 1. Reads upload_artifact records from agent output.
 * 2. Validates each request against the workflow's policy configuration.
 * 3. Resolves the requested files (path or filter-based) from the staging directory.
 * 4. Copies approved files into per-slot directories under ${RUNNER_TEMP}/gh-aw/upload-artifacts/slot_N/.
 * 5. Sets step outputs so the wrapping job's actions/upload-artifact steps can run conditionally.
 * 6. Generates a temporary artifact ID for each slot.
 *
 * Environment variables consumed (set by the Go job builder):
 *   GH_AW_ARTIFACT_MAX_UPLOADS           - Max number of upload_artifact calls allowed
 *   GH_AW_ARTIFACT_DEFAULT_RETENTION_DAYS - Default retention period
 *   GH_AW_ARTIFACT_MAX_RETENTION_DAYS    - Maximum retention cap
 *   GH_AW_ARTIFACT_MAX_SIZE_BYTES        - Maximum total bytes per upload
 *   GH_AW_ARTIFACT_ALLOWED_PATHS         - JSON array of allowed path patterns
 *   GH_AW_ARTIFACT_ALLOW_SKIP_ARCHIVE    - "true" if skip_archive is permitted
 *   GH_AW_ARTIFACT_DEFAULT_SKIP_ARCHIVE  - "true" if skip_archive defaults to true
 *   GH_AW_ARTIFACT_DEFAULT_IF_NO_FILES   - "error" or "ignore"
 *   GH_AW_ARTIFACT_FILTERS_INCLUDE       - JSON array of default include patterns
 *   GH_AW_ARTIFACT_FILTERS_EXCLUDE       - JSON array of default exclude patterns
 *   GH_AW_AGENT_OUTPUT                   - Path to agent output file
 *   GH_AW_SAFE_OUTPUTS_STAGED            - "true" for staged/dry-run mode
 */

const fs = require("fs");
const path = require("path");
const crypto = require("crypto");
const { loadAgentOutput } = require("./load_agent_output.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { globPatternToRegex } = require("./glob_pattern_helpers.cjs");
const { ERR_CONFIG, ERR_SYSTEM, ERR_VALIDATION } = require("./error_codes.cjs");

/** Staging directory where the model places files to be uploaded. */
const STAGING_DIR = `${process.env.RUNNER_TEMP}/gh-aw/safeoutputs/upload-artifacts/`;

/** Base directory for per-slot artifact staging used by actions/upload-artifact. */
const SLOT_BASE_DIR = `${process.env.RUNNER_TEMP}/gh-aw/upload-artifacts/`;

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
 * Parse a JSON array from an environment variable, returning an empty array on failure.
 * @param {string|undefined} envVar
 * @returns {string[]}
 */
function parseJsonArrayEnv(envVar) {
  if (!envVar) return [];
  try {
    const parsed = JSON.parse(envVar);
    return Array.isArray(parsed) ? parsed.filter(v => typeof v === "string") : [];
  } catch {
    return [];
  }
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
 * Copy resolved files from STAGING_DIR into the per-slot directory.
 * @param {string[]} files - Relative paths from STAGING_DIR
 * @param {string} slotDir - Absolute target slot directory
 */
function stageFilesToSlot(files, slotDir) {
  fs.mkdirSync(slotDir, { recursive: true });
  for (const relPath of files) {
    const src = path.join(STAGING_DIR, relPath);
    const dest = path.join(slotDir, relPath);
    fs.mkdirSync(path.dirname(dest), { recursive: true });
    fs.copyFileSync(src, dest);
  }
}

async function main() {
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  // Load policy configuration from environment variables.
  const maxUploads = parseInt(process.env.GH_AW_ARTIFACT_MAX_UPLOADS || "1", 10) || 1;
  const defaultRetentionDays = parseInt(process.env.GH_AW_ARTIFACT_DEFAULT_RETENTION_DAYS || "7", 10) || 7;
  const maxRetentionDays = parseInt(process.env.GH_AW_ARTIFACT_MAX_RETENTION_DAYS || "30", 10) || 30;
  const maxSizeBytes = parseInt(process.env.GH_AW_ARTIFACT_MAX_SIZE_BYTES || "104857600", 10) || 104857600;
  const allowSkipArchive = process.env.GH_AW_ARTIFACT_ALLOW_SKIP_ARCHIVE === "true";
  const defaultSkipArchive = process.env.GH_AW_ARTIFACT_DEFAULT_SKIP_ARCHIVE === "true";
  const defaultIfNoFiles = process.env.GH_AW_ARTIFACT_DEFAULT_IF_NO_FILES || "error";
  const allowedPaths = parseJsonArrayEnv(process.env.GH_AW_ARTIFACT_ALLOWED_PATHS);
  const filtersInclude = parseJsonArrayEnv(process.env.GH_AW_ARTIFACT_FILTERS_INCLUDE);
  const filtersExclude = parseJsonArrayEnv(process.env.GH_AW_ARTIFACT_FILTERS_EXCLUDE);

  core.info(`upload_artifact handler: max_uploads=${maxUploads}, default_retention=${defaultRetentionDays}, max_retention=${maxRetentionDays}`);
  core.info(`Allowed paths: ${allowedPaths.length > 0 ? allowedPaths.join(", ") : "(none – all staging files allowed)"}`);

  // Load agent output to find upload_artifact records.
  const result = loadAgentOutput();
  if (!result.success) {
    core.info("No agent output found, skipping upload_artifact processing");
    core.setOutput("artifact_count", "0");
    return;
  }

  const uploadRequests = result.items.filter(/** @param {any} item */ item => item.type === "upload_artifact");

  if (uploadRequests.length === 0) {
    core.info("No upload_artifact records in agent output");
    core.setOutput("artifact_count", "0");
    return;
  }

  core.info(`Found ${uploadRequests.length} upload_artifact request(s)`);

  // Enforce max-uploads policy.
  if (uploadRequests.length > maxUploads) {
    core.setFailed(`${ERR_VALIDATION}: upload_artifact: ${uploadRequests.length} requests exceed max-uploads policy (${maxUploads}). Reduce the number of upload_artifact calls or raise max-uploads in workflow configuration.`);
    return;
  }

  if (!fs.existsSync(STAGING_DIR)) {
    core.warning(`Staging directory ${STAGING_DIR} does not exist. Did the model copy files there before calling upload_artifact?`);
    fs.mkdirSync(STAGING_DIR, { recursive: true });
  }

  /** @type {Record<string, string>} resolver: tmpId → artifact name */
  const resolver = {};

  let successfulUploads = 0;

  for (let i = 0; i < uploadRequests.length; i++) {
    const request = uploadRequests[i];
    core.info(`Processing upload_artifact request ${i + 1}/${uploadRequests.length}`);

    // Resolve skip_archive.
    const skipArchive = typeof request.skip_archive === "boolean" ? request.skip_archive : defaultSkipArchive;
    if (skipArchive && !allowSkipArchive) {
      core.setFailed(`${ERR_VALIDATION}: upload_artifact request ${i + 1}: skip_archive=true is not permitted. Enable it with allow.skip-archive: true in workflow configuration.`);
      return;
    }

    // Resolve files.
    const { files, error: resolveError } = resolveFiles(request, allowedPaths, filtersInclude, filtersExclude);
    if (resolveError) {
      core.setFailed(`${ERR_VALIDATION}: upload_artifact request ${i + 1}: ${resolveError}`);
      return;
    }

    if (files.length === 0) {
      if (defaultIfNoFiles === "ignore") {
        core.warning(`upload_artifact request ${i + 1}: no files matched, skipping (if-no-files=ignore)`);
        continue;
      } else {
        core.setFailed(`${ERR_VALIDATION}: upload_artifact request ${i + 1}: no files matched the selection criteria. Check allowed-paths, filters, or use defaults.if-no-files: ignore to skip empty uploads.`);
        return;
      }
    }

    // Validate skip_archive file-count constraint.
    const skipArchiveError = validateSkipArchive(skipArchive, files);
    if (skipArchiveError) {
      core.setFailed(`${ERR_VALIDATION}: upload_artifact request ${i + 1}: ${skipArchiveError}`);
      return;
    }

    // Validate total size.
    const totalSize = computeTotalSize(files);
    if (totalSize > maxSizeBytes) {
      core.setFailed(`${ERR_VALIDATION}: upload_artifact request ${i + 1}: total file size ${totalSize} bytes exceeds max-size-bytes limit of ${maxSizeBytes} bytes.`);
      return;
    }

    // Compute retention days.
    const retentionDays = clampRetention(typeof request.retention_days === "number" ? request.retention_days : undefined, defaultRetentionDays, maxRetentionDays);

    // Derive artifact name and generate temporary ID.
    const artifactName = deriveArtifactName(request, i);
    const tmpId = generateTemporaryArtifactId();
    resolver[tmpId] = artifactName;

    core.info(`Slot ${i}: artifact="${artifactName}", files=${files.length}, size=${totalSize}B, retention=${retentionDays}d, skip_archive=${skipArchive}, tmp_id=${tmpId}`);

    if (!isStaged) {
      // Stage files into the per-slot directory for the actions/upload-artifact step.
      const slotDir = path.join(SLOT_BASE_DIR, `slot_${i}`);
      stageFilesToSlot(files, slotDir);
      core.info(`Staged ${files.length} file(s) to ${slotDir}`);
    } else {
      core.info(`Staged mode: skipping file staging for slot ${i}`);
    }

    // Set step outputs for the conditional actions/upload-artifact steps in the job YAML.
    core.setOutput(`slot_${i}_enabled`, "true");
    core.setOutput(`slot_${i}_name`, artifactName);
    core.setOutput(`slot_${i}_retention_days`, String(retentionDays));
    core.setOutput(`slot_${i}_tmp_id`, tmpId);
    core.setOutput(`slot_${i}_file_count`, String(files.length));
    core.setOutput(`slot_${i}_size_bytes`, String(totalSize));

    successfulUploads++;
  }

  // Write resolver mapping so downstream steps can resolve tmp IDs to artifact names.
  try {
    fs.mkdirSync(path.dirname(RESOLVER_FILE), { recursive: true });
    fs.writeFileSync(RESOLVER_FILE, JSON.stringify(resolver, null, 2));
    core.info(`Wrote artifact resolver mapping to ${RESOLVER_FILE}`);
  } catch (err) {
    core.warning(`Failed to write artifact resolver file: ${getErrorMessage(err)}`);
  }

  core.setOutput("artifact_count", String(successfulUploads));
  core.info(`upload_artifact handler complete: ${successfulUploads} artifact(s) staged`);

  if (isStaged) {
    core.summary.addHeading("🎭 Staged Mode: Artifact Upload Preview", 2);
    core.summary.addRaw(`Would upload **${successfulUploads}** artifact(s). Files staged at ${STAGING_DIR}.`);
    await core.summary.write();
  }
}

module.exports = { main };
