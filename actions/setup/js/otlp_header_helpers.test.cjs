import { afterEach, describe, expect, it, vi } from "vitest";
import fs, { readFileSync } from "fs";
import { resolve } from "path";
import { runInNewContext } from "vm";

const { parseOTLPHeaders, getOTLPSecretValues } = require("./otlp_header_helpers.cjs");

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
});

describe("otlp_header_helpers.cjs", () => {
  it("preserves the telemetry and runner masking exports", () => {
    expect(require("./send_otlp_span.cjs").parseOTLPHeaders).toBe(parseOTLPHeaders);
    expect(require("./actions_secret_masking.cjs").getOTLPSecretValues).toBe(getOTLPSecretValues);
  });

  it("collects credentials without runner globals, workflow commands, or runtime imports", () => {
    const password = 'synthetic-isolated-"password\\tail';
    const encoded = Buffer.from(`user:${password}`).toString("base64");
    const headerValue = `Basic ${encoded}`;
    const rawValue = encodeURIComponent(headerValue);
    const isolatedModule = { exports: {} };
    const requireRuntime = vi.fn(() => {
      throw new Error("Runtime imports must not be needed for header collection");
    });
    const write = vi.fn();
    runInNewContext(readFileSync(resolve(__dirname, "otlp_header_helpers.cjs"), "utf8"), {
      module: isolatedModule,
      require: requireRuntime,
      Buffer,
      global: {},
      process: {
        env: { GITHUB_ACTIONS: "true", OTEL_EXPORTER_OTLP_HEADERS: `Authorization=${rawValue}` },
        stdout: { write },
      },
    });

    const values = isolatedModule.exports.getOTLPSecretValues();

    for (const value of [password, JSON.stringify(password).slice(1, -1), encoded, headerValue, rawValue, `user:${password}`]) {
      expect(values).toContain(value);
    }
    expect(requireRuntime).not.toHaveBeenCalled();
    expect(write).not.toHaveBeenCalled();
  });

  it("retains duplicate headers and disabled endpoint credentials", () => {
    vi.stubEnv("OTEL_EXPORTER_OTLP_HEADERS", "x-api-key=synthetic-primary,x-api-key=synthetic-secondary");
    vi.stubEnv("GH_AW_OTLP_ALL_HEADERS", "x-api-key=synthetic-aggregate");
    vi.stubEnv("GH_AW_OTLP_ENDPOINTS", JSON.stringify([{ enabled: false, headers: "x-api-key=synthetic-disabled" }]));

    const values = getOTLPSecretValues();

    for (const value of ["synthetic-primary", "synthetic-secondary", "synthetic-aggregate", "synthetic-disabled"]) {
      expect(values).toContain(value);
    }
  });

  it("retains plain placeholders for artifact and step-summary redaction", () => {
    const { redactArtifactContent, redactStepSummaryContent } = require("./redact_secrets.cjs");
    const placeholders = ["true", "false", "null", "abcd", "abcdefg", "12345678", "%74%72%75%65", "%31%32%33%34%35%36%37%38"];
    vi.stubEnv("OTEL_EXPORTER_OTLP_HEADERS", placeholders.map((value, index) => `X-Placeholder-${index}=${value}`).join(","));
    vi.stubEnv("GH_AW_OTLP_ALL_HEADERS", "");
    vi.stubEnv("GH_AW_OTLP_ENDPOINTS", "");

    const values = getOTLPSecretValues();
    const content = JSON.stringify({ values: placeholders });
    const expected = JSON.stringify({ values: placeholders.map(() => "***") });

    expect(values.sort()).toEqual([...placeholders].sort());
    expect(redactArtifactContent(content, [], values).content).toBe(expected);
    expect(redactStepSummaryContent(content)).toBe(expected);
  });

  it("keeps eval verification rejecting unredacted OTLP credentials", () => {
    const { verifyRedaction } = require("./redact_evals_results.cjs");
    const { redactStepSummaryContent } = require("./redact_secrets.cjs");
    const password = "synthetic-eval-verification-password";
    const encoded = Buffer.from(`user:${password}`).toString("base64");
    vi.stubEnv("OTEL_EXPORTER_OTLP_HEADERS", `Authorization=${encodeURIComponent(`Basic ${encoded}`)}`);
    vi.stubEnv("GH_AW_SECRET_NAMES", "");
    const core = { setFailed: vi.fn(), warning: vi.fn() };
    vi.stubGlobal("core", core);
    let content = JSON.stringify({ question: password });
    vi.spyOn(fs, "existsSync").mockImplementation(file => String(file).endsWith("/evals.jsonl"));
    vi.spyOn(fs, "readFileSync").mockImplementation(file => {
      if (!String(file).endsWith("/evals.jsonl")) throw new Error("Unexpected fixture read");
      return content;
    });

    verifyRedaction();

    expect(core.setFailed).toHaveBeenCalledOnce();
    expect(core.setFailed.mock.calls.flat().join("\n")).not.toContain(password);
    core.setFailed.mockClear();
    content = redactStepSummaryContent(content);

    verifyRedaction();

    expect(content).not.toContain(password);
    expect(core.setFailed).not.toHaveBeenCalled();
  });
});
