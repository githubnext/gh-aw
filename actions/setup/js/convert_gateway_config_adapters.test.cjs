// @ts-check
import { describe, it, expect } from "vitest";
import fs from "fs";
import os from "os";
import path from "path";

import { rewriteUrl, filterAndTransformServers, writeSecureOutput } from "./convert_gateway_config_shared.cjs";
import { transformClaudeEntry } from "./convert_gateway_config_claude.cjs";
import { transformCopilotEntry } from "./convert_gateway_config_copilot.cjs";
import { transformGeminiEntry } from "./convert_gateway_config_gemini.cjs";
import { toCodexTomlSection } from "./convert_gateway_config_codex.cjs";

describe("convert gateway config shared pipeline", () => {
  it("rewrites gateway urls to the provided domain/port", () => {
    const rewritten = rewriteUrl("http://old.example:81/mcp/github", "http://host.docker.internal:80");
    expect(rewritten).toBe("http://host.docker.internal:80/mcp/github");
  });

  it("filters CLI-mounted servers before applying engine transforms", () => {
    const servers = {
      github: { url: "http://old/mcp/github" },
      playwright: { url: "http://old/mcp/playwright" },
    };
    const filtered = filterAndTransformServers(servers, new Set(["playwright"]), (_name, entry) => entry);
    expect(filtered).toEqual({
      github: { url: "http://old/mcp/github" },
    });
  });

  it("enforces output file permission mode to 0o600 even when file already exists", () => {
    const outputDir = fs.mkdtempSync(path.join(os.tmpdir(), "gateway-config-"));
    const outputPath = path.join(outputDir, "mcp-servers.json");
    fs.writeFileSync(outputPath, "old");
    fs.chmodSync(outputPath, 0o644);

    writeSecureOutput(outputPath, "{}");

    const mode = fs.statSync(outputPath).mode & 0o777;
    expect(mode).toBe(0o600);
    expect(fs.readFileSync(outputPath, "utf8")).toBe("{}");

    fs.rmSync(outputDir, { recursive: true, force: true });
  });
});

describe("convert gateway config adapters", () => {
  const urlPrefix = "http://host.docker.internal:80";

  it("claude adapter enforces type=http and rewrites url", () => {
    const converted = transformClaudeEntry({ type: "ignored", url: "http://old/mcp/github", headers: { Authorization: "token" } }, urlPrefix);
    expect(converted).toEqual({
      type: "http",
      url: "http://host.docker.internal:80/mcp/github",
      headers: { Authorization: "token" },
    });
  });

  it("copilot adapter adds tools wildcard only when missing", () => {
    const withoutTools = transformCopilotEntry({ url: "http://old/mcp/github" }, urlPrefix);
    expect(withoutTools.tools).toEqual(["*"]);
    expect(withoutTools.url).toBe("http://host.docker.internal:80/mcp/github");

    const withTools = transformCopilotEntry({ tools: ["repo.read"], url: "http://old/mcp/github" }, urlPrefix);
    expect(withTools.tools).toEqual(["repo.read"]);
  });

  it("gemini adapter removes type while keeping other fields", () => {
    const converted = transformGeminiEntry({ type: "http", url: "http://old/mcp/github", headers: { Authorization: "token" } }, urlPrefix);
    expect(converted).toEqual({
      url: "http://host.docker.internal:80/mcp/github",
      headers: { Authorization: "token" },
    });
  });

  it("codex adapter emits toml section with rewritten URL and auth header", () => {
    const toml = toCodexTomlSection("github", { headers: { Authorization: "Bearer abc" } }, "http://172.30.0.1:80");
    expect(toml).toContain("[mcp_servers.github]");
    expect(toml).toContain('url = "http://172.30.0.1:80/mcp/github"');
    expect(toml).toContain('http_headers = { Authorization = "Bearer abc" }');
  });
});
