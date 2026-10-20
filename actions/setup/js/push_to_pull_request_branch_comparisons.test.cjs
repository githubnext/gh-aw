import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

describe("bundle merge-parent comparison budget", () => {
  const comparisonBudget = 4;
  const trustedTip = "a".repeat(40);
  const oldHead = "b".repeat(40);
  const candidate = "c".repeat(40);
  const repoParts = { owner: "test-owner", repo: "test-repo" };
  const secretDiagnostic = "synthetic secret-bearing API diagnostic";
  const request = { timeout: 5000, retries: 0, signal: expect.any(AbortSignal) };
  const parents = Array.from({ length: 24 }, (_, index) => (index + 1).toString(16).padStart(40, "0"));
  let mockCore;
  let mockExec;
  let compareCommits;
  let resolveBundleTrustedBase;

  beforeEach(() => {
    mockCore = { debug: vi.fn(), warning: vi.fn() };
    vi.stubGlobal("core", mockCore);
    mockExec = {
      async execGitSyncResult(args, options) {
        const result = await mockExec.getExecOutput("git", args, { ...options, silent: true, ignoreReturnCode: true });
        return { ...result, status: result.exitCode };
      },
      exec: vi.fn(),
      getExecOutput: vi.fn(async (_command, args) => {
        if (args[0] === "merge-base") return { exitCode: 128, stdout: "", stderr: secretDiagnostic };
        if (args[1] === "--merges") {
          const merges = parents.map((parent, index) => `${(index + 100).toString(16).padStart(40, "0")} ${oldHead} ${parent}`);
          return { exitCode: 0, stdout: `${merges.join("\n")}\n`, stderr: "" };
        }
        if (args[1] === "--topo-order") return { exitCode: 0, stdout: `${parents.join("\n")}\n`, stderr: "" };
        throw new Error("Unexpected git operation");
      }),
    };
    compareCommits = vi.fn();
    ({ resolveBundleTrustedBase } = require("./push_to_pull_request_branch.cjs"));
  });

  afterEach(() => {
    expect(mockExec.exec).not.toHaveBeenCalled();
    for (const [command, args, options] of mockExec.getExecOutput.mock.calls) {
      expect(command).toBe("git");
      expect(["merge-base", "rev-list"]).toContain(args[0]);
      expect(options).toEqual(
        expect.objectContaining({
          cwd: process.cwd(),
          silent: true,
          ignoreReturnCode: true,
          env: expect.objectContaining({ GIT_ALLOW_PROTOCOL: "", GIT_NO_LAZY_FETCH: "1", GIT_TERMINAL_PROMPT: "0" }),
        })
      );
    }
    expect(JSON.stringify([mockCore.debug.mock.calls, mockCore.warning.mock.calls])).not.toContain(secretDiagnostic);
    vi.unstubAllGlobals();
  });

  async function resolve() {
    return resolveBundleTrustedBase(mockExec, { cwd: process.cwd(), env: { GIT_ALLOW_PROTOCOL: "https", GIT_NO_LAZY_FETCH: "0" } }, oldHead, candidate, trustedTip, { rest: { repos: { compareCommits } } }, repoParts, true);
  }

  it.each(["missing", "ahead", "diverged", "unknown"])("caps 24 attacker-controlled parents at four comparisons after %s results", async status => {
    compareCommits.mockImplementation(async () => {
      if (compareCommits.mock.calls.length > comparisonBudget) return { data: { status: "behind" } };
      if (status === "missing") throw Object.assign(new Error(secretDiagnostic), { status: 404 });
      return { data: { status } };
    });

    expect(await resolve()).toBe("");
    expect(compareCommits).toHaveBeenCalledTimes(comparisonBudget);
    expect(compareCommits.mock.calls).toEqual(parents.slice(0, comparisonBudget).map(head => [{ ...repoParts, base: trustedTip, head, per_page: 1, request }]));
    expect(mockCore.debug).toHaveBeenCalledWith("Bundle validation: merge parent comparison budget exhausted");
  });

  it.each([401, 403, 429, 500, 502, 503, 504])("stops immediately on HTTP %s rather than querying another parent", async status => {
    compareCommits.mockRejectedValueOnce(Object.assign(new Error(secretDiagnostic), { status })).mockResolvedValue({ data: { status: "behind" } });

    expect(await resolve()).toBe("");
    expect(compareCommits).toHaveBeenCalledTimes(1);
  });

  it.each([
    ["unknown error", new Error(secretDiagnostic)],
    ["transport error", Object.assign(new Error(secretDiagnostic), { code: "ECONNRESET" })],
    ["string error", secretDiagnostic],
    ["null error", null],
    ["undefined error", undefined],
    ["unrecognized status", Object.assign(new Error(secretDiagnostic), { status: "404" })],
  ])("stops immediately on %s", async (_label, error) => {
    compareCommits.mockRejectedValueOnce(error).mockResolvedValue({ data: { status: "behind" } });

    expect(await resolve()).toBe("");
    expect(compareCommits).toHaveBeenCalledTimes(1);
  });

  it("aborts a stalled comparison after five seconds and stops discovery", async () => {
    compareCommits.mockImplementation(
      ({ request }) =>
        new Promise((_resolve, reject) => {
          request.signal.addEventListener("abort", () => reject(request.signal.reason), { once: true });
        })
    );
    expect(await resolve()).toBe("");
    expect(compareCommits).toHaveBeenCalledTimes(1);
    expect(compareCommits.mock.calls[0][0].request.signal.aborted).toBe(true);
  });

  it.each(["behind", "identical"])("accepts a confirmed %s ancestor after missing parents within the budget", async status => {
    compareCommits.mockImplementation(async () => {
      if (compareCommits.mock.calls.length < comparisonBudget) throw Object.assign(new Error(secretDiagnostic), { status: 404 });
      return { data: { status } };
    });

    expect(await resolve()).toBe(parents[comparisonBudget - 1]);
    expect(compareCommits).toHaveBeenCalledTimes(comparisonBudget);
    expect(compareCommits).toHaveBeenLastCalledWith({ ...repoParts, base: trustedTip, head: parents[comparisonBudget - 1], per_page: 1, request });
  });

  it.each(["merge history", "parent history", "octopus merge"])("falls back without API comparisons when %s exceeds the discovery budget", async scenario => {
    const original = mockExec.getExecOutput.getMockImplementation();
    mockExec.getExecOutput.mockImplementation(async (command, args, options) => {
      if ((scenario === "merge history" && args[1] === "--merges") || (scenario === "parent history" && args[1] === "--topo-order")) {
        return { exitCode: 0, stdout: Array.from({ length: 101 }, () => `${candidate} ${oldHead} ${parents[0]}`).join("\n"), stderr: "" };
      }
      if (scenario === "octopus merge" && args[1] === "--merges") {
        return { exitCode: 0, stdout: `${candidate} ${oldHead} ${parents[0]} ${parents[1]}\n`, stderr: "" };
      }
      return original(command, args, options);
    });

    expect(await resolve()).toBe("");
    expect(compareCommits).not.toHaveBeenCalled();
    for (const [, args] of mockExec.getExecOutput.mock.calls) {
      if (args[0] === "rev-list") expect(args).toContain("--max-count=101");
    }
  });

  it.each(["merge history", "parent history"])("still recognizes an ancestor at the 100-record %s boundary", async scenario => {
    const original = mockExec.getExecOutput.getMockImplementation();
    mockExec.getExecOutput.mockImplementation(async (command, args, options) => {
      if (scenario === "merge history" && args[1] === "--merges") {
        return { exitCode: 0, stdout: Array.from({ length: 100 }, () => `${candidate} ${oldHead} ${parents[0]}`).join("\n"), stderr: "" };
      }
      if (scenario === "parent history" && args[1] === "--topo-order") {
        return { exitCode: 0, stdout: [...Array.from({ length: 99 }, () => candidate), parents[0]].join("\n"), stderr: "" };
      }
      return original(command, args, options);
    });
    compareCommits.mockResolvedValue({ data: { status: "behind" } });
    expect(await resolve()).toBe(parents[0]);
    expect(compareCommits).toHaveBeenCalledTimes(1);
  });
});

