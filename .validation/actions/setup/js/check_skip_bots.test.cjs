import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

describe("check_skip_bots.cjs", () => {
  let mockCore;
  let mockContext;

  beforeEach(() => {
    // Mock core actions methods
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

    mockContext = {
      actor: "test-user",
      eventName: "issues",
      repo: {
        owner: "test-owner",
        repo: "test-repo",
      },
    };

    // Set up global mocks
    global.core = mockCore;
    global.context = mockContext;

    // Clear environment variables
    delete process.env.GH_AW_SKIP_BOTS;

    // Clear module cache to ensure fresh import
    vi.resetModules();
  });

  afterEach(() => {
    vi.clearAllMocks();
    delete global.core;
    delete global.context;
  });

  it("should allow workflow when no skip-bots configured", async () => {
    delete process.env.GH_AW_SKIP_BOTS;

    const { main } = await import("./check_skip_bots.cjs");
    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_bots_ok", "true");
    expect(mockCore.setOutput).toHaveBeenCalledWith("result", "no_skip_bots");
  });

  it("should skip workflow for exact username match", async () => {
    process.env.GH_AW_SKIP_BOTS = "test-user,other-user";
    mockContext.actor = "test-user";

    const { main } = await import("./check_skip_bots.cjs");
    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_bots_ok", "false");
    expect(mockCore.setOutput).toHaveBeenCalledWith("result", "skipped");
  });

  it("should allow workflow when user not in skip-bots", async () => {
    process.env.GH_AW_SKIP_BOTS = "other-user,another-user";
    mockContext.actor = "test-user";

    const { main } = await import("./check_skip_bots.cjs");
    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_bots_ok", "true");
    expect(mockCore.setOutput).toHaveBeenCalledWith("result", "not_skipped");
  });

  it("should skip workflow for bot with [bot] suffix when base name in skip-bots", async () => {
    process.env.GH_AW_SKIP_BOTS = "github-actions,copilot";
    mockContext.actor = "github-actions[bot]";

    const { main } = await import("./check_skip_bots.cjs");
    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_bots_ok", "false");
    expect(mockCore.setOutput).toHaveBeenCalledWith("result", "skipped");
  });

  it("should skip workflow for base name when skip-bots has [bot] suffix", async () => {
    process.env.GH_AW_SKIP_BOTS = "github-actions[bot],copilot[bot]";
    mockContext.actor = "github-actions";

    const { main } = await import("./check_skip_bots.cjs");
    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_bots_ok", "false");
    expect(mockCore.setOutput).toHaveBeenCalledWith("result", "skipped");
  });

  it("should skip workflow for exact match with [bot] suffix", async () => {
    process.env.GH_AW_SKIP_BOTS = "github-actions[bot]";
    mockContext.actor = "github-actions[bot]";

    const { main } = await import("./check_skip_bots.cjs");
    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_bots_ok", "false");
    expect(mockCore.setOutput).toHaveBeenCalledWith("result", "skipped");
  });

  it("should handle multiple users with mixed bot syntax", async () => {
    process.env.GH_AW_SKIP_BOTS = "user1,github-actions,copilot[bot]";
    mockContext.actor = "copilot";

    const { main } = await import("./check_skip_bots.cjs");
    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_bots_ok", "false");
    expect(mockCore.setOutput).toHaveBeenCalledWith("result", "skipped");
  });

  it("should not skip for partial matches", async () => {
    process.env.GH_AW_SKIP_BOTS = "github-actions";
    mockContext.actor = "github-actions-bot";

    const { main } = await import("./check_skip_bots.cjs");
    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_bots_ok", "true");
    expect(mockCore.setOutput).toHaveBeenCalledWith("result", "not_skipped");
  });

  it("should handle whitespace in skip-bots list", async () => {
    process.env.GH_AW_SKIP_BOTS = " github-actions , copilot , renovate ";
    mockContext.actor = "copilot[bot]";

    const { main } = await import("./check_skip_bots.cjs");
    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_bots_ok", "false");
    expect(mockCore.setOutput).toHaveBeenCalledWith("result", "skipped");
  });

  describe("confused deputy attack protection", () => {
    it("should not skip workflow when actor differs from PR author (pull_request synchronize event)", async () => {
      process.env.GH_AW_SKIP_BOTS = "dependabot[bot]";
      mockContext.actor = "dependabot[bot]";
      mockContext.eventName = "pull_request";
      mockContext.payload = { action: "synchronize", pull_request: { user: { login: "attacker" } } };

      const { main } = await import("./check_skip_bots.cjs");
      await main();

      // Even though dependabot[bot] is in skip-bots, confused deputy prevents skipping
      expect(mockCore.setOutput).toHaveBeenCalledWith("skip_bots_ok", "true");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "not_skipped");
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Potential confused deputy attack detected"));
    });

    it("should skip workflow when actor matches PR author (genuine dependabot PR synchronize)", async () => {
      process.env.GH_AW_SKIP_BOTS = "dependabot[bot]";
      mockContext.actor = "dependabot[bot]";
      mockContext.eventName = "pull_request";
      mockContext.payload = { action: "synchronize", pull_request: { user: { login: "dependabot[bot]" } } };

      const { main } = await import("./check_skip_bots.cjs");
      await main();

      expect(mockCore.setOutput).toHaveBeenCalledWith("skip_bots_ok", "false");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "skipped");
    });

    it("should not skip workflow when actor differs from comment author (issue_comment event)", async () => {
      process.env.GH_AW_SKIP_BOTS = "dependabot[bot]";
      mockContext.actor = "dependabot[bot]";
      mockContext.eventName = "issue_comment";
      mockContext.payload = { comment: { user: { login: "attacker" } } };

      const { main } = await import("./check_skip_bots.cjs");
      await main();

      expect(mockCore.setOutput).toHaveBeenCalledWith("skip_bots_ok", "true");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "not_skipped");
    });

    it("should skip workflow for pull_request:labeled even when actor differs from PR author", async () => {
      // A team member labeling a PR is legitimate — confused deputy only fires on synchronize
      process.env.GH_AW_SKIP_BOTS = "pelikhan";
      mockContext.actor = "pelikhan";
      mockContext.eventName = "pull_request";
      mockContext.payload = { action: "labeled", pull_request: { user: { login: "copilot[bot]" } } };

      const { main } = await import("./check_skip_bots.cjs");
      await main();

      // Not a confused deputy — apply skip-bots normally
      expect(mockCore.setOutput).toHaveBeenCalledWith("skip_bots_ok", "false");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "skipped");
    });

    it("should skip workflow normally when no payload (non-PR events)", async () => {
      process.env.GH_AW_SKIP_BOTS = "dependabot[bot]";
      mockContext.actor = "dependabot[bot]";
      mockContext.eventName = "issues";
      // No payload field (existing test behavior)

      const { main } = await import("./check_skip_bots.cjs");
      await main();

      expect(mockCore.setOutput).toHaveBeenCalledWith("skip_bots_ok", "false");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "skipped");
    });

    it("should apply skip-bots normally for workflow_call events (no false confused deputy positive)", async () => {
      // In workflow_call, context.payload = { inputs: { aw_context: "..." } }
      // aw_context carries event_type but NOT pull_request.user.login
      // Confused deputy check must NOT trigger - apply skip-bots rule normally
      process.env.GH_AW_SKIP_BOTS = "dependabot[bot]";
      mockContext.actor = "dependabot[bot]";
      mockContext.eventName = "workflow_call";
      mockContext.payload = {
        inputs: {
          aw_context: JSON.stringify({ event_type: "pull_request", item_number: "42", actor: "attacker" }),
        },
      };

      const { main } = await import("./check_skip_bots.cjs");
      await main();

      // workflow_call: no confused deputy - normal skip-bots logic applies
      expect(mockCore.setOutput).toHaveBeenCalledWith("skip_bots_ok", "false");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "skipped");
    });
  });
});
