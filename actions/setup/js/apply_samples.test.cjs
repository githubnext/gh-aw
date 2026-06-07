// @ts-check
//
// apply_samples.test.cjs
//
// Smoke test for the deterministic samples replay driver. Spawns the
// driver as a subprocess (so it actually launches the real MCP server) and
// asserts that:
//   - the driver exits 0
//   - the MCP server appends the expected JSONL entry to GH_AW_SAFE_OUTPUTS
//   - the synthetic agent-stdio log includes a `terminal_reason: completed` marker
//
// Tests intentionally use the simplest safe-output tool (`create_issue`) so we
// do not need to set up a git working tree for patch sidecars.

import { describe, it, expect, beforeAll } from "vitest";
import { spawnSync } from "child_process";
import { createRequire } from "module";
import fs from "fs";
import path from "path";
import os from "os";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const driverPath = path.join(__dirname, "apply_samples.cjs");
const require = createRequire(import.meta.url);

function makeTempDir(prefix) {
  return fs.mkdtempSync(path.join(os.tmpdir(), prefix));
}

function git(args, cwd) {
  const r = spawnSync("git", args, { cwd, encoding: "utf8" });
  if (r.status !== 0) {
    throw new Error(`git ${args.join(" ")} failed: ${r.stderr || r.stdout}`);
  }
  return r.stdout;
}

function initRepo(dir, defaultBranch) {
  git(["init", "-q", "-b", defaultBranch], dir);
  git(["config", "user.email", "ghaw-test@example.com"], dir);
  git(["config", "user.name", "ghaw test"], dir);
  fs.writeFileSync(path.join(dir, "README.md"), "# seed\n");
  git(["add", "."], dir);
  git(["commit", "-q", "-m", "seed"], dir);
}

describe.sequential("apply_samples.cjs", () => {
  let tempDir;
  let configPath;
  let outputsPath;
  let logPath;

  beforeAll(() => {
    tempDir = makeTempDir("gh-aw-apply-samples-");
    configPath = path.join(tempDir, "config.json");
    outputsPath = path.join(tempDir, "outputs.jsonl");
    logPath = path.join(tempDir, "agent-stdio.log");

    // Minimal safe-outputs config enabling only the `create_issue` tool. The
    // bootstrap loader keys off the snake-case keys present here.
    fs.writeFileSync(
      configPath,
      JSON.stringify({
        create_issue: { max: 1 },
      })
    );
  });

  it("replays a create_issue sample through the real MCP server and emits a completed marker", () => {
    const samples = [
      {
        tool: "create_issue",
        arguments: {
          title: "Deterministic sample issue",
          body: "This issue was emitted by the apply_samples driver during a unit test.",
        },
      },
    ];

    const result = spawnSync(process.execPath, [driverPath], {
      env: {
        ...process.env,
        GH_AW_SAMPLES: JSON.stringify(samples),
        GH_AW_SAFE_OUTPUTS_CONFIG_PATH: configPath,
        GH_AW_SAFE_OUTPUTS: outputsPath,
        GH_AW_AGENT_STDIO_LOG: logPath,
      },
      encoding: "utf8",
      timeout: 15000,
    });

    if (result.status !== 0) {
      // Surface stderr so failures are diagnosable in CI.
      throw new Error(`driver exited with status ${result.status}\nstderr:\n${result.stderr}\nstdout:\n${result.stdout}`);
    }

    expect(fs.existsSync(outputsPath)).toBe(true);
    const outputLines = fs
      .readFileSync(outputsPath, "utf8")
      .split("\n")
      .filter(line => line.trim().length > 0);
    expect(outputLines.length).toBeGreaterThanOrEqual(1);

    const firstEntry = JSON.parse(outputLines[0]);
    expect(firstEntry.type).toBe("create_issue");
    expect(firstEntry.title).toBe("Deterministic sample issue");

    expect(fs.existsSync(logPath)).toBe(true);
    const logText = fs.readFileSync(logPath, "utf8");
    expect(logText).toContain("terminal_reason");
    expect(logText).toContain("completed");
  });

  it("exits cleanly when GH_AW_SAMPLES is empty", () => {
    const result = spawnSync(process.execPath, [driverPath], {
      env: {
        ...process.env,
        GH_AW_SAMPLES: "[]",
        GH_AW_SAFE_OUTPUTS_CONFIG_PATH: configPath,
        GH_AW_SAFE_OUTPUTS: outputsPath,
        GH_AW_AGENT_STDIO_LOG: path.join(tempDir, "empty-log.log"),
      },
      encoding: "utf8",
      timeout: 10000,
    });

    expect(result.status).toBe(0);
    const logText = fs.readFileSync(path.join(tempDir, "empty-log.log"), "utf8");
    expect(logText).toContain("terminal_reason");
  });

  // Defense in depth: an older compiler that marshaled a nil Go slice would
  // emit `null` into GH_AW_SAMPLES. Newer drivers must tolerate that and
  // treat it as "no samples", not crash with `must be a JSON array`.
  it("exits cleanly when GH_AW_SAMPLES is the literal `null`", () => {
    const logPath = path.join(tempDir, "null-log.log");
    const result = spawnSync(process.execPath, [driverPath], {
      env: {
        ...process.env,
        GH_AW_SAMPLES: "null",
        GH_AW_SAFE_OUTPUTS_CONFIG_PATH: configPath,
        GH_AW_SAFE_OUTPUTS: outputsPath,
        GH_AW_AGENT_STDIO_LOG: logPath,
      },
      encoding: "utf8",
      timeout: 10000,
    });

    if (result.status !== 0) {
      throw new Error(`driver exited with status ${result.status}\nstderr:\n${result.stderr}\nstdout:\n${result.stdout}`);
    }
    expect(result.stderr).toContain("GH_AW_SAMPLES is null");
    const logText = fs.readFileSync(logPath, "utf8");
    expect(logText).toContain("terminal_reason");
  });
});

