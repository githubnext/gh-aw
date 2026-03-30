// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Check compile-agentic version against the remote update configuration.
 *
 * This script:
 * 1. Reads the compiled version from GH_AW_COMPILED_VERSION env var.
 * 2. Fetches .github/aw/config.json from the gh-aw repository via raw.githubusercontent.com.
 * 3. If the download fails, the check is skipped (soft failure).
 * 4. Validates that the compiled version is not in the blocked list.
 * 5. Validates that the compiled version meets the minimum supported version.
 *
 * Fails the activation job when validation fails.
 */

const CONFIG_URL = "https://raw.githubusercontent.com/github/gh-aw/main/.github/aw/config.json";

/**
 * Normalize a version string by stripping the leading "v" prefix if present.
 * This ensures "v1.0.0" and "1.0.0" are treated as equivalent.
 *
 * @param {string} version
 * @returns {string}
 */
function normalizeVersion(version) {
  return version.startsWith("v") ? version.slice(1) : version;
}

/**
 * Parse a semver-like version string into an array of numeric parts.
 * Strips a leading "v" if present.
 * Returns null if the string is not a valid version.
 *
 * @param {string} version
 * @returns {number[]|null}
 */
function parseVersion(version) {
  const stripped = version.startsWith("v") ? version.slice(1) : version;
  const parts = stripped.split(".");
  if (parts.length < 3) return null;
  const nums = parts.slice(0, 3).map(Number);
  if (nums.some(isNaN)) return null;
  return nums;
}

/**
 * Compare two semver-like version strings.
 * Returns a negative number if a < b, 0 if equal, positive if a > b.
 *
 * @param {string} a
 * @param {string} b
 * @returns {number}
 */
function compareVersions(a, b) {
  const pa = parseVersion(a);
  const pb = parseVersion(b);
  if (!pa || !pb) return 0;
  for (let i = 0; i < 3; i++) {
    if (pa[i] !== pb[i]) return pa[i] - pb[i];
  }
  return 0;
}

/**
 * @typedef {object} UpdateConfig
 * @property {string[]} [blockedVersions]
 * @property {string} [minimumVersion]
 */

/**
 * Main entry point.
 */
async function main() {
  const compiledVersion = process.env.GH_AW_COMPILED_VERSION || "";

  if (!compiledVersion || compiledVersion === "dev") {
    core.info(`Skipping version update check: version is '${compiledVersion || "(empty)"}' (dev build)`);
    return;
  }

  core.info(`Checking compile-agentic version: ${compiledVersion}`);
  core.info(`Fetching update configuration from: ${CONFIG_URL}`);

  /** @type {UpdateConfig} */
  let config;
  try {
    const res = await fetch(CONFIG_URL);
    if (!res.ok) {
      throw new Error(`HTTP ${res.status} fetching ${CONFIG_URL}`);
    }
    config = JSON.parse(await res.text());
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    core.info(`Could not fetch update configuration (${message}). Skipping version check.`);
    return;
  }

  const blockedVersions = Array.isArray(config.blockedVersions) ? config.blockedVersions : [];
  const minimumVersion = typeof config.minimumVersion === "string" ? config.minimumVersion : "";

  // Check blocked versions (normalize both sides to ignore leading "v" prefix)
  const normalizedCompiled = normalizeVersion(compiledVersion);
  if (blockedVersions.some(v => normalizeVersion(v) === normalizedCompiled)) {
    core.summary
      .addRaw("### ❌ Blocked compile-agentic version\n\n")
      .addRaw(`The compile-agentic version \`${compiledVersion}\` is **blocked** and cannot be used to run workflows.\n\n`)
      .addRaw("This version has been revoked, typically due to a security issue.\n\n")
      .addRaw("**Action required:** Update `gh-aw` to the latest version and recompile your workflow with `gh aw compile`.\n");
    await core.summary.write();
    core.setFailed(`Blocked compile-agentic version: ${compiledVersion} is in the blocked versions list. Update gh-aw to the latest version and recompile your workflow.`);
    return;
  }

  // Check minimum version
  if (minimumVersion && parseVersion(minimumVersion) !== null) {
    if (compareVersions(compiledVersion, minimumVersion) < 0) {
      core.summary
        .addRaw("### ❌ Outdated compile-agentic version\n\n")
        .addRaw(`The compile-agentic version \`${compiledVersion}\` is below the minimum supported version \`${minimumVersion}\`.\n\n`)
        .addRaw("**Action required:** Update `gh-aw` to the latest version and recompile your workflow with `gh aw compile`.\n");
      await core.summary.write();
      core.setFailed(`Outdated compile-agentic version: ${compiledVersion} is below the minimum supported version ${minimumVersion}. Update gh-aw to the latest version and recompile your workflow.`);
      return;
    }
  }

  core.info(`✅ Version check passed: ${compiledVersion}`);
}

module.exports = { main };
