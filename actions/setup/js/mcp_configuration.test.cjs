import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { normalizeMcpConfigurationData, writeMcpConfiguration } from "./mcp_configuration.cjs";
import { normalizeUnifiedSessionEvent } from "./unified_session_payload.cjs";
import { writeUnifiedSession } from "./unified_session.cjs";
import { generatePlainTextSummary, generateCopilotCliStyleSummary, generateConversationMarkdown } from "./log_parser_shared.cjs";
import { validateSession } from "./scripts/validate_session.cjs";

describe("Privacy-preserving MCP configuration", () => {
  let root;
  let originalEnv;
  beforeEach(() => {
    root = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-mcp-configuration-"));
    originalEnv = { ...process.env };
  });
  afterEach(() => {
    process.env = originalEnv;
    vi.restoreAllMocks();
    fs.rmSync(root, { recursive: true, force: true });
  });

  const config = {
    mcpServers: {
      github: {
        type: "http",
        url: "https://PRIVATE_USER:PRIVATE_PASSWORD@example.com/PRIVATE_PATH?key=PRIVATE_QUERY",
        headers: { Authorization: "Bearer PRIVATE_HEADER", "X-Custom": "PRIVATE_CUSTOM_HEADER" },
        env: { PRIVATE_ENV_NAME: "PRIVATE_ENV_VALUE" },
        auth: { type: "github-oidc", audience: "PRIVATE_AUDIENCE" },
        "guard-policies": { private: "PRIVATE_POLICY" },
        tools: ["get_issue", "list_issues"],
        required: false,
        extension: "PRIVATE_EXTENSION",
      },
      local: {
        container: "PRIVATE_IMAGE",
        command: "PRIVATE_COMMAND",
        args: ["--key=PRIVATE_ARGUMENT"],
        entrypoint: "PRIVATE_ENTRYPOINT",
        entrypointArgs: ["PRIVATE_ENTRYPOINT_ARGUMENT"],
        mounts: ["PRIVATE_MOUNT"],
        tools: [],
        required: true,
      },
      defaults: { type: "stdio" },
    },
    gateway: { apiKey: "PRIVATE_GATEWAY_KEY", agentId: "PRIVATE_AGENT_ID", domain: "PRIVATE_DOMAIN" },
    future: { secret: "PRIVATE_FUTURE" },
  };
  const inventory = {
    servers: [
      { serverName: "github", transport: "http", required: false, tools: ["get_issue", "list_issues"] },
      { serverName: "local", transport: "stdio", required: true, tools: [] },
      { serverName: "defaults", transport: "stdio" },
    ],
  };

  it.each(["", "threat-detection/"])("persists and collects a sanitized %s gateway snapshot without mutating runtime configuration", prefix => {
    const before = structuredClone(config);
    const directory = path.join(root, prefix, "mcp-logs");
    writeMcpConfiguration(directory, config);
    const snapshot = fs.readFileSync(path.join(directory, "configuration.jsonl"), "utf8");
    expect(snapshot).not.toContain("PRIVATE_");
    expect(JSON.parse(snapshot)).toEqual({ type: "mcp.configuration", timestamp: expect.any(String), data: inventory });
    expect(fs.statSync(path.join(directory, "configuration.jsonl")).mode & 0o777).toBe(0o600);
    expect(config).toEqual(before);
    const events = writeUnifiedSession({ rootDir: root });
    expect(events.filter(event => event.type === "mcp.configuration")).toMatchObject([
      {
        data: inventory,
        provenance: { component: "mcp", phase: prefix ? "detection" : "agent", path: `${prefix}mcp-logs/configuration.jsonl`, index: 0 },
      },
    ]);
    const persisted = fs.readFileSync(path.join(root, "usage/aw_session.jsonl"), "utf8");
    expect(persisted).not.toContain("PRIVATE_");
    expect(validateSession(persisted)).toBe(events.length);
    expect(writeUnifiedSession({ rootDir: root })).toEqual(events);
    for (const render of [generatePlainTextSummary, generateCopilotCliStyleSummary, generateConversationMarkdown]) {
      const result = render(events);
      const output = typeof result === "string" ? result : result.markdown;
      expect(output).toContain("mcp.configuration");
      expect(output).toContain("serverName=github");
      expect(output).toContain("required=false");
      expect(output).not.toContain("PRIVATE_");
    }
  });

  it("reapplies the field allowlist to imported snapshots and renderer input", () => {
    const source = { ...inventory, secret: "PRIVATE_TOP_LEVEL", servers: inventory.servers.map(server => ({ ...server, args: ["PRIVATE_ARGUMENT"], headers: { token: "PRIVATE_HEADER" }, extension: "PRIVATE_EXTENSION" })) };
    expect(normalizeMcpConfigurationData(source)).toEqual(inventory);
    expect(normalizeUnifiedSessionEvent({ type: "mcp.configuration", data: source }).data).toEqual(inventory);
    expect(normalizeMcpConfigurationData({ servers: [null, false, {}, { serverName: "server", transport: "PRIVATE_TRANSPORT", required: "PRIVATE_REQUIRED", tools: [null, {}, "lookup"] }] })).toEqual({
      servers: [{ serverName: "server", tools: ["lookup"] }],
    });
  });

  it("preserves an explicitly empty inventory and wildcard tool filters", () => {
    writeMcpConfiguration(path.join(root, "mcp-logs"), { mcpServers: {} });
    expect(writeUnifiedSession({ rootDir: root }).find(event => event.type === "mcp.configuration").data).toEqual({ servers: [] });
    expect(normalizeMcpConfigurationData({ servers: [{ serverName: "all", tools: ["*"] }] })).toEqual({ servers: [{ serverName: "all", tools: ["*"] }] });
  });

  it("redacts registered secrets even when they occur in retained identifiers", () => {
    process.env.GH_AW_SECRET_NAMES = "CONFIG_TEST";
    process.env.SECRET_CONFIG_TEST = "private-registered-configuration-secret";
    writeMcpConfiguration(path.join(root, "mcp-logs"), { mcpServers: { [process.env.SECRET_CONFIG_TEST]: { tools: [process.env.SECRET_CONFIG_TEST] } } });
    expect(fs.readFileSync(path.join(root, "mcp-logs/configuration.jsonl"), "utf8")).not.toContain(process.env.SECRET_CONFIG_TEST);
  });

  it("fails explicitly for missing or invalid inventories", () => {
    for (const mcpServers of [undefined, null, [], "PRIVATE_INVALID"]) {
      expect(() => writeMcpConfiguration(path.join(root, "mcp-logs"), { mcpServers })).toThrow("MCP configuration must contain an mcpServers object");
    }
    for (const servers of [undefined, null, {}, "PRIVATE_INVALID"]) {
      expect(() => normalizeMcpConfigurationData({ servers })).toThrow("MCP configuration snapshot must contain a servers array");
    }
  });

  it("rejects a symlink log directory without writing through it", () => {
    const target = path.join(root, "target");
    fs.mkdirSync(target);
    const directory = path.join(root, "mcp-logs");
    fs.symlinkSync(target, directory);
    expect(() => writeMcpConfiguration(directory, config)).toThrow("must not be a symlink");
    expect(fs.readdirSync(target)).toEqual([]);
  });

  it("does not retain a stale inventory after a failed publication", () => {
    const directory = path.join(root, "mcp-logs");
    writeMcpConfiguration(directory, config);
    const write = fs.writeFileSync;
    vi.spyOn(fs, "writeFileSync").mockImplementation((file, ...args) => {
      if (String(file).endsWith("configuration.jsonl.tmp")) throw new Error("fixture write denied");
      return write(file, ...args);
    });
    expect(() => writeMcpConfiguration(directory, config)).toThrow("fixture write denied");
    expect(fs.existsSync(path.join(directory, "configuration.jsonl"))).toBe(false);
  });
});
