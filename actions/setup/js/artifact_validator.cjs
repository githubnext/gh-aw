// @ts-check
"use strict";

const fs = require("fs");
const path = require("path");
const os = require("os");
const crypto = require("crypto");
const childProcess = require("child_process");
const { artifactError } = require("./resolve_transport_paths.cjs");
const { readRegular } = require("./verified_artifact.cjs");

const TIMEOUT = 300000;
const MAX_OUTPUT = 1024 * 1024;
const contains = (root, filename) => filename === root || filename.startsWith(root + path.sep);

function logOutput(core, label, text) {
  const encoded = JSON.stringify(String(text)).replaceAll("::", "\\u003a\\u003a").replaceAll("##[", "\\u0023\\u0023[");
  core?.info?.(`Artifact validator ${label}: ${encoded}`);
}

function mountDirectory(value) {
  if (typeof value !== "string" || !path.isAbsolute(value) || /[,\r\n\x00]/.test(value)) throw artifactError("validator mount path must be an unambiguous absolute directory");
  const resolved = path.resolve(value);
  try {
    if (resolved === path.parse(resolved).root || !fs.lstatSync(resolved).isDirectory() || fs.realpathSync(resolved) !== resolved) throw artifactError("validator mount directory is unsafe");
  } catch {
    throw artifactError("validator mount directory is unsafe or unavailable");
  }
  return resolved;
}

/** @param {string[]} args @param {NodeJS.ProcessEnv} environment @param {AbortSignal | undefined} [signal] @returns {Promise<{stdout: string, stderr: string}>} */
function docker(args, environment, signal) {
  return new Promise((resolve, reject) => {
    childProcess.execFile("docker", args, { env: environment, shell: false, timeout: args[0] === "start" ? TIMEOUT : 120000, killSignal: "SIGKILL", maxBuffer: MAX_OUTPUT, signal }, (error, stdout, stderr) => {
      if (error) reject(Object.assign(error, { stdout, stderr }));
      else resolve({ stdout, stderr });
    });
  });
}

