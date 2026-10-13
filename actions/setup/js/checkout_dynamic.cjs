// @ts-check

const fs = require("fs");
const path = require("path");
const { execFileSync } = require("child_process");
const { isRepoAllowed } = require("./repo_helpers.cjs");
const { getSetupTimeoutMs } = require("./child_process_timeouts.cjs");

/**
 * @param {string} value
 * @returns {string[]}
 */
function parseRepositories(value) {
  const repos = value
    .split(/\r?\n/)
    .map(repo => repo.trim())
    .filter(Boolean);
  for (const repo of repos) {
    if (!/^[a-zA-Z0-9][a-zA-Z0-9-]*\/[a-zA-Z0-9_.-]+$/.test(repo) || [".", ".."].includes(repo.split("/")[1])) {
      throw new Error("Runtime checkout repositories must be newline-separated owner/repo slugs");
    }
  }
  return [...new Set(repos.map(repo => repo.toLowerCase()))];
}

/**
 * @param {string} value
 * @returns {Set<string>}
 */
function parseAllowRepos(value) {
  const patterns = value
    .split(/[\r\n,]+/)
    .map(pattern => pattern.trim().toLowerCase())
    .filter(Boolean);
  if (patterns.length === 0) {
    throw new Error("Runtime checkout requires a non-empty allow-repos list");
  }
  for (const pattern of patterns) {
    if (pattern !== "*" && !/^[a-zA-Z0-9*-]+\/[a-zA-Z0-9_.*-]+$/.test(pattern)) {
      throw new Error("checkout.allow-repos must contain owner/repo patterns, owner/*, or *");
    }
  }
  return new Set(patterns);
}

/**
 * Check every existing ancestor before writing or allowing actions/checkout to clean a directory.
 * @param {string} workspace
 * @param {string} relative
 * @param {boolean} [mustBeAbsent]
 * @returns {string}
 */
function workspacePath(workspace, relative, mustBeAbsent = false) {
  if (!relative || path.isAbsolute(relative) || relative.includes("\\") || relative.includes("${{") || relative.split("/").some(part => part === ".." || part === ".git") || /[\x00-\x1f\x7f]/.test(relative)) {
    throw new Error("Runtime checkout paths must stay within GITHUB_WORKSPACE");
  }
  let root;
  try {
    root = fs.realpathSync(workspace);
  } catch (error) {
    throw new Error("Unable to resolve the runtime checkout workspace", { cause: error });
  }
  const target = path.resolve(root, relative);
  if (target === root || !target.startsWith(root + path.sep)) {
    throw new Error("Runtime checkout paths must stay within GITHUB_WORKSPACE");
  }
  let current = root;
  for (const part of path.relative(root, target).split(path.sep)) {
    current = path.join(current, part);
    let stat;
    try {
      stat = fs.lstatSync(current);
    } catch (error) {
      if (error instanceof Error && "code" in error && error.code === "ENOENT") continue;
      throw error;
    }
    if (!stat.isDirectory() || stat.isSymbolicLink()) {
      throw new Error("Runtime checkout paths must not contain symlinks or non-directory ancestors");
    }
    if (mustBeAbsent && current === target) {
      throw new Error("Runtime checkout destination already exists; use a different path prefix");
    }
  }
  return target;
}

function readConfig() {
  const index = process.env.GH_AW_DYNAMIC_CHECKOUT_INDEX || "";
  if (!/^\d+$/.test(index)) throw new Error("Runtime checkout index must be a non-negative integer");
  const workspace = process.env.GITHUB_WORKSPACE;
  const runnerTemp = process.env.RUNNER_TEMP;
  if (!workspace || !runnerTemp) throw new Error("GITHUB_WORKSPACE and RUNNER_TEMP are required for runtime checkout");
  const prefix = process.env.GH_AW_DYNAMIC_PREFIX || "";
  workspacePath(workspace, prefix);
  if (prefix.split("/").includes(".gh-aw")) throw new Error("Runtime checkout prefix must not use the reserved .gh-aw directory");
  return { index, workspace, runnerTemp, prefix };
}

/**
 * @typedef {{ repository: string, path: string, default_branch: string }} DynamicCheckout
 * @param {{ core: { setSecret: (value: string) => void }, github: import('@octokit/core').Octokit, getOctokit: (token: string) => import('@octokit/core').Octokit }} runtime
 */
