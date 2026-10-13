// @ts-check
// @safe-outputs-exempt SEC-005: repository policy is enforced by the shared artifact resolver.
"use strict";

const fs = require("fs");
const path = require("path");
const crypto = require("crypto");
const { execFileSync } = require("child_process");
const { AsyncLocalStorage } = require("async_hooks");
const { resolveTransportPaths, artifactError, scanArtifactRoot } = require("./resolve_transport_paths.cjs");
const { resolveTargetRepoConfig } = require("./repo_helpers.cjs");
const { checkFileProtectionPostApply } = require("./manifest_file_helpers.cjs");
const { currentClaimHandle, claimIdentity, normalizeRuntimeMessage, readClaimScopeContext, withClaimExecution } = require("./work_queue_claim_scope.cjs");
const { parseAllowedBaseBranches, isBaseBranchAllowed } = require("./create_pull_request_helpers.cjs");

const publication = new AsyncLocalStorage();
const TYPES = new Set(["create_pull_request", "push_to_pull_request_branch"]);
const protectedExecutors = new WeakSet();
const hash = bytes => crypto.createHash("sha256").update(bytes).digest("hex");
const json = value => JSON.stringify(value);

function parseArtifactJSON(bytes, label) {
  try {
    return JSON.parse(new TextDecoder("utf-8", { fatal: true, ignoreBOM: true }).decode(bytes));
  } catch {
    throw artifactError(`${label} is not valid UTF-8 JSON`);
  }
}

function credentialFreeEnvironment(environment) {
  const allowed = new Set([
    "PATH",
    "SYSTEMROOT",
    "WINDIR",
    "COMSPEC",
    "PATHEXT",
    "HOME",
    "USERPROFILE",
    "LANG",
    "LC_ALL",
    "LC_CTYPE",
    "TZ",
    "GIT_AUTHOR_NAME",
    "GIT_AUTHOR_EMAIL",
    "GIT_AUTHOR_DATE",
    "GIT_COMMITTER_NAME",
    "GIT_COMMITTER_EMAIL",
    "GIT_COMMITTER_DATE",
  ]);
  return Object.fromEntries(Object.entries(environment).filter(([key]) => allowed.has(key.toUpperCase())));
}

function cleanPolicy(config) {
  const strip = value => {
    if (Array.isArray(value)) return value.map(strip);
    if (value && typeof value === "object")
      return Object.fromEntries(
        Object.keys(value)
          .sort()
          .filter(key => !key.startsWith("_") && !["github-token", "head-github-token"].includes(key))
          .map(key => [key, strip(value[key])])
      );
    return value;
  };
  return strip(config);
}

function hasContentFileReference(value, depth = 0) {
  if (depth > 64) throw artifactError("output structure exceeds depth limit");
  if (!value || typeof value !== "object") return false;
  if (Object.prototype.hasOwnProperty.call(value, "filePath")) return true;
  return Object.values(value).some(child => hasContentFileReference(child, depth + 1));
}

function policyBytes(configPath) {
  const raw = process.env.GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG || (configPath ? readRegular(configPath).toString("utf8") : "");
  if (!raw) throw artifactError("trusted handler policy is missing");
  return Buffer.from(json(cleanPolicy(parseArtifactJSON(Buffer.from(raw), "trusted policy"))));
}

function readRegular(filename) {
  const resolved = path.resolve(filename);
  let descriptor;
  try {
    const before = fs.lstatSync(resolved);
    if (!before.isFile() || before.nlink !== 1 || before.size > 128 * 1024 * 1024 || fs.realpathSync(resolved) !== resolved) throw artifactError("unsafe imported file");
    descriptor = fs.openSync(resolved, fs.constants.O_RDONLY | (fs.constants.O_NOFOLLOW || 0));
    const opened = fs.fstatSync(descriptor);
    if (opened.dev !== before.dev || opened.ino !== before.ino || opened.size !== before.size) throw artifactError("artifact changed during import");
    const bytes = fs.readFileSync(descriptor);
    const after = fs.fstatSync(descriptor);
    if (after.size !== opened.size || after.mtimeMs !== opened.mtimeMs || bytes.length !== opened.size) throw artifactError("artifact changed during import");
    return bytes;
  } catch (error) {
    if (error.code === "CTR-033") throw error;
    throw artifactError("required artifact file is unavailable");
  } finally {
    if (descriptor !== undefined) fs.closeSync(descriptor);
  }
}

