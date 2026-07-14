// @ts-check

import { describe, it, expect } from "vitest";
const { resolveRetryConfig, parseRetryConfigNumber } = require("./harness_retry_config.cjs");

describe("parseRetryConfigNumber", () => {
  it("returns defaultValue when env var is not set", () => {
    const result = parseRetryConfigNumber({}, { envVar: "MY_VAR", defaultValue: 42, minimum: 0 });
    expect(result).toBe(42);
  });

  it("returns defaultValue when env var is empty string", () => {
    const result = parseRetryConfigNumber({ MY_VAR: "" }, { envVar: "MY_VAR", defaultValue: 42, minimum: 0 });
    expect(result).toBe(42);
  });

  it("parses a valid integer", () => {
    const result = parseRetryConfigNumber({ MY_VAR: "10" }, { envVar: "MY_VAR", defaultValue: 5, minimum: 0 });
    expect(result).toBe(10);
  });

  it("rejects exponential notation integers", () => {
    const result = parseRetryConfigNumber({ MY_VAR: "1e3" }, { envVar: "MY_VAR", defaultValue: 5, minimum: 0 });
    expect(result).toBe(5);
  });

  it("rejects hex notation", () => {
    const result = parseRetryConfigNumber({ MY_VAR: "0x10" }, { envVar: "MY_VAR", defaultValue: 5, minimum: 0 });
    expect(result).toBe(5);
  });

  it("returns defaultValue when value is below minimum", () => {
    const result = parseRetryConfigNumber({ MY_VAR: "0" }, { envVar: "MY_VAR", defaultValue: 5, minimum: 1 });
    expect(result).toBe(5);
  });

  it("accepts float when allowFloat is true", () => {
    const result = parseRetryConfigNumber({ MY_VAR: "1.5" }, { envVar: "MY_VAR", defaultValue: 2, minimum: 1, allowFloat: true });
    expect(result).toBe(1.5);
  });

  it("rejects float when allowFloat is false (default)", () => {
    const result = parseRetryConfigNumber({ MY_VAR: "1.5" }, { envVar: "MY_VAR", defaultValue: 2, minimum: 1 });
    expect(result).toBe(2);
  });

  it("calls logger on invalid value", () => {
    /** @type {string[]} */
    const logs = [];
    parseRetryConfigNumber({ MY_VAR: "bad" }, { envVar: "MY_VAR", defaultValue: 5, minimum: 0, logger: m => logs.push(m) });
    expect(logs.length).toBeGreaterThan(0);
    expect(logs[0]).toContain("MY_VAR");
  });

  it("trims whitespace from value", () => {
    const result = parseRetryConfigNumber({ MY_VAR: "  7  " }, { envVar: "MY_VAR", defaultValue: 5, minimum: 0 });
    expect(result).toBe(7);
  });
});

describe("resolveRetryConfig", () => {
  it("returns defaults when no env vars are set", () => {
    const config = resolveRetryConfig({});
    expect(config).toEqual({
      maxRetries: 3,
      initialDelayMs: 5000,
      backoffMultiplier: 2,
      maxDelayMs: 60000,
    });
  });

  it("reads maxRetries from env", () => {
    const config = resolveRetryConfig({ GH_AW_HARNESS_MAX_RETRIES: "5" });
    expect(config.maxRetries).toBe(5);
  });

  it("clamps maxRetries to MAX_RETRIES_CAP (100)", () => {
    /** @type {string[]} */
    const logs = [];
    const config = resolveRetryConfig({ GH_AW_HARNESS_MAX_RETRIES: "200" }, m => logs.push(m));
    expect(config.maxRetries).toBe(100);
    expect(logs.some(m => m.includes("clamping"))).toBe(true);
  });

  it("reads initialDelayMs from env", () => {
    const config = resolveRetryConfig({ GH_AW_HARNESS_INITIAL_DELAY_MS: "1000" });
    expect(config.initialDelayMs).toBe(1000);
  });

  it("reads backoffMultiplier as float from env", () => {
    const config = resolveRetryConfig({ GH_AW_HARNESS_BACKOFF_MULTIPLIER: "1.5" });
    expect(config.backoffMultiplier).toBe(1.5);
  });

  it("clamps maxDelayMs to initialDelayMs when lower", () => {
    /** @type {string[]} */
    const logs = [];
    const config = resolveRetryConfig({ GH_AW_HARNESS_INITIAL_DELAY_MS: "5000", GH_AW_HARNESS_MAX_DELAY_MS: "1000" }, m => logs.push(m));
    expect(config.maxDelayMs).toBe(5000);
    expect(logs.some(m => m.includes("clamping max delay"))).toBe(true);
  });

  it("reads maxDelayMs from env", () => {
    const config = resolveRetryConfig({ GH_AW_HARNESS_MAX_DELAY_MS: "30000" });
    expect(config.maxDelayMs).toBe(30000);
  });

  it("uses process.env when no env argument provided", () => {
    const config = resolveRetryConfig();
    expect(config).toHaveProperty("maxRetries");
    expect(config).toHaveProperty("initialDelayMs");
    expect(config).toHaveProperty("backoffMultiplier");
    expect(config).toHaveProperty("maxDelayMs");
  });
});
