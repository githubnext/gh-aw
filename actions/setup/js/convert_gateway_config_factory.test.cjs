// @ts-check
import { afterEach, describe, expect, it, vi } from "vitest";
import { createRequire } from "module";

const req = createRequire(import.meta.url);
const { buildGatewayConversionOptions, runGatewayProfile } = req("./convert_gateway_config_factory.cjs");

describe("convert_gateway_config_factory", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("builds runGatewayConversion options from a declarative entry transform profile", () => {
    const options = buildGatewayConversionOptions({
      format: "Test Format",
      engine: "Test Engine",
      outputPath: "/tmp/test-output.json",
      transformEntry: (entry, urlPrefix, _context, name) => ({ ...entry, name, urlPrefix }),
      serialize: servers => JSON.stringify({ mcpServers: servers }),
    });

    expect(options.format).toBe("Test Format");
    expect(options.engine).toBe("Test Engine");
    expect(options.outputPath).toBe("/tmp/test-output.json");
    expect(options.transformServer("github", { url: "http://old/mcp/github" }, "http://host:80", {})).toEqual({
      url: "http://old/mcp/github",
      name: "github",
      urlPrefix: "http://host:80",
    });
  });

  it("resolves output paths before running when a profile requests it", () => {
    const options = buildGatewayConversionOptions({
      format: "Test",
      engine: "Test",
      preRunOutputPath: () => "/tmp/resolved.json",
      transformEntry: entry => entry,
      serialize: servers => JSON.stringify({ mcpServers: servers }),
    });

    expect(options.outputPath).toBe("/tmp/resolved.json");
  });

  it("reports profile failures through core.setFailed when requested", () => {
    const originalCore = global.core;
    const setFailed = vi.fn();
    // @ts-ignore
    global.core = { setFailed };

    try {
      const result = runGatewayProfile({
        format: "Test",
        engine: "Test",
        preRunOutputPath: () => {
          throw new Error("missing output path");
        },
        transformEntry: entry => entry,
        serialize: servers => JSON.stringify({ mcpServers: servers }),
        setFailedOnError: true,
      });

      expect(result).toBeUndefined();
      expect(setFailed).toHaveBeenCalledWith("ERROR: missing output path");
    } finally {
      // @ts-ignore
      global.core = originalCore;
    }
  });
});
