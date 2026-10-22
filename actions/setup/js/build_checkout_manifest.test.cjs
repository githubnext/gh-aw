import { afterEach, describe, expect, it, vi } from "vitest";
import fs from "fs";
import os from "os";
import path from "path";
import { spawnSync } from "child_process";
import { createRequire } from "module";

import { getSetupTimeoutMs } from "./child_process_timeouts.cjs";
import { buildCheckoutManifest, main, readManifestEntriesFromEnv, resolveDefaultBranch } from "./build_checkout_manifest.cjs";
import { _resetCache as resetCheckoutManifestCache, lookupCheckout } from "./checkout_manifest.cjs";
import { findGitDirectories, findRepoCheckout } from "./find_repo_checkout.cjs";

const require = createRequire(import.meta.url);

function execGit(args, options = {}) {
  const result = spawnSync("git", ["-c", "core.hooksPath=/dev/null", "-c", "commit.gpgsign=false", ...args], { encoding: "utf8", ...options });
  if (result.error) throw result.error;
  if (result.status !== 0) {
    throw new Error(`git ${args.join(" ")} failed:\nstdout: ${result.stdout}\nstderr: ${result.stderr}`);
  }
  return result.stdout;
}

function createTempDir(prefix) {
  return fs.mkdtempSync(path.join(os.tmpdir(), prefix));
}

function removeDir(dir) {
  if (dir && fs.existsSync(dir)) {
    fs.rmSync(dir, { recursive: true, force: true });
  }
}

function setEnv(key, value) {
  if (value === undefined) {
    delete process.env[key];
  } else {
    process.env[key] = value;
  }
}

