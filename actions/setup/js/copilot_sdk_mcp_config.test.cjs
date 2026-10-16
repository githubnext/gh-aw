import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createRequire } from "node:module";
import { createHash } from "node:crypto";
import { chmodSync, linkSync, lstatSync, mkdirSync, mkdtempSync, readFileSync, readdirSync, rmSync, statSync, symlinkSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";

const require = createRequire(import.meta.url);
const { COPILOT_SDK_MCP_CONFIG_SHA256, digestCopilotSDKMCPConfig, readCopilotSDKMCPConfig, writeCopilotSDKMCPConfig, publishCopilotSDKMCPConfig } = require("./copilot_sdk_mcp_config.cjs");
const { resolveCopilotSDKMCPConfigPath } = require("./copilot_sdk_mcp_config_path.cjs");

describe("Copilot SDK MCP catalog integrity", () => {
  let root;
  let configPath;
  const source = JSON.stringify({
    mcpServers: {
      safeoutputs: { type: "http", url: "http://gateway:80/mcp/safeoutputs", tools: ["noop"], headers: { Authorization: "test-only" }, timeout: 30000 },
    },
  });

  beforeEach(() => {
    root = mkdtempSync(join(process.cwd(), ".sdk-mcp-integrity-"));
    vi.stubEnv("RUNNER_TEMP", root);
    vi.stubEnv("GITHUB_RUN_ID", "123");
    vi.stubEnv("GITHUB_RUN_ATTEMPT", "1");
    vi.stubEnv("GITHUB_JOB", "agent");
    configPath = resolveCopilotSDKMCPConfigPath();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllEnvs();
    rmSync(root, { recursive: true, force: true });
  });

  function publish() {
    vi.stubEnv(COPILOT_SDK_MCP_CONFIG_SHA256, writeCopilotSDKMCPConfig(source));
  }

  it("atomically publishes a private catalog with a run-bound digest", () => {
    publish();
    expect(readCopilotSDKMCPConfig(configPath)).toBe(source);
    expect(statSync(configPath).mode & 0o777).toBe(0o600);
    expect(statSync(dirname(configPath)).mode & 0o777).toBe(0o700);
    expect(readdirSync(dirname(configPath))).toEqual(["copilot-sdk.json"]);
    expect(process.env[COPILOT_SDK_MCP_CONFIG_SHA256]).toBe(createHash("sha256").update('["123","1","agent"]\0').update(source).digest("hex"));
  });

  it.each([
    ["URL", value => (value.mcpServers.safeoutputs.url = "https://unexpected.example/mcp")],
    ["tools", value => value.mcpServers.safeoutputs.tools.push("create_issue")],
    ["headers", value => (value.mcpServers.safeoutputs.headers.Authorization = "changed-test-only")],
    ["timeout", value => (value.mcpServers.safeoutputs.timeout = 1)],
  ])("rejects modified %s before registration", (_name, mutate) => {
    publish();
    const changed = JSON.parse(source);
    mutate(changed);
    writeFileSync(configPath, JSON.stringify(changed));
    expect(() => readCopilotSDKMCPConfig(configPath)).toThrow("integrity check failed");
    expect(() => readCopilotSDKMCPConfig(configPath)).not.toThrow("test-only");
  });

  it.each(["GITHUB_RUN_ID", "GITHUB_RUN_ATTEMPT", "GITHUB_JOB"])("rejects catalog replay under a different %s", name => {
    publish();
    vi.stubEnv(name, `${process.env[name]}2`);
    expect(() => readCopilotSDKMCPConfig(configPath)).toThrow("integrity check failed");
  });

  it("refreshes integrity between attempts even when catalog contents are identical", () => {
    publish();
    const previous = process.env[COPILOT_SDK_MCP_CONFIG_SHA256];
    vi.stubEnv("GITHUB_RUN_ATTEMPT", "2");
    publish();
    expect(process.env[COPILOT_SDK_MCP_CONFIG_SHA256]).not.toBe(previous);
    expect(readCopilotSDKMCPConfig(configPath)).toBe(source);
  });

  it.each(["", "not-a-digest", "a".repeat(63), "a".repeat(65)])("requires a complete integrity value: %j", digest => {
    publish();
    vi.stubEnv(COPILOT_SDK_MCP_CONFIG_SHA256, digest);
    expect(() => readCopilotSDKMCPConfig(configPath)).toThrow("configuration integrity is required");
  });

  it.each(["GITHUB_RUN_ID", "GITHUB_RUN_ATTEMPT", "GITHUB_JOB"])("fails closed without %s", name => {
    publish();
    vi.stubEnv(name, "");
    expect(() => readCopilotSDKMCPConfig(configPath)).toThrow("required for Copilot SDK MCP configuration integrity");
    expect(() => writeCopilotSDKMCPConfig(source)).toThrow("required for Copilot SDK MCP configuration integrity");
  });

  it.each(["symlink", "hardlink"])("replaces a pre-existing %s without changing its target", kind => {
    mkdirSync(dirname(configPath), { recursive: true });
    const target = join(root, "unrelated.json");
    writeFileSync(target, "must not change");
    if (kind === "symlink") symlinkSync(target, configPath);
    else linkSync(target, configPath);
    publish();
    expect(readFileSync(target, "utf8")).toBe("must not change");
    expect(lstatSync(configPath).isSymbolicLink()).toBe(false);
    expect(statSync(configPath).nlink).toBe(1);
    expect(readCopilotSDKMCPConfig(configPath)).toBe(source);
  });

  it.each(["symlink", "hardlink"])("rejects a catalog replaced by a %s even when bytes match", kind => {
    publish();
    const target = join(root, "copy.json");
    writeFileSync(target, source);
    rmSync(configPath);
    if (kind === "symlink") symlinkSync(target, configPath);
    else linkSync(target, configPath);
    expect(() => readCopilotSDKMCPConfig(configPath)).toThrow("Cannot read generated Copilot SDK MCP configuration");
  });

  it.each(["gh-aw", "mcp-config"])("rejects a redirected %s directory without writing outside the runtime tree", name => {
    const target = join(root, "unrelated");
    mkdirSync(target);
    const link = name === "gh-aw" ? join(root, "gh-aw") : dirname(configPath);
    if (name === "mcp-config") mkdirSync(dirname(link));
    symlinkSync(target, link);
    expect(() => writeCopilotSDKMCPConfig(source)).toThrow("configuration directory must not be a symlink");
    expect(readdirSync(target)).toEqual([]);
  });

  it("rejects a redirected catalog directory when reading", () => {
    publish();
    const target = join(root, "redirected");
    mkdirSync(target);
    writeFileSync(join(target, "copilot-sdk.json"), source);
    rmSync(configPath);
    rmSync(dirname(configPath), { recursive: true });
    symlinkSync(target, dirname(configPath));
    expect(() => readCopilotSDKMCPConfig(configPath)).toThrow("Cannot read generated Copilot SDK MCP configuration");
  });

  it.each(["gh-aw", "mcp-config"])("rejects an existing %s directory writable by other users", name => {
    mkdirSync(dirname(configPath), { recursive: true });
    const directory = name === "gh-aw" ? join(root, name) : dirname(configPath);
    chmodSync(directory, 0o777);
    expect(() => writeCopilotSDKMCPConfig(source)).toThrow("writable by other users");
    expect(readdirSync(dirname(configPath))).toEqual([]);
  });

  it("does not publish a partial catalog or leave temporary files after a failed rename", () => {
    publish();
    vi.spyOn(require("node:fs"), "renameSync").mockImplementation(() => {
      throw new Error("test rename failure");
    });
    expect(() => writeCopilotSDKMCPConfig("{}")).toThrow("test rename failure");
    expect(readCopilotSDKMCPConfig(configPath)).toBe(source);
    expect(readdirSync(dirname(configPath))).toEqual(["copilot-sdk.json"]);
  });

  it("hashes raw bytes rather than a parsed and reserialized catalog", () => {
    expect(digestCopilotSDKMCPConfig(source)).not.toBe(digestCopilotSDKMCPConfig(`${source}\n`));
  });

  it("publishes the digest without logging catalog contents or replacing other step variables", () => {
    const envFile = join(root, "github-env");
    writeFileSync(envFile, "PREVIOUS=value\n");
    vi.stubEnv("GITHUB_ENV", envFile);
    publishCopilotSDKMCPConfig(source);
    expect(readFileSync(envFile, "utf8")).toBe(`PREVIOUS=value\n${COPILOT_SDK_MCP_CONFIG_SHA256}=${digestCopilotSDKMCPConfig(source)}\n`);
    expect(readFileSync(envFile, "utf8")).not.toContain("test-only");
  });

  it("fails closed when the trusted environment file is unavailable", () => {
    vi.stubEnv("GITHUB_ENV", "");
    expect(() => publishCopilotSDKMCPConfig(source)).toThrow("GITHUB_ENV is required");
    vi.stubEnv("GITHUB_ENV", join(root, "missing-env"));
    expect(() => publishCopilotSDKMCPConfig(source)).toThrow("Cannot publish Copilot SDK MCP configuration integrity");
  });
});
