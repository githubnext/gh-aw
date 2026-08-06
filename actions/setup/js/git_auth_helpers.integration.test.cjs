// Integration tests for git_auth_helpers.cjs.
//
// These tests use real git repositories so that --get-all, --unset-all,
// --replace-all, and --add operations interact with the same on-disk state.
// Global git config is isolated via GIT_CONFIG_GLOBAL so no test can pollute
// the developer's real ~/.gitconfig.
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import os from "os";
import path from "path";
import { spawnSync } from "child_process";
import { createRequire } from "module";

const require = createRequire(import.meta.url);

const SERVER_URL = "https://github.com";
const EXTRAHEADER_KEY = `http.${SERVER_URL}/.extraheader`;

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/**
 * Run a real git command with GIT_CONFIG_GLOBAL pointing at the isolated
 * config file.  Returns the spawnSync result; never throws on non-zero exit.
 */
function runGit(args, repoDir, globalConfigPath) {
  const env = { ...process.env, GIT_CONFIG_GLOBAL: globalConfigPath };
  const result = spawnSync("git", args, { encoding: "utf8", cwd: repoDir, env });
  if (result.error) throw result.error;
  return result;
}

/**
 * Create a temporary directory containing:
 *   - a bare global gitconfig file (.gitconfig-global)
 *   - an initialised git repository (repo/)
 *
 * Returns { repoDir, globalConfigPath }.
 */
function createIsolatedRepo(prefix) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), prefix));
  const globalConfigPath = path.join(root, ".gitconfig-global");
  // Minimal global config so git commits work
  fs.writeFileSync(globalConfigPath, "[user]\n\tname = Test User\n\temail = test@example.com\n");
  const repoDir = path.join(root, "repo");
  fs.mkdirSync(repoDir);
  runGit(["init", "-q"], repoDir, globalConfigPath);
  fs.writeFileSync(path.join(repoDir, "README.md"), "init\n");
  runGit(["add", "."], repoDir, globalConfigPath);
  runGit(["commit", "-q", "-m", "init"], repoDir, globalConfigPath);
  return { root, repoDir, globalConfigPath };
}

/**
 * Build the exec API passed to git_auth_helpers.cjs.
 * Every git subprocess it spawns carries GIT_CONFIG_GLOBAL so that --global
 * operations hit the isolated config file rather than the real user config.
 */
function createExecApi(repoDir, globalConfigPath) {
  function spawnGit(args, apiOptions = {}) {
    const cwd = apiOptions.cwd || repoDir;
    const env = { ...process.env, GIT_CONFIG_GLOBAL: globalConfigPath };
    return spawnSync("git", args, { encoding: "utf8", cwd, env });
  }
  return {
    async exec(cmd, args = [], options = {}) {
      if (cmd !== "git") throw new Error(`unexpected command: ${cmd}`);
      const r = spawnGit(args, options);
      if (r.error) throw r.error;
      if (r.status !== 0) {
        throw new Error(`git ${args.join(" ")} failed (${r.status}): ${r.stderr}`);
      }
      return r.status;
    },
    async getExecOutput(cmd, args = [], options = {}) {
      if (cmd !== "git") throw new Error(`unexpected command: ${cmd}`);
      const r = spawnGit(args, options);
      if (r.error) throw r.error;
      if (r.status !== 0 && !options.ignoreReturnCode) {
        throw new Error(`git ${args.join(" ")} failed (${r.status}): ${r.stderr}`);
      }
      return { exitCode: r.status, stdout: r.stdout, stderr: r.stderr };
    },
  };
}

/**
 * Read all values for key from a specific scope (or all scopes if scopeFlag
 * is omitted) using a real git process.
 */
function readConfigValues(key, scopeFlag, repoDir, globalConfigPath) {
  const args = scopeFlag ? ["config", scopeFlag, "--get-all", key] : ["config", "--get-all", key];
  const r = runGit(args, repoDir, globalConfigPath);
  // exit 5 = key absent; any other non-zero = error
  if (r.status !== 0) return [];
  return r.stdout.trim().split("\n").filter(Boolean);
}

/**
 * Write a single value to the given scope.
 */
function writeConfigValue(key, value, scopeFlag, repoDir, globalConfigPath) {
  const r = runGit(["config", scopeFlag, "--add", key, value], repoDir, globalConfigPath);
  if (r.status !== 0) throw new Error(`git config write failed: ${r.stderr}`);
}

// ---------------------------------------------------------------------------
// Suite
// ---------------------------------------------------------------------------