function binding(configBytes) {
  const values = ["GITHUB_REPOSITORY", "GITHUB_RUN_ID", "GITHUB_RUN_ATTEMPT", "GITHUB_WORKFLOW_REF", "GITHUB_SHA"].map(name => process.env[name] || "");
  if (values.some(value => !value)) throw artifactError("trusted workflow/run binding is missing");
  return { run: values, policy: hash(configBytes), policyRevision: process.env.GH_AW_ARTIFACT_POLICY_REF || process.env.GITHUB_WORKFLOW_SHA || values[4] };
}

function identity(message, repo) {
  return {
    type: message.type,
    repo,
    branch: message.branch || "",
    base: message.base || "",
    pull_number: message.pull_number ?? null,
    claim: currentClaimHandle() ? claimIdentity(currentClaimHandle()) : null,
  };
}

/** @param {any} [options] @returns {any} */
function optionsFromEnv(options = {}) {
  return {
    stagingRoot: process.env.GH_AW_ARTIFACT_STAGING_ROOT,
    approvedRoot: process.env.GH_AW_ARTIFACT_APPROVED_ROOT,
    preparedRoot: process.env.GH_AW_ARTIFACT_PREPARED_ROOT,
    outputPath: process.env.GH_AW_AGENT_OUTPUT,
    configPath: process.env.GH_AW_ARTIFACT_POLICY_FILE,
    baseRef: process.env.GH_AW_ARTIFACT_BASE_REF,
    ...options,
  };
}

/** Local-only Git execution. No hooks, external diff/textconv, lazy fetch, or auth. */
function localGit(cwd, args, trim = true) {
  const env = credentialFreeEnvironment(process.env);
  Object.assign(env, {
    GIT_TERMINAL_PROMPT: "0",
    GIT_NO_LAZY_FETCH: "1",
    GIT_CONFIG_NOSYSTEM: "1",
    GIT_CONFIG_GLOBAL: process.platform === "win32" ? "NUL" : "/dev/null",
    GIT_AUTHOR_NAME: "GitHub Agentic Workflows",
    GIT_AUTHOR_EMAIL: "noreply@github.com",
    GIT_COMMITTER_NAME: "GitHub Agentic Workflows",
    GIT_COMMITTER_EMAIL: "noreply@github.com",
  });
  try {
    const output = execFileSync(
      "git",
      ["-c", `safe.directory=${cwd}`, "-c", "core.hooksPath=" + __filename, "-c", "protocol.allow=never", "-c", "protocol.file.allow=always", "-c", "core.fsmonitor=false", "-c", "commit.gpgsign=false", ...args],
      {
        cwd,
        env,
        encoding: "utf8",
        maxBuffer: 128 * 1024 * 1024,
        stdio: ["ignore", "pipe", "pipe"],
      }
    );
    return trim ? output.trim() : output;
  } catch {
    throw artifactError("local artifact application or tree validation failed");
  }
}

/**
 * Keep local application credential-free and suppress repository-controlled
 * hooks even in the publisher. Remote operations retain only compiler-provided
 * credentials; they are not semantic validation.
 */
function protectArtifactGitExecution(execApi) {
  if (process.env.GH_AW_ARTIFACT_APPROVAL_REQUIRED !== "true" || protectedExecutors.has(execApi)) return;
  protectedExecutors.add(execApi);
  for (const method of ["exec", "getExecOutput"]) {
    const original = execApi[method].bind(execApi);
    execApi[method] = (command, args = [], options = {}) => {
      if (process.env.GH_AW_ARTIFACT_APPROVAL_REQUIRED !== "true") return original(command, args, options);
      if (command !== "git") return original(command, args, options);
      const cwd = options.cwd || process.env.GITHUB_WORKSPACE || process.cwd();
      const local = !["push", "pull", "clone", "ls-remote"].includes(args[0]) && !(args[0] === "fetch" && !path.isAbsolute(args.find(arg => path.isAbsolute(arg)) || ""));
      const guard = ["-c", `safe.directory=${cwd}`, "-c", `core.hooksPath=${__filename}`, "-c", "core.fsmonitor=false"];
      let env = { ...process.env, ...(options.env || {}) };
      if (local) {
        env = credentialFreeEnvironment(env);
        Object.assign(env, { GIT_NO_LAZY_FETCH: "1", GIT_TERMINAL_PROMPT: "0", GIT_CONFIG_NOSYSTEM: "1", GIT_CONFIG_GLOBAL: process.platform === "win32" ? "NUL" : "/dev/null" });
        guard.push("-c", "credential.helper=", "-c", "http.extraheader=");
      }
      return original(command, [...guard, ...args], { ...options, env });
    };
  }
}

