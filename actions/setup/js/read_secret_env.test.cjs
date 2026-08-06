import { afterAll, afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createRequire } from "module";
import { spawnSync } from "child_process";
import { resolve, dirname } from "path";
import { fileURLToPath } from "url";

const req = createRequire(import.meta.url);
const __dirname = dirname(fileURLToPath(import.meta.url));
const originalCore = global.core;
const setSecret = vi.fn();
// Install the core mock before requiring read_secret_env.cjs. The standalone
// shim fallback is covered in a subprocess below so this module can keep using
// the mocked core captured at require time.
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

  it("masks a whitespace-only secret", () => {
    vi.stubEnv("TEST_SECRET", "   ");

    expect(readSecretEnv("TEST_SECRET")).toBe("   ");
    expect(setSecret).toHaveBeenCalledWith("   ");
  });

  it("uses the core shim when loaded by a standalone Node.js process", () => {
    const result = spawnSync(process.execPath, ["-e", 'const { readSecretEnv } = require("./read_secret_env.cjs"); process.stdout.write(readSecretEnv("TEST_SECRET"));'], {
      cwd: __dirname,
      encoding: "utf8",
      env: { ...process.env, TEST_SECRET: "standalone-secret" },
    });

    expect(result.status).toBe(0);
    expect(result.stdout).toBe("standalone-secret");
    expect(result.stderr).toBe("::add-mask::standalone-secret\n");
  });
});