async function prepare({ core, github, getOctokit }) {
  const config = readConfig();
  const repos = parseRepositories(process.env.GH_AW_DYNAMIC_REPOSITORIES || "");
  const allowed = parseAllowRepos(process.env.GH_AW_DYNAMIC_ALLOW_REPOS || "");
  for (const repository of repos) {
    if (!isRepoAllowed(repository, allowed)) throw new Error(`Runtime checkout repository ${repository} is not permitted by allow-repos`);
  }
  const token = process.env.GH_AW_DYNAMIC_TOKEN || "";
  if (!token) throw new Error("Runtime checkout requires a GitHub token");
  core.setSecret(token);
  const client = getOctokit(token);
  const host = parseRepositories(process.env.GITHUB_REPOSITORY || "");
  if (host.length !== 1) throw new Error("GITHUB_REPOSITORY must identify the workflow repository");
  const [owner, repo] = host[0].split("/");
  let hostData;
  try {
    hostData = (await github.request("GET /repos/{owner}/{repo}", { owner, repo })).data;
  } catch {
    throw new Error("Unable to verify workflow repository visibility; runtime checkout refused");
  }
  if (typeof hostData.private !== "boolean" || !["public", "private", "internal"].includes(hostData.visibility)) {
    throw new Error("Unable to verify workflow repository visibility; runtime checkout refused");
  }
  const isPublic = hostData.visibility === "public" || hostData.private === false;
  const submodules = process.env.GH_AW_DYNAMIC_SUBMODULES || "false";
  if (isPublic && submodules !== "false") {
    throw new Error("Runtime checkout in public workflows cannot enable submodules because their visibility is not verified");
  }
  /** @type {DynamicCheckout[]} */
  const entries = [];
  for (const repository of repos) {
    const [targetOwner, targetRepo] = repository.split("/");
    let data;
    try {
      data = (await client.request("GET /repos/{owner}/{repo}", { owner: targetOwner, repo: targetRepo })).data;
    } catch {
      throw new Error(`Unable to verify runtime checkout repository ${repository}; checkout refused`);
    }
    const canonical = parseRepositories(String(data.full_name || ""));
    if (
      canonical.length !== 1 ||
      !isRepoAllowed(canonical[0], allowed) ||
      typeof data.private !== "boolean" ||
      !["public", "private", "internal"].includes(data.visibility) ||
      typeof data.default_branch !== "string" ||
      !data.default_branch
    ) {
      throw new Error(`Unable to verify allowlist and metadata for runtime checkout repository ${repository}`);
    }
    if (isPublic && (data.private || data.visibility !== "public")) {
      throw new Error(`Public workflows cannot check out non-public repository ${repository}`);
    }
    if (entries.some(entry => entry.repository === canonical[0])) continue;
    const checkoutPath = path.posix.join(config.prefix, canonical[0]);
    workspacePath(config.workspace, checkoutPath, true);
    entries.push({ repository: canonical[0], path: checkoutPath, default_branch: data.default_branch });
  }
  const action = process.env.GH_AW_DYNAMIC_CHECKOUT_ACTION || "";
  if (!/^actions\/checkout@[a-f0-9]{40}$/.test(action)) throw new Error("Runtime checkout requires a SHA-pinned actions/checkout");
  const actionDir = workspacePath(config.workspace, `.gh-aw/checkouts/${config.index}`, true);
  const steps = entries.map(entry => ({
    uses: action,
    with: {
      repository: entry.repository,
      path: entry.path,
      ref: "${{ env.GH_AW_DYNAMIC_REF }}",
      token: "${{ env.GH_AW_DYNAMIC_TOKEN }}",
      "fetch-depth": "${{ env.GH_AW_DYNAMIC_FETCH_DEPTH }}",
      "sparse-checkout": "${{ env.GH_AW_DYNAMIC_SPARSE }}",
      filter: process.env.GH_AW_DYNAMIC_SPARSE ? "blob:limit=1073741824" : "",
      submodules: "${{ env.GH_AW_DYNAMIC_SUBMODULES }}",
      lfs: "${{ env.GH_AW_DYNAMIC_LFS }}",
      "persist-credentials": "${{ env.GH_AW_DYNAMIC_PERSIST }}",
    },
  }));
  // A composite action must contain at least one step, even for an empty scheduled batch.
  try {
    fs.mkdirSync(actionDir, { recursive: true });
    fs.writeFileSync(
      path.join(actionDir, "action.yml"),
      JSON.stringify({
        name: "Runtime repository checkout",
        description: "Checkout prevalidated runtime repositories",
        runs: { using: "composite", steps: steps.length ? steps : [{ shell: "bash", run: "true" }] },
      })
    );
  } catch (error) {
    throw new Error("Unable to write the validated runtime checkout action", { cause: error });
  }
  const planDir = path.join(config.runnerTemp, "gh-aw", "dynamic-checkouts");
  try {
    fs.mkdirSync(planDir, { recursive: true });
    fs.writeFileSync(path.join(planDir, `${config.index}.json`), JSON.stringify(entries));
  } catch (error) {
    throw new Error("Unable to write the runtime checkout plan", { cause: error });
  }
  return entries;
}

