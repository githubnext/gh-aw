import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

describe("report_pre_activation_skip.cjs", () => {
  let mockCore;

  beforeEach(() => {
    mockCore = {
      info: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
      setFailed: vi.fn(),
      setOutput: vi.fn(),
      summary: {
        addRaw: vi.fn().mockReturnThis(),
        write: vi.fn().mockResolvedValue(undefined),
      },
    };

    global.core = mockCore;

    // Clear all relevant env vars
    const vars = [
      "GH_AW_IS_TEAM_MEMBER",
      "GH_AW_MEMBERSHIP_RESULT",
      "GH_AW_MEMBERSHIP_ERROR_MESSAGE",
      "GH_AW_STOP_TIME_OK",
      "GH_AW_RATE_LIMIT_OK",
      "GH_AW_SKIP_CHECK_OK",
      "GH_AW_SKIP_NO_MATCH_OK",
      "GH_AW_SKIP_IF_CHECK_FAILING_OK",
      "GH_AW_SKIP_ROLES_OK",
      "GH_AW_SKIP_ROLES_ERROR_MESSAGE",
      "GH_AW_SKIP_BOTS_OK",
      "GH_AW_SKIP_BOTS_ERROR_MESSAGE",
      "GH_AW_COMMAND_POSITION_OK",
    ];
    for (const v of vars) {
      delete process.env[v];
    }

    vi.resetModules();
  });

  afterEach(() => {
    vi.clearAllMocks();
    delete global.core;
  });

  // ---- main() ----

  it("should not write summary when all checks pass", async () => {
    process.env.GH_AW_IS_TEAM_MEMBER = "true";
    process.env.GH_AW_STOP_TIME_OK = "true";
    process.env.GH_AW_RATE_LIMIT_OK = "true";

    const { main } = await import("./report_pre_activation_skip.cjs");
    await main();

    expect(mockCore.summary.addRaw).not.toHaveBeenCalled();
    expect(mockCore.summary.write).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("All pre-activation checks passed"));
  });

  it("should not write summary when no check env vars are set (checks not configured)", async () => {
    const { main } = await import("./report_pre_activation_skip.cjs");
    await main();

    expect(mockCore.summary.addRaw).not.toHaveBeenCalled();
  });

  it("should write summary when membership check fails", async () => {
    process.env.GH_AW_IS_TEAM_MEMBER = "false";
    process.env.GH_AW_MEMBERSHIP_RESULT = "insufficient_permissions";
    process.env.GH_AW_MEMBERSHIP_ERROR_MESSAGE = "Access denied: User 'prd-to-prod-pipeline[bot]' is not authorized.";

    const { main } = await import("./report_pre_activation_skip.cjs");
    await main();

    expect(mockCore.summary.addRaw).toHaveBeenCalledTimes(1);
    expect(mockCore.summary.write).toHaveBeenCalledTimes(1);

    const summaryText = mockCore.summary.addRaw.mock.calls[0][0];
    expect(summaryText).toContain("⏭️ Workflow Activation Skipped");
    expect(summaryText).toContain("Role / bot check");
    expect(summaryText).toContain("prd-to-prod-pipeline[bot]");
    expect(summaryText).toContain("on.bots:");
    expect(summaryText).toContain("on.roles:");
  });

  it("should write summary when stop-time check fails", async () => {
    process.env.GH_AW_STOP_TIME_OK = "false";

    const { main } = await import("./report_pre_activation_skip.cjs");
    await main();

    expect(mockCore.summary.addRaw).toHaveBeenCalledTimes(1);
    const summaryText = mockCore.summary.addRaw.mock.calls[0][0];
    expect(summaryText).toContain("Stop-time limit");
    expect(summaryText).toContain("on.stop-time:");
  });

  it("should write summary when rate-limit check fails", async () => {
    process.env.GH_AW_RATE_LIMIT_OK = "false";

    const { main } = await import("./report_pre_activation_skip.cjs");
    await main();

    const summaryText = mockCore.summary.addRaw.mock.calls[0][0];
    expect(summaryText).toContain("Rate-limit check");
    expect(summaryText).toContain("on.rate-limit:");
  });

  it("should write summary when skip-if-match check fails", async () => {
    process.env.GH_AW_SKIP_CHECK_OK = "false";

    const { main } = await import("./report_pre_activation_skip.cjs");
    await main();

    const summaryText = mockCore.summary.addRaw.mock.calls[0][0];
    expect(summaryText).toContain("Skip-if-match check");
    expect(summaryText).toContain("on.skip-if-match:");
  });

  it("should write summary when skip-if-no-match check fails", async () => {
    process.env.GH_AW_SKIP_NO_MATCH_OK = "false";

    const { main } = await import("./report_pre_activation_skip.cjs");
    await main();

    const summaryText = mockCore.summary.addRaw.mock.calls[0][0];
    expect(summaryText).toContain("Skip-if-no-match check");
    expect(summaryText).toContain("on.skip-if-no-match:");
  });

  it("should write summary when skip-if-check-failing check fails", async () => {
    process.env.GH_AW_SKIP_IF_CHECK_FAILING_OK = "false";

    const { main } = await import("./report_pre_activation_skip.cjs");
    await main();

    const summaryText = mockCore.summary.addRaw.mock.calls[0][0];
    expect(summaryText).toContain("Skip-if-check-failing");
    expect(summaryText).toContain("on.skip-if-check-failing:");
  });

  it("should write summary when skip-roles check fails with error message", async () => {
    process.env.GH_AW_SKIP_ROLES_OK = "false";
    process.env.GH_AW_SKIP_ROLES_ERROR_MESSAGE = "Workflow skipped: User 'admin-user' has role 'admin'";

    const { main } = await import("./report_pre_activation_skip.cjs");
    await main();

    const summaryText = mockCore.summary.addRaw.mock.calls[0][0];
    expect(summaryText).toContain("Skip-roles check");
    expect(summaryText).toContain("admin-user");
    expect(summaryText).toContain("on.skip-roles:");
  });

  it("should write summary when skip-bots check fails with error message", async () => {
    process.env.GH_AW_SKIP_BOTS_OK = "false";
    process.env.GH_AW_SKIP_BOTS_ERROR_MESSAGE = "Workflow skipped: User 'renovate[bot]' is in skip-bots";

    const { main } = await import("./report_pre_activation_skip.cjs");
    await main();

    const summaryText = mockCore.summary.addRaw.mock.calls[0][0];
    expect(summaryText).toContain("Skip-bots check");
    expect(summaryText).toContain("renovate[bot]");
    expect(summaryText).toContain("on.skip-bots:");
  });

  it("should write summary when command-position check fails", async () => {
    process.env.GH_AW_COMMAND_POSITION_OK = "false";

    const { main } = await import("./report_pre_activation_skip.cjs");
    await main();

    const summaryText = mockCore.summary.addRaw.mock.calls[0][0];
    expect(summaryText).toContain("Command check");
    expect(summaryText).toContain("on.command:");
  });

  it("should write summary with multiple failures", async () => {
    process.env.GH_AW_IS_TEAM_MEMBER = "false";
    process.env.GH_AW_MEMBERSHIP_RESULT = "insufficient_permissions";
    process.env.GH_AW_STOP_TIME_OK = "false";

    const { main } = await import("./report_pre_activation_skip.cjs");
    await main();

    const summaryText = mockCore.summary.addRaw.mock.calls[0][0];
    expect(summaryText).toContain("Role / bot check");
    expect(summaryText).toContain("Stop-time limit");
  });

  it("should include a link to gh-aw in the summary footer", async () => {
    process.env.GH_AW_IS_TEAM_MEMBER = "false";

    const { main } = await import("./report_pre_activation_skip.cjs");
    await main();

    const summaryText = mockCore.summary.addRaw.mock.calls[0][0];
    expect(summaryText).toContain("gh-aw");
    expect(summaryText).toContain("pre_activation");
  });

  // ---- collectSkipReasons() ----

  it("should return empty array when no checks failed", async () => {
    const { collectSkipReasons } = await import("./report_pre_activation_skip.cjs");
    const reasons = collectSkipReasons();
    expect(reasons).toHaveLength(0);
  });

  it("should return one reason when membership check fails", async () => {
    process.env.GH_AW_IS_TEAM_MEMBER = "false";
    process.env.GH_AW_MEMBERSHIP_RESULT = "bot_not_active";
    process.env.GH_AW_MEMBERSHIP_ERROR_MESSAGE = "Bot is not active";

    const { collectSkipReasons } = await import("./report_pre_activation_skip.cjs");
    const reasons = collectSkipReasons();

    expect(reasons).toHaveLength(1);
    expect(reasons[0].check).toBe("Role / bot check");
    expect(reasons[0].result).toBe("bot_not_active");
    expect(reasons[0].message).toBe("Bot is not active");
  });

  // ---- buildMembershipRemediation() ----

  it("should return insufficient_permissions remediation mentioning on.bots and on.roles", async () => {
    const { buildMembershipRemediation } = await import("./report_pre_activation_skip.cjs");
    const hint = buildMembershipRemediation("insufficient_permissions");
    expect(hint).toContain("on.bots:");
    expect(hint).toContain("on.roles:");
  });

  it("should return bot_not_active remediation about GitHub App installation", async () => {
    const { buildMembershipRemediation } = await import("./report_pre_activation_skip.cjs");
    const hint = buildMembershipRemediation("bot_not_active");
    expect(hint).toContain("Install the GitHub App");
  });

  it("should return api_error remediation mentioning the job log", async () => {
    const { buildMembershipRemediation } = await import("./report_pre_activation_skip.cjs");
    const hint = buildMembershipRemediation("api_error");
    expect(hint).toContain("pre_activation");
  });

  it("should return config_error remediation mentioning the administrator", async () => {
    const { buildMembershipRemediation } = await import("./report_pre_activation_skip.cjs");
    const hint = buildMembershipRemediation("config_error");
    expect(hint).toContain("administrator");
  });

  it("should return default remediation for unknown result codes", async () => {
    const { buildMembershipRemediation } = await import("./report_pre_activation_skip.cjs");
    const hint = buildMembershipRemediation("unknown_result");
    expect(hint).toContain("on.bots:");
  });
});
