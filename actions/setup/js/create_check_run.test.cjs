// @ts-check
import { describe, it, expect, beforeEach, afterEach } from "vitest";

describe("create_check_run", () => {
  let mockCore;
  let mockGithub;
  let mockContext;
  let originalGlobals;
  let originalEnv;

  const makeChecksCreate = (onCall) => {
    return async (params) => {
      onCall(params);
      return {
        data: {
          id: 77313480284,
          html_url: `https://github.com/test-owner/test-repo/runs/77313480284`,
        },
      };
    };
  };

  beforeEach(() => {
    originalGlobals = {
      core: global.core,
      github: global.github,
      context: global.context,
      getOctokit: global.getOctokit,
    };
    originalEnv = { ...process.env };

    mockCore = {
      debug: () => {},
      info: () => {},
      warning: () => {},
      error: () => {},
      setOutput: () => {},
      setFailed: () => {},
    };

    mockGithub = {
      rest: {
        checks: {
          create: async (params) => ({
            data: {
              id: 77313480284,
              html_url: `https://github.com/test-owner/test-repo/runs/77313480284`,
            },
          }),
        },
      },
    };

    mockContext = {
      eventName: "push",
      runId: 12345,
      repo: {
        owner: "test-owner",
        repo: "test-repo",
      },
      sha: "abc123def456",
      payload: {},
    };

    global.core = mockCore;
    global.github = mockGithub;
    global.context = mockContext;
    global.getOctokit = () => mockGithub;

    delete process.env.GITHUB_SHA;
    delete process.env.GITHUB_WORKFLOW;
    delete process.env.GH_AW_SAFE_OUTPUTS_STAGED;
  });

  afterEach(() => {
    global.core = originalGlobals.core;
    global.github = originalGlobals.github;
    global.context = originalGlobals.context;
    global.getOctokit = originalGlobals.getOctokit;
    Object.keys(process.env).forEach((k) => {
      if (!(k in originalEnv)) delete process.env[k];
    });
    Object.assign(process.env, originalEnv);
  });

  describe("SHA resolution", () => {
    it("uses PR head SHA (not GITHUB_SHA) on pull_request events", async () => {
      const prHeadSha = "pr-head-sha-abc123";
      process.env.GITHUB_SHA = "merge-commit-sha-xyz789";
      mockContext.eventName = "pull_request";
      mockContext.payload = {
        pull_request: {
          head: { sha: prHeadSha },
        },
      };

      let capturedParams;
      mockGithub.rest.checks.create = makeChecksCreate((p) => {
        capturedParams = p;
      });

      const { main } = require("./create_check_run.cjs");
      const handler = await main({ name: "Test Check", max: 10 });
      await handler({ type: "create_check_run", conclusion: "success", title: "All good", summary: "Tests passed." }, {});

      expect(capturedParams.head_sha).toBe(prHeadSha);
      expect(capturedParams.head_sha).not.toBe("merge-commit-sha-xyz789");
    });

    it("falls back to GITHUB_SHA on push events (no PR payload)", async () => {
      process.env.GITHUB_SHA = "push-sha-abc123";
      mockContext.eventName = "push";
      mockContext.payload = {};

      let capturedParams;
      mockGithub.rest.checks.create = makeChecksCreate((p) => {
        capturedParams = p;
      });

      const { main } = require("./create_check_run.cjs");
      const handler = await main({ name: "Test Check", max: 10 });
      await handler({ type: "create_check_run", conclusion: "success", title: "All good", summary: "Tests passed." }, {});

      expect(capturedParams.head_sha).toBe("push-sha-abc123");
    });

    it("falls back to context.sha when GITHUB_SHA is not set", async () => {
      mockContext.sha = "context-sha-xyz";
      mockContext.payload = {};

      let capturedParams;
      mockGithub.rest.checks.create = makeChecksCreate((p) => {
        capturedParams = p;
      });

      const { main } = require("./create_check_run.cjs");
      const handler = await main({ max: 10 });
      await handler({ type: "create_check_run", conclusion: "success", title: "All good", summary: "Tests passed." }, {});

      expect(capturedParams.head_sha).toBe("context-sha-xyz");
    });

    it("returns error when no SHA is available", async () => {
      delete process.env.GITHUB_SHA;
      mockContext.sha = "";
      mockContext.payload = {};

      const { main } = require("./create_check_run.cjs");
      const handler = await main({ max: 10 });
      const result = await handler({ type: "create_check_run", conclusion: "success", title: "Title", summary: "Summary" }, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("SHA");
    });
  });

  describe("required field validation", () => {
    beforeEach(() => {
      process.env.GITHUB_SHA = "sha-abc123";
    });

    it("returns error when conclusion is missing", async () => {
      const { main } = require("./create_check_run.cjs");
      const handler = await main({ max: 10 });
      const result = await handler({ type: "create_check_run", title: "Title", summary: "Summary" }, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("conclusion");
    });

    it("returns error for invalid conclusion value", async () => {
      const { main } = require("./create_check_run.cjs");
      const handler = await main({ max: 10 });
      const result = await handler({ type: "create_check_run", conclusion: "invalid-value", title: "Title", summary: "Summary" }, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("invalid conclusion");
    });

    it("returns error when title is missing", async () => {
      const { main } = require("./create_check_run.cjs");
      const handler = await main({ max: 10 });
      const result = await handler({ type: "create_check_run", conclusion: "success", summary: "Summary" }, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("title");
    });

    it("returns error when summary is missing", async () => {
      const { main } = require("./create_check_run.cjs");
      const handler = await main({ max: 10 });
      const result = await handler({ type: "create_check_run", conclusion: "success", title: "Title" }, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("summary");
    });

    it("accepts all valid conclusion values", async () => {
      const validConclusions = ["success", "failure", "neutral", "cancelled", "skipped", "timed_out", "action_required"];
      for (const conclusion of validConclusions) {
        const { main } = require("./create_check_run.cjs");
        const handler = await main({ max: 10 });
        const result = await handler({ type: "create_check_run", conclusion, title: "Title", summary: "Summary" }, {});
        expect(result.success).toBe(true);
      }
    });
  });

  describe("max limit enforcement", () => {
    beforeEach(() => {
      process.env.GITHUB_SHA = "sha-abc123";
    });

    it("enforces max count and skips excess messages", async () => {
      const { main } = require("./create_check_run.cjs");
      const handler = await main({ name: "Check", max: 1 });

      const msg = { type: "create_check_run", conclusion: "success", title: "Title", summary: "Summary" };
      const first = await handler(msg, {});
      const second = await handler(msg, {});

      expect(first.success).toBe(true);
      expect(second.success).toBe(false);
      expect(second.error).toContain("Max count");
    });
  });

  describe("truncation", () => {
    beforeEach(() => {
      process.env.GITHUB_SHA = "sha-abc123";
    });

    it("truncates summary to 65535 characters", async () => {
      let capturedParams;
      mockGithub.rest.checks.create = makeChecksCreate((p) => {
        capturedParams = p;
      });

      const longSummary = "x".repeat(70000);
      const { main } = require("./create_check_run.cjs");
      const handler = await main({ max: 10 });
      const result = await handler({ type: "create_check_run", conclusion: "success", title: "Title", summary: longSummary }, {});

      expect(result.success).toBe(true);
      expect(capturedParams.output.summary.length).toBe(65535);
    });

    it("truncates text to 65535 characters", async () => {
      let capturedParams;
      mockGithub.rest.checks.create = makeChecksCreate((p) => {
        capturedParams = p;
      });

      const longText = "y".repeat(70000);
      const { main } = require("./create_check_run.cjs");
      const handler = await main({ max: 10 });
      const result = await handler({ type: "create_check_run", conclusion: "success", title: "Title", summary: "Summary", text: longText }, {});

      expect(result.success).toBe(true);
      expect(capturedParams.output.text.length).toBe(65535);
    });

    it("omits text field from output when text is empty", async () => {
      let capturedParams;
      mockGithub.rest.checks.create = makeChecksCreate((p) => {
        capturedParams = p;
      });

      const { main } = require("./create_check_run.cjs");
      const handler = await main({ max: 10 });
      await handler({ type: "create_check_run", conclusion: "success", title: "Title", summary: "Summary" }, {});

      expect("text" in capturedParams.output).toBe(false);
    });
  });

  describe("check run API call shape", () => {
    beforeEach(() => {
      process.env.GITHUB_SHA = "sha-abc123";
    });

    it("passes correct parameters to checks.create", async () => {
      let capturedParams;
      mockGithub.rest.checks.create = makeChecksCreate((p) => {
        capturedParams = p;
      });

      const { main } = require("./create_check_run.cjs");
      const handler = await main({ name: "My Check Run", max: 10 });
      const result = await handler(
        { type: "create_check_run", conclusion: "failure", title: "3 issues found", summary: "Details here", text: "More detail" },
        {},
      );

      expect(result.success).toBe(true);
      expect(capturedParams.owner).toBe("test-owner");
      expect(capturedParams.repo).toBe("test-repo");
      expect(capturedParams.name).toBe("My Check Run");
      expect(capturedParams.head_sha).toBe("sha-abc123");
      expect(capturedParams.status).toBe("completed");
      expect(capturedParams.conclusion).toBe("failure");
      expect(capturedParams.output.title).toBe("3 issues found");
      expect(capturedParams.output.summary).toBe("Details here");
      expect(capturedParams.output.text).toBe("More detail");
      expect(capturedParams.completed_at).toBeDefined();
    });

    it("uses config name for check run name, falling back to GITHUB_WORKFLOW", async () => {
      process.env.GITHUB_WORKFLOW = "My Workflow";
      let capturedName;
      mockGithub.rest.checks.create = makeChecksCreate((p) => {
        capturedName = p.name;
      });

      // With explicit config name
      const { main: mainWithName } = require("./create_check_run.cjs");
      const handlerWithName = await mainWithName({ name: "Explicit Name", max: 10 });
      await handlerWithName({ type: "create_check_run", conclusion: "success", title: "T", summary: "S" }, {});
      expect(capturedName).toBe("Explicit Name");

      // Without config name — falls back to GITHUB_WORKFLOW
      const { main: mainNoName } = require("./create_check_run.cjs");
      const handlerNoName = await mainNoName({ max: 10 });
      await handlerNoName({ type: "create_check_run", conclusion: "success", title: "T", summary: "S" }, {});
      expect(capturedName).toBe("My Workflow");
    });

    it("returns check_run_id and check_run_url on success", async () => {
      const { main } = require("./create_check_run.cjs");
      const handler = await main({ max: 10 });
      const result = await handler({ type: "create_check_run", conclusion: "success", title: "Title", summary: "Summary" }, {});

      expect(result.success).toBe(true);
      expect(result.check_run_id).toBe(77313480284);
      expect(result.check_run_url).toContain("github.com");
      expect(result.conclusion).toBe("success");
    });
  });

  describe("staged mode", () => {
    beforeEach(() => {
      process.env.GITHUB_SHA = "sha-abc123";
    });

    it("returns staged preview without calling checks.create when staged via config", async () => {
      let createCalled = false;
      mockGithub.rest.checks.create = async () => {
        createCalled = true;
        return { data: { id: 1, html_url: "https://github.com/test-owner/test-repo/runs/1" } };
      };

      const { main } = require("./create_check_run.cjs");
      const handler = await main({ name: "My Check", max: 10, staged: true });
      const result = await handler({ type: "create_check_run", conclusion: "failure", title: "Title", summary: "Summary" }, {});

      expect(createCalled).toBe(false);
      expect(result.success).toBe(true);
      expect(result.staged).toBe(true);
      expect(result.previewInfo.conclusion).toBe("failure");
    });

    it("returns staged preview without calling checks.create when GH_AW_SAFE_OUTPUTS_STAGED=true", async () => {
      process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true";
      let createCalled = false;
      mockGithub.rest.checks.create = async () => {
        createCalled = true;
        return { data: { id: 1, html_url: "https://github.com/test-owner/test-repo/runs/1" } };
      };

      const { main } = require("./create_check_run.cjs");
      const handler = await main({ name: "My Check", max: 10 });
      const result = await handler({ type: "create_check_run", conclusion: "success", title: "Title", summary: "Summary" }, {});

      expect(createCalled).toBe(false);
      expect(result.success).toBe(true);
      expect(result.staged).toBe(true);
    });
  });
});