/**
 * @param {{ runGit?: (args: string[], options: import('child_process').ExecFileSyncOptionsWithStringEncoding) => string }} [options]
 */
async function finish(options = {}) {
  const config = readConfig();
  /** @type {DynamicCheckout[]} */
  let entries;
  try {
    entries = JSON.parse(fs.readFileSync(path.join(config.runnerTemp, "gh-aw", "dynamic-checkouts", `${config.index}.json`), "utf8"));
  } catch (error) {
    throw new Error("Unable to read the validated runtime checkout plan", { cause: error });
  }
  if (!Array.isArray(entries)) throw new Error("Runtime checkout plan must contain an array");
  const count = Number(process.env.GH_AW_DYNAMIC_FETCH_COUNT || "0");
  if (!Number.isFinite(count) || !Number.isSafeInteger(count) || count < 0) throw new Error("Runtime checkout fetch count must be a non-negative integer");
  const refs = Array.from({ length: count }, (_, index) => {
    const ref = process.env[`GH_AW_DYNAMIC_FETCH_${index}`];
    if (!ref || !ref.trim()) throw new Error("Runtime checkout fetch refs must not be empty");
    return ref;
  });
  const token = process.env.GH_AW_DYNAMIC_TOKEN || "";
  let server;
  try {
    server = new URL(process.env.GITHUB_SERVER_URL || "https://github.com");
  } catch (error) {
    throw new Error("Runtime checkout requires a valid GitHub server URL", { cause: error });
  }
  if (server.protocol !== "https:" || server.username || server.password) throw new Error("Runtime checkout requires an HTTPS GitHub server");
  for (const entry of entries) {
    const dir = workspacePath(config.workspace, entry.path);
    const run = (/** @type {string[]} */ args, authenticated = false, allowMissingKey = false) => {
      const env = { ...process.env, GIT_TERMINAL_PROMPT: "0" };
      if (authenticated) {
        Object.assign(env, {
          GIT_CONFIG_COUNT: "2",
          GIT_CONFIG_KEY_0: `http.${server.origin}/.extraheader`,
          GIT_CONFIG_VALUE_0: "",
          GIT_CONFIG_KEY_1: `http.${server.origin}/.extraheader`,
          GIT_CONFIG_VALUE_1: `AUTHORIZATION: basic ${Buffer.from(`x-access-token:${token}`).toString("base64")}`,
        });
      }
      try {
        const command = ["-C", dir, ...args];
        /** @type {import('child_process').ExecFileSyncOptionsWithStringEncoding} */
        const execOptions = { encoding: "utf8", env, stdio: ["ignore", "pipe", "pipe"], timeout: getSetupTimeoutMs(authenticated ? "importGit" : "gitBranch") };
        return options.runGit ? options.runGit(command, execOptions) : execFileSync("git", command, execOptions);
      } catch (error) {
        if (allowMissingKey && error instanceof Error && "status" in error && error.status === 5) return "";
        throw new Error(`Unable to finalize runtime checkout of ${entry.repository}`);
      }
    };
    if (process.env.GH_AW_DYNAMIC_SPARSE) {
      run(["config", "--local", "--unset-all", "remote.origin.promisor"], false, true);
      run(["config", "--local", "--unset-all", "remote.origin.partialclonefilter"], false, true);
    }
    if (refs.length) {
      const depth = Number(process.env.GH_AW_DYNAMIC_FETCH_DEPTH || "1");
      if (!Number.isSafeInteger(depth) || depth < 0) throw new Error("Runtime checkout fetch-depth must be a non-negative integer");
      const refspecs = refs.map(ref => {
        if (ref === "*") return "+refs/heads/*:refs/remotes/origin/*";
        if (ref === "refs/pulls/open/*") return "+refs/pull/*/head:refs/remotes/origin/pull/*/head";
        return `+refs/heads/${ref}:refs/remotes/origin/${ref}`;
      });
      run(["fetch", "origin", ...(depth ? [`--depth=${depth}`] : []), ...refspecs], true);
    }
  }
  const actionDir = workspacePath(config.workspace, `.gh-aw/checkouts/${config.index}`);
  try {
    fs.unlinkSync(path.join(actionDir, "action.yml"));
    fs.rmdirSync(actionDir);
  } catch (error) {
    throw new Error("Unable to remove the temporary runtime checkout action", { cause: error });
  }
  return entries;
}

module.exports = { prepare, finish, parseRepositories, parseAllowRepos, workspacePath };
