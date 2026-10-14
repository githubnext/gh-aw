import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import fs from "fs";
import os from "os";
import path from "path";
import { createRequire } from "module";

const req = createRequire(import.meta.url);
const { main } = req("./generate_usage_artifacts.cjs");
const { main: generateUnifiedSession } = req("./unified_session.cjs");
const { generateCopilotCliStyleSummary } = req("./log_parser_shared.cjs");
const diskUsage = req("./disk_usage_logger.cjs");

describe("generate_usage_artifacts.cjs", () => {
  let tempDirectory;
  let previousEnv;
  let previousCore;

  beforeEach(() => {
    previousEnv = { ...process.env };
    previousCore = global.core;
    vi.spyOn(diskUsage, "collectDiskUsageArtifact").mockImplementation(() => undefined);
  });

  afterEach(() => {
    process.env = previousEnv;
    global.core = previousCore;
    vi.restoreAllMocks();
    if (tempDirectory) {
      fs.rmSync(tempDirectory, { recursive: true, force: true });
      tempDirectory = undefined;
    }
  });

  it("renders the persisted unified session after the preceding generator finishes assembling its inputs", async () => {
    tempDirectory = fs.mkdtempSync(path.join(os.tmpdir(), "usage-artifacts-"));
    const usageDirectory = path.join(tempDirectory, "usage");
    const sessionPath = path.join(usageDirectory, "aw_session.jsonl");
    const summaryPath = path.join(tempDirectory, "step-summary.md");
    process.env.GITHUB_STEP_SUMMARY = summaryPath;
    process.env.GH_AW_SECRET_NAMES = "ASSEMBLY_TEST";
    process.env.SECRET_ASSEMBLY_TEST = "assembly-secret-fixture";
    fs.writeFileSync(path.join(tempDirectory, "agent-session.jsonl"), JSON.stringify({ type: "assistant.message", data: { content: "Finished assembly-secret-fixture" } }) + "\n");

    const core = {
      info: vi.fn(message => {
        if (!message.startsWith("=== Unified session ===")) return;
        const persisted = fs.readFileSync(sessionPath, "utf8");
        expect(persisted).not.toContain("assembly-secret-fixture");
        const events = persisted.trim().split("\n").map(JSON.parse);
        expect(events[0].type).toBe("session.format");
        expect(events.some(event => event.type === "detection.result")).toBe(true);
        expect(events.some(event => event.type === "session.collection")).toBe(true);
        expect(fs.existsSync(`${sessionPath}.tmp`)).toBe(false);
      }),
      warning: vi.fn(message => {
        throw new Error(message);
      }),
    };
    global.core = core;
    await main({
      core,
      usageDirectory,
      generateSummary: async () => {
        expect(diskUsage.collectDiskUsageArtifact).toHaveBeenCalledOnce();
        await Promise.resolve();
        fs.mkdirSync(path.join(usageDirectory, "detection"), { recursive: true });
        fs.writeFileSync(path.join(usageDirectory, "detection", "detection_result.json"), JSON.stringify({ job_result: "success", conclusion: "success" }));
      },
      generateSession: () => generateUnifiedSession({ rootDir: tempDirectory }),
    });

    const events = fs.readFileSync(sessionPath, "utf8").trim().split("\n").map(JSON.parse);
    const summary = fs.readFileSync(summaryPath, "utf8");
    expect(summary).toBe(`\n${generateCopilotCliStyleSummary(events)}\n`);
    expect(summary).toContain("### Conversation");
    expect(summary).toContain("detection.result jobResult=success conclusion=success");
    expect(summary).not.toContain("assembly-secret-fixture");
    expect(summary.match(/<summary>Unified session<\/summary>/g)).toHaveLength(1);
    expect(core.info).toHaveBeenCalledWith(expect.stringContaining("=== Unified session ==="));
    expect(core.warning).not.toHaveBeenCalled();
  });

  it("warns on generator failures and still lists usage files", async () => {
    tempDirectory = fs.mkdtempSync(path.join(os.tmpdir(), "usage-artifacts-"));
    fs.mkdirSync(path.join(tempDirectory, "nested"));
    fs.writeFileSync(path.join(tempDirectory, "nested", "session.jsonl"), "");
    fs.writeFileSync(path.join(tempDirectory, "summary.json"), "");

    const core = { info: vi.fn(), warning: vi.fn() };
    const generateSummary = vi.fn().mockRejectedValue(new Error("summary failed"));
    const generateSession = vi.fn().mockRejectedValue("session failed");

    await main({
      core,
      generateSummary,
      generateSession,
      usageDirectory: tempDirectory,
    });

    expect(generateSummary).toHaveBeenCalledOnce();
    expect(diskUsage.collectDiskUsageArtifact).toHaveBeenCalledOnce();
    expect(generateSession).toHaveBeenCalledOnce();
    expect(core.warning).toHaveBeenCalledWith("Unable to generate usage activity summary: summary failed");
    expect(core.warning).toHaveBeenCalledWith("Unable to generate unified session: session failed");
    expect(core.info.mock.calls.map(([file]) => file)).toEqual([path.join(tempDirectory, "nested", "session.jsonl"), path.join(tempDirectory, "summary.json")]);
  });

  it("reports a contextual error when the usage directory cannot be read", async () => {
    tempDirectory = fs.mkdtempSync(path.join(os.tmpdir(), "usage-artifacts-"));
    const missingDirectory = path.join(tempDirectory, "missing");

    await expect(
      main({
        core: { info: vi.fn(), warning: vi.fn() },
        generateSummary: vi.fn(),
        generateSession: vi.fn(),
        usageDirectory: missingDirectory,
      })
    ).rejects.toThrow(`Failed to read usage artifact directory ${missingDirectory}`);
  });
});
