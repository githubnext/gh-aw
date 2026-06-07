import { describe, it, expect } from "vitest";
import { createRequire } from "module";

const require = createRequire(import.meta.url);
const { buildSessionConfig } = require("../../../.github/drivers/copilot_sdk_driver_sample_node.cjs");

describe("copilot_sdk_driver_sample_node", () => {
  it("includes provider config when GH_AW_COPILOT_SDK_PROVIDER_BASE_URL is set", () => {
    process.env.GH_AW_COPILOT_SDK_PROVIDER_BASE_URL = "http://api-proxy:10002";
    const onPermissionRequest = () => {};
    const config = buildSessionConfig("test-model", onPermissionRequest);
    expect(config.provider).toEqual({ type: "openai", baseUrl: "http://api-proxy:10002" });
    expect(config.model).toBe("test-model");
    expect(config.onPermissionRequest).toBe(onPermissionRequest);
    delete process.env.GH_AW_COPILOT_SDK_PROVIDER_BASE_URL;
  });

  it("omits provider config when GH_AW_COPILOT_SDK_PROVIDER_BASE_URL is unset", () => {
    delete process.env.GH_AW_COPILOT_SDK_PROVIDER_BASE_URL;
    const onPermissionRequest = () => {};
    const config = buildSessionConfig("test-model", onPermissionRequest);
    expect(config.provider).toBeUndefined();
    expect(config.model).toBe("test-model");
    expect(config.onPermissionRequest).toBe(onPermissionRequest);
  });
});
