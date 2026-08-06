// @ts-check
import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { createRequire } from "module";
import { mkdtempSync, rmSync, writeFileSync } from "fs";
import { join } from "path";
import { tmpdir } from "os";

const req = createRequire(import.meta.url);
const { toCodexTomlSection, main } = req("./convert_gateway_config_codex.cjs");

describe("convert_gateway_config_codex", () => {
  describe("toCodexTomlSection", () => {
    it("emits a TOML section with the correct server URL", () => {
      const toml = toCodexTomlSection("github", { headers: { Authorization: "token abc" } }, "http://172.30.0.1:80");
      expect(toml).toContain("[mcp_servers.github]");
      expect(toml).toContain('url = "http://172.30.0.1:80/mcp/github"');
    });

    it("includes the Authorization header in http_headers", () => {
      const toml = toCodexTomlSection("myserver", { headers: { Authorization: "******" } }, "http://host:80");
      expect(toml).toContain('http_headers = { Authorization = "******" }');
    });

    it("emits an empty Authorization when no headers present", () => {
      const toml = toCodexTomlSection("noauth", {}, "http://host:80");
      expect(toml).toContain('http_headers = { Authorization = "" }');
    });

    it("ignores non-string header values", () => {
      const toml = toCodexTomlSection("srv", { headers: { Authorization: 123, Other: "keep" } }, "http://host:80");
      expect(toml).toContain('http_headers = { Authorization = "" }');
    });
  });

  describe("main", () => {
    /** @type {string} */
    let tempDir;
    /** @type {string} */
    let gatewayOutputFile;
    /** @type {Record<string, string | undefined>} */
    let savedEnv;

    beforeEach(() => {
      tempDir = mkdtempSync(join(tmpdir(), "codex-config-test-"));
      gatewayOutputFile = join(tempDir, "gateway-output.json");

      savedEnv = {
        MCP_GATEWAY_OUTPUT: process.env.MCP_GATEWAY_OUTPUT,
        MCP_GATEWAY_DOMAIN: process.env.MCP_GATEWAY_DOMAIN,
        MCP_GATEWAY_PORT: process.env.MCP_GATEWAY_PORT,
        RUNNER_TEMP: process.env.RUNNER_TEMP,
        GH_AW_MCP_CLI_SERVERS: process.env.GH_AW_MCP_CLI_SERVERS,
      };

      process.env.MCP_GATEWAY_DOMAIN = "host.docker.internal";
      process.env.MCP_GATEWAY_PORT = "80";
      process.env.RUNNER_TEMP = tempDir;
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

    it("resolves host.docker.internal to 172.30.0.1 in TOML server URLs", () => {
      writeGatewayOutput({
        github: { url: "http://host.docker.internal:80/mcp/github", headers: { Authorization: "token abc" } },
      });

      const toml = main();

      expect(toml).toContain('url = "http://172.30.0.1:80/mcp/github"');
      expect(toml).not.toContain("host.docker.internal");
    });

    it("uses the domain directly when it is not host.docker.internal", () => {
      process.env.MCP_GATEWAY_DOMAIN = "gateway.internal";
      writeGatewayOutput({
        github: { url: "http://gateway.internal:80/mcp/github", headers: { Authorization: "token" } },
      });

      const toml = main();

      expect(toml).toContain('url = "http://gateway.internal:80/mcp/github"');
    });

    it("filters out CLI-mounted servers before serializing", () => {
      writeGatewayOutput({
        github: { url: "http://host.docker.internal:80/mcp/github", headers: {} },
        playwright: { url: "http://host.docker.internal:80/mcp/playwright", headers: {} },
      });
      process.env.GH_AW_MCP_CLI_SERVERS = JSON.stringify(["playwright"]);

      const toml = main();

      expect(toml).toContain("[mcp_servers.github]");
      expect(toml).not.toContain("[mcp_servers.playwright]");
    });

    it("includes TOML persistence header in output", () => {
      writeGatewayOutput({ github: { url: "http://host.docker.internal:80/mcp/github", headers: {} } });

      const toml = main();

      expect(toml).toContain('[history]\npersistence = "none"');
    });
  });
});
