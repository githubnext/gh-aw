import { describe, it, expect, beforeEach, vi } from "vitest";

describe("validate_lockdown_requirements", () => {
  let mockCore;
  let validateLockdownRequirements;

  beforeEach(async () => {
    vi.resetModules();

    // Setup mock core
    mockCore = {
      info: vi.fn(),
      setFailed: vi.fn(),
      setOutput: vi.fn(),
    };

    // Reset process.env
    delete process.env.GITHUB_MCP_LOCKDOWN_EXPLICIT;
    delete process.env.GH_AW_GITHUB_TOKEN;
    delete process.env.GH_AW_GITHUB_MCP_SERVER_TOKEN;
    delete process.env.CUSTOM_GITHUB_TOKEN;
    delete process.env.GITHUB_REPOSITORY_VISIBILITY;
    delete process.env.GH_AW_COMPILED_STRICT;

    // Import the module
    validateLockdownRequirements = (await import("./validate_lockdown_requirements.cjs")).default;
  });

  it("should skip lockdown validation when lockdown is not explicitly enabled", () => {
    // GITHUB_MCP_LOCKDOWN_EXPLICIT not set

    validateLockdownRequirements(mockCore);

    expect(mockCore.info).toHaveBeenCalledWith("Lockdown mode not explicitly enabled, skipping validation");
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });

  it("should pass validation when lockdown is enabled and GH_AW_GITHUB_TOKEN is configured", () => {
    process.env.GITHUB_MCP_LOCKDOWN_EXPLICIT = "true";
    process.env.GH_AW_GITHUB_TOKEN = "ghp_test_token";

    validateLockdownRequirements(mockCore);

    expect(mockCore.info).toHaveBeenCalledWith("Lockdown mode is explicitly enabled, validating requirements...");
    expect(mockCore.info).toHaveBeenCalledWith("GH_AW_GITHUB_TOKEN configured: true");
    expect(mockCore.info).toHaveBeenCalledWith("✓ Lockdown mode requirements validated: Custom GitHub token is configured");
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });

  it("should pass validation when lockdown is enabled and GH_AW_GITHUB_MCP_SERVER_TOKEN is configured", () => {
    process.env.GITHUB_MCP_LOCKDOWN_EXPLICIT = "true";
    process.env.GH_AW_GITHUB_MCP_SERVER_TOKEN = "ghp_mcp_token";

    validateLockdownRequirements(mockCore);

    expect(mockCore.info).toHaveBeenCalledWith("Lockdown mode is explicitly enabled, validating requirements...");
    expect(mockCore.info).toHaveBeenCalledWith("GH_AW_GITHUB_MCP_SERVER_TOKEN configured: true");
    expect(mockCore.info).toHaveBeenCalledWith("✓ Lockdown mode requirements validated: Custom GitHub token is configured");
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });

  it("should pass validation when lockdown is enabled and custom github-token is configured", () => {
    process.env.GITHUB_MCP_LOCKDOWN_EXPLICIT = "true";
    process.env.CUSTOM_GITHUB_TOKEN = "ghp_custom_token";

    validateLockdownRequirements(mockCore);

    expect(mockCore.info).toHaveBeenCalledWith("Lockdown mode is explicitly enabled, validating requirements...");
    expect(mockCore.info).toHaveBeenCalledWith("Custom github-token configured: true");
    expect(mockCore.info).toHaveBeenCalledWith("✓ Lockdown mode requirements validated: Custom GitHub token is configured");
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });

  it("should fail when lockdown is enabled but no custom tokens are configured", () => {
    process.env.GITHUB_MCP_LOCKDOWN_EXPLICIT = "true";
    // No custom tokens set

    expect(() => {
      validateLockdownRequirements(mockCore);
    }).toThrow("Lockdown mode is enabled");

    expect(mockCore.info).toHaveBeenCalledWith("Lockdown mode is explicitly enabled, validating requirements...");
    expect(mockCore.info).toHaveBeenCalledWith("GH_AW_GITHUB_TOKEN configured: false");
    expect(mockCore.info).toHaveBeenCalledWith("GH_AW_GITHUB_MCP_SERVER_TOKEN configured: false");
    expect(mockCore.info).toHaveBeenCalledWith("Custom github-token configured: false");
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Lockdown mode is enabled (lockdown: true) but no custom GitHub token is configured"));
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("GH_AW_GITHUB_TOKEN (recommended)"));
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("GH_AW_GITHUB_MCP_SERVER_TOKEN (alternative)"));
    expect(mockCore.setOutput).toHaveBeenCalledWith("lockdown_check_failed", "true");
  });

  it("should include documentation link in error message", () => {
    process.env.GITHUB_MCP_LOCKDOWN_EXPLICIT = "true";
    // No custom tokens set

    expect(() => {
      validateLockdownRequirements(mockCore);
    }).toThrow();

    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("https://github.com/github/gh-aw/blob/main/docs/src/content/docs/reference/auth.mdx"));
  });

  it("should handle empty string tokens as not configured", () => {
    process.env.GITHUB_MCP_LOCKDOWN_EXPLICIT = "true";
    process.env.GH_AW_GITHUB_TOKEN = "";
    process.env.GH_AW_GITHUB_MCP_SERVER_TOKEN = "";
    process.env.CUSTOM_GITHUB_TOKEN = "";

    expect(() => {
      validateLockdownRequirements(mockCore);
    }).toThrow("Lockdown mode is enabled");

    expect(mockCore.setFailed).toHaveBeenCalled();
  });

  it("should skip lockdown validation when GITHUB_MCP_LOCKDOWN_EXPLICIT is false", () => {
    process.env.GITHUB_MCP_LOCKDOWN_EXPLICIT = "false";
    // GH_AW_GITHUB_TOKEN not set

    validateLockdownRequirements(mockCore);

    expect(mockCore.info).toHaveBeenCalledWith("Lockdown mode not explicitly enabled, skipping validation");
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });

  // Strict mode enforcement for public repositories
  describe("strict mode enforcement for public repositories", () => {
    it("should fail when repository is public and not compiled with strict mode", () => {
      process.env.GITHUB_REPOSITORY_VISIBILITY = "public";
      process.env.GH_AW_COMPILED_STRICT = "false";

      expect(() => {
        validateLockdownRequirements(mockCore);
      }).toThrow("not compiled with strict mode");

      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("public repository but was not compiled with strict mode"));
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("gh aw compile --strict"));
      expect(mockCore.setOutput).toHaveBeenCalledWith("lockdown_check_failed", "true");
    });

    it("should fail when repository is public and GH_AW_COMPILED_STRICT is not set", () => {
      process.env.GITHUB_REPOSITORY_VISIBILITY = "public";
      // GH_AW_COMPILED_STRICT not set

      expect(() => {
        validateLockdownRequirements(mockCore);
      }).toThrow("not compiled with strict mode");

      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("public repository but was not compiled with strict mode"));
      expect(mockCore.setOutput).toHaveBeenCalledWith("lockdown_check_failed", "true");
    });

    it("should pass when repository is public and compiled with strict mode", () => {
      process.env.GITHUB_REPOSITORY_VISIBILITY = "public";
      process.env.GH_AW_COMPILED_STRICT = "true";

      validateLockdownRequirements(mockCore);

      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(mockCore.info).toHaveBeenCalledWith("✓ Strict mode requirements validated: Public repository compiled with strict mode");
    });

    it("should pass when repository is private and not compiled with strict mode", () => {
      process.env.GITHUB_REPOSITORY_VISIBILITY = "private";
      process.env.GH_AW_COMPILED_STRICT = "false";

      validateLockdownRequirements(mockCore);

      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should pass when repository is internal and not compiled with strict mode", () => {
      process.env.GITHUB_REPOSITORY_VISIBILITY = "internal";
      process.env.GH_AW_COMPILED_STRICT = "false";

      validateLockdownRequirements(mockCore);

      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should pass when visibility is unknown and not compiled with strict mode", () => {
      // GITHUB_REPOSITORY_VISIBILITY not set
      process.env.GH_AW_COMPILED_STRICT = "false";

      validateLockdownRequirements(mockCore);

      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should include documentation link in strict mode error message", () => {
      process.env.GITHUB_REPOSITORY_VISIBILITY = "public";
      process.env.GH_AW_COMPILED_STRICT = "false";

      expect(() => {
        validateLockdownRequirements(mockCore);
      }).toThrow();

      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("https://github.com/github/gh-aw/blob/main/docs/src/content/docs/reference/security.mdx"));
    });

    it("should validate both lockdown and strict mode when both are required", () => {
      process.env.GITHUB_MCP_LOCKDOWN_EXPLICIT = "true";
      process.env.GH_AW_GITHUB_TOKEN = "ghp_test_token";
      process.env.GITHUB_REPOSITORY_VISIBILITY = "public";
      process.env.GH_AW_COMPILED_STRICT = "true";

      validateLockdownRequirements(mockCore);

      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(mockCore.info).toHaveBeenCalledWith("✓ Lockdown mode requirements validated: Custom GitHub token is configured");
      expect(mockCore.info).toHaveBeenCalledWith("✓ Strict mode requirements validated: Public repository compiled with strict mode");
    });

    it("should fail on lockdown check before strict mode check when both fail", () => {
      process.env.GITHUB_MCP_LOCKDOWN_EXPLICIT = "true";
      // No custom tokens - will fail on lockdown check
      process.env.GITHUB_REPOSITORY_VISIBILITY = "public";
      process.env.GH_AW_COMPILED_STRICT = "false";

      expect(() => {
        validateLockdownRequirements(mockCore);
      }).toThrow("Lockdown mode is enabled");

      // Strict mode error should not be reached since lockdown check throws first
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Lockdown mode is enabled"));
    });
  });
});