describe("build_checkout_manifest.cjs", () => {
  const originalEnv = { ...process.env };
  const tempDirs = [];

  afterEach(() => {
    for (const key of Object.keys(process.env)) {
      if (!(key in originalEnv)) {
        delete process.env[key];
      }
    }
    Object.assign(process.env, originalEnv);
    while (tempDirs.length > 0) {
      removeDir(tempDirs.pop());
    }
  });

  it("reads entries from environment variables", () => {
    setEnv("GH_AW_CHECKOUT_MANIFEST_COUNT", "2");
    setEnv("GH_AW_CHECKOUT_REPO_0", "owner/a");
    setEnv("GH_AW_CHECKOUT_PATH_0", "./a");
    setEnv("GH_AW_CHECKOUT_TOKEN_0", "${{ secrets.REPO_A_TOKEN }}");
    setEnv("GH_AW_CHECKOUT_REF_0", "release/1");
    setEnv("GH_AW_CHECKOUT_REPO_1", "owner/b");
    setEnv("GH_AW_CHECKOUT_PATH_1", "");

    expect(readManifestEntriesFromEnv()).toEqual([
      { repository: "owner/a", path: "./a", ref: "release/1", token: "${{ secrets.REPO_A_TOKEN }}" },
      { repository: "owner/b", path: "", ref: "", token: "" },
    ]);
  });

  it("resolves default branch from local git checkout before gh fallback", () => {
    const workspace = createTempDir("checkout-manifest-workspace-");
    tempDirs.push(workspace);
    const checkoutPath = "target";
    const repoDir = path.join(workspace, checkoutPath);
    fs.mkdirSync(repoDir, { recursive: true });

    execGit(["init", "-q"], { cwd: repoDir });
    execGit(["symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main"], { cwd: repoDir });

    const ghCalls = [];
    const defaultBranch = resolveDefaultBranch("owner/repo", checkoutPath, {
      workspace,
      runGH: args => {
        ghCalls.push(args);
        return "should-not-be-used\n";
      },
    });

    expect(defaultBranch).toBe("main");
    expect(ghCalls).toHaveLength(0);
  });

  it("passes a timeout to local git default branch lookup", () => {
    const workspace = createTempDir("checkout-manifest-workspace-");
    tempDirs.push(workspace);
    const checkoutPath = "target";
    const repoDir = path.join(workspace, checkoutPath);
    fs.mkdirSync(path.join(repoDir, ".git"), { recursive: true });
    /** @type {any} */
    let gitOptions = null;

    const defaultBranch = resolveDefaultBranch("owner/repo", checkoutPath, {
      workspace,
      runGit: (_args, options) => {
        gitOptions = options;
        return "origin/main\n";
      },
    });

    expect(defaultBranch).toBe("main");
    expect(gitOptions?.timeout).toBe(getSetupTimeoutMs("gitBranch"));
  });

  it("falls back to gh api when local git default branch is unavailable", () => {
    const workspace = createTempDir("checkout-manifest-workspace-");
    tempDirs.push(workspace);
    /** @type {any} */
    let ghOptions = null;

    const defaultBranch = resolveDefaultBranch("owner/repo", "missing", {
      workspace,
      checkoutToken: "${{ secrets.CROSS_REPO_PAT }}",
      runGH: (_args, options) => {
        ghOptions = options;
        return "trunk\n";
      },
    });

    expect(defaultBranch).toBe("trunk");
    expect(ghOptions?.env?.GH_TOKEN).toBe("${{ secrets.CROSS_REPO_PAT }}");
    expect(ghOptions?.timeout).toBe(getSetupTimeoutMs("outcomeGh"));
  });

  it.each([
    { host: "tenant.ghe.com", checkoutToken: "checkout-token" },
    { host: "tenant.ghe.com", checkoutToken: "" },
    { host: undefined, checkoutToken: "checkout-token" },
    { host: undefined, checkoutToken: "" },
  ])("preserves the gh subprocess environment with host=$host and checkoutToken=$checkoutToken", ({ host, checkoutToken }) => {
    const workspace = createTempDir("checkout-manifest-workspace-");
    tempDirs.push(workspace);
    setEnv("GH_HOST", host);
    setEnv("GH_TOKEN", "ambient-token");
    setEnv("GH_AW_ENV_TEST", "inherited");
    const ambientEnv = { ...process.env };
    const childProcess = require("child_process");
    const modulePath = require.resolve("./build_checkout_manifest.cjs");
    const cached = require.cache[modulePath];
    const mock = vi.spyOn(childProcess, "execFileSync").mockReturnValue("trunk\n");
    delete require.cache[modulePath];
    try {
      const defaultBranch = require("./build_checkout_manifest.cjs").resolveDefaultBranch("owner/repo", "missing", {
        workspace,
        checkoutToken,
      });

      expect(defaultBranch).toBe("trunk");
      expect(mock).toHaveBeenCalledTimes(1);
      const [command, args, execOptions] = mock.mock.calls[0];
      expect(command).toBe("gh");
      expect(args).toEqual(["api", "repos/owner/repo", "--jq", ".default_branch"]);
      expect({
        encoding: execOptions.encoding,
        stdio: execOptions.stdio,
        timeout: execOptions.timeout,
      }).toEqual({
        encoding: "utf8",
        stdio: ["ignore", "pipe", "pipe"],
        timeout: getSetupTimeoutMs("outcomeGh"),
      });
      expect(execOptions.env.GH_HOST).toBe(host);
      expect(execOptions.env.GH_TOKEN).toBe(checkoutToken || "ambient-token");
      expect(execOptions.env.GH_AW_ENV_TEST).toBe("inherited");
      expect(execOptions.env.PATH).toBe(ambientEnv.PATH);
      expect(process.env.GH_TOKEN).toBe("ambient-token");
      expect(execOptions.env).not.toBe(process.env);
    } finally {
      mock.mockRestore();
      require.cache[modulePath] = cached;
    }
  });

  it("writes manifest with lowercase keys", () => {
    const workspace = createTempDir("checkout-manifest-workspace-");
    const runnerTemp = createTempDir("checkout-manifest-runner-temp-");
    tempDirs.push(workspace, runnerTemp);

    const { manifestPath, manifest } = buildCheckoutManifest(
      [
        { repository: "Owner/Repo", path: "./repo" },
        { repository: "", path: "./skip" },
      ],
      {
        workspace,
        runnerTemp,
        runGH: () => "main\n",
      }
    );

    expect(manifestPath).toBe(path.join(runnerTemp, "gh-aw", "safeoutputs", "checkout-manifest.json"));
    expect(manifest).toEqual({
      "owner/repo": {
        repository: "Owner/Repo",
        path: "./repo",
        default_branch: "main",
      },
    });

    const fileContents = JSON.parse(fs.readFileSync(manifestPath, "utf8"));
    expect(fileContents).toEqual(manifest);
  });

  it.each(["release/1", "refs/heads/release/1", "detached-branch"])(
    "snapshots selected branch %s without mistaking it for the repository default",
    ref => {
      const workspace = createTempDir("checkout-manifest-selected-");
      const runnerTemp = createTempDir("checkout-manifest-runner-");
      tempDirs.push(workspace, runnerTemp);
      execGit(["init", "-b", "release/1", "-q", workspace]);
      execGit(["-C", workspace, "-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "--allow-empty", "-m", "release"]);
      const sha = execGit(["-C", workspace, "rev-parse", "HEAD"]).trim();
      execGit(["-C", workspace, "update-ref", "refs/remotes/origin/release/1", sha]);
      if (ref === "detached-branch") execGit(["-C", workspace, "checkout", "--detach", "HEAD"]);
      const runGH = vi.fn(() => "main\n");
      const { manifest } = buildCheckoutManifest([{ repository: "owner/repo", path: "", ref: ref === "detached-branch" ? "release/1" : ref }], { workspace, runnerTemp, runGH });
      expect(manifest["owner/repo"]).toMatchObject({ default_branch: "main", checked_out_ref: "release/1" });
      execGit(["-C", workspace, "checkout", "-b", "agent-feature"]);
      expect(JSON.parse(fs.readFileSync(path.join(runnerTemp, "gh-aw/safeoutputs/checkout-manifest.json"), "utf8"))["owner/repo"].checked_out_ref).toBe("release/1");
    },
    30000
  );

  it.each(["", "v1", "refs/tags/v1", "sha"])(
    "does not infer a branch for an unselected or detached ref %s",
    ref => {
      const workspace = createTempDir("checkout-manifest-detached-");
      const runnerTemp = createTempDir("checkout-manifest-runner-");
      tempDirs.push(workspace, runnerTemp);
      execGit(["init", "-b", "main", "-q", workspace]);
      execGit(["-C", workspace, "-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "--allow-empty", "-m", "base"]);
      const sha = execGit(["-C", workspace, "rev-parse", "HEAD"]).trim();
      execGit(["-C", workspace, "update-ref", "refs/remotes/origin/main", sha]);
      execGit(["-C", workspace, "tag", "v1"]);
      execGit(["-C", workspace, "checkout", "--detach", "v1"]);
      const { manifest } = buildCheckoutManifest([{ repository: "owner/repo", path: "", ref: ref === "sha" ? sha : ref }], { workspace, runnerTemp, runGH: () => "main\n" });
      expect(manifest["owner/repo"]).not.toHaveProperty("checked_out_ref");
      expect(manifest["owner/repo"].default_branch).toBe("main");
    },
    30000
  );

  it("resolves runtime-plan checkouts beyond the scan depth through the manifest without scanning", async () => {
    const workspace = createTempDir("checkout-manifest-deep-workspace-");
    const runnerTemp = createTempDir("checkout-manifest-deep-runner-");
    tempDirs.push(workspace, runnerTemp);
    const checkoutPath = "one/two/three/four/five/six/targets/owner/repo";
    const repoDir = path.join(workspace, checkoutPath);
    fs.mkdirSync(path.join(repoDir, ".git"), { recursive: true });
    const planDir = path.join(runnerTemp, "gh-aw", "dynamic-checkouts");
    fs.mkdirSync(planDir, { recursive: true });
    fs.writeFileSync(path.join(planDir, "0.json"), JSON.stringify([{ repository: "Owner/Repo", path: checkoutPath, default_branch: "main" }]));
    setEnv("RUNNER_TEMP", runnerTemp);
    setEnv("GITHUB_WORKSPACE", workspace);
    setEnv("GH_AW_CHECKOUT_MANIFEST", undefined);
    setEnv("GH_AW_CHECKOUT_MANIFEST_COUNT", "0");
    setEnv("GH_AW_DYNAMIC_CHECKOUT_INDEXES", "0");

    expect(findGitDirectories(workspace)).not.toContain(repoDir);
    await main();
    resetCheckoutManifestCache();
    const scan = vi.spyOn(fs, "readdirSync").mockImplementation(() => {
      throw new Error("Manifest lookup must not scan the workspace");
    });
    try {
      expect(findRepoCheckout("owner/repo", workspace)).toEqual({
        success: true,
        path: repoDir,
        repoSlug: "owner/repo",
      });
      expect(lookupCheckout("owner/repo")).toEqual({
        repository: "Owner/Repo",
        path: checkoutPath,
        default_branch: "main",
      });
      expect(scan).not.toHaveBeenCalled();
    } finally {
      scan.mockRestore();
      resetCheckoutManifestCache();
    }
  });
});
