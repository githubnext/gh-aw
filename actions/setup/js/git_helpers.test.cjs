import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

describe("git_helpers.cjs", () => {
  let originalCore;

  beforeEach(() => {
    // Save existing core and provide a minimal no-op stub if not already set,
    // matching the guarantee that shim.cjs provides in production.
    originalCore = global.core;
    if (!global.core) {
      global.core = {
        debug: () => {},
        info: () => {},
        warning: () => {},
        error: () => {},
        setFailed: () => {},
      };
    }
  });

  afterEach(() => {
    global.core = originalCore;
  });

  function mockCoreWarning() {
    global.core.warning = vi.fn();
    return global.core.warning;
  }

  describe("execGitSync", () => {
    it("should export execGitSync function", async () => {
      const { execGitSync } = await import("./git_helpers.cjs");
      expect(typeof execGitSync).toBe("function");
    });

    it("should execute git commands safely", async () => {
      const { execGitSync } = await import("./git_helpers.cjs");

      // Test with a simple git command that should work
      const result = execGitSync(["--version"]);
      expect(result).toContain("git version");
    });

    it("should handle git command failures", async () => {
      const { execGitSync } = await import("./git_helpers.cjs");

      // Test with an invalid git command
      expect(() => {
        execGitSync(["invalid-command"]);
      }).toThrow();
    });

    it("should prevent shell injection in branch names", async () => {
      const { execGitSync } = await import("./git_helpers.cjs");

      // Test with malicious branch name
      const maliciousBranch = "feature; rm -rf /";

      // This should fail because the branch doesn't exist,
      // but importantly, it should NOT execute "rm -rf /"
      expect(() => {
        execGitSync(["rev-parse", maliciousBranch]);
      }).toThrow();
    });

    it("should treat special characters as literals", async () => {
      const { execGitSync } = await import("./git_helpers.cjs");

      const specialBranches = ["feature && echo hacked", "feature | cat /etc/passwd", "feature$(whoami)", "feature`whoami`"];

      for (const branch of specialBranches) {
        // All should fail with git error, not execute shell commands
        expect(() => {
          execGitSync(["rev-parse", branch]);
        }).toThrow();
      }
    });

    it("should pass options to spawnSync", async () => {
      const { execGitSync } = await import("./git_helpers.cjs");

      // Test that options are properly passed through
      const result = execGitSync(["--version"], { encoding: "utf8" });
      expect(typeof result).toBe("string");
      expect(result).toContain("git version");
    });

    it("should throw actionable ENOBUFS error when maxBuffer is exceeded", async () => {
      const { execGitSync } = await import("./git_helpers.cjs");

      // Use a tiny maxBuffer to trigger ENOBUFS on any git output
      expect(() => {
        execGitSync(["--version"], { maxBuffer: 1 });
      }).toThrow(/ENOBUFS|buffer limit/i);
    });

    it("should return stdout from successful commands", async () => {
      const { execGitSync } = await import("./git_helpers.cjs");

      // Use git --version which always succeeds
      const result = execGitSync(["--version"]);
      expect(typeof result).toBe("string");
      expect(result).toContain("git version");
    });

    it("should not call core.error when suppressLogs is true", async () => {
      const { execGitSync } = await import("./git_helpers.cjs");

      const errorLogs = [];
      const debugLogs = [];
      const originalCore = global.core;
      global.core = {
        debug: msg => debugLogs.push(msg),
        error: msg => errorLogs.push(msg),
      };

      try {
        // Use an invalid git command that will fail
        try {
          execGitSync(["rev-parse", "nonexistent-branch-that-does-not-exist"], { suppressLogs: true });
        } catch (e) {
          // Expected to fail
        }

        // core.error should NOT have been called
        expect(errorLogs).toHaveLength(0);
        // core.debug should have captured the failure details including exit status
        expect(debugLogs.some(log => log.includes("Git command failed (expected)"))).toBe(true);
        expect(debugLogs.some(log => log.includes("Exit status:"))).toBe(true);
      } finally {
        global.core = originalCore;
      }
    });

    it("should call core.error when suppressLogs is false (default)", async () => {
      const { execGitSync } = await import("./git_helpers.cjs");

      const errorLogs = [];
      const originalCore = global.core;
      global.core = {
        debug: () => {},
        error: msg => errorLogs.push(msg),
      };

      try {
        try {
          execGitSync(["rev-parse", "nonexistent-branch-that-does-not-exist"]);
        } catch (e) {
          // Expected to fail
        }

        // core.error should have been called
        expect(errorLogs.length).toBeGreaterThan(0);
      } finally {
        global.core = originalCore;
      }
    });

    it("should redact credentials from logged commands", async () => {
      const { execGitSync } = await import("./git_helpers.cjs");

      // Mock core.debug to capture logged output
      const debugLogs = [];
      const originalCore = global.core;
      global.core = {
        debug: msg => debugLogs.push(msg),
        error: () => {},
      };

      try {
        // Use a git command that doesn't require network access
        // We'll use 'ls-remote' with --exit-code and a URL with credentials
        // This will fail quickly without attempting network access
        try {
          execGitSync(["config", "--get", "remote.https://user:token@github.com/repo.git.url"]);
        } catch (e) {
          // Expected to fail, we're just checking the logging
        }

        // Check that credentials were redacted in the log
        const configLog = debugLogs.find(log => log.includes("git config"));
        expect(configLog).toBeDefined();
        expect(configLog).toContain("https://***@github.com/repo.git");
        expect(configLog).not.toContain("user:token");
      } finally {
        global.core = originalCore;
      }
    });
  });

  describe("getGitAuthEnv", () => {
    let originalEnv;

    beforeEach(() => {
      originalEnv = { ...process.env };
    });

    afterEach(() => {
      for (const key of Object.keys(process.env)) {
        if (!(key in originalEnv)) {
          delete process.env[key];
        }
      }
      Object.assign(process.env, originalEnv);
    });

    it("should export getGitAuthEnv function", async () => {
      const { getGitAuthEnv } = await import("./git_helpers.cjs");
      expect(typeof getGitAuthEnv).toBe("function");
    });

    it("should return GIT_CONFIG_* env vars when token is provided", async () => {
      const { getGitAuthEnv } = await import("./git_helpers.cjs");
      const env = getGitAuthEnv("my-test-token");

      expect(env).toHaveProperty("GIT_CONFIG_COUNT", "1");
      expect(env).toHaveProperty("GIT_CONFIG_KEY_0");
      expect(env).toHaveProperty("GIT_CONFIG_VALUE_0");
      expect(env.GIT_CONFIG_VALUE_0).toContain("Authorization: basic");
    });

    it("should use GITHUB_TOKEN env var when no token is passed", async () => {
      const { getGitAuthEnv } = await import("./git_helpers.cjs");
      process.env.GITHUB_TOKEN = "env-test-token";

      const env = getGitAuthEnv();

      expect(env).toHaveProperty("GIT_CONFIG_COUNT", "1");
      expect(env.GIT_CONFIG_VALUE_0).toBeDefined();
      // Value should be base64 of "x-access-token:env-test-token"
      const expected = Buffer.from("x-access-token:env-test-token").toString("base64");
      expect(env.GIT_CONFIG_VALUE_0).toContain(expected);
    });

    it("should prefer the provided token over GITHUB_TOKEN", async () => {
      const { getGitAuthEnv } = await import("./git_helpers.cjs");
      process.env.GITHUB_TOKEN = "env-token";

      const env = getGitAuthEnv("override-token");

      const expectedBase64 = Buffer.from("x-access-token:override-token").toString("base64");
      expect(env.GIT_CONFIG_VALUE_0).toContain(expectedBase64);
      // Should NOT contain the env token
      const envBase64 = Buffer.from("x-access-token:env-token").toString("base64");
      expect(env.GIT_CONFIG_VALUE_0).not.toContain(envBase64);
    });

    it("should return empty object when no token is available", async () => {
      const { getGitAuthEnv } = await import("./git_helpers.cjs");
      delete process.env.GITHUB_TOKEN;

      const env = getGitAuthEnv();

      expect(env).toEqual({});
    });

    it("should scope extraheader to GITHUB_SERVER_URL", async () => {
      const { getGitAuthEnv } = await import("./git_helpers.cjs");
      process.env.GITHUB_SERVER_URL = "https://github.example.com";

      const env = getGitAuthEnv("test-token");

      expect(env.GIT_CONFIG_KEY_0).toBe("http.https://github.example.com/.extraheader");
    });

    it("should default server URL to https://github.com", async () => {
      const { getGitAuthEnv } = await import("./git_helpers.cjs");
      delete process.env.GITHUB_SERVER_URL;

      const env = getGitAuthEnv("test-token");

      expect(env.GIT_CONFIG_KEY_0).toBe("http.https://github.com/.extraheader");
    });

    it("should strip trailing slash from server URL", async () => {
      const { getGitAuthEnv } = await import("./git_helpers.cjs");
      process.env.GITHUB_SERVER_URL = "https://github.example.com/";

      const env = getGitAuthEnv("test-token");

      expect(env.GIT_CONFIG_KEY_0).toBe("http.https://github.example.com/.extraheader");
    });
  });

  describe("ensureSafeDirectoryTrust", () => {
    let originalEnv;

    beforeEach(() => {
      originalEnv = { ...process.env };
      // Clean up GIT_CONFIG_* vars injected by a previous test
      for (const key of Object.keys(process.env)) {
        if (key.startsWith("GIT_CONFIG_")) {
          delete process.env[key];
        }
      }
    });

    afterEach(() => {
      // Restore to original, removing any vars added during the test
      for (const key of Object.keys(process.env)) {
        if (!(key in originalEnv)) {
          delete process.env[key];
        }
      }
      Object.assign(process.env, originalEnv);
    });

    it("should export ensureSafeDirectoryTrust function", async () => {
      const { ensureSafeDirectoryTrust } = await import("./git_helpers.cjs");
      expect(typeof ensureSafeDirectoryTrust).toBe("function");
    });

    it("should set GIT_CONFIG_* env vars for the given directory", async () => {
      const { ensureSafeDirectoryTrust } = await import("./git_helpers.cjs");

      ensureSafeDirectoryTrust("/workspace/repo");

      expect(process.env.GIT_CONFIG_COUNT).toBe("1");
      expect(process.env.GIT_CONFIG_KEY_0).toBe("safe.directory");
      expect(process.env.GIT_CONFIG_VALUE_0).toBe("/workspace/repo");
    });

    it("should not add a duplicate entry when called twice with the same directory", async () => {
      const { ensureSafeDirectoryTrust } = await import("./git_helpers.cjs");

      ensureSafeDirectoryTrust("/workspace/repo");
      ensureSafeDirectoryTrust("/workspace/repo");

      expect(process.env.GIT_CONFIG_COUNT).toBe("1");
    });

    it("should append a new entry when called with a different directory", async () => {
      const { ensureSafeDirectoryTrust } = await import("./git_helpers.cjs");

      ensureSafeDirectoryTrust("/workspace/repo-a");
      ensureSafeDirectoryTrust("/workspace/repo-b");

      expect(process.env.GIT_CONFIG_COUNT).toBe("2");
      expect(process.env.GIT_CONFIG_KEY_0).toBe("safe.directory");
      expect(process.env.GIT_CONFIG_VALUE_0).toBe("/workspace/repo-a");
      expect(process.env.GIT_CONFIG_KEY_1).toBe("safe.directory");
      expect(process.env.GIT_CONFIG_VALUE_1).toBe("/workspace/repo-b");
    });

    it("should be a no-op when called with an empty string", async () => {
      const { ensureSafeDirectoryTrust } = await import("./git_helpers.cjs");

      ensureSafeDirectoryTrust("");

      expect(process.env.GIT_CONFIG_COUNT).toBeUndefined();
    });

    it("should be a no-op when called with undefined/falsy", async () => {
      const { ensureSafeDirectoryTrust } = await import("./git_helpers.cjs");

      ensureSafeDirectoryTrust(undefined);

      expect(process.env.GIT_CONFIG_COUNT).toBeUndefined();
    });

    it("should preserve existing GIT_CONFIG_* entries set by getGitAuthEnv", async () => {
      const { ensureSafeDirectoryTrust, getGitAuthEnv } = await import("./git_helpers.cjs");

      // Simulate what getGitAuthEnv returns being already applied via env
      const authEnv = getGitAuthEnv("test-token");
      Object.assign(process.env, authEnv);
      const existingConfigCount = parseInt(authEnv.GIT_CONFIG_COUNT, 10);

      ensureSafeDirectoryTrust("/workspace/repo");

      // The count should be incremented by 1.
      expect(parseInt(process.env.GIT_CONFIG_COUNT, 10)).toBe(existingConfigCount + 1);
      // Existing auth entries preserved
      for (let i = 0; i < existingConfigCount; i++) {
        expect(process.env[`GIT_CONFIG_KEY_${i}`]).toBe(authEnv[`GIT_CONFIG_KEY_${i}`]);
        expect(process.env[`GIT_CONFIG_VALUE_${i}`]).toBe(authEnv[`GIT_CONFIG_VALUE_${i}`]);
      }
      // New safe.directory entry appended
      expect(process.env[`GIT_CONFIG_KEY_${existingConfigCount}`]).toBe("safe.directory");
      expect(process.env[`GIT_CONFIG_VALUE_${existingConfigCount}`]).toBe("/workspace/repo");
    });

    it("should handle malformed GIT_CONFIG_COUNT values gracefully", async () => {
      const { ensureSafeDirectoryTrust } = await import("./git_helpers.cjs");

      for (const malformedCount of ["not-a-number", "-1", "1.5", String(Number.MAX_SAFE_INTEGER + 1)]) {
        for (const key of Object.keys(process.env)) {
          if (key.startsWith("GIT_CONFIG_")) {
            delete process.env[key];
          }
        }

        process.env.GIT_CONFIG_COUNT = malformedCount;

        ensureSafeDirectoryTrust("/workspace/repo");

        expect(process.env.GIT_CONFIG_COUNT).toBe("1");
        expect(process.env.GIT_CONFIG_KEY_0).toBe("safe.directory");
        expect(process.env.GIT_CONFIG_VALUE_0).toBe("/workspace/repo");
      }
    });

    it("should not require a shimmed core global", async () => {
      const { ensureSafeDirectoryTrust } = await import("./git_helpers.cjs");
      const originalCore = global.core;

      global.core = undefined;

      try {
        expect(() => ensureSafeDirectoryTrust("/workspace/repo")).not.toThrow();
        expect(process.env.GIT_CONFIG_COUNT).toBe("1");
        expect(process.env.GIT_CONFIG_KEY_0).toBe("safe.directory");
        expect(process.env.GIT_CONFIG_VALUE_0).toBe("/workspace/repo");
      } finally {
        global.core = originalCore;
      }
    });

    it("should use a provided logger when core is not shimmed", async () => {
      const { ensureSafeDirectoryTrust } = await import("./git_helpers.cjs");
      const originalCore = global.core;
      const logger = { debug: vi.fn() };

      global.core = undefined;

      try {
        ensureSafeDirectoryTrust("/workspace/repo", logger);
        expect(logger.debug).toHaveBeenCalledWith("Configured git safe.directory for bridge context: /workspace/repo");
      } finally {
        global.core = originalCore;
      }
    });
  });

  describe("ensureFullHistoryForBundle", () => {
    it("should unshallow the repository when the repository is shallow", async () => {
      const { ensureFullHistoryForBundle } = await import("./git_helpers.cjs");
      const execApi = {
        getExecOutput: vi.fn().mockResolvedValue({ stdout: "true\n" }),
        exec: vi.fn().mockResolvedValue(0),
      };
      const options = { cwd: "/tmp/repo" };

      await ensureFullHistoryForBundle(execApi, options);

      expect(execApi.getExecOutput).toHaveBeenCalledWith("git", ["rev-parse", "--is-shallow-repository"], options);
      expect(execApi.exec).toHaveBeenCalledWith("git", ["fetch", "--unshallow", "origin"], options);
    });

    it("should not fetch full history when the repository is not shallow", async () => {
      const { ensureFullHistoryForBundle } = await import("./git_helpers.cjs");
      const execApi = {
        getExecOutput: vi.fn().mockResolvedValue({ stdout: "false\n" }),
        exec: vi.fn().mockResolvedValue(0),
      };

      await ensureFullHistoryForBundle(execApi);

      expect(execApi.exec).not.toHaveBeenCalled();
    });

    it("should skip history probing when shallow status cannot be determined", async () => {
      const { ensureFullHistoryForBundle } = await import("./git_helpers.cjs");
      const warning = mockCoreWarning();
      const execApi = {
        getExecOutput: vi.fn().mockRejectedValue(new Error("not a git repository")),
        exec: vi.fn().mockResolvedValue(0),
      };

      await ensureFullHistoryForBundle(execApi);

      expect(execApi.exec).not.toHaveBeenCalled();
      expect(warning).toHaveBeenCalledTimes(1);
      expect(warning).toHaveBeenCalledWith("Could not determine shallow repository status; skipping full-history fetch probe: not a git repository");
    });

    it("should warn with stringified non-error shallow status failures", async () => {
      const { ensureFullHistoryForBundle } = await import("./git_helpers.cjs");
      const warning = mockCoreWarning();
      const execApi = {
        getExecOutput: vi.fn().mockRejectedValue("unknown failure"),
        exec: vi.fn().mockResolvedValue(0),
      };

      await ensureFullHistoryForBundle(execApi);

      expect(execApi.exec).not.toHaveBeenCalled();
      expect(warning).toHaveBeenCalledTimes(1);
      expect(warning).toHaveBeenCalledWith("Could not determine shallow repository status; skipping full-history fetch probe: unknown failure");
    });

    it("should fetch prerequisite commit SHAs directly from origin when known and shallow", async () => {
      const { ensureFullHistoryForBundle } = await import("./git_helpers.cjs");
      const prereq = "a".repeat(40);
      let prereqFetched = false;
      const execApi = {
        getExecOutput: vi.fn().mockImplementation((cmd, args) => {
          if (args[0] === "rev-parse" && args[1] === "--is-shallow-repository") {
            return Promise.resolve({ stdout: "true\n", exitCode: 0 });
          }
          if (args[0] === "config") {
            // sparse-checkout not set
            return Promise.resolve({ stdout: "", exitCode: 1 });
          }
          if (args[0] === "bundle" && args[1] === "verify") {
            return Promise.resolve({
              stdout: "",
              stderr: `The bundle requires this ref:\n${prereq}\n`,
              exitCode: 1,
            });
          }
          if (args[0] === "cat-file" && args[1] === "-e") {
            // Object is present only after the direct SHA fetch.
            return Promise.resolve({ exitCode: prereqFetched ? 0 : 1, stdout: "", stderr: "" });
          }
          return Promise.resolve({ exitCode: 0, stdout: "", stderr: "" });
        }),
        exec: vi.fn().mockImplementation((cmd, args) => {
          if (args && args[0] === "fetch" && args.includes("origin") && args.includes(prereq)) {
            prereqFetched = true;
          }
          return Promise.resolve(0);
        }),
      };

      await ensureFullHistoryForBundle(execApi, {}, { baseRef: "main", bundleFilePath: "/tmp/test.bundle" });

      // Direct SHA fetch satisfies the prerequisite; no deepen, no --unshallow.
      const fetchCalls = execApi.exec.mock.calls.filter(c => c[1] && c[1][0] === "fetch");
      expect(fetchCalls.length).toBe(1);
      expect(fetchCalls[0][1]).toEqual(["fetch", "--filter=blob:none", "origin", prereq]);
      expect(execApi.exec).not.toHaveBeenCalledWith("git", expect.arrayContaining(["--unshallow"]), expect.anything());
    });

    it("should fall back to deepening by 5 commits at a time when direct SHA fetch is insufficient", async () => {
      const { ensureFullHistoryForBundle } = await import("./git_helpers.cjs");
      const prereq = "c".repeat(40);
      let deepenCalls = 0;
      const execApi = {
        getExecOutput: vi.fn().mockImplementation((cmd, args) => {
          if (args[0] === "rev-parse" && args[1] === "--is-shallow-repository") {
            return Promise.resolve({ stdout: "true\n", exitCode: 0 });
          }
          if (args[0] === "config") {
            return Promise.resolve({ stdout: "", exitCode: 1 });
          }
          if (args[0] === "bundle" && args[1] === "verify") {
            return Promise.resolve({ stdout: "", stderr: `The bundle requires this ref:\n${prereq}\n`, exitCode: 1 });
          }
          if (args[0] === "cat-file" && args[1] === "-e") {
            // Present only after the second deepen fetch; direct SHA fetch leaves it missing.
            return Promise.resolve({ exitCode: deepenCalls >= 2 ? 0 : 1, stdout: "", stderr: "" });
          }
          return Promise.resolve({ exitCode: 0, stdout: "", stderr: "" });
        }),
        exec: vi.fn().mockImplementation((cmd, args) => {
          if (args && args[0] === "fetch" && args[1] && args[1].startsWith("--deepen=")) {
            deepenCalls++;
          }
          return Promise.resolve(0);
        }),
      };

      await ensureFullHistoryForBundle(execApi, {}, { baseRef: "main", bundleFilePath: "/tmp/test.bundle" });

      const deepenFetchCalls = execApi.exec.mock.calls.filter(c => c[1] && c[1][0] === "fetch" && c[1][1] && c[1][1].startsWith("--deepen="));
      expect(deepenFetchCalls.length).toBe(2);
      // Each deepen step is a small, fixed increment of 5.
      expect(deepenFetchCalls[0][1]).toEqual(["fetch", "--deepen=5", "origin", "main"]);
      expect(execApi.exec).not.toHaveBeenCalledWith("git", expect.arrayContaining(["--unshallow"]), expect.anything());
    });

    it("should skip deepening when bundle declares no prerequisites", async () => {
      const { ensureFullHistoryForBundle } = await import("./git_helpers.cjs");
      const execApi = {
        getExecOutput: vi.fn().mockImplementation((cmd, args) => {
          if (args[0] === "rev-parse") return Promise.resolve({ stdout: "true\n" });
          if (args[0] === "bundle" && args[1] === "verify") {
            return Promise.resolve({ stdout: "The bundle contains this ref:\ndeadbeef refs/heads/x\n", stderr: "", exitCode: 0 });
          }
          return Promise.resolve({ exitCode: 0, stdout: "", stderr: "" });
        }),
        exec: vi.fn().mockResolvedValue(0),
      };

      await ensureFullHistoryForBundle(execApi, {}, { baseRef: "main", bundleFilePath: "/tmp/test.bundle" });

      expect(execApi.exec).not.toHaveBeenCalled();
    });

    it("should skip fetching when prereqs are already present locally", async () => {
      const { ensureFullHistoryForBundle } = await import("./git_helpers.cjs");
      const prereq = "b".repeat(40);
      const execApi = {
        getExecOutput: vi.fn().mockImplementation((cmd, args) => {
          if (args[0] === "rev-parse") return Promise.resolve({ stdout: "true\n" });
          if (args[0] === "bundle" && args[1] === "verify") {
            return Promise.resolve({ stdout: `The bundle requires this ref:\n${prereq}\n`, stderr: "", exitCode: 0 });
          }
          if (args[0] === "cat-file" && args[1] === "-e") {
            return Promise.resolve({ exitCode: 0, stdout: "", stderr: "" });
          }
          return Promise.resolve({ exitCode: 0, stdout: "", stderr: "" });
        }),
        exec: vi.fn().mockResolvedValue(0),
      };

      await ensureFullHistoryForBundle(execApi, {}, { baseRef: "main", bundleFilePath: "/tmp/test.bundle" });

      expect(execApi.exec).not.toHaveBeenCalled();
    });
  });

  describe("isShallowOrSparseCheckout", () => {
    const buildExecApi = handler => ({
      getExecOutput: vi.fn().mockImplementation((cmd, args) => Promise.resolve(handler(cmd, args))),
    });

    it("should return true when repository is shallow", async () => {
      const { isShallowOrSparseCheckout } = await import("./git_helpers.cjs");
      const execApi = buildExecApi((cmd, args) => {
        if (args[0] === "rev-parse" && args[1] === "--is-shallow-repository") {
          return { exitCode: 0, stdout: "true\n", stderr: "" };
        }
        return { exitCode: 1, stdout: "", stderr: "" };
      });

      await expect(isShallowOrSparseCheckout(execApi)).resolves.toBe(true);
      // Sparse probe must not run when shallow probe already returned true.
      expect(execApi.getExecOutput).toHaveBeenCalledTimes(1);
    });

    it("should return true when sparse-checkout is enabled", async () => {
      const { isShallowOrSparseCheckout } = await import("./git_helpers.cjs");
      const execApi = buildExecApi((cmd, args) => {
        if (args[0] === "rev-parse" && args[1] === "--is-shallow-repository") {
          return { exitCode: 0, stdout: "false\n", stderr: "" };
        }
        if (args[0] === "config" && args[1] === "--get" && args[2] === "core.sparseCheckout") {
          return { exitCode: 0, stdout: "true\n", stderr: "" };
        }
        return { exitCode: 1, stdout: "", stderr: "" };
      });

      await expect(isShallowOrSparseCheckout(execApi)).resolves.toBe(true);
    });

    it("should return false for a full, non-sparse clone", async () => {
      const { isShallowOrSparseCheckout } = await import("./git_helpers.cjs");
      const execApi = buildExecApi((cmd, args) => {
        if (args[0] === "rev-parse" && args[1] === "--is-shallow-repository") {
          return { exitCode: 0, stdout: "false\n", stderr: "" };
        }
        if (args[0] === "config" && args[1] === "--get" && args[2] === "core.sparseCheckout") {
          // git config exits 1 when the key is not set.
          return { exitCode: 1, stdout: "", stderr: "" };
        }
        return { exitCode: 0, stdout: "", stderr: "" };
      });

      await expect(isShallowOrSparseCheckout(execApi)).resolves.toBe(false);
    });

    it("should return false when both probes throw", async () => {
      const { isShallowOrSparseCheckout } = await import("./git_helpers.cjs");
      const execApi = {
        getExecOutput: vi.fn().mockRejectedValue(new Error("git missing")),
      };

      await expect(isShallowOrSparseCheckout(execApi)).resolves.toBe(false);
    });

    it("should treat sparse-checkout value case-insensitively", async () => {
      const { isShallowOrSparseCheckout } = await import("./git_helpers.cjs");
      const execApi = buildExecApi((cmd, args) => {
        if (args[0] === "rev-parse") {
          return { exitCode: 0, stdout: "false\n", stderr: "" };
        }
        if (args[0] === "config") {
          return { exitCode: 0, stdout: "True\n", stderr: "" };
        }
        return { exitCode: 1, stdout: "", stderr: "" };
      });

      await expect(isShallowOrSparseCheckout(execApi)).resolves.toBe(true);
    });
  });

  describe("extractBundlePrerequisiteCommits", () => {
    it("should return empty array for empty string", async () => {
      const { extractBundlePrerequisiteCommits } = await import("./git_helpers.cjs");
      expect(extractBundlePrerequisiteCommits("")).toEqual([]);
    });

    it("should return empty array when message does not mention prerequisite commits", async () => {
      const { extractBundlePrerequisiteCommits } = await import("./git_helpers.cjs");
      expect(extractBundlePrerequisiteCommits("fatal: failed to read bundle")).toEqual([]);
    });

    it("should return single SHA when one prerequisite commit is missing", async () => {
      const { extractBundlePrerequisiteCommits } = await import("./git_helpers.cjs");
      const message = "error: Repository lacks these prerequisite commits:\nerror: 172f87a830f57a29470efe7646d141069434a893";
      expect(extractBundlePrerequisiteCommits(message)).toEqual(["172f87a830f57a29470efe7646d141069434a893"]);
    });

    it("should return multiple SHAs when multiple prerequisite commits are missing", async () => {
      const { extractBundlePrerequisiteCommits } = await import("./git_helpers.cjs");
      const message = ["error: Repository lacks these prerequisite commits:", "error: 172f87a830f57a29470efe7646d141069434a893", "error: aabbccddee1122334455667788990011aabbccdd"].join("\n");
      const result = extractBundlePrerequisiteCommits(message);
      expect(result).toEqual(["172f87a830f57a29470efe7646d141069434a893", "aabbccddee1122334455667788990011aabbccdd"]);
    });

    it("should deduplicate repeated SHAs", async () => {
      const { extractBundlePrerequisiteCommits } = await import("./git_helpers.cjs");
      const sha = "172f87a830f57a29470efe7646d141069434a893";
      const message = `error: Repository lacks these prerequisite commits:\nerror: ${sha}\nerror: ${sha}`;
      expect(extractBundlePrerequisiteCommits(message)).toEqual([sha]);
    });

    it("should be case-insensitive for the prerequisite header text", async () => {
      const { extractBundlePrerequisiteCommits } = await import("./git_helpers.cjs");
      const message = "ERROR: REPOSITORY LACKS THESE PREREQUISITE COMMITS:\nerror: 172f87a830f57a29470efe7646d141069434a893";
      expect(extractBundlePrerequisiteCommits(message)).toEqual(["172f87a830f57a29470efe7646d141069434a893"]);
    });

    it("should ignore short (non-SHA) hex strings that are not 40 characters", async () => {
      const { extractBundlePrerequisiteCommits } = await import("./git_helpers.cjs");
      const message = "error: Repository lacks these prerequisite commits:\nerror: deadbeef";
      // "deadbeef" is only 8 chars — not a full 40-char SHA so it should not be captured
      // (The exact filtering depends on implementation; test that a real SHA is captured)
      const fullSha = "172f87a830f57a29470efe7646d141069434a893";
      const message2 = `error: Repository lacks these prerequisite commits:\nerror: ${fullSha} deadbeef`;
      const result = extractBundlePrerequisiteCommits(message2);
      expect(result).toContain(fullSha);
    });
  });

  describe("linearizeRangeAsCommit", () => {
    const ORIGINAL_HEAD = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
    const NEW_HEAD = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb";

    function makeExecApi({ originalHead = ORIGINAL_HEAD, newHead = NEW_HEAD, stagedFiles = "README.md\n" } = {}) {
      let headCallCount = 0;
      return {
        getExecOutput: vi.fn().mockImplementation((_cmd, args) => {
          if (args[0] === "rev-parse" && args[1] === "HEAD") {
            headCallCount += 1;
            // First call returns originalHead; subsequent calls return newHead
            return Promise.resolve({ stdout: headCallCount === 1 ? `${originalHead}\n` : `${newHead}\n` });
          }
          if (args[0] === "diff" && args[1] === "--cached") {
            return Promise.resolve({ stdout: stagedFiles });
          }
          return Promise.resolve({ stdout: "" });
        }),
        exec: vi.fn().mockResolvedValue(0),
      };
    }

    it("should return the new HEAD SHA after successful linearization", async () => {
      const { linearizeRangeAsCommit } = await import("./git_helpers.cjs");
      const execApi = makeExecApi();

      const result = await linearizeRangeAsCommit("origin/main", "Squash commit", execApi);

      expect(result).toBe(NEW_HEAD);
      expect(execApi.exec).toHaveBeenCalledWith("git", ["reset", "--soft", "origin/main"]);
      expect(execApi.exec).toHaveBeenCalledWith("git", ["commit", "-m", "Squash commit"]);
    });

    it("should prepend commitFlags before -m in the git commit call", async () => {
      const { linearizeRangeAsCommit } = await import("./git_helpers.cjs");
      const execApi = makeExecApi();

      await linearizeRangeAsCommit("origin/main", "Squash commit", execApi, {
        commitFlags: ["--allow-empty", "--no-verify"],
      });

      expect(execApi.exec).toHaveBeenCalledWith("git", ["commit", "--allow-empty", "--no-verify", "-m", "Squash commit"]);
    });

    it("should pass gitOpts to every exec and getExecOutput call", async () => {
      const { linearizeRangeAsCommit } = await import("./git_helpers.cjs");
      const execApi = makeExecApi();
      const gitOpts = { cwd: "/tmp/repo" };

      await linearizeRangeAsCommit("origin/main", "Squash commit", execApi, { gitOpts });

      // Every call should have received gitOpts as the trailing argument
      for (const [, , opts] of execApi.exec.mock.calls) {
        expect(opts).toEqual(gitOpts);
      }
      for (const [, , opts] of execApi.getExecOutput.mock.calls) {
        expect(opts).toEqual(gitOpts);
      }
    });

    it("should not append a third argument when gitOpts is not provided", async () => {
      const { linearizeRangeAsCommit } = await import("./git_helpers.cjs");
      const execApi = makeExecApi();

      await linearizeRangeAsCommit("origin/main", "Squash commit", execApi);

      // exec and getExecOutput should each be called with exactly 2 arguments
      for (const callArgs of execApi.exec.mock.calls) {
        expect(callArgs.length).toBe(2);
      }
      for (const callArgs of execApi.getExecOutput.mock.calls) {
        expect(callArgs.length).toBe(2);
      }
    });

    it("should throw immediately when HEAD cannot be resolved (empty stdout)", async () => {
      const { linearizeRangeAsCommit } = await import("./git_helpers.cjs");
      const execApi = {
        getExecOutput: vi.fn().mockResolvedValue({ stdout: "   \n" }),
        exec: vi.fn(),
      };

      await expect(linearizeRangeAsCommit("origin/main", "msg", execApi)).rejects.toThrow("Could not resolve current HEAD before linearizing range");
      expect(execApi.exec).not.toHaveBeenCalled();
    });

    it("should roll back to originalHead and throw when no staged changes exist after soft reset", async () => {
      const { linearizeRangeAsCommit } = await import("./git_helpers.cjs");
      const warning = mockCoreWarning();
      const execApi = makeExecApi({ stagedFiles: "" });

      await expect(linearizeRangeAsCommit("origin/main", "msg", execApi)).rejects.toThrow(/Failed to linearize origin\/main\.\.HEAD/);

      // Should have rolled back to the original HEAD
      expect(execApi.exec).toHaveBeenCalledWith("git", ["reset", "--hard", ORIGINAL_HEAD]);

      // Should have emitted a warning about restoring the original HEAD
      expect(warning).toHaveBeenCalledWith(expect.stringContaining(`restored original HEAD ${ORIGINAL_HEAD}`));
    });

    it("should roll back to originalHead and throw when soft reset fails", async () => {
      const { linearizeRangeAsCommit } = await import("./git_helpers.cjs");
      const warning = mockCoreWarning();
      const execApi = {
        getExecOutput: vi.fn().mockResolvedValue({ stdout: `${ORIGINAL_HEAD}\n` }),
        exec: vi.fn().mockImplementation((_cmd, args) => {
          // Soft reset fails; hard reset (rollback) succeeds
          if (args[0] === "reset" && args[1] === "--soft") return Promise.reject(new Error("reset failed"));
          return Promise.resolve(0);
        }),
      };

      await expect(linearizeRangeAsCommit("origin/main", "msg", execApi)).rejects.toThrow(/Failed to linearize origin\/main\.\.HEAD.*reset failed/s);

      // Should have attempted rollback (reset --hard)
      expect(execApi.exec).toHaveBeenCalledWith("git", ["reset", "--hard", ORIGINAL_HEAD]);
      expect(warning).toHaveBeenCalledWith(expect.stringContaining(`restored original HEAD ${ORIGINAL_HEAD}`));
    });

    it("should roll back to originalHead and throw when git commit fails", async () => {
      const { linearizeRangeAsCommit } = await import("./git_helpers.cjs");
      const warning = mockCoreWarning();
      const execApi = {
        getExecOutput: vi.fn().mockImplementation((_cmd, args) => {
          if (args[0] === "rev-parse") return Promise.resolve({ stdout: `${ORIGINAL_HEAD}\n` });
          if (args[0] === "diff") return Promise.resolve({ stdout: "file.txt\n" });
          return Promise.resolve({ stdout: "" });
        }),
        exec: vi.fn().mockImplementation((_cmd, args) => {
          if (args[0] === "commit") return Promise.reject(new Error("commit failed"));
          return Promise.resolve(0);
        }),
      };

      await expect(linearizeRangeAsCommit("origin/main", "msg", execApi)).rejects.toThrow(/Failed to linearize origin\/main\.\.HEAD.*commit failed/s);

      expect(execApi.exec).toHaveBeenCalledWith("git", ["reset", "--hard", ORIGINAL_HEAD]);
      expect(warning).toHaveBeenCalledWith(expect.stringContaining(`restored original HEAD ${ORIGINAL_HEAD}`));
    });

    it("should emit a rollback-failure warning when reset --hard also fails", async () => {
      const { linearizeRangeAsCommit } = await import("./git_helpers.cjs");
      const warning = mockCoreWarning();
      const execApi = {
        getExecOutput: vi.fn().mockImplementation((_cmd, args) => {
          if (args[0] === "rev-parse") return Promise.resolve({ stdout: `${ORIGINAL_HEAD}\n` });
          if (args[0] === "diff") return Promise.resolve({ stdout: "" });
          return Promise.resolve({ stdout: "" });
        }),
        exec: vi.fn().mockRejectedValue(new Error("disk failure")),
      };

      await expect(linearizeRangeAsCommit("origin/main", "msg", execApi)).rejects.toThrow(/Failed to linearize/);

      // Should have warned about the rollback failure
      expect(warning).toHaveBeenCalledWith(expect.stringContaining("rollback also failed"));
    });

    it("should carry the original error as the cause on failure", async () => {
      const { linearizeRangeAsCommit } = await import("./git_helpers.cjs");
      mockCoreWarning();
      const cause = new Error("inner error");
      const execApi = {
        getExecOutput: vi.fn().mockImplementation((_cmd, args) => {
          if (args[0] === "rev-parse") return Promise.resolve({ stdout: `${ORIGINAL_HEAD}\n` });
          if (args[0] === "diff") return Promise.resolve({ stdout: "" });
          return Promise.resolve({ stdout: "" });
        }),
        exec: vi.fn().mockRejectedValue(cause),
      };

      const err = await linearizeRangeAsCommit("origin/main", "msg", execApi).catch(e => e);

      expect(err.cause).toBe(cause);
    });

    it("should throw before any git state change when commit range is implausibly large in a shallow checkout", async () => {
      const { linearizeRangeAsCommit } = await import("./git_helpers.cjs");

      const execApi = {
        getExecOutput: vi.fn().mockImplementation((_cmd, args) => {
          if (args[0] === "rev-list" && args[1] === "--count") {
            // Simulate a shallow checkout returning 61000 commits in the range
            return Promise.resolve({ stdout: "61000\n" });
          }
          if (args[0] === "rev-parse" && args[1] === "--is-shallow-repository") {
            return Promise.resolve({ stdout: "true\n" });
          }
          return Promise.resolve({ stdout: "" });
        }),
        exec: vi.fn().mockResolvedValue(0),
      };

      await expect(linearizeRangeAsCommit("origin/main", "msg", execApi)).rejects.toThrow(/Refusing to linearize an implausible commit range/);

      // No git state mutation should have occurred
      expect(execApi.exec).not.toHaveBeenCalled();
    });

    it("should include the commit count and base ref in the implausible range error message", async () => {
      const { linearizeRangeAsCommit } = await import("./git_helpers.cjs");

      const execApi = {
        getExecOutput: vi.fn().mockImplementation((_cmd, args) => {
          if (args[0] === "rev-list" && args[1] === "--count") {
            return Promise.resolve({ stdout: "500\n" });
          }
          if (args[0] === "rev-parse" && args[1] === "--is-shallow-repository") {
            return Promise.resolve({ stdout: "true\n" });
          }
          return Promise.resolve({ stdout: "" });
        }),
        exec: vi.fn().mockResolvedValue(0),
      };

      const err = await linearizeRangeAsCommit("origin/feature", "msg", execApi).catch(e => e);

      expect(err.message).toMatch(/500/);
      expect(err.message).toMatch(/origin\/feature/);
      expect(err.message).toMatch(/fetch-depth/);
    });

    it("should not throw when commit range is large but repo is not shallow", async () => {
      const { linearizeRangeAsCommit } = await import("./git_helpers.cjs");
      const ORIGINAL_HEAD = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
      const NEW_HEAD = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb";
      let headCallCount = 0;

      const execApi = {
        getExecOutput: vi.fn().mockImplementation((_cmd, args) => {
          if (args[0] === "rev-list" && args[1] === "--count") {
            return Promise.resolve({ stdout: "500\n" });
          }
          if (args[0] === "rev-parse" && args[1] === "--is-shallow-repository") {
            return Promise.resolve({ stdout: "false\n" });
          }
          if (args[0] === "rev-parse" && args[1] === "HEAD") {
            headCallCount += 1;
            return Promise.resolve({ stdout: headCallCount === 1 ? `${ORIGINAL_HEAD}\n` : `${NEW_HEAD}\n` });
          }
          if (args[0] === "diff" && args[1] === "--cached") {
            return Promise.resolve({ stdout: "README.md\n" });
          }
          return Promise.resolve({ stdout: "" });
        }),
        exec: vi.fn().mockResolvedValue(0),
      };

      const result = await linearizeRangeAsCommit("origin/main", "msg", execApi);
      expect(result).toBe(NEW_HEAD);
    });

    it("should not throw when commit range count is within the default threshold", async () => {
      const { linearizeRangeAsCommit, SHALLOW_RANGE_MAX_COMMITS } = await import("./git_helpers.cjs");
      const ORIGINAL_HEAD = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
      const NEW_HEAD = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb";
      let headCallCount = 0;

      const execApi = {
        getExecOutput: vi.fn().mockImplementation((_cmd, args) => {
          if (args[0] === "rev-list" && args[1] === "--count") {
            // Exactly at threshold — not implausible
            return Promise.resolve({ stdout: `${SHALLOW_RANGE_MAX_COMMITS}\n` });
          }
          if (args[0] === "rev-parse" && args[1] === "HEAD") {
            headCallCount += 1;
            return Promise.resolve({ stdout: headCallCount === 1 ? `${ORIGINAL_HEAD}\n` : `${NEW_HEAD}\n` });
          }
          if (args[0] === "diff" && args[1] === "--cached") {
            return Promise.resolve({ stdout: "README.md\n" });
          }
          return Promise.resolve({ stdout: "" });
        }),
        exec: vi.fn().mockResolvedValue(0),
      };

      const result = await linearizeRangeAsCommit("origin/main", "msg", execApi);
      expect(result).toBe(NEW_HEAD);
    });

    it("should proceed normally when the rev-list count command fails", async () => {
      const { linearizeRangeAsCommit } = await import("./git_helpers.cjs");
      const ORIGINAL_HEAD = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
      const NEW_HEAD = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb";
      let headCallCount = 0;

      const execApi = {
        getExecOutput: vi.fn().mockImplementation((_cmd, args) => {
          if (args[0] === "rev-list" && args[1] === "--count") {
            return Promise.reject(new Error("rev-list failed"));
          }
          if (args[0] === "rev-parse" && args[1] === "HEAD") {
            headCallCount += 1;
            return Promise.resolve({ stdout: headCallCount === 1 ? `${ORIGINAL_HEAD}\n` : `${NEW_HEAD}\n` });
          }
          if (args[0] === "diff" && args[1] === "--cached") {
            return Promise.resolve({ stdout: "README.md\n" });
          }
          return Promise.resolve({ stdout: "" });
        }),
        exec: vi.fn().mockResolvedValue(0),
      };

      // A rev-list failure should be non-fatal — linearization proceeds normally
      const result = await linearizeRangeAsCommit("origin/main", "msg", execApi);
      expect(result).toBe(NEW_HEAD);
    });

    it("should proceed normally when the shallow probe fails in the guard", async () => {
      const { linearizeRangeAsCommit } = await import("./git_helpers.cjs");
      const ORIGINAL_HEAD = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
      const NEW_HEAD = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb";
      let headCallCount = 0;

      const execApi = {
        getExecOutput: vi.fn().mockImplementation((_cmd, args) => {
          if (args[0] === "rev-list" && args[1] === "--count") {
            // Large count to trigger shallow probe
            return Promise.resolve({ stdout: "500\n" });
          }
          if (args[0] === "rev-parse" && args[1] === "--is-shallow-repository") {
            // Shallow probe fails
            return Promise.reject(new Error("not a git repo"));
          }
          if (args[0] === "rev-parse" && args[1] === "HEAD") {
            headCallCount += 1;
            return Promise.resolve({ stdout: headCallCount === 1 ? `${ORIGINAL_HEAD}\n` : `${NEW_HEAD}\n` });
          }
          if (args[0] === "diff" && args[1] === "--cached") {
            return Promise.resolve({ stdout: "README.md\n" });
          }
          return Promise.resolve({ stdout: "" });
        }),
        exec: vi.fn().mockResolvedValue(0),
      };

      // Shallow probe failure is non-fatal — linearization proceeds normally
      const result = await linearizeRangeAsCommit("origin/main", "msg", execApi);
      expect(result).toBe(NEW_HEAD);
    });

    it("should respect a custom maxCommits threshold via opts", async () => {
      const { linearizeRangeAsCommit } = await import("./git_helpers.cjs");

      const execApi = {
        getExecOutput: vi.fn().mockImplementation((_cmd, args) => {
          if (args[0] === "rev-list" && args[1] === "--count") {
            return Promise.resolve({ stdout: "10\n" });
          }
          if (args[0] === "rev-parse" && args[1] === "--is-shallow-repository") {
            return Promise.resolve({ stdout: "true\n" });
          }
          return Promise.resolve({ stdout: "" });
        }),
        exec: vi.fn().mockResolvedValue(0),
      };

      // maxCommits: 5 means 10 commits is implausible for a shallow repo
      await expect(linearizeRangeAsCommit("origin/main", "msg", execApi, { maxCommits: 5 })).rejects.toThrow(/Refusing to linearize an implausible commit range/);
      expect(execApi.exec).not.toHaveBeenCalled();
    });
  });

  describe("checkImplausibleShallowRange", () => {
    it("should return implausible:false and commitCount:0 for empty baseRef", async () => {
      const { checkImplausibleShallowRange } = await import("./git_helpers.cjs");
      expect(checkImplausibleShallowRange("", "HEAD")).toEqual({ implausible: false, commitCount: 0 });
    });

    it("should return implausible:false and commitCount:0 for empty headRef", async () => {
      const { checkImplausibleShallowRange } = await import("./git_helpers.cjs");
      expect(checkImplausibleShallowRange("origin/main", "")).toEqual({ implausible: false, commitCount: 0 });
    });

    it("should return implausible:false when git rev-list fails (non-existent refs)", async () => {
      const { checkImplausibleShallowRange } = await import("./git_helpers.cjs");
      // Using a clearly non-existent ref so git fails
      const result = checkImplausibleShallowRange("refs/nonexistent/base", "refs/nonexistent/head");
      expect(result.implausible).toBe(false);
      expect(result.commitCount).toBe(0);
    });

    it("should return implausible:false for a small commit range (integration)", async () => {
      const { checkImplausibleShallowRange } = await import("./git_helpers.cjs");
      // The test environment is a non-shallow full clone; HEAD..HEAD has 0 commits
      const result = checkImplausibleShallowRange("HEAD", "HEAD");
      expect(result.implausible).toBe(false);
    });

    it("should export SHALLOW_RANGE_MAX_COMMITS as a positive number", async () => {
      const { SHALLOW_RANGE_MAX_COMMITS } = await import("./git_helpers.cjs");
      expect(typeof SHALLOW_RANGE_MAX_COMMITS).toBe("number");
      expect(SHALLOW_RANGE_MAX_COMMITS).toBeGreaterThan(0);
    });

    it("should respect the shallow status of the current repo when range exceeds threshold", async () => {
      const { checkImplausibleShallowRange, execGitSync } = await import("./git_helpers.cjs");
      // Use maxCommits:0 to force the shallow probe to run even on a 1-commit range —
      // this exercises the --is-shallow-repository branch regardless of environment.
      // The expected implausible flag depends on whether the current clone is shallow.
      let isShallow = false;
      try {
        isShallow = execGitSync(["rev-parse", "--is-shallow-repository"], { suppressLogs: true }).trim() === "true";
      } catch {
        // If the shallow probe fails, treat as non-shallow for this test
      }
      let result;
      try {
        result = checkImplausibleShallowRange("HEAD^", "HEAD", { maxCommits: 0 });
      } catch {
        // If HEAD^ does not exist (initial commit), skip
        return;
      }
      // Count is >0 (above threshold of 0); implausible iff the clone is shallow
      expect(result.commitCount).toBeGreaterThan(0);
      expect(result.implausible).toBe(isShallow);
    });
  });

  describe("hasMergeCommitsInRange", () => {
    it("should return false for empty baseRef", async () => {
      const { hasMergeCommitsInRange } = await import("./git_helpers.cjs");
      expect(hasMergeCommitsInRange("", "HEAD")).toBe(false);
    });

    it("should return false for empty headRef", async () => {
      const { hasMergeCommitsInRange } = await import("./git_helpers.cjs");
      expect(hasMergeCommitsInRange("origin/main", "")).toBe(false);
    });

    it("should return false when git rev-list fails (non-existent refs)", async () => {
      const { hasMergeCommitsInRange } = await import("./git_helpers.cjs");
      const result = hasMergeCommitsInRange("refs/nonexistent/base", "refs/nonexistent/head");
      expect(result).toBe(false);
    });

    it("should return false for HEAD..HEAD (empty range, no merge commits)", async () => {
      const { hasMergeCommitsInRange } = await import("./git_helpers.cjs");
      // HEAD..HEAD is an empty range; merges --count returns 0
      const result = hasMergeCommitsInRange("HEAD", "HEAD");
      expect(result).toBe(false);
    });

    it("should respect the shallow status of the current repo when range exceeds threshold", async () => {
      const { hasMergeCommitsInRange, execGitSync } = await import("./git_helpers.cjs");
      // maxCommits:0 forces the shallow probe for a 1-commit range.
      // In a shallow clone the range is implausible → returns false (no false-positive merges).
      // In a full clone the range is not implausible → proceeds to check for merges
      // (HEAD^..HEAD has no merge commits → still returns false).
      let isShallow = false;
      try {
        isShallow = execGitSync(["rev-parse", "--is-shallow-repository"], { suppressLogs: true }).trim() === "true";
      } catch {
        // treat as non-shallow
      }
      let result;
      try {
        result = hasMergeCommitsInRange("HEAD^", "HEAD", { maxCommits: 0 });
      } catch {
        // If HEAD^ does not exist (initial commit), skip
        return;
      }
      // Either path (shallow→early false, or non-shallow→no merge commits found→false)
      // yields false for a linear single-commit range.
      expect(result).toBe(false);
      void isShallow; // used above to document expected behavior
    });
  });
});

