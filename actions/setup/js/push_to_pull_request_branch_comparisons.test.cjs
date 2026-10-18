import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

describe("bundle merge-parent comparison budget", () => {
  const comparisonBudget = 4;
  const trustedTip = "a".repeat(40);
  const oldHead = "b".repeat(40);
  const candidate = "c".repeat(40);
  const repoParts = { owner: "test-owner", repo: "test-repo" };
  const secretDiagnostic = "synthetic secret-bearing API diagnostic";
  const parents = Array.from({ length: 24 }, (_, index) => (index + 1).toString(16).padStart(40, "0"));
  let mockCore;
  let mockExec;
  let compareCommits;
  let resolveBundleTrustedBase;

  beforeEach(() => {
    mockCore = { debug: vi.fn(), warning: vi.fn() };
    vi.stubGlobal("core", mockCore);
    mockExec = {
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
    return resolveBundleTrustedBase(mockExec, { cwd: process.cwd(), env: { GIT_ALLOW_PROTOCOL: "https", GIT_NO_LAZY_FETCH: "0" } }, oldHead, candidate, trustedTip, { rest: { repos: { compareCommits } } }, repoParts);
  }

  it.each(["missing", "ahead", "diverged", "unknown"])("caps 24 attacker-controlled parents at four comparisons after %s results", async status => {
    compareCommits.mockImplementation(async () => {
      if (compareCommits.mock.calls.length > comparisonBudget) return { data: { status: "behind" } };
      if (status === "missing") throw Object.assign(new Error(secretDiagnostic), { status: 404 });
      return { data: { status } };
    });

    expect(await resolve()).toBe("");
    expect(compareCommits).toHaveBeenCalledTimes(comparisonBudget);
    expect(compareCommits.mock.calls).toEqual(parents.slice(0, comparisonBudget).map(head => [{ ...repoParts, base: trustedTip, head, per_page: 1 }]));
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

  it.each(["behind", "identical"])("accepts a confirmed %s ancestor after missing parents within the budget", async status => {
    compareCommits.mockImplementation(async () => {
      if (compareCommits.mock.calls.length < comparisonBudget) throw Object.assign(new Error(secretDiagnostic), { status: 404 });
      return { data: { status } };
    });

    expect(await resolve()).toBe(parents[comparisonBudget - 1]);
    expect(compareCommits).toHaveBeenCalledTimes(comparisonBudget);
    expect(compareCommits).toHaveBeenLastCalledWith({ ...repoParts, base: trustedTip, head: parents[comparisonBudget - 1], per_page: 1 });
  });
});
