import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { spawn, spawnSync } from "child_process";
import fs from "fs";
import http from "http";
import os from "os";
import path from "path";

const { ensureFullHistoryForBundle, importLocalBundle, offlineGitOptions } = require("./git_helpers.cjs");
const { resolveTrustedBaseTip, getBundleValidationBase, getBundlePreApplyFiles } = require("./push_to_pull_request_branch.cjs");
const { checkFileProtectionPostApply } = require("./manifest_file_helpers.cjs");

const gitEnv = () => {
  const env = { ...process.env, GIT_CONFIG_GLOBAL: "/dev/null", GIT_CONFIG_NOSYSTEM: "1", GIT_TERMINAL_PROMPT: "0" };
  for (const key of Object.keys(env)) {
    if (key.startsWith("GIT_CONFIG_") && key !== "GIT_CONFIG_GLOBAL" && key !== "GIT_CONFIG_NOSYSTEM") delete env[key];
  }
  return env;
};

function git(cwd, args, env = gitEnv()) {
  const result = spawnSync("git", args, { cwd, env, encoding: "utf8" });
  if (result.error) throw result.error;
  if (result.status !== 0) throw new Error(result.stderr);
  return result.stdout.trim();
}

function execApi(cwd, { emulateOldGit = false } = {}) {
  const calls = [];
  const api = {
    calls,
    async getExecOutput(command, args, options = {}) {
      calls.push(args);
      const env = { ...gitEnv(), ...options.env };
      // Exercise the transport guard without relying on GIT_NO_LAZY_FETCH support.
      if (emulateOldGit) delete env.GIT_NO_LAZY_FETCH;
      const result = await new Promise((resolve, reject) => {
        const child = spawn(command, args, { cwd: options.cwd || cwd, env });
        let stdout = "";
        let stderr = "";
        child.stdout.on("data", data => (stdout += data));
        child.stderr.on("data", data => (stderr += data));
        child.on("error", reject);
        child.on("close", exitCode => resolve({ exitCode, stdout, stderr }));
      });
      if (result.exitCode !== 0 && !options.ignoreReturnCode) throw new Error(result.stderr);
      return result;
    },
    async exec(command, args, options = {}) {
      return (await api.getExecOutput(command, args, options)).exitCode;
    },
  };
  return api;
}

function packInventory(repo) {
  const dir = path.join(repo, ".git/objects/pack");
  return fs
    .readdirSync(dir)
    .sort()
    .map(file => [file, fs.statSync(path.join(dir, file)).size]);
}

