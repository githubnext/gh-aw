import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import os from "os";
import path from "path";
import safeOutputsMCPServerHTTP from "./safe_outputs_mcp_server_http.cjs";
import { normalizeSafeOutputToolArguments } from "./safe_outputs_mcp_arguments.cjs";

const { createMCPServer } = safeOutputsMCPServerHTTP;

describe("safe_outputs_mcp wrapped tool arguments", () => {
  let tempDir;

  beforeEach(() => {
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "safe-outputs-mcp-wrapped-"));
  });

  afterEach(() => {
    if (tempDir && fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
    delete process.env.GH_AW_SAFE_OUTPUTS_CONFIG_PATH;
    delete process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH;
    delete process.env.GH_AW_SAFE_OUTPUTS;
  });

  it("logs when wrapped arguments are unwrapped", () => {
    const debug = vi.fn();
    const payloadKeys = ["create_discussion"];

    const normalized = normalizeSafeOutputToolArguments(
      "create_discussion",
      {
        create_discussion: {
          title: "Wrapped title",
          body: "Wrapped body",
        },
      },
      { debug }
    );

    expect(normalized).toEqual({
      title: "Wrapped title",
      body: "Wrapped body",
    });
    expect(debug).toHaveBeenCalledWith(expect.stringContaining("Recovered wrapped safe-output tool arguments for 'create_discussion'"));
    expect(debug).toHaveBeenCalledWith(expect.stringContaining("unwrapping key 'create_discussion'"));
    expect(debug).toHaveBeenCalledWith(expect.stringContaining(JSON.stringify(payloadKeys)));
  });

  it("unwraps child arguments that match the tool name", async () => {
    const configPath = path.join(tempDir, "config.json");
    const toolsPath = path.join(tempDir, "tools.json");
    const outputPath = path.join(tempDir, "output.jsonl");

    fs.writeFileSync(configPath, JSON.stringify({ create_discussion: { enabled: true } }));
    fs.writeFileSync(
      toolsPath,
      JSON.stringify([
        {
          name: "create_discussion",
          description: "Create a discussion",
          inputSchema: {
            type: "object",
            properties: {
              title: { type: "string" },
              body: { type: "string" },
            },
            required: ["title", "body"],
          },
        },
      ])
    );

    process.env.GH_AW_SAFE_OUTPUTS_CONFIG_PATH = configPath;
    process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH = toolsPath;
    process.env.GH_AW_SAFE_OUTPUTS = outputPath;

    const { server } = createMCPServer();
    const response = await server.handleRequest({
      jsonrpc: "2.0",
      id: 1,
      method: "tools/call",
      params: {
        name: "create_discussion",
        arguments: {
          create_discussion: {
            title: "Wrapped title",
            body: "Wrapped body",
          },
        },
      },
    });

    expect(response.error).toBeUndefined();
    expect(response.result.isError).toBe(false);

    const written = fs.readFileSync(outputPath, "utf8");
    expect(JSON.parse(written.trim())).toMatchObject({
      type: "create_discussion",
      title: "Wrapped title",
      body: "Wrapped body",
    });
  });
});
