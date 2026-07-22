// @ts-check
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import os from "os";
import path from "path";

import { rewriteUrl, normalizeGatewayEntry, filterAndTransformServers, writeSecureOutput, loadGatewayContext, logCLIFilters, logServerStats } from "./convert_gateway_config_shared.cjs";

describe("rewriteUrl", () => {
  it("replaces hostname and port with the provided url prefix", () => {
    expect(rewriteUrl("http://old.example:81/mcp/github", "http://host.docker.internal:80")).toBe("http://host.docker.internal:80/mcp/github");
  });

  it("preserves path segments after /mcp/", () => {
    expect(rewriteUrl("http://localhost:9090/mcp/some/deep/path", "http://myhost:1234")).toBe("http://myhost:1234/mcp/some/deep/path");
  });

  it("handles url with no path beyond /mcp/", () => {
    expect(rewriteUrl("http://localhost:80/mcp/", "http://newhost:80")).toBe("http://newhost:80/mcp/");
  });

  it("does not rewrite non-mcp URLs", () => {
    const url = "http://localhost/other/path";
    expect(rewriteUrl(url, "http://newhost:80")).toBe(url);
  });
});

describe("normalizeGatewayEntry", () => {
  it("rewrites url and applies mutation", () => {
    const entry = { type: "ignored", url: "http://old/mcp/github" };
    const result = normalizeGatewayEntry(entry, "http://host:80", t => {
      t.type = "http";
    });
    expect(result.type).toBe("http");
    expect(result.url).toBe("http://host:80/mcp/github");
  });

  it("does not mutate the original entry", () => {
    const entry = { url: "http://old/mcp/github", extra: "value" };
    const result = normalizeGatewayEntry(entry, "http://host:80");
    expect(entry.url).toBe("http://old/mcp/github");
    expect(result).not.toBe(entry);
  });

  it("works without a mutate callback", () => {
    const entry = { url: "http://old/mcp/github" };
    const result = normalizeGatewayEntry(entry, "http://host:80");
    expect(result.url).toBe("http://host:80/mcp/github");
  });

  it("skips url rewrite when url field is missing", () => {
    const entry = { type: "http" };
    const result = normalizeGatewayEntry(entry, "http://host:80");
    expect(result.type).toBe("http");
    expect(result.url).toBeUndefined();
  });
});

describe("filterAndTransformServers", () => {
  it("excludes CLI-mounted servers", () => {
    const servers = {
      github: { url: "http://old/mcp/github" },
      playwright: { url: "http://old/mcp/playwright" },
    };
    const result = filterAndTransformServers(servers, new Set(["playwright"]), (_n, e) => e);
    expect(Object.keys(result)).toEqual(["github"]);
  });

  it("applies the transform to each server", () => {
    const servers = { github: { url: "http://old/mcp/github" } };
    const result = filterAndTransformServers(servers, new Set(), (_n, e) => {
      e.type = "http";
      return e;
    });
    expect(result.github.type).toBe("http");
  });

  it("returns empty object when all servers are CLI-mounted", () => {
    const servers = { github: { url: "http://old/mcp/github" } };
    const result = filterAndTransformServers(servers, new Set(["github"]), (_n, e) => e);
    expect(result).toEqual({});
  });
});

describe("writeSecureOutput", () => {
  /** @type {string} */
  let dir;

  beforeEach(() => {
    dir = fs.mkdtempSync(path.join(os.tmpdir(), "gateway-test-"));
  });

  afterEach(() => {
    fs.rmSync(dir, { recursive: true, force: true });
  });

  it("writes file with mode 0o600", () => {
    const outputPath = path.join(dir, "output.json");
    writeSecureOutput(outputPath, '{"key":"value"}');
    expect(fs.readFileSync(outputPath, "utf8")).toBe('{"key":"value"}');
    expect(fs.statSync(outputPath).mode & 0o777).toBe(0o600);
  });

  it("creates nested directories as needed", () => {
    const outputPath = path.join(dir, "deep/nested/output.json");
    writeSecureOutput(outputPath, "{}");
    expect(fs.existsSync(outputPath)).toBe(true);
  });

  it("overwrites existing file and resets permissions to 0o600", () => {
    const outputPath = path.join(dir, "output.json");
    fs.writeFileSync(outputPath, "old");
    fs.chmodSync(outputPath, 0o644);
    writeSecureOutput(outputPath, "new");
    expect(fs.readFileSync(outputPath, "utf8")).toBe("new");
    expect(fs.statSync(outputPath).mode & 0o777).toBe(0o600);
  });

  it("throws on write failure", () => {
    const spy = vi.spyOn(fs, "mkdirSync").mockImplementationOnce(() => {
      throw new Error("EACCES: permission denied, mkdir");
    });
    try {
      expect(() => writeSecureOutput(path.join(dir, "output.json"), "{}")).toThrow();
    } finally {
      spy.mockRestore();
    }
  });
});

