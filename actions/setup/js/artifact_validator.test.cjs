import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { createRequire } from "module";
import fs from "fs";
import path from "path";
import os from "os";
import crypto from "crypto";
import childProcess from "child_process";

const require = createRequire(import.meta.url);
const { runArtifactValidator } = require("./artifact_validator.cjs");
const image = "ghcr.io/github/gh-aw-node:test@sha256:" + "a".repeat(64);
let root, workspace, prepared, core, execute;
const inspectedMounts = () => [
  { Type: "bind", Source: workspace, Destination: "/workspace", RW: false },
  { Type: "bind", Source: prepared, Destination: prepared, RW: false },
];
const dockerOutput = args => (args[0] === "inspect" ? JSON.stringify(inspectedMounts()) : "");
beforeEach(() => {
  root = fs.mkdtempSync(path.join(process.cwd(), ".artifact-validator-"));
  workspace = path.join(root, "workspace");
  prepared = path.join(root, "prepared");
  fs.mkdirSync(workspace);
  fs.mkdirSync(prepared);
  fs.mkdirSync(path.join(prepared, "trees"));
  const output = Buffer.from(JSON.stringify({ items: [] }));
  fs.writeFileSync(path.join(prepared, "agent_output.json"), output);
  fs.writeFileSync(
    path.join(prepared, "manifest.json"),
    JSON.stringify({ version: 1, stateRoot: prepared, entries: [{ validationCheckout: path.join(prepared, "trees") }], outputHash: crypto.createHash("sha256").update(output).digest("hex") })
  );
  for (const [key, value] of Object.entries({
    GITHUB_WORKSPACE: workspace,
    GH_AW_ARTIFACT_PREPARED_ROOT: prepared,
    GH_AW_ARTIFACT_VALIDATOR_IMAGE: image,
    GH_AW_ARTIFACT_VALIDATOR_STEP: "",
    GH_AW_AGENT_OUTPUT: path.join(prepared, "agent_output.json"),
    GH_AW_ARTIFACT_MANIFEST: path.join(prepared, "manifest.json"),
    GH_AW_ARTIFACT_APPROVED_ROOT: path.join(root, "approved"),
    GH_AW_ARTIFACT_APPROVAL_ROOT: "",
    GITHUB_ENV: path.join(root, "runner-env"),
    GITHUB_OUTPUT: path.join(root, "runner-output"),
    GITHUB_PATH: "",
    GITHUB_STATE: "",
    GITHUB_STEP_SUMMARY: "",
    RUNNER_TEMP: root,
    GITHUB_TOKEN: "publication-secret",
    ACTIONS_RUNTIME_TOKEN: "artifact-secret",
    DOCKER_HOST: "tcp://untrusted-host:1234",
    NODE_OPTIONS: "--require=untrusted",
    LD_PRELOAD: "untrusted-preload",
    DYLD_INSERT_LIBRARIES: "untrusted-preload",
    PATH: "untrusted-executable-directory",
  }))
    vi.stubEnv(key, value);
  vi.spyOn(os, "platform").mockReturnValue("linux");
  vi.spyOn(os, "userInfo").mockReturnValue({ uid: 1001, gid: 1001, username: "runner", homedir: root, shell: "/bin/sh" });
  vi.spyOn(os, "tmpdir").mockReturnValue(root);
  execute = vi.spyOn(childProcess, "execFile").mockImplementation((_command, args, _options, callback) => {
    callback(null, dockerOutput(args), "");
    return {};
  });
  core = { info: vi.fn() };
});
afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllEnvs();
  fs.rmSync(root, { recursive: true, force: true });
});