describe("apply_samples.cjs sendJsonRpc", () => {
  const { sendJsonRpc } = require("./apply_samples.cjs");

  async function* fromLines(lines) {
    for (const line of lines) {
      yield line;
    }
  }

  it("skips non-JSON stdout lines until a JSON-RPC response arrives", async () => {
    const writes = [];
    const stdin = {
      write: chunk => writes.push(chunk),
    };
    const response = await sendJsonRpc({}, stdin, { jsonrpc: "2.0", id: 99, method: "tools/call", params: {} }, fromLines(["[debug] Executing git command: git status", '{"jsonrpc":"2.0","id":99,"result":{"ok":true}}']));

    expect(writes.length).toBe(1);
    expect(writes[0]).toContain('"id":99');
    expect(response).toEqual({ jsonrpc: "2.0", id: 99, result: { ok: true } });
  });

  it("throws a helpful error for malformed JSON lines that look like protocol frames", async () => {
    const stdin = { write: () => {} };
    await expect(sendJsonRpc({}, stdin, { jsonrpc: "2.0", id: 4, method: "initialize", params: {} }, fromLines(["{not-json"]))).rejects.toThrow("failed to parse MCP JSON-RPC response");
  });
});

describe("apply_samples.cjs preStagePatch (create_pull_request / push_to_pull_request_branch)", () => {
  // Load the module under test directly so we can drive preStagePatch in
  // isolation against a real, throwaway git working tree. This is the
  // critical code path that turns a `patch` sidecar on a sample entry into
  // a real branch + commit that the downstream MCP `create_pull_request`
  // handler (which derives a git diff) can act on.
  const { preStagePatch } = require("./apply_samples.cjs");

  /**
   * Build a unified diff that adds a brand-new file. Synthetic but realistic.
   */
  function newFileDiff(filePath, contents) {
    const lines = contents.split("\n");
    // Strip trailing empty element produced by a terminating "\n" so the
    // hunk header line count matches what git apply expects.
    if (lines[lines.length - 1] === "") lines.pop();
    const body = lines.map(l => "+" + l).join("\n");
    return `diff --git a/${filePath} b/${filePath}\n` + `new file mode 100644\n` + `index 0000000..1111111\n` + `--- /dev/null\n` + `+++ b/${filePath}\n` + `@@ -0,0 +1,${lines.length} @@\n` + body + "\n";
  }

  it("checks out the requested branch and commits the patch on it (create_pull_request)", () => {
    const workspace = makeTempDir("gh-aw-prestage-cpr-");
    initRepo(workspace, "main");

    const branchName = "feat/gh-aw-sample-branch";
    const fileToAdd = "sample-feature.txt";
    const fileBody = "hello from a deterministic sample\nsecond line\n";
    const entry = {
      tool: "create_pull_request",
      arguments: {
        title: "Sample PR",
        body: "Sample PR body",
        branch: branchName,
      },
      sidecars: { patch: newFileDiff(fileToAdd, fileBody) },
    };

    // GH_AW_CUSTOM_BASE_BRANCH steers preStagePatch to check out the right
    // base ref inside our fresh repo (default is GITHUB_BASE_REF / "main").
    const prev = process.env.GH_AW_CUSTOM_BASE_BRANCH;
    process.env.GH_AW_CUSTOM_BASE_BRANCH = "main";
    try {
      preStagePatch(entry, 0, workspace);
    } finally {
      if (prev === undefined) delete process.env.GH_AW_CUSTOM_BASE_BRANCH;
      else process.env.GH_AW_CUSTOM_BASE_BRANCH = prev;
    }

    // 1. Branch name on the entry is preserved (driver must forward it to MCP).
    expect(entry.arguments.branch).toBe(branchName);

    // 2. The named branch exists in the working repo.
    const branches = git(["branch", "--list", branchName], workspace).trim();
    expect(branches).toContain(branchName);

    // 3. Current HEAD is that branch.
    const head = git(["rev-parse", "--abbrev-ref", "HEAD"], workspace).trim();
    expect(head).toBe(branchName);

    // 4. The patch was applied AND committed (not just sitting in the worktree).
    const status = git(["status", "--porcelain"], workspace).trim();
    expect(status).toBe("");
    expect(fs.existsSync(path.join(workspace, fileToAdd))).toBe(true);
    expect(fs.readFileSync(path.join(workspace, fileToAdd), "utf8")).toBe(fileBody);

    // 5. The commit message identifies the sample so failures are diagnosable.
    const lastMsg = git(["log", "-1", "--pretty=%s"], workspace).trim();
    expect(lastMsg).toMatch(/gh-aw sample 1: create_pull_request/);

    // 6. The new file shows up as a real diff against the base branch — this is
    // precisely what the downstream MCP create_pull_request handler will read.
    const diff = git(["diff", "main..." + branchName, "--", fileToAdd], workspace);
    expect(diff).toContain("+hello from a deterministic sample");
  });

  it("defaults the branch name to gh-aw-sample-<i+1> when none is supplied", () => {
    const workspace = makeTempDir("gh-aw-prestage-default-");
    initRepo(workspace, "main");

    const entry = {
      tool: "push_to_pull_request_branch",
      arguments: {
        body: "Sample push body",
        // branch intentionally omitted — driver should synthesize one.
      },
      sidecars: { patch: newFileDiff("push-feature.txt", "from push sample\n") },
    };

    const prev = process.env.GH_AW_CUSTOM_BASE_BRANCH;
    process.env.GH_AW_CUSTOM_BASE_BRANCH = "main";
    try {
      preStagePatch(entry, 2, workspace);
    } finally {
      if (prev === undefined) delete process.env.GH_AW_CUSTOM_BASE_BRANCH;
      else process.env.GH_AW_CUSTOM_BASE_BRANCH = prev;
    }

    // Index in preStagePatch is zero-based; the default uses i+1 → "gh-aw-sample-3".
    expect(entry.arguments.branch).toBe("gh-aw-sample-3");
    const head = git(["rev-parse", "--abbrev-ref", "HEAD"], workspace).trim();
    expect(head).toBe("gh-aw-sample-3");
    expect(fs.existsSync(path.join(workspace, "push-feature.txt"))).toBe(true);
  });

  it("is a no-op when the sample tool isn't in the patch-sidecar set", () => {
    // We assert this at the driver level (PATCH_SIDECAR_TOOLS gate in main()),
    // but preStagePatch itself should also be a no-op when called with an
    // entry that has no patch sidecar — protecting against misuse.
    const workspace = makeTempDir("gh-aw-prestage-noop-");
    initRepo(workspace, "main");

    const entry = {
      tool: "create_issue",
      arguments: { title: "x", body: "y" },
    };
    preStagePatch(entry, 0, workspace);

    // Still on main, no extra commits, no new files.
    expect(git(["rev-parse", "--abbrev-ref", "HEAD"], workspace).trim()).toBe("main");
    const log = git(["log", "--pretty=%s"], workspace).trim().split("\n");
    expect(log).toEqual(["seed"]);
  });
});
