// @ts-check
import { describe, it, expect, beforeEach, afterEach } from "vitest";

describe("create_check_run", () => {
  let mockCore;
  let mockGithub;
  let mockContext;
  let originalGlobals;
  let originalEnv;

  const makeChecksCreate = onCall => {
    return async params => {
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
          create: async params => ({
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
    Object.keys(process.env).forEach(k => {
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
      mockGithub.rest.checks.create = makeChecksCreate(p => {
        capturedParams = p;
      });

      const infoCalls = [];
      mockCore.info = msg => infoCalls.push(msg);

      const { main } = require("./create_check_run.cjs");
      const handler = await main({ name: "Test Check", max: 10 });
      await handler({ type: "create_check_run", conclusion: "success", title: "All good", summary: "Tests passed." }, {});

      expect(capturedParams.head_sha).toBe(prHeadSha);
      expect(capturedParams.head_sha).not.toBe("merge-commit-sha-xyz789");
      expect(infoCalls.some(m => m.includes(prHeadSha) && m.includes("pull_request"))).toBe(true);
    });

    it("falls back to GITHUB_SHA on push events (no PR payload)", async () => {
      process.env.GITHUB_SHA = "push-sha-abc123";
      mockContext.eventName = "push";
      mockContext.payload = {};

      let capturedParams;
      mockGithub.rest.checks.create = makeChecksCreate(p => {
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
      mockGithub.rest.checks.create = makeChecksCreate(p => {
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

    it("returns error when title is missing and no config fallback", async () => {
      const { main } = require("./create_check_run.cjs");
      const handler = await main({ max: 10 });
      const result = await handler({ type: "create_check_run", conclusion: "success", summary: "Summary" }, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("title");
    });

    it("returns error when summary is missing and no config fallback", async () => {
      const { main } = require("./create_check_run.cjs");
      const handler = await main({ max: 10 });
      const result = await handler({ type: "create_check_run", conclusion: "success", title: "Title" }, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("summary");
    });

    it("accepts all valid conclusion values", async () => {
      const validConclusions = ["success", "failure", "neutral", "cancelled", "skipped", "timed_out", "action_required"];
      for (const conclusion of validConclusions) {
        let capturedConclusion;
        mockGithub.rest.checks.create = makeChecksCreate(p => {
          capturedConclusion = p.conclusion;
        });
        const { main } = require("./create_check_run.cjs");
        const handler = await main({ max: 10 });
        const result = await handler({ type: "create_check_run", conclusion, title: "Title", summary: "Summary" }, {});
        expect(result.success).toBe(true);
        expect(capturedConclusion).toBe(conclusion);
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

    it("truncates summary exceeding 65535 characters (appends truncation notice)", async () => {
      let capturedParams;
      mockGithub.rest.checks.create = makeChecksCreate(p => {
        capturedParams = p;
      });

      const longSummary = "x".repeat(70000);
      const { main } = require("./create_check_run.cjs");
      const handler = await main({ max: 10 });
      const result = await handler({ type: "create_check_run", conclusion: "success", title: "Title", summary: longSummary }, {});

      expect(result.success).toBe(true);
      expect(capturedParams.output.summary.length).toBeLessThanOrEqual(66000);
      expect(capturedParams.output.summary).toContain("[Content truncated");
    });

    it("truncates text exceeding 65535 characters (appends truncation notice)", async () => {
      let capturedParams;
      mockGithub.rest.checks.create = makeChecksCreate(p => {
        capturedParams = p;
      });

      const longText = "y".repeat(70000);
      const { main } = require("./create_check_run.cjs");
      const handler = await main({ max: 10 });
      const result = await handler({ type: "create_check_run", conclusion: "success", title: "Title", summary: "Summary", text: longText }, {});

      expect(result.success).toBe(true);
      expect(capturedParams.output.text.length).toBeLessThanOrEqual(66000);
      expect(capturedParams.output.text).toContain("[Content truncated");
    });

    it("omits text field from output when text is empty", async () => {
      let capturedParams;
      mockGithub.rest.checks.create = makeChecksCreate(p => {
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
      mockGithub.rest.checks.create = makeChecksCreate(p => {
        capturedParams = p;
      });

      const { main } = require("./create_check_run.cjs");
      const handler = await main({ name: "My Check Run", max: 10 });
      const result = await handler({ type: "create_check_run", conclusion: "failure", title: "3 issues found", summary: "Details here", text: "More detail" }, {});

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

    it("uses config name for check run name, falling back to GITHUB_WORKFLOW with (Result) suffix", async () => {
      process.env.GITHUB_WORKFLOW = "My Workflow";
      let capturedName;
      mockGithub.rest.checks.create = makeChecksCreate(p => {
        capturedName = p.name;
      });

      // With explicit config name different from workflow name — no dedup suffix
      const { main: mainWithName } = require("./create_check_run.cjs");
      const handlerWithName = await mainWithName({ name: "Explicit Name", max: 10 });
      await handlerWithName({ type: "create_check_run", conclusion: "success", title: "T", summary: "S" }, {});
      expect(capturedName).toBe("Explicit Name");

      // Without config name — falls back to GITHUB_WORKFLOW but deduplicates with (Result) suffix
      const { main: mainNoName } = require("./create_check_run.cjs");
      const handlerNoName = await mainNoName({ max: 10 });
      await handlerNoName({ type: "create_check_run", conclusion: "success", title: "T", summary: "S" }, {});
      expect(capturedName).toBe("My Workflow (Result)");
    });

    it("appends (Result) suffix when configured name collides with GITHUB_WORKFLOW", async () => {
      process.env.GITHUB_WORKFLOW = "My Workflow";
      let capturedName;
      mockGithub.rest.checks.create = makeChecksCreate(p => {
        capturedName = p.name;
      });

      // Config name explicitly set to same value as workflow name → dedup suffix applied
      const { main } = require("./create_check_run.cjs");
      const handler = await main({ name: "My Workflow", max: 10 });
      await handler({ type: "create_check_run", conclusion: "success", title: "T", summary: "S" }, {});
      expect(capturedName).toBe("My Workflow (Result)");
    });

    it("does not append (Result) suffix when config name differs from GITHUB_WORKFLOW", async () => {
      process.env.GITHUB_WORKFLOW = "My Workflow";
      let capturedName;
      mockGithub.rest.checks.create = makeChecksCreate(p => {
        capturedName = p.name;
      });

      const { main } = require("./create_check_run.cjs");
      const handler = await main({ name: "My Agent Result", max: 10 });
      await handler({ type: "create_check_run", conclusion: "success", title: "T", summary: "S" }, {});
      expect(capturedName).toBe("My Agent Result");
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

  describe("config output.title / output.summary fallbacks", () => {
    beforeEach(() => {
      process.env.GITHUB_SHA = "sha-abc123";
    });

    it("uses config output_title as fallback when agent omits title", async () => {
      let capturedParams;
      mockGithub.rest.checks.create = makeChecksCreate(p => {
        capturedParams = p;
      });

      const { main } = require("./create_check_run.cjs");
      const handler = await main({ max: 10, output_title: "Config Title" });
      const result = await handler({ type: "create_check_run", conclusion: "success", summary: "Summary" }, {});

      expect(result.success).toBe(true);
      expect(capturedParams.output.title).toBe("Config Title");
    });

    it("uses config output_summary as fallback when agent omits summary", async () => {
      let capturedParams;
      mockGithub.rest.checks.create = makeChecksCreate(p => {
        capturedParams = p;
      });

      const { main } = require("./create_check_run.cjs");
      const handler = await main({ max: 10, output_summary: "Config Summary" });
      const result = await handler({ type: "create_check_run", conclusion: "success", title: "Title" }, {});

      expect(result.success).toBe(true);
      expect(capturedParams.output.summary).toBe("Config Summary");
    });

    it("agent-provided title takes precedence over config output_title", async () => {
      let capturedParams;
      mockGithub.rest.checks.create = makeChecksCreate(p => {
        capturedParams = p;
      });

      const { main } = require("./create_check_run.cjs");
      const handler = await main({ max: 10, output_title: "Config Title" });
      const result = await handler({ type: "create_check_run", conclusion: "success", title: "Agent Title", summary: "Summary" }, {});

      expect(result.success).toBe(true);
      expect(capturedParams.output.title).toBe("Agent Title");
    });

    it("agent-provided summary takes precedence over config output_summary", async () => {
      let capturedParams;
      mockGithub.rest.checks.create = makeChecksCreate(p => {
        capturedParams = p;
      });

      const { main } = require("./create_check_run.cjs");
      const handler = await main({ max: 10, output_summary: "Config Summary" });
      const result = await handler({ type: "create_check_run", conclusion: "success", title: "Title", summary: "Agent Summary" }, {});

      expect(result.success).toBe(true);
      expect(capturedParams.output.summary).toBe("Agent Summary");
    });

    it("succeeds using both config fallbacks when agent omits title and summary", async () => {
      let capturedParams;
      mockGithub.rest.checks.create = makeChecksCreate(p => {
        capturedParams = p;
      });

      const { main } = require("./create_check_run.cjs");
      const handler = await main({ max: 10, output_title: "Config Title", output_summary: "Config Summary" });
      const result = await handler({ type: "create_check_run", conclusion: "failure" }, {});

      expect(result.success).toBe(true);
      expect(capturedParams.output.title).toBe("Config Title");
      expect(capturedParams.output.summary).toBe("Config Summary");
    });

    it("sanitizes config output_title (neutralizes @mentions into backtick-escaped form)", async () => {
      let capturedParams;
      mockGithub.rest.checks.create = makeChecksCreate(p => {
        capturedParams = p;
      });

      const { main } = require("./create_check_run.cjs");
      // sanitizeContent wraps bare @mentions in backticks so they don't trigger notifications
      const handler = await main({ max: 10, output_title: "Check by @admin" });
      const result = await handler({ type: "create_check_run", conclusion: "success", summary: "Summary" }, {});

      expect(result.success).toBe(true);
      // @mention is escaped to `@admin` — no longer a bare @mention
      expect(capturedParams.output.title).not.toBe("Check by @admin");
      expect(capturedParams.output.title).toContain("`@admin`");
    });

    it("sanitizes agent-provided title (neutralizes @mentions into backtick-escaped form)", async () => {
      let capturedParams;
      mockGithub.rest.checks.create = makeChecksCreate(p => {
        capturedParams = p;
      });

      const { main } = require("./create_check_run.cjs");
      const handler = await main({ max: 10 });
      const result = await handler(
        {
          type: "create_check_run",
          conclusion: "success",
          title: "Review by @admin",
          summary: "Summary",
        },
        {}
      );

      expect(result.success).toBe(true);
      // @mention is escaped to `@admin` — no longer a bare @mention
      expect(capturedParams.output.title).not.toBe("Review by @admin");
      expect(capturedParams.output.title).toContain("`@admin`");
    });

    it("still errors when title and summary are both absent with no config fallbacks", async () => {
      const { main } = require("./create_check_run.cjs");
      const handler = await main({ max: 10 });
      const result = await handler({ type: "create_check_run", conclusion: "success" }, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("title");
    });
  });
});
