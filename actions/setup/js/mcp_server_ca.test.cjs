import fs from "fs";
import os from "os";
import path from "path";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { prepareMCPServerCAs } from "./mcp_server_ca.cjs";

describe("per-server MCP CA mounts", () => {
  let dir;
  let env;
  beforeEach(() => {
    dir = fs.mkdtempSync(path.join(os.tmpdir(), "mcp-ca-"));
    env = { GITHUB_WORKSPACE: dir, RUNNER_TEMP: dir };
    fs.writeFileSync(path.join(dir, "private ca.pem"), "certificate validated by gateway", { mode: 0o600 });
  });
  afterEach(() => fs.rmSync(dir, { recursive: true, force: true }));

  it.each(["private ca.pem", "${GITHUB_WORKSPACE}/private ca.pem", "${RUNNER_TEMP}/private ca.pem", "absolute", "symlink"])("resolves %s into a read-only file mount", source => {
    if (source === "absolute") source = path.join(dir, "private ca.pem");
    if (source === "symlink") {
      fs.symlinkSync(path.join(dir, "private ca.pem"), path.join(dir, "link.pem"));
      source = "link.pem";
    }
    const server = { type: "http", url: "https://private.example/mcp", caCertFile: source };
    const publicServer = { type: "http", url: "https://public.example/mcp" };
    const config = { mcpServers: { private: server, public: publicServer } };
    const args = prepareMCPServerCAs(config, env);
    expect(args).toEqual(["--mount", `type=bind,source=${fs.realpathSync(path.join(dir, "private ca.pem"))},target=${server.caCertFile},readonly`]);
    expect(server.caCertFile).toMatch(/^\/etc\/gh-aw\/mcp-ca\/[a-f0-9]{64}\.pem$/);
    expect(publicServer).toEqual({ type: "http", url: "https://public.example/mcp" });
    expect(env).toEqual({ GITHUB_WORKSPACE: dir, RUNNER_TEMP: dir });
    expect(fs.readFileSync(path.join(dir, "private ca.pem"), "utf8")).toBe("certificate validated by gateway");
  });

  it("gives distinct servers distinct paths, with no mount or trust settings for unconfigured servers", () => {
    const config = {
      mcpServers: {
        first: { type: "http", url: "https://first.example", caCertFile: "private ca.pem" },
        second: { type: "http", url: "https://second.example", caCertFile: "private ca.pem" },
        unconfigured: { type: "http", url: "https://third.example" },
      },
    };
    expect(prepareMCPServerCAs(config, env)).toHaveLength(4);
    expect(config.mcpServers.first.caCertFile).not.toBe(config.mcpServers.second.caCertFile);
    expect(config.mcpServers.unconfigured).not.toHaveProperty("caCertFile");
    expect(prepareMCPServerCAs({ mcpServers: { unconfigured: config.mcpServers.unconfigured } }, env)).toEqual([]);
    expect(prepareMCPServerCAs({}, {})).toEqual([]);
  });

  it.each(["missing.pem", ".", "", null, false, "${RUNNER_TEMP}/missing.pem"])("rejects missing, directory or malformed CA paths: %s", source => {
    expect(() => prepareMCPServerCAs({ mcpServers: { private: { type: "http", url: "https://private.example", caCertFile: source } } }, env)).toThrow(/private.*readable PEM file/);
  });

  it("rejects unsupported transports, missing runtime roots and mount separators without exposing file contents", () => {
    for (const server of [
      { type: "stdio", caCertFile: "private ca.pem" },
      { type: "http", url: "http://private.example", caCertFile: "private ca.pem" },
    ]) {
      expect(() => prepareMCPServerCAs({ mcpServers: { private: server } }, env)).toThrow(/HTTPS HTTP server/);
    }
    for (const source of ["private ca.pem", "${RUNNER_TEMP}/private ca.pem"]) {
      expect(() => prepareMCPServerCAs({ mcpServers: { private: { type: "http", url: "https://private.example", caCertFile: source } } }, {})).toThrow(/readable PEM file/);
    }
    fs.writeFileSync(path.join(dir, "comma,ca.pem"), "do-not-print");
    expect(() => prepareMCPServerCAs({ mcpServers: { private: { type: "http", url: "https://private.example", caCertFile: "comma,ca.pem" } } }, env)).toThrow(/readable PEM file/);
  });
});
