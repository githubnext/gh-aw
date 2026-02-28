// @ts-check
/// <reference types="@actions/github-script" />

import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

// Mock @actions/github so we can control getOctokit
vi.mock("@actions/github", () => ({
  getOctokit: vi.fn(token => ({
    _token: token,
    rest: {},
    graphql: vi.fn(),
  })),
}));

import { createAuthenticatedGitHubClient } from "./handler_auth.cjs";

describe("createAuthenticatedGitHubClient", () => {
  let mockCore;
  let mockGithub;
  let originalGlobals;

  beforeEach(() => {
    originalGlobals = {
      core: global.core,
      github: global.github,
    };

    mockCore = {
      info: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
    };

    // The global github object (step-level token)
    mockGithub = {
      _token: "step-level-token",
      rest: {},
      graphql: vi.fn(),
    };

    global.core = mockCore;
    global.github = mockGithub;
  });

  afterEach(() => {
    global.core = originalGlobals.core;
    global.github = originalGlobals.github;
    vi.clearAllMocks();
  });

  it("returns the global github when no github-token in config", async () => {
    const client = await createAuthenticatedGitHubClient({});
    expect(client).toBe(mockGithub);
    expect(mockCore.info).not.toHaveBeenCalled();
  });

  it("returns the global github when config is empty object", async () => {
    const client = await createAuthenticatedGitHubClient({ max: 10, target: "triggering" });
    expect(client).toBe(mockGithub);
  });

  it("creates a new Octokit when github-token is set in config", async () => {
    const client = await createAuthenticatedGitHubClient({ "github-token": "my-pat-token" });

    // Should NOT be the global github
    expect(client).not.toBe(mockGithub);
    // Should have the token from config
    expect(client._token).toBe("my-pat-token");
  });

  it("logs a message when using per-handler token", async () => {
    await createAuthenticatedGitHubClient({ "github-token": "my-pat-token" });
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("per-handler github-token"));
  });

  it("creates distinct Octokit instances for different tokens", async () => {
    const client1 = await createAuthenticatedGitHubClient({ "github-token": "token-1" });
    const client2 = await createAuthenticatedGitHubClient({ "github-token": "token-2" });

    expect(client1._token).toBe("token-1");
    expect(client2._token).toBe("token-2");
    expect(client1).not.toBe(client2);
  });

  it("does not mutate the global github object", async () => {
    const originalGithub = global.github;
    await createAuthenticatedGitHubClient({ "github-token": "my-pat-token" });

    // Global should be unchanged
    expect(global.github).toBe(originalGithub);
    expect(global.github._token).toBe("step-level-token");
  });
});
