import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  notice: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  exportVariable: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

// Set up global variables
global.core = mockCore;

describe("analyze_security_trends.cjs", () => {
  let analyzeTrendsScript;
  let tempDir;
  let tempCacheDir;
  let tempAgentOutputPath;

  beforeEach(() => {
    // Reset all mocks
    vi.clearAllMocks();

    // Create temp directories
    tempDir = path.join("/tmp", `test_trends_${Date.now()}_${Math.random().toString(36).slice(2)}`);
    tempCacheDir = path.join(tempDir, "cache-memory");
    fs.mkdirSync(tempCacheDir, { recursive: true });

    // Set environment variable
    process.env.GH_AW_CACHE_DIR = tempCacheDir;

    // Read the script
    const scriptPath = path.join(process.cwd(), "analyze_security_trends.cjs");
    analyzeTrendsScript = fs.readFileSync(scriptPath, "utf8");
  });

  afterEach(() => {
    // Clean up temp files
    if (tempAgentOutputPath && fs.existsSync(tempAgentOutputPath)) {
      fs.unlinkSync(tempAgentOutputPath);
    }
    if (fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }

    // Clear environment variables
    delete process.env.GH_AW_CACHE_DIR;
    delete process.env.GH_AW_AGENT_OUTPUT;
  });

  // Helper function to create agent output file
  const createAgentOutput = data => {
    tempAgentOutputPath = path.join(tempDir, `agent_output_${Date.now()}.json`);
    // Wrap in items array as expected by loadAgentOutput
    const wrappedData = Array.isArray(data) ? { items: data } : data;
    fs.writeFileSync(tempAgentOutputPath, JSON.stringify(wrappedData));
    process.env.GH_AW_AGENT_OUTPUT = tempAgentOutputPath;
  };

  // Helper function to create historical scan data
  const createHistoricalScan = (date, totalFindings) => {
    const scansDir = path.join(tempCacheDir, "security-scans");
    if (!fs.existsSync(scansDir)) {
      fs.mkdirSync(scansDir, { recursive: true });
    }

    const scanData = {
      date,
      timestamp: new Date(date).toISOString(),
      total: totalFindings,
      byTool: {
        zizmor: {
          total: Math.floor(totalFindings * 0.4),
          bySeverity: { critical: 0, high: 0, medium: 0, low: Math.floor(totalFindings * 0.4) },
        },
        poutine: {
          total: Math.floor(totalFindings * 0.3),
          bySeverity: { critical: 0, high: 0, medium: 0, low: Math.floor(totalFindings * 0.3) },
        },
        actionlint: { total: Math.floor(totalFindings * 0.3) },
      },
    };

    const scanPath = path.join(scansDir, `${date}.json`);
    fs.writeFileSync(scanPath, JSON.stringify(scanData, null, 2));

    // Update index
    const indexPath = path.join(scansDir, "index.json");
    let index = { scans: [] };
    if (fs.existsSync(indexPath)) {
      index = JSON.parse(fs.readFileSync(indexPath, "utf8"));
    }
    index.scans.push({ date, total: totalFindings });
    fs.writeFileSync(indexPath, JSON.stringify(index, null, 2));

    return scanData;
  };

  it("should handle no agent output gracefully", async () => {
    process.env.GH_AW_AGENT_OUTPUT = "/nonexistent/file.json";

    const scriptFunc = new Function("require", "process", "core", analyzeTrendsScript);
    await scriptFunc(require, process, mockCore);

    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("No agent output found"));
    expect(mockCore.setOutput).toHaveBeenCalledWith("trend_analysis", "");
  });

  it("should parse scan results from discussion body", async () => {
    const agentOutput = [
      {
        type: "create_discussion",
        title: "Static Analysis Report",
        body: `# Static Analysis Report

| Tool | Total | Critical | High | Medium | Low |
|------|-------|----------|------|--------|-----|
| zizmor (security) | 50 | 0 | 0 | 5 | 45 |
| poutine (supply chain) | 30 | 0 | 0 | 0 | 30 |
| actionlint (linting) | 20 | - | - | - | - |
`,
      },
    ];

    createAgentOutput(agentOutput);

    const scriptFunc = new Function("require", "process", "core", analyzeTrendsScript);
    await scriptFunc(require, process, mockCore);

    // Check that scanning and processing occurred
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Analyzing trends from 1 agent output item(s)"));
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("total findings"));
    expect(mockCore.setOutput).toHaveBeenCalledWith("trend_analysis", expect.any(String));
  });

  it("should create initial scan when no historical data exists", async () => {
    const agentOutput = [
      {
        type: "create_discussion",
        title: "Static Analysis Report",
        body: `| Tool | Total | Critical | High | Medium | Low |
|------|-------|----------|------|--------|-----|
| zizmor (security) | 50 | 0 | 0 | 5 | 45 |
| poutine (supply chain) | 30 | 0 | 0 | 0 | 30 |
| actionlint (linting) | 20 | - | - | - | - |`,
      },
    ];

    createAgentOutput(agentOutput);

    const scriptFunc = new Function("require", "process", "core", analyzeTrendsScript);
    await scriptFunc(require, process, mockCore);

    const today = new Date().toISOString().split("T")[0];
    const scanPath = path.join(tempCacheDir, "security-scans", `${today}.json`);

    expect(fs.existsSync(scanPath)).toBe(true);

    const trendAnalysis = mockCore.setOutput.mock.calls.find(call => call[0] === "trend_analysis");
    expect(trendAnalysis[1]).toContain("No historical data available");
  });

  it("should compare with previous scan and detect improvement", async () => {
    // Create historical scan with more findings
    const yesterday = new Date();
    yesterday.setDate(yesterday.getDate() - 1);
    const yesterdayStr = yesterday.toISOString().split("T")[0];
    createHistoricalScan(yesterdayStr, 150);

    // Create current scan with fewer findings
    const agentOutput = [
      {
        type: "create_discussion",
        title: "Static Analysis Report",
        body: `| Tool | Total | Critical | High | Medium | Low |
|------|-------|----------|------|--------|-----|
| zizmor (security) | 40 | 0 | 0 | 5 | 35 |
| poutine (supply chain) | 30 | 0 | 0 | 0 | 30 |
| actionlint (linting) | 30 | - | - | - | - |`,
      },
    ];

    createAgentOutput(agentOutput);

    const scriptFunc = new Function("require", "process", "core", analyzeTrendsScript);
    await scriptFunc(require, process, mockCore);

    const trendAnalysis = mockCore.setOutput.mock.calls.find(call => call[0] === "trend_analysis");
    expect(trendAnalysis[1]).toContain("improvement");
    expect(trendAnalysis[1]).toContain("Resolved");
    expect(trendAnalysis[1]).toContain("↓");
  });

  it("should compare with previous scan and detect regression", async () => {
    // Create historical scan with fewer findings
    const yesterday = new Date();
    yesterday.setDate(yesterday.getDate() - 1);
    const yesterdayStr = yesterday.toISOString().split("T")[0];
    createHistoricalScan(yesterdayStr, 50);

    // Create current scan with MORE findings (regression)
    const agentOutput = [
      {
        type: "create_discussion",
        title: "Static Analysis Report",
        body: `| Tool | Total | Critical | High | Medium | Low |
|------|-------|----------|------|--------|-----|
| zizmor (security) | 60 | 0 | 0 | 10 | 50 |
| poutine (supply chain) | 40 | 0 | 0 | 0 | 40 |
| actionlint (linting) | 20 | - | - | - | - |`,
      },
    ];

    createAgentOutput(agentOutput);

    const scriptFunc = new Function("require", "process", "core", analyzeTrendsScript);
    await scriptFunc(require, process, mockCore);

    const trendAnalysis = mockCore.setOutput.mock.calls.find(call => call[0] === "trend_analysis");
    // Since we have more findings now (120) than before (50), it's a regression
    expect(trendAnalysis[1]).toContain("regression");
    expect(trendAnalysis[1]).toContain("New Issues");
    expect(trendAnalysis[1]).toContain("↑");
  });

  it("should calculate weekly and monthly aggregates", async () => {
    // Create multiple historical scans
    for (let i = 1; i <= 30; i++) {
      const date = new Date();
      date.setDate(date.getDate() - i);
      const dateStr = date.toISOString().split("T")[0];
      createHistoricalScan(dateStr, 100 - i);
    }

    // Create current scan
    const agentOutput = [
      {
        type: "create_discussion",
        body: `| Tool | Total | Critical | High | Medium | Low |
|------|-------|----------|------|--------|-----|
| zizmor (security) | 30 | 0 | 0 | 0 | 30 |
| poutine (supply chain) | 20 | 0 | 0 | 0 | 20 |
| actionlint (linting) | 20 | - | - | - | - |`,
      },
    ];

    createAgentOutput(agentOutput);

    const scriptFunc = new Function("require", "process", "core", analyzeTrendsScript);
    await scriptFunc(require, process, mockCore);

    // Check that aggregates were created
    const weeklyPath = path.join(tempCacheDir, "security-scans", "trends", "weekly.json");
    const monthlyPath = path.join(tempCacheDir, "security-scans", "trends", "monthly.json");

    expect(fs.existsSync(weeklyPath)).toBe(true);
    expect(fs.existsSync(monthlyPath)).toBe(true);

    const weeklyData = JSON.parse(fs.readFileSync(weeklyPath, "utf8"));
    const monthlyData = JSON.parse(fs.readFileSync(monthlyPath, "utf8"));

    expect(Object.keys(weeklyData).length).toBeGreaterThan(0);
    expect(Object.keys(monthlyData).length).toBeGreaterThan(0);
  });

  it("should maintain scan index with max 90 days", async () => {
    // Create 100 historical scans (more than 90)
    for (let i = 1; i <= 100; i++) {
      const date = new Date();
      date.setDate(date.getDate() - i);
      const dateStr = date.toISOString().split("T")[0];
      createHistoricalScan(dateStr, 100);
    }

    // Create current scan
    const agentOutput = [
      {
        type: "create_discussion",
        body: `| Tool | Total | Critical | High | Medium | Low |
|------|-------|----------|------|--------|-----|
| zizmor (security) | 30 | 0 | 0 | 0 | 30 |
| poutine (supply chain) | 20 | 0 | 0 | 0 | 20 |
| actionlint (linting) | 20 | - | - | - | - |`,
      },
    ];

    createAgentOutput(agentOutput);

    const scriptFunc = new Function("require", "process", "core", analyzeTrendsScript);
    await scriptFunc(require, process, mockCore);

    // Check that index only has 90 scans
    const indexPath = path.join(tempCacheDir, "security-scans", "index.json");
    const index = JSON.parse(fs.readFileSync(indexPath, "utf8"));

    expect(index.scans.length).toBeLessThanOrEqual(90);
  });

  it("should handle stable trend (no change)", async () => {
    // Create historical scan with same total findings as current
    const yesterday = new Date();
    yesterday.setDate(yesterday.getDate() - 1);
    const yesterdayStr = yesterday.toISOString().split("T")[0];
    createHistoricalScan(yesterdayStr, 60); // Match the total we'll create

    // Create current scan with same total findings (60)
    const agentOutput = [
      {
        type: "create_discussion",
        body: `| Tool | Total | Critical | High | Medium | Low |
|------|-------|----------|------|--------|-----|
| zizmor (security) | 20 | 0 | 0 | 0 | 20 |
| poutine (supply chain) | 20 | 0 | 0 | 0 | 20 |
| actionlint (linting) | 20 | - | - | - | - |`,
      },
    ];

    createAgentOutput(agentOutput);

    const scriptFunc = new Function("require", "process", "core", analyzeTrendsScript);
    await scriptFunc(require, process, mockCore);

    const trendAnalysis = mockCore.setOutput.mock.calls.find(call => call[0] === "trend_analysis");
    expect(trendAnalysis[1]).toContain("→");
    expect(trendAnalysis[1]).toContain("No change");
  });

  it("should format trend analysis with correct markdown", async () => {
    const yesterday = new Date();
    yesterday.setDate(yesterday.getDate() - 1);
    const yesterdayStr = yesterday.toISOString().split("T")[0];
    createHistoricalScan(yesterdayStr, 150);

    const agentOutput = [
      {
        type: "create_discussion",
        body: `| Tool | Total | Critical | High | Medium | Low |
|------|-------|----------|------|--------|-----|
| zizmor (security) | 40 | 0 | 0 | 5 | 35 |
| poutine (supply chain) | 30 | 0 | 0 | 0 | 30 |
| actionlint (linting) | 30 | - | - | - | - |`,
      },
    ];

    createAgentOutput(agentOutput);

    const scriptFunc = new Function("require", "process", "core", analyzeTrendsScript);
    await scriptFunc(require, process, mockCore);

    const trendAnalysis = mockCore.setOutput.mock.calls.find(call => call[0] === "trend_analysis");
    const analysis = trendAnalysis[1];

    // Check for required markdown elements
    expect(analysis).toContain("## 📈 Trend Analysis");
    expect(analysis).toContain("**Week-over-Week Change**");
    expect(analysis).toContain("| Metric | Current | Previous | Change |");
    expect(analysis).toContain("Total Findings");
  });

  it("should add summary to step summary", async () => {
    const agentOutput = [
      {
        type: "create_discussion",
        body: `| Tool | Total | Critical | High | Medium | Low |
|------|-------|----------|------|--------|-----|
| zizmor (security) | 30 | 0 | 0 | 0 | 30 |
| poutine (supply chain) | 20 | 0 | 0 | 0 | 20 |
| actionlint (linting) | 20 | - | - | - | - |`,
      },
    ];

    createAgentOutput(agentOutput);

    const scriptFunc = new Function("require", "process", "core", analyzeTrendsScript);
    await scriptFunc(require, process, mockCore);

    expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("Trend Analysis Complete"));
    expect(mockCore.summary.write).toHaveBeenCalled();
  });

  it("should handle missing tool data gracefully", async () => {
    const agentOutput = [
      {
        type: "create_discussion",
        body: "Some discussion without proper tables",
      },
    ];

    createAgentOutput(agentOutput);

    const scriptFunc = new Function("require", "process", "core", analyzeTrendsScript);
    await scriptFunc(require, process, mockCore);

    // Should not crash, should handle gracefully
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("0 total findings"));
  });
});
