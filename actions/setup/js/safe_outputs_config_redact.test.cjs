import { describe, expect, it } from "vitest";

import { isSensitiveConfigKey, redactSensitiveConfig } from "./safe_outputs_config_redact.cjs";

describe("safe_outputs_config_redact.cjs", () => {
  it.each(["token", "apiKey", "API_KEY", "Authorization", "headers", "password", "set-cookie", "privateKey", "client_secret"])("classifies %s as sensitive", key => {
    expect(isSensitiveConfigKey(key)).toBe(true);
  });

  it("redacts mixed-case sensitive keys in nested objects and arrays", () => {
    const config = {
      servers: [
        {
          HEADERS: { Authorization: "sentinel-authorization" },
          nested: { apiKey: "sentinel-api-key", password: "sentinel-password" },
        },
        { cookie: "sentinel-cookie", PRIVATE_KEY: "sentinel-private-key", safe: "visible" },
      ],
    };

    const redacted = redactSensitiveConfig(config);
    const serialized = JSON.stringify(redacted);

    expect(serialized).not.toContain("sentinel-");
    expect(redacted).toEqual({
      servers: [
        {
          HEADERS: "***REDACTED***",
          nested: { apiKey: "***REDACTED***", password: "***REDACTED***" },
        },
        { cookie: "***REDACTED***", PRIVATE_KEY: "***REDACTED***", safe: "visible" },
      ],
    });
  });
});