function applyAndValidate(entry, config, checkout, stateRoot, baseRef, contextApi = global.context) {
  if (!entry.identity.branch) throw artifactError("artifact branch identity is missing");
  localGit(checkout, ["check-ref-format", "--branch", entry.identity.branch]);
  const directory = path.join(stateRoot, "trees", entry.id);
  fs.mkdirSync(path.dirname(directory), { recursive: true });
  const trustedPRHead = process.env.GH_AW_PR_HEAD_BASE_SHA || contextApi?.payload?.pull_request?.head?.sha;
  const authorizedBase = baseRef || (entry.identity.type === "push_to_pull_request_branch" ? trustedPRHead : config.base_branch || "HEAD");
  if (!authorizedBase || String(authorizedBase).startsWith("-")) throw artifactError("authorized base is missing");
  const authorizedBaseOid = localGit(checkout, ["rev-parse", "--verify", "--end-of-options", `${authorizedBase}^{commit}`]);
  const base = entry.identity.base ? localGit(checkout, ["rev-parse", "--verify", "--end-of-options", `${entry.identity.base}^{commit}`]) : authorizedBaseOid;
  if (base !== authorizedBaseOid && !isBaseBranchAllowed(entry.identity.base, parseAllowedBaseBranches(config.allowed_base_branches))) throw artifactError("base override is not authorized");
  if (!/^[0-9a-f]{40,64}$/.test(base)) throw artifactError("authorized base cannot be resolved");
  localGit(checkout, ["worktree", "add", "--detach", directory, base]);
  const patch = entry.artifacts.find(artifact => artifact.transport === "patch");
  const bundle = entry.artifacts.find(artifact => artifact.transport === "bundle");
  if (bundle) {
    const bundlePath = path.join(stateRoot, bundle.path);
    const heads = localGit(directory, ["bundle", "list-heads", bundlePath])
      .split("\n")
      .filter(Boolean)
      .map(line => line.split(/\s+/));
    const requested = heads.filter(([, ref]) => ref === `refs/heads/${entry.identity.branch}`);
    const selected = requested.length === 1 ? requested[0] : heads.length === 1 ? heads[0] : null;
    if (!selected || !/^[0-9a-f]{40,64}$/.test(selected[0])) throw artifactError("ambiguous bundle head");
    if (selected[1] !== "HEAD") {
      if (!selected[1]?.startsWith("refs/heads/")) throw artifactError("unsupported bundle head identity");
      localGit(directory, ["check-ref-format", "--branch", selected[1].slice("refs/heads/".length)]);
    }
    localGit(directory, ["fetch", "--no-tags", bundlePath, selected[1]]);
    localGit(directory, ["checkout", "--detach", selected[0]]);
    if (localGit(directory, ["merge-base", base, "HEAD"]) !== base) throw artifactError("bundle does not extend the authorized base");
    entry.bundleHead = selected[0];
  } else if (patch && readRegular(path.join(stateRoot, patch.path)).toString("utf8").trim()) {
    localGit(directory, ["am", "--3way", path.join(stateRoot, patch.path)]);
  } else if (config.allow_empty === true) {
    // Empty commits preserve the approved base tree.
  } else {
    // Existing-branch publication needs an explicitly approved local branch.
    localGit(directory, ["checkout", "--detach", entry.identity.branch]);
  }
  const files = localGit(directory, ["diff", "--name-only", "-z", "--no-renames", base, "HEAD"], false).split("\0").filter(Boolean);
  if (files.some(filename => filename.includes("\ufffd"))) throw artifactError("resulting tree requires unambiguous UTF-8 file paths");
  const protection = checkFileProtectionPostApply(files, config);
  if (protection.action === "deny") throw artifactError("resulting tree violates file policy");
  if (files.length > (config.max_patch_files || 100)) throw artifactError("resulting tree exceeds file count policy");
  if (config.signed_commits !== false && files.length > 0) {
    const modes = localGit(directory, ["ls-tree", "-rz", "HEAD", "--", ...files], false)
      .split("\0")
      .filter(Boolean);
    if (modes.some(record => !record.startsWith("100644 "))) throw artifactError("signed publication cannot preserve the resulting file modes");
  }
  const diff = localGit(directory, ["diff", "--binary", "--no-ext-diff", "--no-textconv", base, "HEAD"], false);
  if (Buffer.byteLength(diff) > (config.max_patch_size || 4096) * 1024) throw artifactError("resulting tree exceeds patch size policy");
  entry.base = base;
  entry.tree = localGit(directory, ["rev-parse", "HEAD^{tree}"]);
  entry.validationCheckout = directory;
}