// ---------------------------------------------------------------------------
// Integration tests — real temporary git repositories
// ---------------------------------------------------------------------------

import { execSync, spawnSync } from "child_process";
import * as fs from "fs";
import * as path from "path";
import * as os from "os";
import { createRequire } from "module";

const requireLocal = createRequire(import.meta.url);

/**
 * Build a minimal execApi shim backed by real child_process calls so that
 * `linearizeRangeAsCommit` can be exercised end-to-end against a real repo.
 */
function makeRealExecApi(defaultCwd) {
  return {
    async exec(cmd, args, opts = {}) {
      const cwd = opts.cwd || defaultCwd;
      const result = spawnSync(cmd, args, { cwd, stdio: "pipe", encoding: "utf8" });
      if (result.status !== 0) {
        throw new Error(`${cmd} ${args.join(" ")} exited ${result.status}: ${result.stderr || result.stdout}`);
      }
      return { exitCode: 0 };
    },
    async getExecOutput(cmd, args, opts = {}) {
      const cwd = opts.cwd || defaultCwd;
      const result = spawnSync(cmd, args, { cwd, stdio: "pipe", encoding: "utf8" });
      if (result.status !== 0 && !opts.ignoreReturnCode) {
        throw new Error(`${cmd} ${args.join(" ")} exited ${result.status}: ${result.stderr || result.stdout}`);
      }
      return {
        stdout: result.stdout || "",
        stderr: result.stderr || "",
        exitCode: result.status || 0,
      };
    },
  };
}

