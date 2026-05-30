import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import os from "os";
import path from "path";
import { spawnSync } from "child_process";

const MERGE_BRANCH_COUNT = 12;

function execGit(args, cwd) {
  const result = spawnSync("git", args, {
    cwd,
    encoding: "utf8",
  });
  if (result.error) {
    throw result.error;
  }
  if (result.status !== 0) {
    throw new Error(`git ${args.join(" ")} failed: ${result.stderr || result.stdout}`);
  }
  return result;
}

function createMergeHeavyRepo() {
  const repoDir = fs.mkdtempSync(path.join(os.tmpdir(), "extra-empty-commit-integration-"));
  execGit(["init"], repoDir);
  execGit(["config", "user.name", "Test User"], repoDir);
  execGit(["config", "user.email", "test@example.com"], repoDir);

  fs.writeFileSync(path.join(repoDir, "README.md"), "# Integration Test\n");
  execGit(["add", "README.md"], repoDir);
  execGit(["commit", "-m", "initial commit"], repoDir);
  execGit(["branch", "-M", "main"], repoDir);

  for (let i = 0; i < MERGE_BRANCH_COUNT; i++) {
    execGit(["checkout", "-b", `feature-${i}`], repoDir);
    fs.writeFileSync(path.join(repoDir, `feature-${i}.txt`), `feature ${i}\n`);
    execGit(["add", `feature-${i}.txt`], repoDir);
    execGit(["commit", "-m", `feature commit ${i}`], repoDir);

    execGit(["checkout", "main"], repoDir);
    fs.writeFileSync(path.join(repoDir, `main-${i}.txt`), `main ${i}\n`);
    execGit(["add", `main-${i}.txt`], repoDir);
    execGit(["commit", "-m", `main commit ${i}`], repoDir);
    execGit(["merge", "--no-ff", `feature-${i}`, "-m", `merge feature-${i}`], repoDir);
  }

  return repoDir;
}

describe("extra_empty_commit git integration", () => {
  let repoDir;
  let originalCwd;
  let originalToken;
  let originalGithubRepo;
  let originalGithubServerUrl;
  let mockCore;
  let commandLog;

  beforeEach(() => {
    repoDir = createMergeHeavyRepo();
    originalCwd = process.cwd();
    process.chdir(repoDir);

    originalToken = process.env.GH_AW_CI_TRIGGER_TOKEN;
    originalGithubRepo = process.env.GITHUB_REPOSITORY;
    originalGithubServerUrl = process.env.GITHUB_SERVER_URL;
    process.env.GH_AW_CI_TRIGGER_TOKEN = "ghp_test_token_123";
    process.env.GITHUB_REPOSITORY = "test-owner/test-repo";
    process.env.GITHUB_SERVER_URL = "https://github.com";

    mockCore = {
      info: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
      debug: vi.fn(),
    };
    commandLog = [];

    global.core = mockCore;
    global.exec = {
      exec: vi.fn().mockImplementation(async (cmd, args = [], options = {}) => {
        if (cmd !== "git") {
          throw new Error(`unexpected command: ${cmd}`);
        }
        commandLog.push(args.join(" "));
        const gitSubcommand = args[0];
        const allowedSubcommands = new Set(["log", "remote", "fetch", "reset", "commit", "push"]);
        if (!allowedSubcommands.has(gitSubcommand)) {
          throw new Error(`unexpected git subcommand: ${gitSubcommand}`);
        }

        if (gitSubcommand === "log") {
          const result = execGit(args, repoDir);
          if (options.listeners && options.listeners.stdout) {
            options.listeners.stdout(Buffer.from(result.stdout));
          }
        }
        return 0;
      }),
    };

    delete require.cache[require.resolve("./extra_empty_commit.cjs")];
  });

  afterEach(() => {
    process.chdir(originalCwd);
    if (repoDir && fs.existsSync(repoDir)) {
      fs.rmSync(repoDir, { recursive: true, force: true });
    }
    if (originalToken !== undefined) {
      process.env.GH_AW_CI_TRIGGER_TOKEN = originalToken;
    } else {
      delete process.env.GH_AW_CI_TRIGGER_TOKEN;
    }
    if (originalGithubRepo !== undefined) {
      process.env.GITHUB_REPOSITORY = originalGithubRepo;
    } else {
      delete process.env.GITHUB_REPOSITORY;
    }
    if (originalGithubServerUrl !== undefined) {
      process.env.GITHUB_SERVER_URL = originalGithubServerUrl;
    } else {
      delete process.env.GITHUB_SERVER_URL;
    }
    delete global.core;
    delete global.exec;
    vi.clearAllMocks();
  });

  it("uses real git log output and does not count merge commits as empty commits", async () => {
    const { pushExtraEmptyCommit } = require("./extra_empty_commit.cjs");
    const rawLog = execGit(["log", "--max-count=60", "--format=%H %P", "HEAD"], repoDir).stdout;
    const rawMergeCount = rawLog
      .split("\n")
      .filter(Boolean)
      .filter(line => line.trim().split(/\s+/).length >= 3).length;
    expect(rawMergeCount).toBe(MERGE_BRANCH_COUNT);

    const result = await pushExtraEmptyCommit({
      branchName: "main",
      repoOwner: "test-owner",
      repoName: "test-repo",
    });

    expect(result).toEqual({ success: true });
    expect(commandLog.some(command => command.includes("--format=COMMIT:%H %P"))).toBe(true);
    expect(mockCore.warning).not.toHaveBeenCalledWith(expect.stringContaining("Cycle prevention"));

    const detailLogCall = mockCore.info.mock.calls.find(call => call[0].startsWith("Cycle check details:"));
    if (!detailLogCall || typeof detailLogCall[0] !== "string") {
      const infoMessages = mockCore.info.mock.calls.map(call => call[0]).filter(Boolean);
      throw new Error(`missing cycle-check detail log; captured ${infoMessages.length} info log(s): ${infoMessages.join(" | ")}`);
    }
    const detailMessage = detailLogCall[0];
    const analyzedMatch = detailMessage.match(/analyzed (\d+)/);
    const ignoredMergeMatch = detailMessage.match(/ignored (\d+) merge commit\(s\)/);
    const emptyMatch = detailMessage.match(/counted (\d+) empty non-merge commit\(s\)/);
    expect(analyzedMatch).toBeDefined();
    expect(ignoredMergeMatch).toBeDefined();
    expect(emptyMatch).toBeDefined();

    const analyzedCount = Number(analyzedMatch[1]);
    const ignoredMergeCount = Number(ignoredMergeMatch[1]);
    const emptyCount = Number(emptyMatch[1]);
    expect(analyzedCount).toBeGreaterThan(0);
    expect(ignoredMergeCount).toBe(MERGE_BRANCH_COUNT);
    expect(emptyCount).toBe(0);
  });
});
