import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

describe("check_skip_users.cjs", () => {
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
    delete process.env.GH_AW_SKIP_USERS;

    // Clear module cache to ensure fresh import
    vi.resetModules();
  });

  afterEach(() => {
    vi.clearAllMocks();
    delete global.core;
    delete global.context;
  });

  it("should allow workflow when no skip-users configured", async () => {
    delete process.env.GH_AW_SKIP_USERS;

    const { main } = await import("./check_skip_users.cjs");
    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_users_ok", "true");
    expect(mockCore.setOutput).toHaveBeenCalledWith("result", "no_skip_users");
  });

  it("should skip workflow for exact username match", async () => {
    process.env.GH_AW_SKIP_USERS = "test-user,other-user";
    mockContext.actor = "test-user";

    const { main } = await import("./check_skip_users.cjs");
    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_users_ok", "false");
    expect(mockCore.setOutput).toHaveBeenCalledWith("result", "skipped");
  });

  it("should allow workflow when user not in skip-users", async () => {
    process.env.GH_AW_SKIP_USERS = "other-user,another-user";
    mockContext.actor = "test-user";

    const { main } = await import("./check_skip_users.cjs");
    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_users_ok", "true");
    expect(mockCore.setOutput).toHaveBeenCalledWith("result", "not_skipped");
  });

  it("should skip workflow for bot with [bot] suffix when base name in skip-users", async () => {
    process.env.GH_AW_SKIP_USERS = "github-actions,copilot";
    mockContext.actor = "github-actions[bot]";

    const { main } = await import("./check_skip_users.cjs");
    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_users_ok", "false");
    expect(mockCore.setOutput).toHaveBeenCalledWith("result", "skipped");
  });

  it("should skip workflow for base name when skip-users has [bot] suffix", async () => {
    process.env.GH_AW_SKIP_USERS = "github-actions[bot],copilot[bot]";
    mockContext.actor = "github-actions";

    const { main } = await import("./check_skip_users.cjs");
    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_users_ok", "false");
    expect(mockCore.setOutput).toHaveBeenCalledWith("result", "skipped");
  });

  it("should skip workflow for exact match with [bot] suffix", async () => {
    process.env.GH_AW_SKIP_USERS = "github-actions[bot]";
    mockContext.actor = "github-actions[bot]";

    const { main } = await import("./check_skip_users.cjs");
    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_users_ok", "false");
    expect(mockCore.setOutput).toHaveBeenCalledWith("result", "skipped");
  });

  it("should handle multiple users with mixed bot syntax", async () => {
    process.env.GH_AW_SKIP_USERS = "user1,github-actions,copilot[bot]";
    mockContext.actor = "copilot";

    const { main } = await import("./check_skip_users.cjs");
    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_users_ok", "false");
    expect(mockCore.setOutput).toHaveBeenCalledWith("result", "skipped");
  });

  it("should not skip for partial matches", async () => {
    process.env.GH_AW_SKIP_USERS = "github-actions";
    mockContext.actor = "github-actions-bot";

    const { main } = await import("./check_skip_users.cjs");
    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_users_ok", "true");
    expect(mockCore.setOutput).toHaveBeenCalledWith("result", "not_skipped");
  });

  it("should handle whitespace in skip-users list", async () => {
    process.env.GH_AW_SKIP_USERS = " github-actions , copilot , renovate ";
    mockContext.actor = "copilot[bot]";

    const { main } = await import("./check_skip_users.cjs");
    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_users_ok", "false");
    expect(mockCore.setOutput).toHaveBeenCalledWith("result", "skipped");
  });
});