/**
 * Import once, then expose the snapshot and local resulting trees to validators.
 * Compiler must run this in a trusted read-only job, not in the agent workspace.
 * @param {any} [supplied]
 */
async function prepare(supplied = {}) {
  const options = optionsFromEnv(supplied);
  const coreApi = options.core || global.core;
  const contextApi = options.context || global.context;
  if (!options.stagingRoot || !options.approvedRoot) throw artifactError("trusted import paths are missing");
  const stateRoot = path.resolve(options.preparedRoot || options.approvedRoot + ".prepared");
  if (fs.existsSync(stateRoot) || fs.existsSync(options.approvedRoot)) throw artifactError("import destination must be new");
  const stagingRoot = path.resolve(options.stagingRoot);
  const approvedRoot = path.resolve(options.approvedRoot);
  if (stateRoot === stagingRoot || stateRoot.startsWith(stagingRoot + path.sep) || approvedRoot === stagingRoot || approvedRoot.startsWith(stagingRoot + path.sep)) throw artifactError("immutable import must be outside staging");
  if (stateRoot === approvedRoot || stateRoot.startsWith(approvedRoot + path.sep) || approvedRoot.startsWith(stateRoot + path.sep)) throw artifactError("prepared and approved storage must be separate");
  const extractedFiles = scanArtifactRoot(stagingRoot);
  if (options.discoverOutput) {
    const outputs = extractedFiles.filter(filename => path.basename(filename) === "agent_output.json");
    if (outputs.length !== 1) throw artifactError("downloaded agent output is missing or ambiguous");
    options.outputPath = outputs[0];
  }
  if (!options.outputPath) throw artifactError("trusted agent output path is missing");
  const outputBytes = readRegular(options.outputPath);
  const configBytes = policyBytes(options.configPath);
  const config = parseArtifactJSON(configBytes, "trusted policy");
  const output = parseArtifactJSON(outputBytes, "agent output");
  if (!output || !Array.isArray(output.items) || output.items.some(item => !item || typeof item !== "object" || Array.isArray(item) || typeof item.type !== "string")) throw artifactError("output items are missing or invalid");
  if (options.rejectContentReferences && hasContentFileReference(output.items)) throw artifactError("artifact-validation does not support unbound content file references");
  fs.mkdirSync(stateRoot, { recursive: true });
  fs.writeFileSync(path.join(stateRoot, "agent_output.json"), outputBytes, { flag: "wx" });
  fs.writeFileSync(path.join(stateRoot, "config.json"), configBytes, { flag: "wx" });
  /** @type {any} */
  const manifest = { version: 1, binding: binding(configBytes), outputHash: hash(outputBytes), entries: [], stateRoot, outputPath: options.outputPath, configPath: options.configPath };
  if (options.rejectClaims && (process.env.GH_AW_WORK_QUEUE_ENABLED === "true" || output.items.some(item => item.claim_handle !== undefined))) throw artifactError("artifact-validation does not support claim-scoped publication");
  const scope = readClaimScopeContext();
  for (const raw of output.items.filter(item => TYPES.has(item.type))) {
    const message = scope ? normalizeRuntimeMessage(raw) : raw;
    const collect = async () => {
      const handlerConfig = config[message.type] || config[message.type.replaceAll("_", "-")] || config.handlers?.[message.type] || {};
      const { defaultTargetRepo, allowedRepos } = resolveTargetRepoConfig(handlerConfig);
      const paths = resolveTransportPaths(message, defaultTargetRepo, { root: stagingRoot, allowedRepos, config: handlerConfig, approval: false });
      if (options.sameRepoOnly && (paths.repo?.toLowerCase() !== process.env.GITHUB_REPOSITORY?.toLowerCase() || (handlerConfig["head-repo"] && handlerConfig["head-repo"].toLowerCase() !== process.env.GITHUB_REPOSITORY?.toLowerCase())))
        throw artifactError("artifact-validation currently requires same-repository publication");
      const prHeadRepo = contextApi?.payload?.pull_request?.head?.repo?.full_name;
      if (options.sameRepoOnly && message.type === "push_to_pull_request_branch" && prHeadRepo && prHeadRepo.toLowerCase() !== process.env.GITHUB_REPOSITORY?.toLowerCase())
        throw artifactError("artifact-validation currently requires same-repository PR heads");
      if (options.sameRepoOnly && message.type === "push_to_pull_request_branch") {
        const pull = contextApi?.payload?.pull_request;
        const target = handlerConfig.target === undefined || handlerConfig.target === null || handlerConfig.target === "" ? "triggering" : String(handlerConfig.target);
        if (target !== "triggering") throw artifactError("artifact-validation currently supports only the triggering PR target");
        if (!Number.isSafeInteger(pull?.number) || pull.number <= 0 || !/^[0-9a-f]{40,64}$/.test(pull?.head?.sha || "") || prHeadRepo?.toLowerCase() !== process.env.GITHUB_REPOSITORY?.toLowerCase())
          throw artifactError("artifact-validation requires trusted triggering PR head context");
        for (const name of ["pull_request_number", "pullRequestNumber", "target", "pr_number", "pull_number", "item_number", "pr"]) {
          const requested = message[name];
          if (requested !== undefined && requested !== null && requested !== "" && ((typeof requested !== "string" && typeof requested !== "number") || String(requested).trim() !== String(pull.number)))
            throw artifactError("artifact-validation PR target differs from trusted context");
        }
        if ((process.env.GH_AW_PR_HEAD_BASE_SHA && process.env.GH_AW_PR_HEAD_BASE_SHA !== pull.head.sha) || (options.baseRef && options.baseRef !== pull.head.sha))
          throw artifactError("artifact-validation PR baseline differs from trusted head");
        const headCheckout = options.checkout || process.env.GITHUB_WORKSPACE || process.cwd();
        try {
          localGit(headCheckout, ["rev-parse", "--verify", "--end-of-options", `${pull.head.sha}^{commit}`]);
        } catch {
          throw artifactError("trusted PR head is unavailable locally; read-only validation checkout must contain it");
        }
        if (message.base && localGit(headCheckout, ["rev-parse", "--verify", "--end-of-options", `${message.base}^{commit}`]) !== pull.head.sha) throw artifactError("artifact-validation PR base override differs from trusted head");
      }
      const key = identity(message, paths.repo);
      const id = hash(json(key));
      if (manifest.entries.some(entry => entry.id === id)) throw artifactError("duplicate operation identity");
      /** @type {any} */
      const entry = {
        id,
        identity: key,
        policyHash: hash(json(cleanPolicy(handlerConfig))),
        policy: cleanPolicy(handlerConfig),
        artifacts: [],
        transport: paths.bundlePath ? "bundle" : paths.patchPath ? "patch" : "existing_branch",
      };
      for (const [transport, filename] of [
        ["patch", paths.patchPath],
        ["bundle", paths.bundlePath],
      ]) {
        if (!filename) continue;
        const bytes = readRegular(filename);
        const relative = path.join("artifacts", id, `payload.${transport}`);
        fs.mkdirSync(path.dirname(path.join(stateRoot, relative)), { recursive: true });
        fs.writeFileSync(path.join(stateRoot, relative), bytes, { flag: "wx", mode: 0o400 });
        entry.artifacts.push({ transport, path: relative, digest: hash(bytes), source: path.relative(stagingRoot, filename) });
      }
      if (entry.transport === "existing_branch" && !message.branch) throw artifactError("artifact or existing branch is required");
      if (entry.transport === "existing_branch" && handlerConfig.allow_empty !== true) throw artifactError("required publication artifact is missing");
      const checkout = (paths.repo ? handlerConfig.checkout_mapping?.[paths.repo] : null) || options.checkout || process.env.GITHUB_WORKSPACE || process.cwd();
      applyAndValidate(entry, handlerConfig, path.resolve(checkout), stateRoot, options.baseRef, contextApi);
      manifest.entries.push(entry);
    };
    if (scope) await withClaimExecution({ ...scope, claim_handle: message.claim_handle }, collect);
    else await collect();
  }
  const manifestPath = path.join(stateRoot, "manifest.json");
  fs.writeFileSync(manifestPath, json(manifest), { flag: "wx" });
  coreApi?.exportVariable?.("GH_AW_RESOLVED_ARTIFACT_MANIFEST", manifestPath);
  coreApi?.exportVariable?.("GH_AW_ARTIFACT_MANIFEST", manifestPath);
  coreApi?.exportVariable?.("GH_AW_AGENT_OUTPUT", path.join(stateRoot, "agent_output.json"));
  coreApi?.setOutput?.("manifest", manifestPath);
  coreApi?.setOutput?.("prepare-digest", hash(Buffer.from(json(manifest))));
  coreApi?.setOutput?.("prepare_digest", hash(Buffer.from(json(manifest))));
  return manifest;
}