describe("loadGatewayContext", () => {
  /** @type {NodeJS.ProcessEnv} */
  let savedEnv;

  beforeEach(() => {
    savedEnv = { ...process.env };
  });

  afterEach(() => {
    for (const key of Object.keys(process.env)) {
      if (!(key in savedEnv)) delete process.env[key];
    }
    Object.assign(process.env, savedEnv);
  });

  it("throws when MCP_GATEWAY_OUTPUT is missing", () => {
    delete process.env.MCP_GATEWAY_OUTPUT;
    expect(() => loadGatewayContext()).toThrow("MCP_GATEWAY_OUTPUT");
  });

  it("throws when gateway output file does not exist", () => {
    process.env.MCP_GATEWAY_OUTPUT = "/nonexistent/gateway.json";
    expect(() => loadGatewayContext()).toThrow("Gateway output file not found");
  });

  it("throws when MCP_GATEWAY_DOMAIN is missing", () => {
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), "gateway-test-"));
    const gatewayFile = path.join(dir, "gateway.json");
    fs.writeFileSync(gatewayFile, JSON.stringify({ mcpServers: {} }));
    process.env.MCP_GATEWAY_OUTPUT = gatewayFile;
    delete process.env.MCP_GATEWAY_DOMAIN;
    try {
      expect(() => loadGatewayContext()).toThrow("MCP_GATEWAY_DOMAIN");
    } finally {
      fs.rmSync(dir, { recursive: true, force: true });
    }
  });

  it("parses gateway output and returns structured context", () => {
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), "gateway-test-"));
    const gatewayFile = path.join(dir, "gateway.json");
    const servers = { github: { url: "http://old/mcp/github", type: "http" } };
    fs.writeFileSync(gatewayFile, JSON.stringify({ mcpServers: servers }));
    process.env.MCP_GATEWAY_OUTPUT = gatewayFile;
    process.env.MCP_GATEWAY_DOMAIN = "host.docker.internal";
    process.env.MCP_GATEWAY_PORT = "80";
    process.env.GH_AW_MCP_CLI_SERVERS = "[]";

    try {
      const ctx = loadGatewayContext();
      expect(ctx.domain).toBe("host.docker.internal");
      expect(ctx.port).toBe("80");
      expect(ctx.urlPrefix).toBe("http://host.docker.internal:80");
      expect(ctx.cliServers).toBeInstanceOf(Set);
      expect(Object.keys(ctx.servers)).toContain("github");
    } finally {
      fs.rmSync(dir, { recursive: true, force: true });
    }
  });

  it("collects extraRequiredEnv values into extraEnv", () => {
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), "gateway-test-"));
    const gatewayFile = path.join(dir, "gateway.json");
    fs.writeFileSync(gatewayFile, JSON.stringify({ mcpServers: {} }));
    process.env.MCP_GATEWAY_OUTPUT = gatewayFile;
    process.env.MCP_GATEWAY_DOMAIN = "localhost";
    process.env.MCP_GATEWAY_PORT = "9090";
    process.env.GH_AW_MCP_CLI_SERVERS = "[]";
    process.env.MY_CUSTOM_VAR = "custom-value";

    try {
      const ctx = loadGatewayContext({ extraRequiredEnv: ["MY_CUSTOM_VAR"] });
      expect(ctx.extraEnv.MY_CUSTOM_VAR).toBe("custom-value");
    } finally {
      fs.rmSync(dir, { recursive: true, force: true });
    }
  });
});

describe("logCLIFilters and logServerStats", () => {
  /** @type {{ info: ReturnType<typeof vi.fn>, warning: ReturnType<typeof vi.fn> }} */
  let mockCore;
  /** @type {unknown} */
  let originalCore;

  beforeEach(() => {
    originalCore = global.core;
    mockCore = { info: vi.fn(), warning: vi.fn() };
    // @ts-ignore
    global.core = mockCore;
  });

  afterEach(() => {
    // @ts-ignore
    global.core = originalCore;
  });

  it("logCLIFilters calls core.info when servers are present", () => {
    logCLIFilters(new Set(["github", "playwright"]));
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("github"));
  });

  it("logCLIFilters does nothing when no CLI servers", () => {
    logCLIFilters(new Set());
    expect(mockCore.info).not.toHaveBeenCalled();
  });

  it("logServerStats reports included and filtered counts", () => {
    const servers = { github: {}, playwright: {}, bash: {} };
    logServerStats(servers, 2);
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringMatching(/2 included/));
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringMatching(/1 filtered/));
  });
});
