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
});