describe("bounded optional validation Git", () => {
  const childProcess = require("child_process");
  const gitHelpers = require("./git_helpers.cjs");
  const { resolveTrustedBaseTip, resolveBundleTrustedBase, getBundleValidationBase } = require("./push_to_pull_request_branch.cjs");
  let spawn;
  let realSpawn;
  let debug;

  beforeEach(() => {
    debug = vi.fn();
    vi.stubGlobal("core", { debug, warning: vi.fn() });
    realSpawn = childProcess.spawnSync;
    spawn = vi.spyOn(childProcess, "spawnSync");
  });

  afterEach(() => {
    spawn.mockRestore();
    vi.unstubAllGlobals();
  });

  it("enforces process and output bounds that caller options cannot relax", async () => {
    spawn.mockReturnValue({ status: 0, stdout: "git output", stderr: "private diagnostic" });
    expect(await resolveTrustedBaseTip(gitHelpers, { cwd: "/checkout", env: { GIT_ALLOW_PROTOCOL: "" }, timeout: 0, maxBuffer: Infinity, shell: true }, "main")).toBe("");
    expect(spawn).toHaveBeenCalledWith(
      "git",
      ["ls-remote", "--refs", "--exit-code", "origin", "refs/heads/main"],
      expect.objectContaining({ cwd: "/checkout", env: expect.objectContaining({ GIT_ALLOW_PROTOCOL: "" }), timeout: 5000, maxBuffer: 1024 * 1024, killSignal: "SIGKILL", stdio: "pipe" })
    );
    expect(spawn.mock.calls.every(([, , options]) => options.shell !== true)).toBe(true);
    expect(JSON.stringify(debug.mock.calls)).not.toContain("private");
  });

  it.each(["ETIMEDOUT", "ENOBUFS", "ENOENT", "SIGKILL", "SIGINT"])("discards incomplete output and falls back after %s", async failure => {
    spawn.mockReturnValue({
      status: failure.startsWith("SIG") ? null : 0,
      signal: failure.startsWith("SIG") ? failure : null,
      error: failure.startsWith("SIG") ? undefined : Object.assign(new Error("private diagnostic"), { code: failure }),
      stdout: "partial private output",
      stderr: "private diagnostic",
    });
    const compareCommits = vi.fn();
    const client = { rest: { repos: { compareCommits } } };
    expect(await resolveTrustedBaseTip(gitHelpers, {}, "main")).toBe("");
    expect(await resolveBundleTrustedBase(gitHelpers, {}, "old", "candidate", "a".repeat(40), client, { owner: "owner", repo: "repo" }, true)).toBe("");
    expect(await getBundleValidationBase(gitHelpers, {}, "old", "candidate", "a".repeat(40))).toEqual({ ref: "old", conflictedFiles: [] });
    expect(compareCommits).not.toHaveBeenCalled();
    expect(JSON.stringify(debug.mock.calls)).not.toContain("private");
  });

  it("preserves merge-tree conflict output for conservative file protection", async () => {
    spawn
      .mockReturnValueOnce({ status: 0, stdout: "", stderr: "" })
      .mockReturnValueOnce({ status: 0, stdout: "", stderr: "" })
      .mockReturnValueOnce({ status: 1, stdout: "", stderr: "" })
      .mockReturnValue({ status: 1, stdout: `${"a".repeat(40)}\0package.json\0\0`, stderr: "private diagnostic" });
    expect(await getBundleValidationBase(gitHelpers, {}, "old", "candidate", "base")).toEqual({ ref: "a".repeat(40), conflictedFiles: ["package.json"] });
    expect(JSON.stringify(debug.mock.calls)).not.toContain("private");
  });

  it.each([
    ["ETIMEDOUT", "setInterval(() => {}, 1000)"],
    ["ENOBUFS", "process.stdout.write(Buffer.alloc(2 * 1024 * 1024)); setInterval(() => {}, 1000)"],
  ])("terminates a real subprocess on %s without returning partial output", async (code, script) => {
    let result;
    spawn.mockImplementation((_command, _args, options) => {
      result = realSpawn(process.execPath, ["-e", script], options);
      return result;
    });
    expect(await resolveTrustedBaseTip(gitHelpers, {}, "main")).toBe("");
    expect(result.error.code).toBe(code);
    expect(result.signal).toBe("SIGKILL");
  });
});
