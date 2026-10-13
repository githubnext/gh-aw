// @ts-check
// @safe-outputs-exempt SEC-005: shared local import boundary; repository authorization is supplied by callers.

const fs = require("fs");
const path = require("path");
const { getPatchPathForBranch, getPatchPathForBranchInRepo } = require("./git_patch_utils.cjs");
const { getBundlePathForBranch, getBundlePathForBranchInRepo } = require("./generate_git_bundle.cjs");
const { resolveAndValidateRepo } = require("./repo_helpers.cjs");
const { currentClaimHandle, claimArtifactPath } = require("./work_queue_claim_scope.cjs");

function artifactError(reason) {
  return Object.assign(new Error(`CTR-033: ${reason}`), { code: "CTR-033", reasonCode: "artifact_approval_denied" });
}

/**
 * Inspect the actual extracted layout, never following links. Extraction itself
 * must also reject archive entry collisions: overwritten entries cannot be
 * reconstructed from the filesystem after a downloader has extracted them.
 */
function scanArtifactRoot(root) {
  const resolvedRoot = path.resolve(root);
  if (!fs.lstatSync(resolvedRoot).isDirectory() || fs.realpathSync(resolvedRoot) !== resolvedRoot) throw artifactError("unsafe staging root");
  /** @type {string[]} */
  const files = [];
  const names = new Set();
  let entries = 0;
  const visit = (directory, depth) => {
    if (depth > 32) throw artifactError("extracted layout exceeds depth limit");
    for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
      if (++entries > 4096) throw artifactError("extracted layout exceeds entry limit");
      const filename = path.join(directory, entry.name);
      const relative = path.relative(resolvedRoot, filename);
      if (relative.startsWith("..") || path.isAbsolute(relative) || /[\\/:]/.test(entry.name) || entry.name === "." || entry.name === "..") throw artifactError("unsafe extracted entry");
      const collisionKey = relative.toLowerCase();
      if (names.has(collisionKey)) throw artifactError("colliding extracted entries");
      names.add(collisionKey);
      const stat = fs.lstatSync(filename);
      if (stat.isSymbolicLink() || fs.realpathSync(filename) !== filename) throw artifactError("linked extracted entry");
      if (stat.isDirectory()) visit(filename, depth + 1);
      else if (stat.isFile() && stat.nlink === 1 && stat.size <= 128 * 1024 * 1024) files.push(filename);
      else throw artifactError("artifact must be a bounded regular file");
    }
  };
  visit(resolvedRoot, 0);
  return files;
}

/**
 * Repository authorization precedes resolution. A bare alias and its qualified
 * spelling are the same identity, not ordered fallback candidates. Multiple
 * matching files (including aliases in nested layouts) always deny publication.
 *
 * @param {any} message
 * @param {string} [defaultTargetRepo]
 * @param {{root?: string, allowedRepos?: Set<string>, config?: any, approval?: boolean}} [options]
 * @returns {{patchPath: string|undefined, bundlePath: string|undefined, repo?: string}}
 */
function resolveTransportPaths(message, defaultTargetRepo, options = {}) {
  if (options.approval !== false && process.env.GH_AW_ARTIFACT_APPROVAL_REQUIRED === "true") {
    return require("./verified_artifact.cjs").approvedTransport(message, defaultTargetRepo, options);
  }
  if (!message.branch) return { patchPath: undefined, bundlePath: undefined };
  if (typeof message.branch !== "string" || message.branch.length > 4096 || (message.repo !== undefined && typeof message.repo !== "string")) throw artifactError("invalid artifact identity fields");
  if (Buffer.from(message.branch).toString("utf8") !== message.branch || (message.repo && Buffer.from(message.repo).toString("utf8") !== message.repo)) throw artifactError("invalid artifact identity encoding");
  let repo;
  if (defaultTargetRepo) {
    const result = resolveAndValidateRepo(message, defaultTargetRepo, options.allowedRepos || new Set(), "artifact");
    if (!result.success) throw artifactError("artifact repository is not authorized");
    repo = result.repo;
  } else if (message.repo) {
    if (!/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/.test(message.repo)) throw artifactError("artifact repository must be qualified");
    repo = message.repo;
  }
  const patchCandidates = [getPatchPathForBranch(message.branch)];
  const bundleCandidates = [getBundlePathForBranch(message.branch)];
  if (repo) {
    patchCandidates.push(getPatchPathForBranchInRepo(message.branch, repo));
    bundleCandidates.push(getBundlePathForBranchInRepo(message.branch, repo));
  }
  // Legacy bare aliases must not select another artifact when the producer used
  // the qualified repository. Count them only to detect conflicting identities.
  if (message.repo && message.repo !== repo) {
    patchCandidates.push(getPatchPathForBranchInRepo(message.branch, message.repo));
    bundleCandidates.push(getBundlePathForBranchInRepo(message.branch, message.repo));
  }
  const root = options.root || process.env.GH_AW_ARTIFACT_STAGING_ROOT;
  const handle = currentClaimHandle();
  let files = root ? scanArtifactRoot(root) : [...new Set([...patchCandidates, ...bundleCandidates])].filter(filename => fs.existsSync(filename));
  if (root && handle) {
    const claimDirectory = path.basename(claimArtifactPath("", handle));
    files = files.filter(filename => {
      const components = path.relative(path.resolve(root), filename).split(path.sep);
      return components.some((component, index) => component === "claims" && components[index + 1] === claimDirectory);
    });
  }
  const find = candidates => {
    const basenames = new Set(candidates.map(filename => path.basename(filename).toLowerCase()));
    const matches = files.filter(filename => basenames.has(path.basename(filename).toLowerCase()));
    if (matches.length > 1) throw artifactError("ambiguous artifact aliases or duplicate basenames");
    if (matches.length === 1) {
      const stat = fs.lstatSync(matches[0]);
      if (!stat.isFile() || stat.nlink !== 1 || stat.size > 128 * 1024 * 1024 || fs.realpathSync(matches[0]) !== path.resolve(matches[0])) throw artifactError("unsafe artifact file");
    }
    return matches[0];
  };
  const patchPath = find(patchCandidates);
  const bundlePath = find(bundleCandidates);
  // A bundle's patch is a required preview, not a second application transport.
  // When configured, the transport must match the sealed contract exactly.
  const format = options.config?.patch_format;
  if ((format === "patch" || format === "am") && bundlePath) throw artifactError("bundle conflicts with patch transport policy");
  if (format === "bundle" && (!bundlePath || !patchPath)) throw artifactError("bundle transport requires bundle and patch preview");
  if (patchPath && bundlePath && path.basename(patchPath, ".patch") !== path.basename(bundlePath, ".bundle")) throw artifactError("patch preview and bundle have conflicting identities");
  return { patchPath, bundlePath, repo };
}

module.exports = { resolveTransportPaths, scanArtifactRoot, artifactError };
