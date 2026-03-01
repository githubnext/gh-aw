// @ts-check

/**
 * Tests for dynamic_checkout.cjs - Multi-repo checkout utilities
 *
 * Note: These tests are limited because dynamic_checkout.cjs relies heavily on:
 * - GitHub Actions `exec` global for git commands
 * - GitHub Actions `core` global for logging
 *
 * Testing the actual checkout functionality requires mocking these globals,
 * which is done via setup_globals_mock.cjs in more complex integration tests.
 * Here we test the module structure and pure functions where possible.
 */

import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

const { getCurrentCheckoutRepo, checkoutRepo, createCheckoutManager } = require("./dynamic_checkout.cjs");

describe("dynamic_checkout exports", () => {
  it("should export getCurrentCheckoutRepo function", () => {
    expect(typeof getCurrentCheckoutRepo).toBe("function");
  });

  it("should export checkoutRepo function", () => {
    expect(typeof checkoutRepo).toBe("function");
  });

  it("should export createCheckoutManager function", () => {
    expect(typeof createCheckoutManager).toBe("function");
  });
});

describe("createCheckoutManager", () => {
  let mockExec;
  let mockCore;
  let originalExec;
  let originalCore;

  beforeEach(() => {
    // Save original globals
    originalExec = global.exec;
    originalCore = global.core;

    // Create mocks
    mockExec = {
      exec: vi.fn().mockResolvedValue(0),
      getExecOutput: vi.fn().mockResolvedValue({
        stdout: "https://github.com/owner/original-repo.git\n",
        stderr: "",
        exitCode: 0,
      }),
    };

    mockCore = {
      info: vi.fn(),
      error: vi.fn(),
      warning: vi.fn(),
      debug: vi.fn(),
    };

    global.exec = mockExec;
    global.core = mockCore;
  });

  afterEach(() => {
    // Restore globals
    global.exec = originalExec;
    global.core = originalCore;
  });

  it("should create a manager with getCurrent and switchTo methods", () => {
    const manager = createCheckoutManager("fake-token");

    expect(typeof manager.getCurrent).toBe("function");
    expect(typeof manager.switchTo).toBe("function");
  });

  it("should respect defaultBaseBranch option", () => {
    const manager = createCheckoutManager("fake-token", { defaultBaseBranch: "develop" });

    // The manager is created, we can't easily test internal state
    // but we verify it doesn't throw
    expect(manager).toBeDefined();
  });
});

describe("checkoutRepo slug validation", () => {
  let mockExec;
  let mockCore;
  let originalExec;
  let originalCore;

  beforeEach(() => {
    originalExec = global.exec;
    originalCore = global.core;

    mockExec = {
      exec: vi.fn().mockResolvedValue(0),
      getExecOutput: vi.fn().mockResolvedValue({
        stdout: "",
        stderr: "",
        exitCode: 0,
      }),
    };

    mockCore = {
      info: vi.fn(),
      error: vi.fn(),
      warning: vi.fn(),
      debug: vi.fn(),
    };

    global.exec = mockExec;
    global.core = mockCore;
  });

  afterEach(() => {
    global.exec = originalExec;
    global.core = originalCore;
  });

  it("should reject invalid repo slug without slash", async () => {
    const result = await checkoutRepo("invalid-slug-no-slash", "fake-token");

    expect(result.success).toBe(false);
    expect(result.error).toContain("Invalid repository slug");
    expect(result.error).toContain("Expected format: owner/repo");
  });

  it("should reject repo slug with extra slashes", async () => {
    const result = await checkoutRepo("owner/repo/extra", "fake-token");

    expect(result.success).toBe(false);
    expect(result.error).toContain("Invalid repository slug");
    expect(result.error).toContain("Expected format: owner/repo");
  });

  it("should reject empty owner", async () => {
    const result = await checkoutRepo("/repo", "fake-token");

    expect(result.success).toBe(false);
    expect(result.error).toContain("Invalid repository slug");
  });

  it("should reject empty repo", async () => {
    const result = await checkoutRepo("owner/", "fake-token");

    expect(result.success).toBe(false);
    expect(result.error).toContain("Invalid repository slug");
  });

  it("should reject empty string", async () => {
    const result = await checkoutRepo("", "fake-token");

    expect(result.success).toBe(false);
    expect(result.error).toContain("Invalid repository slug");
  });

  it("should accept valid repo slug format", async () => {
    // This will proceed to try git commands, which will fail in mocks
    // but that's ok - we're testing slug validation passed
    const result = await checkoutRepo("owner/repo", "fake-token");

    // Should have proceeded past validation (may fail at git commands)
    // Either it succeeds or fails for git-related reasons, not slug validation
    if (!result.success) {
      expect(result.error).not.toContain("Invalid repository slug");
    }
  });
});

describe("getCurrentCheckoutRepo URL parsing", () => {
  let mockCore;
  let originalExec;
  let originalCore;

  beforeEach(() => {
    originalExec = global.exec;
    originalCore = global.core;

    mockCore = {
      info: vi.fn(),
      error: vi.fn(),
      warning: vi.fn(),
      debug: vi.fn(),
    };

    global.core = mockCore;
  });

  afterEach(() => {
    global.exec = originalExec;
    global.core = originalCore;
  });

  it("should parse HTTPS URL format", async () => {
    global.exec = {
      getExecOutput: vi.fn().mockResolvedValue({
        stdout: "https://github.com/owner/repo.git\n",
        stderr: "",
        exitCode: 0,
      }),
    };

    const result = await getCurrentCheckoutRepo();
    expect(result).toBe("owner/repo");
  });

  it("should parse HTTPS URL without .git suffix", async () => {
    global.exec = {
      getExecOutput: vi.fn().mockResolvedValue({
        stdout: "https://github.com/my-org/my-project\n",
        stderr: "",
        exitCode: 0,
      }),
    };

    const result = await getCurrentCheckoutRepo();
    expect(result).toBe("my-org/my-project");
  });

  it("should parse SSH URL format", async () => {
    global.exec = {
      getExecOutput: vi.fn().mockResolvedValue({
        stdout: "git@github.com:owner/repo.git\n",
        stderr: "",
        exitCode: 0,
      }),
    };

    const result = await getCurrentCheckoutRepo();
    expect(result).toBe("owner/repo");
  });

  it("should handle GitHub Enterprise URLs", async () => {
    global.exec = {
      getExecOutput: vi.fn().mockResolvedValue({
        stdout: "https://github.mycompany.com/team/project.git\n",
        stderr: "",
        exitCode: 0,
      }),
    };

    const result = await getCurrentCheckoutRepo();
    expect(result).toBe("team/project");
  });

  it("should return null on git command error", async () => {
    global.exec = {
      getExecOutput: vi.fn().mockRejectedValue(new Error("git not found")),
    };

    const result = await getCurrentCheckoutRepo();
    expect(result).toBeNull();
  });

  it("should normalize to lowercase", async () => {
    global.exec = {
      getExecOutput: vi.fn().mockResolvedValue({
        stdout: "https://github.com/Owner/Repo.git\n",
        stderr: "",
        exitCode: 0,
      }),
    };

    const result = await getCurrentCheckoutRepo();
    expect(result).toBe("owner/repo");
  });
});
