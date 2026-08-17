// @ts-check
import { describe, expect, it } from "vitest";
import { createRequire } from "module";

const req = createRequire(import.meta.url);
const { gatewayConversionProfiles } = req("./convert_gateway_config_profiles.cjs");

describe("convert_gateway_config_profiles", () => {
  it("declares all supported gateway converter engines in one profile table", () => {
    expect(Object.keys(gatewayConversionProfiles).sort()).toEqual(["claude", "codex", "copilot", "gemini"]);
  });

  it("keeps engine-specific conversion knobs auditable in profiles", () => {
    expect(gatewayConversionProfiles.copilot).toMatchObject({ format: "Copilot", engine: "Copilot", setFailedOnError: true });
    expect(gatewayConversionProfiles.copilot.preRunOutputPath).toBeTypeOf("function");
    expect(gatewayConversionProfiles.codex).toMatchObject({ format: "Codex TOML", engine: "Codex" });
    expect(gatewayConversionProfiles.claude).toMatchObject({ format: "Claude", engine: "Claude" });
    expect(gatewayConversionProfiles.gemini).toMatchObject({ format: "Gemini", engine: "Gemini", contextOptions: { extraRequiredEnv: ["GITHUB_WORKSPACE"] } });
  });
});
