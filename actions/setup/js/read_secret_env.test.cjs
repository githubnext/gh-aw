import { afterAll, afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createRequire } from "module";

const req = createRequire(import.meta.url);
const originalCore = global.core;
const setSecret = vi.fn();
global.core = { setSecret };
const { readSecretEnv } = req("./read_secret_env.cjs");

describe("readSecretEnv", () => {
  afterAll(() => {
    global.core = originalCore;
  });

  beforeEach(() => {
    setSecret.mockClear();
  });

  afterEach(() => {
    vi.unstubAllEnvs();
  });

  it("returns and masks a configured secret", () => {
    vi.stubEnv("TEST_SECRET", "secret-value");

    expect(readSecretEnv("TEST_SECRET")).toBe("secret-value");
    expect(setSecret).toHaveBeenCalledWith("secret-value");
  });

  it("does not mask a missing secret", () => {
    vi.stubEnv("TEST_SECRET", undefined);

    expect(readSecretEnv("TEST_SECRET")).toBeUndefined();
    expect(setSecret).not.toHaveBeenCalled();
  });

  it("does not mask an empty secret", () => {
    vi.stubEnv("TEST_SECRET", "");

    expect(readSecretEnv("TEST_SECRET")).toBe("");
    expect(setSecret).not.toHaveBeenCalled();
  });
});
