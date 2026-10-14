import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const { escapeWorkflowCommandValue, maskSecret, getOTLPSecretValues } = require("./actions_secret_masking.cjs");
const { getAuthorizationSecretValues } = require("./otlp_header_helpers.cjs");

describe("actions_secret_masking.cjs", () => {
  let originalCore;
  let originalGithubActions;

  beforeEach(() => {
    originalCore = global.core;
    originalGithubActions = process.env.GITHUB_ACTIONS;
  });

  afterEach(() => {
    if (originalCore === undefined) {
      delete global.core;
    } else {
      global.core = originalCore;
    }
    if (originalGithubActions === undefined) {
      delete process.env.GITHUB_ACTIONS;
    } else {
      process.env.GITHUB_ACTIONS = originalGithubActions;
    }
    vi.restoreAllMocks();
    vi.unstubAllEnvs();
  });

  it("uses core.setSecret when a real implementation is available", () => {
    const setSecret = vi.fn();
    global.core = { setSecret };

    maskSecret("derived-secret");

    expect(setSecret).toHaveBeenCalledWith("derived-secret");
  });

  it("retains the real core receiver and masks falsy scalar secrets without using the shim marker", () => {
    const receivers = [];
    const setSecret = vi.fn(function () {
      receivers.push(this);
    });
    Object.defineProperty(setSecret, "__ghAwUnavailable", { value: false });
    global.core = { setSecret };
    maskSecret(0);
    maskSecret(false);
    for (const value of [null, undefined, ""]) maskSecret(value);
    expect(setSecret.mock.calls).toEqual([["0"], ["false"]]);
    expect(receivers).toEqual([global.core, global.core]);
  });

  it("emits an add-mask workflow command in GitHub Actions when core.setSecret is unavailable", () => {
    process.env.GITHUB_ACTIONS = "true";
    global.core = {};
    const write = vi.spyOn(process.stdout, "write").mockImplementation(() => true);

    maskSecret("line1\nline2%tail");

    expect(write).toHaveBeenCalledWith("::add-mask::line1%0Aline2%25tail\n");
  });

  it("uses add-mask instead of the shim setSecret placeholder in GitHub Actions", () => {
    process.env.GITHUB_ACTIONS = "true";
    const setSecret = vi.fn(() => {
      throw new Error("shim placeholder");
    });
    Object.defineProperty(setSecret, "__ghAwUnavailable", { value: true });
    global.core = { setSecret };
    const write = vi.spyOn(process.stdout, "write").mockImplementation(() => true);

    maskSecret("derived-secret");

    expect(setSecret).not.toHaveBeenCalled();
    expect(write).toHaveBeenCalledWith("::add-mask::derived-secret\n");
  });

  it("does nothing outside GitHub Actions when core.setSecret is unavailable", () => {
    delete process.env.GITHUB_ACTIONS;
    const write = vi.spyOn(process.stdout, "write").mockImplementation(() => true);

    maskSecret("derived-secret");

    expect(write).not.toHaveBeenCalled();
  });

  it("escapes workflow command payloads", () => {
    expect(escapeWorkflowCommandValue("a%b\rc\n")).toBe("a%25b%0Dc%0A");
  });

  it("collects Basic credentials in both encoded and decoded forms without collecting the username", () => {
    const password = "synthetic-grafana-password";
    const decoded = `123456:${password}`;
    const encoded = Buffer.from(decoded).toString("base64");

    expect(getAuthorizationSecretValues(`Basic ${encoded}`)).toEqual([`Basic ${encoded}`, encoded, decoded, password]);
  });

  it("collects bare Bearer tokens case-insensitively", () => {
    expect(getAuthorizationSecretValues("bearer synthetic-bearer-token")).toEqual(["bearer synthetic-bearer-token", "synthetic-bearer-token"]);
  });

  it.each(["", "1", "ab", "abc"])("does not collect short header value %j", value => {
    expect(getAuthorizationSecretValues(value)).toEqual([]);
  });

  it("collects a header value at the four-character minimum", () => {
    expect(getAuthorizationSecretValues("abcd")).toEqual(["abcd"]);
  });

  it.each(["a", "ab", "abc", "abcd"])("guards bare credentials independently for %j", credential => {
    const value = `Bearer ${credential}`;

    expect(getAuthorizationSecretValues(value)).toEqual(credential.length < 4 ? [value] : [value, credential]);
  });

  it.each(["", "a", "ab", "abc", "abcd"])("guards decoded Basic-auth passwords independently for %j", password => {
    const decoded = `user:${password}`;
    const encoded = Buffer.from(decoded).toString("base64");
    const value = `Basic ${encoded}`;

    expect(getAuthorizationSecretValues(value)).toEqual(password.length < 4 ? [value, encoded, decoded] : [value, encoded, decoded, password]);
  });

  it.each([":a", "u:a", "u:ab"])("guards decoded Basic-auth pairs independently for %j", decoded => {
    const encoded = Buffer.from(decoded).toString("base64");
    const value = `Basic ${encoded}`;

    expect(getAuthorizationSecretValues(value)).toEqual(decoded.length < 4 ? [value, encoded] : [value, encoded, decoded]);
  });

  it.each(["Basic not-base64!", "Basic bm8tY29sb24=", "custom-header-value"])("collects %s without inventing decoded credentials", value => {
    expect(getAuthorizationSecretValues(value)).toEqual(value.startsWith("Basic ") ? [value, value.slice(6)] : [value]);
  });

  describe("getOTLPSecretValues", () => {
    beforeEach(() => {
      vi.stubEnv("OTEL_EXPORTER_OTLP_HEADERS", "");
      vi.stubEnv("GH_AW_OTLP_ALL_HEADERS", "");
      vi.stubEnv("GH_AW_OTLP_ENDPOINTS", "");
    });

    it("warns about malformed endpoint JSON without throwing or discarding other headers", () => {
      const warning = vi.fn();
      global.core = { warning };
      vi.stubEnv("GH_AW_OTLP_ENDPOINTS", "{");
      vi.stubEnv("OTEL_EXPORTER_OTLP_HEADERS", "x-api-key=synthetic-fallback-secret");

      expect(getOTLPSecretValues()).toEqual(["synthetic-fallback-secret"]);
      expect(warning).toHaveBeenCalledExactlyOnceWith("Unable to collect OTLP endpoint credentials: invalid endpoint JSON");
    });

    it("tolerates malformed endpoint JSON without a warning implementation", () => {
      vi.stubEnv("GH_AW_OTLP_ENDPOINTS", "{");
      delete global.core;

      expect(getOTLPSecretValues()).toEqual([]);
    });

    it("skips endpoint entries with non-string headers and keeps string headers", () => {
      vi.stubEnv(
        "GH_AW_OTLP_ENDPOINTS",
        JSON.stringify([
          null,
          {},
          { headers: null },
          { headers: 123456 },
          { headers: false },
          { headers: ["x-api-key=synthetic-skipped-array"] },
          { headers: { "x-api-key": "synthetic-skipped-object" } },
          { headers: "x-api-key=synthetic-valid-endpoint" },
        ])
      );

      expect(getOTLPSecretValues()).toEqual(["synthetic-valid-endpoint"]);
    });

    it("includes the JSON-escaped form of secrets containing backslashes and quotes", () => {
      const password = 'synthetic-password\\with"quote';
      const encoded = Buffer.from(`user:${password}`).toString("base64");
      vi.stubEnv("OTEL_EXPORTER_OTLP_HEADERS", `Authorization=${encodeURIComponent(`Basic ${encoded}`)}`);

      const values = getOTLPSecretValues();

      expect(values).toContain(password);
      expect(values).toContain(JSON.stringify(password).slice(1, -1));
      expect(values).toContain(JSON.stringify(`user:${password}`).slice(1, -1));
    });
  });
});