describe("git_auth_helpers.cjs git integration", () => {
  let repoDir;
  let globalConfigPath;
  let root;
  let mockCore;
  let overridePersistedExtraheader;
  let restorePersistedExtraheader;
  let unsetExtraheaderAllScopes;
  let withGitHubHostToken;

  let origGithubServerUrl;

  beforeEach(() => {
    ({ root, repoDir, globalConfigPath } = createIsolatedRepo("git-auth-helpers-it-"));

    origGithubServerUrl = process.env.GITHUB_SERVER_URL;
    process.env.GITHUB_SERVER_URL = SERVER_URL;

    mockCore = {
      info: vi.fn(),
      warning: vi.fn(),
      setSecret: vi.fn(),
    };

    global.core = mockCore;
    global.exec = createExecApi(repoDir, globalConfigPath);

    delete require.cache[require.resolve("./git_auth_helpers.cjs")];
    ({ overridePersistedExtraheader, restorePersistedExtraheader, unsetExtraheaderAllScopes, withGitHubHostToken } = require("./git_auth_helpers.cjs"));
  });

  afterEach(() => {
    if (root && fs.existsSync(root)) {
      fs.rmSync(root, { recursive: true, force: true });
    }
    if (origGithubServerUrl !== undefined) {
      process.env.GITHUB_SERVER_URL = origGithubServerUrl;
    } else {
      delete process.env.GITHUB_SERVER_URL;
    }
    delete global.core;
    delete global.exec;
    vi.clearAllMocks();
  });

  // ──────────────────────────────────────────────────────
  // unsetExtraheaderAllScopes
  // ──────────────────────────────────────────────────────

  describe("unsetExtraheaderAllScopes", () => {
    it("clears a value from the global scope", async () => {
      writeConfigValue(EXTRAHEADER_KEY, "Authorization: basic abc", "--global", repoDir, globalConfigPath);
      expect(readConfigValues(EXTRAHEADER_KEY, "--global", repoDir, globalConfigPath)).toHaveLength(1);

      await unsetExtraheaderAllScopes(EXTRAHEADER_KEY, repoDir);

      expect(readConfigValues(EXTRAHEADER_KEY, "--global", repoDir, globalConfigPath)).toHaveLength(0);
    });

    it("clears a value from the local scope", async () => {
      writeConfigValue(EXTRAHEADER_KEY, "Authorization: basic abc", "--local", repoDir, globalConfigPath);
      expect(readConfigValues(EXTRAHEADER_KEY, "--local", repoDir, globalConfigPath)).toHaveLength(1);

      await unsetExtraheaderAllScopes(EXTRAHEADER_KEY, repoDir);

      expect(readConfigValues(EXTRAHEADER_KEY, "--local", repoDir, globalConfigPath)).toHaveLength(0);
    });

    it("clears values from both global and local scopes simultaneously", async () => {
      writeConfigValue(EXTRAHEADER_KEY, "Authorization: basic global", "--global", repoDir, globalConfigPath);
      writeConfigValue(EXTRAHEADER_KEY, "Authorization: basic local", "--local", repoDir, globalConfigPath);
      // Both scopes have a value; --get-all should return 2
      expect(readConfigValues(EXTRAHEADER_KEY, null, repoDir, globalConfigPath)).toHaveLength(2);

      await unsetExtraheaderAllScopes(EXTRAHEADER_KEY, repoDir);

      expect(readConfigValues(EXTRAHEADER_KEY, "--global", repoDir, globalConfigPath)).toHaveLength(0);
      expect(readConfigValues(EXTRAHEADER_KEY, "--local", repoDir, globalConfigPath)).toHaveLength(0);
      expect(readConfigValues(EXTRAHEADER_KEY, null, repoDir, globalConfigPath)).toHaveLength(0);
    });

    it("does not throw when the key is absent in both scopes (exit code 5)", async () => {
      // No value written — key is absent in all scopes.
      await expect(unsetExtraheaderAllScopes(EXTRAHEADER_KEY, repoDir)).resolves.toBeUndefined();
    });

    it("clears the value from an includeIf.gitdir-referenced config file (checkout v7 style)", async () => {
      // Simulate actions/checkout@v7: it writes credentials to a separate config
      // file under RUNNER_TEMP and references it via a local includeIf.gitdir entry
      // instead of writing the extraheader directly into local/global config.
      const origRunnerTemp = process.env.RUNNER_TEMP;
      process.env.RUNNER_TEMP = root;
      try {
        const credFile = path.join(root, "git-credentials-abc123.config");
        fs.writeFileSync(credFile, `[http "${SERVER_URL}/"]\n\textraheader = Authorization: basic upstream\n`);
        const gitDirPath = repoDir + path.sep;
        runGit(["config", "--local", `includeIf.gitdir:${gitDirPath}.path`, credFile], repoDir, globalConfigPath);

        // The value is effective (visible via --get-all which resolves includes)
        expect(readConfigValues(EXTRAHEADER_KEY, null, repoDir, globalConfigPath)).toHaveLength(1);

        await unsetExtraheaderAllScopes(EXTRAHEADER_KEY, repoDir);

        // After clearing, the include file's value must be gone and no longer effective.
        const fileContents = fs.readFileSync(credFile, "utf8");
        expect(fileContents).not.toContain("extraheader");
        expect(readConfigValues(EXTRAHEADER_KEY, null, repoDir, globalConfigPath)).toHaveLength(0);
      } finally {
        if (origRunnerTemp !== undefined) {
          process.env.RUNNER_TEMP = origRunnerTemp;
        } else {
          delete process.env.RUNNER_TEMP;
        }
      }
    });
  });

  // ──────────────────────────────────────────────────────
  // overridePersistedExtraheader
  // ──────────────────────────────────────────────────────

  describe("overridePersistedExtraheader", () => {
    it("removes the global token and writes only the fork token to local scope", async () => {
      const upstreamHeader = `Authorization: basic ${Buffer.from("x-access-token:upstream").toString("base64")}`;
      const forkToken = "fork-token-123";
      const expectedForkHeader = `Authorization: basic ${Buffer.from(`x-access-token:${forkToken}`).toString("base64")}`;

      writeConfigValue(EXTRAHEADER_KEY, upstreamHeader, "--global", repoDir, globalConfigPath);

      await overridePersistedExtraheader(SERVER_URL, forkToken, repoDir);

      // Global must be empty
      expect(readConfigValues(EXTRAHEADER_KEY, "--global", repoDir, globalConfigPath)).toHaveLength(0);
      // Local must have exactly the fork token
      expect(readConfigValues(EXTRAHEADER_KEY, "--local", repoDir, globalConfigPath)).toEqual([expectedForkHeader]);
      // get-all must see exactly one header
      expect(readConfigValues(EXTRAHEADER_KEY, null, repoDir, globalConfigPath)).toEqual([expectedForkHeader]);
    });

    it("returns the upstream header that was in the global scope before the override", async () => {
      const upstreamHeader = `Authorization: basic ${Buffer.from("x-access-token:upstream").toString("base64")}`;
      writeConfigValue(EXTRAHEADER_KEY, upstreamHeader, "--global", repoDir, globalConfigPath);

      const previous = await overridePersistedExtraheader(SERVER_URL, "fork-token", repoDir);

      expect(previous).toEqual([upstreamHeader]);
    });

    it("returns an empty array and still writes fork token when no extraheader exists", async () => {
      const forkToken = "fork-only";
      const expectedForkHeader = `Authorization: basic ${Buffer.from(`x-access-token:${forkToken}`).toString("base64")}`;

      const previous = await overridePersistedExtraheader(SERVER_URL, forkToken, repoDir);

      expect(previous).toEqual([]);
      expect(readConfigValues(EXTRAHEADER_KEY, null, repoDir, globalConfigPath)).toEqual([expectedForkHeader]);
    });
  });

  // ──────────────────────────────────────────────────────
  // restorePersistedExtraheader
  // ──────────────────────────────────────────────────────

  describe("restorePersistedExtraheader", () => {
    it("removes the fork token from local and restores the upstream token to local", async () => {
      const upstreamHeader = `Authorization: basic ${Buffer.from("x-access-token:upstream").toString("base64")}`;
      const forkHeader = `Authorization: basic ${Buffer.from("x-access-token:fork").toString("base64")}`;

      // Simulate the state after an override: global is empty, local has the fork token.
      writeConfigValue(EXTRAHEADER_KEY, forkHeader, "--local", repoDir, globalConfigPath);

      await restorePersistedExtraheader(SERVER_URL, [upstreamHeader], repoDir);

      // Global must remain empty
      expect(readConfigValues(EXTRAHEADER_KEY, "--global", repoDir, globalConfigPath)).toHaveLength(0);
      // Local must have the restored upstream token
      expect(readConfigValues(EXTRAHEADER_KEY, "--local", repoDir, globalConfigPath)).toEqual([upstreamHeader]);
      // get-all must see exactly one header
      expect(readConfigValues(EXTRAHEADER_KEY, null, repoDir, globalConfigPath)).toEqual([upstreamHeader]);
    });

    it("clears all scopes and writes nothing when previousValues is empty", async () => {
      const forkHeader = `Authorization: basic ${Buffer.from("x-access-token:fork").toString("base64")}`;
      writeConfigValue(EXTRAHEADER_KEY, forkHeader, "--local", repoDir, globalConfigPath);

      await restorePersistedExtraheader(SERVER_URL, [], repoDir);

      expect(readConfigValues(EXTRAHEADER_KEY, "--global", repoDir, globalConfigPath)).toHaveLength(0);
      expect(readConfigValues(EXTRAHEADER_KEY, "--local", repoDir, globalConfigPath)).toHaveLength(0);
    });
  });

  // ──────────────────────────────────────────────────────
  // withGitHubHostToken — multi-cycle regression
  // ──────────────────────────────────────────────────────

  describe("withGitHubHostToken", () => {
    it("does not accumulate extraheader values across multiple override/restore cycles", async () => {
      // Simulates the exact production failure: actions/checkout writes the upstream
      // token to the global scope.  Without the scope-clearing fix, each restore would
      // leave the global value intact and add a new local copy, so --get-all returns
      // N+1 values on each subsequent cycle.
      const upstreamHeader = `Authorization: basic ${Buffer.from("x-access-token:upstream").toString("base64")}`;
      writeConfigValue(EXTRAHEADER_KEY, upstreamHeader, "--global", repoDir, globalConfigPath);

      const capturedCounts = [];
      mockCore.info.mockImplementation(msg => {
        const m = msg.match(/read (\d+) existing extraheader value/);
        if (m) capturedCounts.push(Number(m[1]));
      });

      // Two full override/restore cycles (simulates two retries)
      for (let i = 0; i < 2; i++) {
        await withGitHubHostToken("fork-token", async () => {}, repoDir);
      }

      // Without the fix: [1, 2] — second cycle sees global + local copies.
      // With the fix: [1, 1] — global is always cleared before each write.
      expect(capturedCounts).toEqual([1, 1]);
    });

    it("leaves exactly one extraheader value active inside the callback", async () => {
      const upstreamHeader = `Authorization: basic ${Buffer.from("x-access-token:upstream").toString("base64")}`;
      const forkToken = "fork-callback-check";
      const expectedForkHeader = `Authorization: basic ${Buffer.from(`x-access-token:${forkToken}`).toString("base64")}`;
      writeConfigValue(EXTRAHEADER_KEY, upstreamHeader, "--global", repoDir, globalConfigPath);

      let valuesInsideCallback;
      await withGitHubHostToken(
        forkToken,
        async () => {
          valuesInsideCallback = readConfigValues(EXTRAHEADER_KEY, null, repoDir, globalConfigPath);
        },
        repoDir
      );

      // Exactly one Authorization header must be active during the callback
      expect(valuesInsideCallback).toEqual([expectedForkHeader]);
    });

    it("restores exactly one extraheader value after the callback completes", async () => {
      const upstreamHeader = `Authorization: basic ${Buffer.from("x-access-token:upstream").toString("base64")}`;
      writeConfigValue(EXTRAHEADER_KEY, upstreamHeader, "--global", repoDir, globalConfigPath);

      await withGitHubHostToken("fork-token", async () => {}, repoDir);

      // After restore, exactly the upstream token should be in local scope (global was cleared)
      const allValues = readConfigValues(EXTRAHEADER_KEY, null, repoDir, globalConfigPath);
      expect(allValues).toEqual([upstreamHeader]);
    });

    it("restores the config even when the callback throws", async () => {
      const upstreamHeader = `Authorization: basic ${Buffer.from("x-access-token:upstream").toString("base64")}`;
      writeConfigValue(EXTRAHEADER_KEY, upstreamHeader, "--global", repoDir, globalConfigPath);

      await expect(
        withGitHubHostToken(
          "fork-token",
          async () => {
            throw new Error("simulated callback failure");
          },
          repoDir
        )
      ).rejects.toThrow("simulated callback failure");

      const allValues = readConfigValues(EXTRAHEADER_KEY, null, repoDir, globalConfigPath);
      expect(allValues).toEqual([upstreamHeader]);
    });
  });
});
