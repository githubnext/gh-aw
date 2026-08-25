// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Check compile-agentic version against the remote update configuration.
 *
 * This script:
 * 1. Reads the compiled version from GH_AW_COMPILED_VERSION env var.
 * 2. Skips the check if the version is not in official SemVer release format.
 * 3. Fetches .github/aw/compat.json from the gh-aw repository via raw.githubusercontent.com.
 *    - Uses withRetry to handle transient network failures.
 * 4. If the download fails or config is invalid JSON, the check is skipped (soft failure).
 * 5. Validates that the compiled version is not in the blocked list.
 * 6. Validates that the compiled version meets the minimum supported version.
 *
 * Fails the activation job when validation fails.
 */

const { withRetry, isTransientError } = require("./error_recovery.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

const CONFIG_URL = "https://raw.githubusercontent.com/github/gh-aw/main/.github/aw/compat.json";
const FETCH_TIMEOUT_MS = 120_000;
const VERSION_PATTERN = /^v([0-9]+)\.([0-9]+)\.([0-9]+)(?:-((?:0|[1-9][0-9]*|[0-9A-Za-z-]*[A-Za-z-][0-9A-Za-z-]*)(?:\.(?:0|[1-9][0-9]*|[0-9A-Za-z-]*[A-Za-z-][0-9A-Za-z-]*))*))?$/;

/**
 * Parse an official version string (vMAJOR.MINOR.PATCH with an optional prerelease).
 * Versions without a leading "v" are not treated as official releases and return null.
 * Versions with unknown syntax also return null.
 *
 * @param {string} version
 * @returns {{base: string[], prerelease: string[]|null}|null}
 */
function parseVersion(version) {
  const match = VERSION_PATTERN.exec(version);
  if (!match) return null;
  return {
    base: [match[1], match[2], match[3]],
    prerelease: match[4] ? match[4].split(".") : null,
  };
}

/**
 * Compare numeric SemVer identifiers without converting to Number.
 * VERSION_PATTERN rejects leading zeroes, so length plus lexical order is numeric order.
 *
 * @param {string} left
 * @param {string} right
 * @returns {number}
 */
function compareNumericIdentifiers(left, right) {
  if (left.length !== right.length) return left.length - right.length;
  if (left !== right) return left < right ? -1 : 1;
  return 0;
}

/**
 * Compare two official SemVer version strings.
 * Returns a negative number if a < b, 0 if equal, positive if a > b.
 * Returns 0 (treat as equal/unknown) if either version cannot be parsed.
 *
 * @param {string} a
 * @param {string} b
 * @returns {number|null}
 */
function compareVersions(a, b) {
  const pa = parseVersion(a);
  const pb = parseVersion(b);
  if (!pa || !pb) return null;
  for (let i = 0; i < 3; i++) {
    const difference = compareNumericIdentifiers(pa.base[i], pb.base[i]);
    if (difference !== 0) return difference;
  }
  if (pa.prerelease === null && pb.prerelease === null) return 0;
  if (pa.prerelease === null) return 1;
  if (pb.prerelease === null) return -1;
  const length = Math.max(pa.prerelease.length, pb.prerelease.length);
  for (let i = 0; i < length; i++) {
    if (i >= pa.prerelease.length) return -1;
    if (i >= pb.prerelease.length) return 1;
    const left = pa.prerelease[i];
    const right = pb.prerelease[i];
    const leftIsNumeric = /^\d+$/.test(left);
    const rightIsNumeric = /^\d+$/.test(right);
    if (leftIsNumeric && rightIsNumeric) {
      const difference = compareNumericIdentifiers(left, right);
      if (difference !== 0) return difference;
    } else if (leftIsNumeric) {
      return -1;
    } else if (rightIsNumeric) {
      return 1;
    } else if (left !== right) {
      return left < right ? -1 : 1;
    }
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

  // Only check official SemVer releases; ignore unknown syntax.
  if (!parseVersion(compiledVersion)) {
    core.info(`Skipping version update check: '${compiledVersion}' is not an official release version (expected vMAJOR.MINOR.PATCH with an optional prerelease)`);
    return;
  }

  core.info(`Checking compile-agentic version: ${compiledVersion}`);
  core.info(`Fetching update configuration from: ${CONFIG_URL}`);

  /** @type {UpdateConfig} */
  let config;
  try {
    config = await withRetry(
      async () => {
        const res = await fetch(CONFIG_URL, { signal: AbortSignal.timeout(FETCH_TIMEOUT_MS) });
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
    const message = getErrorMessage(err);
    core.info(`Could not fetch update configuration (${message}). Skipping version check.`);
    return;
  }

  const blockedVersions = Array.isArray(config.blockedVersions) ? config.blockedVersions : [];
  const minimumVersion = typeof config.minimumVersion === "string" ? config.minimumVersion : "";
  const minRecommendedVersion = typeof config.minRecommendedVersion === "string" ? config.minRecommendedVersion : "";

  // Check blocked versions — only consider entries in official SemVer format; ignore unknown syntax.
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
    const comparison = compareVersions(compiledVersion, minimumVersion);
    if (comparison !== null && comparison < 0) {
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
    const comparison = compareVersions(compiledVersion, minRecommendedVersion);
    if (comparison !== null && comparison < 0) {
      core.warning(
        `Recommended upgrade: compile-agentic version ${compiledVersion} is below the recommended version ${minRecommendedVersion}. Consider updating gh-aw to the latest version and recompiling your workflow with \`gh aw compile\`.`
      );
    }
  }

  core.info(`✅ Version check passed: ${compiledVersion}`);
}

module.exports = { main };
