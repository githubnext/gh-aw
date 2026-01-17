// @ts-check
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";

/**
 * Tests for handle_missing_secret.cjs
 */

describe("handle_missing_secret", () => {
  let handleMissingSecret;

  beforeEach(async () => {
    vi.clearAllMocks();

    // Mock fs.existsSync
    vi.spyOn(fs, "existsSync").mockReturnValue(false);

    // The module exports main function that depends on global context, github, and core
    handleMissingSecret = await import("./handle_missing_secret.cjs");
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("should export main function", () => {
    expect(handleMissingSecret.main).toBeDefined();
    expect(typeof handleMissingSecret.main).toBe("function");
  });

  it("should skip when missing_secret_info.json does not exist", async () => {
    // Setup global mocks
    global.core = {
      info: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
    };
    global.context = {
      repo: { owner: "test-owner", repo: "test-repo" },
    };
    global.github = {
      rest: {
        search: { issuesAndPullRequests: vi.fn() },
        issues: { create: vi.fn(), createComment: vi.fn() },
      },
      graphql: vi.fn(),
    };

    // Mock fs to return file doesn't exist
    fs.existsSync.mockReturnValue(false);

    await handleMissingSecret.main();

    expect(core.info).toHaveBeenCalledWith("No missing secret info file found, skipping issue creation");
  });

  it("should read and process missing_secret_info.json when it exists", async () => {
    // Setup global mocks
    global.core = {
      info: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
    };
    global.context = {
      repo: { owner: "test-owner", repo: "test-repo" },
    };
    global.github = {
      rest: {
        search: {
          issuesAndPullRequests: vi.fn().mockResolvedValue({
            data: { total_count: 0, items: [] },
          }),
        },
        issues: {
          create: vi.fn().mockResolvedValue({
            data: { number: 123, html_url: "https://github.com/test/issue/123", node_id: "node123" },
          }),
          createComment: vi.fn(),
        },
      },
      graphql: vi.fn(),
    };
    global.process = {
      env: {
        GH_AW_WORKFLOW_NAME: "Test Workflow",
        GH_AW_WORKFLOW_SOURCE: "test.md",
        GH_AW_WORKFLOW_SOURCE_URL: "https://github.com/test/workflow",
        GH_AW_RUN_URL: "https://github.com/test/actions/runs/123",
      },
    };

    // Mock fs to return file exists and provide content
    fs.existsSync.mockReturnValue(true);
    fs.readFileSync = vi.fn().mockReturnValue(
      JSON.stringify({
        missing_secrets: ["SECRET_1", "SECRET_2"],
        engine_name: "Test Engine",
        docs_url: "https://docs.example.com",
      })
    );

    await handleMissingSecret.main();

    expect(fs.readFileSync).toHaveBeenCalledWith("/tmp/gh-aw/missing_secret_info.json", "utf8");
    expect(core.info).toHaveBeenCalledWith(expect.stringContaining("Missing secret info loaded"));
  });

  // Additional integration tests would require more comprehensive mocking
  // of the GitHub API and the issue_helpers module
});
