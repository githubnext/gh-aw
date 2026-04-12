import { describe, it, expect, beforeEach, afterEach } from "vitest";

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
});
