// @ts-check
import { afterEach, describe, expect, it, vi } from "vitest";
import { createRequire } from "module";

const req = createRequire(import.meta.url);
const { gatewayConversionProfiles, transformClaudeEntry, transformCopilotEntry, transformGeminiEntry } = req("./convert_gateway_config_profiles.cjs");

describe("convert_gateway_config_profiles", () => {
  afterEach(() => {
    vi.unstubAllEnvs();
  });

  it("declares all supported gateway converter engines in one profile table", () => {
    expect(Object.keys(gatewayConversionProfiles).sort()).toEqual(["claude", "codex", "copilot", "gemini"]);
  });

  it("keeps engine-specific conversion knobs auditable in profiles", () => {
    expect(gatewayConversionProfiles.copilot).toMatchObject({ format: "Copilot", engine: "Copilot", setFailedOnError: true });
    expect(gatewayConversionProfiles.copilot.preRunOutputPath).toBeTypeOf("function");
    expect(gatewayConversionProfiles.codex).toMatchObject({ format: "Codex TOML", engine: "Codex", preRunOutputPath: expect.any(Function) });
    expect(gatewayConversionProfiles.claude).toMatchObject({ format: "Claude", engine: "Claude", preRunOutputPath: expect.any(Function) });
    expect(gatewayConversionProfiles.gemini).toMatchObject({ format: "Gemini", engine: "Gemini", contextOptions: { extraRequiredEnv: ["GITHUB_WORKSPACE"] } });
  });

  it("resolves runner temp output paths at conversion time", () => {
    vi.stubEnv("RUNNER_TEMP", "/tmp/current-runner-temp");

    expect(gatewayConversionProfiles.codex.preRunOutputPath()).toBe("/tmp/current-runner-temp/gh-aw/mcp-config/config.toml");
    expect(gatewayConversionProfiles.claude.preRunOutputPath()).toBe("/tmp/current-runner-temp/gh-aw/mcp-config/mcp-servers.json");
  });

  it("applies Copilot entry defaults while rewriting the gateway URL", () => {
    expect(transformCopilotEntry({ url: "http://gateway/mcp/github" }, "http://host:80")).toEqual({
      url: "http://host:80/mcp/github",
      tools: ["*"],
    });
  });

  it("converts Claude entries to HTTP without tool restrictions", () => {
    expect(transformClaudeEntry({ url: "http://gateway/mcp/github", tools: ["*"] }, "http://host:80")).toEqual({
      url: "http://host:80/mcp/github",
      type: "http",
    });
  });

  it("removes Gemini entry types while rewriting the gateway URL", () => {
    expect(transformGeminiEntry({ url: "http://gateway/mcp/github", type: "http" }, "http://host:80")).toEqual({
      url: "http://host:80/mcp/github",
    });
  });
});