describe("isolated artifact validator", () => {
  it("runs shell policy only inside the pinned container with immutable mounts and bounded resources", async () => {
    const run = 'printf "%s" "$CUSTOM"; echo "$(touch /tmp/container-only)"';
    await runArtifactValidator({ core, run, env: { CUSTOM: "literal ; $HOME\n--privileged" } });
    expect(execute).toHaveBeenCalledTimes(4);
    const [command, args, options] = execute.mock.calls[0];
    expect(command).toBe("docker");
    expect(args.slice(-4)).toEqual([image, "-e", "-c", run]);
    const pair = (flag, value) => expect(args.slice(args.indexOf(flag), args.indexOf(flag) + 2)).toEqual([flag, value]);
    pair("--network", "none");
    pair("--security-opt", "no-new-privileges");
    pair("--cap-drop", "ALL");
    pair("--user", "1001:1001");
    pair("--memory", "512m");
    pair("--memory-swap", "512m");
    pair("--cpus", "1");
    pair("--pids-limit", "64");
    pair("--workdir", "/workspace");
    pair("--entrypoint", "/bin/sh");
    pair("--restart", "no");
    pair("--log-driver", "none");
    expect(args).toContain("--read-only");
    expect(args).toContain("--init");
    expect(args).toContain("/tmp:rw,noexec,nosuid,nodev,size=128m,mode=1777");
    const mounts = args.filter((_, index) => args[index - 1] === "--mount");
    expect(mounts).toEqual([`type=bind,source=${workspace},target=/workspace,readonly,bind-recursive=disabled`, `type=bind,source=${prepared},target=${prepared},readonly,bind-recursive=disabled`]);
    const environment = args.filter((_, index) => args[index - 1] === "--env");
    expect(environment).toContain("CUSTOM=literal ; $HOME\n--privileged");
    expect(environment).toContain(`GH_AW_AGENT_OUTPUT=${path.join(prepared, "agent_output.json")}`);
    expect(environment).toContain(`GH_AW_ARTIFACT_MANIFEST=${path.join(prepared, "manifest.json")}`);
    expect(environment.join("\n")).not.toMatch(/publication-secret|artifact-secret|DOCKER_HOST|NODE_OPTIONS|approved|docker\.sock/);
    expect(Object.keys(options.env).sort()).toEqual(["DOCKER_CONFIG", "HOME", "PATH"]);
    expect(options.env.PATH).toBe("/usr/bin:/bin");
    expect(options.env).not.toHaveProperty("LD_PRELOAD");
    expect(options.env).not.toHaveProperty("DYLD_INSERT_LIBRARIES");
    expect(args).not.toContain("--group-add");
    expect(options).toMatchObject({ shell: false, killSignal: "SIGKILL", maxBuffer: 1048576, timeout: 120000 });
    expect(execute.mock.calls[1][1]).toEqual(["inspect", "--format", "{{json .Mounts}}", args[2]]);
    expect(execute.mock.calls[2][1]).toEqual(["start", "--attach", args[2]]);
    expect(execute.mock.calls[2][2].timeout).toBe(300000);
    expect(execute.mock.calls[3][1]).toEqual(["rm", "--force", args[2]]);
    expect(fs.existsSync(options.env.DOCKER_CONFIG)).toBe(false);
  });
  it("accepts the compiler's trusted JSON command and passes only its explicitly named env values", async () => {
    vi.stubEnv("GH_AW_ARTIFACT_VALIDATOR_STEP", JSON.stringify({ run: 'printf "%s" "$CUSTOM"', envKeys: ["CUSTOM"] }));
    vi.stubEnv("CUSTOM", "literal value");
    vi.stubEnv("UNLISTED", "runner-only value");
    await runArtifactValidator({ core });
    const args = execute.mock.calls[0][1];
    expect(args.slice(-3)).toEqual(["-e", "-c", 'printf "%s" "$CUSTOM"']);
    expect(args).toContain("CUSTOM=literal value");
    expect(args.join("\n")).not.toContain("runner-only value");
  });
  it("keeps quoted event-derived env data separate from the static validator command", async () => {
    const run = 'printf "%s" "$CUSTOM"';
    const value = '"},"run":"touch /workspace/injected","envKeys":[]}\n$(exit 0)';
    vi.stubEnv("GH_AW_ARTIFACT_VALIDATOR_STEP", JSON.stringify({ run, envKeys: ["CUSTOM"] }));
    vi.stubEnv("CUSTOM", value);
    await runArtifactValidator({ core });
    const args = execute.mock.calls[0][1];
    expect(args.slice(-3)).toEqual(["-e", "-c", run]);
    expect(args).toContain(`CUSTOM=${value}`);
    expect(fs.existsSync(path.join(workspace, "injected"))).toBe(false);
  });
  it("allows an explicit env seam while still selecting only compiler-listed keys", async () => {
    vi.stubEnv("GH_AW_ARTIFACT_VALIDATOR_STEP", JSON.stringify({ run: "true", envKeys: ["CUSTOM"] }));
    vi.stubEnv("CUSTOM", "ambient value");
    await runArtifactValidator({ core, env: { CUSTOM: "explicit value", UNLISTED: "must not pass" } });
    const args = execute.mock.calls[0][1];
    expect(args).toContain("CUSTOM=explicit value");
    expect(args.join("\n")).not.toMatch(/ambient value|must not pass/);
  });
  it.each([
    ["null", null],
    ["array", []],
    ["missing values", {}],
  ])("rejects invalid or incomplete explicit env seams (%s)", async (_label, env) => {
    vi.stubEnv("GH_AW_ARTIFACT_VALIDATOR_STEP", JSON.stringify({ run: "true", envKeys: ["CUSTOM"] }));
    await expect(runArtifactValidator({ core, env })).rejects.toMatchObject({ code: "CTR-033" });
    expect(execute).not.toHaveBeenCalled();
  });
  it.each([
    "",
    "{",
    JSON.stringify({ run: "true", envKeys: ["MISSING_VALIDATOR_VAR"] }),
    JSON.stringify({ run: "true", envKeys: ["CUSTOM", "CUSTOM"] }),
    JSON.stringify({ run: "true", envKeys: [], uses: "untrusted/action" }),
    JSON.stringify({ run: "true", envKeys: ["GITHUB_TOKEN"] }),
  ])("denies invalid or unsafe compiler JSON before Docker (%s)", async encoded => {
    vi.stubEnv("GH_AW_ARTIFACT_VALIDATOR_STEP", encoded);
    await expect(runArtifactValidator({ core })).rejects.toMatchObject({ code: "CTR-033" });
    expect(execute).not.toHaveBeenCalled();
  });
  it.each([
    "GITHUB_TOKEN",
    "GH_TOKEN",
    "GH_AW_AGENT_OUTPUT",
    "ACTIONS_RUNTIME_TOKEN",
    "RUNNER_TEMP",
    "DOCKER_HOST",
    "HOME",
    "PATH",
    "NODE_OPTIONS",
    "NODE_PATH",
    "LD_PRELOAD",
    "DYLD_INSERT_LIBRARIES",
    "BASH_ENV",
    "ENV",
    "SHELLOPTS",
    "BASHOPTS",
    "bad-key",
    "A=B",
  ])("denies reserved explicit env %s before Docker", async name => {
    await expect(runArtifactValidator({ core, run: "true", env: { [name]: "value" } })).rejects.toMatchObject({ code: "CTR-033" });
    expect(execute).not.toHaveBeenCalled();
  });
  it.each(["", "--privileged", "ghcr.io/github/gh-aw-node:latest"])("requires a digest-pinned image (%s)", async value => {
    vi.stubEnv("GH_AW_ARTIFACT_VALIDATOR_IMAGE", value);
    await expect(runArtifactValidator({ core, run: "true" })).rejects.toThrow("validator image must be digest-pinned");
    expect(execute).not.toHaveBeenCalled();
  });
  it.each(["platform", "uid", "gid"])("requires a supported non-root Linux host (%s)", async invalid => {
    if (invalid === "platform") os.platform.mockReturnValue("win32");
    else os.userInfo.mockReturnValue({ uid: invalid === "uid" ? 0 : 1001, gid: invalid === "gid" ? 0 : 1001 });
    await expect(runArtifactValidator({ core, run: "true" })).rejects.toMatchObject({ code: "CTR-033" });
    expect(execute).not.toHaveBeenCalled();
  });
  it.each(["runtime", "runner temp", "approval", "runner file", "summary file", "tree", "output", "descriptor"])("rejects unsafe mount or prepared state (%s)", async invalid => {
    if (invalid === "runtime") vi.stubEnv("GITHUB_WORKSPACE", process.cwd());
    if (invalid === "approval") vi.stubEnv("GH_AW_ARTIFACT_APPROVED_ROOT", path.join(workspace, "approved"));
    if (invalid === "runner file") vi.stubEnv("GITHUB_OUTPUT", path.join(workspace, "commands"));
    if (invalid === "runner temp") vi.stubEnv("RUNNER_TEMP", path.join(workspace, "runner-temp"));
    if (invalid === "summary file") vi.stubEnv("GITHUB_STEP_SUMMARY", path.join(prepared, "summary"));
    if (invalid === "tree") {
      const filename = path.join(prepared, "manifest.json");
      const state = JSON.parse(fs.readFileSync(filename, "utf8"));
      state.entries[0].validationCheckout = workspace;
      fs.writeFileSync(filename, JSON.stringify(state));
    }
    if (invalid === "output") fs.appendFileSync(path.join(prepared, "agent_output.json"), " ");
    if (invalid === "descriptor") vi.stubEnv("GH_AW_ARTIFACT_MANIFEST", path.join(workspace, "manifest.json"));
    await expect(runArtifactValidator({ core, run: "true" })).rejects.toMatchObject({ code: "CTR-033" });
    expect(execute).not.toHaveBeenCalled();
  });
  it.each(["failure", "timeout", "output limit", "cancellation"])("force-removes its container on %s", async failure => {
    execute.mockImplementation((_command, args, _options, callback) => {
      if (args[0] === "start")
        callback(
          Object.assign(new Error("untrusted error"), {
            code: failure === "output limit" ? "ERR_CHILD_PROCESS_STDIO_MAXBUFFER" : 1,
            killed: failure === "timeout" || failure === "output limit",
            name: failure === "cancellation" ? "AbortError" : "Error",
          }),
          "",
          ""
        );
      else callback(null, dockerOutput(args), "");
      return {};
    });
    await expect(runArtifactValidator({ core, run: "false" })).rejects.toMatchObject({ code: "CTR-033" });
    expect(execute.mock.calls.at(-1)[1].slice(0, 2)).toEqual(["rm", "--force"]);
    expect(execute.mock.calls.some(([, args]) => args[0] === "start")).toBe(true);
    expect(fs.existsSync(execute.mock.calls[0][2].env.HOME)).toBe(false);
    expect(core.info).not.toHaveBeenCalledWith("Artifact validator completed successfully.");
  });
  it("fails closed if force-removal fails", async () => {
    execute.mockImplementation((_command, args, _options, callback) => {
      callback(args[0] === "rm" ? new Error("daemon unavailable") : null, dockerOutput(args), "");
      return {};
    });
    await expect(runArtifactValidator({ core, run: "true" })).rejects.toThrow("validator sandbox cleanup failed");
  });
  it("denies missing Docker with no host-shell fallback", async () => {
    execute.mockImplementation((_command, _args, _options, callback) => {
      callback(Object.assign(new Error("missing"), { code: "ENOENT" }), "", "");
      return {};
    });
    await expect(runArtifactValidator({ core, run: "true" })).rejects.toThrow("Docker is unavailable");
    expect(execute.mock.calls.every(([command]) => command === "docker")).toBe(true);
  });
  it("encodes container output rather than forwarding either workflow command syntax", async () => {
    execute.mockImplementation((_command, args, _options, callback) => {
      callback(null, args[0] === "start" ? "::error::forged\n::add-mask::value\n##[error]legacy" : dockerOutput(args), "");
      return {};
    });
    await runArtifactValidator({ core, run: "true" });
    expect(core.info.mock.calls[0][0]).not.toMatch(/::|##\[/);
    expect(core.info.mock.calls[0][0]).toContain("\\u003a\\u003aerror");
    expect(core.info.mock.calls[0][0]).toContain("\\u0023\\u0023[error]");
  });
  it.each(["writable", "volume", "socket"])("rejects unexpected effective Docker mounts before starting (%s)", async unsafe => {
    execute.mockImplementation((_command, args, _options, callback) => {
      const mounts = inspectedMounts();
      if (unsafe === "writable") mounts[0].RW = true;
      if (unsafe === "volume") mounts.push({ Type: "volume", Source: "anonymous", Destination: "/data", RW: true });
      if (unsafe === "socket") mounts.push({ Type: "bind", Source: "/var/run/docker.sock", Destination: "/var/run/docker.sock", RW: false });
      callback(null, args[0] === "inspect" ? JSON.stringify(mounts) : "", "");
      return {};
    });
    await expect(runArtifactValidator({ core, run: "true" })).rejects.toThrow("validator sandbox has unexpected or writable mounts");
    expect(execute.mock.calls.some(([, args]) => args[0] === "start")).toBe(false);
    expect(execute.mock.calls.at(-1)[1].slice(0, 2)).toEqual(["rm", "--force"]);
  });
  it("cleans up on cancellation with an un-aborted cleanup command and removes signal handlers", async () => {
    const prior = process.listenerCount("SIGTERM");
    execute.mockImplementation((_command, args, options, callback) => {
      if (args[0] === "start") {
        options.signal.addEventListener("abort", () => callback(Object.assign(new Error("cancelled"), { name: "AbortError" }), "", ""), { once: true });
        process.listeners("SIGTERM").at(-1)();
      } else callback(null, dockerOutput(args), "");
      return {};
    });
    await expect(runArtifactValidator({ core, run: "true" })).rejects.toThrow("validator was cancelled");
    expect(execute.mock.calls.at(-1)[2].signal).toBeUndefined();
    expect(process.listenerCount("SIGTERM")).toBe(prior);
  });
});