describe("offline bundle transport with real Git", () => {
  let root;
  let source;
  let originalCore;
  let server;

  beforeEach(() => {
    originalCore = global.core;
    global.core = { debug: vi.fn(), info: vi.fn(), warning: vi.fn(), error: vi.fn() };
    root = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-offline-bundle-"));
    source = path.join(root, "source");
    fs.mkdirSync(source);
    git(source, ["init", "-b", "main"]);
    git(source, ["config", "user.name", "Bundle Test"]);
    git(source, ["config", "user.email", "bundle@example.com"]);
    fs.writeFileSync(path.join(source, "package.json"), "{}\n");
    fs.writeFileSync(path.join(source, "shared.txt"), "original\n");
    git(source, ["add", "."]);
    git(source, ["commit", "-m", "initial"]);
  });

  afterEach(async () => {
    if (server) {
      await new Promise(resolve => server.close(resolve));
      server = undefined;
    }
    global.core = originalCore;
    fs.rmSync(root, { recursive: true, force: true });
  });

  function addCommit(file, content) {
    fs.writeFileSync(path.join(source, file), content);
    git(source, ["add", "."]);
    git(source, ["commit", "-m", file]);
    return git(source, ["rev-parse", "HEAD"]);
  }

  function mergeFixture() {
    git(source, ["checkout", "-b", "feature"]);
    const oldHead = addCommit("shared.txt", "PR version\n");
    git(source, ["checkout", "main"]);
    const trustedBase = addCommit("shared.txt", "upstream version\n");
    git(source, ["checkout", "feature"]);
    const merge = spawnSync("git", ["merge", "main"], { cwd: source, env: gitEnv(), encoding: "utf8" });
    expect(merge.status).toBe(1);
    const candidate = addCommit("shared.txt", "agent resolution\n");
    return { oldHead, trustedBase, candidate };
  }

  it.each([false, true])("does not lazily transfer blobs while inspecting a partial clone (old Git guard: %s)", async emulateOldGit => {
    const { oldHead, trustedBase, candidate } = mergeFixture();
    git(source, ["config", "uploadpack.allowFilter", "true"]);
    const partial = path.join(root, "partial");
    git(root, ["clone", "--no-checkout", "--filter=blob:none", `file://${source}`, partial]);
    expect(git(partial, ["config", "--get", "remote.origin.promisor"])).toBe("true");
    const missingBlob = git(source, ["rev-parse", `${oldHead}:shared.txt`]);
    const exec = execApi(partial, { emulateOldGit });
    const before = packInventory(partial);
    expect(await resolveTrustedBaseTip(exec, {}, "main")).toBe(trustedBase);
    expect(await getBundleValidationBase(exec, {}, oldHead, candidate, trustedBase)).toEqual({ ref: oldHead, conflictedFiles: [] });
    const absent = await exec.getExecOutput("git", ["cat-file", "-e", missingBlob], { ...offlineGitOptions(), ignoreReturnCode: true });
    expect(absent.exitCode).not.toBe(0);
    expect(packInventory(partial)).toEqual(before);
    expect(exec.calls.some(args => args[0] === "fetch")).toBe(false);
  });

  it.each([false, true])("fails without unshallowing when bundle prerequisites are absent (filter support: %s)", async allowFilter => {
    const prerequisite = git(source, ["rev-parse", "HEAD"]);
    const bundle = path.join(root, "incremental.bundle");
    addCommit("agent.txt", "agent\n");
    git(source, ["bundle", "create", bundle, `${prerequisite}..HEAD`]);
    addCommit("later.txt", "later\n");
    git(source, ["config", "uploadpack.allowFilter", String(allowFilter)]);
    const shallow = path.join(root, "shallow");
    git(root, ["clone", "--depth=1", "--filter=blob:none", "--no-checkout", `file://${source}`, shallow]);
    const exec = execApi(shallow);
    const before = packInventory(shallow);
    const shallowFile = fs.readFileSync(path.join(shallow, ".git/shallow"), "utf8");
    await expect(ensureFullHistoryForBundle(exec, {}, { bundleFilePath: bundle })).rejects.toThrow("allow-full-history-fetch: true");
    await expect(importLocalBundle(exec, bundle, "HEAD", "refs/bundles/test")).rejects.toThrow("allow-full-history-fetch: true");
    expect(exec.calls.some(args => args[0] === "fetch")).toBe(false);
    expect(packInventory(shallow)).toEqual(before);
    expect(fs.readFileSync(path.join(shallow, ".git/shallow"), "utf8")).toBe(shallowFile);
  });

  it("imports an incremental bundle into a shallow checkout with locally provisioned prerequisites", async () => {
    const prerequisite = git(source, ["rev-parse", "HEAD"]);
    const shallow = path.join(root, "shallow");
    git(root, ["clone", "--depth=1", `file://${source}`, shallow]);
    git(source, ["checkout", "-b", "feature"]);
    const candidate = addCommit("agent.txt", "agent\n");
    const bundle = path.join(root, "agent.bundle");
    git(source, ["bundle", "create", bundle, "feature", `^${prerequisite}`]);
    const exec = execApi(shallow);
    await ensureFullHistoryForBundle(exec, {}, { bundleFilePath: bundle });
    await importLocalBundle(exec, bundle, "refs/heads/feature", "refs/bundles/test");
    expect(git(shallow, ["rev-parse", "refs/bundles/test"])).toBe(candidate);
    expect(await getBundlePreApplyFiles(exec, {}, prerequisite, "refs/bundles/test")).toEqual(["agent.txt"]);
    expect(git(shallow, ["rev-parse", "--is-shallow-repository"])).toBe("true");
    expect(exec.calls.some(args => args[0] === "fetch")).toBe(false);
  });

  it("retains protected-file checks when shallow history prevents a trusted baseline", async () => {
    const { oldHead, trustedBase } = mergeFixture();
    addCommit("package.json", '{"agent":true}\n');
    const shallow = path.join(root, "shallow");
    git(root, ["clone", "--no-checkout", "--depth=1", `file://${source}`, shallow]);
    // Provision endpoint commits, but not their common history.
    git(shallow, ["fetch", "--depth=1", "origin", oldHead, trustedBase]);
    const exec = execApi(shallow);
    const before = packInventory(shallow);
    const baseline = await getBundleValidationBase(exec, {}, oldHead, "HEAD", trustedBase);
    expect(baseline).toEqual({ ref: oldHead, conflictedFiles: [] });
    const files = await getBundlePreApplyFiles(exec, {}, baseline.ref, "HEAD");
    expect(checkFileProtectionPostApply(files, { protected_files: ["package.json"] }).action).toBe("deny");
    expect(packInventory(shallow)).toEqual(before);
  });

  it("allows full history recovery only with an explicit opt-in", async () => {
    const prerequisite = git(source, ["rev-parse", "HEAD"]);
    const bundle = path.join(root, "agent.bundle");
    addCommit("agent.txt", "agent\n");
    git(source, ["bundle", "create", bundle, `${prerequisite}..HEAD`]);
    addCommit("later.txt", "later\n");
    const shallow = path.join(root, "shallow");
    git(root, ["clone", "--depth=1", `file://${source}`, shallow]);
    const exec = execApi(shallow);
    await ensureFullHistoryForBundle(exec, {}, { bundleFilePath: bundle, allowFullHistoryFetch: true });
    expect(exec.calls).toContainEqual(["fetch", "--unshallow", "origin"]);
    expect(git(shallow, ["rev-parse", "--is-shallow-repository"])).toBe("false");
    await importLocalBundle(exec, bundle, "HEAD", "refs/bundles/test");
  });

  it("uses only target-checkout credentials for metadata and falls back silently when unavailable", async () => {
    const tip = git(source, ["rev-parse", "HEAD"]);
    const requests = [];
    let authorized = true;
    const syntheticHeader = "Basic synthetic-target-checkout-credential";
    server = http.createServer((req, res) => {
      requests.push({ method: req.method, url: req.url, authorized: req.headers.authorization === syntheticHeader });
      if (!authorized || req.headers.authorization !== syntheticHeader) {
        res.writeHead(403);
        res.end("synthetic secret-bearing remote diagnostic");
      } else if (req.url.startsWith("/target.git/info/refs")) {
        res.writeHead(200, { "Content-Type": "text/plain" });
        res.end(`${tip}\trefs/heads/main\n`);
      } else if (req.url === "/target.git/HEAD") {
        res.end("ref: refs/heads/main\n");
      } else {
        res.writeHead(404);
        res.end();
      }
    });
    await new Promise(resolve => server.listen(0, "127.0.0.1", resolve));
    const origin = `http://127.0.0.1:${server.address().port}/target.git`;
    git(source, ["remote", "add", "origin", origin]);
    git(source, ["config", `http.${origin}.extraheader`, `Authorization: ${syntheticHeader}`]);
    const exec = execApi(source);
    const before = git(source, ["count-objects", "-v"]);
    expect(await resolveTrustedBaseTip(exec, {}, "main")).toBe(tip);
    expect(git(source, ["count-objects", "-v"])).toBe(before);
    expect(requests.length).toBeGreaterThan(0);
    expect(requests.every(req => req.authorized && req.method === "GET" && /^\/target\.git\/(info\/refs(\?.*)?|HEAD)$/.test(req.url))).toBe(true);
    authorized = false;
    expect(await resolveTrustedBaseTip(exec, {}, "main")).toBe("");
    expect(JSON.stringify(global.core.warning.mock.calls)).not.toContain("synthetic");
    expect(global.core.warning).toHaveBeenCalledWith("Unable to resolve trusted PR base metadata; using PR-head validation");
  });

  it.each([
    { exitCode: 0, stdout: "not-a-sha\trefs/heads/main\n" },
    { exitCode: 0, stdout: `${"a".repeat(40)}\trefs/heads/other\n` },
    { exitCode: 0, stdout: `${"a".repeat(40)}\trefs/heads/main\n${"b".repeat(40)}\trefs/heads/main\n` },
    { exitCode: 128, stdout: `${"a".repeat(40)}\trefs/heads/main\n` },
  ])("rejects untrusted or unsuccessful metadata results %#", async result => {
    const exec = execApi(source);
    const run = exec.getExecOutput;
    exec.getExecOutput = vi.fn((command, args, options) => (args[0] === "ls-remote" ? Promise.resolve({ ...result, stderr: "synthetic private diagnostic" }) : run(command, args, options)));
    expect(await resolveTrustedBaseTip(exec, {}, "main")).toBe("");
    expect(JSON.stringify(global.core.warning.mock.calls)).not.toContain("synthetic");
    expect(exec.getExecOutput).toHaveBeenCalledWith("git", ["ls-remote", "--refs", "--exit-code", "origin", "refs/heads/main"], expect.objectContaining({ silent: true }));
  });

  it("rejects invalid base refs before accessing origin", async () => {
    const exec = execApi(source);
    expect(await resolveTrustedBaseTip(exec, {}, "../invalid")).toBe("");
    expect(await resolveTrustedBaseTip(exec, {}, undefined)).toBe("");
    expect(exec.calls.some(args => args[0] === "ls-remote")).toBe(false);
  });

  it("reports opted-in recovery failure without exposing remote diagnostics", async () => {
    const prerequisite = git(source, ["rev-parse", "HEAD"]);
    addCommit("agent.txt", "agent\n");
    const bundle = path.join(root, "agent.bundle");
    git(source, ["bundle", "create", bundle, `${prerequisite}..HEAD`]);
    const empty = path.join(root, "empty");
    fs.mkdirSync(empty);
    git(empty, ["init"]);
    const exec = execApi(empty);
    const run = exec.getExecOutput;
    exec.getExecOutput = vi.fn((command, args, options) => (args[0] === "fetch" ? Promise.resolve({ exitCode: 128, stdout: "", stderr: "synthetic private diagnostic" }) : run(command, args, options)));
    await expect(ensureFullHistoryForBundle(exec, {}, { bundleFilePath: bundle, allowFullHistoryFetch: true })).rejects.toThrow("Provision the required objects during authenticated checkout with credentials for the target repository.");
    expect(exec.getExecOutput).toHaveBeenCalledWith("git", ["fetch", "origin", prerequisite], expect.objectContaining({ silent: true }));
    expect(JSON.stringify(global.core.warning.mock.calls)).not.toContain("synthetic");
  });
});
