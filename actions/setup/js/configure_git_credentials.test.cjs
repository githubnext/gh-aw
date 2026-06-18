import { afterEach, beforeEach, describe, expect, it } from "vitest";
import fs from "fs";
import os from "os";
import path from "path";
import { spawnSync } from "child_process";

// Path to the shell script under test.
const SCRIPT_PATH = path.join(__dirname, "..", "sh", "configure_git_credentials.sh");

function createTempDir(prefix) {
  return fs.mkdtempSync(path.join(os.tmpdir(), prefix));
}

function removeDir(dir) {
  if (dir && fs.existsSync(dir)) {
    fs.rmSync(dir, { recursive: true, force: true });
  }
}

/**
 * Run configure_git_credentials.sh in an isolated HOME so that the global git
 * config it writes does not affect the developer's real ~/.gitconfig.
 */
function runScript(env) {
  return spawnSync("sh", [SCRIPT_PATH], {
    encoding: "utf8",
    env: { ...process.env, ...env },
  });
}

function readSafeDirectories(home) {
  const result = spawnSync("git", ["config", "--global", "--get-all", "safe.directory"], {
    encoding: "utf8",
    env: { ...process.env, HOME: home },
  });
  if (result.status !== 0) {
    return [];
  }
  return result.stdout.split("\n").filter(Boolean);
}

describe("configure_git_credentials.sh checkout manifest trust", () => {
  const tempDirs = [];

  function tempDir(prefix) {
    const dir = createTempDir(prefix);
    tempDirs.push(dir);
    return dir;
  }

  afterEach(() => {
    while (tempDirs.length > 0) {
      removeDir(tempDirs.pop());
    }
  });

  function setup(manifest) {
    const root = tempDir("cfg-git-creds-");
    const home = path.join(root, "home");
    const workspace = path.join(root, "ws");
    const runnerTemp = path.join(root, "runner");
    const safeOutputs = path.join(runnerTemp, "gh-aw", "safeoutputs");
    fs.mkdirSync(home, { recursive: true });
    fs.mkdirSync(workspace, { recursive: true });
    fs.mkdirSync(safeOutputs, { recursive: true });
    if (manifest !== undefined) {
      fs.writeFileSync(path.join(safeOutputs, "checkout-manifest.json"), JSON.stringify(manifest), "utf8");
    }
    return { home, workspace, runnerTemp };
  }

  it("trusts cross-repo checkout subdirectories listed in the manifest", () => {
    const { home, workspace, runnerTemp } = setup({
      "owner/repo": { repository: "owner/repo", path: "github", default_branch: "main" },
      "owner/tools": { repository: "owner/tools", path: "vendor/tools", default_branch: "main" },
    });

    const result = runScript({ HOME: home, GITHUB_WORKSPACE: workspace, RUNNER_TEMP: runnerTemp });
    expect(result.status).toBe(0);

    const entries = readSafeDirectories(home);
    expect(entries).toContain(workspace);
    expect(entries).toContain(path.join(workspace, "github"));
    expect(entries).toContain(path.join(workspace, "vendor", "tools"));
  });

  it("skips manifest entries with an empty path", () => {
    const { home, workspace, runnerTemp } = setup({
      "owner/repo": { repository: "owner/repo", path: "", default_branch: "main" },
    });

    const result = runScript({ HOME: home, GITHUB_WORKSPACE: workspace, RUNNER_TEMP: runnerTemp });
    expect(result.status).toBe(0);

    const entries = readSafeDirectories(home);
    // Only the workspace itself is trusted; the empty-path entry adds nothing extra.
    expect(entries).toEqual([workspace]);
  });

  it("rejects manifest paths that escape the workspace (path traversal)", () => {
    const { home, workspace, runnerTemp } = setup({
      "evil/repo": { repository: "evil/repo", path: "../../escape", default_branch: "main" },
    });

    const result = runScript({ HOME: home, GITHUB_WORKSPACE: workspace, RUNNER_TEMP: runnerTemp });
    expect(result.status).toBe(0);

    const entries = readSafeDirectories(home);
    expect(entries).toEqual([workspace]);
    expect(entries.some(e => e.includes("escape"))).toBe(false);
  });

  it("honors GH_AW_CHECKOUT_MANIFEST override", () => {
    const root = tempDir("cfg-git-creds-override-");
    const home = path.join(root, "home");
    const workspace = path.join(root, "ws");
    fs.mkdirSync(home, { recursive: true });
    fs.mkdirSync(workspace, { recursive: true });
    const manifestPath = path.join(root, "custom-manifest.json");
    fs.writeFileSync(manifestPath, JSON.stringify({ "owner/repo": { repository: "owner/repo", path: "sub", default_branch: "main" } }), "utf8");

    const result = runScript({ HOME: home, GITHUB_WORKSPACE: workspace, GH_AW_CHECKOUT_MANIFEST: manifestPath });
    expect(result.status).toBe(0);

    const entries = readSafeDirectories(home);
    expect(entries).toContain(path.join(workspace, "sub"));
  });

  it("succeeds when no manifest is present", () => {
    const { home, workspace, runnerTemp } = setup(undefined);

    const result = runScript({ HOME: home, GITHUB_WORKSPACE: workspace, RUNNER_TEMP: runnerTemp });
    expect(result.status).toBe(0);

    const entries = readSafeDirectories(home);
    expect(entries).toEqual([workspace]);
  });
});
