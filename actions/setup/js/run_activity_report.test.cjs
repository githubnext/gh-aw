// @ts-check
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

describe("run_activity_report", () => {
  let originalGlobals;
  let originalEnv;
  let mockCore;
  let mockGithub;
  let mockContext;
  let mockExec;

  beforeEach(() => {
    originalEnv = { ...process.env };
    process.env.GH_AW_CMD_PREFIX = "gh aw";

    originalGlobals = {
      core: global.core,
      github: global.github,
      context: global.context,
      exec: global.exec,
    };

    mockCore = {
      info: vi.fn(),
      warning: vi.fn(),
    };
    mockGithub = {
      rest: {
        issues: {
          create: vi.fn().mockResolvedValue({
            data: { number: 42, html_url: "https://github.com/testowner/testrepo/issues/42" },
          }),
        },
      },
    };
    mockContext = {
      repo: {
        owner: "testowner",
        repo: "testrepo",
      },
    };
    mockExec = {
      getExecOutput: vi.fn(),
    };

    global.core = mockCore;
    global.github = mockGithub;
    global.context = mockContext;
    global.exec = mockExec;
  });

  afterEach(() => {
    process.env = originalEnv;
    global.core = originalGlobals.core;
    global.github = originalGlobals.github;
    global.context = originalGlobals.context;
    global.exec = originalGlobals.exec;
    vi.clearAllMocks();
  });

  it("creates an activity report issue with all three time ranges", async () => {
    mockExec.getExecOutput
      .mockResolvedValueOnce({ stdout: "## 24h report\nok", stderr: "", exitCode: 0 })
      .mockResolvedValueOnce({ stdout: "## 7d report\nok", stderr: "", exitCode: 0 })
      .mockResolvedValueOnce({ stdout: "## 30d report\nok", stderr: "", exitCode: 0 });

    const { main } = await import("./run_activity_report.cjs");
    await main();

    expect(mockExec.getExecOutput).toHaveBeenCalledTimes(3);
    expect(mockExec.getExecOutput).toHaveBeenNthCalledWith(
      1,
      "gh",
      expect.arrayContaining(["aw", "logs", "--repo", "testowner/testrepo", "--start-date", "-1d", "--count", "100", "--format", "markdown"]),
      expect.objectContaining({ ignoreReturnCode: true })
    );
    expect(mockExec.getExecOutput).toHaveBeenNthCalledWith(
      2,
      "gh",
      expect.arrayContaining(["aw", "logs", "--repo", "testowner/testrepo", "--start-date", "-1w", "--count", "100", "--format", "markdown"]),
      expect.objectContaining({ ignoreReturnCode: true })
    );
    expect(mockExec.getExecOutput).toHaveBeenNthCalledWith(
      3,
      "gh",
      expect.arrayContaining(["aw", "logs", "--repo", "testowner/testrepo", "--start-date", "-1mo", "--count", "100", "--format", "markdown"]),
      expect.objectContaining({ ignoreReturnCode: true })
    );

    expect(mockGithub.rest.issues.create).toHaveBeenCalledWith(
      expect.objectContaining({
        owner: "testowner",
        repo: "testrepo",
        title: "[aw] agentic status report",
        labels: ["agentic-workflows"],
      })
    );

    const issueBody = mockGithub.rest.issues.create.mock.calls[0][0].body;
    expect(issueBody).toContain("### Agentic workflow activity report");
    expect(issueBody).toContain("<details>");
    expect(issueBody).toContain("<summary>Last 24 hours</summary>");
    expect(issueBody).toContain("<summary>Last 7 days</summary>");
    expect(issueBody).toContain("<summary>Last 30 days</summary>");
    expect(issueBody).toContain("#### 24h report");
  });

  it("skips the 30-day query when rate limited", async () => {
    mockExec.getExecOutput
      .mockResolvedValueOnce({ stdout: "24h", stderr: "", exitCode: 0 })
      .mockResolvedValueOnce({ stdout: "7d", stderr: "", exitCode: 0 })
      .mockResolvedValueOnce({ stdout: "", stderr: "API rate limit exceeded", exitCode: 1 });

    const { main } = await import("./run_activity_report.cjs");
    await main();

    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Skipping Last 30 days report"));
    const issueBody = mockGithub.rest.issues.create.mock.calls[0][0].body;
    expect(issueBody).toContain("Skipped due to GitHub API rate limiting.");
  });

  it("detects rate limit text helper", async () => {
    const { hasRateLimitText } = await import("./run_activity_report.cjs");
    expect(hasRateLimitText("API rate limit exceeded")).toBe(true);
    expect(hasRateLimitText("secondary rate limit")).toBe(true);
    expect(hasRateLimitText("normal output")).toBe(false);
  });

  it("demotes report headings by two levels", async () => {
    const { normalizeReportMarkdown } = await import("./run_activity_report.cjs");
    const transformed = normalizeReportMarkdown("# H1\n## H2\n### H3");
    expect(transformed).toContain("### H1");
    expect(transformed).toContain("#### H2");
    expect(transformed).toContain("##### H3");
  });
});