/** Seal only after all compiler-configured validators succeeded. */
function finalize(supplied = {}) {
  const options = optionsFromEnv(supplied);
  const coreApi = options.core || global.core;
  const stateRoot = path.resolve(options.preparedRoot || options.approvedRoot + ".prepared");
  const stateBytes = readRegular(path.join(stateRoot, "manifest.json"));
  const expectedState = supplied.prepareDigest || process.env.GH_AW_ARTIFACT_PREPARE_DIGEST;
  if (!expectedState || hash(stateBytes) !== expectedState) throw artifactError("trusted preparation digest is missing or changed");
  const manifest = JSON.parse(stateBytes.toString("utf8"));
  if (json(manifest.binding) !== json(binding(policyBytes(manifest.configPath)))) throw artifactError("policy or run changed during validation");
  if (hash(readRegular(path.join(stateRoot, "config.json"))) !== manifest.binding.policy) throw artifactError("imported policy changed during validation");
  if (hash(readRegular(manifest.outputPath)) !== manifest.outputHash || hash(readRegular(path.join(stateRoot, "agent_output.json"))) !== manifest.outputHash) throw artifactError("output changed during validation");
  for (const entry of manifest.entries) {
    for (const artifact of entry.artifacts) if (hash(readRegular(path.join(stateRoot, artifact.path))) !== artifact.digest) throw artifactError("artifact changed during validation");
    if (localGit(entry.validationCheckout, ["rev-parse", "HEAD^{tree}"]) !== entry.tree || localGit(entry.validationCheckout, ["status", "--porcelain", "--untracked-files=no"])) throw artifactError("validated tree changed");
  }
  fs.mkdirSync(options.approvedRoot, { recursive: false });
  fs.copyFileSync(path.join(stateRoot, "agent_output.json"), path.join(options.approvedRoot, "agent_output.json"), fs.constants.COPYFILE_EXCL);
  for (const entry of manifest.entries) {
    for (const artifact of entry.artifacts) {
      const target = path.join(options.approvedRoot, artifact.path);
      fs.mkdirSync(path.dirname(target), { recursive: true });
      fs.copyFileSync(path.join(stateRoot, artifact.path), target, fs.constants.COPYFILE_EXCL);
      delete artifact.source;
    }
    delete entry.validationCheckout;
  }
  delete manifest.stateRoot;
  delete manifest.outputPath;
  delete manifest.configPath;
  const bytes = Buffer.from(json(manifest));
  fs.writeFileSync(path.join(options.approvedRoot, "manifest.json"), bytes, { flag: "wx" });
  coreApi?.setOutput?.("approval-digest", hash(bytes));
  coreApi?.setOutput?.("approval_digest", hash(bytes));
  return manifest;
}

