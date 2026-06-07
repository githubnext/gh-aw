// @ts-check

import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { createRequire } from "module";

const require = createRequire(import.meta.url);

describe("handle_agent_failure", () => {
  let main;
  let buildCodePushFailureContext;
  let buildPushRepoMemoryFailureContext;
  let buildReportIncompleteContext;
  let getActionFailureIssueExpiresHours;

  beforeEach(() => {
    // Provide minimal GitHub Actions globals expected by require-time code
    global.core = {
      info: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
      debug: vi.fn(),
      setOutput: vi.fn(),
      setFailed: vi.fn(),
    };
    global.github = {};
    global.context = { repo: { owner: "owner", repo: "repo" } };

    // Reset module registry so each test gets a fresh require
    vi.resetModules();
    ({ main, buildCodePushFailureContext, buildPushRepoMemoryFailureContext, buildReportIncompleteContext, getActionFailureIssueExpiresHours } = require("./handle_agent_failure.cjs"));
  });

  afterEach(() => {
    delete global.core;
    delete global.github;
    delete global.context;
    delete process.env.GITHUB_SHA;
    delete process.env.GH_AW_ACTION_FAILURE_ISSUE_EXPIRES_HOURS;
  });

  describe("getActionFailureIssueExpiresHours", () => {
    it("returns default when env var is missing", () => {
      expect(getActionFailureIssueExpiresHours()).toBe(168);
    });

    it("returns configured value when env var is a positive integer", () => {
      process.env.GH_AW_ACTION_FAILURE_ISSUE_EXPIRES_HOURS = "48";
      expect(getActionFailureIssueExpiresHours()).toBe(48);
    });

    it("returns default for invalid values", () => {
      process.env.GH_AW_ACTION_FAILURE_ISSUE_EXPIRES_HOURS = "0";
      expect(getActionFailureIssueExpiresHours()).toBe(168);
      process.env.GH_AW_ACTION_FAILURE_ISSUE_EXPIRES_HOURS = "invalid";
      expect(getActionFailureIssueExpiresHours()).toBe(168);
    });
  });

  describe("detection caution placement in main()", () => {
    const fs = require("fs");
    const path = require("path");
    const os = require("os");

    /** @type {string} */
    let tmpDir;
    /** @type {string} */
    let promptsDir;

    beforeEach(() => {
      tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "aw-handle-agent-failure-"));
      promptsDir = path.join(tmpDir, "gh-aw", "prompts");
      fs.mkdirSync(promptsDir, { recursive: true });

      // Minimal templates used by main()
      fs.writeFileSync(path.join(promptsDir, "agent_failure_comment.md"), "COMMENT TEMPLATE CONTENT");
      fs.writeFileSync(path.join(promptsDir, "agent_failure_issue.md"), "ISSUE TEMPLATE CONTENT");
      fs.writeFileSync(path.join(promptsDir, "daily_cap_rollup_issue.md"), "Daily cap rollup issue body cap={cap} window={window_hours}");
      fs.writeFileSync(path.join(promptsDir, "daily_cap_rollup_comment.md"), "Failure suppressed workflow={workflow_name} run={run_url} categories={summary} cap={cap} window={window_hours}h");

      process.env.RUNNER_TEMP = tmpDir;
      process.env.GH_AW_WORKFLOW_NAME = "Test Workflow";
      process.env.GH_AW_WORKFLOW_ID = "test-workflow";
      process.env.GH_AW_RUN_URL = "https://github.com/owner/repo/actions/runs/123456";
      process.env.GH_AW_AGENT_CONCLUSION = "failure";
      process.env.GH_AW_DETECTION_CONCLUSION = "warning";
      process.env.GH_AW_DETECTION_REASON = "threat_detected";
      process.env.GITHUB_HEAD_REF = "feature/detection-caution";
      process.env.GITHUB_WORKSPACE = tmpDir;
    });

    afterEach(() => {
      delete process.env.RUNNER_TEMP;
      delete process.env.GH_AW_WORKFLOW_NAME;
      delete process.env.GH_AW_WORKFLOW_ID;
      delete process.env.GH_AW_RUN_URL;
      delete process.env.GH_AW_AGENT_CONCLUSION;
      delete process.env.GH_AW_DETECTION_CONCLUSION;
      delete process.env.GH_AW_DETECTION_REASON;
      delete process.env.GITHUB_HEAD_REF;
      delete process.env.GITHUB_WORKSPACE;

      if (tmpDir && fs.existsSync(tmpDir)) {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("prepends caution callout to existing-issue comment body and includes it only once", async () => {
      /** @type {string} */
      let capturedCommentBody = "";

      global.github = {
        rest: {
          search: {
            issuesAndPullRequests: vi.fn(async ({ q }) => {
              if (q.includes("is:pr")) {
                return { data: { total_count: 0, items: [] } };
              }
              return {
                data: {
                  total_count: 1,
                  items: [
                    {
                      number: 42,
                      html_url: "https://github.com/owner/repo/issues/42",
                      body:
                        "> footer\n> - [x] expires <!-- gh-aw-expires: 2099-01-01T00:00:00.000Z --> on Jan 1, 2099, 12:00 AM UTC\n\n" +
                        "<!-- gh-aw-agentic-workflow: Test Workflow, workflow_id: test-workflow, run: https://github.com/owner/repo/actions/runs/123456 -->\n" +
                        "<!-- gh-aw-failure-issue: true, workflow_id: test-workflow, branch: feature/detection-caution, failure_categories: agent_failure -->",
                    },
                  ],
                },
              };
            }),
          },
          issues: {
            createComment: vi.fn(async ({ body }) => {
              capturedCommentBody = body;
              return { data: { id: 1001 } };
            }),
          },
          pulls: {
            get: vi.fn(),
          },
        },
        graphql: vi.fn(),
      };

      await main();

      expect(capturedCommentBody).toBeTruthy();
      expect(capturedCommentBody.startsWith("> [!CAUTION]")).toBe(true);
      expect(capturedCommentBody.indexOf("> [!CAUTION]")).toBeLessThan(capturedCommentBody.indexOf("COMMENT TEMPLATE CONTENT"));
      expect((capturedCommentBody.match(/> \[!CAUTION\]/g) || []).length).toBe(1);
      expect(capturedCommentBody).toContain("> Generated from [Test Workflow]");
    });

    it("prepends caution callout to new issue body and includes it only once", async () => {
      /** @type {string} */
      let capturedIssueBody = "";

      global.github = {
        rest: {
          search: {
            issuesAndPullRequests: vi.fn(async ({ q }) => {
              if (q.includes("is:pr")) {
                return { data: { total_count: 0, items: [] } };
              }
              return { data: { total_count: 0, items: [] } };
            }),
          },
          issues: {
            create: vi.fn(async ({ body }) => {
              capturedIssueBody = body;
              return {
                data: { number: 101, html_url: "https://github.com/owner/repo/issues/101", node_id: "I_123" },
              };
            }),
          },
          pulls: {
            get: vi.fn(),
          },
        },
        graphql: vi.fn(),
      };

      await main();

      expect(capturedIssueBody).toBeTruthy();
      expect(capturedIssueBody.startsWith("> [!CAUTION]")).toBe(true);
      expect(capturedIssueBody.indexOf("> [!CAUTION]")).toBeLessThan(capturedIssueBody.indexOf("ISSUE TEMPLATE CONTENT"));
      expect((capturedIssueBody.match(/> \[!CAUTION\]/g) || []).length).toBe(1);
      expect(capturedIssueBody).toContain("> Generated from [Test Workflow]");
    });

    it("includes AIC and ambient context metrics in the generated failure issue footer", async () => {
      process.env.GH_AW_AIC = "1.25";
      process.env.GH_AW_AMBIENT_CONTEXT = "900";
      /** @type {string} */
      let capturedIssueBody = "";

      global.github = {
        rest: {
          search: {
            issuesAndPullRequests: vi.fn(async ({ q }) => {
              if (q.includes("is:pr")) {
                return { data: { total_count: 0, items: [] } };
              }
              return { data: { total_count: 0, items: [] } };
            }),
          },
          issues: {
            create: vi.fn(async ({ body }) => {
              capturedIssueBody = body;
              return {
                data: { number: 101, html_url: "https://github.com/owner/repo/issues/101", node_id: "I_123" },
              };
            }),
          },
          pulls: {
            get: vi.fn(),
          },
        },
        graphql: vi.fn(),
      };

      try {
        await main();

        expect(capturedIssueBody).toContain("> Generated from [Test Workflow](https://github.com/owner/repo/actions/runs/123456) · 1.25 AIC · ⊞ 900");
      } finally {
        delete process.env.GH_AW_AIC;
        delete process.env.GH_AW_AMBIENT_CONTEXT;
      }
    });
  });

  describe("main() precise failure issue matching", () => {
    const fs = require("fs");
    const path = require("path");
    const os = require("os");

    /** @type {string} */
    let tmpDir;
    /** @type {string} */
    let promptsDir;

    function buildExistingIssueBody({ branch, categories, expires = "2099-01-01T00:00:00.000Z", pullRequestNumber, workflowName = "Test Workflow", workflowId = "test-workflow" } = {}) {
      const prPart = pullRequestNumber ? `, pull_request: ${pullRequestNumber}` : "";
      return (
        `> Generated from [${workflowName}](https://github.com/owner/repo/actions/runs/123456)\n` +
        `> - [x] expires <!-- gh-aw-expires: ${expires} --> on Jan 1, 2099, 12:00 AM UTC\n\n` +
        `<!-- gh-aw-agentic-workflow: ${workflowName}, workflow_id: ${workflowId}, run: https://github.com/owner/repo/actions/runs/123456 -->\n` +
        `<!-- gh-aw-failure-issue: true, workflow_id: ${workflowId}, branch: ${branch || ""}, failure_categories: ${categories.join("|")}${prPart} -->`
      );
    }

    function buildWorkflowMarkerOnlyIssueBody({ expires = "2099-01-01T00:00:00.000Z", workflowName = "Test Workflow", workflowId = "test-workflow" } = {}) {
      return (
        `> Generated from [${workflowName}](https://github.com/owner/repo/actions/runs/123456)\n` +
        `> - [x] expires <!-- gh-aw-expires: ${expires} --> on Jan 1, 2099, 12:00 AM UTC\n\n` +
        `<!-- gh-aw-agentic-workflow: ${workflowName}, workflow_id: ${workflowId}, run: https://github.com/owner/repo/actions/runs/123456 -->`
      );
    }

    beforeEach(() => {
      tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "aw-handle-agent-failure-match-"));
      promptsDir = path.join(tmpDir, "gh-aw", "prompts");
      fs.mkdirSync(promptsDir, { recursive: true });
      fs.writeFileSync(path.join(promptsDir, "agent_failure_comment.md"), "COMMENT TEMPLATE CONTENT");
      fs.writeFileSync(path.join(promptsDir, "agent_failure_issue.md"), "ISSUE TEMPLATE CONTENT");
      fs.writeFileSync(path.join(promptsDir, "daily_cap_rollup_issue.md"), "Daily cap rollup issue body cap={cap} window={window_hours}");
      fs.writeFileSync(path.join(promptsDir, "daily_cap_rollup_comment.md"), "Failure suppressed workflow={workflow_name} run={run_url} categories={summary} cap={cap} window={window_hours}h");

      process.env.RUNNER_TEMP = tmpDir;
      process.env.GH_AW_WORKFLOW_NAME = "Test Workflow";
      process.env.GH_AW_WORKFLOW_ID = "test-workflow";
      process.env.GH_AW_RUN_URL = "https://github.com/owner/repo/actions/runs/123456";
      process.env.GH_AW_AGENT_CONCLUSION = "success";
      process.env.GITHUB_HEAD_REF = "feature/current";
      process.env.GITHUB_WORKSPACE = tmpDir;
    });

    afterEach(() => {
      delete process.env.RUNNER_TEMP;
      delete process.env.GH_AW_WORKFLOW_NAME;
      delete process.env.GH_AW_WORKFLOW_ID;
      delete process.env.GH_AW_RUN_URL;
      delete process.env.GH_AW_AGENT_CONCLUSION;
      delete process.env.GITHUB_HEAD_REF;
      delete process.env.GITHUB_WORKSPACE;
      if (tmpDir && fs.existsSync(tmpDir)) {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("adds a comment only when the existing issue metadata matches exactly", async () => {
      const createCommentMock = vi.fn(async () => ({ data: { id: 1001 } }));
      const createIssueMock = vi.fn();

      global.github = {
        rest: {
          search: {
            issuesAndPullRequests: vi.fn(async ({ q }) => {
              if (q.includes("is:pr")) {
                return { data: { total_count: 0, items: [] } };
              }
              return {
                data: {
                  total_count: 1,
                  items: [
                    {
                      number: 42,
                      html_url: "https://github.com/owner/repo/issues/42",
                      body: buildExistingIssueBody({ branch: "feature/current", categories: ["missing_safe_outputs"] }),
                    },
                  ],
                },
              };
            }),
          },
          issues: {
            create: createIssueMock,
            createComment: createCommentMock,
          },
          pulls: { get: vi.fn() },
        },
        graphql: vi.fn(),
      };

      await main();

      expect(createCommentMock).toHaveBeenCalledOnce();
      expect(createIssueMock).not.toHaveBeenCalled();
    });

    it("adds a comment when existing issue metadata contains commas in free-form values", async () => {
      const createCommentMock = vi.fn(async () => ({ data: { id: 1001 } }));
      const createIssueMock = vi.fn();

      process.env.GH_AW_WORKFLOW_NAME = "Test Workflow, Retry";
      process.env.GITHUB_HEAD_REF = "feature/current,with-comma";

      global.github = {
        rest: {
          search: {
            issuesAndPullRequests: vi.fn(async ({ q }) => {
              if (q.includes("is:pr")) {
                return { data: { total_count: 0, items: [] } };
              }
              return {
                data: {
                  total_count: 1,
                  items: [
                    {
                      number: 42,
                      html_url: "https://github.com/owner/repo/issues/42",
                      body: buildExistingIssueBody({
                        workflowName: "Test Workflow, Retry",
                        branch: "feature/current,with-comma",
                        categories: ["missing_safe_outputs"],
                      }),
                    },
                  ],
                },
              };
            }),
          },
          issues: {
            create: createIssueMock,
            createComment: createCommentMock,
          },
          pulls: { get: vi.fn() },
        },
        graphql: vi.fn(),
      };

      await main();

      expect(createCommentMock).toHaveBeenCalledOnce();
      expect(createIssueMock).not.toHaveBeenCalled();
    });

    it("creates a new issue when an open issue only has workflow XML metadata without failure metadata", async () => {
      const createCommentMock = vi.fn();
      const createIssueMock = vi.fn();
      const searchMock = vi.fn(async ({ q }) => {
        if (q.includes("is:pr")) {
          return { data: { total_count: 0, items: [] } };
        }
        return {
          data: {
            total_count: 1,
            items: [
              {
                number: 42,
                title: "[aw] Test Workflow failed",
                html_url: "https://github.com/owner/repo/issues/42",
                body: buildWorkflowMarkerOnlyIssueBody(),
              },
            ],
          },
        };
      });

      global.github = {
        rest: {
          search: {
            issuesAndPullRequests: searchMock,
          },
          issues: {
            create: createIssueMock,
            createComment: createCommentMock,
          },
          pulls: { get: vi.fn() },
        },
        graphql: vi.fn(),
      };

      await main();

      expect(createCommentMock).not.toHaveBeenCalled();
      expect(createIssueMock).toHaveBeenCalledOnce();
      expect(searchMock).toHaveBeenCalledWith(expect.objectContaining({ q: expect.stringContaining('"gh-aw-agentic-workflow:"') }));
      expect(searchMock).toHaveBeenCalledWith(expect.objectContaining({ q: expect.stringContaining('"workflow_id: test-workflow" in:body') }));
    });

    it("escapes workflow IDs before searching for legacy XML marker matches", async () => {
      const createCommentMock = vi.fn(async () => ({ data: { id: 1001 } }));
      const createIssueMock = vi.fn();
      const workflowId = 'test"workflow\\path\nnext-line';
      const searchMock = vi.fn(async ({ q }) => {
        if (q.includes("is:pr")) {
          return { data: { total_count: 0, items: [] } };
        }
        return {
          data: {
            total_count: 1,
            items: [
              {
                number: 42,
                title: "[aw] Test Workflow failed",
                html_url: "https://github.com/owner/repo/issues/42",
                body: buildExistingIssueBody({
                  workflowId,
                  branch: "feature/current",
                  categories: ["missing_safe_outputs"],
                }),
              },
            ],
          },
        };
      });

      process.env.GH_AW_WORKFLOW_ID = workflowId;

      global.github = {
        rest: {
          search: {
            issuesAndPullRequests: searchMock,
          },
          issues: {
            create: createIssueMock,
            createComment: createCommentMock,
          },
          pulls: { get: vi.fn() },
        },
        graphql: vi.fn(),
      };

      await main();

      expect(createCommentMock).toHaveBeenCalledOnce();
      expect(createIssueMock).not.toHaveBeenCalled();
      expect(searchMock).toHaveBeenCalledWith(expect.objectContaining({ q: expect.stringContaining('"workflow_id: test\\"workflow\\\\path next-line" in:body') }));
    });

    it("adds a comment when branch differs but workflow/category markers match", async () => {
      const createCommentMock = vi.fn(async () => ({ data: { id: 1001 } }));
      const createIssueMock = vi.fn(async () => ({
        data: { number: 101, html_url: "https://github.com/owner/repo/issues/101", node_id: "I_123" },
      }));

      global.github = {
        rest: {
          search: {
            issuesAndPullRequests: vi.fn(async ({ q }) => {
              if (q.includes("is:pr")) {
                return { data: { total_count: 0, items: [] } };
              }
              return {
                data: {
                  total_count: 2,
                  items: [
                    {
                      number: 42,
                      title: "[aw] Test Workflow failed",
                      html_url: "https://github.com/owner/repo/issues/42",
                      body: buildExistingIssueBody({ branch: "feature/other", categories: ["missing_safe_outputs"] }),
                    },
                    {
                      number: 43,
                      title: "[aw] Test Workflow failed",
                      html_url: "https://github.com/owner/repo/issues/43",
                      body: buildExistingIssueBody({ branch: "feature/current", categories: ["agent_failure"] }),
                    },
                  ],
                },
              };
            }),
          },
          issues: {
            create: createIssueMock,
            createComment: createCommentMock,
          },
          pulls: { get: vi.fn() },
        },
        graphql: vi.fn(),
      };

      await main();

      expect(createCommentMock).toHaveBeenCalledOnce();
      expect(createIssueMock).not.toHaveBeenCalled();
    });

    it("creates a new issue instead of commenting on an expired issue", async () => {
      const createCommentMock = vi.fn();
      const createIssueMock = vi.fn(async () => ({
        data: { number: 101, html_url: "https://github.com/owner/repo/issues/101", node_id: "I_123" },
      }));

      global.github = {
        rest: {
          search: {
            issuesAndPullRequests: vi.fn(async ({ q }) => {
              if (q.includes("is:pr")) {
                return { data: { total_count: 0, items: [] } };
              }
              return {
                data: {
                  total_count: 1,
                  items: [
                    {
                      number: 42,
                      html_url: "https://github.com/owner/repo/issues/42",
                      body: buildExistingIssueBody({
                        branch: "feature/current",
                        categories: ["missing_safe_outputs"],
                        expires: "2000-01-01T00:00:00.000Z",
                      }),
                    },
                  ],
                },
              };
            }),
          },
          issues: {
            create: createIssueMock,
            createComment: createCommentMock,
          },
          pulls: { get: vi.fn() },
        },
        graphql: vi.fn(),
      };

      await main();

      expect(createCommentMock).not.toHaveBeenCalled();
      expect(createIssueMock).toHaveBeenCalledOnce();
    });

    it("creates a new issue when the only matching issue is older than 24 hours", async () => {
      const createCommentMock = vi.fn();
      const createIssueMock = vi.fn(async () => ({
        data: { number: 101, html_url: "https://github.com/owner/repo/issues/101", node_id: "I_123" },
      }));
      const oldTimestamp = new Date(Date.now() - 25 * 60 * 60 * 1000).toISOString();

      global.github = {
        rest: {
          search: {
            issuesAndPullRequests: vi.fn(async ({ q }) => {
              if (q.includes("is:pr")) {
                return { data: { total_count: 0, items: [] } };
              }
              return {
                data: {
                  total_count: 1,
                  items: [
                    {
                      number: 42,
                      html_url: "https://github.com/owner/repo/issues/42",
                      created_at: oldTimestamp,
                      body: buildExistingIssueBody({
                        branch: "feature/current",
                        categories: ["missing_safe_outputs"],
                      }),
                    },
                  ],
                },
              };
            }),
          },
          issues: {
            create: createIssueMock,
            createComment: createCommentMock,
          },
          pulls: { get: vi.fn() },
        },
        graphql: vi.fn(),
      };

      await main();

      expect(createCommentMock).not.toHaveBeenCalled();
      expect(createIssueMock).toHaveBeenCalledOnce();
    });

    it("continues searching later pages until it finds an exact metadata match", async () => {
      const createCommentMock = vi.fn(async () => ({ data: { id: 1001 } }));
      const createIssueMock = vi.fn();
      const searchMock = vi.fn(async ({ q, page }) => {
        if (q.includes("is:pr")) {
          return { data: { total_count: 0, items: [] } };
        }

        if (page === 1) {
          return {
            data: {
              total_count: 101,
              items: Array.from({ length: 100 }, (_, index) => ({
                number: index + 1,
                html_url: `https://github.com/owner/repo/issues/${index + 1}`,
                body: buildExistingIssueBody({ branch: `feature/other-${index + 1}`, categories: ["agent_failure"] }),
              })),
            },
          };
        }

        return {
          data: {
            total_count: 101,
            items: [
              {
                number: 101,
                html_url: "https://github.com/owner/repo/issues/101",
                body: buildExistingIssueBody({ branch: "feature/current", categories: ["missing_safe_outputs"] }),
              },
            ],
          },
        };
      });

      global.github = {
        rest: {
          search: {
            issuesAndPullRequests: searchMock,
          },
          issues: {
            create: createIssueMock,
            createComment: createCommentMock,
          },
          pulls: { get: vi.fn() },
        },
        graphql: vi.fn(),
      };

      await main();

      expect(createCommentMock).toHaveBeenCalledOnce();
      expect(createIssueMock).not.toHaveBeenCalled();
      expect(searchMock).toHaveBeenCalledWith(expect.objectContaining({ page: 1 }));
      expect(searchMock).toHaveBeenCalledWith(expect.objectContaining({ page: 2 }));
    });

    it("adds a comment when pull request metadata differs but workflow/category markers match", async () => {
      const createCommentMock = vi.fn(async () => ({ data: { id: 1001 } }));
      const createIssueMock = vi.fn(async () => ({
        data: { number: 101, html_url: "https://github.com/owner/repo/issues/101", node_id: "I_123" },
      }));

      global.github = {
        rest: {
          search: {
            issuesAndPullRequests: vi.fn(async ({ q }) => {
              if (q.includes("is:pr")) {
                return {
                  data: {
                    total_count: 1,
                    items: [{ number: 123, html_url: "https://github.com/owner/repo/pull/123" }],
                  },
                };
              }
              return {
                data: {
                  total_count: 1,
                  items: [
                    {
                      number: 42,
                      title: "[aw] Test Workflow failed",
                      html_url: "https://github.com/owner/repo/issues/42",
                      body: buildExistingIssueBody({
                        branch: "feature/current",
                        categories: ["missing_safe_outputs"],
                        pullRequestNumber: 999,
                      }),
                    },
                  ],
                },
              };
            }),
          },
          issues: {
            create: createIssueMock,
            createComment: createCommentMock,
          },
          pulls: {
            get: vi.fn(async () => ({
              data: {
                head: { sha: "abc123" },
                mergeable: true,
                mergeable_state: "clean",
                updated_at: "2026-05-18T00:00:00Z",
              },
            })),
          },
        },
        graphql: vi.fn(),
      };

      await main();

      expect(createCommentMock).toHaveBeenCalledOnce();
      expect(createIssueMock).not.toHaveBeenCalled();
    });

    it("creates a centralized rollup issue and adds a comment when per-category daily cap is reached", async () => {
      const createCommentMock = vi.fn(async () => ({ data: { id: 2001 } }));
      const createIssueMock = vi.fn(async () => ({
        data: { number: 999, html_url: "https://github.com/owner/repo/issues/999", node_id: "I_999" },
      }));
      const now = new Date().toISOString();

      global.github = {
        rest: {
          search: {
            issuesAndPullRequests: vi.fn(async ({ q }) => {
              if (q.includes("is:pr")) {
                return { data: { total_count: 0, items: [] } };
              }
              if (q.includes('"workflow_id: test-workflow" in:body')) {
                return { data: { total_count: 0, items: [] } };
              }
              if (q.includes('"gh-aw-failure-issue:"') && q.includes('"failure_categories:"')) {
                return {
                  data: {
                    total_count: 50,
                    items: Array.from({ length: 50 }, (_, index) => ({
                      number: index + 1,
                      html_url: `https://github.com/owner/repo/issues/${index + 1}`,
                      created_at: now,
                      body: buildExistingIssueBody({
                        branch: `feature/any-${index + 1}`,
                        categories: ["missing_safe_outputs"],
                      }),
                    })),
                  },
                };
              }
              return { data: { total_count: 0, items: [] } };
            }),
          },
          issues: {
            create: createIssueMock,
            createComment: createCommentMock,
          },
          pulls: {
            get: vi.fn(),
          },
        },
        graphql: vi.fn(),
      };

      await main();

      expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("Daily per-category issue cap reached"));
      expect(global.core.info).toHaveBeenCalledWith(expect.stringContaining("Summarize-and-stop"));
      expect(createIssueMock).toHaveBeenCalledWith(expect.objectContaining({ title: "[aw] Daily failure issue cap exceeded" }));
      expect(createCommentMock).toHaveBeenCalledOnce();
      expect(createCommentMock).toHaveBeenCalledWith(expect.objectContaining({ issue_number: 999 }));
      expect(global.github.rest.search.issuesAndPullRequests).toHaveBeenCalledWith(expect.objectContaining({ q: expect.stringContaining("is:open") }));
    });

    it("reuses an existing rollup issue when per-category daily cap is reached", async () => {
      const createCommentMock = vi.fn(async () => ({ data: { id: 2002 } }));
      const createIssueMock = vi.fn();
      const now = new Date().toISOString();

      global.github = {
        rest: {
          search: {
            issuesAndPullRequests: vi.fn(async ({ q }) => {
              if (q.includes("is:pr")) {
                return { data: { total_count: 0, items: [] } };
              }
              if (q.includes('"workflow_id: test-workflow" in:body')) {
                return { data: { total_count: 0, items: [] } };
              }
              if (q.includes('"gh-aw-failure-issue:"') && q.includes('"failure_categories:"')) {
                return {
                  data: {
                    total_count: 50,
                    items: Array.from({ length: 50 }, (_, index) => ({
                      number: index + 1,
                      html_url: `https://github.com/owner/repo/issues/${index + 1}`,
                      created_at: now,
                      body: buildExistingIssueBody({
                        branch: `feature/any-${index + 1}`,
                        categories: ["missing_safe_outputs"],
                      }),
                    })),
                  },
                };
              }
              if (q.includes("[aw] Daily failure issue cap exceeded")) {
                return {
                  data: {
                    total_count: 1,
                    items: [{ number: 888, html_url: "https://github.com/owner/repo/issues/888" }],
                  },
                };
              }
              return { data: { total_count: 0, items: [] } };
            }),
          },
          issues: {
            create: createIssueMock,
            createComment: createCommentMock,
          },
          pulls: {
            get: vi.fn(),
          },
        },
        graphql: vi.fn(),
      };

      await main();

      expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("Daily per-category issue cap reached"));
      expect(createIssueMock).not.toHaveBeenCalled();
      expect(createCommentMock).toHaveBeenCalledOnce();
      expect(createCommentMock).toHaveBeenCalledWith(expect.objectContaining({ issue_number: 888 }));
    });
  });

  describe("agent failure templates", () => {
    const fs = require("fs");
    const path = require("path");
    const { renderTemplate } = require("./messages_core.cjs");
    const reportIncompleteMarker = "MARKER: cannot continue";

    it("renders report_incomplete context in both comment and issue templates", () => {
      const reportIncompleteContext = buildReportIncompleteContext([{ type: "report_incomplete", reason: reportIncompleteMarker }]);
      const templateContext = {
        run_id: "123456",
        run_url: "https://github.com/owner/repo/actions/runs/123456",
        workflow_name: "Test Workflow",
        workflow_source_url: "https://github.com/owner/repo/blob/main/.github/workflows/test.md",
        branch: "main",
        pull_request_info: "",
        secret_verification_context: "",
        credential_auth_error_context: "",
        inference_access_error_context: "",
        mcp_policy_error_context: "",
        model_not_supported_error_context: "",
        effective_tokens_rate_limit_error_context: "",
        ai_credits_rate_limit_error_context: "",
        app_token_minting_failed_context: "",
        lockdown_check_failed_context: "",
        stale_lock_file_failed_context: "",
        assignment_errors_context: "",
        assign_copilot_failure_context: "",
        create_discussion_errors_context: "",
        code_push_failure_context: "",
        repo_memory_validation_context: "",
        push_repo_memory_failure_context: "",
        missing_data_context: "",
        missing_tool_context: "",
        permission_denied_context: "",
        tool_denials_exceeded_context: "",
        report_incomplete_context: reportIncompleteContext,
        missing_safe_outputs_context: "",
        engine_failure_context: "",
        timeout_context: "",
        fork_context: "",
      };

      const commentTemplate = fs.readFileSync(path.join(__dirname, "../md/agent_failure_comment.md"), "utf8");
      const issueTemplate = fs.readFileSync(path.join(__dirname, "../md/agent_failure_issue.md"), "utf8");

      expect(renderTemplate(commentTemplate, templateContext)).toContain(reportIncompleteMarker);
      expect(renderTemplate(issueTemplate, templateContext)).toContain(reportIncompleteMarker);
    });
  });

  describe("buildCodePushFailureContext", () => {
    it("returns empty string when no errors", () => {
      expect(buildCodePushFailureContext("")).toBe("");
      expect(buildCodePushFailureContext(null)).toBe("");
      expect(buildCodePushFailureContext(undefined)).toBe("");
    });

    it("shows protected file protection section for protected file errors", () => {
      const errors = "create_pull_request:Cannot create pull request: patch modifies protected files (package.json). Set manifest-files: fallback-to-issue to create a review issue instead.";
      const result = buildCodePushFailureContext(errors);
      expect(result).toContain("🛡️ Protected Files");
      expect(result).toContain("package.json");
      expect(result).toContain("protected-files: fallback-to-issue");
      // Should NOT contain generic "Code Push Failed" for pure manifest errors
      expect(result).not.toContain("Code Push Failed");
    });

    it("shows protected file protection section for legacy 'package manifest files' error messages", () => {
      // Old error message format – must still be detected
      const errors = "create_pull_request:Cannot create pull request: patch modifies package manifest files (package.json). Set allow-manifest-files: true in your workflow to allow this.";
      const result = buildCodePushFailureContext(errors);
      expect(result).toContain("🛡️ Protected Files");
      expect(result).not.toContain("Code Push Failed");
    });

    it("shows protected file protection section for push_to_pull_request_branch errors", () => {
      const errors = "push_to_pull_request_branch:Cannot push to pull request branch: patch modifies protected files (go.mod, go.sum). Set manifest-files: fallback-to-issue to create a review issue.";
      const result = buildCodePushFailureContext(errors);
      expect(result).toContain("🛡️ Protected Files");
      expect(result).toContain("go.mod");
      expect(result).toContain("`push_to_pull_request_branch`");
      expect(result).not.toContain("Code Push Failed");
    });

    it("shows protected file protection for .github/ protected path errors", () => {
      const errors = "create_pull_request:Cannot create pull request: patch modifies protected files (.github/workflows/ci.yml). Set manifest-files: fallback-to-issue to create a review issue.";
      const result = buildCodePushFailureContext(errors);
      expect(result).toContain("🛡️ Protected Files");
      expect(result).toContain(".github/workflows/ci.yml");
    });

    it("includes PR link in protected file protection section when PR is provided", () => {
      const errors = "create_pull_request:Cannot create pull request: patch modifies package manifest files (package.json). Set allow-manifest-files: true in your workflow to allow this.";
      const pullRequest = { number: 42, html_url: "https://github.com/owner/repo/pull/42" };
      const result = buildCodePushFailureContext(errors, pullRequest);
      expect(result).toContain("🛡️ Protected Files");
      expect(result).toContain("#42");
      expect(result).toContain("https://github.com/owner/repo/pull/42");
      // PR state diagnostics should NOT appear for protected-file-only failures
      expect(result).not.toContain("PR State at Push Time");
    });

    it("shows generic code push failure section for non-manifest errors", () => {
      const errors = "push_to_pull_request_branch:Branch not found";
      const result = buildCodePushFailureContext(errors);
      expect(result).toContain("Code Push Failed");
      expect(result).toContain("Branch not found");
      expect(result).not.toContain("Protected Files");
    });

    it("shows both sections when protected file and non-protected-file errors are mixed", () => {
      const errors = [
        "create_pull_request:Cannot create pull request: patch modifies package manifest files (package.json). Set allow-manifest-files: true in your workflow to allow this.",
        "push_to_pull_request_branch:Branch not found",
      ].join("\n");
      const result = buildCodePushFailureContext(errors);
      expect(result).toContain("🛡️ Protected Files");
      expect(result).toContain("Code Push Failed");
      expect(result).toContain("package.json");
      expect(result).toContain("Branch not found");
    });

    it("includes yaml remediation snippet in protected file protection section", () => {
      const errors = "create_pull_request:Cannot create pull request: patch modifies package manifest files (requirements.txt). Set allow-manifest-files: true in your workflow to allow this.";
      const result = buildCodePushFailureContext(errors);
      expect(result).toContain("```yaml");
      expect(result).toContain("create-pull-request:");
      expect(result).toContain("protected-files: fallback-to-issue");
    });

    it("uses push-to-pull-request-branch key in yaml snippet for push type", () => {
      const errors = "push_to_pull_request_branch:Cannot push to pull request branch: patch modifies package manifest files (go.mod). Set manifest-files: fallback-to-issue in your workflow to allow this.";
      const result = buildCodePushFailureContext(errors);
      expect(result).toContain("push-to-pull-request-branch:");
      expect(result).toContain("protected-files: fallback-to-issue");
      expect(result).not.toContain("create-pull-request:");
    });

    it("includes both yaml keys when both types have protected file errors", () => {
      const errors = [
        "create_pull_request:Cannot create pull request: patch modifies package manifest files (package.json). Set manifest-files: fallback-to-issue in your workflow to allow this.",
        "push_to_pull_request_branch:Cannot push to pull request branch: patch modifies package manifest files (go.mod). Set manifest-files: fallback-to-issue in your workflow to allow this.",
      ].join("\n");
      const result = buildCodePushFailureContext(errors);
      expect(result).toContain("create-pull-request:");
      expect(result).toContain("push-to-pull-request-branch:");
    });

    // ──────────────────────────────────────────────────────
    // Patch Size Exceeded
    // ──────────────────────────────────────────────────────

    it("shows patch size exceeded section for create_pull_request patch size error", () => {
      const errors = "create_pull_request:Patch size (2048 KB) exceeds maximum allowed size (1024 KB)";
      const result = buildCodePushFailureContext(errors);
      expect(result).toContain("📦 Patch Size Exceeded");
      expect(result).toContain("create-pull-request:");
      expect(result).toContain("max-patch-size:");
      expect(result).not.toContain("Code Push Failed");
      expect(result).not.toContain("Protected Files");
    });

    it("shows patch size exceeded section for push_to_pull_request_branch patch size error", () => {
      const errors = "push_to_pull_request_branch:Patch size (3072 KB) exceeds maximum allowed size (1024 KB)";
      const result = buildCodePushFailureContext(errors);
      expect(result).toContain("📦 Patch Size Exceeded");
      expect(result).toContain("push-to-pull-request-branch:");
      expect(result).toContain("max-patch-size:");
      expect(result).not.toContain("Code Push Failed");
    });

    it("shows patch size exceeded yaml snippet with both types when both have patch size errors", () => {
      const errors = ["create_pull_request:Patch size (2048 KB) exceeds maximum allowed size (1024 KB)", "push_to_pull_request_branch:Patch size (3072 KB) exceeds maximum allowed size (1024 KB)"].join("\n");
      const result = buildCodePushFailureContext(errors);
      expect(result).toContain("📦 Patch Size Exceeded");
      expect(result).toContain("create-pull-request:");
      expect(result).toContain("push-to-pull-request-branch:");
      expect(result).toContain("max-patch-size:");
    });

    it("includes PR link in patch size exceeded section when PR is provided", () => {
      const errors = "create_pull_request:Patch size (2048 KB) exceeds maximum allowed size (1024 KB)";
      const pullRequest = { number: 99, html_url: "https://github.com/owner/repo/pull/99" };
      const result = buildCodePushFailureContext(errors, pullRequest);
      expect(result).toContain("📦 Patch Size Exceeded");
      expect(result).toContain("#99");
      expect(result).toContain("https://github.com/owner/repo/pull/99");
    });

    it("does not show patch size section for generic errors", () => {
      const errors = "push_to_pull_request_branch:Branch not found";
      const result = buildCodePushFailureContext(errors);
      expect(result).not.toContain("📦 Patch Size Exceeded");
    });

    it("shows both patch size and generic sections when mixed", () => {
      const errors = ["create_pull_request:Patch size (2048 KB) exceeds maximum allowed size (1024 KB)", "push_to_pull_request_branch:Branch not found"].join("\n");
      const result = buildCodePushFailureContext(errors);
      expect(result).toContain("📦 Patch Size Exceeded");
      expect(result).toContain("Code Push Failed");
      expect(result).toContain("Branch not found");
    });

    // ──────────────────────────────────────────────────────
    // Patch Apply Failed (merge conflict)
    // ──────────────────────────────────────────────────────

    it("shows patch apply failed section for create_pull_request patch apply error", () => {
      const errors = "create_pull_request:Failed to apply patch";
      const result = buildCodePushFailureContext(errors);
      expect(result).toContain("🔀 Patch Apply Failed");
      expect(result).toContain("merge conflict");
      expect(result).toContain("`create_pull_request`");
      expect(result).toContain("Failed to apply patch");
      // Should NOT show generic "Code Push Failed" for pure patch apply errors
      expect(result).not.toContain("Code Push Failed");
    });

    it("shows patch apply failed section for push_to_pull_request_branch patch apply error", () => {
      const errors = "push_to_pull_request_branch:Failed to apply patch";
      const result = buildCodePushFailureContext(errors);
      expect(result).toContain("🔀 Patch Apply Failed");
      expect(result).toContain("`push_to_pull_request_branch`");
      expect(result).not.toContain("Code Push Failed");
    });

    it("includes PR link in patch apply failed section when PR is provided", () => {
      const errors = "create_pull_request:Failed to apply patch";
      const pullRequest = { number: 77, html_url: "https://github.com/owner/repo/pull/77" };
      const result = buildCodePushFailureContext(errors, pullRequest);
      expect(result).toContain("🔀 Patch Apply Failed");
      expect(result).toContain("#77");
      expect(result).toContain("https://github.com/owner/repo/pull/77");
    });

    it("includes patch download instructions with run ID when runUrl is provided", () => {
      const errors = "create_pull_request:Failed to apply patch";
      const runUrl = "https://github.com/owner/repo/actions/runs/12345678";
      const result = buildCodePushFailureContext(errors, null, runUrl);
      expect(result).toContain("🔀 Patch Apply Failed");
      expect(result).toContain("gh run download 12345678");
      expect(result).toContain("-n agent");
      expect(result).toContain("/tmp/agent-");
      expect(result).toContain("git am --3way");
      expect(result).toContain(runUrl);
      // Should use progressive disclosure for the apply commands
      expect(result).toContain("<details>");
      expect(result).toContain("Apply the patch manually");
    });

    it("shows generic download instructions when runUrl is not provided", () => {
      const errors = "create_pull_request:Failed to apply patch";
      const result = buildCodePushFailureContext(errors);
      expect(result).toContain("🔀 Patch Apply Failed");
      expect(result).toContain("git am --3way");
      // No specific run ID in instructions
      expect(result).not.toContain("gh run download");
      // Should still use progressive disclosure
      expect(result).toContain("<details>");
      expect(result).toContain("Apply the patch manually");
    });

    it("shows both patch apply failed and generic sections when mixed", () => {
      const errors = ["create_pull_request:Failed to apply patch", "push_to_pull_request_branch:Branch not found"].join("\n");
      const result = buildCodePushFailureContext(errors);
      expect(result).toContain("🔀 Patch Apply Failed");
      expect(result).toContain("Code Push Failed");
      expect(result).toContain("Branch not found");
    });

    it("does not show patch apply section for generic errors", () => {
      const errors = "push_to_pull_request_branch:Branch not found";
      const result = buildCodePushFailureContext(errors);
      expect(result).not.toContain("🔀 Patch Apply Failed");
    });
  });

  // ──────────────────────────────────────────────────────
  // buildPushRepoMemoryFailureContext
  // ──────────────────────────────────────────────────────

  describe("buildPushRepoMemoryFailureContext", () => {
    it("returns empty string when no failure", () => {
      expect(buildPushRepoMemoryFailureContext(false, [], "https://example.com/run")).toBe("");
    });

    it("shows generic failure message when failure but no patch size exceeded", () => {
      const result = buildPushRepoMemoryFailureContext(true, [], "https://example.com/run");
      expect(result).toContain("> [!WARNING]");
      expect(result).toContain("Repo-Memory Push Failed");
      expect(result).toContain("https://example.com/run");
      expect(result).not.toContain("📦 Repo-Memory Patch Size Exceeded");
    });

    it("shows patch size exceeded message with front matter example when patch size exceeded", () => {
      const result = buildPushRepoMemoryFailureContext(true, ["default"], "https://example.com/run");
      expect(result).toContain("📦 Repo-Memory Patch Size Exceeded");
      expect(result).toContain("`default`");
      expect(result).toContain("max-patch-size:");
      expect(result).toContain("repo-memory:");
      expect(result).not.toContain("⚠️ Repo-Memory Push Failed");
    });

    it("includes all affected memory IDs in patch size exceeded message", () => {
      const result = buildPushRepoMemoryFailureContext(true, ["default", "secondary"], "https://example.com/run");
      expect(result).toContain("`default`");
      expect(result).toContain("`secondary`");
      expect(result).toContain("id: default");
      expect(result).toContain("id: secondary");
    });

    it("shows yaml front matter snippet for each affected memory ID", () => {
      const result = buildPushRepoMemoryFailureContext(true, ["my-memory"], "https://example.com/run");
      expect(result).toContain("```yaml");
      expect(result).toContain("repo-memory:");
      expect(result).toContain("id: my-memory");
      expect(result).toContain("max-patch-size: 51200");
    });
  });

  // ──────────────────────────────────────────────────────
  // buildAppTokenMintingFailedContext
  // ──────────────────────────────────────────────────────

  describe("buildAppTokenMintingFailedContext", () => {
    let buildAppTokenMintingFailedContext;
    const fs = require("fs");
    const path = require("path");
    const templateContent = fs.readFileSync(path.join(__dirname, "../md/app_token_minting_failed.md"), "utf8");
    const originalReadFileSync = fs.readFileSync.bind(fs);

    beforeEach(() => {
      vi.resetModules();
      // Stub readFileSync so the runtime path resolves to the source-tree template
      fs.readFileSync = (filePath, encoding) => {
        if (typeof filePath === "string" && filePath.includes("app_token_minting_failed.md")) {
          return templateContent;
        }
        return originalReadFileSync(filePath, encoding);
      };
      ({ buildAppTokenMintingFailedContext } = require("./handle_agent_failure.cjs"));
    });

    afterEach(() => {
      fs.readFileSync = originalReadFileSync;
    });

    it("returns empty string when no failure", () => {
      expect(buildAppTokenMintingFailedContext(false)).toBe("");
    });

    it("returns formatted error message when app token minting failed", () => {
      const result = buildAppTokenMintingFailedContext(true);
      expect(result).toContain("GitHub App Authentication Failed");
      expect(result).toContain("App ID");
      expect(result).toContain("private key");
      expect(result).toContain("installed");
    });

    it("includes actionable remediation steps", () => {
      const result = buildAppTokenMintingFailedContext(true);
      expect(result).toContain("required permissions");
      expect(result).toContain("https://github.github.com/gh-aw/reference/safe-outputs/");
    });
  });

  // ──────────────────────────────────────────────────────
  // buildLockdownCheckFailedContext
  // ──────────────────────────────────────────────────────

  describe("buildLockdownCheckFailedContext", () => {
    let buildLockdownCheckFailedContext;
    const fs = require("fs");
    const path = require("path");
    const templateContent = fs.readFileSync(path.join(__dirname, "../md/lockdown_check_failed.md"), "utf8");
    const originalReadFileSync = fs.readFileSync.bind(fs);

    beforeEach(() => {
      vi.resetModules();
      process.env.RUNNER_TEMP = "/nonexistent";
      // Stub readFileSync so the runtime path resolves to the source-tree template
      fs.readFileSync = (filePath, encoding) => {
        if (typeof filePath === "string" && filePath.includes("lockdown_check_failed.md")) {
          return templateContent;
        }
        return originalReadFileSync(filePath, encoding);
      };
      ({ buildLockdownCheckFailedContext } = require("./handle_agent_failure.cjs"));
    });

    afterEach(() => {
      fs.readFileSync = originalReadFileSync;
      delete process.env.RUNNER_TEMP;
    });

    it("returns empty string when no failure", () => {
      expect(buildLockdownCheckFailedContext(false)).toBe("");
    });

    it("returns formatted error message when lockdown check failed", () => {
      const result = buildLockdownCheckFailedContext(true);
      expect(result).toContain("Lockdown Check Failed");
    });

    it("includes token configuration guidance", () => {
      const result = buildLockdownCheckFailedContext(true);
      expect(result).toContain("GH_AW_GITHUB_TOKEN");
      expect(result).toContain("gh aw secrets set");
    });

    it("includes strict mode guidance", () => {
      const result = buildLockdownCheckFailedContext(true);
      expect(result).toContain("gh aw compile --strict");
    });
  });

  // ──────────────────────────────────────────────────────
  // buildStaleLockFileFailedContext
  // ──────────────────────────────────────────────────────

  describe("buildStaleLockFileFailedContext", () => {
    let buildStaleLockFileFailedContext;
    const fs = require("fs");
    const path = require("path");
    const templateContent = fs.readFileSync(path.join(__dirname, "../md/stale_lock_file_failed.md"), "utf8");
    const originalReadFileSync = fs.readFileSync.bind(fs);

    beforeEach(() => {
      vi.resetModules();
      process.env.RUNNER_TEMP = "/nonexistent";
      fs.readFileSync = (filePath, encoding) => {
        if (typeof filePath === "string" && filePath.includes("stale_lock_file_failed.md")) {
          return templateContent;
        }
        return originalReadFileSync(filePath, encoding);
      };
      ({ buildStaleLockFileFailedContext } = require("./handle_agent_failure.cjs"));
    });

    afterEach(() => {
      fs.readFileSync = originalReadFileSync;
      delete process.env.RUNNER_TEMP;
    });

    it("returns empty string when check did not fail", () => {
      expect(buildStaleLockFileFailedContext(false)).toBe("");
    });

    it("returns formatted context when stale lock file check failed", () => {
      const result = buildStaleLockFileFailedContext(true);
      expect(result).toBeTruthy();
      expect(result.length).toBeGreaterThan(0);
    });

    it("includes recompile guidance", () => {
      const result = buildStaleLockFileFailedContext(true);
      expect(result).toContain("gh aw compile");
    });

    it("includes guidance on how to disable the check", () => {
      const result = buildStaleLockFileFailedContext(true);
      expect(result).toContain("stale-check: false");
    });

    it("includes debug logging guidance", () => {
      const result = buildStaleLockFileFailedContext(true);
      expect(result).toContain("[hash-debug]");
    });
  });

  // ──────────────────────────────────────────────────────
  // buildTimeoutContext
  // ──────────────────────────────────────────────────────

  describe("buildTimeoutContext", () => {
    let buildTimeoutContext;
    const fs = require("fs");
    const path = require("path");
    const templateContent = fs.readFileSync(path.join(__dirname, "../md/agent_timeout.md"), "utf8");
    const originalReadFileSync = fs.readFileSync.bind(fs);

    beforeEach(() => {
      vi.resetModules();
      process.env.RUNNER_TEMP = "/nonexistent";
      // Stub readFileSync so the runtime path resolves to the source-tree template
      fs.readFileSync = (filePath, encoding) => {
        if (typeof filePath === "string" && filePath.includes("agent_timeout.md")) {
          return templateContent;
        }
        return originalReadFileSync(filePath, encoding);
      };
      ({ buildTimeoutContext } = require("./handle_agent_failure.cjs"));
    });

    afterEach(() => {
      fs.readFileSync = originalReadFileSync;
      delete process.env.RUNNER_TEMP;
    });

    it("returns empty string when not timed out", () => {
      expect(buildTimeoutContext(false, "20")).toBe("");
      expect(buildTimeoutContext(false, "")).toBe("");
    });

    it("returns formatted error message when timed out", () => {
      const result = buildTimeoutContext(true, "20");
      expect(result).toContain("Agent Timed Out");
      expect(result).toContain("20");
      expect(result).toContain("30");
      expect(result).toContain("timeout-minutes");
    });

    it("uses default of 20 minutes when timeoutMinutes is empty", () => {
      const result = buildTimeoutContext(true, "");
      expect(result).toContain("20");
      expect(result).toContain("30");
    });

    it("suggests current + 10 minutes", () => {
      const result = buildTimeoutContext(true, "45");
      expect(result).toContain("45");
      expect(result).toContain("55");
    });
  });

  // ──────────────────────────────────────────────────────
  // timeout classification (isTimedOut logic in main)
  // ──────────────────────────────────────────────────────

  describe("timeout classification", () => {
    // Mirrors the classification logic in main():
    //   const isTimedOut = agentConclusion === "timed_out" || agenticEngineTimeout;
    // This ensures step-level timeouts (detected via signal in the engine log)
    // are treated as timeouts even when agentConclusion is "failure".
    function classifyTimeout(agentConclusion, agenticEngineTimeout) {
      return agentConclusion === "timed_out" || agenticEngineTimeout;
    }

    it("detects job-level timeout (agentConclusion === 'timed_out')", () => {
      expect(classifyTimeout("timed_out", false)).toBe(true);
    });

    it("detects step-level timeout (agentConclusion === 'failure' with agenticEngineTimeout)", () => {
      expect(classifyTimeout("failure", true)).toBe(true);
    });

    it("detects timeout when both indicators are present", () => {
      expect(classifyTimeout("timed_out", true)).toBe(true);
    });

    it("does not flag timeout for plain failure without engine timeout signal", () => {
      expect(classifyTimeout("failure", false)).toBe(false);
    });

    it("does not flag timeout for successful completion", () => {
      expect(classifyTimeout("success", false)).toBe(false);
    });

    it("does not flag timeout for cancelled job", () => {
      expect(classifyTimeout("cancelled", false)).toBe(false);
    });
  });

  // ──────────────────────────────────────────────────────
  // buildEngineFailureContext
  // ──────────────────────────────────────────────────────

  describe("buildEngineFailureContext", () => {
    let buildEngineFailureContext;
    const fs = require("fs");
    const path = require("path");
    const os = require("os");

    /** @type {string} */
    let tmpDir;
    /** @type {string} */
    let stdioLogPath;

    /** @type {string} */
    let promptsDir;

    beforeEach(() => {
      vi.resetModules();
      tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "aw-test-"));
      stdioLogPath = path.join(tmpDir, "agent-stdio.log");
      process.env.GH_AW_OTEL_JSONL_PATH = path.join(tmpDir, "otel.jsonl");
      promptsDir = path.join(tmpDir, "gh-aw", "prompts");
      fs.mkdirSync(promptsDir, { recursive: true });
      process.env.GH_AW_AGENT_OUTPUT = path.join(tmpDir, "agent_output.json");
      process.env.RUNNER_TEMP = tmpDir;
      ({ buildEngineFailureContext } = require("./handle_agent_failure.cjs"));
    });

    afterEach(() => {
      delete process.env.GH_AW_AGENT_OUTPUT;
      delete process.env.GH_AW_ENGINE_ID;
      delete process.env.GH_AW_OTEL_JSONL_PATH;
      delete process.env.RUNNER_TEMP;
      // Clean up temp dir
      if (fs.existsSync(tmpDir)) {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("returns empty string when log file does not exist", () => {
      // stdioLogPath not written — file does not exist
      expect(buildEngineFailureContext()).toBe("");
    });

    it("returns empty string when log file is empty", () => {
      fs.writeFileSync(stdioLogPath, "");
      expect(buildEngineFailureContext()).toBe("");
    });

    it("returns empty string when log file contains only whitespace", () => {
      fs.writeFileSync(stdioLogPath, "   \n\n   ");
      expect(buildEngineFailureContext()).toBe("");
    });

    it("detects ERROR: prefix pattern (Codex/generic CLI)", () => {
      fs.writeFileSync(stdioLogPath, "ERROR: quota exceeded\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("Engine Failure");
      expect(result).toContain("quota exceeded");
      expect(result).toContain("Error details:");
      expect(result).toContain("> [!WARNING]");
    });

    it("returns dedicated context for engine 429/rate-limit failures in stdio logs", () => {
      fs.writeFileSync(stdioLogPath, "Failed to get response from the AI model; retried 5 times. Last error: CAPIError: 429 429 Sorry, you've exceeded your rate limit for utility models.\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("Engine Rate Limited (HTTP 429)");
      expect(result).toContain("OTLP telemetry");
      expect(result).not.toContain("Last agent output");
    });

    it("suppresses generic 429 context for maximum effective tokens exceeded failures", () => {
      fs.writeFileSync(stdioLogPath, "Failed to get response from the AI model; retried 5 times. Last error: CAPIError: 429 Maximum effective tokens exceeded (25296477.30 / 25000000).\n");
      expect(buildEngineFailureContext()).toBe("");
    });

    it("returns dedicated context when 429/rate-limit is only present in OTLP mirror", () => {
      fs.writeFileSync(stdioLogPath, "Agent terminated unexpectedly without clear error details\n");
      fs.writeFileSync(
        process.env.GH_AW_OTEL_JSONL_PATH,
        JSON.stringify({
          resourceSpans: [
            {
              scopeSpans: [
                {
                  spans: [
                    {
                      name: "gh-aw.agent.conclusion",
                      status: { code: 2, message: "agent failure: CAPIError: 429 Too Many Requests" },
                    },
                  ],
                },
              ],
            },
          ],
        }) + "\n"
      );
      const result = buildEngineFailureContext();
      expect(result).toContain("Engine Rate Limited (HTTP 429)");
      expect(result).toContain("OTLP telemetry");
      expect(result).not.toContain("Last agent output");
    });

    it("detects Error: prefix pattern (Node.js style)", () => {
      fs.writeFileSync(stdioLogPath, "Error: connect ECONNREFUSED 127.0.0.1:8080\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("Engine Failure");
      expect(result).toContain("connect ECONNREFUSED 127.0.0.1:8080");
    });

    it("extracts AWF startup errors from dependency lines and container startup failures", () => {
      const lines = [
        " Container awf-squid  Error",
        "dependency failed to start: container awf-squid is unhealthy",
        "[ERROR] Failed to start containers: Error: Command failed with exit code 1: docker compose up -d --pull never",
        "  stdout: undefined,",
        "  stderr: undefined,",
      ];
      fs.writeFileSync(stdioLogPath, lines.join("\n") + "\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("Engine Failure");
      expect(result).toContain("dependency failed to start: container awf-squid is unhealthy");
      expect(result).toContain("Failed to start containers: Error: Command failed with exit code 1: docker compose up -d --pull never");
      expect(result).not.toContain("Last agent output");
      expect(result).not.toContain("stdout: undefined");
      expect(result).not.toContain("stderr: undefined");
    });

    it("detects Fatal: prefix pattern", () => {
      fs.writeFileSync(stdioLogPath, "Fatal: out of memory\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("Engine Failure");
      expect(result).toContain("out of memory");
    });

    it("detects FATAL: prefix pattern", () => {
      fs.writeFileSync(stdioLogPath, "FATAL: unexpected shutdown\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("Engine Failure");
      expect(result).toContain("unexpected shutdown");
    });

    it("detects panic: prefix pattern (Go runtime)", () => {
      fs.writeFileSync(stdioLogPath, "panic: runtime error: index out of range\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("Engine Failure");
      expect(result).toContain("runtime error: index out of range");
    });

    it("detects Reconnecting pattern", () => {
      fs.writeFileSync(stdioLogPath, "Reconnecting... 1/3 (connection lost)\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("Engine Failure");
      expect(result).toContain("connection lost");
    });

    it("deduplicates repeated error messages", () => {
      fs.writeFileSync(stdioLogPath, "ERROR: quota exceeded\nERROR: quota exceeded\nERROR: quota exceeded\n");
      const result = buildEngineFailureContext();
      const count = (result.match(/quota exceeded/g) || []).length;
      expect(count).toBe(1);
    });

    it("collects multiple distinct error messages", () => {
      fs.writeFileSync(stdioLogPath, "ERROR: quota exceeded\nERROR: auth failed\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("quota exceeded");
      expect(result).toContain("auth failed");
    });

    it("falls back to last lines when no known error patterns match", () => {
      const logLines = ["Starting agent...", "Running tool: list_branches", '{"branches": ["main"]}', "Running tool: get_file_contents", "Agent interrupted"];
      fs.writeFileSync(stdioLogPath, logLines.join("\n") + "\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("Engine Failure");
      expect(result).toContain("Last agent output");
      expect(result).toContain("Agent interrupted");
    });

    it("fallback includes at most 10 non-empty lines", () => {
      const lines = Array.from({ length: 20 }, (_, i) => `line ${i + 1}`);
      fs.writeFileSync(stdioLogPath, lines.join("\n") + "\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("line 20");
      expect(result).toContain("line 11");
      // Lines 1-10 should not appear in the tail
      expect(result).not.toContain("line 10\n");
      expect(result).not.toContain("line 1\n");
    });

    it("fallback ignores empty lines when counting tail", () => {
      const lines = ["line 1", "", "line 2", "", "line 3", "", "", "line 4"];
      fs.writeFileSync(stdioLogPath, lines.join("\n") + "\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("Last agent output");
      expect(result).toContain("line 4");
      expect(result).toContain("line 1");
    });

    it("shows startup-failure message when log contains only AWF infrastructure lines", () => {
      // This is the exact pattern from the Apr 8 systemic failure incident:
      // containers stop cleanly, engine exits with code 1, no substantive output produced.
      const infraLines = [
        " Container awf-squid  Removing",
        " Container awf-squid  Removed",
        "[SUCCESS] Containers stopped successfully",
        "[INFO] Agent session state preserved at: /tmp/awf-agent-session-state-abc123",
        "[INFO] API proxy logs available at: /tmp/gh-aw/sandbox/firewall/logs/api-proxy-logs",
        "[ERROR] Command completed with exit code: 1",
        "[WARN] Command completed with exit code: 1",
        "Process exiting with code: 1",
      ];
      fs.writeFileSync(stdioLogPath, infraLines.join("\n") + "\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("Engine Failure");
      expect(result).toContain("terminated before producing output");
      expect(result).toContain("transient infrastructure issue");
      // Infrastructure lines should NOT appear as "Last agent output"
      expect(result).not.toContain("Last agent output");
      expect(result).not.toContain("awf-squid");
      expect(result).not.toContain("Command completed with exit code");
      expect(result).not.toContain("[ERROR]");
      expect(result).not.toContain("Process exiting with code");
    });

    it("filters infrastructure lines from fallback tail when mixed with real agent output", () => {
      // Real agent output followed by AWF infrastructure shutdown lines.
      // Only the real agent output should appear in the fallback.
      const logLines = [
        "Starting agent...",
        "● list_files",
        "  └ Found 12 files",
        " Container awf-squid  Removing",
        " Container awf-squid  Removed",
        "[SUCCESS] Containers stopped successfully",
        "[WARN] Command completed with exit code: 1",
        "Process exiting with code: 1",
      ];
      fs.writeFileSync(stdioLogPath, logLines.join("\n") + "\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("Last agent output");
      expect(result).toContain("Starting agent");
      expect(result).toContain("Found 12 files");
      // Infrastructure lines must be excluded from the displayed output
      expect(result).not.toContain("awf-squid");
      expect(result).not.toContain("Command completed with exit code");
      expect(result).not.toContain("Process exiting with code");
    });

    it("includes [entrypoint] and [health-check] infra lines in the infra filter", () => {
      // AWF container scripts emit lowercase [entrypoint] and [health-check] prefixes.
      // The INFRA_LINE_RE pattern is intentionally case-sensitive and matches exactly
      // the casing produced by each AWF component (consistent with parse_copilot_log.cjs).
      const lines = ["[entrypoint] Starting firewall...", "[health-check] Proxy ready", "[INFO] API proxy logs available at: /tmp/gh-aw/logs", "Process exiting with code: 1"];
      fs.writeFileSync(stdioLogPath, lines.join("\n") + "\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("Engine Failure");
      expect(result).toContain("terminated before producing output");
      // None of the infra lines should appear
      expect(result).not.toContain("entrypoint");
      expect(result).not.toContain("health-check");
      expect(result).not.toContain("API proxy");
    });

    it("includes engine ID in startup-failure message", () => {
      process.env.GH_AW_ENGINE_ID = "copilot";
      vi.resetModules();
      ({ buildEngineFailureContext } = require("./handle_agent_failure.cjs"));
      const infraLines = ["[WARN] Command completed with exit code: 1", "Process exiting with code: 1"];
      fs.writeFileSync(stdioLogPath, infraLines.join("\n") + "\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("`copilot` engine");
      expect(result).toContain("terminated before producing output");
      // Copilot-specific status page guidance
      expect(result).toContain("GitHub Copilot status page");
    });

    it("shows provider-agnostic status page guidance for non-copilot engines", () => {
      process.env.GH_AW_ENGINE_ID = "claude";
      vi.resetModules();
      ({ buildEngineFailureContext } = require("./handle_agent_failure.cjs"));
      const infraLines = ["[WARN] Command completed with exit code: 1", "Process exiting with code: 1"];
      fs.writeFileSync(stdioLogPath, infraLines.join("\n") + "\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("`claude` engine");
      expect(result).toContain("terminated before producing output");
      // Generic guidance for non-copilot engines
      expect(result).toContain("provider status page");
      expect(result).not.toContain("GitHub Copilot status page");
    });

    it("includes engine ID in failure message when GH_AW_ENGINE_ID is set", () => {
      process.env.GH_AW_ENGINE_ID = "copilot";
      vi.resetModules();
      ({ buildEngineFailureContext } = require("./handle_agent_failure.cjs"));
      fs.writeFileSync(stdioLogPath, "ERROR: quota exceeded\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("`copilot` engine");
    });

    it("includes engine ID in fallback message when GH_AW_ENGINE_ID is set", () => {
      process.env.GH_AW_ENGINE_ID = "claude";
      vi.resetModules();
      ({ buildEngineFailureContext } = require("./handle_agent_failure.cjs"));
      fs.writeFileSync(stdioLogPath, "Agent did something unexpected\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("`claude` engine");
    });

    it("uses generic 'AI engine' label when GH_AW_ENGINE_ID is not set", () => {
      fs.writeFileSync(stdioLogPath, "ERROR: connection reset\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("The AI engine");
    });

    it("returns dedicated cyber_policy_violation message when template exists", () => {
      const templateContent = "**OpenAI Cyber Policy Violation**: The Codex engine was blocked by OpenAI's safety policy.";
      fs.writeFileSync(path.join(promptsDir, "cyber_policy_violation.md"), templateContent);
      fs.writeFileSync(stdioLogPath, "ERROR: cyber_policy_violation\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("Cyber Policy Violation");
      expect(result).not.toContain("Engine Failure");
      expect(result).not.toContain("cyber_policy_violation");
    });

    it("falls back to generic message when cyber_policy_violation template is missing", () => {
      // No template file written — promptsDir exists but template is absent
      fs.writeFileSync(stdioLogPath, "ERROR: cyber_policy_violation\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("Engine Failure");
      expect(result).toContain("cyber_policy_violation");
    });

    it("returns dedicated message when cyber_policy_violation appears among multiple errors", () => {
      const templateContent = "**OpenAI Cyber Policy Violation**: The Codex engine was blocked by OpenAI's safety policy.";
      fs.writeFileSync(path.join(promptsDir, "cyber_policy_violation.md"), templateContent);
      fs.writeFileSync(stdioLogPath, "ERROR: connection reset\nERROR: cyber_policy_violation\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("Cyber Policy Violation");
      expect(result).not.toContain("Engine Failure");
    });
  });

  // ──────────────────────────────────────────────────────
  // buildMCPPolicyErrorContext
  // ──────────────────────────────────────────────────────

  describe("buildMCPPolicyErrorContext", () => {
    let buildMCPPolicyErrorContext;
    const fs = require("fs");
    const path = require("path");
    const os = require("os");

    /** @type {string} */
    let tmpDir;

    /** @type {string} */
    let promptsDir;

    beforeEach(() => {
      vi.resetModules();
      tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "aw-test-mcp-"));
      promptsDir = path.join(tmpDir, "gh-aw", "prompts");
      fs.mkdirSync(promptsDir, { recursive: true });
      process.env.RUNNER_TEMP = tmpDir;
      ({ buildMCPPolicyErrorContext } = require("./handle_agent_failure.cjs"));
    });

    afterEach(() => {
      delete process.env.RUNNER_TEMP;
      if (fs.existsSync(tmpDir)) {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("returns empty string when no MCP policy error", () => {
      expect(buildMCPPolicyErrorContext(false)).toBe("");
    });

    it("returns template content when MCP policy error and template exists", () => {
      const templateContent = "\n**🔒 MCP Servers Blocked by Policy**: Test message.\n";
      fs.writeFileSync(path.join(promptsDir, "mcp_policy_error.md"), templateContent);
      const result = buildMCPPolicyErrorContext(true);
      expect(result).toContain("MCP Servers Blocked by Policy");
    });

    it("includes link to official documentation when template exists", () => {
      const templateContent = "**🔒 MCP Servers Blocked by Policy**: See [docs](https://docs.github.com/en/copilot/how-tos/administer-copilot/manage-mcp-usage/configure-mcp-server-access).\n";
      fs.writeFileSync(path.join(promptsDir, "mcp_policy_error.md"), templateContent);
      const result = buildMCPPolicyErrorContext(true);
      expect(result).toContain("docs.github.com/en/copilot/how-tos/administer-copilot/manage-mcp-usage/configure-mcp-server-access");
    });

    it("returns inline fallback message when template is missing", () => {
      // No template file written
      const result = buildMCPPolicyErrorContext(true);
      expect(result).toContain("MCP Servers Blocked by Policy");
      expect(result).toContain("configure-mcp-server-access");
    });
  });

  // buildModelNotSupportedErrorContext
  // ──────────────────────────────────────────────────────

  describe("buildModelNotSupportedErrorContext", () => {
    let buildModelNotSupportedErrorContext;
    const fs = require("fs");
    const path = require("path");
    const os = require("os");

    /** @type {string} */
    let tmpDir;

    /** @type {string} */
    let promptsDir;

    beforeEach(() => {
      vi.resetModules();
      tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "aw-test-model-not-supported-"));
      promptsDir = path.join(tmpDir, "gh-aw", "prompts");
      fs.mkdirSync(promptsDir, { recursive: true });
      process.env.RUNNER_TEMP = tmpDir;
      ({ buildModelNotSupportedErrorContext } = require("./handle_agent_failure.cjs"));
    });

    afterEach(() => {
      delete process.env.RUNNER_TEMP;
      if (fs.existsSync(tmpDir)) {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("returns empty string when no model-not-supported error", () => {
      expect(buildModelNotSupportedErrorContext(false)).toBe("");
    });

    it("returns template content when model-not-supported error and template exists", () => {
      const templateContent = "\n**🚫 Model Not Supported**: Test message.\n";
      fs.writeFileSync(path.join(promptsDir, "model_not_supported_error.md"), templateContent);
      const result = buildModelNotSupportedErrorContext(true);
      expect(result).toContain("Model Not Supported");
    });

    it("returns inline fallback message when template is missing", () => {
      // No template file written
      const result = buildModelNotSupportedErrorContext(true);
      expect(result).toContain("Model Not Supported");
      expect(result).toContain("gpt-5-mini");
    });
  });

  // buildMissingDataContext
  // ──────────────────────────────────────────────────────

  describe("buildMissingDataContext", () => {
    let buildMissingDataContext;
    const fs = require("fs");
    const path = require("path");
    const os = require("os");

    /** @type {string} */
    let tmpDir;

    /** @type {string} */
    let promptsDir;

    beforeEach(() => {
      vi.resetModules();
      tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "aw-test-missing-data-"));
      promptsDir = path.join(tmpDir, "gh-aw", "prompts");
      fs.mkdirSync(promptsDir, { recursive: true });
      process.env.RUNNER_TEMP = tmpDir;
      process.env.GH_AW_AGENT_OUTPUT = path.join(tmpDir, "agent_output.json");
      ({ buildMissingDataContext } = require("./handle_agent_failure.cjs"));
    });

    afterEach(() => {
      delete process.env.RUNNER_TEMP;
      delete process.env.GH_AW_AGENT_OUTPUT;
      if (fs.existsSync(tmpDir)) {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("returns empty string when agent output file does not exist", () => {
      expect(buildMissingDataContext(false)).toBe("");
      expect(buildMissingDataContext(true)).toBe("");
    });

    it("returns empty string when agent output has no missing_data items", () => {
      fs.writeFileSync(path.join(tmpDir, "agent_output.json"), JSON.stringify({ items: [{ type: "noop", reason: "done" }] }));
      vi.resetModules();
      ({ buildMissingDataContext } = require("./handle_agent_failure.cjs"));
      expect(buildMissingDataContext(false)).toBe("");
      expect(buildMissingDataContext(true)).toBe("");
    });

    it("returns missing data context without cache warning when cacheMemoryEnabled is false", () => {
      fs.writeFileSync(
        path.join(tmpDir, "agent_output.json"),
        JSON.stringify({
          items: [{ type: "missing_data", data_type: "cache_memory", reason: "cache_memory_miss" }],
        })
      );
      vi.resetModules();
      ({ buildMissingDataContext } = require("./handle_agent_failure.cjs"));
      const result = buildMissingDataContext(false);
      expect(result).toContain("Missing Data Reported");
      expect(result).toContain("cache\\_memory"); // data_type after markdown escaping
      expect(result).not.toContain("Cache Configuration Problem");
    });

    it("appends cache configuration warning when cacheMemoryEnabled is true and cache_memory_miss item present", () => {
      fs.writeFileSync(
        path.join(tmpDir, "agent_output.json"),
        JSON.stringify({
          items: [{ type: "missing_data", data_type: "cache_memory", reason: "cache_memory_miss" }],
        })
      );
      const templateContent =
        "> [!WARNING]\n" +
        "> <details>\n" +
        "> <summary>Cache Configuration Problem: cache miss detected despite cache-memory being configured.</summary>\n>\n" +
        "> Review the [cache-memory configuration](https://github.github.com/gh-aw/reference/cache-memory/) and ensure the agent prompt correctly references files inside the cache directory.\n>\n" +
        "> **File naming convention:** Cache files are stored at `/tmp/gh-aw/cache-memory/`.\n>\n" +
        "> </details>";
      fs.writeFileSync(path.join(promptsDir, "cache_memory_miss.md"), templateContent);
      vi.resetModules();
      ({ buildMissingDataContext } = require("./handle_agent_failure.cjs"));
      const result = buildMissingDataContext(true);
      expect(result).not.toContain("Missing Data Reported");
      expect(result).toContain("Cache Configuration Problem");
      expect(result).toContain("> [!WARNING]");
      expect(result).toContain("<summary>");
      expect(result).toContain("<details>");
      expect(result).toContain("/gh-aw/reference/cache-memory/");
      expect(result).toContain("File naming convention");
    });

    it("captures reason-only missing_data items (no data_type) and detects cache miss", () => {
      // Agents may emit missing_data with only reason (no data_type) — ensure it is still captured
      fs.writeFileSync(
        path.join(tmpDir, "agent_output.json"),
        JSON.stringify({
          items: [{ type: "missing_data", reason: "cache_memory_miss" }],
        })
      );
      const templateContent = "> [!WARNING]\n" + "> <details>\n" + "> <summary>Cache Configuration Problem: cache miss detected despite cache-memory being configured.</summary>\n>\n" + "> Details here.\n>\n" + "> </details>";
      fs.writeFileSync(path.join(promptsDir, "cache_memory_miss.md"), templateContent);
      vi.resetModules();
      ({ buildMissingDataContext } = require("./handle_agent_failure.cjs"));
      const result = buildMissingDataContext(true);
      expect(result).not.toContain("Missing Data Reported");
      expect(result).toContain("Cache Configuration Problem");
      expect(result).toContain("> [!WARNING]");
      expect(result).toContain("<summary>");
      expect(result).toContain("<details>");
    });

    it("shows generic missing-data context for non-cache items while still appending cache warning", () => {
      fs.writeFileSync(
        path.join(tmpDir, "agent_output.json"),
        JSON.stringify({
          items: [
            { type: "missing_data", data_type: "cache_memory", reason: "cache_memory_miss" },
            { type: "missing_data", data_type: "user_data", reason: "not_provided" },
          ],
        })
      );
      const templateContent = "> [!WARNING]\n> <details>\n> <summary>Cache Configuration Problem: cache miss detected despite cache-memory being configured.</summary>\n>\n> Details here.\n>\n> </details>";
      fs.writeFileSync(path.join(promptsDir, "cache_memory_miss.md"), templateContent);
      vi.resetModules();
      ({ buildMissingDataContext } = require("./handle_agent_failure.cjs"));
      const result = buildMissingDataContext(true);
      expect(result).toContain("Missing Data Reported");
      expect(result).toContain("user\\_data");
      expect(result).toContain("Cache Configuration Problem");
      expect(result).not.toContain("cache\\_memory");
      expect(result).not.toContain("cache\\_memory\\_miss");
    });

    it("keeps cache warning and report_incomplete context without duplicate missing-data section", () => {
      fs.writeFileSync(
        path.join(tmpDir, "agent_output.json"),
        JSON.stringify({
          items: [
            { type: "missing_data", data_type: "cache_memory", reason: "cache_memory_miss" },
            { type: "report_incomplete", reason: "mcp_crash" },
          ],
        })
      );
      const templateContent = "> [!WARNING]\n> <details>\n> <summary>Cache Configuration Problem: cache miss detected despite cache-memory being configured.</summary>\n>\n> Details here.\n>\n> </details>";
      fs.writeFileSync(path.join(promptsDir, "cache_memory_miss.md"), templateContent);
      vi.resetModules();
      const { buildMissingDataContext: buildMissingDataContextFn, buildReportIncompleteContext } = require("./handle_agent_failure.cjs");
      const missingDataResult = buildMissingDataContextFn(true);
      const reportIncompleteResult = buildReportIncompleteContext();
      expect(missingDataResult).not.toContain("Missing Data Reported");
      expect(missingDataResult).toContain("Cache Configuration Problem");
      expect(reportIncompleteResult).toContain("Task Could Not Be Completed");
      expect(reportIncompleteResult).toContain("mcp_crash");
    });

    it("does not append cache warning for unrelated missing_data reasons when cacheMemoryEnabled is true", () => {
      fs.writeFileSync(
        path.join(tmpDir, "agent_output.json"),
        JSON.stringify({
          items: [{ type: "missing_data", data_type: "user_data", reason: "not_provided" }],
        })
      );
      vi.resetModules();
      ({ buildMissingDataContext } = require("./handle_agent_failure.cjs"));
      const result = buildMissingDataContext(true);
      expect(result).toContain("Missing Data Reported");
      expect(result).not.toContain("Cache Configuration Problem");
    });
  });

  // ──────────────────────────────────────────────────────
  // buildMissingToolContext
  // ──────────────────────────────────────────────────────

  describe("buildMissingToolContext", () => {
    let buildMissingToolContext;
    const fs = require("fs");
    const path = require("path");
    const os = require("os");

    /** @type {string} */
    let tmpDir;

    beforeEach(() => {
      vi.resetModules();
      tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "aw-test-missing-tool-"));
      process.env.RUNNER_TEMP = tmpDir;
      process.env.GH_AW_AGENT_OUTPUT = path.join(tmpDir, "agent_output.json");
      ({ buildMissingToolContext } = require("./handle_agent_failure.cjs"));
    });

    afterEach(() => {
      delete process.env.RUNNER_TEMP;
      delete process.env.GH_AW_AGENT_OUTPUT;
      if (fs.existsSync(tmpDir)) {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("returns empty string when agent output file does not exist", () => {
      expect(buildMissingToolContext()).toBe("");
    });

    it("returns empty string when agent output has no missing_tool items", () => {
      fs.writeFileSync(path.join(tmpDir, "agent_output.json"), JSON.stringify({ items: [{ type: "noop", reason: "done" }] }));
      vi.resetModules();
      ({ buildMissingToolContext } = require("./handle_agent_failure.cjs"));
      expect(buildMissingToolContext()).toBe("");
    });

    it("returns missing tool context with tool name and reason", () => {
      fs.writeFileSync(
        path.join(tmpDir, "agent_output.json"),
        JSON.stringify({
          items: [{ type: "missing_tool", tool: "bash", reason: "bash is not available" }],
        })
      );
      vi.resetModules();
      ({ buildMissingToolContext } = require("./handle_agent_failure.cjs"));
      const result = buildMissingToolContext();
      expect(result).toContain("Missing Tools Reported");
      expect(result).toContain("bash");
      expect(result).toContain("bash is not available");
      expect(result).toContain("> [!WARNING]");
      expect(result).not.toContain("**⚠️ Missing Tools Reported**");
    });

    it("returns missing tool context for tool with alternatives", () => {
      fs.writeFileSync(
        path.join(tmpDir, "agent_output.json"),
        JSON.stringify({
          items: [{ type: "missing_tool", tool: "docker", reason: "docker is not installed", alternatives: "podman" }],
        })
      );
      vi.resetModules();
      ({ buildMissingToolContext } = require("./handle_agent_failure.cjs"));
      const result = buildMissingToolContext();
      expect(result).toContain("Missing Tools Reported");
      expect(result).toContain("docker");
      expect(result).toContain("podman");
    });

    it("skips missing_tool items without a reason field", () => {
      fs.writeFileSync(
        path.join(tmpDir, "agent_output.json"),
        JSON.stringify({
          items: [{ type: "missing_tool", tool: "bash" }],
        })
      );
      vi.resetModules();
      ({ buildMissingToolContext } = require("./handle_agent_failure.cjs"));
      expect(buildMissingToolContext()).toBe("");
    });

    it("handles multiple missing_tool items", () => {
      fs.writeFileSync(
        path.join(tmpDir, "agent_output.json"),
        JSON.stringify({
          items: [
            { type: "missing_tool", tool: "tool1", reason: "not available" },
            { type: "missing_tool", tool: "tool2", reason: "not installed" },
          ],
        })
      );
      vi.resetModules();
      ({ buildMissingToolContext } = require("./handle_agent_failure.cjs"));
      const result = buildMissingToolContext();
      expect(result).toContain("Missing Tools Reported");
      expect(result).toContain("tool1");
      expect(result).toContain("tool2");
    });
  });

  // ──────────────────────────────────────────────────────
  // buildPermissionDeniedContext
  // ──────────────────────────────────────────────────────

  describe("buildPermissionDeniedContext", () => {
    let buildPermissionDeniedContext;
    const fs = require("fs");
    const path = require("path");
    const os = require("os");

    /** @type {string} */
    let tmpDir;

    beforeEach(() => {
      vi.resetModules();
      tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "aw-test-permission-denied-"));
      process.env.RUNNER_TEMP = tmpDir;
      process.env.GH_AW_AGENT_OUTPUT = path.join(tmpDir, "agent_output.json");
      ({ buildPermissionDeniedContext } = require("./handle_agent_failure.cjs"));
    });

    afterEach(() => {
      delete process.env.RUNNER_TEMP;
      delete process.env.GH_AW_AGENT_OUTPUT;
      if (fs.existsSync(tmpDir)) {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("returns empty string when agent output file does not exist", () => {
      expect(buildPermissionDeniedContext()).toBe("");
    });

    it("returns empty string when there are no tool/permission items", () => {
      fs.writeFileSync(path.join(tmpDir, "agent_output.json"), JSON.stringify({ items: [{ type: "noop", reason: "done" }] }));
      expect(buildPermissionDeniedContext()).toBe("");
    });

    it("returns empty string when tool/permission item has no denied_commands", () => {
      fs.writeFileSync(
        path.join(tmpDir, "agent_output.json"),
        JSON.stringify({
          items: [{ type: "missing_tool", tool: "tool/permission", reason: "permission denied", denied_commands: [] }],
        })
      );
      expect(buildPermissionDeniedContext()).toBe("");
    });

    it("returns inline fallback when template is not available (RUNNER_TEMP not set)", () => {
      delete process.env.RUNNER_TEMP;
      const items = [{ type: "missing_tool", tool: "tool/permission", reason: "permission denied", denied_commands: ["go version 2>&1"] }];
      const result = buildPermissionDeniedContext(items);
      expect(result).toContain("go version 2>&1");
      expect(result).toContain("Repeated Permission Denied");
    });

    it("renders fallback with denied commands listed", () => {
      delete process.env.RUNNER_TEMP;
      const items = [{ type: "missing_tool", tool: "tool/permission", reason: "permission denied", denied_commands: ["go version 2>&1", "ls /usr/local/go/bin/go"] }];
      const result = buildPermissionDeniedContext(items);
      expect(result).toContain("`go version 2>&1`");
      expect(result).toContain("`ls /usr/local/go/bin/go`");
    });

    it("deduplicates denied commands across multiple tool/permission items", () => {
      delete process.env.RUNNER_TEMP;
      const items = [
        { type: "missing_tool", tool: "tool/permission", reason: "permission denied", denied_commands: ["go version 2>&1", "ls /usr/local/go/bin/go"] },
        { type: "missing_tool", tool: "tool/permission", reason: "permission denied", denied_commands: ["go version 2>&1", "which go"] },
      ];
      const result = buildPermissionDeniedContext(items);
      // "go version 2>&1" should appear exactly once (deduplicated) in the denied commands list
      const listOccurrences = (result.match(/`go version 2>&1`/g) || []).length;
      expect(listOccurrences).toBe(1);
      expect(result).toContain("`ls /usr/local/go/bin/go`");
      expect(result).toContain("`which go`");
    });

    it("renders template when permission_denied_context.md is available", () => {
      const promptsDir = path.join(tmpDir, "gh-aw", "prompts");
      fs.mkdirSync(promptsDir, { recursive: true });
      fs.copyFileSync(path.join(__dirname, "../md/permission_denied_context.md"), path.join(promptsDir, "permission_denied_context.md"));
      const items = [{ type: "missing_tool", tool: "tool/permission", reason: "permission denied", denied_commands: ["go version 2>&1"] }];
      const result = buildPermissionDeniedContext(items, "my-workflow");
      expect(result).toContain("go version 2>&1");
      expect(result).toContain("my-workflow");
      expect(result).toContain("Repeated Permission Denied");
    });
  });

  describe("tool_denials_exceeded context", () => {
    const fs = require("fs");
    const path = require("path");
    const os = require("os");
    let loadToolDenialsExceededEvents;
    let buildToolDenialsExceededContext;
    /** @type {string} */
    let tmpDir;

    beforeEach(() => {
      vi.resetModules();
      try {
        fs.rmSync(path.join(os.tmpdir(), "gh-aw", "sandbox", "agent", "logs", "copilot-session-state"), { recursive: true, force: true });
      } catch {}
      tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "aw-test-tool-denials-exceeded-"));
      process.env.RUNNER_TEMP = tmpDir;
      ({ loadToolDenialsExceededEvents, buildToolDenialsExceededContext } = require("./handle_agent_failure.cjs"));
    });

    afterEach(() => {
      delete process.env.RUNNER_TEMP;
      if (fs.existsSync(tmpDir)) {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
      try {
        fs.rmSync(path.join(os.tmpdir(), "gh-aw", "sandbox", "agent", "logs", "copilot-session-state"), { recursive: true, force: true });
      } catch {}
    });

    it("loads guard.tool_denials_exceeded events from copilot session events.jsonl", () => {
      const sessionDir = path.join(os.tmpdir(), "gh-aw", "sandbox", "agent", "logs", "copilot-session-state", "session-1");
      fs.mkdirSync(sessionDir, { recursive: true });
      fs.writeFileSync(
        path.join(sessionDir, "events.jsonl"),
        [
          JSON.stringify({ type: "assistant.message", timestamp: "2026-06-06T00:00:00Z", data: { content: "" } }),
          JSON.stringify({
            type: "guard.tool_denials_exceeded",
            timestamp: "2026-06-06T00:00:01Z",
            data: { denialCount: 5, threshold: 5, reason: "permission denied: read" },
          }),
        ].join("\n") + "\n"
      );

      const events = loadToolDenialsExceededEvents();
      expect(events).toEqual([{ denialCount: 5, threshold: 5, reason: "permission denied: read" }]);
    });

    it("renders dedicated context for tool denial threshold events", () => {
      const promptsDir = path.join(tmpDir, "gh-aw", "prompts");
      fs.mkdirSync(promptsDir, { recursive: true });
      fs.copyFileSync(path.join(__dirname, "../md/tool_denials_exceeded_context.md"), path.join(promptsDir, "tool_denials_exceeded_context.md"));

      const result = buildToolDenialsExceededContext([{ denialCount: 5, threshold: 5, reason: "permission denied: read" }], "daily-spdd-spec-planner");
      expect(result).toContain("Excessive Tool Denials");
      expect(result).toContain("5/5");
      expect(result).toContain("guard.tool_denials_exceeded");
      expect(result).toContain("daily-spdd-spec-planner");
      expect(result).toContain("> [!WARNING]");
    });
  });

  // ──────────────────────────────────────────────────────
  // report-as-failure feature flags (GH_AW_MISSING_TOOL_REPORT_AS_FAILURE / GH_AW_MISSING_DATA_REPORT_AS_FAILURE)
  // ──────────────────────────────────────────────────────

  describe("missing_tool and missing_data report-as-failure flags", () => {
    const fs = require("fs");
    const path = require("path");
    const os = require("os");

    /** @type {string} */
    let tmpDir;

    beforeEach(() => {
      vi.resetModules();
      tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "aw-test-report-as-failure-"));
      process.env.RUNNER_TEMP = tmpDir;
      process.env.GH_AW_AGENT_OUTPUT = path.join(tmpDir, "agent_output.json");
      // Write agent output with both missing_tool and missing_data items
      fs.writeFileSync(
        path.join(tmpDir, "agent_output.json"),
        JSON.stringify({
          items: [
            { type: "missing_tool", tool: "bash", reason: "not available" },
            { type: "missing_data", data_type: "config", reason: "file not found" },
          ],
        })
      );
    });

    afterEach(() => {
      delete process.env.RUNNER_TEMP;
      delete process.env.GH_AW_AGENT_OUTPUT;
      delete process.env.GH_AW_MISSING_TOOL_REPORT_AS_FAILURE;
      delete process.env.GH_AW_MISSING_DATA_REPORT_AS_FAILURE;
      if (fs.existsSync(tmpDir)) {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("buildMissingToolContext returns context when GH_AW_MISSING_TOOL_REPORT_AS_FAILURE is not set (default true)", () => {
      const { buildMissingToolContext } = require("./handle_agent_failure.cjs");
      const result = buildMissingToolContext();
      expect(result).toContain("Missing Tools Reported");
    });

    it("buildMissingToolContext returns context when GH_AW_MISSING_TOOL_REPORT_AS_FAILURE is true", () => {
      process.env.GH_AW_MISSING_TOOL_REPORT_AS_FAILURE = "true";
      vi.resetModules();
      const { buildMissingToolContext } = require("./handle_agent_failure.cjs");
      const result = buildMissingToolContext();
      expect(result).toContain("Missing Tools Reported");
    });

    it("buildMissingToolContext still returns context when GH_AW_MISSING_TOOL_REPORT_AS_FAILURE is false (context building is independent of flag)", () => {
      // buildMissingToolContext always builds context; the flag controls hasMissingTool in main()
      process.env.GH_AW_MISSING_TOOL_REPORT_AS_FAILURE = "false";
      vi.resetModules();
      const { buildMissingToolContext } = require("./handle_agent_failure.cjs");
      const result = buildMissingToolContext();
      // buildMissingToolContext reads agent output directly, not the env flag
      expect(result).toContain("Missing Tools Reported");
    });
  });

  // ──────────────────────────────────────────────────────
  // hasAgentTerminalReasonCompleted
  // ──────────────────────────────────────────────────────

  describe("hasAgentTerminalReasonCompleted", () => {
    let hasAgentTerminalReasonCompleted;
    const fs = require("fs");
    const path = require("path");
    const os = require("os");

    /** @type {string} */
    let tmpDir;
    /** @type {string} */
    let stdioLogPath;

    beforeEach(() => {
      vi.resetModules();
      tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "aw-test-terminal-reason-"));
      stdioLogPath = path.join(tmpDir, "agent-stdio.log");
      process.env.GH_AW_AGENT_OUTPUT = path.join(tmpDir, "agent_output.json");
      ({ hasAgentTerminalReasonCompleted } = require("./handle_agent_failure.cjs"));
    });

    afterEach(() => {
      delete process.env.GH_AW_AGENT_OUTPUT;
      if (fs.existsSync(tmpDir)) {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("returns false when log file does not exist", () => {
      expect(hasAgentTerminalReasonCompleted()).toBe(false);
    });

    it("returns false when log file is empty", () => {
      fs.writeFileSync(stdioLogPath, "");
      expect(hasAgentTerminalReasonCompleted()).toBe(false);
    });

    it("returns true for plain JSON result line with terminal_reason: completed", () => {
      fs.writeFileSync(stdioLogPath, '{"type":"result","subtype":"success","terminal_reason":"completed","num_turns":53,"total_cost_usd":1.91}\n');
      expect(hasAgentTerminalReasonCompleted()).toBe(true);
    });

    it("returns true for timestamp-prefixed result line", () => {
      fs.writeFileSync(stdioLogPath, '2026-04-27T21:45:00.080Z  {"type":"result","subtype":"success","terminal_reason":"completed","num_turns":53}\n');
      expect(hasAgentTerminalReasonCompleted()).toBe(true);
    });

    it("returns false when terminal_reason is not completed", () => {
      fs.writeFileSync(stdioLogPath, '{"type":"result","subtype":"error_max_turns","terminal_reason":"max_turns","num_turns":50}\n');
      expect(hasAgentTerminalReasonCompleted()).toBe(false);
    });

    it("returns false when log contains only non-JSON lines", () => {
      fs.writeFileSync(stdioLogPath, "Starting agent...\nRunning tool: list_files\nAgent interrupted\n");
      expect(hasAgentTerminalReasonCompleted()).toBe(false);
    });

    it("returns true when terminal_reason: completed appears among other log lines", () => {
      const lines = ["Starting agent...", '{"type":"system","subtype":"init","model":"claude-sonnet-4"}', '{"type":"result","subtype":"success","terminal_reason":"completed","num_turns":10}', "Process exiting with code: 0"];
      fs.writeFileSync(stdioLogPath, lines.join("\n") + "\n");
      expect(hasAgentTerminalReasonCompleted()).toBe(true);
    });

    it("returns true when JSON is truncated but substring is present in content", () => {
      // A truncated line that can't be fully parsed as JSON but contains the literal substring
      fs.writeFileSync(stdioLogPath, '"terminal_reason":"completed","num_turns":53 (truncated\n');
      expect(hasAgentTerminalReasonCompleted()).toBe(true);
    });

    it("returns true with no spaces around colon (compact JSON)", () => {
      fs.writeFileSync(stdioLogPath, '{"terminal_reason":"completed"}\n');
      expect(hasAgentTerminalReasonCompleted()).toBe(true);
    });

    it("returns true with one space on each side of colon", () => {
      fs.writeFileSync(stdioLogPath, '{"terminal_reason" : "completed"}\n');
      expect(hasAgentTerminalReasonCompleted()).toBe(true);
    });
  });

  // ──────────────────────────────────────────────────────
  // buildEngineFailureContext — terminal_reason guard
  // ──────────────────────────────────────────────────────

  describe("buildEngineFailureContext with terminal_reason guard", () => {
    let buildEngineFailureContext;
    const fs = require("fs");
    const path = require("path");
    const os = require("os");

    /** @type {string} */
    let tmpDir;
    /** @type {string} */
    let stdioLogPath;

    beforeEach(() => {
      vi.resetModules();
      tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "aw-test-engine-fail-guard-"));
      stdioLogPath = path.join(tmpDir, "agent-stdio.log");
      process.env.GH_AW_AGENT_OUTPUT = path.join(tmpDir, "agent_output.json");
      process.env.RUNNER_TEMP = tmpDir;
      ({ buildEngineFailureContext } = require("./handle_agent_failure.cjs"));
    });

    afterEach(() => {
      delete process.env.GH_AW_AGENT_OUTPUT;
      delete process.env.GH_AW_ENGINE_ID;
      delete process.env.RUNNER_TEMP;
      if (fs.existsSync(tmpDir)) {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("returns empty string when log contains terminal_reason: completed (plain JSON)", () => {
      fs.writeFileSync(stdioLogPath, '{"type":"result","subtype":"success","terminal_reason":"completed","num_turns":53,"total_cost_usd":1.91}\n');
      expect(buildEngineFailureContext()).toBe("");
    });

    it("returns empty string when log contains terminal_reason: completed with timestamp prefix", () => {
      const lines = [
        "Starting claude workflow...",
        "2026-04-27T21:44:49.870Z  safeoutputs.create_discussion: completed successfully in 76ms",
        '2026-04-27T21:45:00.080Z  {"type":"result","subtype":"success","terminal_reason":"completed","num_turns":53,"total_cost_usd":1.91}',
        "[WARN] Command completed with exit code: 1",
        "Process exiting with code: 1",
      ];
      fs.writeFileSync(stdioLogPath, lines.join("\n") + "\n");
      expect(buildEngineFailureContext()).toBe("");
    });

    it("still surfaces errors when terminal_reason is not completed", () => {
      fs.writeFileSync(stdioLogPath, 'ERROR: quota exceeded\n{"type":"result","terminal_reason":"max_turns"}\n');
      const result = buildEngineFailureContext();
      expect(result).toContain("Engine Failure");
      expect(result).toContain("quota exceeded");
    });

    it("still uses fallback tail when no terminal_reason and no error patterns", () => {
      fs.writeFileSync(stdioLogPath, "Starting agent...\nAgent interrupted unexpectedly\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("Engine Failure");
      expect(result).toContain("Agent interrupted unexpectedly");
    });
  });

  // ──────────────────────────────────────────────────────
  // main() — hasCompletedDespiteJobFailure early-return
  // ──────────────────────────────────────────────────────

  describe("main() hasCompletedDespiteJobFailure early-return", () => {
    const fs = require("fs");
    const path = require("path");
    const os = require("os");

    /** @type {string} */
    let tmpDir;
    /** @type {string} */
    let promptsDir;

    beforeEach(() => {
      vi.resetModules();
      tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "aw-test-completed-despite-failure-"));
      promptsDir = path.join(tmpDir, "gh-aw", "prompts");
      fs.mkdirSync(promptsDir, { recursive: true });

      // Minimal templates required by main()
      fs.writeFileSync(path.join(promptsDir, "agent_failure_comment.md"), "COMMENT TEMPLATE");
      fs.writeFileSync(path.join(promptsDir, "agent_failure_issue.md"), "ISSUE TEMPLATE");

      process.env.RUNNER_TEMP = tmpDir;
      process.env.GH_AW_WORKFLOW_NAME = "Test Workflow";
      process.env.GH_AW_WORKFLOW_ID = "test-workflow";
      process.env.GH_AW_RUN_URL = "https://github.com/owner/repo/actions/runs/123456";
      process.env.GH_AW_AGENT_CONCLUSION = "failure";
      process.env.GH_AW_AGENT_OUTPUT = path.join(tmpDir, "agent_output.json");
    });

    afterEach(() => {
      delete process.env.RUNNER_TEMP;
      delete process.env.GH_AW_WORKFLOW_NAME;
      delete process.env.GH_AW_WORKFLOW_ID;
      delete process.env.GH_AW_RUN_URL;
      delete process.env.GH_AW_AGENT_CONCLUSION;
      delete process.env.GH_AW_AGENT_OUTPUT;
      if (fs.existsSync(tmpDir)) {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("skips failure issue creation when terminal_reason: completed and non-noop safe outputs present", async () => {
      // Agent produced a valid non-noop safe output
      fs.writeFileSync(path.join(tmpDir, "agent_output.json"), JSON.stringify({ items: [{ type: "create_discussion", title: "Done", body: "All set." }] }));
      // stdio log contains terminal_reason: completed
      fs.writeFileSync(path.join(tmpDir, "agent-stdio.log"), '{"type":"result","subtype":"success","terminal_reason":"completed","num_turns":10}\n');

      const createIssueMock = vi.fn();
      const createCommentMock = vi.fn();

      global.github = {
        rest: {
          search: {
            issuesAndPullRequests: vi.fn(async () => ({ data: { total_count: 0, items: [] } })),
          },
          issues: {
            create: createIssueMock,
            createComment: createCommentMock,
          },
          pulls: { get: vi.fn() },
        },
        graphql: vi.fn(),
      };

      vi.resetModules();
      const { main: mainFn } = require("./handle_agent_failure.cjs");
      await mainFn();

      expect(createIssueMock).not.toHaveBeenCalled();
      expect(createCommentMock).not.toHaveBeenCalled();
    });

    it("still creates failure issue when terminal_reason: completed but report_incomplete is also present", async () => {
      // Agent produced both a non-noop item and a report_incomplete signal
      fs.writeFileSync(
        path.join(tmpDir, "agent_output.json"),
        JSON.stringify({
          items: [
            { type: "create_discussion", title: "Done", body: "All set." },
            { type: "report_incomplete", reason: "mcp_crash" },
          ],
        })
      );
      fs.writeFileSync(path.join(tmpDir, "agent-stdio.log"), '{"type":"result","subtype":"success","terminal_reason":"completed","num_turns":10}\n');

      const createIssueMock = vi.fn(async () => ({ data: { number: 101, html_url: "https://github.com/owner/repo/issues/101", node_id: "I_123" } }));
      const createCommentMock = vi.fn(async () => ({ data: { id: 1001 } }));

      global.github = {
        rest: {
          search: {
            issuesAndPullRequests: vi.fn(async ({ q }) => {
              if (q.includes("is:pr")) return { data: { total_count: 0, items: [] } };
              return { data: { total_count: 0, items: [] } };
            }),
          },
          issues: {
            create: createIssueMock,
            createComment: createCommentMock,
          },
          pulls: { get: vi.fn() },
        },
        graphql: vi.fn(),
      };

      vi.resetModules();
      const { main: mainFn } = require("./handle_agent_failure.cjs");
      await mainFn();

      // report_incomplete overrides the hasCompletedDespiteJobFailure exemption
      expect(createIssueMock).toHaveBeenCalled();
    });
  });

  describe("parseFirewallAuthErrors", () => {
    const fs = require("fs");
    const os = require("os");
    const path = require("path");

    let tmpDir;
    let parseFirewallAuthErrors;

    beforeEach(() => {
      global.core = { info: vi.fn(), warning: vi.fn(), error: vi.fn(), debug: vi.fn(), setOutput: vi.fn(), setFailed: vi.fn() };
      global.github = {};
      global.context = { repo: { owner: "owner", repo: "repo" } };
      vi.resetModules();
      ({ parseFirewallAuthErrors } = require("./handle_agent_failure.cjs"));
      tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "aw-firewall-auth-"));
    });

    afterEach(() => {
      delete global.core;
      delete global.github;
      delete global.context;
      delete process.env.GH_AW_ENGINE_API_HOSTS;
      delete process.env.GH_AW_ENGINE_ID;
      if (tmpDir && fs.existsSync(tmpDir)) {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("returns empty array when file does not exist", () => {
      const result = parseFirewallAuthErrors(path.join(tmpDir, "nonexistent.jsonl"));
      expect(result).toEqual([]);
    });

    it("returns empty array when file is empty", () => {
      const jsonlPath = path.join(tmpDir, "audit.jsonl");
      fs.writeFileSync(jsonlPath, "");
      expect(parseFirewallAuthErrors(jsonlPath)).toEqual([]);
    });

    it("returns empty array when no 401/403 entries", () => {
      const jsonlPath = path.join(tmpDir, "audit.jsonl");
      fs.writeFileSync(jsonlPath, [JSON.stringify({ ts: 1000, host: "api.enterprise.githubcopilot.com:443", status: 200 }), JSON.stringify({ ts: 1001, host: "api.openai.com:443", status: 200 })].join("\n"));
      expect(parseFirewallAuthErrors(jsonlPath)).toEqual([]);
    });

    it("detects Copilot 401 auth rejection via hardcoded fallback", () => {
      const jsonlPath = path.join(tmpDir, "audit.jsonl");
      fs.writeFileSync(jsonlPath, JSON.stringify({ ts: 1000, host: "api.enterprise.githubcopilot.com:443", status: 401 }));
      const result = parseFirewallAuthErrors(jsonlPath);
      expect(result).toHaveLength(1);
      expect(result[0].provider).toBe("GitHub Copilot");
      expect(result[0].credential).toContain("COPILOT_GITHUB_TOKEN");
    });

    it("detects OpenAI 401 auth rejection via hardcoded fallback", () => {
      const jsonlPath = path.join(tmpDir, "audit.jsonl");
      fs.writeFileSync(jsonlPath, JSON.stringify({ ts: 1000, host: "api.openai.com:443", status: 401 }));
      const result = parseFirewallAuthErrors(jsonlPath);
      expect(result).toHaveLength(1);
      expect(result[0].provider).toBe("OpenAI Codex");
      expect(result[0].credential).toContain("OPENAI_API_KEY");
    });

    it("detects Anthropic 403 auth rejection via hardcoded fallback", () => {
      const jsonlPath = path.join(tmpDir, "audit.jsonl");
      fs.writeFileSync(jsonlPath, JSON.stringify({ ts: 1000, host: "api.anthropic.com:443", status: 403 }));
      const result = parseFirewallAuthErrors(jsonlPath);
      expect(result).toHaveLength(1);
      expect(result[0].provider).toBe("Anthropic Claude");
      expect(result[0].credential).toContain("ANTHROPIC_API_KEY");
    });

    it("detects Gemini 403 auth rejection via hardcoded fallback", () => {
      const jsonlPath = path.join(tmpDir, "audit.jsonl");
      fs.writeFileSync(jsonlPath, JSON.stringify({ ts: 1000, host: "generativelanguage.googleapis.com:443", status: 403 }));
      const result = parseFirewallAuthErrors(jsonlPath);
      expect(result).toHaveLength(1);
      expect(result[0].provider).toBe("Google Gemini");
      expect(result[0].credential).toContain("GEMINI_API_KEY");
    });

    it("deduplicates multiple auth errors for the same provider", () => {
      const jsonlPath = path.join(tmpDir, "audit.jsonl");
      fs.writeFileSync(jsonlPath, [JSON.stringify({ ts: 1000, host: "api.enterprise.githubcopilot.com:443", status: 401 }), JSON.stringify({ ts: 1001, host: "api.githubcopilot.com:443", status: 401 })].join("\n"));
      const result = parseFirewallAuthErrors(jsonlPath);
      expect(result).toHaveLength(1);
      expect(result[0].provider).toBe("GitHub Copilot");
    });

    it("reports multiple different providers", () => {
      const jsonlPath = path.join(tmpDir, "audit.jsonl");
      fs.writeFileSync(jsonlPath, [JSON.stringify({ ts: 1000, host: "api.openai.com:443", status: 401 }), JSON.stringify({ ts: 1001, host: "api.anthropic.com:443", status: 403 })].join("\n"));
      const result = parseFirewallAuthErrors(jsonlPath);
      expect(result).toHaveLength(2);
      const providers = result.map(r => r.provider);
      expect(providers).toContain("OpenAI Codex");
      expect(providers).toContain("Anthropic Claude");
    });

    it("skips non-JSON lines without throwing", () => {
      const jsonlPath = path.join(tmpDir, "audit.jsonl");
      fs.writeFileSync(jsonlPath, ["# comment line", "not json", JSON.stringify({ ts: 1000, host: "api.openai.com:443", status: 401 }), ""].join("\n"));
      const result = parseFirewallAuthErrors(jsonlPath);
      expect(result).toHaveLength(1);
      expect(result[0].provider).toBe("OpenAI Codex");
    });

    it("uses GH_AW_ENGINE_API_HOSTS env var when set", () => {
      process.env.GH_AW_ENGINE_API_HOSTS = "api.enterprise.githubcopilot.com,api.githubcopilot.com";
      process.env.GH_AW_ENGINE_ID = "copilot";
      vi.resetModules();
      ({ parseFirewallAuthErrors } = require("./handle_agent_failure.cjs"));

      const jsonlPath = path.join(tmpDir, "audit.jsonl");
      fs.writeFileSync(jsonlPath, JSON.stringify({ ts: 1000, host: "api.enterprise.githubcopilot.com:443", status: 401 }));
      const result = parseFirewallAuthErrors(jsonlPath);
      expect(result).toHaveLength(1);
      expect(result[0].provider).toBe("GitHub Copilot");
      expect(result[0].credential).toContain("COPILOT_GITHUB_TOKEN");
    });

    it("uses engine label from ENGINE_ID_TO_LABEL when env var host matches", () => {
      process.env.GH_AW_ENGINE_API_HOSTS = "api.anthropic.com";
      process.env.GH_AW_ENGINE_ID = "claude";
      vi.resetModules();
      ({ parseFirewallAuthErrors } = require("./handle_agent_failure.cjs"));

      const jsonlPath = path.join(tmpDir, "audit.jsonl");
      fs.writeFileSync(jsonlPath, JSON.stringify({ ts: 1000, host: "api.anthropic.com:443", status: 401 }));
      const result = parseFirewallAuthErrors(jsonlPath);
      expect(result).toHaveLength(1);
      expect(result[0].provider).toBe("Anthropic Claude");
      expect(result[0].credential).toContain("ANTHROPIC_API_KEY");
    });

    it("uses engine ID as provider label when not in lookup table", () => {
      process.env.GH_AW_ENGINE_API_HOSTS = "custom-llm.internal.example.com";
      process.env.GH_AW_ENGINE_ID = "custom";
      vi.resetModules();
      ({ parseFirewallAuthErrors } = require("./handle_agent_failure.cjs"));

      const jsonlPath = path.join(tmpDir, "audit.jsonl");
      fs.writeFileSync(jsonlPath, JSON.stringify({ ts: 1000, host: "custom-llm.internal.example.com:443", status: 401 }));
      const result = parseFirewallAuthErrors(jsonlPath);
      expect(result).toHaveLength(1);
      expect(result[0].provider).toBe("custom");
    });

    it("selective pre-scan: skips full parse when no 4xx entries in large file", () => {
      const jsonlPath = path.join(tmpDir, "audit.jsonl");
      // Write many 200 entries — should bail on pre-scan without parsing each line
      const lines = [];
      for (let i = 0; i < 100; i++) {
        lines.push(JSON.stringify({ ts: 1000 + i, host: "api.github.com:443", status: 200 }));
      }
      fs.writeFileSync(jsonlPath, lines.join("\n"));
      const result = parseFirewallAuthErrors(jsonlPath);
      expect(result).toEqual([]);
    });
  });

  describe("parseMaxEffectiveTokensFromAuditLog", () => {
    const fs = require("fs");
    const os = require("os");
    const path = require("path");

    let tmpDir;
    let parseMaxEffectiveTokensFromAuditLog;
    let parseEffectiveTokensErrorInfoFromAuditLog;

    beforeEach(() => {
      global.core = { info: vi.fn(), warning: vi.fn(), error: vi.fn(), debug: vi.fn(), setOutput: vi.fn(), setFailed: vi.fn() };
      global.github = {};
      global.context = { repo: { owner: "owner", repo: "repo" } };
      vi.resetModules();
      ({ parseMaxEffectiveTokensFromAuditLog, parseEffectiveTokensErrorInfoFromAuditLog } = require("./handle_agent_failure.cjs"));
      tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "aw-max-et-"));
    });

    afterEach(() => {
      delete global.core;
      delete global.github;
      delete global.context;
      delete process.env.GH_AW_AGENT_OUTPUT;
      if (tmpDir && fs.existsSync(tmpDir)) {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("returns empty string when file does not exist", () => {
      const result = parseMaxEffectiveTokensFromAuditLog(path.join(tmpDir, "missing.jsonl"));
      expect(result).toBe("");
    });

    it("parses max_effective_tokens from audit log entries", () => {
      const jsonlPath = path.join(tmpDir, "log.jsonl");
      fs.writeFileSync(jsonlPath, [JSON.stringify({ _schema: "audit/v0.26.0", ts: 1, max_effective_tokens: 12345 }), JSON.stringify({ _schema: "audit/v0.26.0", ts: 2, max_effective_tokens: 23456 })].join("\n"));
      const result = parseMaxEffectiveTokensFromAuditLog(jsonlPath);
      expect(result).toBe("23456");
    });

    it("parses nested camelCase maxEffectiveTokens values", () => {
      const jsonlPath = path.join(tmpDir, "log.jsonl");
      fs.writeFileSync(jsonlPath, JSON.stringify({ _schema: "audit/v0.26.0", ts: 1, awf: { budget: { maxEffectiveTokens: "9999" } } }));
      const result = parseMaxEffectiveTokensFromAuditLog(jsonlPath);
      expect(result).toBe("9999");
    });

    it("normalizes suffix maxEffectiveTokens values", () => {
      const jsonlPath = path.join(tmpDir, "log.jsonl");
      fs.writeFileSync(jsonlPath, JSON.stringify({ _schema: "audit/v0.26.0", ts: 1, awf: { budget: { maxEffectiveTokens: "100M" } } }));
      const result = parseMaxEffectiveTokensFromAuditLog(jsonlPath);
      expect(result).toBe("100000000");
    });

    it("uses derived default path and prefers log.jsonl", () => {
      const auditDir = path.join(tmpDir, "sandbox", "firewall", "audit");
      fs.mkdirSync(auditDir, { recursive: true });
      fs.writeFileSync(path.join(auditDir, "audit.jsonl"), JSON.stringify({ _schema: "audit/v0.26.0", ts: 1, max_effective_tokens: 1111 }));
      fs.writeFileSync(path.join(auditDir, "log.jsonl"), JSON.stringify({ _schema: "audit/v0.26.0", ts: 1, max_effective_tokens: 2222 }));
      process.env.GH_AW_AGENT_OUTPUT = path.join(tmpDir, "agent_output.json");
      const result = parseMaxEffectiveTokensFromAuditLog();
      expect(result).toBe("2222");
    });

    it("falls back to sandbox/firewall/logs/audit.jsonl when the audit directory JSONL is absent", () => {
      const logsDir = path.join(tmpDir, "sandbox", "firewall", "logs");
      fs.mkdirSync(logsDir, { recursive: true });
      fs.writeFileSync(path.join(logsDir, "audit.jsonl"), JSON.stringify({ _schema: "audit/v0.26.0", ts: 1, max_effective_tokens: 3333 }));
      process.env.GH_AW_AGENT_OUTPUT = path.join(tmpDir, "agent_output.json");
      expect(parseMaxEffectiveTokensFromAuditLog()).toBe("3333");
    });

    it("parses effective token rate-limit metadata from audit log entries", () => {
      const jsonlPath = path.join(tmpDir, "log.jsonl");
      fs.writeFileSync(
        jsonlPath,
        [
          JSON.stringify({ _schema: "audit/v0.26.0", ts: 1, effective_tokens: 4321, effective_tokens_rate_limit_error: false }),
          JSON.stringify({ _schema: "audit/v0.26.0", ts: 2, effective_tokens: 5432, effective_tokens_rate_limit_error: true }),
        ].join("\n")
      );
      const result = parseEffectiveTokensErrorInfoFromAuditLog(jsonlPath);
      expect(result).toEqual({ effectiveTokens: "5432", rateLimitError: true });
    });

    it("parses nested camelCase effective token metadata from default log.jsonl", () => {
      const auditDir = path.join(tmpDir, "sandbox", "firewall", "audit");
      fs.mkdirSync(auditDir, { recursive: true });
      fs.writeFileSync(path.join(auditDir, "log.jsonl"), JSON.stringify({ _schema: "audit/v0.26.0", ts: 1, awf: { budget: { effectiveTokens: "7777" }, errors: { effectiveTokensRateLimitError: "true" } } }));
      process.env.GH_AW_AGENT_OUTPUT = path.join(tmpDir, "agent_output.json");
      const result = parseEffectiveTokensErrorInfoFromAuditLog();
      expect(result).toEqual({ effectiveTokens: "7777", rateLimitError: true });
    });

    it("detects effective token rate-limit errors from text fields", () => {
      const jsonlPath = path.join(tmpDir, "log.jsonl");
      fs.writeFileSync(jsonlPath, JSON.stringify({ _schema: "audit/v0.26.0", ts: 1, details: "429 too many requests: effective tokens budget exceeded" }));
      const result = parseEffectiveTokensErrorInfoFromAuditLog(jsonlPath);
      expect(result).toEqual({ effectiveTokens: "", rateLimitError: true });
    });
  });

  describe("resolveEffectiveTokensFailureState", () => {
    const fs = require("fs");
    const os = require("os");
    const path = require("path");

    let tmpDir;
    let resolveEffectiveTokensFailureState;

    beforeEach(() => {
      vi.resetModules();
      ({ resolveEffectiveTokensFailureState } = require("./effective_tokens_context.cjs"));
      tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "aw-resolve-et-"));
    });

    describe("resolveAICreditsFailureState", () => {
      const fs = require("fs");
      const os = require("os");
      const path = require("path");

      let tmpDir;
      let resolveAICreditsFailureState;

      beforeEach(() => {
        vi.resetModules();
        ({ resolveAICreditsFailureState } = require("./effective_tokens_context.cjs"));
        tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "aw-resolve-aic-"));
      });

      afterEach(() => {
        delete process.env.GH_AW_AGENT_OUTPUT;
        delete process.env.GH_AW_AIC;
        delete process.env.GH_AW_MAX_AI_CREDITS;
        delete process.env.GH_AW_AI_CREDITS_RATE_LIMIT_ERROR;
        if (tmpDir && fs.existsSync(tmpDir)) {
          fs.rmSync(tmpDir, { recursive: true, force: true });
        }
      });

      it("suppresses AI credits budget exhaustion when usage is below the configured maximum", () => {
        const auditDir = path.join(tmpDir, "sandbox", "firewall", "audit");
        fs.mkdirSync(auditDir, { recursive: true });
        fs.writeFileSync(
          path.join(auditDir, "log.jsonl"),
          JSON.stringify({
            _schema: "audit/v0.26.0",
            ts: 1,
            ai_credits: 900,
            max_ai_credits: 1000,
            ai_credits_rate_limit_error: true,
          })
        );
        process.env.GH_AW_AGENT_OUTPUT = path.join(tmpDir, "agent_output.json");

        expect(resolveAICreditsFailureState()).toEqual({
          aiCredits: "900",
          maxAICredits: "1000",
          aiCreditsRateLimitError: false,
        });
      });

      it("keeps AI credits budget exhaustion when usage meets the configured maximum", () => {
        const auditDir = path.join(tmpDir, "sandbox", "firewall", "audit");
        fs.mkdirSync(auditDir, { recursive: true });
        fs.writeFileSync(
          path.join(auditDir, "log.jsonl"),
          JSON.stringify({
            _schema: "audit/v0.26.0",
            ts: 1,
            ai_credits: 1000,
            max_ai_credits: 1000,
            ai_credits_rate_limit_error: true,
          })
        );
        process.env.GH_AW_AGENT_OUTPUT = path.join(tmpDir, "agent_output.json");

        expect(resolveAICreditsFailureState()).toEqual({
          aiCredits: "1000",
          maxAICredits: "1000",
          aiCreditsRateLimitError: true,
        });
      });
    });

    afterEach(() => {
      delete process.env.GH_AW_AGENT_OUTPUT;
      delete process.env.GH_AW_EFFECTIVE_TOKENS;
      delete process.env.GH_AW_MAX_EFFECTIVE_TOKENS;
      delete process.env.GH_AW_MAX_AI_CREDITS;
      delete process.env.GH_AW_EFFECTIVE_TOKENS_RATE_LIMIT_ERROR;
      if (tmpDir && fs.existsSync(tmpDir)) {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("does not suppress ET budget exhaustion from max_effective_tokens when max_ai_credits is absent", () => {
      const auditDir = path.join(tmpDir, "sandbox", "firewall", "audit");
      fs.mkdirSync(auditDir, { recursive: true });
      fs.writeFileSync(
        path.join(auditDir, "log.jsonl"),
        JSON.stringify({
          _schema: "audit/v0.26.0",
          ts: 1,
          effective_tokens: 2097968,
          max_effective_tokens: 10000000,
          effective_tokens_rate_limit_error: true,
        })
      );
      process.env.GH_AW_AGENT_OUTPUT = path.join(tmpDir, "agent_output.json");

      expect(resolveEffectiveTokensFailureState()).toEqual({
        effectiveTokens: "2097968",
        maxEffectiveTokens: "",
        effectiveTokensRateLimitError: true,
      });
    });

    it("does not suppress ET budget exhaustion from reflect max_effective_tokens when max_ai_credits is absent", () => {
      const firewallDir = path.join(tmpDir, "sandbox", "firewall");
      fs.mkdirSync(firewallDir, { recursive: true });
      fs.writeFileSync(
        path.join(firewallDir, "awf-reflect.json"),
        JSON.stringify({
          effective_tokens: {
            total_effective_tokens: 1355879.9999999995,
            max_effective_tokens: 10000000,
          },
        })
      );
      process.env.GH_AW_AGENT_OUTPUT = path.join(tmpDir, "agent_output.json");
      process.env.GH_AW_EFFECTIVE_TOKENS = "12202920";
      process.env.GH_AW_EFFECTIVE_TOKENS_RATE_LIMIT_ERROR = "true";

      expect(resolveEffectiveTokensFailureState()).toEqual({
        effectiveTokens: "1355879",
        maxEffectiveTokens: "",
        effectiveTokensRateLimitError: true,
      });
    });

    it("keeps ET budget exhaustion when only max_effective_tokens is present", () => {
      const auditDir = path.join(tmpDir, "sandbox", "firewall", "audit");
      fs.mkdirSync(auditDir, { recursive: true });
      fs.writeFileSync(
        path.join(auditDir, "log.jsonl"),
        JSON.stringify({
          _schema: "audit/v0.26.0",
          ts: 1,
          effective_tokens: 10000000,
          max_effective_tokens: 10000000,
          effective_tokens_rate_limit_error: true,
        })
      );
      process.env.GH_AW_AGENT_OUTPUT = path.join(tmpDir, "agent_output.json");

      expect(resolveEffectiveTokensFailureState()).toEqual({
        effectiveTokens: "10000000",
        maxEffectiveTokens: "",
        effectiveTokensRateLimitError: true,
      });
    });

    it("uses AI credits budgets to suppress false ET budget exhaustion signals", () => {
      const auditDir = path.join(tmpDir, "sandbox", "firewall", "audit");
      fs.mkdirSync(auditDir, { recursive: true });
      fs.writeFileSync(
        path.join(auditDir, "log.jsonl"),
        JSON.stringify({
          _schema: "audit/v0.26.0",
          ts: 1,
          effective_tokens: 268104,
          max_ai_credits: 1500,
          effective_tokens_rate_limit_error: true,
        })
      );
      process.env.GH_AW_AGENT_OUTPUT = path.join(tmpDir, "agent_output.json");

      expect(resolveEffectiveTokensFailureState()).toEqual({
        effectiveTokens: "268104",
        maxEffectiveTokens: "15000000",
        effectiveTokensRateLimitError: false,
      });
    });

    it("supports fractional AI credits when deriving ET-equivalent max budgets", () => {
      const auditDir = path.join(tmpDir, "sandbox", "firewall", "audit");
      fs.mkdirSync(auditDir, { recursive: true });
      fs.writeFileSync(
        path.join(auditDir, "log.jsonl"),
        JSON.stringify({
          _schema: "audit/v0.26.0",
          ts: 1,
          effective_tokens: 500,
          max_ai_credits: "0.2509",
          effective_tokens_rate_limit_error: true,
        })
      );
      process.env.GH_AW_AGENT_OUTPUT = path.join(tmpDir, "agent_output.json");

      expect(resolveEffectiveTokensFailureState()).toEqual({
        effectiveTokens: "500",
        maxEffectiveTokens: "2509",
        effectiveTokensRateLimitError: false,
      });
    });

    it("derives ET-equivalent max budget from whole AI credits", () => {
      process.env.GH_AW_EFFECTIVE_TOKENS = "15000";
      process.env.GH_AW_MAX_AI_CREDITS = "2";
      process.env.GH_AW_EFFECTIVE_TOKENS_RATE_LIMIT_ERROR = "true";

      expect(resolveEffectiveTokensFailureState()).toEqual({
        effectiveTokens: "15000",
        maxEffectiveTokens: "20000",
        effectiveTokensRateLimitError: false,
      });
    });

    it("uses max_ai_credits instead of max_effective_tokens when both are present", () => {
      const auditDir = path.join(tmpDir, "sandbox", "firewall", "audit");
      fs.mkdirSync(auditDir, { recursive: true });
      fs.writeFileSync(
        path.join(auditDir, "log.jsonl"),
        JSON.stringify({
          _schema: "audit/v0.26.0",
          ts: 1,
          effective_tokens: 5000,
          max_effective_tokens: 5000,
          max_ai_credits: "1",
          effective_tokens_rate_limit_error: true,
        })
      );
      process.env.GH_AW_AGENT_OUTPUT = path.join(tmpDir, "agent_output.json");

      expect(resolveEffectiveTokensFailureState()).toEqual({
        effectiveTokens: "5000",
        maxEffectiveTokens: "10000",
        effectiveTokensRateLimitError: false,
      });
    });

    it("keeps ET budget exhaustion when the rate-limit signal is present but no max is available", () => {
      process.env.GH_AW_EFFECTIVE_TOKENS = "2097968";
      process.env.GH_AW_EFFECTIVE_TOKENS_RATE_LIMIT_ERROR = "true";

      expect(resolveEffectiveTokensFailureState()).toEqual({
        effectiveTokens: "2097968",
        maxEffectiveTokens: "",
        effectiveTokensRateLimitError: true,
      });
    });

    it("treats maximum effective tokens exceeded in agent logs as ET budget exhaustion when reflect shows the budget is spent", () => {
      const firewallDir = path.join(tmpDir, "sandbox", "firewall");
      fs.mkdirSync(firewallDir, { recursive: true });
      fs.writeFileSync(
        path.join(firewallDir, "awf-reflect.json"),
        JSON.stringify({
          effective_tokens: {
            total_effective_tokens: 25000000,
            max_effective_tokens: 25000000,
          },
        })
      );
      fs.writeFileSync(path.join(tmpDir, "agent-stdio.log"), "Failed to get response from the AI model; retried 5 times. Last error: CAPIError: 429 Maximum effective tokens exceeded (25296477.30 / 25000000).\n");
      process.env.GH_AW_AGENT_OUTPUT = path.join(tmpDir, "agent_output.json");

      expect(resolveEffectiveTokensFailureState()).toEqual({
        effectiveTokens: "25000000",
        maxEffectiveTokens: "",
        effectiveTokensRateLimitError: true,
      });
    });

    it("ignores invalid env token counts when reconciling ET budget exhaustion", () => {
      process.env.GH_AW_EFFECTIVE_TOKENS = "2097968";
      process.env.GH_AW_MAX_EFFECTIVE_TOKENS = "not-a-number";
      process.env.GH_AW_EFFECTIVE_TOKENS_RATE_LIMIT_ERROR = "true";

      expect(resolveEffectiveTokensFailureState()).toEqual({
        effectiveTokens: "2097968",
        maxEffectiveTokens: "",
        effectiveTokensRateLimitError: true,
      });
    });

    it("does not report ET budget exhaustion without a rate-limit signal", () => {
      process.env.GH_AW_EFFECTIVE_TOKENS = "10000000";
      process.env.GH_AW_MAX_EFFECTIVE_TOKENS = "10000000";

      expect(resolveEffectiveTokensFailureState()).toEqual({
        effectiveTokens: "10000000",
        maxEffectiveTokens: "",
        effectiveTokensRateLimitError: false,
      });
    });

    it("normalizes ET suffix env maximums before reconciliation", () => {
      process.env.GH_AW_EFFECTIVE_TOKENS = "10000000";
      process.env.GH_AW_MAX_EFFECTIVE_TOKENS = "100M";

      expect(resolveEffectiveTokensFailureState()).toEqual({
        effectiveTokens: "10000000",
        maxEffectiveTokens: "",
        effectiveTokensRateLimitError: false,
      });
    });
  });

  describe("buildEffectiveTokensRateLimitErrorContext", () => {
    let buildEffectiveTokensRateLimitErrorContext;
    const fs = require("fs");
    const os = require("os");
    const path = require("path");
    let tmpDir;

    beforeEach(() => {
      tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-test-et-error-context-"));
      const promptsDir = path.join(tmpDir, "gh-aw", "prompts");
      fs.mkdirSync(promptsDir, { recursive: true });
      fs.writeFileSync(
        path.join(promptsDir, "effective_tokens_rate_limit_error.md"),
        "> [!WARNING]\n" +
          "> **AI Credits Budget Guidance**: The run hit a legacy effective-token rate-limit signal from the API proxy. gh-aw now uses AI Credits (AIC) as the primary cost metric, so migrate per-run budgeting to `max-ai-credits`.\n\n" +
          "<details>\n" +
          "<summary>Why this happened and how to optimize</summary>\n\n" +
          "- Learn about [AI Credits]({ai_credits_spec_link}).\n" +
          "{budget_line}\n" +
          "- `max-effective-tokens` is deprecated; migrate to `max-ai-credits` by running `gh aw fix --write`, or update manually (1 AIC = 10,000 ET):\n" +
          "  ```yaml\n" +
          "  # before\n" +
          "  max-effective-tokens: 5000000\n" +
          "  # after\n" +
          "  max-ai-credits: 500\n" +
          "  ```\n\n" +
          "- To budget and optimize this workflow, follow the [cost management guidance]({cost_management_link}).\n" +
          "</details>\n"
      );
      process.env.RUNNER_TEMP = tmpDir;

      global.core = { info: vi.fn(), warning: vi.fn(), error: vi.fn(), debug: vi.fn(), setOutput: vi.fn(), setFailed: vi.fn() };
      global.github = {};
      global.context = { repo: { owner: "owner", repo: "repo" } };
      vi.resetModules();
      ({ buildEffectiveTokensRateLimitErrorContext } = require("./handle_agent_failure.cjs"));
    });

    afterEach(() => {
      delete global.core;
      delete global.github;
      delete global.context;
      delete process.env.RUNNER_TEMP;
      if (tmpDir && fs.existsSync(tmpDir)) {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("converts legacy ET budget to max-ai-credits guidance", () => {
      const result = buildEffectiveTokensRateLimitErrorContext(true, "2500000");
      expect(result).toContain("Suggested `max-ai-credits`: `250`");
    });

    it("returns empty string when rate-limit error is not set", () => {
      const result = buildEffectiveTokensRateLimitErrorContext(false, "2500000");
      expect(result).toBe("");
    });

    it("omits suggested AI credits when legacy ET budget is not numeric", () => {
      const result = buildEffectiveTokensRateLimitErrorContext(true, "n/a");
      expect(result).not.toContain("Suggested `max-ai-credits`:");
    });

    it("omits optional lines when values are missing", () => {
      const result = buildEffectiveTokensRateLimitErrorContext(true, "");
      expect(result).not.toContain("Suggested `max-ai-credits`:");
      expect(result).not.toContain("- Run:");
    });

    it("includes a link to the AI Credits specification docs", () => {
      const result = buildEffectiveTokensRateLimitErrorContext(true, "25000000");
      expect(result).toContain("https://github.github.com/gh-aw/specs/ai-credits-specification/");
    });

    it("includes a link to the cost management guide", () => {
      const result = buildEffectiveTokensRateLimitErrorContext(true, "25000000");
      expect(result).toContain("https://github.github.com/gh-aw/reference/cost-management/");
    });

    it("uses GitHub alert syntax for AI credits budget guidance", () => {
      const result = buildEffectiveTokensRateLimitErrorContext(true, "25000000");
      expect(result).toContain("> [!WARNING]");
      expect(result).toContain("> **AI Credits Budget Guidance**");
    });

    it("wraps ET guidance in a collapsible details section", () => {
      const result = buildEffectiveTokensRateLimitErrorContext(true, "25000000");
      expect(result).toContain("<summary>Why this happened and how to optimize</summary>");
      expect(result).toContain("cost management guidance");
      expect(result).toContain("max-ai-credits");
    });

    it("does not include ET computation details", () => {
      const result = buildEffectiveTokensRateLimitErrorContext(true, "25000000");
      expect(result).not.toContain("Effective tokens used:");
      expect(result).not.toContain("- Run:");
      expect(result).not.toContain("<sub>ET formula:");
    });
  });

  describe("readTokenUsageMarkdown", () => {
    let readTokenUsageMarkdown;
    const fs = require("fs");
    const os = require("os");
    const path = require("path");
    let tmpDir;

    beforeEach(() => {
      vi.resetModules();
      tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "rt-usage-test-"));
      ({ readTokenUsageMarkdown } = require("./handle_agent_failure.cjs"));
    });

    afterEach(() => {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    });

    it("returns null when no token-usage.jsonl files exist", () => {
      const result = readTokenUsageMarkdown();
      expect(result).toBeNull();
    });

    it("returns markdown and model aliases when a valid token-usage.jsonl file is present", () => {
      const { TOKEN_USAGE_PATH } = require("./parse_token_usage.cjs");
      fs.mkdirSync(path.dirname(TOKEN_USAGE_PATH), { recursive: true });
      const entry = JSON.stringify({ model: "claude-sonnet-4.5", input_tokens: 1000, output_tokens: 500, cache_read_tokens: 0, cache_write_tokens: 0, duration_ms: 1000 });
      const origContent = fs.existsSync(TOKEN_USAGE_PATH) ? fs.readFileSync(TOKEN_USAGE_PATH, "utf8") : null;
      fs.writeFileSync(TOKEN_USAGE_PATH, entry + "\n");
      try {
        vi.resetModules();
        ({ readTokenUsageMarkdown } = require("./handle_agent_failure.cjs"));
        const result = readTokenUsageMarkdown();
        expect(result).not.toBeNull();
        expect(result.markdown).toContain("sonnet45");
        expect(result.markdown).toContain("1,000");
        expect(result.markdown).toContain("Alias");
        expect(result.modelNames).toEqual(["claude-sonnet-4.5"]);
      } finally {
        if (origContent !== null) {
          fs.writeFileSync(TOKEN_USAGE_PATH, origContent);
        } else if (fs.existsSync(TOKEN_USAGE_PATH)) {
          fs.unlinkSync(TOKEN_USAGE_PATH);
        }
      }
    });
  });

  describe("buildCredentialAuthErrorContext", () => {
    const fs = require("fs");
    const os = require("os");
    const path = require("path");

    let tmpDir;
    let buildCredentialAuthErrorContext;

    beforeEach(() => {
      global.core = { info: vi.fn(), warning: vi.fn(), error: vi.fn(), debug: vi.fn(), setOutput: vi.fn(), setFailed: vi.fn() };
      global.github = {};
      global.context = { repo: { owner: "owner", repo: "repo" } };
      vi.resetModules();
      ({ buildCredentialAuthErrorContext } = require("./handle_agent_failure.cjs"));
      tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "aw-cred-auth-"));
      // Create prompt template so getPromptPath resolves
      const promptsDir = path.join(tmpDir, "gh-aw", "prompts");
      fs.mkdirSync(promptsDir, { recursive: true });
      fs.writeFileSync(path.join(promptsDir, "credential_auth_error.md"), "**🔑 Credential Authentication Failed**: Missing/invalid credentials for:\n{providers}\n");
      process.env.RUNNER_TEMP = tmpDir;
    });

    afterEach(() => {
      delete global.core;
      delete global.github;
      delete global.context;
      delete process.env.GH_AW_AGENT_OUTPUT;
      delete process.env.RUNNER_TEMP;
      if (tmpDir && fs.existsSync(tmpDir)) {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("returns empty string when audit.jsonl does not exist", () => {
      const result = buildCredentialAuthErrorContext(path.join(tmpDir, "nonexistent.jsonl"));
      expect(result).toBe("");
    });

    it("returns empty string when no auth errors in audit.jsonl", () => {
      const jsonlPath = path.join(tmpDir, "audit.jsonl");
      fs.writeFileSync(jsonlPath, JSON.stringify({ ts: 1000, host: "api.github.com:443", status: 200 }));
      const result = buildCredentialAuthErrorContext(jsonlPath);
      expect(result).toBe("");
    });

    it("returns credential alert when auth rejection found", () => {
      const jsonlPath = path.join(tmpDir, "audit.jsonl");
      fs.writeFileSync(jsonlPath, JSON.stringify({ ts: 1000, host: "api.openai.com:443", status: 401 }));
      const result = buildCredentialAuthErrorContext(jsonlPath);
      expect(result).toBeTruthy();
      expect(result).toContain("OpenAI");
      expect(result).toContain("OPENAI_API_KEY");
    });

    it("includes all affected providers in the output", () => {
      const jsonlPath = path.join(tmpDir, "audit.jsonl");
      fs.writeFileSync(jsonlPath, [JSON.stringify({ ts: 1000, host: "api.enterprise.githubcopilot.com:443", status: 401 }), JSON.stringify({ ts: 1001, host: "api.anthropic.com:443", status: 403 })].join("\n"));
      const result = buildCredentialAuthErrorContext(jsonlPath);
      expect(result).toContain("GitHub Copilot");
      expect(result).toContain("Anthropic Claude");
    });

    it("derives audit.jsonl path from GH_AW_AGENT_OUTPUT when no override provided", () => {
      const auditDir = path.join(tmpDir, "sandbox", "firewall", "audit");
      fs.mkdirSync(auditDir, { recursive: true });
      const jsonlPath = path.join(auditDir, "audit.jsonl");
      fs.writeFileSync(jsonlPath, JSON.stringify({ ts: 1000, host: "api.openai.com:443", status: 401 }));

      process.env.GH_AW_AGENT_OUTPUT = path.join(tmpDir, "agent_output.json");
      vi.resetModules();
      ({ buildCredentialAuthErrorContext } = require("./handle_agent_failure.cjs"));

      const result = buildCredentialAuthErrorContext();
      expect(result).toContain("OpenAI");
    });
  });

  describe("detectAndHandleFailureCascade", () => {
    let detectAndHandleFailureCascade;
    let findRecentFailureIssues;
    let CASCADE_THRESHOLD;
    let CASCADE_LABEL;
    let CASCADE_ROLLUP_LABEL;
    let CASCADE_ROLLUP_TITLE;
    let CASCADE_WINDOW_MINUTES;

    /**
     * Build a list of N fake `[aw] * failed` items starting at issue number `startNum`.
     * @param {number} count
     * @param {number} [startNum=100]
     */
    function makeFailureItems(count, startNum = 100) {
      return Array.from({ length: count }, (_, i) => ({
        number: startNum + i,
        title: `[aw] Workflow ${i} failed`,
        html_url: `https://github.com/owner/repo/issues/${startNum + i}`,
        created_at: new Date().toISOString(),
      }));
    }

    beforeEach(() => {
      vi.resetModules();
      ({ detectAndHandleFailureCascade, findRecentFailureIssues, CASCADE_THRESHOLD, CASCADE_LABEL, CASCADE_ROLLUP_LABEL, CASCADE_ROLLUP_TITLE, CASCADE_WINDOW_MINUTES } = require("./handle_agent_failure.cjs"));
    });

    it("exports cascade constants with expected values", () => {
      expect(CASCADE_THRESHOLD).toBe(10);
      expect(CASCADE_WINDOW_MINUTES).toBe(60);
      expect(CASCADE_LABEL).toBe("cascade-suspected");
      expect(CASCADE_ROLLUP_LABEL).toBe("cascade-rollup");
      expect(CASCADE_ROLLUP_TITLE).toBe("[aw] Failure cascade detected");
    });

    it("does nothing when fewer than CASCADE_THRESHOLD issues are in the window", async () => {
      const createIssueMock = vi.fn();
      const addLabelsMock = vi.fn();
      const searchMock = vi.fn(async ({ q }) => {
        if (q.includes("cascade-rollup")) return { data: { total_count: 0, items: [] } };
        // Return 5 issues — below threshold
        const items = makeFailureItems(5);
        return { data: { total_count: items.length, items } };
      });

      global.github = {
        rest: {
          search: { issuesAndPullRequests: searchMock },
          issues: {
            getLabel: vi.fn().mockResolvedValue({ data: {} }),
            create: createIssueMock,
            update: vi.fn(),
            addLabels: addLabelsMock,
          },
        },
        graphql: vi.fn(),
      };

      await detectAndHandleFailureCascade("owner", "repo", 999);

      expect(createIssueMock).not.toHaveBeenCalled();
      expect(addLabelsMock).not.toHaveBeenCalled();
    });

    it("creates a rollup issue and labels individual issues when CASCADE_THRESHOLD is met", async () => {
      const createIssueMock = vi.fn().mockResolvedValue({
        data: { number: 200, html_url: "https://github.com/owner/repo/issues/200", node_id: "I_200" },
      });
      const addLabelsMock = vi.fn().mockResolvedValue({});
      const getLabelMock = vi.fn().mockResolvedValue({ data: {} });

      const recentItems = makeFailureItems(10);

      const searchMock = vi.fn(async ({ q }) => {
        if (q.includes("cascade-rollup")) return { data: { total_count: 0, items: [] } };
        return { data: { total_count: recentItems.length, items: recentItems } };
      });

      global.github = {
        rest: {
          search: { issuesAndPullRequests: searchMock },
          issues: {
            getLabel: getLabelMock,
            create: createIssueMock,
            update: vi.fn(),
            addLabels: addLabelsMock,
          },
        },
        graphql: vi.fn(),
      };

      // triggeringIssueNumber is already in recentItems (100)
      await detectAndHandleFailureCascade("owner", "repo", 100);

      // Rollup issue created
      expect(createIssueMock).toHaveBeenCalledOnce();
      const createCall = createIssueMock.mock.calls[0][0];
      expect(createCall.title).toBe(CASCADE_ROLLUP_TITLE);
      expect(createCall.labels).toContain(CASCADE_ROLLUP_LABEL);
      expect(createCall.labels).toContain("agentic-workflows");

      // All 10 issues labeled
      expect(addLabelsMock).toHaveBeenCalledTimes(10);
      for (const call of addLabelsMock.mock.calls) {
        expect(call[0].labels).toContain(CASCADE_LABEL);
      }
    });

    it("triggers cascade when triggering issue is not in search results (search indexing lag)", async () => {
      const createIssueMock = vi.fn().mockResolvedValue({
        data: { number: 201, html_url: "https://github.com/owner/repo/issues/201", node_id: "I_201" },
      });
      const addLabelsMock = vi.fn().mockResolvedValue({});

      // Search returns only 9 issues (not the triggering one — simulating indexing lag)
      const recentItems = makeFailureItems(9);

      const searchMock = vi.fn(async ({ q }) => {
        if (q.includes("cascade-rollup")) return { data: { total_count: 0, items: [] } };
        return { data: { total_count: recentItems.length, items: recentItems } };
      });

      global.github = {
        rest: {
          search: { issuesAndPullRequests: searchMock },
          issues: {
            getLabel: vi.fn().mockResolvedValue({ data: {} }),
            create: createIssueMock,
            update: vi.fn(),
            addLabels: addLabelsMock,
          },
        },
        graphql: vi.fn(),
      };

      // Triggering issue (109) is NOT in recentItems — total becomes 10 once merged
      await detectAndHandleFailureCascade("owner", "repo", 109);

      expect(createIssueMock).toHaveBeenCalledOnce();
      // 9 from search + 1 triggering issue = 10 addLabels calls
      expect(addLabelsMock).toHaveBeenCalledTimes(10);
    });

    it("updates existing rollup issue instead of creating a new one", async () => {
      const createIssueMock = vi.fn();
      const updateIssueMock = vi.fn().mockResolvedValue({});
      const addLabelsMock = vi.fn().mockResolvedValue({});

      const recentItems = makeFailureItems(10);

      const searchMock = vi.fn(async ({ q }) => {
        if (q.includes("cascade-rollup")) {
          return {
            data: {
              total_count: 1,
              items: [{ number: 50, html_url: "https://github.com/owner/repo/issues/50" }],
            },
          };
        }
        return { data: { total_count: recentItems.length, items: recentItems } };
      });

      global.github = {
        rest: {
          search: { issuesAndPullRequests: searchMock },
          issues: {
            getLabel: vi.fn().mockResolvedValue({ data: {} }),
            create: createIssueMock,
            update: updateIssueMock,
            addLabels: addLabelsMock,
          },
        },
        graphql: vi.fn(),
      };

      await detectAndHandleFailureCascade("owner", "repo", 100);

      // Should update, not create
      expect(createIssueMock).not.toHaveBeenCalled();
      expect(updateIssueMock).toHaveBeenCalledOnce();
      expect(updateIssueMock.mock.calls[0][0].issue_number).toBe(50);
    });

    it("creates cascade-suspected label when it does not exist (404)", async () => {
      const createLabelMock = vi.fn().mockResolvedValue({});
      const createIssueMock = vi.fn().mockResolvedValue({
        data: { number: 202, html_url: "https://github.com/owner/repo/issues/202", node_id: "I_202" },
      });
      const addLabelsMock = vi.fn().mockResolvedValue({});

      const recentItems = makeFailureItems(10);

      const searchMock = vi.fn(async ({ q }) => {
        if (q.includes("cascade-rollup")) return { data: { total_count: 0, items: [] } };
        return { data: { total_count: recentItems.length, items: recentItems } };
      });

      const getLabelMock = vi.fn().mockRejectedValue(Object.assign(new Error("Not Found"), { status: 404 }));

      global.github = {
        rest: {
          search: { issuesAndPullRequests: searchMock },
          issues: {
            getLabel: getLabelMock,
            createLabel: createLabelMock,
            create: createIssueMock,
            update: vi.fn(),
            addLabels: addLabelsMock,
          },
        },
        graphql: vi.fn(),
      };

      await detectAndHandleFailureCascade("owner", "repo", 100);

      // Both cascade-suspected and cascade-rollup labels should be created
      expect(createLabelMock).toHaveBeenCalledTimes(2);
      const createdNames = createLabelMock.mock.calls.map(c => c[0].name);
      expect(createdNames).toContain(CASCADE_LABEL);
      expect(createdNames).toContain(CASCADE_ROLLUP_LABEL);
    });

    it("rollup body includes affected workflow list", async () => {
      let capturedBody = "";
      const createIssueMock = vi.fn(async ({ body }) => {
        capturedBody = body;
        return { data: { number: 203, html_url: "https://github.com/owner/repo/issues/203", node_id: "I_203" } };
      });

      const recentItems = makeFailureItems(10);

      const searchMock = vi.fn(async ({ q }) => {
        if (q.includes("cascade-rollup")) return { data: { total_count: 0, items: [] } };
        return { data: { total_count: recentItems.length, items: recentItems } };
      });

      global.github = {
        rest: {
          search: { issuesAndPullRequests: searchMock },
          issues: {
            getLabel: vi.fn().mockResolvedValue({ data: {} }),
            create: createIssueMock,
            update: vi.fn(),
            addLabels: vi.fn().mockResolvedValue({}),
          },
        },
        graphql: vi.fn(),
      };

      await detectAndHandleFailureCascade("owner", "repo", 100);

      expect(capturedBody).toContain("⚠️ Failure Cascade Detected");
      expect(capturedBody).toContain("cascade-suspected");
      expect(capturedBody).toContain("#100");
      expect(capturedBody).toContain("[aw] Workflow 0 failed");
    });

    it("filters out non-[aw] failed issues from findRecentFailureIssues", async () => {
      const mixedItems = [
        {
          number: 1,
          title: "[aw] My Workflow failed",
          html_url: "https://github.com/owner/repo/issues/1",
          created_at: new Date().toISOString(),
        },
        {
          number: 2,
          title: "Some other issue title",
          html_url: "https://github.com/owner/repo/issues/2",
          created_at: new Date().toISOString(),
        },
        {
          number: 3,
          title: "[aw] Another Workflow failed",
          html_url: "https://github.com/owner/repo/issues/3",
          created_at: new Date().toISOString(),
        },
      ];

      const searchMock = vi.fn().mockResolvedValue({ data: { total_count: mixedItems.length, items: mixedItems } });

      global.github = {
        rest: {
          search: { issuesAndPullRequests: searchMock },
          issues: { getLabel: vi.fn(), addLabels: vi.fn() },
        },
        graphql: vi.fn(),
      };

      const result = await findRecentFailureIssues("owner", "repo");
      expect(result).toHaveLength(2);
      expect(result.map(i => i.number)).toEqual([1, 3]);
    });

    it("is non-fatal: handles search API errors gracefully", async () => {
      global.github = {
        rest: {
          search: {
            issuesAndPullRequests: vi.fn().mockRejectedValue(new Error("API rate limit exceeded")),
          },
          issues: {
            getLabel: vi.fn(),
            create: vi.fn(),
            addLabels: vi.fn(),
          },
        },
        graphql: vi.fn(),
      };

      // Should not throw
      await expect(detectAndHandleFailureCascade("owner", "repo", 999)).resolves.toBeUndefined();
    });
  });
});
