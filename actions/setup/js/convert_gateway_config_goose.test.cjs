// @ts-check
import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { createRequire } from "module";
import { mkdtempSync, rmSync, writeFileSync, readFileSync, statSync } from "fs";
import { join } from "path";
import { tmpdir } from "os";

const req = createRequire(import.meta.url);
const { transformGooseEntry, main } = req("./convert_gateway_config_goose.cjs");

describe("convert_gateway_config_goose", () => {
  describe("transformGooseEntry", () => {
    const urlPrefix = "http://localhost:8080";

    it("sets type to streamable_http", () => {
      const entry = { type: "http", url: "http://old/mcp/github" };
      const result = transformGooseEntry(entry, urlPrefix);
      expect(result.type).toBe("streamable_http");
    });

    it("rewrites the url to use the configured domain and port", () => {
      const entry = { url: "http://host.docker.internal:80/mcp/github" };
      const result = transformGooseEntry(entry, urlPrefix);
      expect(result.url).toBe("http://localhost:8080/mcp/github");
    });

    it("removes the tools field from the entry", () => {
      const entry = {
        type: "http",
        url: "http://old/mcp/server",
        headers: { Authorization: "******" },
        tools: ["read", "write"],
      };
      const result = transformGooseEntry(entry, urlPrefix);
      expect(result).not.toHaveProperty("tools");
      expect(result.headers).toEqual({ Authorization: "******" });
    });

    it("does not mutate the original entry (including nested fields)", () => {
      const entry = {
        type: "http",
        url: "http://old/mcp/github",
        headers: { Authorization: "******", "X-Custom": "value" },
        tools: ["read", "write"],
      };
      const original = JSON.parse(JSON.stringify(entry));
      transformGooseEntry(entry, urlPrefix);
      expect(entry).toEqual(original);
    });

    it("handles entries without a url field gracefully", () => {
      const entry = { type: "http", headers: { Authorization: "******" } };
      const result = transformGooseEntry(entry, urlPrefix);
      expect(result).not.toHaveProperty("url");
      expect(result.type).toBe("streamable_http");
      expect(result.headers).toEqual({ Authorization: "******" });
    });

    it("handles entries with non-string url values unchanged", () => {
      const entry = { type: "http", url: 42 };
      const result = transformGooseEntry(entry, urlPrefix);
      expect(result.url).toBe(42);
    });

    it("works with a different urlPrefix", () => {
      const entry = { url: "http://host.docker.internal:80/mcp/playwright" };
      const result = transformGooseEntry(entry, "http://host.docker.internal:9090");
      expect(result.url).toBe("http://host.docker.internal:9090/mcp/playwright");
    });

    it("handles entries with empty object", () => {
      const entry = {};
      const result = transformGooseEntry(entry, urlPrefix);
      expect(result.type).toBe("streamable_http");
    });
  });

  describe("main", () => {
    /** @type {string} */
    let tempDir;
    /** @type {string} */
    let workspace;
    /** @type {string} */
    let gatewayOutputFile;
    /** @type {Record<string, string | undefined>} */
    let savedEnv;

    beforeEach(() => {
      tempDir = mkdtempSync(join(tmpdir(), "goose-config-test-"));
      workspace = join(tempDir, "workspace");
      gatewayOutputFile = join(tempDir, "gateway-output.json");

      savedEnv = {
        MCP_GATEWAY_OUTPUT: process.env.MCP_GATEWAY_OUTPUT,
        MCP_GATEWAY_DOMAIN: process.env.MCP_GATEWAY_DOMAIN,
        MCP_GATEWAY_HOST_DOMAIN: process.env.MCP_GATEWAY_HOST_DOMAIN,
        MCP_GATEWAY_PORT: process.env.MCP_GATEWAY_PORT,
        GITHUB_WORKSPACE: process.env.GITHUB_WORKSPACE,
        GH_AW_MCP_CLI_SERVERS: process.env.GH_AW_MCP_CLI_SERVERS,
      };

      process.env.MCP_GATEWAY_DOMAIN = "host.docker.internal";
      process.env.MCP_GATEWAY_PORT = "80";
      process.env.MCP_GATEWAY_HOST_DOMAIN = "localhost";
      process.env.GITHUB_WORKSPACE = workspace;
      process.env.GH_AW_MCP_CLI_SERVERS = "[]";
    });

    afterEach(() => {
      for (const [key, value] of Object.entries(savedEnv)) {
        if (value === undefined) {
          delete process.env[key];
        } else {
          process.env[key] = value;
        }
      }
      rmSync(tempDir, { recursive: true, force: true });
    });

    /**
     * @param {object} mcpServers - MCP servers config to write to the gateway output
     */
    function writeGatewayOutput(mcpServers) {
      writeFileSync(gatewayOutputFile, JSON.stringify({ mcpServers }));
      process.env.MCP_GATEWAY_OUTPUT = gatewayOutputFile;
    }

    it("writes mcp.json to .goose directory in workspace", () => {
      writeGatewayOutput({ github: { url: "http://host.docker.internal:80/mcp/github" } });

      main();

      const configPath = join(workspace, ".goose", "mcp.json");
      const config = JSON.parse(readFileSync(configPath, "utf8"));
      expect(config).toHaveProperty("mcpServers");
    });

    it("rewrites server URLs to use MCP_GATEWAY_HOST_DOMAIN", () => {
      writeGatewayOutput({ github: { type: "http", url: "http://host.docker.internal:80/mcp/github" } });

      main();

      const config = JSON.parse(readFileSync(join(workspace, ".goose", "mcp.json"), "utf8"));
      expect(config.mcpServers.github.url).toBe("http://localhost:80/mcp/github");
    });

    it("sets type to streamable_http on all server entries", () => {
      writeGatewayOutput({
        github: { type: "http", url: "http://old/mcp/github" },
        playwright: { type: "http", url: "http://old/mcp/playwright" },
      });

      main();

      const config = JSON.parse(readFileSync(join(workspace, ".goose", "mcp.json"), "utf8"));
      expect(config.mcpServers.github.type).toBe("streamable_http");
      expect(config.mcpServers.playwright.type).toBe("streamable_http");
    });

    it("filters out CLI-mounted servers", () => {
      writeGatewayOutput({
        github: { url: "http://old/mcp/github" },
        playwright: { url: "http://old/mcp/playwright" },
      });
      process.env.GH_AW_MCP_CLI_SERVERS = JSON.stringify(["playwright"]);

      main();

      const config = JSON.parse(readFileSync(join(workspace, ".goose", "mcp.json"), "utf8"));
      expect(config.mcpServers).toHaveProperty("github");
      expect(config.mcpServers).not.toHaveProperty("playwright");
    });

    it("writes mcp.json with 0o600 file permissions", () => {
      writeGatewayOutput({ github: { url: "http://old/mcp/github" } });

      main();

      const configPath = join(workspace, ".goose", "mcp.json");
      const mode = statSync(configPath).mode & 0o777;
      expect(mode).toBe(0o600);
    });

    it("preserves headers on server entries", () => {
      writeGatewayOutput({ github: { url: "http://old/mcp/github", headers: { Authorization: "******" } } });

      main();

      const config = JSON.parse(readFileSync(join(workspace, ".goose", "mcp.json"), "utf8"));
      expect(config.mcpServers.github.headers).toEqual({ Authorization: "******" });
    });
  });
});