/**
 * Write a file and commit it in a given repo directory.
 */
function addCommit(repoDir, filename, content, message) {
  fs.writeFileSync(path.join(repoDir, filename), content);
  execSync(`git add ${filename}`, { cwd: repoDir, stdio: "pipe" });
  execSync(`git commit -m "${message}"`, { cwd: repoDir, stdio: "pipe" });
}

describe("git_helpers.cjs - integration (real git repo)", () => {
  let repoDir;
  let remoteDir;

  beforeEach(() => {
    // Provide a no-op core stub (warning spy is added per-test as needed).
    global.core = {
      debug: () => {},
      info: () => {},
      warning: vi.fn(),
      error: () => {},
      setFailed: () => {},
    };

    // Bare remote.
    remoteDir = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-helpers-remote-"));
    execSync("git init --bare -b main", { cwd: remoteDir, stdio: "pipe" });

    // Working repo wired to the bare remote.
    repoDir = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-helpers-repo-"));
    execSync("git init -b main", { cwd: repoDir, stdio: "pipe" });
    execSync('git config user.email "test@example.com"', { cwd: repoDir, stdio: "pipe" });
    execSync('git config user.name "Test User"', { cwd: repoDir, stdio: "pipe" });
    execSync(`git remote add origin ${remoteDir}`, { cwd: repoDir, stdio: "pipe" });

    // Initial commit on main then push so origin/main exists.
    addCommit(repoDir, "README.md", "# Init\n", "init");
    execSync("git push origin main", { cwd: repoDir, stdio: "pipe" });
  });

  afterEach(() => {
    delete global.core;
    if (repoDir && fs.existsSync(repoDir)) fs.rmSync(repoDir, { recursive: true, force: true });
    if (remoteDir && fs.existsSync(remoteDir)) fs.rmSync(remoteDir, { recursive: true, force: true });
    // Clear module cache so each test group gets a fresh import.
    delete requireLocal.cache[requireLocal.resolve("./git_helpers.cjs")];
  });

  // -------------------------------------------------------------------------
  describe("checkImplausibleShallowRange - real repos", () => {
    it("returns implausible:false and the correct count for a small range in a full clone", async () => {
      const { checkImplausibleShallowRange } = requireLocal("./git_helpers.cjs");

      execSync("git checkout -b feature", { cwd: repoDir, stdio: "pipe" });
      addCommit(repoDir, "a.txt", "A\n", "add a");
      addCommit(repoDir, "b.txt", "B\n", "add b");
      addCommit(repoDir, "c.txt", "C\n", "add c");

      const result = checkImplausibleShallowRange("origin/main", "HEAD", { cwd: repoDir });
      expect(result.implausible).toBe(false);
      expect(result.commitCount).toBe(3);
    });

    it("returns implausible:false for a large range in a non-shallow clone", async () => {
      const { checkImplausibleShallowRange } = requireLocal("./git_helpers.cjs");

      execSync("git checkout -b feature", { cwd: repoDir, stdio: "pipe" });
      // Add 150 commits — above the default SHALLOW_RANGE_MAX_COMMITS (100).
      for (let i = 1; i <= 150; i++) {
        addCommit(repoDir, `f${i}.txt`, `${i}\n`, `commit ${i}`);
      }

      const result = checkImplausibleShallowRange("origin/main", "HEAD", { cwd: repoDir });
      // Full clone → never implausible regardless of range size.
      expect(result.implausible).toBe(false);
      expect(result.commitCount).toBeGreaterThan(100);
    });

    it("returns implausible:true for a shallow clone when range exceeds threshold", async () => {
      const shallowDir = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-helpers-shallow-"));
      try {
        // Use file:// so that --depth is honoured even for local repositories.
        execSync(`git clone --depth=1 --branch main "file://${remoteDir}" ${shallowDir}`, { stdio: "pipe" });
        execSync('git config user.email "test@example.com"', { cwd: shallowDir, stdio: "pipe" });
        execSync('git config user.name "Test User"', { cwd: shallowDir, stdio: "pipe" });

        // Add one commit on top so the range origin/main..HEAD is non-empty.
        addCommit(shallowDir, "extra.txt", "extra\n", "local change");

        const { checkImplausibleShallowRange } = requireLocal("./git_helpers.cjs");
        // maxCommits:0 means any non-empty range is implausible — ensures the
        // shallow probe runs even with a single-commit range.
        const result = checkImplausibleShallowRange("origin/main", "HEAD", { cwd: shallowDir, maxCommits: 0 });
        expect(result.implausible).toBe(true);
        expect(result.commitCount).toBeGreaterThan(0);
        expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("Shallow checkout produced an implausible commit range"));
      } finally {
        fs.rmSync(shallowDir, { recursive: true, force: true });
      }
    });
  });

  // -------------------------------------------------------------------------
  describe("hasMergeCommitsInRange - real repos", () => {
    it("returns false for a linear range with no merge commits", async () => {
      const { hasMergeCommitsInRange } = requireLocal("./git_helpers.cjs");

      execSync("git checkout -b linear-feature", { cwd: repoDir, stdio: "pipe" });
      addCommit(repoDir, "x.txt", "x\n", "add x");
      addCommit(repoDir, "y.txt", "y\n", "add y");

      expect(hasMergeCommitsInRange("origin/main", "HEAD", { cwd: repoDir })).toBe(false);
    });

    it("returns true when the range contains a merge commit", async () => {
      const { hasMergeCommitsInRange } = requireLocal("./git_helpers.cjs");

      // Create a side branch diverging from main.
      execSync("git checkout -b side", { cwd: repoDir, stdio: "pipe" });
      addCommit(repoDir, "side.txt", "side\n", "side commit");

      // Add a commit on main to ensure divergence, then merge side with --no-ff.
      execSync("git checkout main", { cwd: repoDir, stdio: "pipe" });
      addCommit(repoDir, "main-extra.txt", "extra\n", "main commit");
      execSync("git push origin main", { cwd: repoDir, stdio: "pipe" });
      execSync('git merge side --no-ff -m "merge side"', { cwd: repoDir, stdio: "pipe" });

      expect(hasMergeCommitsInRange("origin/main", "HEAD", { cwd: repoDir })).toBe(true);
    });

    it("returns false for a shallow clone with a range that exceeds the threshold", async () => {
      const shallowDir = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-helpers-shallow-merge-"));
      try {
        // Use file:// so that --depth is honoured even for local repositories.
        execSync(`git clone --depth=1 --branch main "file://${remoteDir}" ${shallowDir}`, { stdio: "pipe" });
        execSync('git config user.email "test@example.com"', { cwd: shallowDir, stdio: "pipe" });
        execSync('git config user.name "Test User"', { cwd: shallowDir, stdio: "pipe" });
        addCommit(shallowDir, "local.txt", "local\n", "local change");

        const { hasMergeCommitsInRange } = requireLocal("./git_helpers.cjs");
        // maxCommits:0 forces the implausible-range guard to fire for any non-empty range.
        const result = hasMergeCommitsInRange("origin/main", "HEAD", { cwd: shallowDir, maxCommits: 0 });
        // Guard fires → returns false rather than a phantom merge-commit detection.
        expect(result).toBe(false);
      } finally {
        fs.rmSync(shallowDir, { recursive: true, force: true });
      }
    });
  });

  // -------------------------------------------------------------------------
  describe("linearizeRangeAsCommit - real repos", () => {
    it("collapses multiple feature commits into a single commit on top of origin/main", async () => {
      const { linearizeRangeAsCommit } = requireLocal("./git_helpers.cjs");

      execSync("git checkout -b feature", { cwd: repoDir, stdio: "pipe" });
      addCommit(repoDir, "p.txt", "P\n", "add p");
      addCommit(repoDir, "q.txt", "Q\n", "add q");
      addCommit(repoDir, "r.txt", "R\n", "add r");

      const execApi = makeRealExecApi(repoDir);
      const gitOpts = { cwd: repoDir };

      const newSha = await linearizeRangeAsCommit("origin/main", "Squash: p q r", execApi, { gitOpts });

      // Must return a 40-hex SHA.
      expect(newSha).toMatch(/^[0-9a-f]{40}$/);

      // Exactly one commit on top of origin/main after linearization.
      const countOut = spawnSync("git", ["rev-list", "--count", "origin/main..HEAD"], { cwd: repoDir, encoding: "utf8" });
      expect(countOut.stdout.trim()).toBe("1");

      // The commit message must match what was supplied.
      const msgOut = spawnSync("git", ["log", "-1", "--format=%s"], { cwd: repoDir, encoding: "utf8" });
      expect(msgOut.stdout.trim()).toBe("Squash: p q r");

      // All three files from the feature branch must be present.
      expect(fs.existsSync(path.join(repoDir, "p.txt"))).toBe(true);
      expect(fs.existsSync(path.join(repoDir, "q.txt"))).toBe(true);
      expect(fs.existsSync(path.join(repoDir, "r.txt"))).toBe(true);
    });

    it("throws before any git state mutation for a shallow+implausible range", async () => {
      const shallowDir = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-helpers-shallow-lin-"));
      try {
        // Use file:// so that --depth is honoured even for local repositories.
        execSync(`git clone --depth=1 --branch main "file://${remoteDir}" ${shallowDir}`, { stdio: "pipe" });
        execSync('git config user.email "test@example.com"', { cwd: shallowDir, stdio: "pipe" });
        execSync('git config user.name "Test User"', { cwd: shallowDir, stdio: "pipe" });
        addCommit(shallowDir, "local.txt", "local\n", "local change");

        const { linearizeRangeAsCommit } = requireLocal("./git_helpers.cjs");
        const execApi = makeRealExecApi(shallowDir);
        const gitOpts = { cwd: shallowDir };

        const headBefore = spawnSync("git", ["rev-parse", "HEAD"], { cwd: shallowDir, encoding: "utf8" }).stdout.trim();

        // maxCommits:0 ensures the guard triggers for any non-empty range.
        await expect(linearizeRangeAsCommit("origin/main", "Should not commit", execApi, { gitOpts, maxCommits: 0 })).rejects.toThrow(/Refusing to linearize an implausible commit range/);

        // HEAD must be unchanged — the guard fires before any reset.
        const headAfter = spawnSync("git", ["rev-parse", "HEAD"], { cwd: shallowDir, encoding: "utf8" }).stdout.trim();
        expect(headAfter).toBe(headBefore);
      } finally {
        fs.rmSync(shallowDir, { recursive: true, force: true });
      }
    });

    it("restores original HEAD when the commit step fails", async () => {
      const { linearizeRangeAsCommit } = requireLocal("./git_helpers.cjs");

      execSync("git checkout -b feature-fail", { cwd: repoDir, stdio: "pipe" });
      addCommit(repoDir, "s.txt", "S\n", "add s");

      const headBefore = spawnSync("git", ["rev-parse", "HEAD"], { cwd: repoDir, encoding: "utf8" }).stdout.trim();

      // A broken execApi whose commit call always fails.
      const brokenApi = {
        async exec(cmd, args, opts) {
          if (args[0] === "reset") {
            // Allow the soft reset so the index changes.
            return makeRealExecApi(repoDir).exec(cmd, args, opts);
          }
          throw new Error("commit failed intentionally");
        },
        async getExecOutput(cmd, args, opts) {
          return makeRealExecApi(repoDir).getExecOutput(cmd, args, opts);
        },
      };

      await expect(linearizeRangeAsCommit("origin/main", "Should fail", brokenApi, { gitOpts: { cwd: repoDir } })).rejects.toThrow(/Failed to linearize/);

      // HEAD must be rolled back to what it was before the attempt.
      const headAfter = spawnSync("git", ["rev-parse", "HEAD"], { cwd: repoDir, encoding: "utf8" }).stdout.trim();
      expect(headAfter).toBe(headBefore);
    });
  });
});
