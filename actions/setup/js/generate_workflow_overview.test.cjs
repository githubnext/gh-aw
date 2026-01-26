import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import os from "os";
import path from "path";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

// Set up global mocks before importing the module
global.core = mockCore;

describe("generate_workflow_overview.cjs", () => {
  let generateWorkflowOverview;
  let tmpDir;
  let awInfoPath;
  let originalRequireCache;

  beforeEach(async () => {
    // Reset mocks
    vi.clearAllMocks();

    // Create /tmp/gh-aw directory if it doesn't exist
    if (!fs.existsSync("/tmp/gh-aw")) {
      fs.mkdirSync("/tmp/gh-aw", { recursive: true });
    }
    awInfoPath = "/tmp/gh-aw/aw_info.json";

    // Dynamic import to get fresh module state
    const module = await import("./generate_workflow_overview.cjs");
    generateWorkflowOverview = module.generateWorkflowOverview;
  });

  afterEach(() => {
    // Clean up test file
    if (fs.existsSync(awInfoPath)) {
      fs.unlinkSync(awInfoPath);
    }
  });

  it("should generate workflow overview with basic engine info", async () => {
    // Create test aw_info.json
    const awInfo = {
      engine_id: "copilot",
      engine_name: "GitHub Copilot",
      model: "gpt-4",
      firewall_enabled: true,
      awf_version: "1.0.0",
      allowed_domains: [],
    };
    fs.writeFileSync(awInfoPath, JSON.stringify(awInfo));

    await generateWorkflowOverview(mockCore);

    expect(mockCore.summary.addRaw).toHaveBeenCalledTimes(1);
    expect(mockCore.summary.write).toHaveBeenCalledTimes(1);

    const summaryArg = mockCore.summary.addRaw.mock.calls[0][0];
    expect(summaryArg).toContain("<details>");
    expect(summaryArg).toContain("<summary>Run details</summary>");
    expect(summaryArg).toContain("#### Engine Configuration");
    expect(summaryArg).toContain("| Engine ID | copilot |");
    expect(summaryArg).toContain("| Engine Name | GitHub Copilot |");
    expect(summaryArg).toContain("| Model | gpt-4 |");
    expect(summaryArg).toContain("#### Network Configuration");
    expect(summaryArg).toContain("| Firewall | ✅ Enabled |");
    expect(summaryArg).toContain("| Firewall Version | 1.0.0 |");
    expect(summaryArg).toContain("</details>");
  });

  it("should handle missing optional fields with defaults", async () => {
    // Create test aw_info.json with minimal fields
    const awInfo = {
      engine_id: "claude",
      engine_name: "Claude",
      firewall_enabled: false,
    };
    fs.writeFileSync(awInfoPath, JSON.stringify(awInfo));

    await generateWorkflowOverview(mockCore);

    const summaryArg = mockCore.summary.addRaw.mock.calls[0][0];
    expect(summaryArg).toContain("| Model | (default) |");
    expect(summaryArg).toContain("| Firewall | ❌ Disabled |");
    expect(summaryArg).toContain("| Firewall Version | (latest) |");
  });

  it("should include allowed domains when present (up to 10)", async () => {
    const awInfo = {
      engine_id: "copilot",
      engine_name: "GitHub Copilot",
      firewall_enabled: true,
      allowed_domains: ["example.com", "github.com", "api.github.com"],
    };
    fs.writeFileSync(awInfoPath, JSON.stringify(awInfo));

    await generateWorkflowOverview(mockCore);

    const summaryArg = mockCore.summary.addRaw.mock.calls[0][0];
    expect(summaryArg).toContain("##### Allowed Domains");
    expect(summaryArg).toContain("  - example.com");
    expect(summaryArg).toContain("  - github.com");
    expect(summaryArg).toContain("  - api.github.com");
  });

  it("should truncate allowed domains list when more than 10", async () => {
    const domains = Array.from({ length: 15 }, (_, i) => `domain${i + 1}.com`);
    const awInfo = {
      engine_id: "copilot",
      engine_name: "GitHub Copilot",
      firewall_enabled: true,
      allowed_domains: domains,
    };
    fs.writeFileSync(awInfoPath, JSON.stringify(awInfo));

    await generateWorkflowOverview(mockCore);

    const summaryArg = mockCore.summary.addRaw.mock.calls[0][0];
    expect(summaryArg).toContain("##### Allowed Domains");
    expect(summaryArg).toContain("  - domain1.com");
    expect(summaryArg).toContain("  - domain10.com");
    expect(summaryArg).toContain("  - ... and 5 more");
    expect(summaryArg).not.toContain("domain11.com");
  });

  it("should not include Allowed Domains section when empty", async () => {
    const awInfo = {
      engine_id: "copilot",
      engine_name: "GitHub Copilot",
      firewall_enabled: false,
      allowed_domains: [],
    };
    fs.writeFileSync(awInfoPath, JSON.stringify(awInfo));

    await generateWorkflowOverview(mockCore);

    const summaryArg = mockCore.summary.addRaw.mock.calls[0][0];
    expect(summaryArg).not.toContain("##### Allowed Domains");
  });

  it("should log success message", async () => {
    const awInfo = {
      engine_id: "copilot",
      engine_name: "GitHub Copilot",
      firewall_enabled: true,
    };
    fs.writeFileSync(awInfoPath, JSON.stringify(awInfo));

    // Capture console.log
    const consoleSpy = vi.spyOn(console, "log");

    await generateWorkflowOverview(mockCore);

    expect(consoleSpy).toHaveBeenCalledWith("Generated workflow overview in step summary");

    consoleSpy.mockRestore();
  });
});