function readApproval(options = {}) {
  const approvalRoot = process.env.GH_AW_ARTIFACT_APPROVAL_ROOT;
  const manifestPath = options.manifestPath || (approvalRoot ? path.join(approvalRoot, "manifest.json") : process.env.GH_AW_ARTIFACT_APPROVAL_MANIFEST);
  const expected = options.manifestDigest || process.env.GH_AW_ARTIFACT_APPROVAL_DIGEST;
  if (!manifestPath || !/^[0-9a-f]{64}$/.test(expected || "")) throw artifactError("trusted approval is missing");
  const bytes = readRegular(manifestPath);
  if (hash(bytes) !== expected) throw artifactError("approval digest mismatch");
  let manifest;
  try {
    manifest = JSON.parse(bytes.toString("utf8"));
  } catch {
    throw artifactError("approval is not valid JSON");
  }
  if (!Array.isArray(manifest.entries) || manifest.entries.some(entry => !entry || !entry.identity || !Array.isArray(entry.artifacts) || !/^[0-9a-f]{40,64}$/.test(entry.base || "") || !/^[0-9a-f]{40,64}$/.test(entry.tree || "")))
    throw artifactError("approval contract is incomplete");
  const configPath = options.configPath || process.env.GH_AW_ARTIFACT_POLICY_FILE;
  if (manifest.version !== 1 || json(manifest.binding) !== json(binding(policyBytes(configPath)))) throw artifactError("approval run or policy mismatch");
  const outputPath = options.outputPath || (approvalRoot ? path.join(approvalRoot, "agent_output.json") : process.env.GH_AW_AGENT_OUTPUT);
  if (!outputPath || hash(readRegular(outputPath)) !== manifest.outputHash) throw artifactError("approved output digest mismatch");
  const root = path.dirname(path.resolve(manifestPath));
  if (options.verifyArtifacts) {
    for (const entry of manifest.entries) {
      for (const artifact of entry.artifacts) {
        if (!artifact || !["patch", "bundle"].includes(artifact.transport)) throw artifactError("unsupported approved transport");
        if (hash(readRegular(confinedPath(root, artifact.path))) !== artifact.digest) throw artifactError("approved artifact digest mismatch");
      }
    }
  }
  return { manifest, root };
}

