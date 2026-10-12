// @ts-check
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { spawnSync } from "node:child_process";
import { collect, collectTree, guestEnvironment, readELFInterpreter, stage as stageAssets } from "./nvx_assets.cjs";

describe.skipIf(process.platform === "linux" && !fs.existsSync("/usr/bin/patchelf"))("NVX compiler asset handoff (Linux requires patchelf)", () => {
  /** @type {string} */
  let temp;
  /** @type {string} */
  let workspace;
  /** @type {string} */
  let hostRoot;
  /** @type {string} */
  let state;
  /** @type {NodeJS.ProcessEnv} */
  let environment;

  /** @param {NodeJS.ProcessEnv} env @param {string} stateRoot */
  function stage(env, stateRoot) {
    stageAssets(env, stateRoot, path.join(workspace, ".gh-aw-nvx"));
  }

  beforeEach(() => {
    temp = fs.mkdtempSync(path.join(os.tmpdir(), "nvx-assets-"));
    workspace = path.join(temp, "workspace with spaces");
    hostRoot = path.join(temp, "runner-temp", "gh-aw");
    state = path.join(temp, "state");
    for (const dir of [workspace, path.join(hostRoot, "actions"), path.join(state, "aw-prompts")]) fs.mkdirSync(dir, { recursive: true });
    fs.copyFileSync(path.join(import.meta.dirname, "nvx_assets.cjs"), path.join(hostRoot, "actions", "nvx_assets.cjs"));
    fs.writeFileSync(path.join(state, "aw-prompts", "prompt.txt"), "fixture prompt");
    fs.writeFileSync(path.join(hostRoot, "provider-token"), "must not be exported");
    fs.writeFileSync(path.join(state, "gateway-secret"), "must not be exported");
    environment = { GITHUB_WORKSPACE: workspace, RUNNER_TEMP: path.dirname(hostRoot) };
  });

  afterEach(() => fs.rmSync(temp, { recursive: true, force: true }));

  it("exports only runtime assets and selected inputs", () => {
    stage(environment, state);
    const root = path.join(workspace, ".gh-aw-nvx");
    expect(fs.readFileSync(path.join(root, "state", "aw-prompts", "prompt.txt"), "utf8")).toBe("fixture prompt");
    expect(fs.existsSync(path.join(root, "gh-aw", "bin", "node"))).toBe(true);
    expect(fs.existsSync(path.join(root, "gh-aw", "provider-token"))).toBe(false);
    expect(fs.existsSync(path.join(root, "state", "gateway-secret"))).toBe(false);
  });

  it("distinguishes ELF executables from scripts and shared libraries", () => {
    const file = path.join(temp, "elf-fixture");
    fs.writeFileSync(file, "#!/bin/sh\nexit 0\n");
    expect(readELFInterpreter(file)).toBeUndefined();
    const elf = Buffer.alloc(256);
    Buffer.from([0x7f, 0x45, 0x4c, 0x46, 2, 1]).copy(elf);
    elf.writeBigUInt64LE(64n, 32);
    elf.writeUInt16LE(56, 54);
    elf.writeUInt16LE(1, 56);
    elf.writeUInt32LE(3, 64);
    elf.writeBigUInt64LE(128n, 72);
    const interpreter = "/lib64/ld-linux-x86-64.so.2\0";
    elf.writeBigUInt64LE(BigInt(Buffer.byteLength(interpreter)), 96);
    elf.write(interpreter, 128);
    fs.writeFileSync(file, elf);
    expect(readELFInterpreter(file)).toBe("/lib64/ld-linux-x86-64.so.2");
    elf.writeUInt32LE(1, 64);
    fs.writeFileSync(file, elf);
    expect(readELFInterpreter(file)).toBeUndefined();
  });

  it.skipIf(process.platform !== "linux")("exports native CLI loaders and keeps Node subprocesses on the portable entrypoint", () => {
    fs.mkdirSync(path.join(hostRoot, "bin"));
    fs.copyFileSync("/bin/echo", path.join(hostRoot, "bin", "copilot"));
    stage(environment, state);
    const runtime = path.join(workspace, ".gh-aw-nvx", "gh-aw");
    const copilot = path.join(runtime, "bin", "copilot");
    const interpreter = readELFInterpreter(copilot);
    expect(interpreter?.startsWith(`${runtime}/lib/`)).toBe(true);
    expect(fs.existsSync(interpreter || "")).toBe(true);
    const cliResult = spawnSync(copilot, ["literal $HOME; $(exit 1)"], { encoding: "utf8" });
    expect(cliResult.status, cliResult.stderr).toBe(0);
    expect(cliResult.stdout.trim()).toBe("literal $HOME; $(exit 1)");
    const node = path.join(runtime, "bin", "node");
    const result = spawnSync(node, ["-e", 'const cp = require("node:child_process"); const child = cp.spawnSync(process.execPath, ["-e", "process.exit(17)"]); if (child.error) throw child.error; process.exit(child.status);'], {
      encoding: "utf8",
    });
    expect(result.signal, result.stderr).toBeNull();
    expect(result.status, result.stderr).toBe(17);
  });

  it("rejects pre-existing checkout-controlled staging directories and symlinks", () => {
    const root = path.join(workspace, ".gh-aw-nvx");
    fs.symlinkSync(state, root);
    expect(() => stage(environment, state)).toThrow(/EEXIST/);
    expect(fs.readFileSync(path.join(state, "aw-prompts", "prompt.txt"), "utf8")).toBe("fixture prompt");
  });

  it("rejects asset symlinks escaping the exported package", () => {
    fs.symlinkSync("../provider-token", path.join(hostRoot, "actions", "escape"));
    expect(() => stage(environment, state)).toThrow(/symlink escapes/);
    collect(environment, state);
    expect(fs.existsSync(path.join(workspace, ".gh-aw-nvx"))).toBe(false);
  });

  it("preserves confined relative npm bin symlinks", () => {
    const cli = path.join(hostRoot, "engine-cli");
    fs.mkdirSync(path.join(cli, "bin"), { recursive: true });
    fs.mkdirSync(path.join(cli, "node_modules"), { recursive: true });
    fs.writeFileSync(path.join(cli, "node_modules", "cli.js"), "fixture");
    fs.symlinkSync("../node_modules/cli.js", path.join(cli, "bin", "fixture"));
    stage(environment, state);
    expect(fs.readFileSync(path.join(workspace, ".gh-aw-nvx", "gh-aw", "engine-cli", "bin", "fixture"), "utf8")).toBe("fixture");
  });

  it.each([undefined, "/opt/toolcache"])("runs the staged runtime without host runner assets (tool cache %s)", toolCache => {
    stage(environment, state);
    const root = path.join(workspace, ".gh-aw-nvx");
    const stateLink = path.join(temp, "guest-state");
    const bootstrap = path.join(root, "gh-aw", "actions", "nvx_assets.cjs");
    const node = path.join(root, "gh-aw", "bin", "node");
    fs.writeFileSync(
      path.join(root, "gh-aw", "actions", "fixture_harness.cjs"),
      `
const fs = require("node:fs");
if (!fs.readFileSync(process.env.GH_AW_PROMPT, "utf8").includes("fixture prompt")) process.exit(2);
fs.writeFileSync(process.env.GH_AW_OUTPUT, "guest output");
process.exit(17);
`
    );
    const harness = path.join(root, "gh-aw", "actions", "fixture_harness.cjs");
    const command = `"$GH_AW_NODE_BIN" ${JSON.stringify(harness)}`;
    const result = spawnSync(node, ["-e", `require(${JSON.stringify(bootstrap)}).guest(${JSON.stringify(command)}, ${JSON.stringify(root)}, ${JSON.stringify(stateLink)})`], {
      env: {
        PATH: "/usr/bin:/bin",
        HOME: path.join(temp, "guest-home"),
        GH_AW_PROMPT: path.join(stateLink, "aw-prompts", "prompt.txt"),
        GH_AW_OUTPUT: path.join(stateLink, "result.txt"),
        ...(toolCache ? { RUNNER_TOOL_CACHE: toolCache } : {}),
      },
      encoding: "utf8",
    });
    expect(result.error).toBeUndefined();
    expect(result.signal, result.stderr).toBeNull();
    expect(result.status, result.stderr).toBe(17);
    collect(environment, state);
    expect(fs.readFileSync(path.join(state, "result.txt"), "utf8")).toBe("guest output");
    expect(fs.existsSync(root)).toBe(false);
    expect(fs.existsSync(path.join(hostRoot, "nvx-assets.json"))).toBe(false);
  });

  it("rewrites path-valued environment without rewriting credentials or endpoint values", () => {
    const env = guestEnvironment({
      HOME: "/home/awf",
      PATH: "/usr/local/go/bin:/usr/bin",
      GH_AW_NVX_HOST_ROOT: "/runner/temp/gh-aw",
      GH_AW_NVX_HOST_WORKSPACE: "/runner/repo",
      GH_AW_NVX_HOST_HOME: "/home/runner",
      GH_AW_MCP_CONFIG: "/home/runner/.copilot/mcp-config.json",
      DRIVER_PATH: "/runner/repo/driver.py",
      GH_AW_NODE_BIN: "/opt/hostedtoolcache/node/bin/node",
      GH_AW_SAFE_OUTPUTS: "/runner/temp/gh-aw/safeoutputs/result.json",
      ENDPOINT: "https://example.com/runner/repo",
    });
    expect(env.RUNNER_TEMP).toBe("/workspace/.gh-aw-nvx");
    expect(env.GH_AW_MCP_CONFIG).toBe("/home/awf/.copilot/mcp-config.json");
    expect(env.DRIVER_PATH).toBe("/workspace/driver.py");
    expect(env.GH_AW_SAFE_OUTPUTS).toBe("/workspace/.gh-aw-nvx/gh-aw/safeoutputs/result.json");
    expect(env.GH_AW_NODE_BIN).toBe("/workspace/.gh-aw-nvx/gh-aw/bin/node");
    expect(env.PATH).toContain("/usr/local/go/bin");
    expect(env.ENDPOINT).toBe("https://example.com/runner/repo");
    expect(env.RUNNER_TOOL_CACHE).toBeUndefined();
    expect(env.GH_AW_NVX_HOST_ROOT).toBeUndefined();
  });

  it("refuses guest symlinks during output collection and still removes runtime assets", () => {
    stage(environment, state);
    const root = path.join(workspace, ".gh-aw-nvx");
    fs.symlinkSync(path.join(hostRoot, "provider-token"), path.join(root, "state", "escape"));
    expect(() => collect(environment, state)).toThrow(/regular file or directory/);
    expect(fs.existsSync(root)).toBe(false);
    expect(fs.readFileSync(path.join(hostRoot, "provider-token"), "utf8")).toBe("must not be exported");
  });

  it("refuses a symlink destination during collection", () => {
    const source = path.join(temp, "output");
    const destination = path.join(temp, "destination");
    fs.writeFileSync(source, "output");
    fs.symlinkSync(path.join(hostRoot, "provider-token"), destination);
    expect(() => collectTree(source, destination)).toThrow(/Unsafe NVX output destination/);
  });

  it("refuses dangling destination symlinks", () => {
    const source = path.join(temp, "output");
    const target = path.join(temp, "missing");
    const destination = path.join(temp, "destination");
    fs.writeFileSync(source, "output");
    fs.symlinkSync(target, destination);
    expect(() => collectTree(source, destination)).toThrow(/Unsafe NVX output destination/);
    expect(fs.existsSync(target)).toBe(false);
  });

  it.each(["state", "gh-aw"])("refuses replaced asset subdirectories (%s)", directory => {
    stage(environment, state);
    const root = path.join(workspace, ".gh-aw-nvx");
    const original = path.join(root, directory);
    fs.renameSync(original, `${original}-original`);
    fs.symlinkSync(hostRoot, original);
    expect(() => collect(environment, state)).toThrow(/must not be a symlink/);
    expect(fs.existsSync(root)).toBe(false);
    expect(fs.readFileSync(path.join(hostRoot, "provider-token"), "utf8")).toBe("must not be exported");
  });

  it("removes read-only MCP CLI directories after collection", () => {
    stage(environment, state);
    const root = path.join(workspace, ".gh-aw-nvx");
    const bin = path.join(root, "gh-aw", "mcp-cli", "bin");
    fs.mkdirSync(bin, { recursive: true });
    fs.writeFileSync(path.join(bin, "fixture"), "fixture");
    fs.chmodSync(bin, 0o555);
    collect(environment, state);
    expect(fs.existsSync(root)).toBe(false);
  });

  it("does not remove a replaced staging directory", () => {
    stage(environment, state);
    const root = path.join(workspace, ".gh-aw-nvx");
    fs.renameSync(root, `${root}-original`);
    fs.mkdirSync(root);
    fs.writeFileSync(path.join(root, "keep"), "checkout content");
    expect(() => collect(environment, state)).toThrow(/directory was replaced/);
    expect(fs.readFileSync(path.join(root, "keep"), "utf8")).toBe("checkout content");
  });
});
