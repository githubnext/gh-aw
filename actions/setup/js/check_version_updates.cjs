// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Check compile-agentic version against the remote update configuration.
 *
 * This script:
 * 1. Reads the compiled version from GH_AW_COMPILED_VERSION env var.
 * 2. Skips the check if the version is not in vMAJOR.MINOR.PATCH official release format.
 * 3. Fetches .github/aw/releases.json from the gh-aw repository via raw.githubusercontent.com.
 *    - Uses withRetry to handle transient network failures.
 * 4. If the download fails or config is invalid JSON, the check is skipped (soft failure).
 * 5. Validates that the compiled version is not in the blocked list.
 * 6. Validates that the compiled version meets the minimum supported version.
 *
 * Fails the activation job when validation fails.
 */

const { withRetry, isTransientError } = require("./error_recovery.cjs");

const CONFIG_URL = "https://raw.githubusercontent.com/github/gh-aw/main/.github/aw/releases.json";

/**
 * Parse an official version string (must be in vMAJOR.MINOR.PATCH format).
 * Versions without a leading "v" are not treated as official releases and return null.
 * Versions with unknown syntax also return null.
 *
 * @param {string} version
 * @returns {number[]|null}
 */
function parseVersion(version) {
  if (!version.startsWith("v")) return null;
  const parts = version.slice(1).split(".");
  if (parts.length !== 3) return null;
  const nums = parts.map(Number);
  if (nums.some(isNaN)) return null;
  return nums;
}

/**
 * Compare two official version strings (both must be in vMAJOR.MINOR.PATCH format).
 * Returns a negative number if a < b, 0 if equal, positive if a > b.
 * Returns 0 (treat as equal/unknown) if either version cannot be parsed.
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
 * @property {string} [minRecommendedVersion]
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

  // Only check official releases in vMAJOR.MINOR.PATCH format; ignore unknown syntax
  if (!parseVersion(compiledVersion)) {
    core.info(`Skipping version update check: '${compiledVersion}' is not an official release version (expected vMAJOR.MINOR.PATCH format)`);
    return;
  }

  core.info(`Checking compile-agentic version: ${compiledVersion}`);
  core.info(`Fetching update configuration from: ${CONFIG_URL}`);

  /** @type {UpdateConfig} */
  let config;
  try {
    config = await withRetry(
      async () => {
        const res = await fetch(CONFIG_URL);
        if (!res.ok) {
          const err = new Error(`HTTP ${res.status} fetching ${CONFIG_URL}`);
          // @ts-ignore - Attach status so the retry predicate can inspect it
          err.status = res.status;
          throw err;
        }
        const parsed = JSON.parse(await res.text());
        // Guard: JSON.parse("null") returns null; treat non-object/null/array as empty config
        return parsed !== null && typeof parsed === "object" && !Array.isArray(parsed) ? parsed : {};
      },
      {
        shouldRetry: err =>
          isTransientError(err) ||
          // Retry on any HTTP 5xx response (server errors)
          (err !== null && typeof err === "object" && "status" in err && Number(err.status) >= 500),
      },
      "fetch update configuration"
    );
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    core.info(`Could not fetch update configuration (${message}). Skipping version check.`);
    return;
  }

  const blockedVersions = Array.isArray(config.blockedVersions) ? config.blockedVersions : [];
  const minimumVersion = typeof config.minimumVersion === "string" ? config.minimumVersion : "";
  const minRecommendedVersion = typeof config.minRecommendedVersion === "string" ? config.minRecommendedVersion : "";

  // Check blocked versions — only consider entries in vMAJOR.MINOR.PATCH format; ignore unknown syntax
  const isBlocked = blockedVersions.some(v => parseVersion(v) !== null && compareVersions(compiledVersion, v) === 0);
  if (isBlocked) {
    core.summary
      .addRaw("### ❌ Blocked compile-agentic version\n\n")
      .addRaw(`The compile-agentic version \`${compiledVersion}\` is **blocked** and cannot be used to run workflows.\n\n`)
      .addRaw("This version has been revoked, typically due to a security issue.\n\n")
      .addRaw("**Action required:** Update `gh-aw` to the latest version and recompile your workflow with `gh aw compile`.\n");
    await core.summary.write();
    core.setFailed(`Blocked compile-agentic version: ${compiledVersion} is in the blocked versions list. Update gh-aw to the latest version and recompile your workflow.`);
    return;
  }

  // Check minimum version — skip if minimumVersion is absent, empty, or has unknown syntax
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

  // Check recommended version — skip if minRecommendedVersion is absent, empty, or has unknown syntax
  if (minRecommendedVersion && parseVersion(minRecommendedVersion) !== null) {
    if (compareVersions(compiledVersion, minRecommendedVersion) < 0) {
      core.warning(
        `Recommended upgrade: compile-agentic version ${compiledVersion} is below the recommended version ${minRecommendedVersion}. Consider updating gh-aw to the latest version and recompiling your workflow with \`gh aw compile\`.`
      );
    }
  }

  core.info(`✅ Version check passed: ${compiledVersion}`);
}

module.exports = { main };
