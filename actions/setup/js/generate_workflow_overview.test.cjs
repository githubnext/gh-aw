import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";

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
  let awInfoPath;

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
      version: "v1.2.3",
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
    // engine_id and version should appear in the summary label
    expect(summaryArg).toContain("<summary>Run details - copilot v1.2.3</summary>");
    // All fields should be rendered as bullet points with humanified keys
    expect(summaryArg).toContain("- **engine id**: copilot");
    expect(summaryArg).toContain("- **engine name**: GitHub Copilot");
    expect(summaryArg).toContain("- **model**: gpt-4");
    expect(summaryArg).toContain("- **version**: v1.2.3");
    expect(summaryArg).toContain("- **firewall enabled**: true");
    expect(summaryArg).toContain("- **awf version**: 1.0.0");
    expect(summaryArg).toContain("</details>");
    // Ensure no table syntax is present
    expect(summaryArg).not.toContain("| Property | Value |");
    expect(summaryArg).not.toContain("|----------|-------|");
  });

  it("should show only engine_id in summary label when version is missing", async () => {
    const awInfo = {
      engine_id: "claude",
      engine_name: "Claude",
      firewall_enabled: false,
    };
    fs.writeFileSync(awInfoPath, JSON.stringify(awInfo));

    await generateWorkflowOverview(mockCore);

    const summaryArg = mockCore.summary.addRaw.mock.calls[0][0];
    expect(summaryArg).toContain("<summary>Run details - claude</summary>");
    expect(summaryArg).toContain("- **engine id**: claude");
    expect(summaryArg).toContain("- **firewall enabled**: false");
  });

  it("should show plain 'Run details' in summary label when both engine_id and version are missing", async () => {
    const awInfo = {
      engine_name: "Unknown Engine",
    };
    fs.writeFileSync(awInfoPath, JSON.stringify(awInfo));

    await generateWorkflowOverview(mockCore);

    const summaryArg = mockCore.summary.addRaw.mock.calls[0][0];
    expect(summaryArg).toContain("<summary>Run details</summary>");
  });

  it("should render all fields from aw_info including nested objects and arrays", async () => {
    const awInfo = {
      engine_id: "copilot",
      version: "v2.0.0",
      allowed_domains: ["example.com", "github.com"],
      steps: { firewall: "iptables" },
    };
    fs.writeFileSync(awInfoPath, JSON.stringify(awInfo));

    await generateWorkflowOverview(mockCore);

    const summaryArg = mockCore.summary.addRaw.mock.calls[0][0];
    expect(summaryArg).toContain("- **engine id**: copilot");
    expect(summaryArg).toContain("- **allowed domains**:");
    expect(summaryArg).toContain("  - example.com");
    expect(summaryArg).toContain("  - github.com");
    expect(summaryArg).toContain("- **steps**:");
    expect(summaryArg).toContain("  - **firewall**: iptables");
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
