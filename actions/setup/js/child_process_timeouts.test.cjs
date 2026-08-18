import { describe, expect, it } from "vitest";
import { createRequire } from "module";

const req = createRequire(import.meta.url);
const { MAX_SETUP_TIMEOUT_MS, SETUP_TIMEOUTS, getPositiveEnvIntOrDefault, getSetupTimeoutMs } = req("./child_process_timeouts.cjs");

describe("child process timeout helpers", () => {
  it("uses defaults when overrides are unset or invalid", () => {
    expect(getPositiveEnvIntOrDefault("MISSING_TIMEOUT", 123, {})).toBe(123);
    expect(getPositiveEnvIntOrDefault("BAD_TIMEOUT", 123, { BAD_TIMEOUT: "0" })).toBe(123);
    expect(getPositiveEnvIntOrDefault("BAD_TIMEOUT", 123, { BAD_TIMEOUT: "-1" })).toBe(123);
    expect(getPositiveEnvIntOrDefault("BAD_TIMEOUT", 123, { BAD_TIMEOUT: "10s" })).toBe(123);
  });

  it("accepts positive integer millisecond overrides", () => {
    expect(getPositiveEnvIntOrDefault("CUSTOM_TIMEOUT", 123, { CUSTOM_TIMEOUT: " 456 " })).toBe(456);
  });

  it("clamps overrides above Node's maximum timer delay", () => {
    expect(MAX_SETUP_TIMEOUT_MS).toBe(2_147_483_647);
    expect(getPositiveEnvIntOrDefault("BIG_TIMEOUT", 123, { BIG_TIMEOUT: String(MAX_SETUP_TIMEOUT_MS) })).toBe(MAX_SETUP_TIMEOUT_MS);
    expect(getPositiveEnvIntOrDefault("BIG_TIMEOUT", 123, { BIG_TIMEOUT: String(MAX_SETUP_TIMEOUT_MS + 1) })).toBe(MAX_SETUP_TIMEOUT_MS);
    expect(getPositiveEnvIntOrDefault("BIG_TIMEOUT", 123, { BIG_TIMEOUT: "99999999999999999999" })).toBe(123);
  });

  it("documents an environment override for every setup timeout", () => {
    for (const [name, timeout] of Object.entries(SETUP_TIMEOUTS)) {
      expect(name).toBeTruthy();
      expect(timeout.env).toMatch(/^GH_AW_[A-Z0-9_]+_TIMEOUT_MS$/);
      expect(timeout.defaultMs).toBeGreaterThan(0);
      expect(timeout.defaultMs).toBeLessThanOrEqual(MAX_SETUP_TIMEOUT_MS);
      expect(getSetupTimeoutMs(name, { [timeout.env]: "9876" })).toBe(9876);
      expect(getSetupTimeoutMs(name, { [timeout.env]: "9999999999" })).toBe(MAX_SETUP_TIMEOUT_MS);
    }
  });
});
