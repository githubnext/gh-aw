import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createRequire } from "module";
import fs from "fs";
import os from "os";
import path from "path";
import { spawnSync } from "child_process";
import { useExecutionProvenanceFixture } from "./test_execution_provenance.js";

useExecutionProvenanceFixture();
const require = createRequire(import.meta.url);
const { buildCheckoutManifest, main: buildManifestFromEnv } = require("./build_checkout_manifest.cjs");
const { _resetCache } = require("./checkout_manifest.cjs");
const { createHandlers } = require("./safe_outputs_handlers.cjs");
const { validateItem, resetValidationConfigCache } = require("./safe_output_type_validator.cjs");

function git(cwd, args, allowFailure = false) {
  const result = spawnSync("git", ["-c", "core.hooksPath=/dev/null", "-c", "commit.gpgsign=false", ...args], { cwd, encoding: "utf8" });
  if (result.error) throw result.error;
  if (result.status !== 0 && !allowFailure) throw new Error(`git ${args.join(" ")}: ${result.stderr}`);
  return result;
}

describe("side-repository selected base end to end", () => {
  let root;
  let originalCwd;
  const transports = new Set();
  beforeEach(() => {
    root = fs.mkdtempSync(path.join(os.tmpdir(), "side-repo-base-"));
    originalCwd = process.cwd();
    global.core = {
      debug: vi.fn(),
      info: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
      setFailed: vi.fn(),
      setOutput: vi.fn(),
      startGroup: vi.fn(),
      endGroup: vi.fn(),
      summary: { addRaw: vi.fn().mockReturnThis(), write: vi.fn().mockResolvedValue() },
    };
    global.context = { repo: { owner: "host", repo: "workflows" }, eventName: "workflow_dispatch", payload: {} };
    global.github = {
      rest: {
        repos: { get: vi.fn().mockResolvedValue({ data: { default_branch: "main" } }), getBranch: vi.fn().mockResolvedValue({ data: {} }) },
        pulls: { create: vi.fn().mockResolvedValue({ data: { number: 42, html_url: "https://github.com/side/target/pull/42", head: { sha: "a".repeat(40) } } }) },
        issues: {},
      },
    };
    global.exec = {
      async exec(command, args, options = {}) {
        expect(command).toBe("git");
        const result = git(options.cwd || process.cwd(), args, options.ignoreReturnCode);
        return result.status;
      },
      async getExecOutput(command, args, options = {}) {
        expect(command).toBe("git");
        const result = git(options.cwd || process.cwd(), args, options.ignoreReturnCode);
        return { exitCode: result.status, stdout: result.stdout, stderr: result.stderr };
      },
    };
    vi.stubEnv("GITHUB_REPOSITORY", "host/workflows");
    vi.stubEnv("GITHUB_BASE_REF", "");
    vi.stubEnv("GITHUB_SHA", "");
    vi.stubEnv("GH_AW_PROMPTS_DIR", path.join(__dirname, "../md"));
    _resetCache();
  });
  afterEach(() => {
    process.chdir(originalCwd);
    _resetCache();
    resetValidationConfigCache();
    for (const file of transports) fs.rmSync(file, { force: true });
    transports.clear();
    fs.rmSync(root, { recursive: true, force: true });
    delete global.core;
    delete global.context;
    delete global.github;
    delete global.exec;
  });

  it.each([{ base_branch: "release/forged" }, { base_branch: "release/../main" }, { base_branch: "main", base: "release/1" }])("rejects forged or inconsistent base metadata before fetching or creating a PR: %j", async metadata => {
    const { main } = require("./create_pull_request.cjs");
    const handler = await main({ base_branch: "main", allow_empty: true });
    const execute = vi.spyOn(global.exec, "exec");
    const result = await handler({ title: "Fix release behavior", body: "A real release fix.", branch: "feature/fix", ...metadata }, {});
    expect(result.success).toBe(false);
    expect(result.error).toMatch(/not authorized|Invalid patch base|does not match/);
    expect(execute).not.toHaveBeenCalled();
    expect(global.github.rest.pulls.create).not.toHaveBeenCalled();
  });

  it.each(["static", "runtime-ref", "runtime-config", "explicit-base"].flatMap(selection => ["am", "bundle"].map(format => [selection, format])))(
    "carries %s through %s generation, canonical collection and a separate apply checkout",
    async (selection, format) => {
      const remote = path.join(root, "origin.git");
      const agentWorkspace = path.join(root, "agent");
      const agent = path.join(agentWorkspace, "target");
      const applyWorkspace = path.join(root, "apply");
      const apply = path.join(applyWorkspace, "target");
      fs.mkdirSync(agent, { recursive: true });
      fs.mkdirSync(applyWorkspace);
      git(root, ["init", "--bare", "-b", "main", remote]);
      git(agent, ["init", "-b", "main"]);
      git(agent, ["config", "user.name", "Test"]);
      git(agent, ["config", "user.email", "test@example.com"]);
      git(agent, ["remote", "add", "origin", remote]);
      fs.writeFileSync(path.join(agent, "shared.txt"), "shared\n");
      git(agent, ["add", "."]);
      git(agent, ["commit", "-m", "shared history"]);
      git(agent, ["checkout", "-b", "release/1"]);
      fs.writeFileSync(path.join(agent, "release-only.txt"), "existing release divergence\n");
      git(agent, ["add", "."]);
      git(agent, ["commit", "-m", "release history"]);
      const releaseBase = git(agent, ["rev-parse", "HEAD"]).stdout.trim();
      git(agent, ["push", "origin", "release/1"]);
      git(agent, ["checkout", "main"]);
      fs.writeFileSync(path.join(agent, "main-only.txt"), "existing default divergence\n");
      git(agent, ["add", "."]);
      git(agent, ["commit", "-m", "main history"]);
      git(agent, ["push", "origin", "main"]);
      git(agent, ["symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main"]);
      git(agent, ["checkout", "release/1"]);

      vi.stubEnv("GITHUB_WORKSPACE", agentWorkspace);
      vi.stubEnv("RUNNER_TEMP", path.join(root, "agent-temp"));
      vi.stubEnv("GH_AW_CHECKOUT_MANIFEST_COUNT", "1");
      vi.stubEnv("GH_AW_CHECKOUT_REPO_0", "side/target");
      vi.stubEnv("GH_AW_CHECKOUT_PATH_0", "target");
      // Simulate a pre-step output which is resolved only in the agent job.
      vi.stubEnv("GH_AW_CHECKOUT_REF_0", ["explicit-base", "runtime-config"].includes(selection) ? "" : "release/1");
      const { manifestPath } = await buildManifestFromEnv();
      vi.stubEnv("GH_AW_CHECKOUT_MANIFEST", manifestPath);
      _resetCache();
      const branch = `fix/${selection}-${format}-${path.basename(root)}`;
      git(agent, ["checkout", "-b", branch]);
      fs.writeFileSync(path.join(agent, "fix.txt"), "only the agent change\n");
      git(agent, ["add", "."]);
      git(agent, ["commit", "-m", "fix release behavior"]);
      const config = {
        "target-repo": "side/target",
        patch_format: format,
        ...(selection === "runtime-config" ? { base_branch: "release/1" } : {}),
        ...(selection === "explicit-base" ? { allowed_base_branches: ["release/*"] } : {}),
      };
      let produced;
      const handlers = createHandlers(
        { debug: vi.fn() },
        entry => {
          produced = entry;
        },
        { create_pull_request: config }
      );
      const result = await handlers.createPullRequestHandler({
        title: "Fix release behavior",
        body: "Limit the PR to the intended release fix.",
        branch,
        ...(selection === "explicit-base" ? { base: "release/1" } : {}),
      });
      expect(result.isError, JSON.stringify(result)).not.toBe(true);
      const response = JSON.parse(result.content[0].text);
      transports.add(response.patch.path);
      if (response.bundle) transports.add(response.bundle.path);
      expect(fs.readFileSync(response.patch.path, "utf8")).not.toContain("release history");
      expect(produced).toMatchObject({ base_branch: "release/1", base_commit: releaseBase });
      if (selection !== "explicit-base") expect(produced).not.toHaveProperty("base");

      // This is the same canonicalization engine used by collect_ndjson_output.
      vi.stubEnv(
        "GH_AW_VALIDATION_CONFIG",
        JSON.stringify({
          create_pull_request: {
            fields: {
              title: { type: "string" },
              body: { type: "string" },
              branch: { type: "string" },
              base: { type: "string" },
              base_branch: { type: "string", maxLength: 128 },
              base_commit: { type: "string", pattern: "^[a-fA-F0-9]{40}$" },
            },
          },
        })
      );
      resetValidationConfigCache();
      const canonical = validateItem(produced, 1);
      expect(canonical.isValid, canonical.error).toBe(true);
      const message = JSON.parse(JSON.stringify(canonical.normalizedItem));
      expect(message.base_branch).toBe("release/1");
      expect(message.base_commit).toBe(releaseBase);

      git(applyWorkspace, ["clone", remote, "target"]);
      git(apply, ["config", "user.name", "Test"]);
      git(apply, ["config", "user.email", "test@example.com"]);
      if (selection === "static") git(apply, ["checkout", "release/1"]);
      vi.stubEnv("GITHUB_WORKSPACE", applyWorkspace);
      vi.stubEnv("RUNNER_TEMP", path.join(root, "apply-temp"));
      const applyManifest = buildCheckoutManifest([{ repository: "side/target", path: "target", ref: selection === "static" ? "release/1" : "" }]);
      vi.stubEnv("GH_AW_CHECKOUT_MANIFEST", applyManifest.manifestPath);
      vi.stubEnv("GH_AW_CHECKOUT_REF_0", "");
      _resetCache();
      const { main } = require("./create_pull_request.cjs");
      const handler = await main({
        "target-repo": "side/target",
        checkout_mapping: { "side/target": "target" },
        preserve_branch_name: true,
        signed_commits: false,
        fallback_as_issue: false,
        max_patch_files: 1,
        ...(selection === "static" ? { stacked: false } : { base_branch: "main", allowed_base_branches: ["release/*"], stacked: selection === "explicit-base" }),
      });
      const applied = await handler(message, {});
      expect(applied.success, JSON.stringify(applied)).toBe(true);
      expect(global.github.rest.pulls.create).toHaveBeenCalledWith(expect.objectContaining({ base: "release/1", head: branch }));
      git(apply, ["fetch", "origin", branch]);
      expect(git(apply, ["diff", "--name-only", `origin/release/1...origin/${branch}`]).stdout.trim()).toBe("fix.txt");
      expect(git(apply, ["rev-list", "--count", `origin/release/1..origin/${branch}`]).stdout.trim()).toBe("1");
      expect(git(apply, ["show", `origin/${branch}:release-only.txt`]).stdout).toBe("existing release divergence\n");
      expect(git(apply, ["cat-file", "-e", `origin/${branch}:main-only.txt`], true).status).not.toBe(0);

      if (selection === "runtime-ref") {
        const previousHead = git(agent, ["rev-parse", "HEAD"]).stdout.trim();
        // The next agent checkout has the existing PR head available locally.
        git(agent, ["update-ref", `refs/remotes/origin/${branch}`, previousHead]);
        fs.writeFileSync(path.join(agent, "fix.txt"), "follow-up agent change\n");
        git(agent, ["commit", "-am", "refine the release fix"]);
        vi.stubEnv("GITHUB_WORKSPACE", agentWorkspace);
        vi.stubEnv("GH_AW_CHECKOUT_MANIFEST", manifestPath);
        _resetCache();
        let pushProduced;
        const pushHandlers = createHandlers(
          { debug: vi.fn() },
          entry => {
            pushProduced = entry;
          },
          {
            push_to_pull_request_branch: { "target-repo": "side/target", target: "*", patch_format: format },
          }
        );
        const pushResult = await pushHandlers.pushToPullRequestBranchHandler({ repo: "side/target", branch, pull_request_number: 42, message: "Refine the release fix." });
        expect(pushResult.isError, JSON.stringify(pushResult)).not.toBe(true);
        const pushResponse = JSON.parse(pushResult.content[0].text);
        transports.add(pushResponse.patch.path);
        if (pushResponse.bundle) transports.add(pushResponse.bundle.path);
        expect(pushProduced.base_branch).toBe("release/1");
        vi.stubEnv(
          "GH_AW_VALIDATION_CONFIG",
          JSON.stringify({
            push_to_pull_request_branch: {
              fields: {
                message: { type: "string" },
                branch: { type: "string" },
                repo: { type: "string" },
                pull_request_number: { issueOrPRNumber: true },
                base_branch: { type: "string", maxLength: 128 },
                base_commit: { type: "string", pattern: "^[a-fA-F0-9]{40}$" },
              },
            },
          })
        );
        resetValidationConfigCache();
        const pushCanonical = validateItem(pushProduced, 1);
        expect(pushCanonical.isValid, pushCanonical.error).toBe(true);
        expect(pushCanonical.normalizedItem.base_branch).toBe("release/1");
        vi.stubEnv("GITHUB_WORKSPACE", applyWorkspace);
        vi.stubEnv("GH_AW_CHECKOUT_MANIFEST", applyManifest.manifestPath);
        _resetCache();
        global.github.rest.pulls.get = vi.fn().mockResolvedValue({
          data: {
            number: 42,
            state: "open",
            title: "Fix release behavior",
            labels: [],
            head: { ref: branch, sha: previousHead, repo: { full_name: "side/target", fork: false } },
            base: { ref: "release/1", sha: releaseBase, repo: { full_name: "side/target" } },
          },
        });
        const { main: createPushHandler } = require("./push_to_pull_request_branch.cjs");
        const pushHandler = await createPushHandler({
          "target-repo": "side/target",
          target: "*",
          checkout_mapping: { "side/target": "target" },
          base_branch: "main",
          signed_commits: false,
          check_branch_protection: false,
          fallback_as_pull_request: false,
        });
        const pushed = await pushHandler(JSON.parse(JSON.stringify(pushCanonical.normalizedItem)), {});
        expect(pushed.success, JSON.stringify(pushed)).toBe(true);
        git(apply, ["fetch", "origin", branch]);
        expect(git(apply, ["show", `origin/${branch}:fix.txt`]).stdout).toBe("follow-up agent change\n");
        expect(git(apply, ["diff", "--name-only", `origin/release/1...origin/${branch}`]).stdout.trim()).toBe("fix.txt");
        expect(global.github.rest.pulls.create).toHaveBeenCalledTimes(1);
      }
    },
    60000
  );
});