function confinedPath(root, relative) {
  if (typeof relative !== "string" || path.isAbsolute(relative) || /^[A-Za-z]:|^[/\\]/.test(relative)) throw artifactError("unsafe approval artifact path");
  const filename = path.resolve(root, relative);
  if (!filename.startsWith(root + path.sep)) throw artifactError("approval artifact escapes root");
  return filename;
}

function approvedTransport(message, defaultTargetRepo, options = {}) {
  const { manifest, root } = readApproval(options);
  const { resolveAndValidateRepo } = require("./repo_helpers.cjs");
  const repo = resolveAndValidateRepo(message, defaultTargetRepo, options.allowedRepos || new Set(), "artifact");
  if (!repo.success) throw artifactError("approved repository is not authorized");
  const key = identity(message, repo.repo);
  const matches = manifest.entries.filter(entry => json(entry.identity) === json(key));
  if (matches.length !== 1) throw artifactError("approved operation or policy mismatch");
  const entry = matches[0];
  const suppliedPolicy = { ...(options.config || {}) };
  delete suppliedPolicy.allowedMentionAliases;
  if (!Object.prototype.hasOwnProperty.call(entry.policy, "mentions")) delete suppliedPolicy.mentions;
  if (entry.policyHash !== hash(json(cleanPolicy(suppliedPolicy)))) throw artifactError("approved handler policy mismatch");
  /** @type {{patchPath: string|undefined, bundlePath: string|undefined, repo: string}} */
  const result = { patchPath: undefined, bundlePath: undefined, repo: repo.repo };
  for (const artifact of entry.artifacts) {
    const filename = confinedPath(root, artifact.path);
    if (hash(readRegular(filename)) !== artifact.digest) throw artifactError("approved artifact digest mismatch");
    if (artifact.transport === "patch") result.patchPath = filename;
    else if (artifact.transport === "bundle") result.bundlePath = filename;
    else throw artifactError("unsupported approved transport");
  }
  publication.enterWith({ entry, root, options });
  return result;
}

function assertApprovedTree({ cwd, baseRef, repo }) {
  if (process.env.GH_AW_ARTIFACT_APPROVAL_REQUIRED !== "true") return;
  const approval = publication.getStore();
  if (!approval || approval.entry.identity.repo !== repo) throw artifactError("publication has no approved operation");
  // Re-read every digest at the mutation boundary. Never retry path resolution.
  readApproval(approval.options);
  for (const artifact of approval.entry.artifacts) if (hash(readRegular(confinedPath(approval.root, artifact.path))) !== artifact.digest) throw artifactError("approved artifact replaced before publication");
  const base = localGit(cwd, ["rev-parse", `${baseRef}^{commit}`]);
  const tree = localGit(cwd, ["rev-parse", "HEAD^{tree}"]);
  if (base !== approval.entry.base || tree !== approval.entry.tree) throw artifactError("publication base or resulting tree differs from approval");
}

module.exports = { prepare, finalize, approvedTransport, assertApprovedTree, readApproval, readRegular, protectArtifactGitExecution };
