// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const path = require("path");

const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Loader for the checkout manifest written by the compiler-emitted
 * "Build checkout manifest for safe-outputs handlers" step.
 *
 * Layout on disk (JSON object keyed by lowercase repo slug):
 *   {
 *     "owner/repo": { "repository": "owner/repo", "path": "github", "default_branch": "master" }
 *   }
 *
 * The MCP server runs in a credential-less container, so the manifest is the
 * authoritative source for resolving the on-disk checkout path and base branch
 * of cross-repo checkouts without any network access.
 *
 * The default location is $RUNNER_TEMP/gh-aw/safeoutputs/checkout-manifest.json.
 * It lives under safeoutputs/ specifically because that subdirectory is the only
 * part of $RUNNER_TEMP/gh-aw that is bind-mounted into the containerized
 * safe-outputs MCP server; a sibling file at $RUNNER_TEMP/gh-aw/ would be
 * invisible inside the container. Override with GH_AW_CHECKOUT_MANIFEST when
 * running outside of a GitHub Actions runner (tests, local dev).
 */

/** @type {any} */
let cached = null;

function resolveManifestPath() {
  const explicit = process.env.GH_AW_CHECKOUT_MANIFEST;
  if (explicit && explicit.trim() !== "") {
    return explicit.trim();
  }
  const runnerTemp = process.env.RUNNER_TEMP;
  if (!runnerTemp || runnerTemp.trim() === "") {
    return null;
  }
  return path.join(runnerTemp, "gh-aw", "safeoutputs", "checkout-manifest.json");
}

function loadManifest() {
  if (cached !== null) {
    return cached;
  }
  const manifestPath = resolveManifestPath();
  if (!manifestPath) {
    cached = {};
    return cached;
  }
  try {
    const raw = fs.readFileSync(manifestPath, "utf8");
    const parsed = JSON.parse(raw);
    cached = parsed && typeof parsed === "object" ? parsed : {};
  } catch (err) {
    if (err && err.code !== "ENOENT" && typeof core !== "undefined") {
      core.debug(`checkout_manifest: failed to read ${manifestPath}: ${getErrorMessage(err)}`);
    }
    cached = {};
  }
  return cached;
}

/**
 * Look up a checkout manifest entry by repo slug ("owner/repo", case-insensitive).
 * Returns null when no entry exists.
 *
 * @param {string | undefined | null} repoSlug
 * @returns {{ repository: string, path: string, default_branch: string } | null}
 */
function lookupCheckout(repoSlug) {
  if (!repoSlug || typeof repoSlug !== "string") {
    return null;
  }
  const key = repoSlug.trim().toLowerCase();
  if (!key) {
    return null;
  }
  const manifest = loadManifest();
  const entry = manifest[key];
  if (!entry || typeof entry !== "object") {
    return null;
  }
  const repository = typeof entry.repository === "string" ? entry.repository : repoSlug;
  const entryPath = typeof entry.path === "string" ? entry.path : "";
  const defaultBranch = typeof entry.default_branch === "string" ? entry.default_branch : "";
  return { repository, path: entryPath, default_branch: defaultBranch };
}

/**
 * Reset the cached manifest. Intended for tests.
 */
function _resetCache() {
  cached = null;
}

/**
 * Return all checkout manifest entries as a Map keyed by lowercase repo slug.
 * Each value has the shape { repository, path, default_branch }.
 *
 * @returns {Map<string, { repository: string, path: string, default_branch: string }>}
 */
function loadAllCheckouts() {
  const manifest = loadManifest();
  const map = new Map();
  for (const [key, entry] of Object.entries(manifest)) {
    if (!entry || typeof entry !== "object") continue;
    const slug = key.trim().toLowerCase();
    if (!slug) continue;
    const repository = typeof entry.repository === "string" ? entry.repository : slug;
    const entryPath = typeof entry.path === "string" ? entry.path : "";
    const defaultBranch = typeof entry.default_branch === "string" ? entry.default_branch : "";
    map.set(slug, { repository, path: entryPath, default_branch: defaultBranch });
  }
  return map;
}

module.exports = {
  lookupCheckout,
  loadAllCheckouts,
  _resetCache,
};
