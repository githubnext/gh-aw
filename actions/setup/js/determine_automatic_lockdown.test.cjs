import { describe, it, expect, beforeEach, vi } from "vitest";

describe("determine_automatic_lockdown", () => {
  let mockContext;
  let mockGithub;
  let mockCore;
  let determineAutomaticLockdown;

  beforeEach(async () => {
    vi.resetModules();

    // Setup mock context
    mockContext = {
      repo: {
        owner: "test-owner",
        repo: "test-repo",
      },
    };

    // Setup mock GitHub API
    mockGithub = {
      rest: {
        repos: {
          get: vi.fn(),
        },
      },
    };

    // Setup mock core
    mockCore = {
      info: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
      setOutput: vi.fn(),
      summary: {
        addRaw: vi.fn().mockReturnThis(),
        write: vi.fn().mockResolvedValue(undefined),
      },
    };

    // Reset process.env
    delete process.env.GH_AW_GITHUB_TOKEN;
    delete process.env.GH_AW_GITHUB_MCP_SERVER_TOKEN;
    delete process.env.CUSTOM_GITHUB_TOKEN;
    delete process.env.GH_AW_GITHUB_MIN_INTEGRITY;
    delete process.env.GH_AW_GITHUB_REPOS;

    // Import the module
    determineAutomaticLockdown = (await import("./determine_automatic_lockdown.cjs")).default;
  });

  it("should set min_integrity=approved and repos=all for public repository (no guard policy configured)", async () => {
    mockGithub.rest.repos.get.mockResolvedValue({
      data: {
        private: false,
        visibility: "public",
      },
    });

    await determineAutomaticLockdown(mockGithub, mockContext, mockCore);

    expect(mockGithub.rest.repos.get).toHaveBeenCalledWith({
      owner: "test-owner",
      repo: "test-repo",
    });
    expect(mockCore.setOutput).toHaveBeenCalledWith("min_integrity", "approved");
    expect(mockCore.setOutput).toHaveBeenCalledWith("repos", "all");
    expect(mockCore.setOutput).toHaveBeenCalledWith("visibility", "public");
    expect(mockCore.setOutput).not.toHaveBeenCalledWith("lockdown", expect.anything());
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("guard policy automatically applied"));
  });

  it("should not override min_integrity when already configured", async () => {
    process.env.GH_AW_GITHUB_MIN_INTEGRITY = "merged";

    mockGithub.rest.repos.get.mockResolvedValue({
      data: {
        private: false,
        visibility: "public",
      },
    });

    await determineAutomaticLockdown(mockGithub, mockContext, mockCore);

    expect(mockCore.setOutput).toHaveBeenCalledWith("min_integrity", "merged");
    expect(mockCore.setOutput).toHaveBeenCalledWith("repos", "all");
    expect(mockCore.setOutput).toHaveBeenCalledWith("visibility", "public");
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("min-integrity already configured as 'merged'"));
  });

  it("should not override repos when already configured", async () => {
    process.env.GH_AW_GITHUB_REPOS = "public";

    mockGithub.rest.repos.get.mockResolvedValue({
      data: {
        private: false,
        visibility: "public",
      },
    });

    await determineAutomaticLockdown(mockGithub, mockContext, mockCore);

    expect(mockCore.setOutput).toHaveBeenCalledWith("min_integrity", "approved");
    expect(mockCore.setOutput).toHaveBeenCalledWith("repos", "public");
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("repos already configured as 'public'"));
  });

  it("should set min_integrity=none and repos=all for private repository (no guard policy configured)", async () => {
    mockGithub.rest.repos.get.mockResolvedValue({
      data: {
        private: true,
        visibility: "private",
      },
    });

    await determineAutomaticLockdown(mockGithub, mockContext, mockCore);

    expect(mockGithub.rest.repos.get).toHaveBeenCalledWith({
      owner: "test-owner",
      repo: "test-repo",
    });
    expect(mockCore.setOutput).toHaveBeenCalledWith("min_integrity", "none");
    expect(mockCore.setOutput).toHaveBeenCalledWith("repos", "all");
    expect(mockCore.setOutput).toHaveBeenCalledWith("visibility", "private");
    expect(mockCore.setOutput).not.toHaveBeenCalledWith("lockdown", expect.anything());
    expect(mockCore.warning).not.toHaveBeenCalled();
  });

  it("should set min_integrity=none and repos=all for internal repository (no guard policy configured)", async () => {
    mockGithub.rest.repos.get.mockResolvedValue({
      data: {
        private: true,
        visibility: "internal",
      },
    });

    await determineAutomaticLockdown(mockGithub, mockContext, mockCore);

    expect(mockCore.setOutput).toHaveBeenCalledWith("min_integrity", "none");
    expect(mockCore.setOutput).toHaveBeenCalledWith("repos", "all");
    expect(mockCore.setOutput).toHaveBeenCalledWith("visibility", "internal");
  });

  it("should handle API failure and default to safe guard policy", async () => {
    const error = new Error("API request failed");
    mockGithub.rest.repos.get.mockRejectedValue(error);

    await determineAutomaticLockdown(mockGithub, mockContext, mockCore);

    expect(mockCore.error).toHaveBeenCalledWith("Failed to determine automatic guard policy: API request failed");
    expect(mockCore.setOutput).toHaveBeenCalledWith("min_integrity", "approved");
    expect(mockCore.setOutput).toHaveBeenCalledWith("repos", "all");
    expect(mockCore.setOutput).toHaveBeenCalledWith("visibility", "unknown");
    expect(mockCore.setOutput).not.toHaveBeenCalledWith("lockdown", expect.anything());
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Failed to determine repository visibility"));
  });

  it("should infer visibility from private field when visibility field is missing", async () => {
    mockGithub.rest.repos.get.mockResolvedValue({
      data: {
        private: false,
        // visibility field not present
      },
    });

    await determineAutomaticLockdown(mockGithub, mockContext, mockCore);

    expect(mockCore.setOutput).toHaveBeenCalledWith("min_integrity", "approved");
    expect(mockCore.setOutput).toHaveBeenCalledWith("repos", "all");
    expect(mockCore.setOutput).toHaveBeenCalledWith("visibility", "public");
  });

  it("should not override either guard policy field when both are already configured", async () => {
    process.env.GH_AW_GITHUB_MIN_INTEGRITY = "approved";
    process.env.GH_AW_GITHUB_REPOS = "public";

    mockGithub.rest.repos.get.mockResolvedValue({
      data: {
        private: false,
        visibility: "public",
      },
    });

    await determineAutomaticLockdown(mockGithub, mockContext, mockCore);

    expect(mockCore.setOutput).toHaveBeenCalledWith("min_integrity", "approved");
    expect(mockCore.setOutput).toHaveBeenCalledWith("repos", "public");
    expect(mockCore.setOutput).toHaveBeenCalledWith("visibility", "public");
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("min-integrity already configured as 'approved'"));
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("repos already configured as 'public'"));
  });

  it("should log appropriate info messages for public repo", async () => {
    mockGithub.rest.repos.get.mockResolvedValue({
      data: {
        private: false,
        visibility: "public",
      },
    });

    await determineAutomaticLockdown(mockGithub, mockContext, mockCore);

    expect(mockCore.info).toHaveBeenCalledWith("Determining automatic guard policy for GitHub MCP server");
    expect(mockCore.info).toHaveBeenCalledWith("Checking repository: test-owner/test-repo");
    expect(mockCore.info).toHaveBeenCalledWith("Repository visibility: public");
    expect(mockCore.info).toHaveBeenCalledWith("Repository is private: false");
    expect(mockCore.info).toHaveBeenCalledWith("Automatic guard policy determination complete for public repository");
  });

  it("should write resolved guard policy values to step summary for public repository", async () => {
    mockGithub.rest.repos.get.mockResolvedValue({
      data: {
        private: false,
        visibility: "public",
      },
    });

    await determineAutomaticLockdown(mockGithub, mockContext, mockCore);

    expect(mockCore.summary.addRaw).toHaveBeenCalledTimes(1);
    const publicSummaryArg = mockCore.summary.addRaw.mock.calls[0][0];
    expect(publicSummaryArg).toContain("<details>");
    expect(publicSummaryArg).toContain("GitHub MCP Guard Policy");
    // Ensure we have a well-formed <details> block with a <summary> and closing </details>
    expect(publicSummaryArg).toMatch(/<details>[\s\S]*<\/details>/);
    expect(publicSummaryArg).toMatch(/<summary>[\s\S]*GitHub MCP Guard Policy[\s\S]*<\/summary>/);
    // Ensure the markdown table header and separator are present
    expect(publicSummaryArg).toMatch(/\| *Field *\| *Value *\| *Source *\|/);
    expect(publicSummaryArg).toMatch(/\|[- ]+\|[- ]+\|[- ]+\|/);
    expect(publicSummaryArg).toContain("min-integrity");
    expect(publicSummaryArg).toContain("approved");
    expect(publicSummaryArg).toContain("automatic (public repo)");
    expect(publicSummaryArg).toContain("repos");
    expect(publicSummaryArg).toContain("all");
    expect(mockCore.summary.write).toHaveBeenCalled();
  });

  it("should show workflow config as source in summary when guard policy fields are pre-configured", async () => {
    process.env.GH_AW_GITHUB_MIN_INTEGRITY = "merged";
    process.env.GH_AW_GITHUB_REPOS = "public";

    mockGithub.rest.repos.get.mockResolvedValue({
      data: {
        private: false,
        visibility: "public",
      },
    });

    await determineAutomaticLockdown(mockGithub, mockContext, mockCore);

    expect(mockCore.summary.addRaw).toHaveBeenCalledTimes(1);
    const configuredSummaryArg = mockCore.summary.addRaw.mock.calls[0][0];
    // Ensure we have a well-formed <details> block with closing </details>
    expect(configuredSummaryArg).toMatch(/<details>[\s\S]*<\/details>/);
    // Ensure the markdown table header and separator are present
    expect(configuredSummaryArg).toMatch(/\| *Field *\| *Value *\| *Source *\|/);
    expect(configuredSummaryArg).toMatch(/\|[- ]+\|[- ]+\|[- ]+\|/);
    expect(configuredSummaryArg).toContain("min-integrity");
    expect(configuredSummaryArg).toContain("merged");
    expect(configuredSummaryArg).toContain("workflow config");
    expect(configuredSummaryArg).toContain("repos");
    expect(configuredSummaryArg).toContain("public");
    expect(mockCore.summary.write).toHaveBeenCalled();
  });

  it("should write resolved guard policy values to step summary for private repository", async () => {
    mockGithub.rest.repos.get.mockResolvedValue({
      data: {
        private: true,
        visibility: "private",
      },
    });

    await determineAutomaticLockdown(mockGithub, mockContext, mockCore);

    expect(mockCore.summary.addRaw).toHaveBeenCalledTimes(1);
    const privateSummaryArg = mockCore.summary.addRaw.mock.calls[0][0];
    // Ensure we have a well-formed <details> block with closing </details>
    expect(privateSummaryArg).toMatch(/<details>[\s\S]*<\/details>/);
    // Ensure the markdown table header and separator are present
    expect(privateSummaryArg).toMatch(/\| *Field *\| *Value *\| *Source *\|/);
    expect(privateSummaryArg).toMatch(/\|[- ]+\|[- ]+\|[- ]+\|/);
    expect(privateSummaryArg).toContain("min-integrity");
    expect(privateSummaryArg).toContain("none");
    expect(privateSummaryArg).toContain("automatic (private repo)");
    expect(privateSummaryArg).toContain("repos");
    expect(privateSummaryArg).toContain("all");
    expect(mockCore.summary.write).toHaveBeenCalled();
  });
});