/** Run a trusted policy command, never agent code on the host. */
/** @param {{core: any, run?: string, env?: Record<string, string>}} options */
async function runArtifactValidator({ core, run, env }) {
  if (env !== undefined && (!env || typeof env !== "object" || Array.isArray(env) || Object.keys(env).length > 128)) throw artifactError("validator env must be a bounded explicit string map");
  if (run === undefined) {
    const encoded = process.env.GH_AW_ARTIFACT_VALIDATOR_STEP;
    let step;
    try {
      if (!encoded || Buffer.byteLength(encoded) > 131072) throw artifactError("validator configuration is missing or oversized");
      step = JSON.parse(encoded);
    } catch {
      throw artifactError("trusted validator configuration is missing or invalid");
    }
    if (
      !step ||
      typeof step !== "object" ||
      Object.keys(step).some(key => !["run", "envKeys"].includes(key)) ||
      !Array.isArray(step.envKeys) ||
      step.envKeys.length > 128 ||
      step.envKeys.some(key => typeof key !== "string") ||
      new Set(step.envKeys).size !== step.envKeys.length
    )
      throw artifactError("trusted validator configuration has unsupported fields");
    run = step.run;
    const explicitEnv = env;
    env = Object.fromEntries(
      step.envKeys.map(key => {
        const value = explicitEnv === undefined ? process.env[key] : Object.prototype.hasOwnProperty.call(explicitEnv, key) ? explicitEnv[key] : undefined;
        if (value === undefined) throw artifactError("configured validator environment variable is missing");
        return [key, value];
      })
    );
  }
  if (env === undefined) env = {};
  if (os.platform() !== "linux") throw artifactError("artifact validators require a Linux Docker host");
  const { uid, gid } = os.userInfo();
  if (!Number.isSafeInteger(uid) || !Number.isSafeInteger(gid) || uid <= 0 || gid <= 0) throw artifactError("artifact validators require a non-root host UID and GID");
  if (typeof run !== "string" || !run.trim() || Buffer.byteLength(run) > 65536 || run.includes("\x00")) throw artifactError("validator run must be a bounded trusted shell command");
  const image = process.env.GH_AW_ARTIFACT_VALIDATOR_IMAGE;
  if (!image || !/^[a-zA-Z0-9][a-zA-Z0-9._/:-]*@sha256:[0-9a-f]{64}$/.test(image)) throw artifactError("validator image must be digest-pinned");
  const workspace = mountDirectory(process.env.GITHUB_WORKSPACE);
  const prepared = mountDirectory(process.env.GH_AW_ARTIFACT_PREPARED_ROOT);
  if (contains(workspace, prepared) || contains(prepared, workspace)) throw artifactError("validator mounts must be separate");
  for (const protectedPath of [
    __dirname,
    process.env.RUNNER_TEMP,
    os.tmpdir(),
    process.env.GH_AW_ARTIFACT_APPROVED_ROOT,
    process.env.GH_AW_ARTIFACT_APPROVAL_ROOT,
    process.env.GITHUB_ENV,
    process.env.GITHUB_OUTPUT,
    process.env.GITHUB_PATH,
    process.env.GITHUB_STATE,
    process.env.GITHUB_STEP_SUMMARY,
  ]) {
    if (protectedPath && [workspace, prepared].some(root => contains(root, path.resolve(protectedPath)))) throw artifactError("validator mount exposes trusted runtime or runner state");
  }
  const manifestPath = path.join(prepared, "manifest.json");
  const outputPath = path.join(prepared, "agent_output.json");
  if ((process.env.GH_AW_ARTIFACT_MANIFEST && path.resolve(process.env.GH_AW_ARTIFACT_MANIFEST) !== manifestPath) || (process.env.GH_AW_AGENT_OUTPUT && path.resolve(process.env.GH_AW_AGENT_OUTPUT) !== outputPath))
    throw artifactError("validator descriptor paths differ from the prepared package");
  const stateBytes = readRegular(manifestPath);
  let state;
  try {
    state = JSON.parse(stateBytes.toString("utf8"));
  } catch {
    throw artifactError("validator preparation manifest is invalid");
  }
  if (
    !state ||
    state.version !== 1 ||
    state.stateRoot !== prepared ||
    !Array.isArray(state.entries) ||
    state.entries.some(entry => typeof entry?.validationCheckout !== "string" || !contains(prepared, path.resolve(entry.validationCheckout)))
  )
    throw artifactError("validator resulting trees are outside the prepared package");
  if (crypto.createHash("sha256").update(readRegular(outputPath)).digest("hex") !== state.outputHash) throw artifactError("validator output differs from the prepared snapshot");
  if (!env || typeof env !== "object" || Array.isArray(env) || Object.keys(env).length > 128) throw artifactError("validator env must be a bounded explicit string map");
  /** @type {Record<string, string>} */
  const containerEnv = {
    HOME: "/tmp",
    TMPDIR: "/tmp",
    GITHUB_WORKSPACE: "/workspace",
    GH_AW_ARTIFACT_PREPARED_ROOT: prepared,
    GH_AW_ARTIFACT_MANIFEST: manifestPath,
    GH_AW_RESOLVED_ARTIFACT_MANIFEST: manifestPath,
    GH_AW_AGENT_OUTPUT: outputPath,
  };
  let envSize = 0;
  for (const [key, value] of Object.entries(env)) {
    if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(key) || /^(?:GH_|GITHUB_|ACTIONS_|RUNNER_|DOCKER_|LD_|DYLD_|NODE_)/i.test(key) || /^(?:HOME|PATH|TMPDIR|ENV|BASH_ENV|SHELLOPTS|BASHOPTS)$/i.test(key))
      throw artifactError("validator env cannot override reserved or credential variables");
    if (typeof value !== "string" || value.includes("\x00") || (envSize += Buffer.byteLength(key) + Buffer.byteLength(value)) > 65536) throw artifactError("validator env must contain bounded literal strings");
    containerEnv[key] = value;
  }
  const scratch = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-validator-"));
  if ([workspace, prepared].some(root => contains(root, scratch))) {
    fs.rmSync(scratch, { recursive: true, force: true });
    throw artifactError("validator private host state must be outside mounted inputs");
  }
  const name = "gh-aw-validator-" + crypto.randomUUID();
  const environment = { PATH: "/usr/bin:/bin", HOME: scratch, DOCKER_CONFIG: scratch };
  const args = [
    "create",
    "--name",
    name,
    "--network",
    "none",
    "--read-only",
    "--security-opt",
    "no-new-privileges",
    "--cap-drop",
    "ALL",
    "--user",
    `${uid}:${gid}`,
    "--pids-limit",
    "64",
    "--memory",
    "512m",
    "--memory-swap",
    "512m",
    "--cpus",
    "1",
    "--ulimit",
    "nofile=256:256",
    "--ulimit",
    "core=0:0",
    "--ipc",
    "none",
    "--cgroupns",
    "private",
    "--init",
    "--restart",
    "no",
    "--log-driver",
    "none",
    "--tmpfs",
    "/tmp:rw,noexec,nosuid,nodev,size=128m,mode=1777",
    "--mount",
    `type=bind,source=${workspace},target=/workspace,readonly,bind-recursive=disabled`,
    "--mount",
    `type=bind,source=${prepared},target=${prepared},readonly,bind-recursive=disabled`,
    "--workdir",
    "/workspace",
    "--entrypoint",
    "/bin/sh",
  ];
  for (const [key, value] of Object.entries(containerEnv)) args.push("--env", `${key}=${value}`);
  args.push(image, "-e", "-c", run);
  const controller = new AbortController();
  const cancel = () => controller.abort();
  process.once("SIGTERM", cancel);
  process.once("SIGINT", cancel);
  let failure;
  let created = false;
  try {
    await docker(args, environment, controller.signal);
    created = true;
    const inspected = await docker(["inspect", "--format", "{{json .Mounts}}", name], environment, controller.signal);
    let mounts;
    try {
      mounts = JSON.parse(inspected.stdout);
    } catch {
      throw artifactError("validator sandbox mount inspection failed");
    }
    const binds = Array.isArray(mounts) ? mounts.filter(mount => mount.Type === "bind") : [];
    if (
      !Array.isArray(mounts) ||
      mounts.some(mount => mount.Type !== "bind" && !(mount.Type === "tmpfs" && mount.Destination === "/tmp")) ||
      binds.length !== 2 ||
      !binds.some(mount => mount.Source === workspace && mount.Destination === "/workspace" && mount.RW === false) ||
      !binds.some(mount => mount.Source === prepared && mount.Destination === prepared && mount.RW === false)
    )
      throw artifactError("validator sandbox has unexpected or writable mounts");
    const result = await docker(["start", "--attach", name], environment, controller.signal);
    // Encode untrusted output so it cannot become a GitHub Actions workflow command.
    if (result.stdout) logOutput(core, "stdout", result.stdout);
    if (result.stderr) logOutput(core, "stderr", result.stderr);
  } catch (error) {
    if (created && error.stdout) logOutput(core, "stdout", error.stdout);
    if (created && error.stderr) logOutput(core, "stderr", error.stderr);
    const reason =
      error.code === "ENOENT"
        ? "Docker is unavailable"
        : error.name === "AbortError"
          ? "validator was cancelled"
          : error.killed
            ? "validator exceeded its time or output limit"
            : created
              ? "validator failed"
              : "validator sandbox startup failed";
    failure = error.code === "CTR-033" ? error : artifactError(reason);
  } finally {
    try {
      await docker(["rm", "--force", name], environment);
    } catch (error) {
      const absent = !created && (error.code === "ENOENT" || String(error.stderr).trim() === `Error response from daemon: No such container: ${name}`);
      if (!absent) failure = artifactError("validator sandbox cleanup failed" + (failure ? ` after ${failure.message}` : ""));
    } finally {
      process.removeListener("SIGTERM", cancel);
      process.removeListener("SIGINT", cancel);
      fs.rmSync(scratch, { recursive: true, force: true });
    }
  }
  if (failure) throw failure;
  core?.info?.("Artifact validator completed successfully.");
}

module.exports = { runArtifactValidator };
