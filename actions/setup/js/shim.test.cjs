import { afterEach, beforeEach, describe, expect, it } from "vitest";
import fs from "fs";
import os from "os";
import path from "path";
import { createRequire } from "module";

const require = createRequire(import.meta.url);
const shimPath = require.resolve("./shim.cjs");

describe("shim", () => {
  let originalGlobals;
  let originalEnv;
  let tempDir;

  beforeEach(() => {
    originalGlobals = {
      core: global.core,
      context: global.context,
      github: global.github,
      octokit: global.octokit,
      getOctokit: global.getOctokit,
      exec: global.exec,
      glob: global.glob,
      io: global.io,
      __original_require__: global.__original_require__,
    };

    originalEnv = {};
    for (const key of [
      "GITHUB_REPOSITORY",
      "GITHUB_EVENT_NAME",
      "GITHUB_EVENT_PATH",
      "GITHUB_RUN_ID",
      "GITHUB_RUN_NUMBER",
      "GITHUB_REF",
      "GITHUB_REF_NAME",
      "GITHUB_HEAD_REF",
      "GITHUB_BASE_REF",
      "GITHUB_WORKFLOW",
      "GITHUB_ACTION",
      "GITHUB_ACTION_PATH",
      "GITHUB_ACTOR",
      "GITHUB_JOB",
      "GITHUB_API_URL",
      "GITHUB_SERVER_URL",
      "GITHUB_GRAPHQL_URL",
      "GITHUB_WORKSPACE",
      "GITHUB_ENV",
      "GITHUB_OUTPUT",
      "GITHUB_PATH",
      "GITHUB_STATE",
      "GITHUB_STEP_SUMMARY",
      "GITHUB_TOKEN",
      "GH_TOKEN",
      "RUNNER_TEMP",
    ]) {
      originalEnv[key] = process.env[key];
      delete process.env[key];
    }

    delete global.core;
    delete global.context;
    delete global.github;
    delete global.octokit;
    delete global.getOctokit;
    delete global.exec;
    delete global.glob;
    delete global.io;
    delete global.__original_require__;

    delete require.cache[shimPath];
  });

  afterEach(() => {
    for (const [key, value] of Object.entries(originalEnv)) {
      if (value === undefined) {
        delete process.env[key];
      } else {
        process.env[key] = value;
      }
    }

    Object.assign(global, originalGlobals);

    delete require.cache[shimPath];

    if (tempDir) {
      fs.rmSync(tempDir, { recursive: true, force: true });
      tempDir = undefined;
    }
  });

  it("installs github-script style globals for plain node runs", async () => {
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-shim-test-"));
    const eventPath = path.join(tempDir, "event.json");
    fs.writeFileSync(eventPath, JSON.stringify({ issue: { number: 42 } }));

    process.env.RUNNER_TEMP = tempDir;
    process.env.GITHUB_REPOSITORY = "octo/repo";
    process.env.GITHUB_EVENT_NAME = "issues";
    process.env.GITHUB_EVENT_PATH = eventPath;
    process.env.GITHUB_RUN_ID = "123";
    process.env.GITHUB_RUN_NUMBER = "7";
    process.env.GITHUB_REF = "refs/heads/main";
    process.env.GITHUB_REF_NAME = "main";
    process.env.GITHUB_HEAD_REF = "feature";
    process.env.GITHUB_BASE_REF = "main";
    process.env.GITHUB_WORKFLOW = "shim-test";
    process.env.GITHUB_ACTION = "shim";
    process.env.GITHUB_ACTOR = "copilot";
    process.env.GITHUB_JOB = "test";
    process.env.GITHUB_WORKSPACE = tempDir;

    require("./shim.cjs");

    expect(global.core).toBeTruthy();
    expect(global.context).toBeTruthy();
    expect(global.github).toBeTruthy();
    expect(global.octokit).toBe(global.github);
    expect(global.getOctokit).toBeTypeOf("function");
    expect(global.exec).toBeTruthy();
    expect(global.glob).toBeTruthy();
    expect(global.io).toBeTruthy();
    expect(global.__original_require__).toBeTypeOf("function");
    expect(global.__original_require__.resolve("./shim.cjs")).toBe(shimPath);

    expect(global.context.repo).toEqual({ owner: "octo", repo: "repo" });
    expect(global.context.issue).toEqual({ number: 42 });
    expect(global.context.runId).toBe(123);
    expect(global.context.runNumber).toBe(7);
    expect(global.context.refName).toBe("main");
    expect(global.context.headRef).toBe("feature");
    expect(global.context.baseRef).toBe("main");

    global.core.exportVariable("SHIM_EXPORT_TEST", "ok");
    expect(process.env.SHIM_EXPORT_TEST).toBe("ok");

    await global.core.summary.addRaw("shim summary works").write();
    expect(fs.readFileSync(process.env.GITHUB_STEP_SUMMARY, "utf8")).toContain("shim summary works");

    global.core.setOutput("shim-output", "value");
    expect(fs.readFileSync(process.env.GITHUB_OUTPUT, "utf8")).toContain("shim-output<<");

    global.core.setFailed("shim failure");
    expect(process.exitCode).toBe(1);
  });
});
