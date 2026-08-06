import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

describe("git_auth_helpers.cjs", () => {
  let mockCore;
  let mockExec;
  let checkoutHasPersistedExtraheader;
  let findIncludedExtraheaderConfigFiles;
  let getGitAuthEnv;
  let overridePersistedExtraheader;
  let restorePersistedExtraheader;
  let unsetExtraheaderAllScopes;
  let withGitHubHostToken;

  const SERVER_URL = "https://github.com";
  const EXTRAHEADER_KEY = "http.https://github.com/.extraheader";

  beforeEach(() => {
    mockCore = {
      info: vi.fn(),
      warning: vi.fn(),
      setSecret: vi.fn(),
    };

    mockExec = {
      exec: vi.fn().mockResolvedValue(0),
      // Exit code 5 = key absent (the most common default state).
      // This is safe for both --get-all (returns []) and --unset-all (key not found, ignored).
      getExecOutput: vi.fn().mockResolvedValue({ exitCode: 5, stdout: "", stderr: "" }),
    };

    global.core = mockCore;
    global.exec = mockExec;

    delete require.cache[require.resolve("./git_auth_helpers.cjs")];
    ({ checkoutHasPersistedExtraheader, findIncludedExtraheaderConfigFiles, getGitAuthEnv, overridePersistedExtraheader, restorePersistedExtraheader, unsetExtraheaderAllScopes, withGitHubHostToken } = require("./git_auth_helpers.cjs"));
  });

  afterEach(() => {
    delete global.core;
    delete global.exec;
    vi.clearAllMocks();
  });

  describe("getGitAuthEnv", () => {
    it("masks the raw token and its base64 authorization value", () => {
      const env = getGitAuthEnv("derived-secret");
      const encoded = Buffer.from("x-access-token:derived-secret").toString("base64");

      expect(mockCore.setSecret.mock.calls).toEqual([["derived-secret"], [encoded]]);
      expect(env.GIT_CONFIG_VALUE_0).toBe(`Authorization: basic ${encoded}`);
    });
  });

  // ──────────────────────────────────────────────────────
  // checkoutHasPersistedExtraheader
  // ──────────────────────────────────────────────────────

  describe("checkoutHasPersistedExtraheader", () => {
    it("should return false when no extraheader is configured", async () => {
      mockExec.getExecOutput.mockResolvedValue({ exitCode: 1, stdout: "", stderr: "" });

      const result = await checkoutHasPersistedExtraheader(SERVER_URL);

      expect(result).toBe(false);
    });

    it("should return true when an extraheader is configured", async () => {
      const header = `Authorization: basic ${Buffer.from("x-access-token:tok").toString("base64")}`;
      mockExec.getExecOutput.mockResolvedValue({ exitCode: 0, stdout: header + "\n", stderr: "" });

      const result = await checkoutHasPersistedExtraheader(SERVER_URL);

      expect(result).toBe(true);
    });

    it("should strip a trailing slash from the server URL", async () => {
      mockExec.getExecOutput.mockResolvedValue({ exitCode: 1, stdout: "", stderr: "" });

      await checkoutHasPersistedExtraheader("https://github.com/");

      expect(mockExec.getExecOutput).toHaveBeenCalledWith("git", ["config", "--get-all", EXTRAHEADER_KEY], expect.anything());
    });
  });

  // ──────────────────────────────────────────────────────
  // unsetExtraheaderAllScopes
  // ──────────────────────────────────────────────────────

  describe("unsetExtraheaderAllScopes", () => {
    it("should unset from global and local scopes", async () => {
      await unsetExtraheaderAllScopes(EXTRAHEADER_KEY);

      expect(mockExec.getExecOutput).toHaveBeenCalledWith("git", ["config", "--global", "--unset-all", EXTRAHEADER_KEY], expect.objectContaining({ ignoreReturnCode: true }));
      expect(mockExec.getExecOutput).toHaveBeenCalledWith("git", ["config", "--local", "--unset-all", EXTRAHEADER_KEY], expect.objectContaining({ ignoreReturnCode: true }));
    });

    it("should pass cwd to both scope unset calls", async () => {
      const cwd = "/some/repo";
      await unsetExtraheaderAllScopes(EXTRAHEADER_KEY, cwd);

      for (const call of mockExec.getExecOutput.mock.calls) {
        expect(call[2]).toMatchObject({ cwd });
      }
    });

    it("should not throw when key is absent (exit code 5)", async () => {
      mockExec.getExecOutput.mockResolvedValue({ exitCode: 5, stdout: "", stderr: "" });

      await expect(unsetExtraheaderAllScopes(EXTRAHEADER_KEY)).resolves.toBeUndefined();
    });

    it("should throw when a scope unset fails with an unexpected non-zero exit code", async () => {
      // Exit code 4 = file write error (permission denied, config lock, etc.)
      mockExec.getExecOutput.mockResolvedValue({ exitCode: 4, stdout: "", stderr: "error: could not lock config file" });

      await expect(unsetExtraheaderAllScopes(EXTRAHEADER_KEY)).rejects.toThrow(/--unset-all.*failed \(exit 4\)/);
    });

    it("should also unset the key from includeIf.gitdir-referenced config files (checkout v7 case)", async () => {
      const originalRunnerTemp = process.env.RUNNER_TEMP;
      try {
        process.env.RUNNER_TEMP = "/home/runner/work/_temp";
        const credFile = "/home/runner/work/_temp/git-credentials-abc123.config";
        const includeKey = "includeif.gitdir:/home/runner/work/repo/.git/.path";
        mockExec.getExecOutput.mockImplementation(async (_cmd, args) => {
          if (args.includes("--name-only") && args.includes("--get-regexp")) {
            return { exitCode: 0, stdout: `${includeKey}\n`, stderr: "" };
          }
          if (args.includes("--get-all") && args.includes(includeKey)) {
            return { exitCode: 0, stdout: `${credFile}\n`, stderr: "" };
          }
          return { exitCode: 5, stdout: "", stderr: "" };
        });

        await unsetExtraheaderAllScopes(EXTRAHEADER_KEY);

        expect(mockExec.getExecOutput).toHaveBeenCalledWith("git", ["config", "--file", credFile, "--unset-all", EXTRAHEADER_KEY], expect.objectContaining({ ignoreReturnCode: true }));
      } finally {
        if (originalRunnerTemp === undefined) {
          delete process.env.RUNNER_TEMP;
        } else {
          process.env.RUNNER_TEMP = originalRunnerTemp;
        }
      }
    });
  });

  // ──────────────────────────────────────────────────────
  // findIncludedExtraheaderConfigFiles
  // ──────────────────────────────────────────────────────

  describe("findIncludedExtraheaderConfigFiles", () => {
    const originalRunnerTemp = process.env.RUNNER_TEMP;

    afterEach(() => {
      if (originalRunnerTemp === undefined) {
        delete process.env.RUNNER_TEMP;
      } else {
        process.env.RUNNER_TEMP = originalRunnerTemp;
      }
    });

    it("should return an empty array when there are no includeIf.gitdir entries", async () => {
      mockExec.getExecOutput.mockResolvedValue({ exitCode: 1, stdout: "", stderr: "" });

      const files = await findIncludedExtraheaderConfigFiles();

      expect(files).toEqual([]);
    });

    it("should return the resolved path of an includeIf.gitdir config file under RUNNER_TEMP", async () => {
      process.env.RUNNER_TEMP = "/home/runner/work/_temp";
      const credFile = "/home/runner/work/_temp/git-credentials-abc123.config";
      const includeKey = "includeif.gitdir:/home/runner/work/repo/.git/.path";
      mockExec.getExecOutput.mockImplementation(async (_cmd, args) => {
        if (args.includes("--name-only") && args.includes("--get-regexp")) {
          return { exitCode: 0, stdout: `${includeKey}\n`, stderr: "" };
        }
        if (args.includes("--get-all") && args.includes(includeKey)) {
          return { exitCode: 0, stdout: `${credFile}\n`, stderr: "" };
        }
        return { exitCode: 5, stdout: "", stderr: "" };
      });

      const files = await findIncludedExtraheaderConfigFiles();

      expect(files).toEqual([credFile]);
    });

    it("should skip and warn about config files outside safe temp directories", async () => {
      process.env.RUNNER_TEMP = "/home/runner/work/_temp";
      const maliciousPath = "/etc/passwd";
      const includeKey = "includeif.gitdir:/home/runner/work/repo/.git/.path";
      mockExec.getExecOutput.mockImplementation(async (_cmd, args) => {
        if (args.includes("--name-only") && args.includes("--get-regexp")) {
          return { exitCode: 0, stdout: `${includeKey}\n`, stderr: "" };
        }
        if (args.includes("--get-all") && args.includes(includeKey)) {
          return { exitCode: 0, stdout: `${maliciousPath}\n`, stderr: "" };
        }
        return { exitCode: 5, stdout: "", stderr: "" };
      });

      const files = await findIncludedExtraheaderConfigFiles();

      expect(files).toEqual([]);
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("skipping includeIf-referenced config file outside safe temp directories"));
    });
  });

  // ──────────────────────────────────────────────────────
  // overridePersistedExtraheader
  // ──────────────────────────────────────────────────────

  describe("overridePersistedExtraheader", () => {
    it("should clear all scopes before writing the new token", async () => {
      const token = "ghp_test_token";

      await overridePersistedExtraheader(SERVER_URL, token);

      expect(mockExec.getExecOutput).toHaveBeenCalledWith("git", ["config", "--global", "--unset-all", EXTRAHEADER_KEY], expect.objectContaining({ ignoreReturnCode: true }));
      expect(mockExec.getExecOutput).toHaveBeenCalledWith("git", ["config", "--local", "--unset-all", EXTRAHEADER_KEY], expect.objectContaining({ ignoreReturnCode: true }));
    });

    it("should replace the extraheader with the CI token using --local scope", async () => {
      const token = "ghp_test_token";
      const expectedHeader = `Authorization: basic ${Buffer.from(`x-access-token:${token}`).toString("base64")}`;

      await overridePersistedExtraheader(SERVER_URL, token);

      expect(mockExec.exec).toHaveBeenCalledWith("git", ["config", "--local", "--replace-all", EXTRAHEADER_KEY, expectedHeader], expect.objectContaining({ silent: true }));
    });

    it("should silence all exec calls to prevent credential capture in safe-output artifacts", async () => {
      await overridePersistedExtraheader(SERVER_URL, "ghp_test_token");

      // Every credential-bearing exec.exec call is silenced so the command line
      // is not captured in uploaded safe-output artifacts.
      for (const call of mockExec.exec.mock.calls) {
        expect(call[2]).toEqual(expect.objectContaining({ silent: true }));
      }
    });

    it("should register the base64 token with core.setSecret for runner-side masking", async () => {
      const token = "ghp_test_token";
      const expectedBase64 = Buffer.from(`x-access-token:${token}`).toString("base64");

      await overridePersistedExtraheader(SERVER_URL, token);

      expect(mockCore.setSecret).toHaveBeenCalledWith(expectedBase64);
    });

    it("should return empty array when no previous extraheader exists", async () => {
      mockExec.getExecOutput.mockImplementation(async (_cmd, args) => {
        // Only the --get-all read returns empty; unset-all calls use ignoreReturnCode
        if (args[1] === "--get-all") return { exitCode: 1, stdout: "", stderr: "" };
        return { exitCode: 5, stdout: "", stderr: "" }; // key not found
      });

      const previous = await overridePersistedExtraheader(SERVER_URL, "ghp_test_token");

      expect(previous).toEqual([]);
    });

    it("should return previous extraheader values when one exists", async () => {
      const prevHeader = `Authorization: basic ${Buffer.from("x-access-token:old_token").toString("base64")}`;
      mockExec.getExecOutput.mockImplementation(async (_cmd, args) => {
        if (args[1] === "--get-all") return { exitCode: 0, stdout: prevHeader + "\n", stderr: "" };
        return { exitCode: 5, stdout: "", stderr: "" };
      });

      const previous = await overridePersistedExtraheader(SERVER_URL, "ghp_new_token");

      expect(previous).toEqual([prevHeader]);
    });

    it("should return multiple previous values when multi-value extraheader exists", async () => {
      const header1 = `Authorization: basic ${Buffer.from("x-access-token:tok1").toString("base64")}`;
      const header2 = `Authorization: basic ${Buffer.from("x-access-token:tok2").toString("base64")}`;
      mockExec.getExecOutput.mockImplementation(async (_cmd, args) => {
        if (args[1] === "--get-all") return { exitCode: 0, stdout: `${header1}\n${header2}\n`, stderr: "" };
        return { exitCode: 5, stdout: "", stderr: "" };
      });

      const previous = await overridePersistedExtraheader(SERVER_URL, "ghp_new_token");

      expect(previous).toEqual([header1, header2]);
    });

    it("should warn and fall back to empty array when reading previous values fails", async () => {
      mockExec.getExecOutput.mockImplementation(async (_cmd, args) => {
        if (args[1] === "--get-all") throw new Error("git read error");
        return { exitCode: 5, stdout: "", stderr: "" };
      });

      const previous = await overridePersistedExtraheader(SERVER_URL, "ghp_test_token");

      expect(previous).toEqual([]);
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("could not read existing extraheader"));
      // Override should still proceed despite read failure
      expect(mockExec.exec).toHaveBeenCalledWith("git", ["config", "--local", "--replace-all", EXTRAHEADER_KEY, expect.any(String)], expect.objectContaining({ silent: true }));
    });

    it("should trim the token before base64-encoding", async () => {
      const token = "  ghp_padded_token  ";

      await overridePersistedExtraheader(SERVER_URL, token);

      const expected = `Authorization: basic ${Buffer.from("x-access-token:ghp_padded_token").toString("base64")}`;
      expect(mockExec.exec).toHaveBeenCalledWith("git", ["config", "--local", "--replace-all", EXTRAHEADER_KEY, expected], expect.objectContaining({ silent: true }));
    });

    it("should log the number of existing values before overriding", async () => {
      const header = `Authorization: basic ${Buffer.from("x-access-token:tok").toString("base64")}`;
      mockExec.getExecOutput.mockImplementation(async (_cmd, args) => {
        if (args[1] === "--get-all") return { exitCode: 0, stdout: header + "\n", stderr: "" };
        return { exitCode: 5, stdout: "", stderr: "" };
      });

      await overridePersistedExtraheader(SERVER_URL, "new_token");

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("1 existing extraheader value(s)"));
    });

    it("should strip trailing slash from server URL when reading previous values", async () => {
      // Use exit code 5 (key absent) for all calls so --unset-all is treated as "not found",
      // which is the expected state when testing URL normalization in isolation.
      mockExec.getExecOutput.mockResolvedValue({ exitCode: 5, stdout: "", stderr: "" });

      await overridePersistedExtraheader("https://github.com/", "ghp_test_token");

      // "https://github.com/" (with trailing slash) must be normalized to
      // "https://github.com" (without) for both the git config read and write.
      // Using the literal normalized key makes the assertion explicit.
      const normalizedKey = "http.https://github.com/.extraheader";
      expect(mockExec.getExecOutput).toHaveBeenCalledWith("git", ["config", "--get-all", normalizedKey], expect.anything());
      expect(mockExec.exec).toHaveBeenCalledWith("git", ["config", "--local", "--replace-all", normalizedKey, expect.any(String)], expect.objectContaining({ silent: true }));
    });
  });

  // ──────────────────────────────────────────────────────
  // restorePersistedExtraheader
  // ──────────────────────────────────────────────────────

  describe("restorePersistedExtraheader", () => {
    it("should clear all scopes when previousValues is empty", async () => {
      await restorePersistedExtraheader(SERVER_URL, []);

      expect(mockExec.getExecOutput).toHaveBeenCalledWith("git", ["config", "--global", "--unset-all", EXTRAHEADER_KEY], expect.objectContaining({ ignoreReturnCode: true }));
      expect(mockExec.getExecOutput).toHaveBeenCalledWith("git", ["config", "--local", "--unset-all", EXTRAHEADER_KEY], expect.objectContaining({ ignoreReturnCode: true }));
      // Should not call exec.exec (no values to write back)
      expect(mockExec.exec).not.toHaveBeenCalled();
    });

    it("should not throw when key is absent (exit code 5) when clearing scopes", async () => {
      mockExec.getExecOutput.mockResolvedValue({ exitCode: 5, stdout: "", stderr: "" });

      await expect(restorePersistedExtraheader(SERVER_URL, [])).resolves.toBeUndefined();
    });

    it("should use --local --replace-all to restore a single previous value", async () => {
      const prevHeader = `Authorization: basic ${Buffer.from("x-access-token:old").toString("base64")}`;

      await restorePersistedExtraheader(SERVER_URL, [prevHeader]);

      expect(mockExec.exec).toHaveBeenCalledWith("git", ["config", "--local", "--replace-all", EXTRAHEADER_KEY, prevHeader], expect.objectContaining({ silent: true }));
      expect(mockExec.exec).not.toHaveBeenCalledWith("git", expect.arrayContaining(["--add"]));
    });

    it("should restore multiple values using --local --replace-all then --local --add", async () => {
      const header1 = `Authorization: basic ${Buffer.from("x-access-token:tok1").toString("base64")}`;
      const header2 = `Authorization: basic ${Buffer.from("x-access-token:tok2").toString("base64")}`;

      await restorePersistedExtraheader(SERVER_URL, [header1, header2]);

      const calls = mockExec.exec.mock.calls;
      const replaceCall = calls.find(c => c[1][1] === "--local" && c[1][2] === "--replace-all");
      const addCall = calls.find(c => c[1][1] === "--local" && c[1][2] === "--add");

      expect(replaceCall).toBeDefined();
      expect(replaceCall[1][4]).toBe(header1);
      expect(addCall).toBeDefined();
      expect(addCall[1][4]).toBe(header2);
      // --replace-all must come before --add
      expect(calls.indexOf(replaceCall)).toBeLessThan(calls.indexOf(addCall));
    });

    it("should attempt cleanup via unsetExtraheaderAllScopes and re-throw when --add fails mid-restore", async () => {
      const header1 = `Authorization: basic ${Buffer.from("x-access-token:tok1").toString("base64")}`;
      const header2 = `Authorization: basic ${Buffer.from("x-access-token:tok2").toString("base64")}`;
      const addError = new Error("git config --add failed");

      mockExec.exec.mockImplementation(async (_cmd, args) => {
        if (args[1] === "--local" && args[2] === "--add") throw addError;
        return 0;
      });

      await expect(restorePersistedExtraheader(SERVER_URL, [header1, header2])).rejects.toThrow(/^git-config-credential failed/);

      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("partial extraheader restore"));
      // Cleanup should use unsetExtraheaderAllScopes (getExecOutput calls for --unset-all)
      const unsetCalls = mockExec.getExecOutput.mock.calls.filter(c => c[1][2] === "--unset-all");
      expect(unsetCalls.length).toBeGreaterThan(0);
    });

    it("should strip trailing slash from server URL for the config key", async () => {
      await restorePersistedExtraheader("https://github.com/", []);

      expect(mockExec.getExecOutput).toHaveBeenCalledWith("git", ["config", "--global", "--unset-all", EXTRAHEADER_KEY], expect.anything());
      expect(mockExec.getExecOutput).toHaveBeenCalledWith("git", ["config", "--local", "--unset-all", EXTRAHEADER_KEY], expect.anything());
    });

    it("should not throw when previousValues is null/undefined (treated as empty)", async () => {
      // null is treated as empty by the length check
      // @ts-expect-error intentional null test
      await expect(restorePersistedExtraheader(SERVER_URL, null)).resolves.toBeUndefined();
    });
  });

  // ──────────────────────────────────────────────────────
  // withGitHubHostToken
  // ──────────────────────────────────────────────────────

  describe("withGitHubHostToken", () => {
    it("should call the callback without any git config changes when token is empty", async () => {
      let callbackCalled = false;
      await withGitHubHostToken("", async () => {
        callbackCalled = true;
      });
      expect(callbackCalled).toBe(true);
      expect(mockExec.exec).not.toHaveBeenCalled();
    });

    it("should call the callback without any git config changes when token is undefined", async () => {
      let callbackCalled = false;
      // @ts-expect-error intentional undefined test
      await withGitHubHostToken(undefined, async () => {
        callbackCalled = true;
      });
      expect(callbackCalled).toBe(true);
      expect(mockExec.exec).not.toHaveBeenCalled();
    });

    it("should override extraheader with fork token before calling callback", async () => {
      const token = "fork-token";
      const expectedHeader = `Authorization: basic ${Buffer.from(`x-access-token:${token}`).toString("base64")}`;
      const execCalls = [];

      mockExec.exec.mockImplementation(async (_cmd, args) => {
        execCalls.push(args);
        return 0;
      });

      await withGitHubHostToken(token, async () => {
        // Inside callback the override should already be applied
        const overrideCall = execCalls.find(a => a[2] === "--replace-all");
        expect(overrideCall).toBeDefined();
        expect(overrideCall[4]).toBe(expectedHeader);
      });
    });

    it("should restore the previous extraheader after the callback completes", async () => {
      const upstreamHeader = `Authorization: basic ${Buffer.from("x-access-token:upstream").toString("base64")}`;
      mockExec.getExecOutput.mockImplementation(async (_cmd, args) => {
        if (args[1] === "--get-all") return { exitCode: 0, stdout: upstreamHeader + "\n", stderr: "" };
        return { exitCode: 5, stdout: "", stderr: "" };
      });

      await withGitHubHostToken("fork-token", async () => {});

      // After callback, the last --replace-all should restore the upstream header
      const replaceAllCalls = mockExec.exec.mock.calls.filter(c => c[1][2] === "--replace-all");
      expect(replaceAllCalls.length).toBeGreaterThanOrEqual(2);
      const restoreCall = replaceAllCalls[replaceAllCalls.length - 1];
      expect(restoreCall[1][4]).toBe(upstreamHeader);
    });

    it("should restore extraheader even when the callback throws", async () => {
      const upstreamHeader = `Authorization: basic ${Buffer.from("x-access-token:upstream").toString("base64")}`;
      mockExec.getExecOutput.mockImplementation(async (_cmd, args) => {
        if (args[1] === "--get-all") return { exitCode: 0, stdout: upstreamHeader + "\n", stderr: "" };
        return { exitCode: 5, stdout: "", stderr: "" };
      });

      const callbackError = new Error("push failed");
      await expect(
        withGitHubHostToken("fork-token", async () => {
          throw callbackError;
        })
      ).rejects.toThrow(callbackError);

      // Restore must still run after the callback throws — the last --replace-all is the restore
      const replaceAllCalls = mockExec.exec.mock.calls.filter(c => c[1][2] === "--replace-all");
      expect(replaceAllCalls.length).toBeGreaterThanOrEqual(2);
      const restoreCall = replaceAllCalls[replaceAllCalls.length - 1];
      expect(restoreCall[1][4]).toBe(upstreamHeader);
    });

    it("should clear all scopes after callback when no previous value existed", async () => {
      // Exit code 5 = key absent (the expected state when no extraheader is configured).
      // Using 1 here would cause unsetExtraheaderAllScopes to throw as of the fixed implementation.
      mockExec.getExecOutput.mockResolvedValue({ exitCode: 5, stdout: "", stderr: "" });

      await withGitHubHostToken("fork-token", async () => {});

      // Restore should call unsetExtraheaderAllScopes (global + local unset-all)
      const unsetCalls = mockExec.getExecOutput.mock.calls.filter(c => c[1][2] === "--unset-all");
      expect(unsetCalls.length).toBeGreaterThanOrEqual(2);
      const keys = unsetCalls.map(c => c[1][3]);
      expect(keys).toContain(EXTRAHEADER_KEY);
      // Should NOT call exec.exec for restore (no values to write back)
      const restoreExecCalls = mockExec.exec.mock.calls.filter(c => c[1][2] === "--replace-all");
      // First --replace-all is the override; no second one since previousValues is []
      // Check only one --replace-all call (the override itself)
      expect(restoreExecCalls.length).toBe(1);
    });

    it("should return the callback's return value", async () => {
      const result = await withGitHubHostToken("fork-token", async () => "expected-result");
      expect(result).toBe("expected-result");
    });

    it("should pass cwd to overridePersistedExtraheader and restorePersistedExtraheader", async () => {
      const cwd = "/some/repo";
      const upstreamHeader = `Authorization: basic ${Buffer.from("x-access-token:upstream").toString("base64")}`;
      mockExec.getExecOutput.mockImplementation(async (_cmd, args) => {
        if (args[1] === "--get-all") return { exitCode: 0, stdout: upstreamHeader + "\n", stderr: "" };
        return { exitCode: 5, stdout: "", stderr: "" };
      });

      await withGitHubHostToken("fork-token", async () => {}, cwd);

      // All git config calls should include cwd
      for (const call of mockExec.exec.mock.calls) {
        expect(call[2]).toMatchObject({ cwd });
      }
      for (const call of mockExec.getExecOutput.mock.calls) {
        expect(call[2]).toMatchObject({ cwd });
      }
    });

    it("should not accumulate extraheader values across multiple override/restore cycles", async () => {
      // Regression test: verifies header cardinality stays at 1 across retries.
      //
      // Simulates real git config state with independent global and local scopes so that
      // --unset-all, --replace-all, and --get-all operations interact with the same in-memory
      // state. Without this, --get-all is hard-coded to one value regardless of what the other
      // commands do, meaning the test would pass even with the old accumulating implementation.
      //
      // Bug: getExtraheaderValues reads ALL scopes (--get-all), but the old --replace-all
      // without an explicit scope only wrote to local, leaving the global value intact.
      // After each restore both scopes held a copy, so the next --get-all returned N+1 values.
      const upstreamHeader = `Authorization: basic ${Buffer.from("x-access-token:upstream").toString("base64")}`;

      // Simulate git config state: actions/checkout writes the upstream token to global scope.
      let globalValues = [upstreamHeader];
      let localValues = [];

      mockExec.getExecOutput.mockImplementation(async (_cmd, args) => {
        if (args[1] === "--get-all") {
          // git config --get-all reads from all scopes combined
          const all = [...globalValues, ...localValues];
          return all.length > 0 ? { exitCode: 0, stdout: all.join("\n") + "\n", stderr: "" } : { exitCode: 5, stdout: "", stderr: "" };
        }
        if (args[2] === "--unset-all") {
          if (args[1] === "--global") {
            const hadValues = globalValues.length > 0;
            globalValues = [];
            return { exitCode: hadValues ? 0 : 5, stdout: "", stderr: "" };
          }
          if (args[1] === "--local") {
            const hadValues = localValues.length > 0;
            localValues = [];
            return { exitCode: hadValues ? 0 : 5, stdout: "", stderr: "" };
          }
        }
        return { exitCode: 5, stdout: "", stderr: "" };
      });

      mockExec.exec.mockImplementation(async (_cmd, args) => {
        if (args[1] === "--local" && args[2] === "--replace-all") {
          localValues = [args[4]];
          return 0;
        }
        if (args[1] === "--local" && args[2] === "--add") {
          localValues.push(args[4]);
          return 0;
        }
        return 0;
      });

      let capturedPreviousValueCounts = [];
      mockCore.info.mockImplementation(msg => {
        const m = msg.match(/read (\d+) existing extraheader value/);
        if (m) capturedPreviousValueCounts.push(Number(m[1]));
      });

      // Run two override/restore cycles (simulating a retry scenario)
      for (let i = 0; i < 2; i++) {
        await withGitHubHostToken("fork-token", async () => {});
      }

      // Both cycles should read exactly 1 value — no accumulation.
      // Without the fix (no --global unset before writing), the second cycle would
      // see both global and local copies of the token and return 2.
      expect(capturedPreviousValueCounts).toEqual([1, 1]);
    });
  });
});
