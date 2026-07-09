import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

describe("git_auth_helpers.cjs", () => {
  let mockCore;
  let mockExec;
  let checkoutHasPersistedExtraheader;
  let overridePersistedExtraheader;
  let restorePersistedExtraheader;

  const SERVER_URL = "https://github.com";
  const EXTRAHEADER_KEY = "http.https://github.com/.extraheader";

  beforeEach(() => {
    mockCore = {
      info: vi.fn(),
      warning: vi.fn(),
    };

    mockExec = {
      exec: vi.fn().mockResolvedValue(0),
      getExecOutput: vi.fn().mockResolvedValue({ exitCode: 1, stdout: "", stderr: "" }),
    };

    global.core = mockCore;
    global.exec = mockExec;

    delete require.cache[require.resolve("./git_auth_helpers.cjs")];
    ({ checkoutHasPersistedExtraheader, overridePersistedExtraheader, restorePersistedExtraheader } = require("./git_auth_helpers.cjs"));
  });

  afterEach(() => {
    delete global.core;
    delete global.exec;
    vi.clearAllMocks();
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
  // overridePersistedExtraheader
  // ──────────────────────────────────────────────────────

  describe("overridePersistedExtraheader", () => {
    it("should replace the extraheader with the CI token", async () => {
      const token = "ghp_test_token";

      await overridePersistedExtraheader(SERVER_URL, token);

      expect(mockExec.exec).toHaveBeenCalledWith("git", ["config", "--replace-all", EXTRAHEADER_KEY, `Authorization: basic ${Buffer.from(`x-access-token:${token}`).toString("base64")}`]);
    });

    it("should return empty array when no previous extraheader exists", async () => {
      mockExec.getExecOutput.mockResolvedValue({ exitCode: 1, stdout: "", stderr: "" });

      const previous = await overridePersistedExtraheader(SERVER_URL, "ghp_test_token");

      expect(previous).toEqual([]);
    });

    it("should return previous extraheader values when one exists", async () => {
      const prevHeader = `Authorization: basic ${Buffer.from("x-access-token:old_token").toString("base64")}`;
      mockExec.getExecOutput.mockResolvedValue({ exitCode: 0, stdout: prevHeader + "\n", stderr: "" });

      const previous = await overridePersistedExtraheader(SERVER_URL, "ghp_new_token");

      expect(previous).toEqual([prevHeader]);
    });

    it("should return multiple previous values when multi-value extraheader exists", async () => {
      const header1 = `Authorization: basic ${Buffer.from("x-access-token:tok1").toString("base64")}`;
      const header2 = `Authorization: basic ${Buffer.from("x-access-token:tok2").toString("base64")}`;
      mockExec.getExecOutput.mockResolvedValue({ exitCode: 0, stdout: `${header1}\n${header2}\n`, stderr: "" });

      const previous = await overridePersistedExtraheader(SERVER_URL, "ghp_new_token");

      expect(previous).toEqual([header1, header2]);
    });

    it("should warn and fall back to empty array when reading previous values fails", async () => {
      mockExec.getExecOutput.mockRejectedValue(new Error("git read error"));

      const previous = await overridePersistedExtraheader(SERVER_URL, "ghp_test_token");

      expect(previous).toEqual([]);
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("could not read existing extraheader"));
      // Override should still proceed despite read failure
      expect(mockExec.exec).toHaveBeenCalledWith("git", ["config", "--replace-all", EXTRAHEADER_KEY, expect.any(String)]);
    });

    it("should trim the token before base64-encoding", async () => {
      const token = "  ghp_padded_token  ";

      await overridePersistedExtraheader(SERVER_URL, token);

      const expected = `Authorization: basic ${Buffer.from("x-access-token:ghp_padded_token").toString("base64")}`;
      expect(mockExec.exec).toHaveBeenCalledWith("git", ["config", "--replace-all", EXTRAHEADER_KEY, expected]);
    });

    it("should log the number of existing values before overriding", async () => {
      const header = `Authorization: basic ${Buffer.from("x-access-token:tok").toString("base64")}`;
      mockExec.getExecOutput.mockResolvedValue({ exitCode: 0, stdout: header + "\n", stderr: "" });

      await overridePersistedExtraheader(SERVER_URL, "new_token");

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("1 existing extraheader value(s)"));
    });
  });

  // ──────────────────────────────────────────────────────
  // restorePersistedExtraheader
  // ──────────────────────────────────────────────────────

  describe("restorePersistedExtraheader", () => {
    it("should unset the extraheader key when previousValues is empty", async () => {
      await restorePersistedExtraheader(SERVER_URL, []);

      expect(mockExec.exec).toHaveBeenCalledWith("git", ["config", "--unset-all", EXTRAHEADER_KEY]);
    });

    it("should not throw when unset-all fails (key already absent)", async () => {
      mockExec.exec.mockRejectedValue(new Error("key not found"));

      await expect(restorePersistedExtraheader(SERVER_URL, [])).resolves.toBeUndefined();
    });

    it("should use --replace-all to restore a single previous value", async () => {
      const prevHeader = `Authorization: basic ${Buffer.from("x-access-token:old").toString("base64")}`;

      await restorePersistedExtraheader(SERVER_URL, [prevHeader]);

      expect(mockExec.exec).toHaveBeenCalledWith("git", ["config", "--replace-all", EXTRAHEADER_KEY, prevHeader]);
      expect(mockExec.exec).not.toHaveBeenCalledWith("git", expect.arrayContaining(["--add"]));
    });

    it("should restore multiple values using --replace-all then --add", async () => {
      const header1 = `Authorization: basic ${Buffer.from("x-access-token:tok1").toString("base64")}`;
      const header2 = `Authorization: basic ${Buffer.from("x-access-token:tok2").toString("base64")}`;

      await restorePersistedExtraheader(SERVER_URL, [header1, header2]);

      const calls = mockExec.exec.mock.calls;
      const replaceCall = calls.find(c => c[1][1] === "--replace-all");
      const addCall = calls.find(c => c[1][1] === "--add");

      expect(replaceCall).toBeDefined();
      expect(replaceCall[1][3]).toBe(header1);
      expect(addCall).toBeDefined();
      expect(addCall[1][3]).toBe(header2);
      // --replace-all must come before --add
      expect(calls.indexOf(replaceCall)).toBeLessThan(calls.indexOf(addCall));
    });

    it("should attempt cleanup and re-throw when --add fails mid-restore", async () => {
      const header1 = `Authorization: basic ${Buffer.from("x-access-token:tok1").toString("base64")}`;
      const header2 = `Authorization: basic ${Buffer.from("x-access-token:tok2").toString("base64")}`;
      const addError = new Error("git config --add failed");

      mockExec.exec.mockImplementation(async (_cmd, args) => {
        if (args[1] === "--add") throw addError;
        return 0;
      });

      await expect(restorePersistedExtraheader(SERVER_URL, [header1, header2])).rejects.toThrow(addError);

      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("partial extraheader restore"));
      // Cleanup --unset-all should have been attempted
      const calls = mockExec.exec.mock.calls;
      const unsetCall = calls.find(c => c[1][1] === "--unset-all");
      expect(unsetCall).toBeDefined();
    });

    it("should strip trailing slash from server URL for the config key", async () => {
      await restorePersistedExtraheader("https://github.com/", []);

      expect(mockExec.exec).toHaveBeenCalledWith("git", ["config", "--unset-all", EXTRAHEADER_KEY]);
    });

    it("should not throw when previousValues is null/undefined (treated as empty)", async () => {
      // null is treated as empty by the length check
      // @ts-expect-error intentional null test
      await expect(restorePersistedExtraheader(SERVER_URL, null)).resolves.toBeUndefined();
    });
  });
});
