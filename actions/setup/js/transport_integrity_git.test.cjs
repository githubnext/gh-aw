import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createRequire } from "module";
import { spawnSync } from "child_process";
import fs from "fs";
import os from "os";
import path from "path";
import { withTransportIntegrity } from "./transport_integrity.cjs";
import { useExecutionProvenanceFixture } from "./test_execution_provenance.js";

useExecutionProvenanceFixture();
const require = createRequire(import.meta.url);

describe("transport integrity Git cleanup", () => {
  let directory;
  let repo;

  function git(args) {
    const result = spawnSync("git", args, {
      cwd: repo,
      encoding: "utf8",
      env: { ...process.env, GIT_CONFIG_GLOBAL: os.devNull, GIT_CONFIG_NOSYSTEM: "1" },
    });
    if (result.error) throw result.error;
    return { exitCode: result.status, stdout: result.stdout, stderr: result.stderr };
  }

  function mustGit(args) {
    const result = git(args);
    if (result.exitCode !== 0) throw new Error(result.stderr);
    return result.stdout;
  }

  function execApi(onCommand) {
    const run = async (command, args, options = {}) => {
      if (command !== "git") throw new Error(`Unexpected command: ${command}`);
      const result = git(args);
      onCommand(args, result);
      if (result.exitCode !== 0 && !options.ignoreReturnCode) throw new Error(result.stderr);
      return result;
    };
    return {
      exec: vi.fn(async (command, args, options) => (await run(command, args, options)).exitCode),
      getExecOutput: vi.fn(run),
    };
  }

  beforeEach(() => {
    directory = fs.mkdtempSync(path.join(process.cwd(), ".transport-git-test-"));
    repo = path.join(directory, "repo");
    fs.mkdirSync(repo);
    mustGit(["init", "-q", "-b", "main"]);
    mustGit(["config", "user.name", "Test"]);
    mustGit(["config", "user.email", "test@example.com"]);
    mustGit(["config", "commit.gpgsign", "false"]);
    fs.writeFileSync(path.join(repo, "file.txt"), "base\n");
    mustGit(["add", "file.txt"]);
    mustGit(["commit", "-qm", "base"]);
    vi.stubGlobal("core", { info: vi.fn(), warning: vi.fn(), error: vi.fn(), debug: vi.fn() });
    vi.stubEnv("GH_AW_PROMPTS_DIR", path.resolve("..", "md"));
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.unstubAllEnvs();
    fs.rmSync(directory, { recursive: true, force: true });
  });

  it("cleans up after a bundle integrity failure without publishing", async () => {
    mustGit(["branch", "feature"]);
    const bundle = path.join(directory, "aw-feature.bundle");
    mustGit(["bundle", "create", bundle, "feature"]);
    const api = execApi(args => {
      if (args[0] === "bundle" && args[1] === "unbundle") fs.writeFileSync(bundle, "changed after import");
    });
    const { applyBundleToBranch } = require("./create_pull_request.cjs");
    const handler = withTransportIntegrity(async (_message, _ids, transport) => {
      transport.captureOptional(bundle);
      await applyBundleToBranch(bundle, "result", "feature", transport.guardExec(api), "main");
      return { success: true };
    });
    await expect(handler({}, {})).resolves.toMatchObject({ success: false, error_type: "ERR_TRANSPORT_INTEGRITY" });
    expect(mustGit(["for-each-ref", "--format=%(refname)", "refs/bundles/"]).trim()).toBe("");
    expect(api.exec.mock.calls.some(([, args]) => args[0] === "push")).toBe(false);
  });

  it.each([false, true])("aborts a real patch conflict and restores the checkout (transport changed: %s)", async tamper => {
    mustGit(["checkout", "-qb", "feature"]);
    fs.writeFileSync(path.join(repo, "file.txt"), "feature\n");
    mustGit(["commit", "-qam", "feature"]);
    const patch = path.join(directory, "aw-feature.patch");
    fs.writeFileSync(patch, mustGit(["format-patch", "-1", "--stdout"]));
    mustGit(["checkout", "-q", "main"]);
    fs.writeFileSync(path.join(repo, "file.txt"), "target\n");
    mustGit(["commit", "-qam", "target"]);
    const before = mustGit(["rev-parse", "HEAD"]);
    const api = execApi(args => {
      if (tamper && args[0] === "am" && args[1] === "--3way") fs.writeFileSync(patch, "changed during apply");
    });
    const handler = withTransportIntegrity(async (_message, _ids, transport) => {
      transport.captureOptional(patch);
      await transport.guardExec(api).exec("git", ["am", "--3way", patch], { cwd: repo });
      return { success: true };
    });
    if (tamper) await expect(handler({}, {})).resolves.toMatchObject({ success: false, error_type: "ERR_TRANSPORT_INTEGRITY" });
    else await expect(handler({}, {})).rejects.toThrow();
    expect(fs.existsSync(path.join(repo, ".git", "rebase-apply"))).toBe(false);
    expect(mustGit(["rev-parse", "HEAD"])).toBe(before);
    expect(fs.readFileSync(path.join(repo, "file.txt"), "utf8")).toBe("target\n");
    expect(mustGit(["status", "--porcelain"])).toBe("");
    expect(api.exec).toHaveBeenCalledWith("git", ["am", "--abort"], expect.objectContaining({ cwd: repo }));
  });

  it.each(["explicit abort", "finalizer cleanup"])("does not mask malformed patch errors with %s when git am never starts", async cleanup => {
    const patch = path.join(directory, "malformed.patch");
    fs.writeFileSync(patch, "not a format-patch file\n");
    const api = execApi(() => {});
    const handler = withTransportIntegrity(async (_message, _ids, transport) => {
      transport.captureOptional(patch);
      const exec = transport.guardExec(api);
      try {
        await exec.exec("git", ["am", "--3way", patch], { cwd: repo });
      } catch (error) {
        if (cleanup === "explicit abort") await exec.exec("git", ["am", "--abort"], { cwd: repo });
        throw error;
      }
    });

    await expect(handler({}, {})).rejects.toThrow("Patch format detection failed");

    expect(api.exec.mock.calls.some(([, args]) => args[0] === "am" && args[1] === "--abort")).toBe(false);
    expect(mustGit(["status", "--porcelain"])).toBe("");
  });

  it("restores the pinned checkout after integrity failure during a conflicting bundle rewrite", async () => {
    const baseHead = mustGit(["rev-parse", "HEAD"]).trim();
    mustGit(["checkout", "-qb", "feature"]);
    fs.writeFileSync(path.join(repo, "file.txt"), "feature\n");
    mustGit(["commit", "-qam", "feature"]);
    const originalHead = mustGit(["rev-parse", "HEAD"]).trim();
    const bundle = path.join(directory, "aw-feature.bundle");
    mustGit(["bundle", "create", bundle, "feature", `^${baseHead}`]);

    mustGit(["checkout", "-q", "main"]);
    fs.writeFileSync(path.join(repo, "file.txt"), "target\n");
    mustGit(["commit", "-qam", "target"]);
    mustGit(["update-ref", "refs/remotes/origin/main", mustGit(["rev-parse", "HEAD"]).trim()]);
    mustGit(["checkout", "-q", "feature"]);

    const api = execApi(args => {
      if (args[0] === "rebase" && args[1] === "--onto") {
        fs.unlinkSync(bundle);
        fs.writeFileSync(bundle, "changed during rebase");
      }
    });
    const handler = withTransportIntegrity(async (_message, _ids, transport) => {
      transport.captureOptional(bundle);
      const { rewriteBundleBranchAsSingleCommit } = require("./create_pull_request.cjs");
      await rewriteBundleBranchAsSingleCommit("main", transport.guardExec(api), bundle, { gitOpts: { cwd: repo } });
    });

    await expect(handler({}, {})).resolves.toMatchObject({ success: false, error_type: "ERR_TRANSPORT_INTEGRITY" });

    expect(api.exec).toHaveBeenCalledWith("git", ["rebase", "--abort"], expect.objectContaining({ cwd: repo }));
    expect(api.exec).toHaveBeenCalledWith("git", ["reset", "--hard", originalHead], expect.objectContaining({ cwd: repo }));
    expect(mustGit(["rev-parse", "HEAD"]).trim()).toBe(originalHead);
    expect(fs.readFileSync(path.join(repo, "file.txt"), "utf8")).toBe("feature\n");
    expect(fs.existsSync(path.join(repo, ".git", "rebase-merge"))).toBe(false);
    expect(fs.existsSync(path.join(repo, ".git", "rebase-apply"))).toBe(false);
    expect(mustGit(["status", "--porcelain"])).toBe("");
  });
});
